-- Migration to change sessions table from token to jti
-- This migration updates the sessions table to store JTI instead of full token

-- Add new columns
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS jti VARCHAR(255);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS token_type VARCHAR(20) DEFAULT 'refresh';

-- Copy existing token data to jti column (for backward compatibility during migration)
-- In production, you would need to invalidate all existing sessions
UPDATE sessions SET jti = LEFT(token, 255) WHERE jti IS NULL;

-- Make jti NOT NULL after data migration
ALTER TABLE sessions ALTER COLUMN jti SET NOT NULL;

-- Drop the old token column
ALTER TABLE sessions DROP COLUMN IF EXISTS token;

-- Create index on jti for faster lookups
CREATE INDEX IF NOT EXISTS idx_sessions_jti ON sessions(jti);
CREATE INDEX IF NOT EXISTS idx_sessions_token_type ON sessions(token_type);
