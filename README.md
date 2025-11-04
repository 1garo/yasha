# MVP Implementation Plan — Bank Operations (practice for Monzo Backend Engineer III)

> Goal: build a focused, runnable MVP that exercises the technologies, design patterns and operational concerns highlighted in Monzo's Backend Engineer III role. You'll practice building resilient, observable, event-driven banking primitives (accounts, ledger, transactions, reconciliations) using a stack and architecture inspired by Monzo.

---

## 1. Objectives & learning outcomes

* Implement a small, production-ish banking backend that supports: account creation, deposits, withdrawals, transfers, and a simple reconciliation job.
* Use Go for service code and idiomatic concurrency/structure.
* Use Kafka for async flows and Cassandra-like persistence (or local replacement for dev) to mirror Monzo's architecture.
* Containerise and run everything on Kubernetes (local: kind/minikube) with Envoy as an edge/proxy for gRPC/HTTP routing.
* Add observability: metrics, logs, structured tracing (OpenTelemetry) and simple chaos tests.
* Practice system-design answers that match Monzo's scale concerns (idempotency, resilience, data modeling for money, eventual consistency, auditability).

---

## 2. Scope (MVP)

**Core features (must-have):**

* Accounts (create, view)
* Balances (available and ledger balance)
* Ledger entries (immutable journal entries) — canonical source of truth
* Transactions API: create deposit/withdrawal/transfer
* Event publishing to Kafka for `ledger.entries`
* Reconciliation worker that verifies ledger vs cached balances

**Nice-to-have (stretch):**

* Publish `balance.updated` events when balances change (other services could consume it)
* Simple anti-fraud rule (e.g., block > X amount in 1-minute window)
* Statement export (CSV)
* Basic UI (React) for creating accounts and making transfers
* Simulated external payments connector (outgoing settlement)

---

## 3. High-level architecture

* **APIs**: HTTP/gRPC front-end service written in Go (handles validation, idempotency keys, and writes to ledger).
* **Ledger service**: single responsibility: append-only ledger (writes to Cassandra or equivalent). Exposes read/query endpoints.
* **Event bus**: Kafka topics for `ledger.entries`, and optionally `balance.updated`.
* **Workers**: consumers for `ledger.entries` to update balances (materialised views), reconciliation, and settlement simulator.
* **Proxy**: Envoy as sidecar/edge for routing and mutual TLS (simulated locally).
* **Infra**: Docker images, Kubernetes manifests (deployments, services), Helm/chart optional.

Flow for a transfer:

1. API receives request → writes **two ledger entries** (one debit, one credit).
2. Publishes `ledger.entries` event.
3. Balance worker consumes event → updates balances table → optionally publishes `balance.updated` event.
4. Other services (notifications, fraud, analytics) can subscribe to `balance.updated` without hitting DB.

---

## 4. Tech choices & local replacements

* **Language:** Go (modules, idiomatic packages)
* **Kafka:** Use `kafka` (local: `confluentinc/cp-kafka` or `bitnami/kafka`) for realistic testing
* **Cassandra:** Monzo uses Cassandra; for local dev use either Cassandra (Docker image) or replace with SQLite/Postgres for the ledger and mark differences. If you want to simulate wide-column modelling, use ScyllaDB/Cassandra image. Keep schema simple: partition by `account_id`; time-series of ledger entries.
* **Kubernetes:** `kind` or `minikube` for local cluster. Use `skaffold` for iteration if desired.
* **Envoy:** use a simple Envoy config for HTTP routing to the API service.
* **Observability:** OpenTelemetry + Prometheus + Grafana (or `prometheus` + `jaeger` docker images)
* **CI:** GitHub Actions to build/test images; integration test job runs against `kind` cluster.

---

## 5. Data model (suggested)

**LedgerEntry** (append-only)

```
id: uuid
account_id: uuid
counterparty_account_id: uuid | null
amount_minor: int64  # in cents/pennies, not float!
currency: string
type: enum {deposit, withdrawal, transfer}
created_at: timestamp
metadata: json
idempotency_key: string
```

> Why `amount_minor`? Money must be represented in **minor units** (like pennies or cents) using integers, never floats, to avoid rounding errors.

**Balances** (materialised view)

```
account_id: uuid
available_balance_minor: int64
ledger_balance_minor: int64
updated_at: timestamp
```

Key rules:

* Use **int64** to represent minor units (no floats).
* All operations are recorded as ledger entries; balances are derived.
* Writes should be idempotent (idempotency key + dedupe before insert).

---

## 6. APIs (surface area)

