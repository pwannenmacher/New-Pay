-- Create table for approvals of averaged reviewer responses
CREATE TABLE IF NOT EXISTS consolidation_averaged_approvals (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    approved_by_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    approved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(assessment_id, category_id, approved_by_user_id)
);

-- Create indexes for performance
CREATE INDEX idx_consolidation_averaged_approvals_assessment ON consolidation_averaged_approvals(assessment_id);
CREATE INDEX idx_consolidation_averaged_approvals_category ON consolidation_averaged_approvals(category_id);
CREATE INDEX idx_consolidation_averaged_approvals_user ON consolidation_averaged_approvals(approved_by_user_id);
