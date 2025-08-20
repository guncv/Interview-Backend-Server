-- name: CreateEvaluationRubric :exec
INSERT INTO evaluation_rubrics (
    id,
    name,
    description_md,
    version_label,
    created_at
) VALUES (
    $1, $2, $3, $4, $5
);