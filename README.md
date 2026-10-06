# Yasha

Yasha is a small production-oriented banking backend written in Go. It supports accounts, balances, an append-only ledger, deposits, withdrawals, and transfers.

PostgreSQL is currently used as the local persistence layer for simplicity. The rest of the roadmap remains focused on event-driven processing, observability, reconciliation, and scalable infrastructure.

## Project goals

The project is also a learning and interview-oriented system-design exercise. The main goals are:

- Build a banking backend with clear domain boundaries.
- Treat the ledger as the canonical, append-only source of truth.
- Practice idempotency, consistency, resilience, and auditability.
- Add asynchronous processing and materialized views.
- Add production-style observability and operational tooling.
- Run the system locally with Docker and eventually deploy it to Kubernetes.

## Scope and architecture

### MVP capabilities

- Accounts and account lookup.
- Available and ledger balances.
- Immutable ledger entries as the canonical source of truth.
- Deposit, withdrawal, and transfer APIs.
- Kafka events for `ledger.entries`.
- A reconciliation worker that compares the ledger with cached balances.

### Planned architecture

- **API:** Go HTTP service responsible for validation, idempotency, and ledger writes.
- **Persistence:** PostgreSQL for the current local ledger and account storage.
- **Event bus:** Kafka topics for `ledger.entries` and optionally `balance.updated`.
- **Workers:** Balance materialization, reconciliation, and future settlement processing.
- **Consumers:** Optional notification, fraud, analytics, and external-payment services.
- **Proxy:** Envoy for HTTP routing and future mTLS experiments.
- **Observability:** Structured logs, Prometheus metrics, OpenTelemetry tracing, and Grafana dashboards.

Transfer flow:

1. The API receives and validates a request.
2. The API writes a debit and credit ledger entry.
3. The API publishes a `ledger.entries` event.
4. The balance worker updates the materialized balance.
5. Other consumers can react to `balance.updated` without querying the ledger.


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

## Roadmap

Each milestone should produce something visible and runnable while keeping the scope small.

### Milestone 1 — Project scaffold and Accounts API

- [x] Set up Go modules, Makefile, Docker Compose, and local development workflow.
- [x] Implement account creation and account lookup.
- [x] Run the service locally against PostgreSQL.
- [x] Browse local data through Adminer.

### Milestone 2 — Ledger and deposits

- [x] Implement the append-only ledger schema and persistence.
- [x] Add deposit transactions.
- [x] Calculate balances directly from ledger entries.
- [x] Keep money in integer minor units rather than floating-point values.

### Milestone 3 — Transfers and withdrawals

- [x] Add withdrawal transactions.
- [x] Add transfers between accounts.
- [x] Represent transfers with two ledger entries: one debit and one credit.
- [ ] Add insufficient-funds validation and stronger transfer invariants.

### Milestone 3.1 — HTTP handler structure

- [x] Move handlers out of `main.go` into feature packages under `internal/`.
- [x] Keep route registration in the server package.
- [x] Add focused handler.

### Milestone 3.2 — Safe controller errors and logging

- [x] Add structured logging around unexpected controller and database failures, including enough request context to investigate incidents later.
- [x] Map internal failures, such as a failed query in `CreateAccountHandler`, to a safe default HTTP status and user-facing message.
- [x] Keep database errors and other implementation details out of API responses.

### Milestone 3.3 — Currency-aware balances

- [x] Accounts support balances in multiple currencies.
- [x] Currency conversion is not implicit; balances are kept separate by currency.
- [x] Deposits, withdrawals, and transfers validate that the transaction currency is supported.
- [x] Existing mixed-currency ledger entries are preserved and calculated independently.

See [issue #1](https://github.com/1garo/yasha/issues/1) for the design discussion.

### Milestone 4 — Kafka and balance worker

- [ ] Add a Kafka topic for `ledger.entries`.
- [ ] Publish an event after a successful ledger transaction.
- [ ] Implement a worker that consumes ledger events and updates materialized balances.
- [ ] Update account reads to use the balance projection where appropriate.
- [ ] Demonstrate the first asynchronous balance update flow.

### Milestone 5 — Reconciliation

- [ ] Recompute balances from the ledger.
- [ ] Compare computed balances with materialized balances.
- [ ] Report discrepancies with enough context to investigate them.
- [ ] Add a runnable reconciliation command or scheduled worker.

### Milestone 6 — Balance events and integrations

- [ ] Publish `balance.updated` after a balance projection changes.
- [ ] Add a small notification consumer as a demonstration.
- [ ] Leave room for fraud, analytics, and settlement consumers.

### Milestone 7 — Observability and infrastructure

- [ ] Add request, database, and worker metrics.
- [ ] Add structured logs with request and transaction correlation IDs.
- [ ] Add OpenTelemetry traces across HTTP, database, and Kafka operations.
- [ ] Add Prometheus/Grafana dashboards and useful alerts.
- [ ] Add Docker images and Kubernetes manifests for local deployment.
- [ ] Add CI checks and integration tests against PostgreSQL and Kafka.

### Stretch features

- [ ] Export account statements as CSV.
- [ ] Add a simple anti-fraud rule, such as blocking unusually large activity in a short window.
- [ ] Add a simulated external payments connector.
- [ ] Add a small UI for creating accounts and making transfers.

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
