-- +goose Up
-- One row per crawler run (scheduled, manual, or seed). Per-URL state lives in
-- crawl_urls (next migration).
CREATE TABLE crawl_jobs (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    trigger      text        NOT NULL CHECK (trigger IN ('schedule', 'manual', 'seed')),
    status       text        NOT NULL DEFAULT 'running'
                             CHECK (status IN ('running', 'succeeded', 'failed', 'cancelled')),
    started_at   timestamptz NOT NULL DEFAULT now(),
    finished_at  timestamptz,
    -- allowlist, rate limits, depth: what this run actually used
    config       jsonb       NOT NULL DEFAULT '{}',
    -- {"fetched":..,"not_modified":..,"new":..,"changed":..,"failed":..,"blocked_robots":..}
    stats        jsonb       NOT NULL DEFAULT '{}',
    error        text
);

CREATE INDEX crawl_jobs_started_idx ON crawl_jobs (started_at DESC);

-- +goose Down
DROP TABLE crawl_jobs;
