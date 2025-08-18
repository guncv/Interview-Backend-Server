-- name: CreateResume :exec
INSERT INTO resumes (
    id,
    user_id,
    file_name,
    storage_key,
    mime_type,
    byte_size,
    is_default
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: ListResumeByUserID :many
SELECT * FROM resumes WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CheckIsDefaultResumeExistsByUserID :one
SELECT EXISTS (SELECT 1 FROM resumes WHERE user_id = $1 AND is_default = TRUE);

-- name: GetDefaultResumeByUserID :one
SELECT * FROM resumes WHERE user_id = $1 AND is_default = TRUE;

-- name: UnsetDefaultResume :exec
UPDATE resumes SET is_default = FALSE WHERE id = $1;

-- name: SetDefaultResume :exec
UPDATE resumes SET is_default = TRUE WHERE id = $1;
