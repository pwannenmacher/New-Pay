-- Create self-assessments table
CREATE TABLE self_assessments (
    id SERIAL PRIMARY KEY,
    catalog_id INTEGER NOT NULL REFERENCES criteria_catalogs(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Status of the self-assessment
    -- draft: created by user, in progress
    -- submitted: submitted for review team
    -- in_review: review team started processing
    -- reviewed: review completed, discussion pending (results not yet published)
    -- discussion: review team member discussing results with user (results visible, final comments possible)
    -- archived: classification complete (incl. result documentation), read-only
    -- closed: prematurely closed without completion, can be reverted within 24h
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'in_review', 'reviewed', 'discussion', 'archived', 'closed')),
    
    -- Timestamps for status transitions
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMP,
    in_review_at TIMESTAMP,
    reviewed_at TIMESTAMP,
    discussion_started_at TIMESTAMP,
    archived_at TIMESTAMP,
    closed_at TIMESTAMP,
    previous_status VARCHAR(20), -- For reverting closed status within 24h
    
    -- One active self-assessment per user per catalog
    UNIQUE(catalog_id, user_id)
);

CREATE INDEX idx_self_assessments_catalog ON self_assessments(catalog_id);
CREATE INDEX idx_self_assessments_user ON self_assessments(user_id);
CREATE INDEX idx_self_assessments_status ON self_assessments(status);

-- Trigger to update updated_at timestamp
CREATE TRIGGER update_self_assessments_updated_at
    BEFORE UPDATE ON self_assessments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
