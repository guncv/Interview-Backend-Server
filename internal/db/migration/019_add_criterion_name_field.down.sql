ALTER TABLE evaluation_scores DROP COLUMN criterion_name;
ALTER TABLE interview_turns ADD COLUMN content TEXT NOT NULL;