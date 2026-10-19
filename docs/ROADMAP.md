# Roadmap

Order: gateway → crawler + deadlines → RAG → eval → voice → observability. Each milestone ends with a demo-able, tested increment. Sizes are rough solo-developer estimates (part-time evenings/weekends) and should be re-planned after M1.

Definition of done for every milestone: `make lint test` green, migrations apply cleanly on an empty DB, README/ARCHITECTURE updated where the behaviour changed, no secrets in git.

## M0 — Scaffolding (done in the first session)
Docs, ADRs, schema, skeleton, compose, Makefile, lint config.
**Accept:** `make up && make migrate` creates the schema on pgvector; `make build` compiles four stub binaries that start, serve `/healthz`, and exit cleanly on SIGINT. *(Not yet verified in this session: no Go toolchain/Docker daemon was available — verify first.)*

## M1 — Gateway MVP (≈ 1–1.5 weeks)
Scope: `/v1/chat/completions` (+SSE), `/v1/embeddings`, providers: OpenAI-compatible + Ollama + Anthropic; alias → provider chain config; retry with exponential backoff + jitter; fallback; circuit breaker; in-memory cache (temperature 0); token/cost accounting (config prices) → response + `llm_usage` + Prometheus; request IDs; typed Go client package for other services.
**Acceptance:**
- Table-driven tests with fake providers cover: retry on 429/5xx honouring `Retry-After`, no retry on 4xx, fallback order, breaker open/half-open, fallback blocked after first streamed token, cache hit/miss and bypass, cost computation.
- Streaming through the gateway passes OpenAI-format chunks unchanged to a stock OpenAI client (smoke test with `curl` and a Python `openai` client).
- Killing the primary provider mid-test results in a successful answer from the fallback with `gateway.fallback_used=true`.
- `/metrics` exposes request count/latency/token/cost/cache/fallback series; `llm_usage` rows appear and the service still works with Postgres stopped.
- p95 gateway overhead (excluding provider time) < 10 ms locally.

## M2 — Crawler + deadlines extraction (≈ 2–3 weeks)
Scope: fetcher (robots per RFC 9309, rate limit, conditional GET) for `www.unipd.it` + `wwwassets.unipd.it` (scope resolved, ADR-0010); URL discovery from `sitemap.xml` + path allowlist + depth-limited link-following from HTML only (never from PDFs); `X-Robots-Tag` and `Last-Modified` fallback stored on the version; HTML + PDF parsers (**task: compare `pdftotext -layout` vs `-raw` vs default mode on the table pages of the real bando, notably "RIEPILOGO SCADENZE", and pick the mode per page/doc type**), year/doc-type/date detection, structure-based chunker (`Art. N`), versioning + change detection with `textnorm.ContentHash`, scheduled recrawl (listing pages daily to find new `_0/_1` files, PDFs weekly); IT/EN translation linking (`translation_of`); then the RAG **indexer worker** (lease-based claiming, heartbeat, backoff, max attempts — ADR-0011), embedding step, **deadline extraction from the Italian original only + validator + review CLI**, fenced promotion transaction; and the **manual ingestion fallback** (`crawler ingest-file` / `POST /admin/ingest`: a locally downloaded file plus its original URL goes through the same pipeline, citations use the original URL).
**Acceptance:**
- Fixture-based tests (saved HTML/PDF from the real site, committed only if licence permits, otherwise synthetic) for parsers and chunker; the 2026/27 scholarship bando chunks per article with correct `section_path` and page numbers.
- Re-crawl of unchanged content creates zero new versions; a modified fixture creates exactly one version and the old one stays current until indexing completes (tested by interrupting the worker).
- robots.txt disallowed paths are never fetched (test with a local server); rate limit ≥ configured delay per host (test).
- Parse failure of a PDF keeps the old version served and raises the failure metric/log.
- Indexer crash safety (integration test on real Postgres): kill a worker mid-version → the lease expires and another worker finishes it; a late worker's completion affects 0 rows; no transaction is open during gateway calls (assert via `pg_stat_activity`); a version that always fails ends as `failed` after `INDEXER_MAX_ATTEMPTS` and raises the alert metric.
- robots.txt: unit tests pin the 2xx/4xx/5xx/429/network mapping (4xx incl. 403 → crawlable; 5xx → paused, retried later). PDF responses yield no links; `x_robots_tag` is stored.
- **PDF mode comparison (table pages):** a short report in `docs/` shows, for each mode, whether each row of "RIEPILOGO SCADENZE" yields a contiguous quote that passes `deadlines.Validate`; the chosen mode is covered by a fixture test. If a mode makes the table usable, its deadlines are cross-checked against the running-text ones (Art. 3 …) and any disagreement is flagged for review.
- **Year detection is not fooled by translations:** the English bando's Art. 1 says "2025/26" but both language versions are tagged `2026/27` (seed config / title wins); a test asserts no chunk of either version carries `2025/26`.
- Both language versions of the 2026/27 bando are indexed and linked; **deadline rows exist only for the Italian one**, and the same event never appears twice.
- Manual ingestion of a local copy of the bando with its original URL produces the same chunks/deadlines as the crawled version (same `content_hash`) and cites the original URL.
- Deadlines: on the real 2026/27 bando(s), **every extracted row has a verbatim quote that passes `textnorm` + `datex` verification**; manually reviewing all rows yields ≥ 95 % correct dates (any wrong row must be caught by the validator or the review state before serving). Zero rows served with `quote_verified = false`.
- Academic-year detection correct on 100 % of a hand-checked sample of ≥ 30 documents (or `NULL`, never wrong).

