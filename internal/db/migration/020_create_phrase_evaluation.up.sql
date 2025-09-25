CREATE TABLE interview_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    phrase_type TEXT NOT NULL,
    is_evaluated BOOLEAN DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE
);

CREATE TABLE phrase_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    state_id UUID NOT NULL REFERENCES interview_states(id) ON DELETE CASCADE,
    state_name TEXT NOT NULL,
    overall_score FLOAT NOT NULL,
    summary_md TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    soft_delete BOOLEAN DEFAULT FALSE
);

CREATE TABLE phrase_rubric_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phrase_evaluation_id UUID NOT NULL REFERENCES phrase_evaluations(id) ON DELETE CASCADE,
    criterion_id UUID NOT NULL REFERENCES evaluation_criteria(id) ON DELETE RESTRICT,
    criterion_name TEXT NOT NULL,
    score FLOAT NOT NULL,
    comment_md TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    soft_delete BOOLEAN DEFAULT FALSE,
    UNIQUE(phrase_evaluation_id, criterion_id)
);

ALTER TABLE interview_sessions ADD COLUMN current_state VARCHAR(50);
ALTER TABLE interview_sessions ADD COLUMN current_state_id UUID REFERENCES interview_states(id);
ALTER TABLE interview_turns ADD COLUMN is_score_evaluated BOOLEAN DEFAULT FALSE;