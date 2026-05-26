package app

import (
	"context"
	"erc20-indexer/internal/chain"
	"erc20-indexer/internal/config"
	"erc20-indexer/internal/indexer"
	"erc20-indexer/internal/storage"

	"github.com/rs/zerolog"
)

func RunIndexer(ctx context.Context, cfg config.Config, log zerolog.Logger) error {
	client, err := chain.Dial(ctx, cfg.Chain.RPCURL)
	if err != nil {
		return err
	}
	defer client.Close()

	db, err := storage.Connect(ctx, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	return indexer.NewSyncer(cfg, client, db, log).Run(ctx)
}
