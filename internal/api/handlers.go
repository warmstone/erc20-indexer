package api

import (
	"encoding/json"
	"erc20-indexer/internal/chain"
	"erc20-indexer/internal/model"
	"erc20-indexer/internal/storage"
	"erc20-indexer/pkg/ethutil"
	"errors"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	chainID     int64
	startBlock  uint64
	repos       storage.Repositories
	chainClient *chain.Client
}

func NewHandler(chainID int64, startBlock uint64, repos storage.Repositories, chainClient *chain.Client) *Handler {
	return &Handler{chainID: chainID, startBlock: startBlock, repos: repos, chainClient: chainClient}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := h.repos.Tokens.List(r.Context(), h.chainID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]TokenDTO, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, tokenDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	token := ethutil.NormalizeAddressString(chi.URLParam(r, "address"))
	t, err := h.repos.Tokens.Get(r.Context(), h.chainID, token)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenDTO(t))
}

func (h *Handler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	addr := common.HexToAddress(req.Address)
	if addr == (common.Address{}) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid token address"})
		return
	}

	meta, err := h.chainClient.ERC20Metadata(r.Context(), addr)
	if err != nil {
		writeError(w, err)
		return
	}
	token := model.Token{
		ChainID:     h.chainID,
		Address:     ethutil.NormalizeAddress(addr),
		Name:        meta.Name,
		Symbol:      meta.Symbol,
		Decimals:    meta.Decimals,
		TotalSupply: ethutil.BigIntString(meta.TotalSupply),
		IsActive:    true,
	}
	if err := h.repos.Tokens.Upsert(r.Context(), token); err != nil {
		writeError(w, err)
		return
	}
	if _, err := h.repos.SyncState.GetOrCreate(r.Context(), h.chainID, token.Address, h.startBlock); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tokenDTO(token))
}

func (h *Handler) UnregisterToken(w http.ResponseWriter, r *http.Request) {
	token := ethutil.NormalizeAddressString(chi.URLParam(r, "address"))
	if err := h.repos.Tokens.SetActive(r.Context(), h.chainID, token, false); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unregistered"})
}

func (h *Handler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	token := ethutil.NormalizeAddressString(chi.URLParam(r, "address"))
	holder := normalizeQueryAddress(r, "holder")
	limit, offset := pagination(r)
	transfers, err := h.repos.Transfers.List(r.Context(), h.chainID, token, holder, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]TransferDTO, 0, len(transfers))
	for _, t := range transfers {
		out = append(out, TransferDTO{
			TokenAddress: t.TokenAddress,
			From:         t.FromAddress,
			To:           t.ToAddress,
			Value:        t.Value,
			TxHash:       t.TxHash,
			LogIndex:     t.LogIndex,
			BlockNumber:  t.BlockNumber,
			Timestamp:    t.Timestamp,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	token := ethutil.NormalizeAddressString(chi.URLParam(r, "address"))
	holder := ethutil.NormalizeAddressString(chi.URLParam(r, "holder"))
	b, err := h.repos.Balances.Get(r.Context(), h.chainID, token, holder)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, balanceDTO(b))
}

func (h *Handler) ListHolders(w http.ResponseWriter, r *http.Request) {
	token := ethutil.NormalizeAddressString(chi.URLParam(r, "address"))
	limit, offset := pagination(r)
	balances, err := h.repos.Balances.Holders(r.Context(), h.chainID, token, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]BalanceDTO, 0, len(balances))
	for _, b := range balances {
		out = append(out, balanceDTO(b))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	token := ethutil.NormalizeAddressString(chi.URLParam(r, "address"))
	owner := normalizeQueryAddress(r, "owner")
	spender := normalizeQueryAddress(r, "spender")
	limit, offset := pagination(r)
	approvals, err := h.repos.Approvals.List(r.Context(), h.chainID, token, owner, spender, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]ApprovalDTO, 0, len(approvals))
	for _, a := range approvals {
		out = append(out, ApprovalDTO{
			TokenAddress: a.TokenAddress,
			Owner:        a.OwnerAddress,
			Spender:      a.SpenderAddress,
			Value:        a.Value,
			TxHash:       a.TxHash,
			LogIndex:     a.LogIndex,
			BlockNumber:  a.BlockNumber,
			Timestamp:    a.Timestamp,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	states, err := h.repos.SyncState.List(r.Context(), h.chainID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]SyncStateDTO, 0, len(states))
	for _, s := range states {
		out = append(out, SyncStateDTO{
			TokenAddress:     s.TokenAddress,
			LastIndexedBlock: s.LastIndexedBlock,
			FinalizedBlock:   s.FinalizedBlock,
			UpdatedAt:        s.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func tokenDTO(t model.Token) TokenDTO {
	return TokenDTO{
		ChainID:     t.ChainID,
		Address:     t.Address,
		Name:        t.Name,
		Symbol:      t.Symbol,
		Decimals:    t.Decimals,
		TotalSupply: t.TotalSupply,
		IsActive:    t.IsActive,
		UpdatedAt:   t.UpdatedAt,
	}
}

func balanceDTO(b model.Balance) BalanceDTO {
	return BalanceDTO{
		TokenAddress:       b.TokenAddress,
		Holder:             b.HolderAddress,
		Balance:            b.Balance,
		UpdatedBlockNumber: b.UpdatedBlockNumber,
		UpdatedAt:          b.UpdatedAt,
	}
}

func pagination(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if offset > 10000 {
		offset = 10000
	}
	return limit, offset
}

func normalizeQueryAddress(r *http.Request, key string) string {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return ""
	}
	return ethutil.NormalizeAddressString(raw)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
