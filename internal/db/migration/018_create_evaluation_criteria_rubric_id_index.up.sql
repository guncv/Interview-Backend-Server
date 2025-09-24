CREATE INDEX idx_evaluation_criteria_rubric_id ON evaluation_criteria (rubric_id);
CREATE INDEX idx_rubrics_version_softdelete ON evaluation_rubrics (version_label, soft_delete);

CREATE INDEX idx_evaluations_turn_id ON evaluations(turn_id);
CREATE INDEX idx_eval_scores_evaluation_id ON evaluation_scores(evaluation_id);
CREATE INDEX idx_turn_improvements_turn_id ON user_turn_improvements(interview_turn_id);