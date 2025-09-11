BEGIN;

-- GREETING RUBRIC
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'Greeting Rubric',
        'Evaluates greeting phase: friendliness, confidence, and communication style.',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('friendliness', 'Friendliness', 'Was the candidate polite and welcoming?', 0.30, 5.0),
        ('confidence', 'Confidence', 'Did the candidate sound comfortable starting the conversation?', 0.30, 5.0),
        ('communication', 'Communication Style', 'Was the greeting clear, professional, and appropriate?', 0.40, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

-- INTRO RUBRIC
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'Intro Rubric',
        'Evaluates candidate introduction: clarity, relevance, confidence, and communication.',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('clarity', 'Clarity of Background', 'Was the introduction clear and structured?', 0.30, 5.0),
        ('relevance', 'Relevance', 'Did they highlight key experiences or studies relevant to the role?', 0.30, 5.0),
        ('confidence', 'Confidence & Presence', 'Did they present themselves confidently?', 0.20, 5.0),
        ('communication', 'Communication Style', 'Was the tone and pacing professional?', 0.20, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

-- EXPERIENCE RUBRIC
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'Experience Rubric',
        'Evaluates candidate work experience explanations.',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('role_clarity', 'Role Clarity', 'Did they clearly explain their role and responsibilities?', 0.20, 5.0),
        ('achievements', 'Achievements', 'Did they highlight measurable impact or contributions?', 0.25, 5.0),
        ('relevance', 'Technical/Domain Relevance', 'Did their experience align with the skills needed?', 0.30, 5.0),
        ('reflection', 'Reflection/Insights', 'Did they reflect on what they learned or improved?', 0.25, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

-- PROJECT RUBRIC
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'Project Rubric',
        'Evaluates candidate project explanations: problem, implementation, outcomes, and challenges.',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('problem', 'Problem Definition', 'Did they clearly describe the project’s goal or challenge?', 0.20, 5.0),
        ('implementation', 'Implementation Details', 'Did they explain tools, tech stack, or design decisions?', 0.30, 5.0),
        ('outcome', 'Impact/Outcome', 'Was there a result or value created?', 0.25, 5.0),
        ('challenges', 'Challenges & Solutions', 'Did they describe obstacles and how they overcame them?', 0.25, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

-- TECHNICAL RUBRIC
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'Technical Rubric',
        'Evaluates answers to technical questions on correctness, clarity, depth, and problem-solving.',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('correctness', 'Technical Correctness', 'Was the answer technically correct?', 0.40, 5.0),
        ('clarity', 'Clarity of Reasoning', 'Was the explanation structured and logical?', 0.20, 5.0),
        ('depth', 'Technical Depth', 'Did the answer show deeper understanding or examples?', 0.25, 5.0),
        ('problem_solving', 'Problem-Solving Approach', 'Did they show how they think or debug?', 0.15, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

-- BEHAVIORAL RUBRIC
WITH inserted_rubric AS (
    INSERT INTO evaluation_rubrics (id, name, description_md, version_label)
    VALUES (
        gen_random_uuid(),
        'Behavioral Rubric',
        'Evaluates responses to behavioral questions (situation, actions, outcomes, reflection).',
        'v1.0'
    )
    RETURNING id
)
INSERT INTO evaluation_criteria (rubric_id, code, name, description_md, weight, max_score)
SELECT id, code, name, description_md, weight, max_score
FROM inserted_rubric,
    (VALUES
        ('situation', 'Situation/Task Clarity', 'Did they explain the context clearly?', 0.20, 5.0),
        ('actions', 'Actions Taken', 'Did they describe their role and specific actions?', 0.30, 5.0),
        ('outcome', 'Outcome', 'Was there a clear result/impact?', 0.25, 5.0),
        ('reflection', 'Reflection/Learning', 'Did they reflect on what they learned or improved?', 0.25, 5.0)
    ) AS criteria(code, name, description_md, weight, max_score);

COMMIT;

ALTER TABLE evaluations
ADD COLUMN step VARCHAR(50) NOT NULL DEFAULT 'UNKNOWN';