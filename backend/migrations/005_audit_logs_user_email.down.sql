-- Remove user_email column from audit_logs table
DROP INDEX IF EXISTS idx_audit_logs_user_email;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS user_email;
