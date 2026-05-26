package model

import "time"

type SyncState struct {
	ChainID          int64
	TokenAddress     string
	LastIndexedBlock uint64
	FinalizedBlock   uint64
	UpdatedAt        time.Time
}
