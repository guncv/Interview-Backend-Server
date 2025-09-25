-- name: CreateEvaluation :exec
INSERT INTO evaluations (
    id,
    session_id,
    turn_id,
    rubric_id,
    evaluator_user_id,
    current_state,
    overall_score,
    summary_md,
    
    created_at,
    updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: GetAllEvaluationsBySessionID :many
SELECT overall_score, summary_md
FROM evaluations
WHERE session_id = $1 and soft_delete = false;

-- name: GetEvaluationSummaryJsonBySessionAndState :one
WITH evals AS (
    SELECT e.id, e.overall_score, e.session_id
    FROM evaluations e
    WHERE e.session_id = $1
        AND e.current_state = $2
        AND e.soft_delete = false
)
SELECT jsonb_build_object(
    'evaluation_avg_score', (SELECT AVG(overall_score) FROM evals),
    'criteria', jsonb_agg(
        jsonb_build_object(
            'criteria_id', es.criterion_id,
            'criteria_name', es.criterion_name,
            'criteria_score', (
                SELECT jsonb_agg(
                    jsonb_build_object(
                        'score', es2.score,
                        'comment', es2.comment_md
                    )
                    ORDER BY es2.created_at
                )
                FROM evaluation_scores es2
                WHERE es2.evaluation_id = ev.id
                    AND es2.criterion_id = es.criterion_id
                    AND es2.soft_delete = false
            )
        )
    )
) AS result
FROM evals ev
JOIN evaluation_scores es ON es.evaluation_id = ev.id
WHERE es.soft_delete = false
GROUP BY ev.session_id, ev.id;

