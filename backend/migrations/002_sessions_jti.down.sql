-- Rollback migration for sessions table JTI changes

-- Drop indexes
DROP INDEX IF EXISTS idx_sessions_token_type;
DROP INDEX IF EXISTS idx_sessions_jti;

-- Add back token column
ALTER TABLE sessions ADD COLUMN token TEXT;

-- Drop new columns
ALTER TABLE sessions DROP COLUMN IF EXISTS token_type;
ALTER TABLE sessions DROP COLUMN IF EXISTS jti;
