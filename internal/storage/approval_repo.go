package storage

import (
	"context"
	"erc20-indexer/internal/model"
	"fmt"
)

type ApprovalRepo struct {
	db DBTX
}

func (r *ApprovalRepo) Insert(ctx context.Context, a model.Approval) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO erc20_approvals (
			chain_id, token_address, block_number, block_hash, tx_hash, log_index,
			owner_address, spender_address, value, timestamp
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (chain_id, tx_hash, log_index) DO NOTHING
	`, a.ChainID, a.TokenAddress, a.BlockNumber, a.BlockHash, a.TxHash, a.LogIndex, a.OwnerAddress, a.SpenderAddress, a.Value, a.Timestamp)
	return err
}

func (r *ApprovalRepo) List(ctx context.Context, chainID int64, token, owner, spender string, limit, offset int) ([]model.Approval, error) {
	args := []any{chainID, token, limit, offset}
	where := "chain_id = $1 AND token_address = $2"
	if owner != "" {
		where += fmt.Sprintf(" AND owner_address = $%d", len(args)+1)
		args = append(args, owner)
	}
	if spender != "" {
		where += fmt.Sprintf(" AND spender_address = $%d", len(args)+1)
		args = append(args, spender)
	}
	rows, err := r.db.Query(ctx, `
		SELECT chain_id, token_address, block_number, block_hash, tx_hash, log_index, owner_address, spender_address, value::text, timestamp
		FROM erc20_approvals
		WHERE `+where+`
		ORDER BY block_number DESC, log_index DESC
		LIMIT $3 OFFSET $4
	`, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var out []model.Approval
	for rows.Next() {
		var a model.Approval
		if err := rows.Scan(&a.ChainID, &a.TokenAddress, &a.BlockNumber, &a.BlockHash, &a.TxHash, &a.LogIndex, &a.OwnerAddress, &a.SpenderAddress, &a.Value, &a.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *ApprovalRepo) DeleteAfterBlock(ctx context.Context, chainID int64, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM erc20_approvals WHERE chain_id = $1 AND block_number > $2`, chainID, blockNumber)
	return err
}
