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

-- name: GetChatHistoryBySessionID :many
SELECT id, turn_no, actor, transcript_text, start_at, end_at, created_at
FROM interview_turns
WHERE session_id = $1
ORDER BY turn_no ASC;

-- name: GetChatHistoryBySessionIDWithEvaluation :many
SELECT
    it.id AS turn_id,
    it.turn_no,
    it.actor,
    it.transcript_text,
    it.current_state,
    uti.corrected_sentence,
    it.start_at,
    it.end_at,

    CASE 
        WHEN e.id IS NOT NULL THEN jsonb_build_object(
            'overall_score', e.overall_score,
            'summary_md', e.summary_md,
            'scores', COALESCE(
                jsonb_agg(
                    jsonb_build_object(
                        'criterion_id', es.criterion_id,
                        'criterion_name', es.criterion_name,
                        'score', es.score,
                        'comment', es.comment_md
                    )
                ) FILTER (WHERE es.id IS NOT NULL), '[]'
            )
        )
        ELSE NULL
    END AS evaluation

FROM interview_turns it
LEFT JOIN evaluations e ON e.turn_id = it.id
LEFT JOIN evaluation_scores es ON es.evaluation_id = e.id
LEFT JOIN user_turn_improvements uti ON uti.interview_turn_id = it.id
WHERE it.session_id = $1 AND it.turn_no > $2
GROUP BY 
    it.id,
    it.turn_no,
    it.actor,
    it.transcript_text,
    it.current_state,
    it.start_at,
    it.end_at,
    uti.corrected_sentence,
    e.id
ORDER BY it.turn_no ASC
LIMIT $3;
