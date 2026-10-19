# ADR-0007: Stack conventions

**Status:** Accepted

- **Go ≥ 1.26**, `net/http` with Go 1.22+ method/pattern routing (no web framework), `log/slog`, `context` on every call that does I/O, graceful shutdown via `signal.NotifyContext` + `Server.Shutdown`. Streaming APIs use `iter.Seq2`.
- **pgx v5 + sqlc** for queries (typed, SQL stays SQL, easy to review); hand-written SQL for the two hybrid-search queries where needed. **goose** for migrations (plain SQL files, `-- +goose Up/Down`, sqlc understands them).
- **No Redis.** The gateway cache is an in-process LRU (temperature-0 requests only); a restart loses it, which costs money but not correctness. Persistent usage data goes to Postgres (`llm_usage`). Postgres is also the indexing queue (`SKIP LOCKED`). Revisit if the gateway ever runs multi-instance.
- **Config**: env vars for scalars and secrets; a YAML file (no secrets, env var *names* only) for gateway routing.
- **Scaffolding is stdlib-only** so it builds with no `go.sum`; dependencies arrive with the milestone that needs them.
- **Tests**: table-driven; provider interfaces are mocked; HTTP clients tested with `httptest`; DB tests use a real Postgres from compose (no DB mocks).
- **Lint**: `golangci-lint` v2 with a small, meaningful set (`errorlint`, `noctx`, `bodyclose`, `gosec`, `sloglint`, …).
