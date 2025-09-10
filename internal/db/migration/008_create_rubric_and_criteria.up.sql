CREATE INDEX idx_rubric_soft_delete ON evaluation_rubrics (soft_delete);
CREATE INDEX idx_criteria_rubric_id ON evaluation_criteria (rubric_id);
CREATE INDEX idx_criteria_code ON evaluation_criteria (code);

WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (
        id,
        name,
        description_md,
        version_label
    )
    VALUES (
        gen_random_uuid(),
        'Technical Interview Rubric (v1)',
        'Evaluates technical interview answers on clarity, correctness, communication, and depth.',
        'v1.0'
    )
    RETURNING id
)

INSERT INTO evaluation_criteria (
    rubric_id,
    code,
    name,
    description_md,
    weight,
    max_score
)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('clarity', 'Clarity of Explanation', 'Was the explanation easy to follow and well-structured?', 0.25, 5.0),
        ('accuracy', 'Technical Accuracy', 'Was the content technically accurate and complete?', 0.30, 5.0),
        ('communication', 'Communication Style', 'Was the tone, pacing, and language professional and effective?', 0.20, 5.0),
        ('depth', 'Technical Depth', 'Did the answer show deep understanding, reasoning, or examples?', 0.25, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);
