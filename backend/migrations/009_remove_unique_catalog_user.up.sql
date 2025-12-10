-- Remove unique constraint to allow multiple self-assessments per user per catalog
-- Users can create new assessments for the same catalog once previous ones are closed
ALTER TABLE self_assessments DROP CONSTRAINT IF EXISTS self_assessments_catalog_id_user_id_key;
