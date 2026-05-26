package storage

import (
	"context"
	"erc20-indexer/internal/model"
	"fmt"
)

type TransferRepo struct {
	db DBTX
}

func (r *TransferRepo) Insert(ctx context.Context, t model.Transfer) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO erc20_transfers (
			chain_id, token_address, block_number, block_hash, tx_hash, log_index, 
			from_address, to_address, value, timestamp
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (chain_id, tx_hash, log_index) DO NOTHING
	`, t.ChainID, t.TokenAddress, t.BlockNumber, t.BlockHash, t.TxHash, t.LogIndex, t.FromAddress, t.ToAddress, t.Value, t.Timestamp)
	return err
}

func (r *TransferRepo) List(ctx context.Context, chainID int64, token, holder string, limit, offset int) ([]model.Transfer, error) {
	args := []any{chainID, token, limit, offset}
	where := "chain_id = $1 AND token_address = $2"
	if holder != "" {
		where += fmt.Sprintf(" AND (from_address = $%d OR to_address = $%d)", len(args)+1, len(args)+1)
		args = append(args, holder)
	}
	rows, err := r.db.Query(ctx, `
		SELECT chain_id, token_address, block_number, block_hash, tx_hash, log_index, from_address, to_address, value, timestamp
		FROM erc20_transfers
		WHERE `+where+`
		ORDER BY block_number DESC, log_index DESC
		LIMIT $3 OFFSET $4
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Transfer
	for rows.Next() {
		var t model.Transfer
		if err := rows.Scan(&t.ChainID, &t.ToAddress, &t.BlockNumber, &t.BlockHash, &t.TxHash, &t.LogIndex, &t.FromAddress, &t.ToAddress, &t.Value, &t.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TransferRepo) DeleteAfterBlock(ctx context.Context, chainID int64, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM erc20_transfer WHERE chain_id = $1 AND block_number > $2`, chainID, blockNumber)
	return err
}
