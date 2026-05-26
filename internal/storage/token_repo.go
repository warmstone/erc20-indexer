package storage

import (
	"context"
	"erc20-indexer/internal/model"
)

type TokenRepo struct {
	db DBTX
}

func (r *TokenRepo) Upsert(ctx context.Context, t model.Token) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tokens (chain_id, address, name, symbol, decimals, total_supply, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (chain_id, address) DO UPDATE SET
			name = EXCLUDED.name,
			symbol = EXCLUDED.symbol,
			decimals = EXCLUDED.decimals,
			total_supply = EXCLUDED.total_supply,
			is_active = EXCLUDED.is_active,
			updated_at = now()
	`, t.ChainID, t.Address, t.Name, t.Symbol, t.Decimals, t.TotalSupply, t.IsActive)
	return err
}

func (r *TokenRepo) SetActive(ctx context.Context, chainID int64, address string, active bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tokens
		SET is_active = $3, updated_at = now()
		WHERE chain_id = $1 AND address = $2
	`, chainID, address, active)
	return err
}

func (r *TokenRepo) Get(ctx context.Context, chainID int64, address string) (model.Token, error) {
	var t model.Token
	err := r.db.QueryRow(ctx, `
		SELECT chain_id, address, name, symbol, decimals, total_supply, is_active, created_at, updated_at
		FROM tokens
		WHERE chain_id = $1 AND address = $2
	`, chainID, address).Scan(&t.ChainID, &t.Address, &t.Name, &t.Symbol, &t.Decimals, &t.TotalSupply, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *TokenRepo) List(ctx context.Context, chainID int64) ([]model.Token, error) {
	rows, err := r.db.Query(ctx, `
		SELECT chain_id, address, name, symbol, decimals, total_supply, is_active, created_at, updated_at
		FROM tokens
		WHERE chain_id = $1
		ORDER BY address
	`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Token
	for rows.Next() {
		var t model.Token
		if err := rows.Scan(&t.ChainID, &t.Address, &t.Name, &t.Symbol, &t.Decimals, &t.TotalSupply, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TokenRepo) ListActive(ctx context.Context, chainID int64) ([]model.Token, error) {
	rows, err := r.db.Query(ctx, `
		SELECT chain_id, address, name, symbol, decimals, total_supply, is_active, created_at, updated_at
		FROM tokens
		WHERE chain_id = $1 AND is_active = true
		ORDER BY address
	`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Token
	for rows.Next() {
		var t model.Token
		if err := rows.Scan(&t.ChainID, &t.Address, &t.Name, &t.Symbol, &t.Decimals, &t.TotalSupply, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
