package storage

import (
	"context"
	"erc20-indexer/internal/model"
	"errors"

	"github.com/jackc/pgx/v5"
)

type SyncStateRepo struct {
	db DBTX
}

func (r *SyncStateRepo) GetOrCreate(ctx context.Context, chainID int64, token string, startBlock uint64) (model.SyncState, error) {
	state, err := r.Get(ctx, chainID, token)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.SyncState{}, err
	}
	lastIndexed := uint64(0)
	if startBlock > 0 {
		lastIndexed = startBlock - 1
	}
	state = model.SyncState{
		ChainID:          chainID,
		TokenAddress:     token,
		LastIndexedBlock: lastIndexed,
		FinalizedBlock:   lastIndexed,
	}
	return state, r.Upsert(ctx, state)
}

func (r *SyncStateRepo) Get(ctx context.Context, chainID int64, token string) (model.SyncState, error) {
	var s model.SyncState
	err := r.db.QueryRow(ctx, `
		SELECT chain_id, token_address, last_indexed_block, finalized_block, updated_at
		FROM sync_state
		WHERE chain_id = $1 AND token_address = $2
	`, chainID, token).Scan(&s.ChainID, &s.TokenAddress, &s.LastIndexedBlock, &s.FinalizedBlock, &s.UpdatedAt)
	return s, err
}

func (r *SyncStateRepo) Upsert(ctx context.Context, s model.SyncState) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sync_state (chain_id, token_address, last_indexed_block, finalized_block)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (chain_id, token_address) DO UPDATE SET
			last_indexed_block = EXCLUDED.last_indexed_block,
			finalized_block = EXCLUDED.finalized_block,
			updated_at = now()
	`, s.ChainID, s.TokenAddress, s.LastIndexedBlock, s.FinalizedBlock)
	return err
}

func (r *SyncStateRepo) List(ctx context.Context, chainID int64) ([]model.SyncState, error) {
	rows, err := r.db.Query(ctx, `
		SELECT chain_id, token_address, last_indexed_block, finalized_block, updated_at
		FROM sync_state
		WHERE chain_id = $1
		ORDER BY token_address
	`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SyncState
	for rows.Next() {
		var s model.SyncState
		if err := rows.Scan(&s.ChainID, &s.TokenAddress, &s.LastIndexedBlock, &s.FinalizedBlock, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SyncStateRepo) RollbackAfterBlock(ctx context.Context, chainID int64, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sync_state
		SET
			last_indexed_block = LEAST(last_indexed_block, $2),
			finalized_block = LEAST(finalized_block, $2),
			updated_at = now()
		WHERE chain_id = $1
	`, chainID, blockNumber)
	return err
}
