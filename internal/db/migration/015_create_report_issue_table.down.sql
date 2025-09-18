DROP TABLE IF EXISTS issue_reports;
DROP TABLE IF EXISTS issue_categories;
DROP TABLE IF EXISTS issue_comments;

DROP INDEX IF EXISTS idx_issue_reports_user_soft_delete_created_at;
DROP INDEX IF EXISTS idx_issue_categories_id_soft_delete;

DROP INDEX IF EXISTS idx_issue_categories_soft_delete;