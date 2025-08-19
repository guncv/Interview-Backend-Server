CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    email VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(100) NOT NULL,

    full_name VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL,
    gender VARCHAR(10) CHECK (gender IN ('male', 'female', 'other')) NOT NULL,
    date_of_birth DATE CHECK (date_of_birth < CURRENT_DATE) NOT NULL,

    is_admin BOOLEAN DEFAULT FALSE,
    is_email_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMP,
    login_attempt_count INT DEFAULT 0,
    is_suspended BOOLEAN DEFAULT FALSE,
    last_login_ip VARCHAR(45),
    last_login_user_agent VARCHAR(256),

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_users_email ON users(email);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(100) NOT NULL,
    user_agent VARCHAR(256) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    login_time TIMESTAMPTZ DEFAULT now(),
    last_active TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(refresh_token_hash);

CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'admin')),
    assigned_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(user_id, role)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role);
CREATE INDEX idx_user_roles_user_role ON user_roles(user_id, role);