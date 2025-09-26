CREATE TYPE finalize_status_enum AS ENUM (
    'ongoing',
    'finalizing',
    'finalized',
    'failed'
);

ALTER TABLE interview_sessions
ADD COLUMN finalize_status finalize_status_enum DEFAULT 'ongoing';