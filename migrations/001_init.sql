CREATE TABLE IF NOT EXISTS tokens (
  chain_id BIGINT NOT NULL,
  address VARCHAR(42) NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  symbol TEXT NOT NULL DEFAULT '',
  decimals SMALLINT NOT NULL DEFAULT 0,
  total_supply NUMERIC(78, 0) NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, address),
  CONSTRAINT chk_tokens_address CHECK (address ~ '^0x[0-9a-f]{40}$')
);

CREATE TABLE IF NOT EXISTS erc20_transfers (
  chain_id BIGINT NOT NULL,
  token_address VARCHAR(42) NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash VARCHAR(66) NOT NULL,
  tx_hash VARCHAR(66) NOT NULL,
  log_index INTEGER NOT NULL,
  from_address VARCHAR(42) NOT NULL,
  to_address VARCHAR(42) NOT NULL,
  value NUMERIC(78, 0) NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, tx_hash, log_index),
  CONSTRAINT chk_transfers_token_address CHECK (token_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_transfers_block_hash CHECK (block_hash ~ '^0x[0-9a-f]{64}$'),
  CONSTRAINT chk_transfers_tx_hash CHECK (tx_hash ~ '^0x[0-9a-f]{64}$'),
  CONSTRAINT chk_transfers_from_address CHECK (from_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_transfers_to_address CHECK (to_address ~ '^0x[0-9a-f]{40}$')
);

CREATE TABLE IF NOT EXISTS erc20_approvals (
  chain_id BIGINT NOT NULL,
  token_address VARCHAR(42) NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash VARCHAR(66) NOT NULL,
  tx_hash VARCHAR(66) NOT NULL,
  log_index INTEGER NOT NULL,
  owner_address VARCHAR(42) NOT NULL,
  spender_address VARCHAR(42) NOT NULL,
  value NUMERIC(78, 0) NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, tx_hash, log_index),
  CONSTRAINT chk_approvals_token_address CHECK (token_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_approvals_block_hash CHECK (block_hash ~ '^0x[0-9a-f]{64}$'),
  CONSTRAINT chk_approvals_tx_hash CHECK (tx_hash ~ '^0x[0-9a-f]{64}$'),
  CONSTRAINT chk_approvals_owner_address CHECK (owner_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_approvals_spender_address CHECK (spender_address ~ '^0x[0-9a-f]{40}$')
);

CREATE TABLE IF NOT EXISTS erc20_balances (
  chain_id BIGINT NOT NULL,
  token_address VARCHAR(42) NOT NULL,
  holder_address VARCHAR(42) NOT NULL,
  balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
  updated_block_number BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, token_address, holder_address),
  CONSTRAINT chk_balances_token_address CHECK (token_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_balances_holder_address CHECK (holder_address ~ '^0x[0-9a-f]{40}$')
);

CREATE TABLE IF NOT EXISTS erc20_allowances (
  chain_id BIGINT NOT NULL,
  token_address VARCHAR(42) NOT NULL,
  owner_address VARCHAR(42) NOT NULL,
  spender_address VARCHAR(42) NOT NULL,
  allowance NUMERIC(78, 0) NOT NULL DEFAULT 0,
  updated_block_number BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, token_address, owner_address, spender_address),
  CONSTRAINT chk_allowances_token_address CHECK (token_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_allowances_owner_address CHECK (owner_address ~ '^0x[0-9a-f]{40}$'),
  CONSTRAINT chk_allowances_spender_address CHECK (spender_address ~ '^0x[0-9a-f]{40}$')
);

CREATE TABLE IF NOT EXISTS sync_state (
  chain_id BIGINT NOT NULL,
  token_address VARCHAR(42) NOT NULL,
  last_indexed_block BIGINT NOT NULL,
  finalized_block BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, token_address),
  CONSTRAINT chk_sync_state_token_address CHECK (token_address ~ '^0x[0-9a-f]{40}$')
);

CREATE TABLE IF NOT EXISTS indexed_blocks (
  chain_id BIGINT NOT NULL,
  block_number BIGINT NOT NULL,
  block_hash VARCHAR(66) NOT NULL,
  parent_hash VARCHAR(66) NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, block_number),
  CONSTRAINT chk_blocks_block_hash CHECK (block_hash ~ '^0x[0-9a-f]{64}$'),
  CONSTRAINT chk_blocks_parent_hash CHECK (parent_hash ~ '^0x[0-9a-f]{64}$')
);

CREATE INDEX IF NOT EXISTS idx_tokens_active
  ON tokens(chain_id, is_active);

CREATE INDEX IF NOT EXISTS idx_transfers_token_block
  ON erc20_transfers(token_address, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_transfers_from
  ON erc20_transfers(token_address, from_address, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_transfers_to
  ON erc20_transfers(token_address, to_address, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_transfers_tx
  ON erc20_transfers(token_address, tx_hash);

CREATE INDEX IF NOT EXISTS idx_approvals_owner
  ON erc20_approvals(token_address, owner_address, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_approvals_spender
  ON erc20_approvals(token_address, spender_address, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_balances_holder
  ON erc20_balances(token_address, holder_address);

CREATE INDEX IF NOT EXISTS idx_balances_token_balance
  ON erc20_balances(token_address, balance DESC);
