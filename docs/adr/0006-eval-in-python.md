# ADR-0006: Eval harness in Python

**Status:** Accepted

## Context
Needed: RAGAS-style metrics (faithfulness, answer relevance, context recall), deterministic checks (date exact match, year correctness), WER on Italian terms, latency measurement.

## Decision
Python (managed with `uv`) in `eval/`. It talks to the Go services **over HTTP only** (`/v1/ask`, `/v1/voice/ask`) and routes the LLM judge through the gateway (OpenAI-compatible `base_url`). Deterministic metrics and WER (`jiwer`) are plain Python. Results go to `eval/reports/` (JSON + Markdown) and optionally into `eval_runs/eval_results`.

A Go harness would be adequate for the deterministic metrics but would mean reimplementing RAGAS prompts and metric definitions.

## Consequences
- Second language/toolchain in the repo, fenced into one directory and treated as a black-box client — it also tests the real HTTP contracts.
- Judge LLM ≠ answer LLM (configurable) to limit self-preference bias; judge scores are advisory, the date/year metrics are the hard gates.
