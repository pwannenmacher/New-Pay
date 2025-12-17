-- Consolidation override approvals table
-- Tracks which reviewers have approved each override
-- Multiple reviewers can approve the same override

CREATE TABLE IF NOT EXISTS consolidation_override_approvals (
    id SERIAL PRIMARY KEY,
    override_id INTEGER NOT NULL REFERENCES consolidation_overrides(id) ON DELETE CASCADE,
    approved_by_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    approved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(override_id, approved_by_user_id)
);

CREATE INDEX idx_consolidation_override_approvals_override ON consolidation_override_approvals(override_id);
CREATE INDEX idx_consolidation_override_approvals_user ON consolidation_override_approvals(approved_by_user_id);

COMMENT ON TABLE consolidation_override_approvals IS 'Tracks reviewer approvals for consolidation overrides';
COMMENT ON COLUMN consolidation_override_approvals.override_id IS 'Reference to the override being approved';
COMMENT ON COLUMN consolidation_override_approvals.approved_by_user_id IS 'Reviewer who approved this override';
