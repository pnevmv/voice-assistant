# ADR-0002: PostgreSQL + pgvector instead of a dedicated vector DB

**Status:** Accepted

## Context
Corpus: hundreds of pages and PDFs → on the order of 10⁴ chunks, even with several versions kept. We need vector search, full-text search, relational filters (year, doc type, current version), transactions (promote a version atomically), and a queue.

## Decision
PostgreSQL 17 with the `pgvector` extension is the only datastore. Chunk embeddings are `vector(1024)` with a partial HNSW index (`WHERE is_current`). Full-text lives in a generated `tsvector` column.

Alternatives considered: Qdrant/Weaviate/Milvus (extra service, no joins/transactions with the relational data, filter-consistency problems on version swaps); SQLite+sqlite-vec (no concurrent writers, weaker FTS for Italian).

## Consequences
- One backup, one migration tool, transactional version promotion.
- Filtered ANN queries can under-return with HNSW post-filtering; at this scale pgvector's iterative scan (`hnsw.iterative_scan`) or even exact scan is fast enough. Revisit only if chunk count exceeds ~10⁶.
- The embedding dimension is fixed in the column type. Changing the model means adding a column and backfilling, never mixing models in one index.
