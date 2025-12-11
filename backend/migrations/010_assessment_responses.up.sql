-- Create assessment_responses table to store user's path/level selections and justifications
CREATE TABLE assessment_responses (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    justification TEXT NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- One response per category per assessment
    UNIQUE(assessment_id, category_id),
    
    -- Validation: Justification must be at least 150 characters
    CONSTRAINT check_justification_length CHECK (LENGTH(justification) >= 150)
);

-- Indexes for performance
CREATE INDEX idx_assessment_responses_assessment ON assessment_responses(assessment_id);
CREATE INDEX idx_assessment_responses_category ON assessment_responses(category_id);
CREATE INDEX idx_assessment_responses_path ON assessment_responses(path_id);
CREATE INDEX idx_assessment_responses_level ON assessment_responses(level_id);

-- Trigger to update updated_at timestamp
CREATE TRIGGER update_assessment_responses_updated_at
    BEFORE UPDATE ON assessment_responses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
