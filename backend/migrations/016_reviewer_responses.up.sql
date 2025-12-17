-- Create reviewer_responses table for individual reviewer assessments
CREATE TABLE reviewer_responses (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    reviewer_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    encrypted_justification_id BIGINT REFERENCES encrypted_records(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(assessment_id, category_id, reviewer_user_id)
);

CREATE INDEX idx_reviewer_responses_assessment ON reviewer_responses(assessment_id);
CREATE INDEX idx_reviewer_responses_reviewer ON reviewer_responses(reviewer_user_id);
CREATE INDEX idx_reviewer_responses_encrypted_justification ON reviewer_responses(encrypted_justification_id);

COMMENT ON TABLE reviewer_responses IS 'Individual reviewer assessments for self-assessments. Each reviewer creates their own independent review.';
COMMENT ON COLUMN reviewer_responses.encrypted_justification_id IS 'Reference to encrypted justification in encrypted_records table';
COMMENT ON COLUMN reviewer_responses.path_id IS 'Reviewer-selected path, may differ from user selection';
COMMENT ON COLUMN reviewer_responses.level_id IS 'Reviewer-selected level, may differ from user selection';
