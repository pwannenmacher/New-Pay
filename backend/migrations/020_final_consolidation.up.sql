-- Create table for final consolidation comments
CREATE TABLE IF NOT EXISTS final_consolidations (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL UNIQUE REFERENCES self_assessments(id) ON DELETE CASCADE,
    encrypted_comment_id BIGINT REFERENCES encrypted_records(id) ON DELETE SET NULL,
    created_by_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create table for final consolidation approvals
CREATE TABLE IF NOT EXISTS final_consolidation_approvals (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    approved_by_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    approved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(assessment_id, approved_by_user_id)
);

-- Create indexes for performance
CREATE INDEX idx_final_consolidations_assessment ON final_consolidations(assessment_id);
CREATE INDEX idx_final_consolidation_approvals_assessment ON final_consolidation_approvals(assessment_id);
CREATE INDEX idx_final_consolidation_approvals_user ON final_consolidation_approvals(approved_by_user_id);
