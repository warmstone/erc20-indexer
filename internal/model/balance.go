package model

import "time"

type Balance struct {
	ChainID            int64
	TokenAddress       string
	HolderAddress      string
	Balance            string
	UpdatedBlockNumber uint64
	UpdatedAt          time.Time
}
