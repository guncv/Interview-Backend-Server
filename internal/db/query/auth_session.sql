-- name: CreateAuthSession :exec
INSERT INTO auth_sessions (
    id,
    user_id,
    refresh_token_hash,
    user_agent,
    ip_address,
    last_active,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: GetAuthSessionByID :one
SELECT * FROM auth_sessions
WHERE id = $1 AND is_revoked = false
LIMIT 1;

-- name: RevokeAuthSessionByID :exec
UPDATE auth_sessions
SET is_revoked = true
WHERE id = $1;

