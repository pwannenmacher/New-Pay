-- Add user_email column to audit_logs table to preserve email even after user deletion
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_email VARCHAR(255);

-- Create index on user_email for faster lookups
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_email ON audit_logs(user_email);

-- Backfill existing audit logs with user emails where user_id is not null
UPDATE audit_logs al
SET user_email = u.email
FROM users u
WHERE al.user_id = u.id AND al.user_email IS NULL;
