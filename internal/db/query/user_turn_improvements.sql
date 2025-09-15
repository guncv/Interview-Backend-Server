-- name: CreateUserTurnImprovement :exec
INSERT INTO user_turn_improvements (
    id,
    interview_turn_id,
    corrected_sentence,
    model_version,
    created_at
) VALUES (
    $1, $2, $3, $4, $5
);
