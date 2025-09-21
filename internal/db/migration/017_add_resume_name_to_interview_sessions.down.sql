ALTER TABLE interview_sessions
DROP COLUMN resume_file_name;

DROP INDEX IF EXISTS idx_evaluations_session_id_soft_delete;
