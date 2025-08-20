-- name: CreateInterviewTurn :exec
INSERT INTO interview_turns (
    id,
    session_id,
    turn_no,
    actor,
    content,
    transcript_text,
    stt_confidence,
    was_interrupted,
    start_at,
    end_at,
    created_at,
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: SetInterruptedTurn :execrows
UPDATE interview_turns
SET was_interrupted = $2
WHERE id = $1;