-- name: CreateAllPhraseRubricScores :exec
INSERT INTO phrase_rubric_scores (
    id,
    phrase_evaluation_id,
    criterion_id,
    criterion_name,
    score,
    comment_md
)
SELECT 
    unnest($1::uuid[]) as id,
    unnest($2::uuid[]) as phrase_evaluation_id,
    unnest($3::uuid[]) as criterion_id,
    unnest($4::text[]) as criterion_name,
    unnest($5::float[]) as score,
    unnest($6::text[]) as comment_md;