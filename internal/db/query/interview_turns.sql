-- name: CreateInterviewTurn :exec
INSERT INTO interview_turns (
    id,
    session_id,
    turn_no,
    actor,
    current_state,
    transcript_text,
    start_at,
    end_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetInterviewerLastMessage :one
SELECT current_state, transcript_text
FROM interview_turns
WHERE session_id = $1
AND actor = 'interviewer'
ORDER BY turn_no DESC
LIMIT 1;

-- name: GetMaxTurnNoBySessionID :one
SELECT COALESCE(MAX(turn_no), 0) AS max_turn_no
FROM interview_turns
WHERE session_id = $1;
