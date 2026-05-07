CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id              TEXT        PRIMARY KEY,
    webhook_id      TEXT        NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    tenant_id       TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_id     TEXT        REFERENCES instances(id) ON DELETE SET NULL,
    event_type      TEXT        NOT NULL,
    webhook_url     TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'queued',
    attempts        INTEGER     NOT NULL DEFAULT 0,
    response_status INTEGER     NOT NULL DEFAULT 0,
    response_body   TEXT        NOT NULL DEFAULT '',
    last_error      TEXT        NOT NULL DEFAULT '',
    payload_json    JSONB       NOT NULL,
    delivered_at    TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant_created
    ON webhook_deliveries (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_created
    ON webhook_deliveries (webhook_id, created_at DESC);
