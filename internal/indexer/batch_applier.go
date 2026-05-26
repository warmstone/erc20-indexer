package indexer

import (
	"context"
	"erc20-indexer/internal/model"
	"erc20-indexer/internal/storage"
	"erc20-indexer/pkg/ethutil"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
)

type BatchApplier struct{}

func NewBatchApplier() *BatchApplier {
	return &BatchApplier{}
}

func (a *BatchApplier) Apply(ctx context.Context, tx pgx.Tx, transfers []DecodedTransfer, approvals []DecodedApproval) error {
	insertedTransfer, err := a.insertTransfer(ctx, tx, transfers)
	if err != nil {
		return err
	}
	insertedApprovals, err := a.insertApprovals(ctx, tx, approvals)
	if err != nil {
		return err
	}
	if err := a.applyBalanceDeltas(ctx, tx, insertedTransfer); err != nil {
		return err
	}
	return a.applyAllowances(ctx, tx, insertedApprovals)
}

func (a *BatchApplier) insertTransfer(ctx context.Context, tx pgx.Tx, transfers []DecodedTransfer) ([]DecodedTransfer, error) {
	if len(transfers) == 0 {
		return nil, nil
	}
	batch := &pgx.Batch{}
	for _, e := range transfers {
		t := transferModel(e)
		batch.Queue(`
			INSERT INTO erc20_transfers (
				chain_id, token_address, block_number, block_hash, tx_hash, log_index,
				from_address, to_address, value, timestamp
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (chain_id, tx_hash, log_index) DO NOTHING
		`, t.ChainID, t.TokenAddress, t.BlockNumber, t.BlockHash, t.TxHash, t.LogIndex, t.FromAddress, t.ToAddress, t.Value, t.Timestamp)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	inserted := make([]DecodedTransfer, 0, len(transfers))
	for i, e := range transfers {
		tag, err := br.Exec()
		if err != nil {
			return nil, fmt.Errorf("insert transfer %d: %w", i, err)
		}
		if tag.RowsAffected() > 0 {
			inserted = append(inserted, e)
		}
	}
	return inserted, nil
}

func (a *BatchApplier) insertApprovals(ctx context.Context, tx pgx.Tx, approvals []DecodedApproval) ([]DecodedApproval, error) {
	if len(approvals) == 0 {
		return nil, nil
	}
	batch := &pgx.Batch{}
	for _, e := range approvals {
		t := approvalModel(e)
		batch.Queue(`
			INSERT INTO erc20_approvals (
				chain_id, token_address, block_number, block_hash, tx_hash, log_index,
				owner_address, spender_address, value, timestamp
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (chain_id, tx_hash, log_index) DO NOTHING
		`, t.ChainID, t.TokenAddress, t.BlockNumber, t.BlockHash, t.TxHash, t.LogIndex, t.OwnerAddress, t.SpenderAddress, t.Value, t.Timestamp)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	inserted := make([]DecodedApproval, 0, len(approvals))
	for i, e := range approvals {
		tag, err := br.Exec()
		if err != nil {
			return nil, fmt.Errorf("insert approval %d: %w", i, err)
		}
		if tag.RowsAffected() > 0 {
			inserted = append(inserted, e)
		}
	}
	return inserted, nil
}

func (a *BatchApplier) applyBalanceDeltas(ctx context.Context, tx pgx.Tx, transfers []DecodedTransfer) error {
	repos := storage.NewRepositories(tx)
	type key struct {
		chainID int64
		token   string
		holder  string
	}
	type aggregate struct {
		value       *big.Int
		blockNumber uint64
	}

	deltas := make(map[key]aggregate)
	addDelta := func(k key, value *big.Int, blockNumber uint64) {
		current := deltas[k]
		if current.value == nil {
			current.value = new(big.Int)
		}
		current.value.Add(current.value, value)
		if blockNumber > current.blockNumber {
			current.blockNumber = blockNumber
		}
		deltas[k] = current
	}

	for _, e := range transfers {
		token := ethutil.NormalizeAddress(e.TokenAddress)
		if e.From != (common.Address{}) {
			addDelta(key{e.ChainID, token, ethutil.NormalizeAddress(e.From)}, new(big.Int).Neg(e.Value), e.BlockNumber)
		}
		if e.To != (common.Address{}) {
			addDelta(key{e.ChainID, token, ethutil.NormalizeAddress(e.To)}, e.Value, e.BlockNumber)
		}
	}

	for k, d := range deltas {
		if d.value.Sign() >= 0 {
			if err := repos.Balances.Add(ctx, k.chainID, k.token, k.holder, d.value.String(), d.blockNumber); err != nil {
				return err
			}
			continue
		}
		if err := repos.Balances.Subtract(ctx, k.chainID, k.token, k.holder, new(big.Int).Abs(d.value).String(), d.blockNumber); err != nil {
			return err
		}
	}
	return nil
}

func (a *BatchApplier) applyAllowances(ctx context.Context, tx pgx.Tx, approvals []DecodedApproval) error {
	repos := storage.NewRepositories(tx)
	type key struct {
		chainID int64
		token   string
		owner   string
		spender string
	}
	type latest struct {
		value       string
		blockNumber uint64
		logIndex    uint
	}
	allowances := make(map[key]latest)
	for _, e := range approvals {
		k := key{
			chainID: e.ChainID,
			token:   ethutil.NormalizeAddress(e.TokenAddress),
			owner:   ethutil.NormalizeAddress(e.Owner),
			spender: ethutil.NormalizeAddress(e.Spender),
		}
		current, ok := allowances[k]
		if !ok || e.BlockNumber > current.blockNumber || (e.BlockNumber == current.blockNumber && e.LogIndex > current.logIndex) {
			allowances[k] = latest{
				value:       ethutil.BigIntString(e.Value),
				blockNumber: e.BlockNumber,
				logIndex:    e.LogIndex,
			}
		}
	}

	for k, v := range allowances {
		if err := repos.Balances.SetAllowance(ctx, k.chainID, k.token, k.owner, k.spender, v.value, v.blockNumber); err != nil {
			return err
		}
	}
	return nil
}

func transferModel(e DecodedTransfer) model.Transfer {
	return model.Transfer{
		ChainID:      e.ChainID,
		TokenAddress: ethutil.NormalizeAddress(e.TokenAddress),
		BlockNumber:  e.BlockNumber,
		BlockHash:    e.BlockHash.Hex(),
		TxHash:       e.TxHash.Hex(),
		LogIndex:     e.LogIndex,
		FromAddress:  ethutil.NormalizeAddress(e.From),
		ToAddress:    ethutil.NormalizeAddress(e.To),
		Value:        ethutil.BigIntString(e.Value),
		Timestamp:    e.Timestamp,
	}
}

func approvalModel(e DecodedApproval) model.Approval {
	return model.Approval{
		ChainID:        e.ChainID,
		TokenAddress:   ethutil.NormalizeAddress(e.TokenAddress),
		BlockNumber:    e.BlockNumber,
		BlockHash:      e.BlockHash.Hex(),
		TxHash:         e.TxHash.Hex(),
		LogIndex:       e.LogIndex,
		OwnerAddress:   ethutil.NormalizeAddress(e.Owner),
		SpenderAddress: ethutil.NormalizeAddress(e.Spender),
		Value:          ethutil.BigIntString(e.Value),
		Timestamp:      e.Timestamp,
	}
}
