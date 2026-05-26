package model

import "time"

type IndexedBlock struct {
	ChainID     int64
	BlockNumber uint64
	BlockHash   string
	ParentHash  string
	Timestamp   time.Time
}
