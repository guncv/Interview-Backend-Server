DELETE FROM evaluation_criteria
WHERE rubric_name = 'General Interview Rubric (v1)';

DROP INDEX IF EXISTS idx_rubrics_name_softdelete;
