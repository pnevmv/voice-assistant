-- +goose Up
-- Structured deadlines extracted by an LLM pass over each bando version.
-- Questions about dates are answered from here, not from vector search alone.
CREATE TABLE deadlines (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version_id     bigint  NOT NULL REFERENCES document_versions (id) ON DELETE CASCADE,
    document_id    bigint  NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    chunk_id       bigint  REFERENCES chunks (id) ON DELETE SET NULL,

    event          text    NOT NULL,                 -- 'Scadenza presentazione domanda'
    event_key      text    NOT NULL,                 -- controlled slug: 'scholarship_application_deadline'
    applies_to     text    NOT NULL DEFAULT 'all',   -- 'matricole', 'anni successivi', 'studenti internazionali'
    academic_year  text    NOT NULL CHECK (academic_year ~ '^\d{4}/\d{2}$'),

    -- Single day: starts_on = ends_on. Cut-off ("entro il"): starts_on NULL.
    starts_on      date,
    ends_on        date,
    time_of_day    time,                             -- Europe/Rome local time
    date_text      text    NOT NULL,                 -- date exactly as written in the source

    source_url     text    NOT NULL,
    source_quote   text    NOT NULL,                 -- verbatim passage
    source_page    integer,

    -- provenance and trust
    extraction_model          text        NOT NULL,
    extraction_prompt_version text        NOT NULL,
    extracted_at              timestamptz NOT NULL DEFAULT now(),
    quote_verified  boolean NOT NULL,                -- source_quote found verbatim in the version text
    date_in_quote   boolean NOT NULL,                -- parsed date is consistent with date_text/quote
    review_status   text    NOT NULL DEFAULT 'unreviewed'
                            CHECK (review_status IN ('unreviewed', 'confirmed', 'rejected')),
    reviewed_by     text,
    reviewed_at     timestamptz,
    review_note     text,

    is_current      boolean NOT NULL DEFAULT false,  -- follows the version; toggled on promotion

    CHECK (starts_on IS NOT NULL OR ends_on IS NOT NULL),
    CHECK (starts_on IS NULL OR ends_on IS NULL OR ends_on >= starts_on)
);

CREATE INDEX deadlines_lookup_idx ON deadlines (academic_year, event_key) WHERE is_current AND review_status <> 'rejected';
CREATE INDEX deadlines_version_idx ON deadlines (version_id);

-- +goose Down
DROP TABLE deadlines;
