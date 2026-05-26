package model

import "time"

type Approval struct {
	ChainID        int64
	TokenAddress   string
	BlockNumber    uint64
	BlockHash      string
	TxHash         string
	LogIndex       uint
	OwnerAddress   string
	SpenderAddress string
	Value          string
	Timestamp      time.Time
}
