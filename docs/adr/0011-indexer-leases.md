# ADR-0011: Indexer work claiming with leases

**Status:** Accepted

## Context
The RAG indexer worker takes a `document_versions` row, calls the gateway (embeddings, deadline extraction; seconds to minutes) and writes results. The first draft said "claim with `SELECT … FOR UPDATE SKIP LOCKED`", which is only safe if the transaction commits immediately; holding it across gateway calls would pin a connection and a row lock for minutes, and a crash would leave the state ambiguous.

## Decision
- **Claim = one short statement that commits at once**, setting a *lease*: `index_status='indexing'`, `claimed_at=now()`, `claimed_by=<worker id>`, `index_attempts+1`. Selection uses `FOR UPDATE SKIP LOCKED` only inside that statement. **No transaction or row lock is held while calling the gateway.**
- **Lease expiry:** a version in `indexing` whose `claimed_at` is older than `INDEXER_LEASE_TIMEOUT` (default 10 min) is runnable again and can be claimed by any worker. Long versions extend the lease with a heartbeat (`INDEXER_HEARTBEAT`, default 2 min) via `ExtendLease`.
- **Fencing:** every worker write (`CompleteIndexing`, `ReleaseForRetry`, `FailIndexing`, the promotion transaction) includes `claimed_by = $worker AND index_status = 'indexing'`. Zero rows affected means "lease lost": the worker discards its result. Embedding and deadline writes are idempotent.
- **Retries:** a transient error → `pending` with `index_next_attempt_at = now() + base·2^(attempts−1)` (`INDEXER_BACKOFF_BASE` 30 s, capped by `INDEXER_BACKOFF_MAX`). Attempts are counted **at claim time**, so a poison version that crashes its worker is also bounded. After `INDEXER_MAX_ATTEMPTS` (default 5) the version becomes terminal `failed` with `index_error`, and an alert fires (`CountIndexerBacklog` feeds the gauges). A permanent error (e.g. unparseable content) can fail immediately.
- Schema (migration 00008): `claimed_at`, `claimed_by`, `index_next_attempt_at`, status `indexing` (renamed from `in_progress`), a CHECK that `indexing` implies owner and timestamp, partial index `document_versions_claim_idx`.
- Queries live in `internal/store/queries/indexer.sql`.

## Alternatives
Advisory locks per version (released automatically when the connection dies, but they need a dedicated connection held for the whole job and give no visibility or retry accounting); an external queue (violates ADR-0007: no extra infrastructure).

## Consequences
- Worker death never wedges a version; recovery is automatic within the lease timeout.
- At-least-once processing: the same version can be processed twice if a worker is slow but alive (heartbeat prevents most cases); fencing plus idempotent writes make this safe. The lease timeout must exceed the heartbeat interval with margin.
- Workers must use a stable unique id (hostname + pid + start time) so fencing distinguishes restarts.
- Behaviour verified against live Postgres (fresh lease not claimable, expired lease taken over, late completion = 0 rows, backoff respected, exhaustion → `failed`).
