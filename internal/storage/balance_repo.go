package storage

import (
	"context"
	"erc20-indexer/internal/model"
)

type BalanceRepo struct {
	db DBTX
}

func (r *BalanceRepo) Add(ctx context.Context, chainID int64, token, holder, delta string, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO erc20_balances (chain_id, token_address, holder_address, balance, updated_block_number)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chain_id, token_address, holder_address) DO UPDATE SET
			balance = erc20_balances.balance + EXCLUDED.balance,
			updated_block_number = EXCLUDED.updated_block_number,
			updated_at = now()
	`, chainID, token, holder, delta, blockNumber)
	return err
}

func (r *BalanceRepo) Subtract(ctx context.Context, chainID int64, token, holder, delta string, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO erc20_balances (chain_id, token_address, holder_address, balance, updated_block_number)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chain_id, token_address, holder_address) DO UPDATE SET
			balance = erc20_balances.balance - EXCLUDED.balance,
			updated_block_number = EXCLUDED.updated_block_number,
			updated_at = now()
	`, chainID, token, holder, delta, blockNumber)
	return err
}

func (r *BalanceRepo) SetAllowance(ctx context.Context, chainID int64, token, owner, spender, value string, blockNumber uint64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO erc20_allowances (chain_id, token_address, owner_address, spender_address, allowance, updated_block_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chain_id, token_address, owner_address, spender_address) DO UPDATE SET
			allowance = EXCLUDED.allowance,
			updated_block_number = EXCLUDED.updated_block_number,
			updated_at = now()
	`, chainID, token, owner, spender, value, blockNumber)
	return err
}

func (r *BalanceRepo) Get(ctx context.Context, chainID int64, token, owner string) (model.Balance, error) {
	var b model.Balance
	err := r.db.QueryRow(ctx, `
		SELECT chain_id, token_address, holder_address, balance::text, updated_block_number, updated_at
		FROM erc20_balances
		WHERE chain_id = $1 AND token_address = $2 AND holder_address = $3
	`, chainID, token, owner).Scan(&b.ChainID, &b.TokenAddress, &b.HolderAddress, &b.Balance, &b.UpdatedBlockNumber, &b.UpdatedAt)
	return b, err
}

func (r *BalanceRepo) Holders(ctx context.Context, chainID int64, token string, limit, offset int) ([]model.Balance, error) {
	rows, err := r.db.Query(ctx, `
		SELECT chain_id, token_address, holder_address, balance::text, updated_block_number, updated_at
		FROM erc20_balances
		WHERE chain_id = $1 AND token_address = $2 AND balance <> 0
		ORDER BY balance DESC
		LIMIT $3 OFFSET $4
	`, chainID, token, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Balance
	for rows.Next() {
		var b model.Balance
		if err := rows.Scan(&b.ChainID, &b.TokenAddress, &b.HolderAddress, &b.Balance, &b.UpdatedBlockNumber, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *BalanceRepo) RebuildFromTransfer(ctx context.Context, chainID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM erc20_balances WHERE chain_id = $1;

		INSERT INTO erc20_balances (chain_id, token_address, holder_address, balance, updated_block_number)
		SELECT chain_id, token_address, holder_address, SUM(delta), MAX(block_number)
		FROM (
			SELECT chain_id, token_address, to_address AS holder_address, value AS delta, block_number
			FROM erc20_transfers
			WHERE chain_id = $1 AND to_address <> '0x0000000000000000000000000000000000000000'
			UNION ALL
			SELECT chain_id, token_address, from_address AS holder_address, (0 - value) AS delta, block_number
			FROM erc20_transfers
			WHERE chain_id = $1 AND from_address <> '0x0000000000000000000000000000000000000000'
		) x
		GROUP BY chain_id, token_address, holder_address
	`, chainID)
	return err
}

func (r *BalanceRepo) RebuildAllowancesFromApprovals(ctx context.Context, chainID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM erc20_allowances WHERE chain_id = $1;

		INSERT INTO erc20_allowances (
			chain_id, token_address, owner_address, spender_address, allowance, updated_block_number
		)
		SELECT DISTINCT ON (chain_id, token_address, owner_address, spender_address)
			chain_id, token_address, owner_address, spender_address, value, block_number
		FROM erc20_approvals
		WHERE chain_id = $1
		ORDER BY chain_id, token_address, owner_address, spender_address, block_number DESC, log_index DESC
	`, chainID)
	return err
}

func (r *BalanceRepo) DeleteAllowancesAfterBlock(ctx context.Context, chainID int64, blockNumber int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM erc20_allowances WHERE chain_id = $1 AND updated_block_number > $2`, chainID, blockNumber)
	return err
}
