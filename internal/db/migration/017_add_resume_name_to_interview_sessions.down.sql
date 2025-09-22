ALTER TABLE interview_sessions
DROP COLUMN resume_file_name;

DROP INDEX IF EXISTS idx_evaluations_session_id_soft_delete;
DROP INDEX IF EXISTS idx_interview_sessions_user_created;
DROP INDEX IF EXISTS idx_interview_sessions_user_created_id;
DROP INDEX IF EXISTS idx_interview_sessions_user_created_asc;