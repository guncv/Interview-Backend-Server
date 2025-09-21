-- name: CreateInterviewSession :exec
INSERT INTO interview_sessions (
    id,
    user_id,
    resume_id,
    resume_file_name,
    position,
    modality,
    status,
    is_consent
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
    user_id,
    position
FROM interview_sessions
WHERE id = $1;

-- name: UpdateStartedAtInterviewSession :execrows
UPDATE interview_sessions
SET started_at = $2
WHERE id = $1 AND started_at IS NULL;

-- name: UpdateIsStartedConversationSession :execrows
UPDATE interview_sessions
SET is_started_conversation = $2
WHERE id = $1;

-- name: GetStartedAndIsStartedConversationSession :one
SELECT started_at, is_started_conversation
FROM interview_sessions
WHERE id = $1;

-- name: ListInterviewSessionsByUserID :many
SELECT id,
    resume_id,
    resume_file_name,
    position,
    status,
    started_at,
    ended_at,
    overall_score,
    created_at
FROM interview_sessions
WHERE user_id = $1
    AND soft_delete = false
    AND (
        $2::text IS NULL
        OR position ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
        OR resume_file_name ILIKE '%' || $2 || '%'
    )
    AND (
        $3::text IS NULL
        OR status = $3
    )
    AND (
        $4::timestamp IS NULL
        OR created_at < $4
        OR (
            created_at = $4
            AND (
                $5::uuid IS NULL
                OR id < $5
            )
        )
    )
ORDER BY created_at DESC, id DESC
LIMIT $6;
