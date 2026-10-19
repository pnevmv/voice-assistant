-- +goose Up
-- Per-request ledger written by the gateway (async, best effort: a DB outage
-- must never fail an LLM call). Prometheus gives rates; this gives exact cost
-- questions ("what did the last eval run cost?").
CREATE TABLE llm_usage (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ts                timestamptz   NOT NULL DEFAULT now(),
    request_id        text          NOT NULL,
    trace_id          text,
    caller            text          NOT NULL,  -- rag | crawler | voice | eval
    purpose           text          NOT NULL,  -- answer | rewrite | extract | embed | judge
    model_alias       text          NOT NULL,
    provider          text,
    model             text,
    cached            boolean       NOT NULL DEFAULT false,
    attempts          integer       NOT NULL DEFAULT 1,
    fallback_used     boolean       NOT NULL DEFAULT false,
    prompt_tokens     integer       NOT NULL DEFAULT 0,
    completion_tokens integer       NOT NULL DEFAULT 0,
    cost_usd          numeric(12,6) NOT NULL DEFAULT 0,
    latency_ms        integer,
    ttft_ms           integer,
    status            text          NOT NULL,  -- ok | error
    error_class       text
);

CREATE INDEX llm_usage_ts_idx ON llm_usage (ts DESC);
CREATE INDEX llm_usage_purpose_idx ON llm_usage (caller, purpose, ts DESC);

-- +goose Down
DROP TABLE llm_usage;
