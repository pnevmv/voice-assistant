# Architecture Decision Records

Short, numbered, immutable once accepted (supersede instead of editing). Format: Context → Decision → Consequences.

| # | Title | Status |
|---|---|---|
| [0001](0001-monorepo-layout.md) | Monorepo with four binaries | Accepted |
| [0002](0002-postgres-pgvector.md) | PostgreSQL + pgvector instead of a dedicated vector DB | Accepted |
| [0003](0003-hybrid-retrieval.md) | Hybrid retrieval (full-text + vector, RRF), year-filtered | Accepted |
| [0004](0004-structured-deadlines.md) | Structured deadlines table with verified quotes | Accepted |
| [0005](0005-gateway-separate-service.md) | LLM gateway as a separate service | Accepted |
| [0006](0006-eval-in-python.md) | Eval harness in Python | Accepted |
| [0007](0007-stack-conventions.md) | Stack: net/http, pgx+sqlc, goose, no Redis | Accepted |
| [0008](0008-answer-contract.md) | Answer contract: citations, "as of", refusal | Accepted |
| [0009](0009-voice-transport.md) | Voice transport: SSE with base64 audio; batch STT first | Accepted |
| [0010](0010-ingestion-design.md) | Ingestion: DB as the interface, versioned documents, crawl scope, robots, translations, manual fallback | Accepted (amended) |
| [0011](0011-indexer-leases.md) | Indexer work claiming with leases | Accepted |
