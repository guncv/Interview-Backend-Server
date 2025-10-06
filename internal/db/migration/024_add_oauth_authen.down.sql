ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT,
    ADD COLUMN IF NOT EXISTS is_email_verified BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS login_attempt_count INTEGER DEFAULT 0,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS provider_id,
    DROP COLUMN IF EXISTS profile_image,
    DROP COLUMN IF EXISTS locale;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_provider_id_unique;

CREATE TABLE IF NOT EXISTS reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    requested_at TIMESTAMPTZ DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address TEXT,
    user_agent TEXT
);

DROP INDEX IF EXISTS idx_auth_sessions_user_id;
DROP INDEX IF EXISTS idx_auth_sessions_refresh_token_hash;
DROP INDEX IF EXISTS idx_auth_sessions_is_revoked;
