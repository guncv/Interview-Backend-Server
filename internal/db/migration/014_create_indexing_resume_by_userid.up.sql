CREATE INDEX idx_resumes_user_id ON resumes(user_id);

CREATE INDEX idx_interview_turns_session_turn ON interview_turns(session_id, turn_no);