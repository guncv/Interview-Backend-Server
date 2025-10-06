-- name: CreateUserWithProvider :exec
INSERT INTO users (
    id,
    email,
    full_name,
    profile_image,
    provider,
    provider_id,
    is_admin,
    is_suspended,
    last_login_at,
    last_login_ip,
    last_login_user_agent,
    locale
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
);

-- name: GetUserByProviderID :one
SELECT * FROM users
WHERE provider_id = $1 AND provider = $2
LIMIT 1;

-- name: CheckUserExistsByProviderID :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE provider_id = $1 AND provider = $2
);

-- name: UpdateUserLoginInfo :exec
UPDATE users
SET last_login_at = NOW(),
    last_login_ip = $2,
    last_login_user_agent = $3,
    locale = $4
WHERE id = $1;
