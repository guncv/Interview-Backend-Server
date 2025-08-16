-- name: CreateResetToken :exec
INSERT INTO reset_tokens (
    id,
    user_id,
    token_hash,
    expires_at,
    ip_address,
    user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: GetResetToken :one
SELECT * FROM reset_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: UpdateResetTokenUsed :execrows
UPDATE reset_tokens
SET used = true,
    used_at = $2
WHERE token_hash = $1;