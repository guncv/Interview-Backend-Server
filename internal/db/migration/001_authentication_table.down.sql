DROP INDEX IF EXISTS idx_alerts_severity;
DROP INDEX IF EXISTS idx_alerts_type;
DROP INDEX IF EXISTS idx_alerts_is_resolved;
DROP INDEX IF EXISTS idx_alerts_user_id;

DROP INDEX IF EXISTS idx_user_roles_user_role;
DROP INDEX IF EXISTS idx_user_roles_role;
DROP INDEX IF EXISTS idx_user_roles_user_id;

DROP INDEX IF EXISTS idx_users_email;

DROP INDEX IF EXISTS idx_sessions_token_hash;
DROP INDEX IF EXISTS idx_sessions_user_id;

DROP TABLE IF EXISTS security_alerts;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
