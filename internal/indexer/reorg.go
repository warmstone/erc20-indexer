package indexer

import (
	"context"
	"erc20-indexer/internal/chain"
	"erc20-indexer/internal/storage"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type ReorgDetector struct {
	blocks *chain.BlockFetcher
	db     storage.DBTX
	depth  uint64
}

func NewReorgDetector(blocks *chain.BlockFetcher, db storage.DBTX, depth uint64) *ReorgDetector {
	if depth == 0 {
		depth = 64
	}
	return &ReorgDetector{blocks: blocks, db: db, depth: depth}
}

func (r *ReorgDetector) FindRollbackBlock(ctx context.Context, chainID int64, lastIndexed uint64) (uint64, bool, error) {
	repos := storage.NewRepositories(r.db)
	if lastIndexed == 0 {
		return 0, false, nil
	}
	start := uint64(1)
	if lastIndexed > r.depth {
		start = lastIndexed - r.depth + 1
	}
	mismatched := false
	for n := lastIndexed; n >= start; n-- {
		local, err := repos.Blocks.Get(ctx, chainID, n)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, false, err
		}
		remote, err := r.blocks.BlockByNumber(ctx, n, chainID)
		if err != nil {
			return 0, false, err
		}
		if local.BlockHash == remote.BlockHash {
			if mismatched {
				return n, true, nil
			}
			return 0, false, nil
		}
		mismatched = true
	}
	if mismatched {
		return 0, false, fmt.Errorf("reorg depth exceeds %d blocks; no common ancestor found from block %d down to %d", r.depth, lastIndexed, start)
	}
	return 0, false, nil
}
