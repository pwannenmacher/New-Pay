-- Create table for discussion confirmations
-- Tracks confirmations from both reviewer and user (owner) that the discussion took place

CREATE TABLE IF NOT EXISTS discussion_confirmations (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_type VARCHAR(20) NOT NULL CHECK (user_type IN ('reviewer', 'owner')),
    confirmed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure only one confirmation per user per assessment
    UNIQUE(assessment_id, user_id)
);

CREATE INDEX idx_discussion_confirmations_assessment ON discussion_confirmations(assessment_id);
CREATE INDEX idx_discussion_confirmations_user ON discussion_confirmations(user_id);

COMMENT ON TABLE discussion_confirmations IS 'Stores confirmations that the discussion meeting took place';
COMMENT ON COLUMN discussion_confirmations.user_type IS 'Type of user: reviewer (confirms meeting happened) or owner (confirms understanding)';
