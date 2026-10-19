-- +goose Up
CREATE TABLE eval_runs (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name               text        NOT NULL,
    kind               text        NOT NULL CHECK (kind IN ('rag', 'voice', 'full')),
    status             text        NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'succeeded', 'failed')),
    git_sha            text,
    golden_set_version text        NOT NULL,
    golden_set_hash    text        NOT NULL,
    -- models, retrieval params, prompt versions: everything needed to reproduce
    target_config      jsonb       NOT NULL DEFAULT '{}',
    judge_model        text,
    started_at         timestamptz NOT NULL DEFAULT now(),
    finished_at        timestamptz,
    -- aggregate: {"faithfulness":0.93,"date_exact_match":0.88,"year_correct":1.0,"ttfa_p50_ms":..}
    metrics            jsonb       NOT NULL DEFAULT '{}',
    notes              text
);

CREATE TABLE eval_results (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    run_id        bigint  NOT NULL REFERENCES eval_runs (id) ON DELETE CASCADE,
    question_id   text    NOT NULL,
    language      text    NOT NULL,
    answer        text,
    status        text,                       -- answered | not_found | partial | error
    citations     jsonb,
    retrieved     jsonb,                      -- chunk ids + ranks, for context-recall analysis
    scores        jsonb   NOT NULL DEFAULT '{}',
    latency_ms    integer,
    ttfa_ms       integer,                    -- voice runs only
    error         text,
    UNIQUE (run_id, question_id, language)
);

-- +goose Down
DROP TABLE eval_results;
DROP TABLE eval_runs;