## M3 — RAG text API (≈ 1.5–2 weeks)
Scope: `POST /v1/ask` (JSON + SSE), query rewrite/translation, hybrid retrieval + RRF, year filter, deadline intent route, prompt with citation rules, citation/date guards, refusal path, degraded modes F2/F4, `GET /v1/deadlines`.
**Acceptance:**
- Contract tests for JSON and SSE shapes exactly as in ARCHITECTURE §4.2.
- Default queries never return chunks outside the requested academic year (property test over a seeded DB containing both 2025/26 and 2026/27 versions of the same bando).
- Unanswerable questions yield `status=not_found` with no citations on ≥ 90 % of a hand-made list of 10 out-of-scope questions.
- A deadline question is answered with the date copied from the `servable_deadlines` row (unit-test the prompt assembly and the date guard with injected wrong dates → `verified=false`; guard tests exist for IT/EN/RU answers against Italian context).
- The answer prompt includes `rag.SourceAuthorityRule`. Golden record `auth-legal-001` ("Which version of the scholarship call is legally valid?") passes in IT/EN/RU: answered, the Italian original is the **first citation**, any English source is flagged `informational_translation`, the answer says the regional resolution (Giunta) prevails over the call without describing its content.
- The answer prompt includes `rag.DateFormatRule`; a test with a fake LLM that answers "30/09/2026" gets `verified` unaffected for the date itself but a `numeric_date` warning, while "30 сентября 2026" passes cleanly.
- Retrieval starts the embedding/vector leg in parallel with the `fast` call; measured retrieval latency is below the sequential sum (assert in a timing test with fake providers).
- RU, EN and IT versions of the same question retrieve the same top-3 sources in a smoke test.
- With embeddings provider stopped → lexical-only answer with warning; with generation stopped → citations + deadline rows, no prose.