**POST /accounts** -> create account (returns id)
**GET /accounts/{id}** -> account + balances
**POST /transactions** -> body: {account\_id, type, amount\_minor, currency, idempotency\_key, counterparty}
**GET /transactions?account\_id=** -> list ledger entries

Behavioural notes to implement:

* Validate currency and amounts.
* Enforce idempotency using idempotency key table or ledger unique constraint.
* Return optimistic errors for insufficient funds.

---

## 7. Implementation milestones

Each milestone gives you **visible progress** — something to run and see working — while keeping scope small.

### Milestone 1 — Project scaffold & Accounts API (1–2 days)

* Setup repo (Go modules, Dockerfile, Makefile).
* Implement `POST /accounts` + `GET /accounts/{id}` (returns static balance = 0).
* Run container locally and test with `curl`.
* [x] First working feature: create and query accounts.

### Milestone 2 — Ledger & Deposits (2–3 days)

* Implement ledger schema & persistence.
* Add `POST /transactions` for **deposits only**.
* Balance is computed directly from ledger (no worker yet).
- [x] First money movement: deposit £10 and see it in account balance.

### Milestone 3 — Transfers & Withdrawals (2–3 days)

* Extend transactions API to support withdrawals and transfers.
* Implement two-entry logic for transfers (debit + credit).
- [x] You can now move money between accounts.

### Milestone 3.1 — Add handlers for endpoints (2–3 days)

* Each endpoint should go into a different folder
* Create a function like `InitServer`
- [ ] Can call endpoints but from handlers and not all in main.go.

### Milestone 4 — Kafka & Balance Worker (2–3 days)

* Produce `ledger.entries` to Kafka.
* Implement balance worker that consumes and updates materialised balances.
* Add `GET /accounts/{id}` to now read from balances table.
- [ ] First async flow: balances update via events, not direct DB queries.

### Milestone 5 — Reconciliation (2 days)

* Implement reconciliation worker to recompute balances from ledger and compare with materialised view.
- [ ] You can run a job and see a consistency check report.

### Milestone 6 — Balance Updated Events (Stretch, 2–3 days)

* After updating balances, publish `balance.updated` events.
* Optional demo consumer: notifications service logs “Balance changed for account X.”
- [ ] Shows event-driven design beyond core ledger.

### Milestone 7 — Observability & Infra Polish (2–3 days)

* Add OpenTelemetry, Prometheus metrics, and Grafana dashboard.
* Deploy services to `kind` or `minikube` with Envoy as gateway.
- [ ] See traces and metrics while hitting APIs.

---

## 8. Testing strategy

* Unit tests for money math and idempotency.
* Integration tests: spin up Kafka + DB + service in CI (use GitHub Actions with `services` or run in `kind`).
* Contract tests for workers: produce events and assert consumers update views.
* Load smoke tests (small JMeter / vegeta) to exercise concurrent transfers and idempotency.

---

## 9. Interview / take-home task mapping

Prepare the following artifacts to talk about in interviews:

* **Architecture diagram** (drawn in Lucid/diagrams.net)
* **Tradeoffs doc**: why append-only ledger, why choose Cassandra vs Postgres locally, consistency model, partitioning strategy for scale
* **Demo script**: simple script that creates accounts, posts transfers, shows reconciliation output
* **Tests & metrics**: show test coverage and Prometheus metrics
* **PR**: have a small, clean PR that implements a meaningful slice (ledger + transaction API) to show code style

---

## 10. Local dev & runbook (quick start)

1. Clone repo
2. `make deps && make build`
3. Start infra: `docker-compose up -d cassandra kafka zookeeper` (or `kind` + `kubectl apply -f k8s/`)
4. Run service: `./bin/service --config ./config.yaml`
5. Use `curl` or Postman to hit APIs. Run reconciliation worker: `./bin/reconciler`.

---

## 11. Deliverables for practice

* Repo with: `cmd/`, `internal/ledger`, `internal/api`, `internal/worker`, Dockerfiles, k8s manifests, Makefile
* README with architecture and demo script
* 3–5 useful Grafana/Prometheus dashboards
* A small recorded demo (optional) or detailed `DEMO.md` describing operations to run during interview

---

## 12. Next steps for me to tailor this to you

If you want, I can:

* Produce a detailed folder layout + starter Go code for the ledger and API (one endpoint fully implemented).
* Generate Kubernetes manifests and a docker-compose for the Kafka + DB stack.
* Create a sample GitHub Actions workflow to run integration tests.

Tell me which of the above you'd like next and I'll generate the code/manifest/snippets for you.

---

Good luck — this plan gives you a focused, interview-relevant project that mirrors Monzo's stack and operational concerns. Build it iteratively, keep the ledger immutable, and instrument everything for observability.

