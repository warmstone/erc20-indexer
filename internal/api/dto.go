package api

import "time"

type TokenDTO struct {
	ChainID     int64     `json:"chain_id"`
	Address     string    `json:"address"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	Decimals    uint8     `json:"decimals"`
	TotalSupply string    `json:"total_supply"`
	IsActive    bool      `json:"is_active"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TransferDTO struct {
	TokenAddress string    `json:"token_address"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Value        string    `json:"value"`
	TxHash       string    `json:"tx_hash"`
	LogIndex     uint      `json:"log_index"`
	BlockNumber  uint64    `json:"block_number"`
	Timestamp    time.Time `json:"timestamp"`
}

type ApprovalDTO struct {
	TokenAddress string    `json:"token_address"`
	Owner        string    `json:"owner"`
	Spender      string    `json:"spender"`
	Value        string    `json:"value"`
	TxHash       string    `json:"tx_hash"`
	LogIndex     uint      `json:"log_index"`
	BlockNumber  uint64    `json:"block_number"`
	Timestamp    time.Time `json:"timestamp"`
}

type BalanceDTO struct {
	TokenAddress       string    `json:"token_address"`
	Holder             string    `json:"holder"`
	Balance            string    `json:"balance"`
	UpdatedBlockNumber uint64    `json:"updated_block_number"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type SyncStateDTO struct {
	TokenAddress     string    `json:"token_address"`
	LastIndexedBlock uint64    `json:"last_indexed_block"`
	FinalizedBlock   uint64    `json:"finalized_block"`
	UpdatedAt        time.Time `json:"updated_at"`
}
