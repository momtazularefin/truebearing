CREATE TABLE runs (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    dataset_id   UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    config       JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary      JSONB,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runs_tenant_id ON runs (tenant_id);
CREATE INDEX idx_runs_dataset_id ON runs (dataset_id);
CREATE INDEX idx_runs_status ON runs (status) WHERE status IN ('pending', 'running');
