package repository

import (
	"database/sql"
	"fmt"
	"new-pay/internal/models"
	"time"
)

// CatalogRepository handles database operations for criteria catalogs
type CatalogRepository struct {
	db *sql.DB
}

// NewCatalogRepository creates a new catalog repository
func NewCatalogRepository(db *sql.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

// CreateCatalog creates a new criteria catalog
func (r *CatalogRepository) CreateCatalog(catalog *models.CriteriaCatalog) error {
	query := `
		INSERT INTO criteria_catalogs (name, description, valid_from, valid_until, phase, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		catalog.Name,
		catalog.Description,
		catalog.ValidFrom,
		catalog.ValidUntil,
		catalog.Phase,
		catalog.CreatedBy,
	).Scan(&catalog.ID, &catalog.CreatedAt, &catalog.UpdatedAt)

	return err
}

// GetCatalogByID retrieves a catalog by ID
func (r *CatalogRepository) GetCatalogByID(id uint) (*models.CriteriaCatalog, error) {
	query := `
		SELECT id, name, description, valid_from, valid_until, phase, created_by,
		       created_at, updated_at, published_at, archived_at
		FROM criteria_catalogs
		WHERE id = $1
	`

	catalog := &models.CriteriaCatalog{}
	err := r.db.QueryRow(query, id).Scan(
		&catalog.ID,
		&catalog.Name,
		&catalog.Description,
		&catalog.ValidFrom,
		&catalog.ValidUntil,
		&catalog.Phase,
		&catalog.CreatedBy,
		&catalog.CreatedAt,
		&catalog.UpdatedAt,
		&catalog.PublishedAt,
		&catalog.ArchivedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return catalog, err
}

// GetAllCatalogs retrieves all catalogs
func (r *CatalogRepository) GetAllCatalogs() ([]models.CriteriaCatalog, error) {
	query := `
		SELECT id, name, description, valid_from, valid_until, phase, created_by,
		       created_at, updated_at, published_at, archived_at
		FROM criteria_catalogs
		ORDER BY valid_from DESC, created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catalogs := []models.CriteriaCatalog{} // Initialize to empty slice instead of nil
	for rows.Next() {
		var catalog models.CriteriaCatalog
		err := rows.Scan(
			&catalog.ID,
			&catalog.Name,
			&catalog.Description,
			&catalog.ValidFrom,
			&catalog.ValidUntil,
			&catalog.Phase,
			&catalog.CreatedBy,
			&catalog.CreatedAt,
			&catalog.UpdatedAt,
			&catalog.PublishedAt,
			&catalog.ArchivedAt,
		)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, catalog)
	}

	return catalogs, rows.Err()
}

// GetCatalogsByPhase retrieves catalogs by phase
func (r *CatalogRepository) GetCatalogsByPhase(phase string) ([]models.CriteriaCatalog, error) {
	query := `
		SELECT id, name, description, valid_from, valid_until, phase, created_by,
		       created_at, updated_at, published_at, archived_at
		FROM criteria_catalogs
		WHERE phase = $1
		ORDER BY valid_from DESC, created_at DESC
	`

	rows, err := r.db.Query(query, phase)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catalogs := []models.CriteriaCatalog{} // Initialize to empty slice instead of nil
	for rows.Next() {
		var catalog models.CriteriaCatalog
		err := rows.Scan(
			&catalog.ID,
			&catalog.Name,
			&catalog.Description,
			&catalog.ValidFrom,
			&catalog.ValidUntil,
			&catalog.Phase,
			&catalog.CreatedBy,
			&catalog.CreatedAt,
			&catalog.UpdatedAt,
			&catalog.PublishedAt,
			&catalog.ArchivedAt,
		)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, catalog)
	}

	return catalogs, rows.Err()
}

