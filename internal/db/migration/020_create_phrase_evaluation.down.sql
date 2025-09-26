DROP TYPE IF EXISTS interview_state_type;

DROP TABLE IF EXISTS phrase_rubric_scores;
DROP TABLE IF EXISTS phrase_evaluations;
DROP TABLE IF EXISTS interview_states;

ALTER TABLE interview_sessions DROP COLUMN IF EXISTS current_state;
ALTER TABLE interview_sessions DROP COLUMN IF EXISTS current_state_id;
ALTER TABLE interview_turns DROP COLUMN IF EXISTS is_score_evaluated;
DROP INDEX IF EXISTS idx_phrase_evaluations_created_at;
ALTER TABLE interview_states DROP COLUMN IF EXISTS last_turn_id;
DROP INDEX IF EXISTS idx_interview_turns_session_id_current_state;