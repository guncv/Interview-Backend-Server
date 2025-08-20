-- name: CreateInterviewSession :exec
INSERT INTO interview_sessions (
    id,
    user_id,
    resume_id,
    requirement_id,
    modality,
    status,
    consent_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: FinishInterviewSession :execrows
UPDATE interview_sessions
SET status = 'completed',
    ended_at = $2,
    overall_score = $3,
    summary_md = $4
WHERE id = $1;

-- name: AbortInterviewSession :execrows
UPDATE interview_sessions
SET status = 'aborted',
    ended_at = $2
WHERE id = $1;

-- name: CancelInterviewSession :execrows
UPDATE interview_sessions
SET status = 'cancelled',
    ended_at = $2,
    overall_score = $3,
    summary_md = $4
WHERE id = $1;