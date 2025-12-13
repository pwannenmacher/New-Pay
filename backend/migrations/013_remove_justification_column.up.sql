-- Check if any non-NULL justification values exist
-- This will cause the migration to fail if there are still unencrypted justifications
DO $$
DECLARE
    unencrypted_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO unencrypted_count
    FROM assessment_responses
    WHERE justification IS NOT NULL;
    
    IF unencrypted_count > 0 THEN
        RAISE EXCEPTION 'Cannot remove justification column: % rows still have non-NULL justification values. Please migrate all data to encrypted storage first.', unencrypted_count;
    END IF;
END $$;

-- Safe to remove the column now
ALTER TABLE assessment_responses DROP COLUMN justification;

COMMENT ON TABLE assessment_responses IS 'Assessment responses - justification is now stored encrypted in encrypted_records table';
