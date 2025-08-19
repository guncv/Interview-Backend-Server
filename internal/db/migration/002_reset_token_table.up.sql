CREATE TABLE reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(100) NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    requested_at TIMESTAMPTZ DEFAULT now(),
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address VARCHAR(45),
    user_agent VARCHAR(256),
    UNIQUE(token_hash)
);

CREATE UNIQUE INDEX idx_reset_tokens_token_hash ON reset_tokens(token_hash);
