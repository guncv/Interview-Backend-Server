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
