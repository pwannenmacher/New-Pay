-- Update existing 'review' phases to 'active' (if any exist)
UPDATE criteria_catalogs SET phase = 'active' WHERE phase = 'review';

-- Update phase constraint to use 'active' instead of 'review'
ALTER TABLE criteria_catalogs DROP CONSTRAINT criteria_catalogs_phase_check;
ALTER TABLE criteria_catalogs ADD CONSTRAINT criteria_catalogs_phase_check CHECK (phase IN ('draft', 'active', 'archived'));
