ALTER TABLE interview_sessions ADD COLUMN resume_context JSONB;
ALTER TABLE interview_sessions DROP COLUMN is_timed_out;