-- name: CheckIssueCategoryExists :one
SELECT EXISTS(
    SELECT 1 FROM issue_categories
    WHERE id = $1 AND soft_delete = false
);

-- name: CreateAdminIssueCategory :exec
INSERT INTO issue_categories (
    id,
    name,
    created_at,
    created_by
)
VALUES ($1, $2, $3, $4);

-- name: ListIssueCategories :many
SELECT id, name
FROM issue_categories
WHERE soft_delete = false
ORDER BY name ASC;