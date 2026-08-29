-- Create keyspace for your app
CREATE KEYSPACE IF NOT EXISTS yasha
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

-- Use the keyspace
USE yasha;

-- Accounts table used by POST /account and GET /account/:id
CREATE TABLE IF NOT EXISTS account (
    id UUID PRIMARY KEY,
    first_name text,
    last_name text,
    email text,
    created_at timestamp,         -- account creation timestamp
    updated_at timestamp          -- last update timestamp
);

-- Ledger entries used by POST /transaction and GET /account/:id/balance
CREATE TABLE IF NOT EXISTS ledger_entries (
    account_id UUID,
    entry_id UUID,
    transaction_id UUID,
    direction text,
    amount_minor bigint,
    currency text,
    description text,
    created_at timestamp,
    PRIMARY KEY ((account_id), created_at, entry_id)
) WITH CLUSTERING ORDER BY (created_at DESC);

-- Lookup table for entries grouped by transaction
CREATE TABLE IF NOT EXISTS ledger_entries_by_transaction (
    transaction_id UUID,
    account_id UUID,
    entry_id UUID,
    direction text,
    amount_minor bigint,
    currency text,
    description text,
    created_at timestamp,
    PRIMARY KEY ((transaction_id), account_id, entry_id)
);
