-- Add session_id column to group access and refresh tokens from same login
ALTER TABLE sessions ADD COLUMN session_id VARCHAR(255);

-- Create index on session_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);

-- Migrate existing sessions: generate unique session_id for each existing entry
-- In production, you might want to delete old sessions instead
UPDATE sessions SET session_id = gen_random_uuid()::text WHERE session_id IS NULL;

-- Make session_id NOT NULL after migration
ALTER TABLE sessions ALTER COLUMN session_id SET NOT NULL;
