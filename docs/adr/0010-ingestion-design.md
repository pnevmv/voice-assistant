# ADR-0010: Ingestion design

**Status:** Accepted (amended after architecture review: scope question resolved, robots/discovery/translations/manual fallback added)

## Decision
1. **Crawler and RAG communicate only via Postgres.** The crawler writes `document_versions` + `chunks` (no embeddings); the RAG indexer worker embeds, extracts deadlines and *promotes* the version (claiming work through leases, ADR-0011). The old version keeps serving until promotion.
2. **Versioning:** a document is a URL; content changes create a new `document_versions` row keyed by `content_hash`. Academic year, publish date, language are properties of the *version* (a URL may be reused for the next year's bando).
3. **One normalisation for hashing and quote checking.** `content_hash = sha256(textnorm.Normalize(text))`, and deadline quotes are verified with the same function (`internal/textnorm`: NFC, whitespace and line-break collapse, join of hyphenated line breaks, apostrophe/quote/dash unification, ligature and soft-hyphen stripping). Cosmetic extractor differences therefore neither create versions nor break quote verification. The normalised form is for comparison and hashing only, never for display or embedding.
4. **Change detection:** conditional GET (ETag/Last-Modified) first, then `raw_hash`, then `content_hash`.
5. **Chunking by structure:** `Art. N` (and `comma`) for bandi, headings for pages, Q/A pairs for FAQs; oversize sections are split on paragraphs and keep the section path. Fixed token windows are only a last resort (F11).
6. **PDF text via poppler `pdftotext`** as a subprocess (avoids the licence issues of commercial Go PDF libs). Scanned PDFs are flagged `needs_ocr`; OCR is a later task. *Observation from the real 2026/27 bando:* `-layout` mode interleaves the columns of the "RIEPILOGO SCADENZE" summary table (a single date appears as fragments in different columns), so quotes from it cannot verify. That table is the most convenient single source of deadlines, so **an M2 task compares `-layout` with `-raw` and the default reading-order mode on the table pages** (accept a mode if the table's rows come out as contiguous, verifiable quotes such as "Paga la 1° rata … entro il 30 settembre 2026"), and chooses the mode per page or per document type. Until then, deadlines come from the running text (Art. 3 etc.) and the table's quotes are rejected by the validator, which is the safe outcome.

### Crawl scope (resolves the former open question 7)
- **Verified facts (2026-11-24):** the 2026/27 regional scholarship bando is published by unipd.it itself. Landing page `https://www.unipd.it/borse-studio-alloggi`; PDFs on `wwwassets.unipd.it` (IT and EN). `www.unipd.it/robots.txt`: `User-Agent: * / Allow: /` and `Sitemap: https://www.unipd.it/sitemap.xml`. `wwwassets.unipd.it/robots.txt` returns **403** while the PDFs return 200; they are served by CloudFront with `cache-control: max-age=31536000`, ETag, Last-Modified and `x-robots-tag: noindex, nofollow`.
- **Allowlist:** `www.unipd.it` and `wwwassets.unipd.it`, plus per-host path prefixes (config). **ESU Padova is out of scope for M2.**
- **Seeds (config):** the landing page and both bando PDFs (`config/crawler.example.yaml`).

### robots.txt (RFC 9309)
2xx → obey the rules. **Any 4xx, including 401 and 403 → no restrictions** (this makes `wwwassets.unipd.it` crawlable, with our User-Agent and rate limit). 5xx, 429 or a network error → treat as disallow *temporarily*: crawl nothing on that host and retry with backoff, never cache "disallow all" permanently. The mapping is implemented explicitly in `ingest.RobotsFromResponse` and pinned by tests; `github.com/temoto/robotstxt` v1.1.2 happens to do the same for 4xx (allow all, including 401/403) and for 5xx (disallow all, but with no "temporary" signal), and a characterisation test will flag any upgrade that changes this.

### URL discovery
- **Primary:** `sitemap.xml` filtered by the path allowlist. **Secondary:** link-following from HTML seed pages up to `max_depth`.
- **Never extract links from PDFs** (they carry `nofollow`); the `X-Robots-Tag` value is stored on the version (`x_robots_tag`). `noindex` is a search-engine directive. **Decision:** acceptable for this personal, non-public project, because we index only for our own retrieval and always cite and link the original URL. **This must be revisited if the service is ever made public** (a public service re-serves the content and the university's intent behind `noindex` then matters): re-check the directive, the sources' terms and, if in doubt, ask the university.
- **Corrections are published as new files** (suffixes `_0`, `_1`, …) and the CDN caches old ones for a year, so discovering *new PDF links on listing pages* matters more than re-fetching known PDFs. Listing pages are re-crawled daily; known PDFs weekly with conditional GET.
- `published_at` = the date in the document; if there is none, the `Last-Modified` header, recorded in `published_at_source`.

### Document authority and the Italian/English versions
**Verified 2026-11-24 against the 2026/27 PDFs** (`pdftotext -layout`; passages are quoted verbatim):

- Italian bando, Art. 1 (*Finalità del bando*): *"Per quanto non espressamente previsto dal presente bando e in caso di difformità o contrasto interpretativo, prevalgono le disposizioni contenute nella Deliberazione della Giunta regionale sulla quale il bando si fonda."*
- Italian bando, final note: *"N.B. Il presente bando è tradotto anche in lingua inglese ai soli fini divulgativi. Ad ogni effetto di legge è valida solo la versione italiana."*
- English bando, final note: *"N.B. This call for applications has been translated into English for information purposes only. Only the Italian version is valid for all legal purposes."*
- English bando, Art. 1: *"…in the event of discrepancies or interpretative conflicts, the provisions contained in the Regional Council's Resolution on which the call is based shall prevail."*

**Hierarchy:** (1) the regional resolution on which the call is based (*Deliberazione della Giunta regionale*) > (2) the Italian bando > (3) the English translation, which is **informational only**.

Notes that matter for implementation:
- The English text says "Regional **Council's** Resolution", but the authoritative Italian says **Giunta** regionale (the regional executive). Answers follow the Italian term; the English wording is a translation inaccuracy and is not corrected in the index.
- The English Art. 1 also says "…for the **2025/26** academic year", whereas the Italian says 2026/27 (the English title page correctly says A.Y. 2026/2027). Year detection must therefore not take the first "academic year" mention at face value: use the title/cover and the document-level majority, and a seed's configured `academic_year` wins (M2 acceptance test).
- **Indexing the resolution itself is out of scope for now.** The assistant can say the resolution prevails in case of discrepancy, but must not claim to know its content.

Consequences for the pipeline:
- Both versions are **indexed for retrieval**; the English document has `documents.translation_of` → the Italian original.
- **Deadlines are extracted from the Italian original only** (`deadlines_status = 'skipped'` for translations). This prevents duplicate or "conflicting" rows (F13) and makes the legally valid text the single source for dates.
- **Citations:** the Italian original is cited first; an English translation, when cited at all, comes after it and is marked **informational translation** (`informational_translation: true`, `original_url` → the Italian document). When both language versions are retrieved, the Italian chunk is always kept; the English chunk is added only as a secondary, informational citation (typically when the question is in English or Russian).
- The golden set records this as `auth-legal-001` ("Which version of the scholarship call is legally valid?").

### Manual ingestion fallback
For a document a site blocks for bots, the operator downloads it by hand and submits it with its **original URL** (`POST /admin/ingest` or `crawler ingest-file`). It goes through the same pipeline (version with `ingest_source='manual'`, chunks, indexing, deadline extraction and validation), and citations point to the original URL, not the local file.

## Consequences
- The crawler runs with no LLM dependency; provider outages never block crawling.
- A version that fails parsing or indexing never replaces a good one; failures are alerts, not outages.
- Extra moving part: `pdftotext` must exist in the crawler image (Dockerfile `runtime-crawler` target).
- Wrong-language or wrong-year content cannot enter through translations: only one language yields deadlines.
