CREATE TABLE user_turn_improvements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_turn_id UUID NOT NULL REFERENCES interview_turns(id) ON DELETE CASCADE,
    corrected_sentence TEXT NOT NULL,
    model_version VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
