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