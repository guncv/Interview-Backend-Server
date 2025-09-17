-- name: CreateInterviewSession :exec
INSERT INTO interview_sessions (
    id,
    user_id,
    resume_id,
    position,
    modality,
    status,
    is_consent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: EndInterviewSession :execrows
UPDATE interview_sessions
SET status = $2,
    ended_at = $3,
    overall_score = $4,
    summary_md = $5
WHERE id = $1;

-- name: UpdateInterviewSessionStatus :execrows
UPDATE interview_sessions
SET status = $2::VARCHAR(20),
    started_at = CASE WHEN $2 = 'on_going' AND started_at IS NULL THEN now() ELSE started_at END,
    ended_at   = CASE WHEN $2 IN ('aborted','cancelled','timed_out') THEN now() ELSE ended_at END
WHERE id = $1;

-- name: CheckInterviewSessionExists :one
SELECT EXISTS (
    SELECT 1
    FROM interview_sessions
    WHERE id = $1
);

-- name: GetInterviewSessionInformation :one
SELECT
    i.user_id AS user_id,
    i.position AS position,
    r.file_name AS file_name
FROM interview_sessions i
JOIN resumes r ON i.resume_id = r.id
WHERE i.id = $1;

