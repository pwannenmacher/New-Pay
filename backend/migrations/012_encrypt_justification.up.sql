-- Add encrypted_justification_id column to assessment_responses
ALTER TABLE assessment_responses 
ADD COLUMN encrypted_justification_id BIGINT REFERENCES encrypted_records(id);

-- Drop the length constraint on justification as it will be encrypted
ALTER TABLE assessment_responses 
DROP CONSTRAINT IF EXISTS check_justification_length;

-- Make justification nullable (will be deprecated in favor of encrypted version)
ALTER TABLE assessment_responses 
ALTER COLUMN justification DROP NOT NULL;

-- Index for faster lookups
CREATE INDEX idx_assessment_responses_encrypted_justification 
ON assessment_responses(encrypted_justification_id);

COMMENT ON COLUMN assessment_responses.encrypted_justification_id IS 'Reference to encrypted justification in encrypted_records table';
COMMENT ON COLUMN assessment_responses.justification IS 'DEPRECATED: Use encrypted_justification_id instead. Will be removed in future version.';
