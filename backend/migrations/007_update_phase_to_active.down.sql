-- Revert phase constraint back to 'review'
ALTER TABLE criteria_catalogs DROP CONSTRAINT criteria_catalogs_phase_check;
ALTER TABLE criteria_catalogs ADD CONSTRAINT criteria_catalogs_phase_check CHECK (phase IN ('draft', 'review', 'archived'));

-- Update existing 'active' phases back to 'review' (if any exist)
UPDATE criteria_catalogs SET phase = 'review' WHERE phase = 'active';
