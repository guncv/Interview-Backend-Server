ALTER TABLE interview_sessions
ADD COLUMN resume_file_name TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_evaluations_session_id_soft_delete ON evaluations (session_id, soft_delete);

CREATE INDEX idx_interview_sessions_user_created ON interview_sessions (user_id, soft_delete, created_at DESC);
CREATE INDEX idx_interview_sessions_user_created_id ON interview_sessions (user_id, soft_delete, created_at DESC, id DESC);

