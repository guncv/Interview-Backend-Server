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

-- name: UpdateEvaluationCriterion :execrows
UPDATE evaluation_criteria
SET name = $2,
    description_md = $3,
    weight = $4,
    max_score = $5
WHERE id = $1;
