-- name: CreatePhraseEvaluation :exec
INSERT INTO phrase_evaluations (
    id,
    session_id,
    state_id,
    state_name,
    overall_score
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
);
