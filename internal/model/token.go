package model

import "time"

type Token struct {
	ChainID     int64
	Address     string
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
