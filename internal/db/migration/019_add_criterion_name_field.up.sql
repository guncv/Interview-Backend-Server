ALTER TABLE evaluation_scores ADD COLUMN criterion_name TEXT NOT NULL;
ALTER TABLE interview_turns DROP COLUMN content;