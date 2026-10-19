# ADR-0003: Hybrid retrieval, filtered by academic year

**Status:** Accepted

## Context
Questions mix exact terms (ISEE, "Art. 7", form names) where lexical search wins, with paraphrased or cross-lingual queries (RU/EN question, IT documents) where embeddings win. Mixing 2025/26 and 2026/27 content is the most dangerous retrieval error.

## Decision
- Run lexical and vector search in parallel, fuse by **Reciprocal Rank Fusion** (k = 60). Rank-based fusion avoids calibrating incomparable scores.
- Lexical: Postgres FTS, `italian` config, `ts_rank_cd`. **This is not true BM25**; accepted for v1. The `TextSearcher` interface isolates a later switch to `pg_search` (ParadeDB) if eval shows a lexical weakness. English pages get imperfect Italian stemming; accepted.
- Cross-lingual: a cheap LLM call translates non-Italian queries to Italian for the lexical leg; the vector leg uses the original (bge-m3 is multilingual).
- **Academic-year filter is part of the SQL `WHERE`**, defaulting to the configured current year. Undated ("evergreen") pages are included only when flagged. Explicit user-requested years change the filter and are echoed in the answer.
- Embeddings via the gateway (`embed` alias), bge-m3 by default.

## Consequences
- Year mixing is structurally impossible in the default path; measured by the `year_correct` eval metric.
- Query-time cost: one cheap LLM call + one embedding call. Rewrites are cached in the gateway.
- No reranker initially; add one if context precision is poor in eval.
