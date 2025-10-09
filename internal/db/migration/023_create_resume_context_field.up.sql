ALTER TABLE interview_sessions ADD COLUMN resume_context JSONB DEFAULT '{}';
ALTER TABLE interview_sessions DROP COLUMN is_timed_out;