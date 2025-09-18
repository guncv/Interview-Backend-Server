-- name: CreateUserIssueReport :exec
INSERT INTO issue_reports (
    id,
    user_id,
    description,
    status,
    category_id,
    priority,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: ListUserIssueReports :many
SELECT
    ir.id,
    ir.description,
    ir.category_id,
    ic.name as category_name,
    ir.status,
    ir.acknowledged,
    ir.comment_count,
    ir.created_at,
    ir.updated_at
FROM issue_reports ir
LEFT JOIN issue_categories ic
    ON ir.category_id = ic.id AND ic.soft_delete = false
WHERE ir.user_id = $1 AND ir.soft_delete = false
ORDER BY ir.created_at DESC;

-- name: UpdateUserIssueReportByID :execrows
UPDATE issue_reports
SET description = $2,
    category_id = $3,
    updated_at = $4
WHERE id = $1;

-- name: GetUserIssueReportUserIDAndStatusByID :one
SELECT user_id, status FROM issue_reports WHERE id = $1;