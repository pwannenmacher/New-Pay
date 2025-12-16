-- Add review_consolidation status to self_assessments
-- This status indicates that at least 3 individual reviews are complete
-- and the review team is consolidating the results

-- Add review_consolidation_at timestamp
ALTER TABLE self_assessments 
ADD COLUMN IF NOT EXISTS review_consolidation_at TIMESTAMP;

-- Drop existing CHECK constraint
ALTER TABLE self_assessments 
DROP CONSTRAINT IF EXISTS self_assessments_status_check;

-- Add new CHECK constraint including review_consolidation
ALTER TABLE self_assessments 
ADD CONSTRAINT self_assessments_status_check 
CHECK (status IN ('draft', 'submitted', 'in_review', 'review_consolidation', 'reviewed', 'discussion', 'archived', 'closed'));

COMMENT ON COLUMN self_assessments.review_consolidation_at IS 'Timestamp when status changed to review_consolidation (at least 3 complete reviews available)';
