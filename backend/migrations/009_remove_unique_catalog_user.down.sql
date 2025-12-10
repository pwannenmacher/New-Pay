-- Restore unique constraint (note: this will fail if there are duplicate entries)
ALTER TABLE self_assessments ADD CONSTRAINT self_assessments_catalog_id_user_id_key UNIQUE(catalog_id, user_id);
