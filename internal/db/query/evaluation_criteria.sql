-- name: CreateEvaluationCriterion :exec
INSERT INTO evaluation_criteria (
    id,
    rubric_id,
    code,
    name,
    description_md,
    weight,
    max_score,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

