CREATE TABLE issue_categories (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    soft_delete     BOOLEAN DEFAULT FALSE
);

CREATE TABLE issue_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    description     TEXT NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    category_id     UUID REFERENCES issue_categories(id) ON DELETE CASCADE NOT NULL,
    priority        VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    assigned_to     UUID REFERENCES users(id) ON DELETE SET NULL,
    comment_count   INT NOT NULL DEFAULT 0,
    acknowledged    BOOLEAN NOT NULL DEFAULT false,
    acknowledged_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    soft_delete     BOOLEAN DEFAULT FALSE
);

CREATE TABLE issue_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id    UUID NOT NULL REFERENCES issue_reports(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    comment     TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_issue_reports_user_soft_delete_created_at ON issue_reports (user_id, soft_delete, created_at DESC);
CREATE INDEX idx_issue_categories_id_soft_delete ON issue_categories (id) WHERE soft_delete = false;
CREATE INDEX idx_issue_categories_soft_delete ON issue_categories (soft_delete);