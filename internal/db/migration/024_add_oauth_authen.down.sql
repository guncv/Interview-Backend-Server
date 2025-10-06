ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT,
    ADD COLUMN IF NOT EXISTS is_email_verified BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS login_attempt_count INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gender VARCHAR(10),
    ADD COLUMN IF NOT EXISTS date_of_birth DATE,
    ADD COLUMN IF NOT EXISTS country VARCHAR(100),
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS provider_id,
    DROP COLUMN IF EXISTS profile_image,
    DROP COLUMN IF EXISTS locale;

ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

INSERT INTO users (
    email,
    password_hash,
    full_name,
    country,
    gender,
    date_of_birth,
    is_admin,
    is_email_verified
) VALUES (
    'chanagun.vir@gmail.com',
    '$2a$10$L0Z5tv0jDo3L8iuZLHW/7ugteHxc7lHm.NvFk9ft0AzeHSYAKE25e',
    'John Doe',
    'Thailand',
    'male',
    '1995-08-16',
    false,
    true
);

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

DROP INDEX IF EXISTS idx_users_provider_id_provider;
