BEGIN;

DELETE FROM evaluation_criteria
WHERE rubric_id IN (
    SELECT id FROM evaluation_rubrics
    WHERE name = 'General Interview Rubric (v1)'
);

DELETE FROM evaluation_rubrics
WHERE name = 'General Interview Rubric (v1)';

-- GENERAL RUBRIC (Fallback for UNKNOWN step)
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'General Rubric',
        'Used when step is UNKNOWN. Evaluates general clarity, correctness, communication, and depth.',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('clarity', 'Clarity of Explanation', 'Was the explanation easy to follow and well-structured?', 0.25, 5.0),
        ('accuracy', 'Technical Accuracy', 'Was the content technically accurate and complete?', 0.30, 5.0),
        ('communication', 'Communication Style', 'Was the tone, pacing, and language professional and effective?', 0.20, 5.0),
        ('depth', 'Technical Depth', 'Did the answer show deep understanding, reasoning, or examples?', 0.25, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

ALTER TABLE interview_turns
ADD COLUMN current_state VARCHAR(50) NOT NULL DEFAULT 'UNKNOWN';

CREATE INDEX idx_interview_turns_session_actor_turn
ON interview_turns (session_id, actor, turn_no DESC);

COMMIT;