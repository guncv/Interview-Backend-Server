ALTER TABLE interview_sessions ADD COLUMN resume_context JSONB;
ALTER TABLE interview_sessions ADD COLUMN is_completed BOOLEAN DEFAULT FALSE;