# UniPD Voice Assistant

A voice + text assistant that answers questions about the University of Padua — admission documents and procedures, deadlines for the current academic year (2026/27) and the regional scholarship — in Russian, English and Italian. Every answer cites official sources with their dates, states which academic year it applies to, and says "not found in official documents" instead of guessing.

> **Status:** M0 — architecture and scaffolding only. The service binaries start and shut down gracefully but implement no business logic yet. See [`docs/ROADMAP.md`](docs/ROADMAP.md).

## Components

| Service | Port | Role |
|---|---|---|
| `gateway` | 8081 | LLM gateway: unified OpenAI-compatible API, retries, fallback, cache, cost accounting, SSE |
| `rag` | 8082 | Hybrid retrieval, structured deadlines, cited answers, indexer worker |
| `voice` | 8083 | ElevenLabs STT → RAG → streaming TTS |
| `crawler` | 8084 | Crawl unipd.it (+ configured hosts), parse HTML/PDF, version, chunk |
| `eval/` | — | Python eval harness and golden set |

Docs: [Architecture](docs/ARCHITECTURE.md) · [Data model](docs/DATA_MODEL.md) · [ADRs](docs/adr/README.md) · [Roadmap](docs/ROADMAP.md) · [Golden set format](docs/eval/GOLDEN_SET_FORMAT.md)

## Prerequisites

Go 1.26+, Docker (with Compose), `make`. Optional: `golangci-lint` v2 (`make lint`). `goose` runs via `go run` at a pinned version; install the `sqlc` binary (`brew install sqlc`, tested with 1.31.1). For PDF parsing (M2): poppler (`brew install poppler`). For local models: Ollama (`docker compose --profile local-llm up -d`, then `ollama pull bge-m3`).

## Quick start

```sh
cp .env.example .env        # fill in keys later; nothing needs keys yet
make up                     # Postgres 17 + pgvector on localhost:5432
make migrate                # apply migrations/
make build                  # builds ./bin/{gateway,crawler,rag,voice}

make run-gateway            # in separate terminals
make run-rag
curl localhost:8081/healthz   # -> ok
```

Stop a service with Ctrl-C: it logs `shutdown signal received` and exits after in-flight requests finish.

Other targets: `make help`. Reset the local DB (destructive): `make db-reset`.

## Configuration

Environment variables only; see [`.env.example`](.env.example). Secrets (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `ELEVENLABS_API_KEY`) live in `.env`, which is git-ignored. Gateway routing lives in `config/gateway.example.yaml` (no secrets; it names the env vars to read).

## Repository layout

```
cmd/<service>/main.go     service entrypoints
internal/platform/        config, logging, HTTP lifecycle (shared)
internal/llm|embed|speech|store|ingest|deadlines|rag/   interfaces & types (LLMProvider, Embedder, STT, TTS, VectorStore, …)
migrations/               goose SQL migrations
config/                   non-secret service config
docs/                     architecture, ADRs, data model, roadmap, eval docs
eval/                     Python eval harness + golden set
```

## Data correctness

Wrong deadlines can cost a student a scholarship. Treat this project as an *assistant to the official documents*, not a replacement: always follow the cited link and check the document date.
