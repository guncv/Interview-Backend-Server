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
SET ended_at = $2, last_turn_id = $3
WHERE id = $1;

-- name: UpdateIsEvaluatedInterviewStateByID :execrows
UPDATE interview_states
SET is_evaluated = $2
WHERE id = $1;

-- name: UpdateLastTurnIDInterviewStateByID :execrows
UPDATE interview_states
SET last_turn_id = $2
WHERE id = $1;

-- name: GetLastTurnIDInterviewStateByID :one
SELECT last_turn_id
FROM interview_states
WHERE id = $1;