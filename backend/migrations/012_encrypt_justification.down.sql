-- Remove encrypted_justification_id column
ALTER TABLE assessment_responses 
DROP COLUMN IF EXISTS encrypted_justification_id;

-- Restore justification as NOT NULL
ALTER TABLE assessment_responses 
ALTER COLUMN justification SET NOT NULL;

-- Restore the length constraint
ALTER TABLE assessment_responses 
ADD CONSTRAINT check_justification_length CHECK (LENGTH(justification) >= 150);