// CheckOverlappingCatalogs checks if a catalog's validity period overlaps with existing non-archived catalogs
func (r *CatalogRepository) CheckOverlappingCatalogs(validFrom, validUntil time.Time, excludeID *uint) (bool, error) {
	query := `
		SELECT COUNT(*) FROM criteria_catalogs
		WHERE phase != 'archived'
		AND (
			(valid_from <= $1 AND valid_until > $1) OR
			(valid_from < $2 AND valid_until >= $2) OR
			(valid_from >= $1 AND valid_until <= $2)
		)
	`

	args := []interface{}{validFrom, validUntil}

	if excludeID != nil {
		query += " AND id != $3"
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateCatalog updates a catalog
func (r *CatalogRepository) UpdateCatalog(catalog *models.CriteriaCatalog) error {
	query := `
		UPDATE criteria_catalogs
		SET name = $1, description = $2, valid_from = $3, valid_until = $4
		WHERE id = $5
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		catalog.Name,
		catalog.Description,
		catalog.ValidFrom,
		catalog.ValidUntil,
		catalog.ID,
	).Scan(&catalog.UpdatedAt)

	return err
}

// UpdateCatalogPhase updates the phase of a catalog
func (r *CatalogRepository) UpdateCatalogPhase(id uint, phase string) error {
	query := `UPDATE criteria_catalogs SET phase = $1`
	args := []interface{}{phase, id}

	switch phase {
	case "review":
		query += `, published_at = $3 WHERE id = $2`
		args = []interface{}{phase, id, time.Now()}
	case "archived":
		query += `, archived_at = $3 WHERE id = $2`
		args = []interface{}{phase, id, time.Now()}
	default:
		query += ` WHERE id = $2`
	}

	_, err := r.db.Exec(query, args...)
	return err
}

// DeleteCatalog deletes a catalog (only allowed in draft phase)
func (r *CatalogRepository) DeleteCatalog(id uint) error {
	query := `DELETE FROM criteria_catalogs WHERE id = $1 AND phase = 'draft'`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("catalog not found or not in draft phase")
	}

	return nil
}

// CreateCategory creates a new category
func (r *CatalogRepository) CreateCategory(category *models.Category) error {
	query := `
		INSERT INTO categories (catalog_id, name, description, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		category.CatalogID,
		category.Name,
		category.Description,
		category.SortOrder,
	).Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)

	return err
}

// GetCategoriesByCatalogID retrieves all categories for a catalog
func (r *CatalogRepository) GetCategoriesByCatalogID(catalogID uint) ([]models.Category, error) {
	query := `
		SELECT id, catalog_id, name, description, sort_order, created_at, updated_at
		FROM categories
		WHERE catalog_id = $1
		ORDER BY sort_order, name
	`

	rows, err := r.db.Query(query, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID,
			&category.CatalogID,
			&category.Name,
			&category.Description,
			&category.SortOrder,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}

// UpdateCategory updates a category
func (r *CatalogRepository) UpdateCategory(category *models.Category) error {
	query := `
		UPDATE categories
		SET name = $1, description = $2, sort_order = $3
		WHERE id = $4
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		category.Name,
		category.Description,
		category.SortOrder,
		category.ID,
	).Scan(&category.UpdatedAt)

	return err
}

// DeleteCategory deletes a category
func (r *CatalogRepository) DeleteCategory(id uint) error {
	query := `DELETE FROM categories WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// CreateLevel creates a new level
func (r *CatalogRepository) CreateLevel(level *models.Level) error {
	query := `
		INSERT INTO levels (catalog_id, name, level_number, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		level.CatalogID,
		level.Name,
		level.LevelNumber,
		level.Description,
	).Scan(&level.ID, &level.CreatedAt, &level.UpdatedAt)

	return err
}

// GetLevelsByCatalogID retrieves all levels for a catalog
func (r *CatalogRepository) GetLevelsByCatalogID(catalogID uint) ([]models.Level, error) {
	query := `
		SELECT id, catalog_id, name, level_number, description, created_at, updated_at
		FROM levels
		WHERE catalog_id = $1
		ORDER BY level_number
	`

	rows, err := r.db.Query(query, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	levels := []models.Level{}
	for rows.Next() {
		var level models.Level
		err := rows.Scan(
			&level.ID,
			&level.CatalogID,
			&level.Name,
			&level.LevelNumber,
			&level.Description,
			&level.CreatedAt,
			&level.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		levels = append(levels, level)
	}

	return levels, rows.Err()
}

// UpdateLevel updates a level
func (r *CatalogRepository) UpdateLevel(level *models.Level) error {
	query := `
		UPDATE levels
		SET name = $1, level_number = $2, description = $3
		WHERE id = $4
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		level.Name,
		level.LevelNumber,
		level.Description,
		level.ID,
	).Scan(&level.UpdatedAt)

	return err
}

// DeleteLevel deletes a level
func (r *CatalogRepository) DeleteLevel(id uint) error {
	query := `DELETE FROM levels WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// CreatePath creates a new path
func (r *CatalogRepository) CreatePath(path *models.Path) error {
	query := `
		INSERT INTO paths (category_id, name, description, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		path.CategoryID,
		path.Name,
		path.Description,
		path.SortOrder,
	).Scan(&path.ID, &path.CreatedAt, &path.UpdatedAt)

	return err
}

// GetPathsByCategoryID retrieves all paths for a category
func (r *CatalogRepository) GetPathsByCategoryID(categoryID uint) ([]models.Path, error) {
	query := `
		SELECT id, category_id, name, description, sort_order, created_at, updated_at
		FROM paths
		WHERE category_id = $1
		ORDER BY sort_order, name
	`

	rows, err := r.db.Query(query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := []models.Path{}
	for rows.Next() {
		var path models.Path
		err := rows.Scan(
			&path.ID,
			&path.CategoryID,
			&path.Name,
			&path.Description,
			&path.SortOrder,
			&path.CreatedAt,
			&path.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	return paths, rows.Err()
}

// UpdatePath updates a path
func (r *CatalogRepository) UpdatePath(path *models.Path) error {
	query := `
		UPDATE paths
		SET name = $1, description = $2, sort_order = $3
		WHERE id = $4
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		path.Name,
		path.Description,
		path.SortOrder,
		path.ID,
	).Scan(&path.UpdatedAt)

	return err
}

// DeletePath deletes a path
func (r *CatalogRepository) DeletePath(id uint) error {
	query := `DELETE FROM paths WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// CreatePathLevelDescription creates or updates a description for a path-level combination
func (r *CatalogRepository) CreatePathLevelDescription(desc *models.PathLevelDescription) error {
	query := `
		INSERT INTO path_level_descriptions (path_id, level_id, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (path_id, level_id)
		DO UPDATE SET description = $3
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		desc.PathID,
		desc.LevelID,
		desc.Description,
	).Scan(&desc.ID, &desc.CreatedAt, &desc.UpdatedAt)

	return err
}

// GetDescriptionsByPathID retrieves all descriptions for a path
func (r *CatalogRepository) GetDescriptionsByPathID(pathID uint) ([]models.PathLevelDescription, error) {
	query := `
		SELECT id, path_id, level_id, description, created_at, updated_at
		FROM path_level_descriptions
		WHERE path_id = $1
		ORDER BY level_id
	`

	rows, err := r.db.Query(query, pathID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	descriptions := []models.PathLevelDescription{}
	for rows.Next() {
		var desc models.PathLevelDescription
		err := rows.Scan(
			&desc.ID,
			&desc.PathID,
			&desc.LevelID,
			&desc.Description,
			&desc.CreatedAt,
			&desc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		descriptions = append(descriptions, desc)
	}

	return descriptions, rows.Err()
}

// GetDescriptionsByCatalogID retrieves all descriptions for a catalog
func (r *CatalogRepository) GetDescriptionsByCatalogID(catalogID uint) ([]models.PathLevelDescription, error) {
	query := `
		SELECT pld.id, pld.path_id, pld.level_id, pld.description, pld.created_at, pld.updated_at
		FROM path_level_descriptions pld
		JOIN paths p ON pld.path_id = p.id
		JOIN categories c ON p.category_id = c.id
		WHERE c.catalog_id = $1
		ORDER BY c.sort_order, p.sort_order, pld.level_id
	`

	rows, err := r.db.Query(query, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	descriptions := []models.PathLevelDescription{}
	for rows.Next() {
		var desc models.PathLevelDescription
		err := rows.Scan(
			&desc.ID,
			&desc.PathID,
			&desc.LevelID,
			&desc.Description,
			&desc.CreatedAt,
			&desc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		descriptions = append(descriptions, desc)
	}

	return descriptions, rows.Err()
}

// DeletePathLevelDescription deletes a description
func (r *CatalogRepository) DeletePathLevelDescription(id uint) error {
	query := `DELETE FROM path_level_descriptions WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// LogChange logs a change to the catalog
func (r *CatalogRepository) LogChange(change *models.CatalogChange) error {
	query := `
		INSERT INTO catalog_changes (catalog_id, entity_type, entity_id, field_name, old_value, new_value, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, changed_at
	`

	err := r.db.QueryRow(
		query,
		change.CatalogID,
		change.EntityType,
		change.EntityID,
		change.FieldName,
		change.OldValue,
		change.NewValue,
		change.ChangedBy,
	).Scan(&change.ID, &change.ChangedAt)

	return err
}

// GetChangesByCatalogID retrieves all changes for a catalog
func (r *CatalogRepository) GetChangesByCatalogID(catalogID uint) ([]models.CatalogChange, error) {
	query := `
		SELECT id, catalog_id, entity_type, entity_id, field_name, old_value, new_value, changed_by, changed_at
		FROM catalog_changes
		WHERE catalog_id = $1
		ORDER BY changed_at DESC
	`

	rows, err := r.db.Query(query, catalogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	changes := []models.CatalogChange{}
	for rows.Next() {
		var change models.CatalogChange
		err := rows.Scan(
			&change.ID,
			&change.CatalogID,
			&change.EntityType,
			&change.EntityID,
			&change.FieldName,
			&change.OldValue,
			&change.NewValue,
			&change.ChangedBy,
			&change.ChangedAt,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, rows.Err()
}

// GetCatalogWithDetails retrieves a catalog with all nested entities
func (r *CatalogRepository) GetCatalogWithDetails(id uint) (*models.CatalogWithDetails, error) {
	catalog, err := r.GetCatalogByID(id)
	if err != nil || catalog == nil {
		return nil, err
	}

	catalogWithDetails := &models.CatalogWithDetails{
		CriteriaCatalog: *catalog,
	}

	// Get levels
	levels, err := r.GetLevelsByCatalogID(id)
	if err != nil {
		return nil, err
	}
	catalogWithDetails.Levels = levels

	// Get categories with paths
	categories, err := r.GetCategoriesByCatalogID(id)
	if err != nil {
		return nil, err
	}

	categoriesWithPaths := []models.CategoryWithPaths{}
	for _, category := range categories {
		paths, err := r.GetPathsByCategoryID(category.ID)
		if err != nil {
			return nil, err
		}

		pathsWithDescriptions := []models.PathWithDescriptions{}
		for _, path := range paths {
			descriptions, err := r.GetDescriptionsByPathID(path.ID)
			if err != nil {
				return nil, err
			}

			pathsWithDescriptions = append(pathsWithDescriptions, models.PathWithDescriptions{
				Path:         path,
				Descriptions: descriptions,
			})
		}

		categoriesWithPaths = append(categoriesWithPaths, models.CategoryWithPaths{
			Category: category,
			Paths:    pathsWithDescriptions,
		})
	}

	catalogWithDetails.Categories = categoriesWithPaths

	return catalogWithDetails, nil
}
