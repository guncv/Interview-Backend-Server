-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    full_name,
    country,
    city,
    address,
    postal_code,
    gender,
    date_of_birth
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = $2,
    password_hash = $3,
    full_name = $4,
    country = $5,
    city = $6,
    address = $7,
    postal_code = $8,
    gender = $9,
    date_of_birth = $10
WHERE id = $1
RETURNING *;

-- name: CheckIsEmailExists :one
SELECT * FROM users
WHERE email = $1;

-- name: CheckIsUserExistsByID :one
SELECT * FROM users
WHERE id = $1;

-- name: VerifyEmail :execrows
UPDATE users
SET is_email_verified = TRUE,
    updated_at = now()
WHERE id = $1;

-- name: SignInUserByEmailAndPassword :execrows
UPDATE users
SET last_login_at = $2,
    last_login_ip = $3,
    last_login_user_agent = $4,
    updated_at = $5
WHERE email = $1;

-- name: ResetUserPassword :execrows
UPDATE users
SET password_hash = $2,
    updated_at = $3
WHERE id = $1;
