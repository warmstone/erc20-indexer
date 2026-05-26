package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"erc20-indexer/internal/app"
	"erc20-indexer/internal/config"
	"erc20-indexer/internal/logger"
)

func main() {
	log := logger.New()
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunIndexer(ctx, cfg, log); err != nil && ctx.Err() == nil {
		log.Fatal().Err(err).Msg("indexer stopped")
	}
}
