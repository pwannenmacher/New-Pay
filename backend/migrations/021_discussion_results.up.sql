-- Create discussion results table
CREATE TABLE IF NOT EXISTS discussion_results (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL UNIQUE REFERENCES self_assessments(id) ON DELETE CASCADE,
    
    -- Overall weighted result
    weighted_overall_level_number DECIMAL(5,2) NOT NULL,
    weighted_overall_level_id INTEGER NOT NULL REFERENCES levels(id),
    
    -- Encrypted final comment from reviewers (stored in encrypted_records)
    encrypted_final_comment_id BIGINT REFERENCES encrypted_records(id) ON DELETE SET NULL,
    
    -- Encrypted discussion note from user (stored in encrypted_records)
    encrypted_discussion_note_id BIGINT REFERENCES encrypted_records(id) ON DELETE SET NULL,
    
    -- User approval
    user_approved_at TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create discussion category results table (one row per category)
CREATE TABLE IF NOT EXISTS discussion_category_results (
    id SERIAL PRIMARY KEY,
    discussion_result_id INTEGER NOT NULL REFERENCES discussion_results(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    
    -- User's self-assessment
    user_level_id INTEGER REFERENCES levels(id),
    
    -- Reviewer assessment
    reviewer_level_id INTEGER NOT NULL REFERENCES levels(id),
    reviewer_level_number DECIMAL(5,2) NOT NULL,
    
    -- Encrypted justification (stored in encrypted_records)
    encrypted_justification_id BIGINT REFERENCES encrypted_records(id) ON DELETE SET NULL,
    
    -- Metadata
    is_override BOOLEAN NOT NULL DEFAULT FALSE,
    
    UNIQUE(discussion_result_id, category_id)
);

-- Create discussion reviewers table (which reviewers participated)
CREATE TABLE IF NOT EXISTS discussion_reviewers (
    id SERIAL PRIMARY KEY,
    discussion_result_id INTEGER NOT NULL REFERENCES discussion_results(id) ON DELETE CASCADE,
    reviewer_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewer_name VARCHAR(255) NOT NULL,
    
    UNIQUE(discussion_result_id, reviewer_user_id)
);

CREATE INDEX IF NOT EXISTS idx_discussion_results_assessment ON discussion_results(assessment_id);
CREATE INDEX IF NOT EXISTS idx_discussion_category_results_discussion ON discussion_category_results(discussion_result_id);
CREATE INDEX IF NOT EXISTS idx_discussion_reviewers_discussion ON discussion_reviewers(discussion_result_id);

-- Trigger to update updated_at timestamp
CREATE TRIGGER update_discussion_results_updated_at
    BEFORE UPDATE ON discussion_results
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
