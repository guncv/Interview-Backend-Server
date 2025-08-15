-- name: IsUserHasRole :one
SELECT * FROM user_roles
WHERE user_id = $1 AND role = $2 AND deleted_at IS NULL;

-- name: CreateUserRole :exec
INSERT INTO user_roles (
    id,
    user_id,
    role
) VALUES (
    $1, $2, $3
);