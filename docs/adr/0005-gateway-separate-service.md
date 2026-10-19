# ADR-0005: LLM gateway as a separate service

**Status:** Accepted

## Context
Requirement: all components reach LLMs only through one place offering unified API, retries, fallback, caching, cost accounting, streaming and metrics. It is also an explicit portfolio component.

## Decision
A standalone Go service exposing an **OpenAI-compatible** HTTP surface (`/v1/chat/completions`, `/v1/embeddings`) with extension fields (`gateway{…}`, `cost_usd`). Callers use **model aliases** (`smart`, `fast`, `extract`, `embed`) mapped to ordered provider chains in a config file. Providers: OpenAI-compatible, Anthropic, Ollama. Embeddings go through it as well.

Alternatives: an in-process library (no isolation; each service re-implements metrics/caching; can't share a cache or budget); LiteLLM proxy (works, but the gateway is a goal of the project itself).

## Consequences
- Extra network hop (~ms locally); worth it for central policy, cost visibility and the eval judge reusing the same path.
- OpenAI-compatible surface lets Python tooling (RAGAS) call it unchanged.
- Fallback is impossible after the first streamed token; documented in ARCHITECTURE §4.1.
- The gateway is a single point of failure for generation; mitigated by provider chains and the degraded mode F2.
