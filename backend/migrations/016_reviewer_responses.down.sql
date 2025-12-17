-- Drop reviewer_responses table
DROP INDEX IF EXISTS idx_reviewer_responses_encrypted_justification;
DROP INDEX IF EXISTS idx_reviewer_responses_reviewer;
DROP INDEX IF EXISTS idx_reviewer_responses_assessment;
DROP TABLE IF EXISTS reviewer_responses;
