-- name: CreatePhraseEvaluation :exec
INSERT INTO phrase_evaluations (
    session_id,
    state_id,
    state_name,
    overall_score,
    summary_md,
    created_at,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
);
