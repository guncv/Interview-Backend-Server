-- name: CreateInterviewState :exec
INSERT INTO interview_states (
    id,
    session_id,
    phrase_type,
    started_at
) VALUES (
    $1, $2, $3, $4
);

-- name: UpdateEndedAtInterviewStateByID :execrows
UPDATE interview_states
SET phrase_type = $2,
    ended_at = $3
WHERE id = $1;

-- name: UpdateIsEvaluatedInterviewStateByID :execrows
UPDATE interview_states
SET is_evaluated = $2
WHERE id = $1;