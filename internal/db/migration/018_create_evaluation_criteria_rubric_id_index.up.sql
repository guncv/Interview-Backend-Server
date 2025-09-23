CREATE INDEX idx_evaluation_criteria_rubric_id ON evaluation_criteria (rubric_id);
CREATE INDEX idx_rubrics_version_softdelete ON evaluation_rubrics (version_label, soft_delete);
