-- schema.cql
-- Monzo MVP Cassandra schema

-- 1. Create keyspace for the project
CREATE KEYSPACE IF NOT EXISTS bank
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

USE bank;

-- 2. Accounts table
-- Represents a customer's account (like a Monzo user wallet)
CREATE TABLE IF NOT EXISTS accounts (
  account_id UUID PRIMARY KEY,
  full_name TEXT,
  email TEXT,
  currency TEXT,          -- e.g. GBP, USD
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- 3. Ledger Entries table
-- Immutable list of all financial movements. Every transaction produces two entries:
-- one debit and one credit (double-entry accounting).
CREATE TABLE IF NOT EXISTS ledger_entries (
  account_id UUID,
  entry_id UUID,
  transaction_id UUID,
  direction TEXT,         -- 'credit' or 'debit'
  amount_minor BIGINT,    -- stored in minor currency units (e.g., pennies)
  currency TEXT,
  description TEXT,
  created_at TIMESTAMP,
  PRIMARY KEY ((account_id), created_at, entry_id)
) WITH CLUSTERING ORDER BY (created_at DESC);

-- Optional index to query by transaction_id quickly
CREATE INDEX IF NOT EXISTS ledger_by_txn_id ON ledger_entries (transaction_id);

-- 4. Balances table
-- Denormalized current balance for fast reads. Updated by a background worker
-- that consumes ledger_entries and recomputes the total.
CREATE TABLE IF NOT EXISTS balances (
  account_id UUID PRIMARY KEY,
  balance_minor BIGINT,
  currency TEXT,
  updated_at TIMESTAMP
);

-- 5. Transactions table (optional for MVP but useful)
-- Logical grouping of a transfer or payment involving multiple ledger entries.
CREATE TABLE IF NOT EXISTS transactions (
  transaction_id UUID PRIMARY KEY,
  from_account_id UUID,
  to_account_id UUID,
  amount_minor BIGINT,
  currency TEXT,
  description TEXT,
  created_at TIMESTAMP
);

-- 6. Audit table (stretch goal)
-- For tracing operational events, e.g. balance.updated or reconciliation actions.
CREATE TABLE IF NOT EXISTS audit_events (
  event_id UUID PRIMARY KEY,
  event_type TEXT,      -- e.g. 'balance.updated', 'transfer.initiated'
  payload TEXT,         -- JSON blob for flexibility
  created_at TIMESTAMP
);
