# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

```bash
# Build both binaries
go build ./cmd/indexer
go build ./cmd/api

# Run (requires PostgreSQL + Ethereum RPC)
CONFIG_PATH=configs/config.yaml go run ./cmd/indexer
CONFIG_PATH=configs/config.yaml go run ./cmd/api

# Run a single test by name pattern (when tests are added)
go test ./internal/... -run TestName -v
```

No Makefile or task runner exists. Configuration is via YAML file path in `CONFIG_PATH` env var (defaults to `configs/config.yaml`). All config values can be overridden with env vars (e.g., `CHAIN_RPC_URL`, `DATABASE_DSN`) thanks to viper's `AutomaticEnv`.

## Architecture

Two independent binaries share the same Postgres database:

### `cmd/indexer` — Blockchain Event Indexer

Polls an EVM-compatible RPC endpoint for ERC-20 `Transfer` and `Approval` events, then materializes balances and allowances in Postgres.

**Startup flow:** `app.RunIndexer` wires `chain.Client` (ethclient wrapper) + `storage.Connect` (pgxpool) → `indexer.Syncer.Run`:
1. **Bootstrap** — reads configured token addresses from config, fetches metadata (name/symbol/decimals/totalSupply) on-chain, upserts tokens + initializes `sync_state` rows
2. **Poll loop** — on each tick (default 10s):
   - Compute `safeBlock = latest - confirmations` to avoid reorgs
   - **Reorg check** — `ReorgDetector.FindRollbackBlock` walks backward from the last indexed block comparing local block hashes against the RPC. On mismatch, deletes all data after the last common ancestor and rebuilds balances/allowances from the surviving transfers/approvals
   - For each active token, scan the batch range `[lastIndexed+1, safeBlock]`:
     - `Scanner` fetches Transfer + Approval logs via `eth_getLogs`, rate-limited per config
     - `Decoder` checks topic hashes and extracts event fields
     - Fetches block headers in parallel to get timestamps (used on decoded events)
     - `BatchApplier` inserts transfers/approvals with `ON CONFLICT DO NOTHING` (idempotent), then applies balance deltas and allowance updates only for newly inserted rows
     - **Adaptive batching** (`BatchSizer`): starts at `batch_size` (1000), doubles after 3 consecutive successes, halves on failure, clamped to `[min_batch_size, max_batch_size]`

### `cmd/api` — REST Query API

Serves indexed data via `chi` router on the configured `api.addr` (default `:8080`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/tokens` | List all tokens |
| POST | `/tokens` | Register a new token (fetches metadata, activates indexing) |
| GET | `/tokens/{address}` | Get token details |
| DELETE | `/tokens/{address}` | Soft-deactivate (sets `is_active=false`) |
| GET | `/tokens/{address}/transfers?holder=&limit=&offset=` | Paginated transfers, optionally filtered by holder |
| GET | `/tokens/{address}/balances/{holder}` | Single balance |
| GET | `/tokens/{address}/holders?limit=&offset=` | Top holders by balance (non-zero only) |
| GET | `/tokens/{address}/approvals?owner=&spender=&limit=&offset=` | Paginated approvals |
| GET | `/sync/status` | Per-token indexing progress |

Pagination defaults to 50, max 500. Addresses are normalized to lowercase hex. The API has its own RPC client (for token registration) and DB pool, independent from the indexer.

### Package Map

```
cmd/indexer        — indexer entrypoint (note: typo'd filename "mian.go")
cmd/api            — API server entrypoint
internal/app       — DI wiring (RunIndexer, NewAPIServer)
internal/indexer   — core sync logic: Syncer, Scanner, Decoder, BatchApplier, BatchSizer, ReorgDetector
internal/chain     — ethclient wrappers with rate limiting + retry: Client, BlockFetcher, LogFetcher, ERC20Metadata
internal/storage   — postgres access: Connect, Repositories aggregate, WithTx helper, individual repos
internal/model     — plain structs: Token, Transfer, Approval, Balance, IndexedBlock, SyncState
internal/api       — chi router + HTTP handlers + DTOs
internal/config    — viper-based YAML config loading
internal/logger    — zerolog initializer (stdout, RFC3339 timestamps)
pkg/ethutil        — shared helpers: address normalization, big.Int ↔ string conversion
migrations         — raw SQL schema (single migration, 7 tables + indexes)
configs            — example config (Sepolia testnet)
```

### Database Design

- `tokens` — registered ERC-20 tokens (PK: chain_id, address)
- `erc20_transfers` — raw Transfer events (PK: chain_id, tx_hash, log_index)
- `erc20_approvals` — raw Approval events (PK: chain_id, tx_hash, log_index)
- `erc20_balances` — materialized balance per holder/token (PK: chain_id, token_address, holder_address)
- `erc20_allowances` — materialized allowance per owner/spender/token (PK: chain_id, token_address, owner_address, spender_address)
- `sync_state` — per-token indexing cursor (PK: chain_id, token_address)
- `indexed_blocks` — block hash cache for reorg detection (PK: chain_id, block_number)

All numeric values (balances, approvals, total_supply) use `NUMERIC(78,0)` for arbitrary precision. Addresses are stored as lowercase hex with regex constraints.

### Key Design Decisions

- **Continuation-based sync**: progress tracked per-token in `sync_state.last_indexed_block` — restarts are safe
- **Confirmation depth**: only indexes blocks that are `confirmations` blocks behind the chain tip (default 24)
- **Reorg safety**: block headers are cached in `indexed_blocks` for hash comparison; on detection, data is deleted past the rollback point and balances/allowances are fully rebuilt from remaining events
- **No migration tool**: the single SQL migration is applied manually or by an operator; no goose/golang-migrate dependency
- **Null address handling**: transfers to/from the zero address are excluded from balance computation (mint/burn)
- **Allowance dedup**: when multiple approvals occur in the same batch, only the latest (by block_number then log_index) is applied per (chain_id, token, owner, spender)

### Known Code Issues

The codebase has several typos that affect runtime behavior:
- `ON CONCLICT` (missing F) in `batch_applier.go` lines 49 and 82 — causes SQL errors
- `UINON ALL` (should be `UNION ALL`) and `block_nubmer` (should be `block_number`) in `balance_repo.go:92,89` RebuildFromTransfer — breaks balance rebuilds during reorg handling
- `addrss` instead of `address` in `token_repo.go` SELECT columns (lines 39, 48, 71) — the Scan targets are also misaligned; this means token queries may return incorrect data
- `update_at` instead of `updated_at` in `sync_state_repo.go:53` — upserts fail to set the correct column
- `erc20_transfer` instead of `erc20_transfers` in `transfer_repo.go:56` — DeleteAfterBlock targets a table that doesn't exist

These are actual bugs, not cosmetic — several affect core query/upsert paths.
