ALTER TABLE users
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS is_email_verified,
    DROP COLUMN IF EXISTS login_attempt_count,
    ADD COLUMN IF NOT EXISTS provider VARCHAR(50) NOT NULL DEFAULT 'google',
    ADD COLUMN IF NOT EXISTS provider_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS profile_image TEXT,
    ADD COLUMN IF NOT EXISTS locale VARCHAR(10);

ALTER TABLE users
    ADD CONSTRAINT users_provider_id_unique UNIQUE (provider_id);

DROP TABLE IF EXISTS reset_tokens CASCADE;

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token_hash ON auth_sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_is_revoked ON auth_sessions(is_revoked);

COMMENT ON TABLE users IS 'User accounts authenticated via Google OAuth (no password stored)';
COMMENT ON COLUMN users.provider IS 'OAuth provider (e.g. google, github, facebook)';
COMMENT ON COLUMN users.provider_id IS 'Unique ID from the OAuth provider';
COMMENT ON COLUMN users.profile_image IS 'User profile image URL from Google';
COMMENT ON COLUMN users.locale IS 'Preferred locale returned from OAuth provider';
