CREATE TABLE prompts (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    template   TEXT NOT NULL,
    version    INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prompts_tenant_id ON prompts (tenant_id);
CREATE UNIQUE INDEX idx_prompts_tenant_name_version ON prompts (tenant_id, name, version);
