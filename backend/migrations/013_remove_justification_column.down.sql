-- Re-add the justification column (for rollback purposes)
ALTER TABLE assessment_responses ADD COLUMN justification TEXT;

COMMENT ON COLUMN assessment_responses.justification IS 'DEPRECATED: Use encrypted_justification_id instead. This column is only for backward compatibility.';
