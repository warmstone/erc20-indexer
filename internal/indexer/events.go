package indexer

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type DecodedTransfer struct {
	ChainID      int64
	TokenAddress common.Address
	From         common.Address
	To           common.Address
	Value        *big.Int
	TxHash       common.Hash
	LogIndex     uint
	BlockNumber  uint64
	BlockHash    common.Hash
	Timestamp    time.Time
}

type DecodedApproval struct {
	ChainID      int64
	TokenAddress common.Address
	Owner        common.Address
	Spender      common.Address
	Value        *big.Int
	TxHash       common.Hash
	LogIndex     uint
	BlockNumber  uint64
	BlockHash    common.Hash
	Timestamp    time.Time
}
