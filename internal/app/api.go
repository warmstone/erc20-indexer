package app

import (
	"context"
	"net/http"

	apihttp "erc20-indexer/internal/api"
	"erc20-indexer/internal/chain"
	"erc20-indexer/internal/config"
	"erc20-indexer/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type APIServer struct {
	httpServer *http.Server
	db         *pgxpool.Pool
	chain      *chain.Client
}

func NewAPIServer(ctx context.Context, cfg config.Config) (*APIServer, error) {
	db, err := storage.Connect(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	chainClient, err := chain.Dial(ctx, cfg.Chain.RPCURL)
	if err != nil {
		db.Close()
		return nil, err
	}
	repos := storage.NewRepositories(db)
	handler := apihttp.NewHandler(cfg.Chain.ChainID, cfg.Chain.StartBlock, repos, chainClient)
	return &APIServer{
		httpServer: &http.Server{
			Addr:    cfg.API.Addr,
			Handler: apihttp.NewRouter(handler),
		},
		db:    db,
		chain: chainClient,
	}, nil
}

func (s *APIServer) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *APIServer) Shutdown(ctx context.Context) error {
	err := s.httpServer.Shutdown(ctx)
	s.chain.Close()
	s.db.Close()
	return err
}
