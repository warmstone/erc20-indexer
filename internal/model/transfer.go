package model

import "time"

type Transfer struct {
	ChainID      int64
	TokenAddress string
	BlockNumber  uint64
	BlockHash    string
	TxHash       string
	LogIndex     uint
	FromAddress  string
	ToAddress    string
	Value        string
	Timestamp    time.Time
}
