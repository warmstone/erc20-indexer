package indexer

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	TransferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	ApprovalTopic = crypto.Keccak256Hash([]byte("Approval(address,address,uint256)"))
)

type Decoder struct {
	chainID int64
}

func NewDecoder(chainID int64) *Decoder {
	return &Decoder{chainID: chainID}
}

func (d *Decoder) DecodedTransfer(log types.Log) (DecodedTransfer, bool) {
	if len(log.Topics) != 3 || log.Topics[0] != TransferTopic {
		return DecodedTransfer{}, false
	}
	return DecodedTransfer{
		ChainID:      d.chainID,
		TokenAddress: log.Address,
		From:         common.BytesToAddress(log.Topics[1].Bytes()),
		To:           common.BytesToAddress(log.Topics[2].Bytes()),
		Value:        decodeUint256Value(log.Data),
		TxHash:       log.TxHash,
		LogIndex:     log.Index,
		BlockNumber:  log.BlockNumber,
		BlockHash:    log.BlockHash,
		Timestamp:    time.Unix(int64(log.BlockTimestamp), 0).UTC(),
	}, true
}

func (d *Decoder) DecodedApproval(log types.Log) (DecodedApproval, bool) {
	if len(log.Topics) != 3 || log.Topics[0] != ApprovalTopic {
		return DecodedApproval{}, false
	}
	return DecodedApproval{
		ChainID:      d.chainID,
		TokenAddress: log.Address,
		Owner:        common.BytesToAddress(log.Topics[1].Bytes()),
		Spender:      common.BytesToAddress(log.Topics[2].Bytes()),
		Value:        decodeUint256Value(log.Data),
		TxHash:       log.TxHash,
		LogIndex:     log.Index,
		BlockNumber:  log.BlockNumber,
		BlockHash:    log.BlockHash,
		Timestamp:    time.Unix(int64(log.BlockTimestamp), 0).UTC(),
	}, true
}

func decodeUint256Value(data []byte) *big.Int {
	if len(data) > 32 {
		data = data[:32]
	}
	return new(big.Int).SetBytes(data)
}
