-- Create table for category-specific discussion comments
-- These are public comments that will be shown to the assessed user in the discussion phase
CREATE TABLE category_discussion_comments (
    id SERIAL PRIMARY KEY,
    assessment_id INTEGER NOT NULL REFERENCES self_assessments(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    encrypted_comment_id BIGINT REFERENCES encrypted_records(id) ON DELETE SET NULL,
    created_by_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(assessment_id, category_id)
);

CREATE INDEX idx_category_discussion_comments_assessment ON category_discussion_comments(assessment_id);
CREATE INDEX idx_category_discussion_comments_category ON category_discussion_comments(category_id);
