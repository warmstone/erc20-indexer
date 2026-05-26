package storage

import (
	"context"
	"erc20-indexer/internal/model"
	"errors"

	"github.com/jackc/pgx/v5"
)

type BlockRepo struct {
	db DBTX
}

func (r *BlockRepo) Upsert(ctx context.Context, b model.IndexedBlock) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO indexed_blocks (chain_id, block_number, block_hash, parent_hash, timestamp)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chain_id, block_number) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			parent_hash = EXCLUDED.parent_hash,
			timestamp = EXCLUDED.timestamp
	`, b.ChainID, b.BlockNumber, b.BlockHash, b.ParentHash, b.Timestamp)
	return err
}

func (r *BlockRepo) Get(ctx context.Context, chainID int64, blockNumber uint64) (model.IndexedBlock, error) {
	var b model.IndexedBlock
	err := r.db.QueryRow(ctx, `
		SELECT chain_id, block_number, block_hash, parent_hash, timestamp
		FROM indexed_blocks
		WHERE chain_id = $1 AND block_number = $2
	`, chainID, blockNumber).Scan(&b.ChainID, &b.BlockNumber, &b.BlockHash, &b.ParentHash, &b.Timestamp)
	return b, err
}

func (r *BlockRepo) LatestNumber(ctx context.Context, chainID int64) (uint64, bool, error) {
	var blockNumber uint64
	err := r.db.QueryRow(ctx, `
		SELECT block_number
		FROM indexed_blocks
		WHERE chain_id = $1
		ORDER BY block_number DESC
		LIMIT 1
	`, chainID).Scan(&blockNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return blockNumber, true, nil
}

func (r *BlockRepo) DeleteAfterBlock(ctx context.Context, chainID int64, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM indexed_blocks WHERE chain_id = $1 AND block_number > $2`, chainID, blockNumber)
	return err
}
