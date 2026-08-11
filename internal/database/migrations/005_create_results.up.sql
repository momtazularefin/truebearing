CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE results (
    id                 UUID NOT NULL DEFAULT uuid_generate_v4(),
    run_id             UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    tenant_id          UUID NOT NULL,
    dataset_item_index INTEGER NOT NULL,
    input              JSONB NOT NULL DEFAULT '{}'::jsonb,
    output             JSONB,
    scores             JSONB NOT NULL DEFAULT '{}'::jsonb,
    evaluator_type     TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('results', 'created_at');

CREATE INDEX idx_results_run_id ON results (run_id, created_at DESC);
CREATE INDEX idx_results_tenant_id ON results (tenant_id, created_at DESC);
