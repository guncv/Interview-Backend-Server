-- name: GetRubricWithCriteriaByName :many
SELECT
    r.id AS rubric_id,
    r.name AS rubric_name,
    r.description_md AS rubric_description_md,
    r.version_label AS rubric_version_label,
    c.id AS criterion_id,
    c.code AS criterion_code,
    c.name AS criterion_name,
    c.description_md AS criterion_description_md,
    c.weight AS criterion_weight,
    c.max_score AS criterion_max_score
FROM evaluation_rubrics r
JOIN evaluation_criteria c ON r.id = c.rubric_id
WHERE r.name = $1
    AND r.version_label = $2
    AND r.soft_delete = false
ORDER BY c.code;

-- name: ListAllRubricsAndCriteria :many
SELECT
    r.id AS rubric_id,
    r.name AS rubric_name,
    r.description_md AS rubric_description_md,
    r.version_label AS rubric_version_label,
    c.id AS criterion_id,
    c.name AS criterion_name,
    c.description_md AS criterion_description_md,
    c.weight AS criterion_weight,
    c.max_score AS criterion_max_score
FROM evaluation_rubrics r
JOIN evaluation_criteria c ON r.id = c.rubric_id
WHERE r.soft_delete = false AND r.version_label = $1
ORDER BY r.name;