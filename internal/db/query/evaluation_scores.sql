-- name: CreateEvaluationScore :exec
INSERT INTO evaluation_scores (
    id,
    evaluation_id,
    criterion_id,
    score,
    comment_md,

    created_at,
    updated_at
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7
);
