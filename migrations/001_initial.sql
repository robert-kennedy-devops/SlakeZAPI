-- ============================================================
-- Migration 001 — Initial schema
-- WhatsApp SaaS Platform
-- ============================================================

-- ── Extensions ──────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Tenants ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tenants (
    id          TEXT        PRIMARY KEY,
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_email ON tenants (email);

-- ── Plans ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS plans (
    id               TEXT    PRIMARY KEY,
    name             TEXT    NOT NULL UNIQUE,   -- starter | growth | pro
    monthly_limit    BIGINT  NOT NULL,
    price_usd_cents  BIGINT  NOT NULL DEFAULT 0,
    webhook_enabled  BOOLEAN NOT NULL DEFAULT false
);

-- Seed default plans
INSERT INTO plans (id, name, monthly_limit, price_usd_cents, webhook_enabled) VALUES
    ('plan_starter', 'starter',   1000,   0,    false),
    ('plan_growth',  'growth',    10000,  2900, true),
    ('plan_pro',     'pro',       100000, 9900, true)
ON CONFLICT (id) DO NOTHING;

-- ── API Keys ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS api_keys (
    id          TEXT        PRIMARY KEY,
    tenant_id   TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash    TEXT        NOT NULL UNIQUE,  -- SHA-256(salt + raw_key)
    key_prefix  TEXT        NOT NULL,         -- first 8 chars, for display
    label       TEXT        NOT NULL DEFAULT 'default',
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_tenant  ON api_keys (tenant_id);
CREATE INDEX idx_api_keys_hash    ON api_keys (key_hash) WHERE active = true;

-- ── Subscriptions ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS subscriptions (
    id          TEXT        PRIMARY KEY,
    tenant_id   TEXT        NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id     TEXT        NOT NULL REFERENCES plans(id),
    status      TEXT        NOT NULL DEFAULT 'active',  -- active | cancelled | past_due
    period_end  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions (tenant_id);

-- ── Usage ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS usage (
    tenant_id   TEXT    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    month       TEXT    NOT NULL,   -- format: "2024-04"
    sent        BIGINT  NOT NULL DEFAULT 0,
    received    BIGINT  NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (tenant_id, month)
);

CREATE INDEX idx_usage_tenant_month ON usage (tenant_id, month);

-- ── Messages ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS messages (
    id           TEXT        PRIMARY KEY,
    tenant_id    TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    whatsapp_id  TEXT        NOT NULL,
    phone        TEXT        NOT NULL,
    body         TEXT        NOT NULL,
    type         TEXT        NOT NULL DEFAULT 'text',
    mime_type    TEXT        NOT NULL DEFAULT '',
    file_name    TEXT        NOT NULL DEFAULT '',
    media_url    TEXT        NOT NULL DEFAULT '',
    direct_path  TEXT        NOT NULL DEFAULT '',
    file_length  BIGINT      NOT NULL DEFAULT 0,
    media_key    BYTEA,
    file_sha256  BYTEA,
    file_enc_sha256 BYTEA,
    direction    TEXT        NOT NULL,   -- inbound | outbound
    status       TEXT        NOT NULL,   -- pending | sent | delivered | read | failed
    sent_at      TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE messages ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'text';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS mime_type TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_name TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_url TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS direct_path TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_length BIGINT NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_key BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_sha256 BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_enc_sha256 BYTEA;

CREATE INDEX idx_messages_tenant     ON messages (tenant_id, created_at DESC);
CREATE INDEX idx_messages_whatsapp   ON messages (whatsapp_id);
CREATE INDEX idx_messages_phone      ON messages (tenant_id, phone);
CREATE UNIQUE INDEX idx_messages_tenant_whatsapp ON messages (tenant_id, whatsapp_id);

-- ── WhatsApp Sessions ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    tenant_id   TEXT        PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    device_jid  TEXT        NOT NULL,
    phone       TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'disconnected',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Webhooks ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS webhooks (
    id          TEXT        PRIMARY KEY,
    tenant_id   TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url         TEXT        NOT NULL,
    events      TEXT        NOT NULL,   -- comma-separated event types
    secret      TEXT        NOT NULL,   -- HMAC secret for signature
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_tenant ON webhooks (tenant_id) WHERE active = true;
