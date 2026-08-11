CREATE TABLE api_keys (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash   TEXT NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_tenant_id ON api_keys (tenant_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash) WHERE is_active = TRUE;
