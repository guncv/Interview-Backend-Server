-- name: CreateInterviewSession :exec
INSERT INTO interview_sessions (
    id,
    user_id,
    resume_id,
    resume_file_name,
    position,
    modality,
    status,
    is_consent,
    resume_context,
    bias_prompt
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
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
    ended_at   = CASE WHEN $2 IN ('aborted','cancelled','timed_out','completed') THEN now() ELSE ended_at END
WHERE id = $1;

-- name: CheckInterviewSessionExists :one
SELECT EXISTS (
    SELECT 1
    FROM interview_sessions
    WHERE id = $1
);
-- name: UpdateStartedAtInterviewSession :execrows
UPDATE interview_sessions
SET started_at = $2
WHERE id = $1 AND started_at IS NULL;

-- name: UpdateIsStartedConversationSession :execrows
UPDATE interview_sessions
SET is_started_conversation = $2
WHERE id = $1;

-- name: GetSessionState :one
SELECT position,
    current_state_id,
    current_state,
    started_at,
    is_started_conversation,
    bias_prompt,
    status,
    finalize_status
FROM interview_sessions
WHERE id = $1;

-- name: ListInterviewSessionsByUserIDFirstPage :many
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
    AND finalize_status = 'finalized'
    AND (
        $2::text IS NULL OR $2::text = ''
        OR position ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
        OR resume_file_name ILIKE '%' || $2 || '%'
    )
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: ListInterviewSessionsByUserIDWithCursor :many
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
    AND finalize_status = 'finalized'
    AND (
        $2::text IS NULL OR $2::text = ''
        OR position ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
        OR resume_file_name ILIKE '%' || $2 || '%'
    )
    AND (
        $4 = 'next'
        AND (
            created_at < $3
            OR (created_at = $3 AND id < $5)
        )
        OR (
            $4 = 'prev'
            AND (
                created_at > $3
                OR (created_at = $3 AND id > $5)
            )
        )
    )
ORDER BY
    CASE WHEN $4 = 'next' THEN created_at END DESC,
    CASE WHEN $4 = 'next' THEN id END DESC,
    CASE WHEN $4 = 'prev' THEN created_at END ASC,
    CASE WHEN $4 = 'prev' THEN id END ASC
LIMIT $6;

-- name: ListInterviewSessionsByUserIDWithJumpPagination :many
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
    AND finalize_status = 'finalized'
    AND (
        $4::text IS NULL OR $4::text = ''
        OR position ILIKE '%' || $4 || '%'
        OR status ILIKE '%' || $4 || '%'
        OR resume_file_name ILIKE '%' || $4 || '%'
    )
ORDER BY created_at DESC, id DESC
OFFSET ($2 - 1) * $3
LIMIT $3;

-- name: CountInterviewSessionsByUserID :one
SELECT COUNT(*)
FROM interview_sessions
WHERE user_id = $1
    AND soft_delete = false
    AND finalize_status = 'finalized'
    AND (
        $2::text IS NULL OR $2::text = ''
        OR position ILIKE '%' || $2 || '%'
        OR status ILIKE '%' || $2 || '%'
        OR resume_file_name ILIKE '%' || $2 || '%'
    );

-- name: DeleteUserInterviewSessionByID :execrows
UPDATE interview_sessions
SET soft_delete = true
WHERE id = $1
    AND user_id = $2;

-- name: GetInterviewSessionInformationByID :one
SELECT
    user_id,
    resume_id,
    resume_file_name,
    position,
    status,
    started_at,
    ended_at,
    overall_score,
    summary_md,
    created_at
FROM interview_sessions
WHERE id = $1 AND soft_delete = false;

-- name: UpdateCurrentStateAndIDInterviewSessionByID :execrows
UPDATE interview_sessions
SET current_state = $2,
    current_state_id = $3
WHERE id = $1;

-- name: UpdateFinalizeStatusInterviewSessionByID :execrows
UPDATE interview_sessions
SET finalize_status = $2
WHERE id = $1;

-- name: GetInterviewSessionStatusByID :one
SELECT status
FROM interview_sessions
WHERE id = $1;

-- name: UpdateSessionStatus :execrows
UPDATE interview_sessions
SET status = $2
WHERE id = $1;
