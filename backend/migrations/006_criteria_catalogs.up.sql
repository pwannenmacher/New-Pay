-- Create criteria catalogs table
CREATE TABLE criteria_catalogs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Validity period
    valid_from DATE NOT NULL,
    valid_until DATE NOT NULL,
    
    -- Approval workflow phases
    -- draft: only admins can edit
    -- review: admins can edit with change tracking, reviewers and users can view
    -- archived: only admins and reviewers can view (+ users who completed self-assessment)
    phase VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (phase IN ('draft', 'review', 'archived')),
    
    -- Metadata
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    archived_at TIMESTAMP,
    
    -- Ensure no overlapping validity periods for non-archived catalogs
    CONSTRAINT valid_date_range CHECK (valid_from < valid_until)
);

-- Create index for validity checks
CREATE INDEX idx_criteria_catalogs_validity ON criteria_catalogs(valid_from, valid_until, phase);

-- Create categories table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    catalog_id INTEGER NOT NULL REFERENCES criteria_catalogs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(catalog_id, name)
);

CREATE INDEX idx_categories_catalog ON categories(catalog_id);

-- Create levels table (columns in the matrix)
CREATE TABLE levels (
    id SERIAL PRIMARY KEY,
    catalog_id INTEGER NOT NULL REFERENCES criteria_catalogs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    level_number INTEGER NOT NULL,
    description TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(catalog_id, level_number),
    UNIQUE(catalog_id, name)
);

CREATE INDEX idx_levels_catalog ON levels(catalog_id);

-- Create paths table (rows within categories)
CREATE TABLE paths (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(category_id, name)
);

CREATE INDEX idx_paths_category ON paths(category_id);

-- Create path_level_descriptions table (matrix cells)
CREATE TABLE path_level_descriptions (
    id SERIAL PRIMARY KEY,
    path_id INTEGER NOT NULL REFERENCES paths(id) ON DELETE CASCADE,
    level_id INTEGER NOT NULL REFERENCES levels(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(path_id, level_id)
);

CREATE INDEX idx_path_level_desc_path ON path_level_descriptions(path_id);
CREATE INDEX idx_path_level_desc_level ON path_level_descriptions(level_id);

-- Create change log table for tracking changes in review phase
CREATE TABLE catalog_changes (
    id SERIAL PRIMARY KEY,
    catalog_id INTEGER NOT NULL REFERENCES criteria_catalogs(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL, -- 'catalog', 'category', 'path', 'level', 'description'
    entity_id INTEGER NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_catalog_changes_catalog ON catalog_changes(catalog_id);
CREATE INDEX idx_catalog_changes_entity ON catalog_changes(entity_type, entity_id);

-- Add audit triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_criteria_catalogs_updated_at BEFORE UPDATE ON criteria_catalogs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_levels_updated_at BEFORE UPDATE ON levels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_paths_updated_at BEFORE UPDATE ON paths
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_path_level_descriptions_updated_at BEFORE UPDATE ON path_level_descriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
