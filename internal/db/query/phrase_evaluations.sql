-- name: CreatePhraseEvaluation :exec
INSERT INTO phrase_evaluations (
    id,
    session_id,
    state_id,
    state_name,
    overall_score
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
);

-- name: GetPhraseEvaluationsWithCriteriaBySessionID :many
WITH latest_evaluations AS (
    SELECT 
        pe.id,
        pe.state_id,
        pe.state_name,
        pe.overall_score,
        pe.created_at,
        ROW_NUMBER() OVER (PARTITION BY pe.state_name ORDER BY pe.created_at DESC) as rn
    FROM phrase_evaluations pe
    WHERE pe.session_id = $1 AND pe.soft_delete = false
)
SELECT 
    le.state_id,
    le.state_name,
    le.overall_score,
    COALESCE(
        JSON_AGG(
            JSON_BUILD_OBJECT(
                'criteria_id', prs.criterion_id,
                'criteria_name', prs.criterion_name,
                'criteria_score', prs.score,
                'criteria_comment', COALESCE(prs.comment_md, '')
            )
        ) FILTER (WHERE prs.criterion_id IS NOT NULL),
        '[]'::json
    ) as criteria
FROM latest_evaluations le
LEFT JOIN phrase_rubric_scores prs ON le.id = prs.phrase_evaluation_id AND prs.soft_delete = false
WHERE le.rn = 1
GROUP BY le.id, le.state_id, le.state_name, le.overall_score, le.created_at
ORDER BY le.created_at ASC;