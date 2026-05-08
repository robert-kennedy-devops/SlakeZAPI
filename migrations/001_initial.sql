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
    name             TEXT    NOT NULL UNIQUE,   -- trial | starter | growth | pro
    monthly_limit    BIGINT  NOT NULL,
    price_usd_cents  BIGINT  NOT NULL DEFAULT 0,
    webhook_enabled  BOOLEAN NOT NULL DEFAULT false
);

-- Seed default plans
INSERT INTO plans (id, name, monthly_limit, price_usd_cents, webhook_enabled) VALUES
    ('plan_trial',   'trial',         0,      0, true),
    ('plan_starter', 'starter',   3000,   7900,  false),
    ('plan_growth',  'growth',    15000,  14900, true),
    ('plan_pro',     'pro',       60000,  29900, true)
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
    id                       TEXT        PRIMARY KEY,
    tenant_id                TEXT        NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id                  TEXT        NOT NULL REFERENCES plans(id),
    status                   TEXT        NOT NULL DEFAULT 'trial',  -- trial | pending | active | past_due | cancelled
    provider                 TEXT        NOT NULL DEFAULT '',
    provider_customer_id     TEXT        NOT NULL DEFAULT '',
    provider_subscription_id TEXT        NOT NULL DEFAULT '',
    provider_price_id        TEXT        NOT NULL DEFAULT '',
    current_period_start     TIMESTAMPTZ,
    period_end               TIMESTAMPTZ NOT NULL,
    trial_ends_at            TIMESTAMPTZ,
    cancel_at_period_end     BOOLEAN     NOT NULL DEFAULT false,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions (tenant_id);
CREATE INDEX idx_subscriptions_provider_subscription_id ON subscriptions (provider_subscription_id);

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

-- ── Instances ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS instances (
    id          TEXT        PRIMARY KEY,
    tenant_id   TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    phone       TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'disconnected',
    is_default  BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE instances ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';
ALTER TABLE instances ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'disconnected';
ALTER TABLE instances ADD COLUMN IF NOT EXISTS is_default BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE instances ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE instances ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_instances_tenant ON instances (tenant_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_tenant_default ON instances (tenant_id) WHERE is_default = true;

-- ── Messages ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS messages (
    id           TEXT        PRIMARY KEY,
    tenant_id    TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_id  TEXT        REFERENCES instances(id) ON DELETE SET NULL,
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
ALTER TABLE messages ADD COLUMN IF NOT EXISTS instance_id TEXT REFERENCES instances(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS mime_type TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_name TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_url TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS direct_path TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_length BIGINT NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_key BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_sha256 BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS file_enc_sha256 BYTEA;

CREATE INDEX idx_messages_tenant     ON messages (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_instance ON messages (instance_id, created_at DESC);
CREATE INDEX idx_messages_whatsapp   ON messages (whatsapp_id);
CREATE INDEX idx_messages_phone      ON messages (tenant_id, phone);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_tenant_instance_whatsapp ON messages (tenant_id, COALESCE(instance_id, ''), whatsapp_id);

CREATE TABLE IF NOT EXISTS conversations (
    id                TEXT        PRIMARY KEY,
    tenant_id         TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_id       TEXT        NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    phone             TEXT        NOT NULL,
    last_message_id   TEXT        NOT NULL DEFAULT '',
    last_message_body TEXT        NOT NULL DEFAULT '',
    last_direction    TEXT        NOT NULL DEFAULT '',
    last_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    state             TEXT        NOT NULL DEFAULT 'open',
    assigned_user_id  TEXT        NOT NULL DEFAULT '',
    note              TEXT        NOT NULL DEFAULT '',
    unread_count      INTEGER     NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, instance_id, phone)
);

CREATE INDEX IF NOT EXISTS idx_conversations_tenant_instance_last_at ON conversations (tenant_id, instance_id, last_at DESC);

-- ── WhatsApp Sessions ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    instance_id TEXT        PRIMARY KEY REFERENCES instances(id) ON DELETE CASCADE,
    tenant_id   TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_jid  TEXT        NOT NULL,
    phone       TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'disconnected',
    last_event  TEXT        NOT NULL DEFAULT '',
    last_error  TEXT        NOT NULL DEFAULT '',
    qr_code     TEXT        NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE whatsapp_sessions ADD COLUMN IF NOT EXISTS instance_id TEXT;
ALTER TABLE whatsapp_sessions ADD COLUMN IF NOT EXISTS tenant_id TEXT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE whatsapp_sessions ADD COLUMN IF NOT EXISTS last_event TEXT NOT NULL DEFAULT '';
ALTER TABLE whatsapp_sessions ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE whatsapp_sessions ADD COLUMN IF NOT EXISTS qr_code TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_tenant ON whatsapp_sessions (tenant_id);

-- ── Webhooks ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS webhooks (
    id          TEXT        PRIMARY KEY,
    tenant_id   TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_id TEXT        REFERENCES instances(id) ON DELETE CASCADE,
    url         TEXT        NOT NULL,
    events      TEXT        NOT NULL,   -- comma-separated event types
    secret      TEXT        NOT NULL,   -- HMAC secret for signature
    active      BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_tenant ON webhooks (tenant_id) WHERE active = true;
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS instance_id TEXT REFERENCES instances(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_webhooks_instance ON webhooks (instance_id) WHERE active = true;

-- ── Campaigns ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS campaigns (
    id                TEXT        PRIMARY KEY,
    tenant_id         TEXT        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    instance_id       TEXT        NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    name              TEXT        NOT NULL,
    message           TEXT        NOT NULL,
    status            TEXT        NOT NULL DEFAULT 'draft',
    scheduled_at      TIMESTAMPTZ,
    last_executed_at  TIMESTAMPTZ,
    total_contacts    INTEGER     NOT NULL DEFAULT 0,
    sent_count        INTEGER     NOT NULL DEFAULT 0,
    failed_count      INTEGER     NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaigns_tenant ON campaigns (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaigns_instance_status ON campaigns (instance_id, status, scheduled_at);

CREATE TABLE IF NOT EXISTS campaign_recipients (
    id           TEXT        PRIMARY KEY,
    campaign_id   TEXT       NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    input_phone   TEXT       NOT NULL,
    phone         TEXT       NOT NULL DEFAULT '',
    name          TEXT       NOT NULL DEFAULT '',
    variables     JSONB      NOT NULL DEFAULT '{}'::jsonb,
    message_id    TEXT       NOT NULL DEFAULT '',
    status        TEXT       NOT NULL DEFAULT 'pending',
    error         TEXT       NOT NULL DEFAULT '',
    is_whatsapp   BOOLEAN    NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_recipients_campaign ON campaign_recipients (campaign_id, created_at);
