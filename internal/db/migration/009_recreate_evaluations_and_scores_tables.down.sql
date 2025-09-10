DROP TABLE IF EXISTS evaluation_scores;
DROP TABLE IF EXISTS evaluations;

CREATE TABLE evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    rubric_id UUID NOT NULL REFERENCES evaluation_rubrics(id) ON DELETE RESTRICT,
    evaluator_type VARCHAR(10) NOT NULL CHECK (evaluator_type IN ('ai')),
    evaluator_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    overall_score NUMERIC(5,2),
    summary_md TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE
);

CREATE TABLE evaluation_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_id UUID NOT NULL REFERENCES evaluations(id) ON DELETE CASCADE,
    criterion_id UUID NOT NULL REFERENCES evaluation_criteria(id) ON DELETE RESTRICT,
    score NUMERIC(6,2) NOT NULL,
    comment_md TEXT,
    
    UNIQUE(evaluation_id, criterion_id)
);