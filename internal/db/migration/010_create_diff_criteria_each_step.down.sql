BEGIN;

DELETE FROM evaluation_criteria
WHERE rubric_id IN (
    SELECT id FROM evaluation_rubrics
    WHERE name IN (
        'Greeting Rubric (v1)',
        'Intro Rubric (v1)',
        'Experience Rubric (v1)',
        'Project Rubric (v1)',
        'Technical Rubric (v1)',
        'Behavioral Rubric (v1)'
    )
);

DELETE FROM evaluation_rubrics
WHERE name IN (
    'Greeting Rubric (v1)',
    'Intro Rubric (v1)',
    'Experience Rubric (v1)',
    'Project Rubric (v1)',
    'Technical Rubric (v1)',
    'Behavioral Rubric (v1)'
);

CREATE INDEX idx_rubrics_name_softdelete ON evaluation_rubrics (name, soft_delete);

WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (
        id,
        name,
        description_md,
        version_label
    )
    VALUES (
        gen_random_uuid(),
        'General Interview Rubric (v1)',
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

COMMIT;

ALTER TABLE evaluations
DROP COLUMN step;