-- name: CreateReviewComment :execrows
INSERT INTO review_comments (
    id,
    session_id,
    author_type,
    author_user_id,
    rating,
    description,
    created_at,
    updated_at
)
SELECT
    $1, $2, $3, $4, $5, $6, $7, $8
WHERE EXISTS (
    SELECT 1
    FROM interview_sessions
    WHERE id = $2 AND soft_delete = false
);
