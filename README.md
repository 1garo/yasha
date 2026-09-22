# Yasha

Yasha is a small production-oriented banking backend written in Go. It supports accounts, balances, an append-only ledger, deposits, withdrawals, and transfers.

PostgreSQL is currently used as the local persistence layer for simplicity. The rest of the roadmap remains focused on event-driven processing, observability, reconciliation, and scalable infrastructure.

## Local development

Requirements: Docker Compose and Go 1.24+.

```bash
make local_dev
go run .
```

The local stack provides:

- PostgreSQL at `localhost:5432` (`database=yasha`, `user=yasha`, `password=yasha`)
- Adminer at [http://localhost:8080](http://localhost:8080)

Adminer connection values:

```text
System: PostgreSQL
Server: postgres
Username: yasha
Password: yasha
Database: yasha
```

The schema in [`schema.sql`](./schema.sql) is applied automatically when the PostgreSQL volume is created. To reset local data:

```bash
make local_reset
make local_dev
```

The API uses `DATABASE_URL` when provided. The default is:

```text
postgres://yasha:yasha@localhost:5432/yasha?sslmode=disable
```

## Current API

- `POST /account` creates an account.
- `GET /account/:id` reads an account.
- `POST /transaction` records a deposit, withdrawal, or transfer.
- `GET /account/:id/balance` calculates the account balance from the ledger.

Amounts are represented as integer minor units, such as cents or pence, never floating-point values.

## Project goals

The project is also a learning and interview-oriented system-design exercise. The main goals are:

- Build a banking backend with clear domain boundaries.
- Treat the ledger as the canonical, append-only source of truth.
- Practice idempotency, consistency, resilience, and auditability.
- Add asynchronous processing and materialized views.
- Add production-style observability and operational tooling.
- Run the system locally with Docker and eventually deploy it to Kubernetes.

## Architecture direction

The planned architecture consists of:

- Go HTTP API for validation, idempotency, and ledger writes.
- PostgreSQL for the current ledger and account persistence.
- Kafka for asynchronous `ledger.entries` and `balance.updated` events.
- A balance worker for materialized balance updates.
- A reconciliation worker that compares balances with the ledger.
- Optional fraud, notification, settlement, and analytics consumers.
- Envoy or another edge proxy for routing and future mTLS experiments.
- Prometheus, Grafana, OpenTelemetry, and structured logs for observability.

For a transfer, the API should write a debit and credit entry, publish a ledger event, and let downstream workers update derived balances.

## Roadmap

### Completed foundation

- [x] Account creation and account lookup.
- [x] Deposits, withdrawals, and transfers.
- [x] Append-only ledger persistence.
- [x] PostgreSQL local development stack.
- [x] Adminer database browser.
- [x] Feature-based HTTP handlers under `internal/`.

### Next milestones

1. **Currency model**

   Decide whether accounts use one fixed currency, support multiple currency balances, support currency conversion, or combine these approaches. Track this separately in [issue #1](https://github.com/1garo/yasha/issues/1).

2. **Ledger hardening**

   Add idempotency keys, stronger account and transaction validation, insufficient-funds checks, and reliable transfer invariants.

3. **Kafka and balance worker**

   Publish `ledger.entries` events and update materialized balances asynchronously.

4. **Reconciliation**

   Recompute balances from the ledger and report differences from materialized balances.

5. **Balance events and integrations**

   Publish `balance.updated` events and add optional notification, fraud, analytics, and settlement consumers.

6. **Observability**

   Add metrics, structured logging, distributed tracing, dashboards, and useful operational alerts.

7. **Infrastructure and delivery**

   Add Docker images, Kubernetes manifests, integration tests, CI, and local deployment workflows.

8. **Stretch features**

   Add statement export, a simple anti-fraud rule, an external payments simulator, and a small demonstration UI.

## Testing strategy

- Unit tests for money calculations, validation, and idempotency.
- PostgreSQL integration tests for repositories and transactions.
- Contract tests for Kafka producers and consumers.
- Reconciliation tests comparing ledger totals with materialized balances.
- Race and concurrency tests for transfers and worker processing.
- Small load-smoke tests for the API and event flow.

## Checks

```bash
go test ./...
```
