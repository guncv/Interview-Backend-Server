DROP INDEX IF EXISTS idx_interview_sessions_user_created;
DROP INDEX IF EXISTS idx_interview_sessions_user_created_id;
DROP INDEX IF EXISTS idx_interview_sessions_user_created_asc;

CREATE INDEX idx_interview_sessions_user_finalized_created_desc 
ON interview_sessions (user_id, soft_delete, finalize_status, created_at DESC, id DESC);

CREATE INDEX idx_interview_sessions_user_finalized_created_asc 
ON interview_sessions (user_id, soft_delete, finalize_status, created_at ASC, id ASC);

CREATE INDEX idx_interview_sessions_user_finalized_created_desc_jump 
ON interview_sessions (user_id, soft_delete, finalize_status, created_at DESC, id DESC);

CREATE INDEX idx_interview_sessions_user_finalized_count 
ON interview_sessions (user_id, soft_delete, finalize_status);

ALTER TABLE interview_sessions ADD COLUMN is_timed_out BOOLEAN DEFAULT FALSE;
ALTER TABLE interview_sessions ADD COLUMN bias_prompt TEXT NOT NULL;
