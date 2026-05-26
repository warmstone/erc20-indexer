package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.RequestID)
	r.Get("/health", handler.Health)
	r.Get("/tokens", handler.ListTokens)
	r.Post("/tokens", handler.RegisterToken)
	r.Get("/tokens/{address}", handler.GetToken)
	r.Delete("/tokens/{address}", handler.UnregisterToken)
	r.Get("/tokens/{address}/transfers", handler.ListTransfers)
	r.Get("/tokens/{address}/balances/{holder}", handler.GetBalance)
	r.Get("/tokens/{address}/holders", handler.ListHolders)
	r.Get("/tokens/{address}/approvals", handler.ListApprovals)
	r.Get("/sync/status", handler.SyncStatus)
	return r
}
