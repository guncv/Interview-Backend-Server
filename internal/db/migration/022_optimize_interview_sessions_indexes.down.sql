CREATE INDEX idx_interview_sessions_user_created ON interview_sessions (user_id, soft_delete, created_at DESC);
CREATE INDEX idx_interview_sessions_user_created_id ON interview_sessions (user_id, soft_delete, created_at DESC, id DESC);
CREATE INDEX idx_interview_sessions_user_created_asc ON interview_sessions (user_id, soft_delete, created_at ASC, id ASC);

DROP INDEX IF EXISTS idx_interview_sessions_user_finalized_created_desc;
DROP INDEX IF EXISTS idx_interview_sessions_user_finalized_created_asc;
DROP INDEX IF EXISTS idx_interview_sessions_user_finalized_created_desc_jump;
DROP INDEX IF EXISTS idx_interview_sessions_user_finalized_count;
