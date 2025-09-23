-- name: CreateReviewComment :exec
INSERT INTO review_comments (
    id,
    session_id,
    author_type,
    author_user_id,
    rating,
    description,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);