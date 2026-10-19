-- +goose Up
-- Architecture review fixes: indexer leases, translations, crawl provenance,
-- a single definition of "servable deadline".

-- 1. Indexer lease. 'in_progress' (00003) is renamed 'indexing'; a worker owns a
--    version only while claimed_at is fresh (INDEXER_LEASE_TIMEOUT). Retry state
--    is 'pending' + index_next_attempt_at; 'failed' is terminal (max attempts).
ALTER TABLE document_versions DROP CONSTRAINT document_versions_index_status_check;
UPDATE document_versions SET index_status = 'indexing' WHERE index_status = 'in_progress';
ALTER TABLE document_versions
    ADD CONSTRAINT document_versions_index_status_check
        CHECK (index_status IN ('pending', 'indexing', 'done', 'failed', 'skipped')),
    ADD COLUMN claimed_at            timestamptz,
    ADD COLUMN claimed_by            text,
    ADD COLUMN index_next_attempt_at timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT document_versions_lease_check
        CHECK (index_status <> 'indexing' OR (claimed_at IS NOT NULL AND claimed_by IS NOT NULL));

DROP INDEX document_versions_queue_idx;
CREATE INDEX document_versions_claim_idx ON document_versions (index_next_attempt_at, id)
    WHERE index_status IN ('pending', 'indexing');

-- 2. Provenance of a version.
ALTER TABLE document_versions
    ADD COLUMN x_robots_tag        text,
    ADD COLUMN published_at_source text
        CHECK (published_at_source IN ('document', 'http_last_modified')),
    ADD COLUMN ingest_source       text NOT NULL DEFAULT 'crawl'
        CHECK (ingest_source IN ('crawl', 'manual'));

-- 3. Translations: the Italian document is the original (translation_of IS NULL);
--    its English version points at it. Both are indexed for retrieval, but
--    deadlines are extracted from the original only (ADR-0010).
ALTER TABLE documents
    ADD COLUMN translation_of bigint REFERENCES documents (id) ON DELETE SET NULL,
    ADD CONSTRAINT documents_translation_not_self CHECK (translation_of IS NULL OR translation_of <> id);
CREATE INDEX documents_translation_of_idx ON documents (translation_of) WHERE translation_of IS NOT NULL;

-- 4. How a URL entered the frontier.
ALTER TABLE crawl_urls
    ADD COLUMN discovered_via text NOT NULL DEFAULT 'link'
        CHECK (discovered_via IN ('seed', 'sitemap', 'link', 'manual'));

-- 5. The ONE definition of which deadline rows may be shown to a user. Queries,
--    the API and eval must read this view, never the table directly.
CREATE VIEW servable_deadlines AS
SELECT *
FROM deadlines
WHERE is_current
  AND quote_verified
  AND date_in_quote
  AND review_status <> 'rejected';

-- +goose Down
DROP VIEW servable_deadlines;

ALTER TABLE crawl_urls DROP COLUMN discovered_via;

DROP INDEX documents_translation_of_idx;
ALTER TABLE documents DROP CONSTRAINT documents_translation_not_self, DROP COLUMN translation_of;

ALTER TABLE document_versions DROP COLUMN ingest_source, DROP COLUMN published_at_source, DROP COLUMN x_robots_tag;

DROP INDEX document_versions_claim_idx;
-- in-flight leases cannot exist after a rollback: put them back in the queue
UPDATE document_versions SET index_status = 'pending' WHERE index_status = 'indexing';
ALTER TABLE document_versions
    DROP CONSTRAINT document_versions_lease_check,
    DROP COLUMN index_next_attempt_at,
    DROP COLUMN claimed_by,
    DROP COLUMN claimed_at,
    DROP CONSTRAINT document_versions_index_status_check;
ALTER TABLE document_versions
    ADD CONSTRAINT document_versions_index_status_check
        CHECK (index_status IN ('pending', 'in_progress', 'done', 'failed', 'skipped'));
CREATE INDEX document_versions_queue_idx ON document_versions (id)
    WHERE index_status IN ('pending', 'failed') OR deadlines_status = 'pending';
