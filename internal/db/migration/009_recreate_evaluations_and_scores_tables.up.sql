-- Drop old tables if they exist
DROP TABLE IF EXISTS evaluation_scores;
DROP TABLE IF EXISTS evaluations;

-- 1) Create evaluations table
CREATE TABLE evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    turn_id UUID NOT NULL REFERENCES interview_turns(id) ON DELETE CASCADE,
    rubric_id UUID NOT NULL REFERENCES evaluation_rubrics(id) ON DELETE RESTRICT,

    evaluator_user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    overall_score INTEGER NOT NULL CHECK (overall_score BETWEEN 0 AND 5),
    summary_md TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE,

    UNIQUE(session_id, turn_id)
);

-- 2) Create evaluation_scores table
CREATE TABLE evaluation_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evaluation_id UUID NOT NULL REFERENCES evaluations(id) ON DELETE CASCADE,
    criterion_id UUID NOT NULL REFERENCES evaluation_criteria(id) ON DELETE RESTRICT,
    score INTEGER NOT NULL CHECK (score BETWEEN 0 AND 5),
    comment_md TEXT NOT NULL,

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE,
    
    UNIQUE(evaluation_id, criterion_id)
);
