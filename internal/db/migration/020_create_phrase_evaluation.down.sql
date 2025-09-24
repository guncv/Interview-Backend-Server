DROP TYPE IF EXISTS interview_state_type;

DROP TABLE IF EXISTS phrase_rubric_scores;
DROP TABLE IF EXISTS phrase_evaluations;
DROP TABLE IF EXISTS interview_states;

ALTER TABLE interview_sessions DROP COLUMN IF EXISTS current_state;
ALTER TABLE interview_sessions DROP COLUMN IF EXISTS current_state_id;
