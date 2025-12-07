-- Remove session_id column
DROP INDEX IF EXISTS idx_sessions_session_id;
ALTER TABLE sessions DROP COLUMN IF EXISTS session_id;
