-- ============================================================
-- Migration 003 — Refresh token rotation for app sessions
-- ============================================================

ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS refresh_token_hash TEXT;

ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS refresh_expires_at TIMESTAMPTZ;

UPDATE user_sessions
SET
    refresh_token_hash = COALESCE(refresh_token_hash, token_hash),
    refresh_expires_at = COALESCE(refresh_expires_at, expires_at)
WHERE refresh_token_hash IS NULL
   OR refresh_expires_at IS NULL;

ALTER TABLE user_sessions
    ALTER COLUMN refresh_token_hash SET NOT NULL;

ALTER TABLE user_sessions
    ALTER COLUMN refresh_expires_at SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_sessions_refresh_hash ON user_sessions (refresh_token_hash);
