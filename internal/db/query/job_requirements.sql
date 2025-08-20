-- name: CreateJobRequirement :exec
INSERT INTO job_requirements (
    id,
    user_id,
    position,
    company_name,
    work_type,
    job_requirements,
    interview_type,
    language,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);