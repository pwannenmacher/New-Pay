-- Drop triggers
DROP TRIGGER IF EXISTS update_path_level_descriptions_updated_at ON path_level_descriptions;
DROP TRIGGER IF EXISTS update_paths_updated_at ON paths;
DROP TRIGGER IF EXISTS update_levels_updated_at ON levels;
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
DROP TRIGGER IF EXISTS update_criteria_catalogs_updated_at ON criteria_catalogs;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_catalog_changes_entity;
DROP INDEX IF EXISTS idx_catalog_changes_catalog;
DROP INDEX IF EXISTS idx_path_level_desc_level;
DROP INDEX IF EXISTS idx_path_level_desc_path;
DROP INDEX IF EXISTS idx_paths_category;
DROP INDEX IF EXISTS idx_levels_catalog;
DROP INDEX IF EXISTS idx_categories_catalog;
DROP INDEX IF EXISTS idx_criteria_catalogs_validity;

-- Drop tables
DROP TABLE IF EXISTS catalog_changes;
DROP TABLE IF EXISTS path_level_descriptions;
DROP TABLE IF EXISTS paths;
DROP TABLE IF EXISTS levels;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS criteria_catalogs;
