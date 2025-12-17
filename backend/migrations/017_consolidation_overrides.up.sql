-- Consolidation overrides table
-- Stores manually adjusted assessment values during review consolidation
-- These values override the averaged reviewer responses for specific categories

CREATE TABLE IF NOT EXISTS consolidation_overrides (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    encrypted_justification_id BIGINT REFERENCES encrypted_records(id) ON DELETE SET NULL,
    created_by_user_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(assessment_id, category_id)
);

CREATE INDEX idx_consolidation_overrides_assessment ON consolidation_overrides(assessment_id);
CREATE INDEX idx_consolidation_overrides_category ON consolidation_overrides(category_id);

COMMENT ON TABLE consolidation_overrides IS 'Manually adjusted values during review consolidation that override averaged reviewer responses';
COMMENT ON COLUMN consolidation_overrides.encrypted_justification_id IS 'Reference to encrypted justification explaining the override';
