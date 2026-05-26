package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type LogFetcher struct {
	client       *Client
	limiter      *RateLimiter
	retries      int
	retryBackoff time.Duration
}

func NewLogFetcher(client *Client, interval time.Duration, retries int, retryBackoff time.Duration) *LogFetcher {
	if retries < 0 {
		retries = 0
	}
	return &LogFetcher{
		client:       client,
		limiter:      NewRateLimiter(interval),
		retries:      retries,
		retryBackoff: retryBackoff,
	}
}

func (f *LogFetcher) FetchLogs(ctx context.Context, address common.Address, fromBlock, toBlock uint64, topics [][]common.Hash) ([]types.Log, error) {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{address},
		Topics:    topics,
	}

	var lastErr error
	for attempt := 0; attempt < f.retries; attempt++ {
		if err := f.limiter.wait(ctx); err != nil {
			return nil, err
		}
		logs, err := f.client.eth.FilterLogs(ctx, query)
		if err == nil {
			return logs, nil
		}
		lastErr = err
		if attempt == f.retries-1 {
			break
		}
		if err := sleepBackOff(ctx, f.retryBackoff, attempt); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("fetch logs [%d-%d] failed after %d attempt(s): %w", fromBlock, toBlock, f.retries, lastErr)
}