## M4 — Eval harness + golden set (≈ 1.5 weeks, partly manual)
Scope (voice eval specifics are in M5): Python package in `eval/`, golden set (50–100 entries, manually verified against the official bandi), runners, RAGAS metrics via the gateway, deterministic metrics (date exact match, year correctness, refusal accuracy), report generation, `eval_runs` persistence, baseline run.
**Acceptance:**
- `make eval` produces a Markdown + JSON report with faithfulness, answer relevance, context recall, date exact-match, year-correct, refusal precision/recall, cost and latency.
- The golden set already contains its first verified record, `auth-legal-001` (category `source_authority`, with verbatim quotes from both PDFs); the runner checks `first_citation_url` and `informational_source_urls`.
- Golden set validator rejects entries with `status: placeholder` unless `--allow-placeholders`; ≥ 50 entries `verified`, each with a source quote and verification date; ≥ 10 date questions, ≥ 5 cross-year traps, ≥ 8 not-found, ≥ 15 each in RU/EN.
- Runs are reproducible: report pins git SHA, golden set hash, models, prompt versions. Re-running with the same cache gives identical deterministic metrics.
- A baseline is committed; a regression (e.g. date exact-match drop > 2 points) fails `make eval-check`.
- **Target (baseline gate, adjust after the first run):** date exact-match ≥ 95 %, year-correct = 100 %.

## M5 — Voice layer (≈ 1.5–2 weeks)
Scope: ElevenLabs STT adapter (`scribe_v3` batch, `keyterms` from `config/keyterms.txt`; `scribe_v3_live` realtime only as a later optimisation) and streaming TTS adapter (`eleven_v4_turbo`, fallback `eleven_flash_v2_5`, both env-configurable), `POST /v1/voice/ask` SSE pipeline, sentence splitter, per-sentence date guard, pre-warmed connections, TTS failure fallback to text; voice eval (TTFA, WER on domain terms) with ≥ 20 real recordings (some Russian-accented Italian terms) plus synthetic audio.
**Acceptance:**
- Fake STT/TTS tests cover the splitter (abbreviations like "art.", "n.", dates "30.04.2027"), cancellation on client disconnect (no goroutine leak), and ordering of `audio` events.
- Measured on the eval set: **TTFA p50 ≤ 3.5 s, p95 ≤ 5 s** with the chosen providers (numbers reported, not promised: tune or document the gap).
- **Keyterms A/B:** the voice eval reports WER and term-level error on the domain terms **with vs without keyterms** (same recordings, `keyterms=true|false`), plus the extra cost. Keyterms stay enabled only if they measurably help; target ≤ 10 % term-level error, otherwise document mitigation.
- **TTS model comparison:** time-to-first-audio p50/p95 for `eleven_v4_turbo` vs `eleven_flash_v2_5` on the same questions (`tts_model` override); the fallback path is tested by forcing a primary failure before the first byte.
- Per-request overrides are rejected with `400 overrides_disabled` unless `VOICE_ALLOW_OVERRIDES=true` (tested both ways).
- The voice service refuses to start with a keyterms list that exceeds the realtime limits (≤ 50 terms, ≤ 1000 characters in total, ≤ 50 characters and ≤ 5 words per term); tested.
- Citations are present as text in the stream before the first `audio` event.
- A sentence whose date fails the guard is not spoken.

## M6 — Observability and hardening (≈ 1 week)
Scope: OpenTelemetry traces across voice → rag → gateway, metrics on all services, Grafana dashboard JSON, alert rules (stale crawl, failed bando parse, pending-index age, error rates, budget), graceful-shutdown tests, load smoke test, per-caller budgets, retention job, backup/restore notes, dependency/license check.
**Acceptance:**
- A single voice request yields one trace containing STT, RAG retrieval, gateway call(s), TTS spans; `trace_id` appears in logs of all three services and in API responses.
- Dashboards show per-alias latency/cost, fallback rate, crawl freshness, TTFA.
- Alert rules fire in a staged failure test (stop provider, block crawl host, corrupt a PDF fixture).
- `SIGTERM` during an in-flight SSE stream lets it finish (up to the shutdown timeout) and rejects new requests.
- `make up` + `make migrate` + documented steps bring a clean machine to a working system in < 15 minutes.

## Backlog (not scheduled)
OCR for scanned PDFs; cross-encoder reranker; BM25 via `pg_search`; WebSocket voice transport and barge-in; streaming STT; web demo UI; multi-turn conversation memory; human review UI for deadlines; pruning/retention of old versions; notifications when a tracked deadline changes.
