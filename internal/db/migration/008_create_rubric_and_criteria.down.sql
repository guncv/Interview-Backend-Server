DELETE FROM evaluation_criteria
WHERE rubric_code = 'tech_interview_v1';

DROP INDEX IF EXISTS idx_criteria_code;
DROP INDEX IF EXISTS idx_criteria_rubric_id;
DROP INDEX IF EXISTS idx_rubric_soft_delete;
