-- Enable Row-Level Security (RLS) on all tenant-scoped tables.
-- FORCE ROW LEVEL SECURITY ensures the policy applies strictly even to the table owner,
-- fulfilling D004 and the structural tenant isolation guarantee.

ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY api_keys_tenant_isolation ON api_keys
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE datasets ENABLE ROW LEVEL SECURITY;
ALTER TABLE datasets FORCE ROW LEVEL SECURITY;
CREATE POLICY datasets_tenant_isolation ON datasets
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE prompts ENABLE ROW LEVEL SECURITY;
ALTER TABLE prompts FORCE ROW LEVEL SECURITY;
CREATE POLICY prompts_tenant_isolation ON prompts
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE runs FORCE ROW LEVEL SECURITY;
CREATE POLICY runs_tenant_isolation ON runs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE results ENABLE ROW LEVEL SECURITY;
ALTER TABLE results FORCE ROW LEVEL SECURITY;
CREATE POLICY results_tenant_isolation ON results
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
