ALTER TABLE interview_sessions
ADD COLUMN resume_file_name TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_evaluations_session_id_soft_delete ON evaluations (session_id, soft_delete);
