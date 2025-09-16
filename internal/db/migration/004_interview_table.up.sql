CREATE TABLE interview_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resume_id UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    position VARCHAR(150) NOT NULL,

    modality VARCHAR(20) NOT NULL DEFAULT 'voice_chat'
            CHECK (modality IN ('voice_chat')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
            CHECK (status IN ('pending','on_going','completed','aborted','cancelled','timed_out')),

    is_consent BOOLEAN NOT NULL DEFAULT TRUE,
    started_at TIMESTAMPTZ DEFAULT now(),
    ended_at TIMESTAMPTZ,
    overall_score NUMERIC(5,2),
    summary_md TEXT,

    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE
);

CREATE TABLE interview_turns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    turn_no BIGINT NOT NULL,
    actor VARCHAR(15) NOT NULL CHECK (actor IN ('user','interviewer')),
    content TEXT,
    transcript_text TEXT NOT NULL,
    stt_confidence NUMERIC(4,3),
    was_interrupted BOOLEAN DEFAULT FALSE,
    start_at VARCHAR(10) NOT NULL,
    end_at VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(session_id, turn_no)
);

CREATE TABLE evaluation_rubrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL,
    description_md TEXT,
    version_label VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE
);

CREATE TABLE evaluation_criteria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rubric_id UUID NOT NULL REFERENCES evaluation_rubrics(id) ON DELETE CASCADE,
    code VARCHAR(60) NOT NULL,
    name VARCHAR(120) NOT NULL,
    description_md TEXT,
    weight NUMERIC(6,3) NOT NULL DEFAULT 1.0,
    max_score NUMERIC(6,2) NOT NULL DEFAULT 5.0,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(rubric_id, code)
);

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

CREATE TABLE review_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    target_turn_id UUID REFERENCES interview_turns(id) ON DELETE CASCADE,
    author_type VARCHAR(10) NOT NULL CHECK (author_type IN ('ai')),
    author_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    category VARCHAR(30) CHECK (category IN ('strength','weakness','improvement','note')),
    body_md TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    soft_delete BOOLEAN DEFAULT FALSE
);
