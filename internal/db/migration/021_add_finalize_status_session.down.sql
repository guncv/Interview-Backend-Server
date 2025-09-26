DROP TYPE IF EXISTS finalize_status_enum;

ALTER TABLE interview_sessions DROP COLUMN IF EXISTS finalize_status;