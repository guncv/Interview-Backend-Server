-- down.sql

BEGIN;

-- 1) Children that depend on turns/sessions/rubrics/evaluations
DROP TABLE IF EXISTS review_comments;

-- 2) Depends on evaluations & criteria
DROP TABLE IF EXISTS evaluation_scores;

-- 3) Depends on sessions & rubrics
DROP TABLE IF EXISTS evaluations;

-- 4) Depends on rubrics
DROP TABLE IF EXISTS evaluation_criteria;

-- 5) Independent of others (parent of criteria/evaluations)
DROP TABLE IF EXISTS evaluation_rubrics;

-- 6) Depends on sessions
DROP TABLE IF EXISTS interview_turns;

-- 7) Depends on users/resumes/job_requirements
DROP TABLE IF EXISTS interview_sessions;

-- 8) Depends on users
DROP TABLE IF EXISTS job_requirements;

COMMIT;
