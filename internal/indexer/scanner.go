package indexer

import (
	"context"
	"erc20-indexer/internal/chain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Scanner struct {
	logFetcher *chain.LogFetcher
}

func NewScanner(logFetcher *chain.LogFetcher) *Scanner {
	return &Scanner{logFetcher: logFetcher}
}

func (s *Scanner) Scan(ctx context.Context, token common.Address, fromBlock, toBlock uint64) ([]types.Log, error) {
	topic := [][]common.Hash{{TransferTopic, ApprovalTopic}}
	return s.logFetcher.FetchLogs(ctx, token, fromBlock, toBlock, topic)
}
