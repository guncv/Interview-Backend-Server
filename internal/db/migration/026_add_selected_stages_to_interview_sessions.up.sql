ALTER TABLE interview_sessions
ADD COLUMN selected_stages varchar[] DEFAULT '{}';

COMMENT ON COLUMN interview_sessions.selected_stages IS 'Array of selected interview stages for the session';

