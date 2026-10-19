-- +goose Up
-- Chunks follow document structure (Art. N sections), not fixed windows.
-- Filter columns are copied from the version at insert time. They never drift
-- because a version's content is immutable; only is_current is toggled, in the
-- same transaction that promotes the version.
CREATE TABLE chunks (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version_id    bigint   NOT NULL REFERENCES document_versions (id) ON DELETE CASCADE,
    document_id   bigint   NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    ordinal       integer  NOT NULL,
    section_path  text,                 -- 'Art. 7' / 'Art. 7 > comma 2'
    heading       text,
    content       text     NOT NULL,
    token_count   integer,
    page_from     integer,
    page_to       integer,

    -- denormalised from document_versions / documents
    doc_type      text     NOT NULL,
    language      text     NOT NULL,
    academic_year text,
    published_at  timestamptz,
    is_current    boolean  NOT NULL DEFAULT false,

    -- dense index. Dimension is fixed by the column type: bge-m3 = 1024.
    -- Changing model/dimension = new column + backfill, never mixing in place.
    embedding       vector(1024),
    embedding_model text,
    embedded_at     timestamptz,

    -- lexical index. 'italian' stemming; English pages get imperfect stemming (accepted, see ADR-0003).
    fts tsvector GENERATED ALWAYS AS (
        to_tsvector('italian'::regconfig, coalesce(heading, '') || ' ' || content)
    ) STORED,

    UNIQUE (version_id, ordinal),
    CHECK ((embedding IS NULL) = (embedding_model IS NULL))
);

CREATE INDEX chunks_hnsw_idx ON chunks USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64)
    WHERE is_current AND embedding IS NOT NULL;
CREATE INDEX chunks_fts_idx ON chunks USING gin (fts) WHERE is_current;
CREATE INDEX chunks_filter_idx ON chunks (academic_year, doc_type) WHERE is_current;
CREATE INDEX chunks_unembedded_idx ON chunks (id) WHERE embedding IS NULL;

-- +goose Down
DROP TABLE chunks;
