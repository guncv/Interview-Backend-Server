CREATE TABLE resumes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name    TEXT NOT NULL,
    
    storage_key  TEXT NOT NULL,
    mime_type    VARCHAR(100) NOT NULL DEFAULT 'application/pdf'
                CHECK (mime_type = 'application/pdf'),
    byte_size    INTEGER NOT NULL CHECK (byte_size > 0 AND byte_size <= 5 * 1024 * 1024),
    is_default   BOOLEAN NOT NULL DEFAULT FALSE,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX uniq_default_resume_per_user
    ON resumes(user_id)
    WHERE is_default;

CREATE INDEX idx_resumes_user_updated_at
    ON resumes (user_id, updated_at DESC)
    WHERE is_default = FALSE;
