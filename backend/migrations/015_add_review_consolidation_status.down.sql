-- Remove review_consolidation status from self_assessments

-- Remove review_consolidation_at timestamp
ALTER TABLE self_assessments 
DROP COLUMN IF EXISTS review_consolidation_at;

-- Drop existing CHECK constraint
ALTER TABLE self_assessments 
DROP CONSTRAINT IF EXISTS self_assessments_status_check;

-- Restore old CHECK constraint without review_consolidation
ALTER TABLE self_assessments 
ADD CONSTRAINT self_assessments_status_check 
CHECK (status IN ('draft', 'submitted', 'in_review', 'reviewed', 'discussion', 'archived', 'closed'));
