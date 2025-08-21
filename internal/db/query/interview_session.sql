-- name: CreateInterviewSession :exec
INSERT INTO interview_sessions (
    id,
    user_id,
    resume_id,
    requirement_id,
    modality,
    status,
    consent_at,
    prompt_json
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: EndInterviewSession :execrows
UPDATE interview_sessions
SET status = $2,
    ended_at = $3,
    overall_score = $4,
    summary_md = $5
WHERE id = $1;

-- name: AbortInterviewSession :execrows
UPDATE interview_sessions
SET status = $2,
    ended_at = $3
WHERE id = $1;

-- name: StartInterviewSession :execrows
UPDATE interview_sessions
SET status = $2,
    started_at = $3
WHERE id = $1;