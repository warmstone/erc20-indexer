package chain

import (
	"context"
	"erc20-indexer/internal/model"
	"fmt"
	"math/big"
	"time"
)

type BlockFetcher struct {
	client       *Client
	limiter      *RateLimiter
	retries      int
	retryBackoff time.Duration
}

func NewBlockFetcher(client *Client, interval time.Duration, retries int, retrcyBackoff time.Duration) *BlockFetcher {
	if retries < 0 {
		retries = 0
	}
	return &BlockFetcher{
		client:       client,
		limiter:      NewRateLimiter(interval),
		retries:      retries,
		retryBackoff: retrcyBackoff,
	}
}

func (f *BlockFetcher) BlockByNumber(ctx context.Context, number uint64, chainID int64) (model.IndexedBlock, error) {
	var lastErr error
	for attempt := 0; attempt < f.retries; attempt++ {
		if err := f.limiter.wait(ctx); err != nil {
			return model.IndexedBlock{}, err
		}
		block, err := f.client.eth.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err == nil {
			return model.IndexedBlock{
				ChainID:     chainID,
				BlockNumber: block.NumberU64(),
				BlockHash:   block.Hash().Hex(),
				ParentHash:  block.ParentHash().Hex(),
				Timestamp:   time.Unix(int64(block.Time()), 0).UTC(),
			}, nil
		}
		lastErr = err
		if attempt == f.retries-1 {
			break
		}
		if err := sleepBackOff(ctx, f.retryBackoff, attempt); err != nil {
			return model.IndexedBlock{}, nil
		}
	}
	return model.IndexedBlock{}, fmt.Errorf("fetch block %d failed after %d attempt(s): %w", number, f.retries, lastErr)
}
