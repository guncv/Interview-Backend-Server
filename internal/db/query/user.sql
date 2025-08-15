-- name: AdminCreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    name
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: CheckIsEmailExists :one
SELECT * FROM users
WHERE email = $1;

-- name: CheckIsUserIDExists :one
SELECT * FROM users
WHERE id = $1;

-- name: ResetUserPassword :exec
UPDATE users
SET password_hash = $2,
    is_temp_password = FALSE,
    updated_at = now()
WHERE id = $1;
