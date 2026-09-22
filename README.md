# Yasha

Small banking API written in Go with PostgreSQL-backed accounts and an append-only ledger.

PostgreSQL is currently used as the local persistence layer for simplicity. The remaining roadmap—Kafka events, balance workers, reconciliation, observability, and Kubernetes deployment—continues unchanged.

## Local development

Requirements: Docker Compose and Go 1.24+.

```bash
make local_dev
go run .
```

The local stack provides:

- PostgreSQL at `localhost:5432` (`database=yasha`, `user=yasha`, `password=yasha`)
- Adminer at [http://localhost:8080](http://localhost:8080)

Use these values to log in to Adminer:

```text
System: PostgreSQL
Server: postgres
Username: yasha
Password: yasha
Database: yasha
```

The schema in [`schema.sql`](./schema.sql) is applied automatically the first time the PostgreSQL volume is created. To reset local data:

```bash
make local_reset
make local_dev
```

The API uses `DATABASE_URL` when provided; the default is `postgres://yasha:yasha@localhost:5432/yasha?sslmode=disable`.

## API

- `POST /account` creates an account.
- `GET /account/:id` reads an account.
- `POST /transaction` records a deposit, withdrawal, or atomic transfer.
- `GET /account/:id/balance` calculates the balance from the ledger.

Amounts are integer minor units (for example, cents or pence), never floating point values.

## Checks

```bash
go test ./...
```

## Roadmap

- Kafka events for ledger entries and balance updates
- Materialized balance worker and reconciliation
- Statement export, anti-fraud rules, and external payment simulation
- Observability with metrics, logs, and tracing
- Kubernetes deployment and integration testing
