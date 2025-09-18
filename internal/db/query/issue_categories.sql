-- name: CheckIssueCategoryExists :one
SELECT EXISTS(
    SELECT 1 FROM issue_categories
    WHERE id = $1 AND soft_delete = false
);