# ADR-0001: Monorepo with four binaries

**Status:** Accepted

## Context
Four components, one developer. They share types (citations, deadlines), config helpers, and one database schema.

## Decision
One Git repo, one Go module (`unipd-voice-assistant`), one `cmd/<service>/main.go` per service, shared code under `internal/`. Eval harness in `eval/` (Python, its own `pyproject.toml`). Docs and migrations at the root.

```
cmd/{gateway,crawler,rag,voice}/   binaries
internal/platform/                 config, logging, HTTP lifecycle
internal/{llm,embed,speech,store,ingest,deadlines,rag}/   interfaces + implementations
migrations/                        goose SQL
config/                            non-secret service config
eval/                              Python harness + golden set
docs/                              architecture, ADRs, data model, roadmap
```

## Consequences
- Atomic refactors across services; one `make test`/`make lint`.
- One `go.mod` means a dependency needed by one service is in all builds; acceptable here. Keeps the option to split into Go workspaces later.
- Services must still communicate through their HTTP/DB contracts, not by importing each other's internals. `internal/llm` is imported by the gateway; other services use the HTTP client package only (enforced by review, optionally `depguard` later).
