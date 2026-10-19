-- +goose Up
-- A document is a stable source URL. Everything that can change over time
-- (text, academic year, publication date) lives on document_versions.
CREATE TABLE documents (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    canonical_url   text        NOT NULL UNIQUE,
    doc_type        text        NOT NULL DEFAULT 'page'
                                CHECK (doc_type IN ('bando', 'avviso', 'faq', 'page', 'other')),
    status          text        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'gone', 'excluded')),
    first_seen_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at    timestamptz NOT NULL DEFAULT now(),
    -- last time a NEW version was created, i.e. the content really changed
    last_changed_at timestamptz
);

CREATE TABLE document_versions (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    document_id        bigint      NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    crawl_job_id       bigint      REFERENCES crawl_jobs (id) ON DELETE SET NULL,

    -- change detection
    raw_hash           text        NOT NULL,  -- sha256 of fetched bytes
    content_hash       text        NOT NULL,  -- sha256 of normalised main text (ignores nav/boilerplate)
    fetched_at         timestamptz NOT NULL,  -- crawl timestamp shown in citations
    http_etag          text,
    http_last_modified timestamptz,
    raw_path           text,                  -- file under DATA_DIR; raw bytes are not kept in the DB

    -- facts about THIS version of the content
    title              text,
    language           text        NOT NULL DEFAULT 'it' CHECK (language ~ '^[a-z]{2}$' OR language = 'und'),
    academic_year      text,                  -- '2026/27'; NULL = not year-specific / unknown
    published_at       timestamptz,           -- date printed in the document, not the crawl date
    text_content       text,

    parse_status       text        NOT NULL DEFAULT 'ok' CHECK (parse_status IN ('ok', 'partial', 'failed')),
    parse_error        text,
    needs_ocr          boolean     NOT NULL DEFAULT false,

    -- indexing pipeline (work queue for the RAG indexer; see ARCHITECTURE.md)
    index_status       text        NOT NULL DEFAULT 'pending'
                                   CHECK (index_status IN ('pending', 'in_progress', 'done', 'failed', 'skipped')),
    index_attempts     integer     NOT NULL DEFAULT 0,
    index_error        text,
    deadlines_status   text        NOT NULL DEFAULT 'pending'
                                   CHECK (deadlines_status IN ('pending', 'done', 'failed', 'skipped')),

    -- serving state: at most one current version per document. A new version is
    -- promoted only after index_status = 'done' AND deadlines_status IN ('done','skipped').
    is_current         boolean     NOT NULL DEFAULT false,
    promoted_at        timestamptz,
    superseded_at      timestamptz,

    CONSTRAINT academic_year_format CHECK (
        academic_year IS NULL OR (
            academic_year ~ '^\d{4}/\d{2}$'
            AND ((substr(academic_year, 1, 4)::int + 1) % 100) = substr(academic_year, 6, 2)::int
        )
    ),
    UNIQUE (document_id, content_hash)
);

CREATE UNIQUE INDEX document_versions_one_current_idx ON document_versions (document_id) WHERE is_current;
CREATE INDEX document_versions_queue_idx ON document_versions (id)
    WHERE index_status IN ('pending', 'failed') OR deadlines_status = 'pending';
CREATE INDEX document_versions_year_idx ON document_versions (academic_year) WHERE is_current;

-- Per-URL crawl state: includes URLs that never become documents (hub pages,
-- 404s, robots-blocked, out of scope).
CREATE TABLE crawl_urls (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url                  text        NOT NULL UNIQUE,
    document_id          bigint      REFERENCES documents (id) ON DELETE SET NULL,
    kind                 text        NOT NULL DEFAULT 'html' CHECK (kind IN ('html', 'pdf', 'other')),
    depth                integer     NOT NULL DEFAULT 0,
    discovered_from      bigint      REFERENCES crawl_urls (id) ON DELETE SET NULL,
    state                text        NOT NULL DEFAULT 'pending'
                                     CHECK (state IN ('pending', 'ok', 'not_modified', 'redirect', 'error',
                                                      'blocked_robots', 'out_of_scope', 'gone')),
    last_status_code     integer,
    last_error           text,
    consecutive_failures integer     NOT NULL DEFAULT 0,
    etag                 text,
    last_modified        timestamptz,
    last_fetched_at      timestamptz,
    next_fetch_at        timestamptz NOT NULL DEFAULT now(),
    last_changed_at      timestamptz
);

CREATE INDEX crawl_urls_due_idx ON crawl_urls (next_fetch_at) WHERE state NOT IN ('out_of_scope', 'blocked_robots', 'gone');

-- +goose Down
DROP TABLE crawl_urls;
DROP TABLE document_versions;
DROP TABLE documents;
