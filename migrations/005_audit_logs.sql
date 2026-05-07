CREATE TABLE IF NOT EXISTS audit_logs (
    id           TEXT        PRIMARY KEY,
    tenant_id    TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_id  TEXT        REFERENCES instances(id) ON DELETE SET NULL,
    user_id      TEXT        REFERENCES users(id) ON DELETE SET NULL,
    request_id   TEXT        NOT NULL DEFAULT '',
    action       TEXT        NOT NULL,
    resource     TEXT        NOT NULL DEFAULT '',
    payload_json JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_instance_created ON audit_logs (instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created ON audit_logs (action, created_at DESC);
