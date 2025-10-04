ALTER TABLE interview_sessions DROP COLUMN resume_context;
ALTER TABLE interview_sessions ADD COLUMN is_timed_out BOOLEAN DEFAULT FALSE;