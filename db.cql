-- Create the account table
CREATE TABLE IF NOT EXISTS account (
    id UUID PRIMARY KEY,
    first_name text,
    last_name text,
    email text,
    created_at timestamp,         -- account creation timestamp
    updated_at timestamp          -- last update timestamp
);
