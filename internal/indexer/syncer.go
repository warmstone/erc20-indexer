package indexer

import (
	"context"
	"erc20-indexer/internal/chain"
	"erc20-indexer/internal/config"
	"erc20-indexer/internal/model"
	"erc20-indexer/internal/storage"
	"erc20-indexer/pkg/ethutil"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Syncer struct {
	cfg     config.Config
	client  *chain.Client
	blocks  *chain.BlockFetcher
	db      *pgxpool.Pool
	scanner *Scanner
	sizer   *BatchSizer
	decoder *Decoder
	applier *BatchApplier
	reorg   *ReorgDetector
	log     zerolog.Logger
}

func NewSyncer(cfg config.Config, client *chain.Client, db *pgxpool.Pool, log zerolog.Logger) *Syncer {
	blockFetcher := chain.NewBlockFetcher(client, cfg.Chain.BlockInterval, cfg.Chain.BlockRetries, cfg.Chain.BlockBackoff)
	return &Syncer{
		cfg:     cfg,
		client:  client,
		db:      db,
		blocks:  blockFetcher,
		scanner: NewScanner(chain.NewLogFetcher(client, cfg.Chain.LogInterval, cfg.Chain.LogRetries, cfg.Chain.LogBackoff)),
		sizer:   NewBatchSizer(cfg.Chain.BatchSize, cfg.Chain.MinBatchSize),
		decoder: NewDecoder(cfg.Chain.ChainID),
		applier: NewBatchApplier(),
		reorg:   NewReorgDetector(blockFetcher, db, cfg.Chain.ReorgDepth),
		log:     log,
	}
}

func (s *Syncer) Run(ctx context.Context) error {
	if err := s.bootstrapTokens(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(s.cfg.Chain.PollInterval)
	defer ticker.Stop()

	for {
		if err := s.syncOnce(ctx); err != nil {
			s.log.Error().Err(err).Msg("sync failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Syncer) bootstrapTokens(ctx context.Context) error {
	repos := storage.NewRepositories(s.db)
	for _, raw := range s.cfg.Tokens {
		addr := common.HexToAddress(raw)
		meta, err := s.client.ERC20Metadata(ctx, addr)
		if err != nil {
			s.log.Warn().Err(err).Str("token", raw).Msg("failed to read token metadata")
		}
		token := model.Token{
			ChainID:     s.cfg.Chain.ChainID,
			Address:     ethutil.NormalizeAddress(addr),
			Name:        meta.Name,
			Symbol:      meta.Symbol,
			Decimals:    meta.Decimals,
			TotalSupply: ethutil.BigIntString(meta.TotalSupply),
			IsActive:    true,
		}
		if err := repos.Tokens.Upsert(ctx, token); err != nil {
			return err
		}
		if _, err := repos.SyncState.GetOrCreate(ctx, s.cfg.Chain.ChainID, token.Address, s.cfg.Chain.StartBlock); err != nil {
			return err
		}
	}
	return nil
}

func (s *Syncer) syncOnce(ctx context.Context) error {
	latest, err := s.client.LatestBlockNumber(ctx)
	if err != nil {
		return err
	}
	if latest <= s.cfg.Chain.Confirmations {
		return nil
	}
	safeBlock := latest - s.cfg.Chain.Confirmations

	repos := storage.NewRepositories(s.db)
	if err := s.handleReorg(ctx, repos); err != nil {
		return err
	}

	tokens, err := repos.Tokens.ListActive(ctx, s.cfg.Chain.ChainID)
	if err != nil {
		return err
	}
	var syncErr error
	for _, token := range tokens {
		tokenAddr := common.HexToAddress(token.Address)
		if err := s.syncToken(ctx, tokenAddr, token.Address, safeBlock); err != nil {
			s.log.Err(err).Str("token", token.Address).Msg("sync token failed")
			syncErr = errors.Join(syncErr, err)
		}
	}
	return syncErr
}

func (s *Syncer) handleReorg(ctx context.Context, repos storage.Repositories) error {
	latestIndexed, ok, err := repos.Blocks.LatestNumber(ctx, s.cfg.Chain.ChainID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	rollbackTo, changed, err := s.reorg.FindRollbackBlock(ctx, s.cfg.Chain.ChainID, latestIndexed)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	s.log.Warn().Uint64("rollback_to", rollbackTo).Msg("reorg detected")
	return storage.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		repos := storage.NewRepositories(tx)
		if err := repos.Transfers.DeleteAfterBlock(ctx, s.cfg.Chain.ChainID, rollbackTo); err != nil {
			return err
		}
		if err := repos.Approvals.DeleteAfterBlock(ctx, s.cfg.Chain.ChainID, rollbackTo); err != nil {
			return err
		}
		if err := repos.Blocks.DeleteAfterBlock(ctx, s.cfg.Chain.ChainID, rollbackTo); err != nil {
			return err
		}
		if err := repos.Balances.RebuildFromTransfer(ctx, s.cfg.Chain.ChainID); err != nil {
			return err
		}
		if err := repos.Balances.RebuildAllowancesFromApprovals(ctx, s.cfg.Chain.ChainID); err != nil {
			return err
		}
		return repos.SyncState.RollbackAfterBlock(ctx, s.cfg.Chain.ChainID, rollbackTo)
	})
}

func (s *Syncer) syncToken(ctx context.Context, tokenAddr common.Address, token string, safeBlock uint64) error {
	repos := storage.NewRepositories(s.db)
	state, err := repos.SyncState.GetOrCreate(ctx, s.cfg.Chain.ChainID, token, s.cfg.Chain.StartBlock)
	if err != nil {
		return err
	}

	from, to, ok := s.sizer.NextRange(token, state.LastIndexedBlock, safeBlock)
	if !ok {
		return nil
	}

	logs, err := s.scanner.Scan(ctx, tokenAddr, from, to)
	if err != nil {
		s.sizer.RecordFailure(token)
		return err
	}
	blockCache, err := s.loadBlocks(ctx, from, to)
	if err != nil {
		s.sizer.RecordFailure(token)
		return err
	}
	transfers, approvals := s.decodeLogs(logs, blockCache)

	if err := s.commitBatch(ctx, token, to, blockCache, transfers, approvals); err != nil {
		s.sizer.RecordFailure(token)
		return err
	}
	s.sizer.RecordSuccess(token)
	return nil
}

func (s *Syncer) decodeLogs(logs []types.Log, blocks map[uint64]model.IndexedBlock) ([]DecodedTransfer, []DecodedApproval) {
	transfers := make([]DecodedTransfer, 0, len(logs))
	approvals := make([]DecodedApproval, 0, len(logs))
	for _, rawLog := range logs {
		block := blocks[rawLog.BlockNumber]
		if event, ok := s.decoder.DecodedTransfer(rawLog); ok {
			event.Timestamp = block.Timestamp
			transfers = append(transfers, event)
		}
		if event, ok := s.decoder.DecodedApproval(rawLog); ok {
			event.Timestamp = block.Timestamp
			approvals = append(approvals, event)
		}
	}
	return transfers, approvals
}

func (s *Syncer) commitBatch(
	ctx context.Context,
	token string,
	toBlock uint64,
	blocks map[uint64]model.IndexedBlock,
	transfers []DecodedTransfer,
	approvals []DecodedApproval,
) error {
	return storage.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		repos := storage.NewRepositories(tx)
		for _, block := range blocks {
			if err := repos.Blocks.Upsert(ctx, block); err != nil {
				return err
			}
		}
		if err := s.applier.Apply(ctx, tx, transfers, approvals); err != nil {
			return err
		}
		return repos.SyncState.Upsert(ctx, model.SyncState{
			ChainID:          s.cfg.Chain.ChainID,
			TokenAddress:     token,
			LastIndexedBlock: toBlock,
			FinalizedBlock:   toBlock,
		})
	})
}

func (s *Syncer) loadBlocks(ctx context.Context, from, to uint64) (map[uint64]model.IndexedBlock, error) {
	count := int(to - from + 1)
	blocks := make(map[uint64]model.IndexedBlock, count)
	blocksMu := sync.Mutex{}

	type result struct {
		number uint64
		block  model.IndexedBlock
		err    error
	}
	sem := make(chan struct{}, 1)
	results := make(chan result, count)

	go func() {
		var wg sync.WaitGroup
		for n := from; n <= to; n++ {
			sem <- struct{}{}
			wg.Add(1)
			go func(blockNum uint64) {
				defer wg.Done()
				defer func() { <-sem }()
				b, err := s.blocks.BlockByNumber(ctx, blockNum, s.cfg.Chain.ChainID)
				results <- result{number: blockNum, block: b, err: err}
			}(n)
		}
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		blocksMu.Lock()
		blocks[r.number] = r.block
		blocksMu.Unlock()
	}
	return blocks, nil
}
