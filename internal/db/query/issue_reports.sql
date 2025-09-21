-- name: CreateUserIssueReport :one
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
) RETURNING 
    id,
    description,
    category_id,
    status,
    acknowledged,
    comment_count,
    created_at,
    updated_at;

-- name: GetUserIssueReportUserIDAndStatusByID :one
SELECT user_id, status FROM issue_reports WHERE id = $1;