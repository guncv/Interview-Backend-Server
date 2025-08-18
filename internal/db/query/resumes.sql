-- name: CreateResume :exec
INSERT INTO resumes (
    id,
    user_id,
    file_name,
    storage_key,
    mime_type,
    byte_size,
    parsed_json,
    raw_text,
    summary_text,
    is_default
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: GetListResumeByUserID :many
SELECT * FROM resumes WHERE user_id = $1;

-- name: GetDefaultResumeByUserID :one
SELECT * FROM resumes WHERE user_id = $1 AND is_default = TRUE;

-- name: UnsetDefaultResume :exec
UPDATE resumes SET is_default = FALSE WHERE id = $1;

-- name: SetDefaultResume :exec
UPDATE resumes SET is_default = TRUE WHERE id = $1;
