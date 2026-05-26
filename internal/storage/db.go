package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

type Repositories struct {
	Tokens    *TokenRepo
	Transfers *TransferRepo
	Approvals *ApprovalRepo
	Balances  *BalanceRepo
	SyncState *SyncStateRepo
	Blocks    *BlockRepo
}

func NewRepositories(db DBTX) Repositories {
	return Repositories{
		Tokens:    &TokenRepo{db: db},
		Transfers: &TransferRepo{db: db},
		Approvals: &ApprovalRepo{db: db},
		Balances:  &BalanceRepo{db: db},
		SyncState: &SyncStateRepo{db: db},
		Blocks:    &BlockRepo{db: db},
	}
}
