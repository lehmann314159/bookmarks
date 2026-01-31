package repository

import (
	"database/sql"
	"strings"

	"github.com/lehmann314159/bookmarks/internal/models"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Categories

func (r *Repository) GetCategories() ([]models.Category, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.name, c.description, c.created_at,
		       (SELECT COUNT(*) FROM sites WHERE category_id = c.id) as site_count
		FROM categories c
		ORDER BY c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		var desc sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &desc, &c.CreatedAt, &c.SiteCount); err != nil {
			return nil, err
		}
		c.Description = desc.String
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *Repository) GetCategory(id int64) (*models.Category, error) {
	var c models.Category
	var desc sql.NullString
	err := r.db.QueryRow(`
		SELECT c.id, c.name, c.description, c.created_at,
		       (SELECT COUNT(*) FROM sites WHERE category_id = c.id) as site_count
		FROM categories c WHERE c.id = ?
	`, id).Scan(&c.ID, &c.Name, &desc, &c.CreatedAt, &c.SiteCount)
	if err != nil {
		return nil, err
	}
	c.Description = desc.String
	return &c, nil
}

func (r *Repository) CreateCategory(name, description string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO categories (name, description) VALUES (?, ?)`, name, nullString(description))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) UpdateCategory(id int64, name, description string) error {
	_, err := r.db.Exec(`UPDATE categories SET name = ?, description = ? WHERE id = ?`, name, nullString(description), id)
	return err
}

func (r *Repository) DeleteCategory(id int64) error {
	_, err := r.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	return err
}

// Sites

func (r *Repository) GetSites(categoryID *int64) ([]models.Site, error) {
	query := `
		SELECT s.id, s.category_id, COALESCE(c.name, '') as category_name,
		       s.domain, s.name, s.description, s.created_at,
		       (SELECT COUNT(*) FROM pages WHERE site_id = s.id) as page_count
		FROM sites s
		LEFT JOIN categories c ON s.category_id = c.id
	`
	args := []interface{}{}
	if categoryID != nil {
		query += ` WHERE s.category_id = ?`
		args = append(args, *categoryID)
	}
	query += ` ORDER BY s.domain`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var s models.Site
		var catID sql.NullInt64
		var name, desc sql.NullString
		if err := rows.Scan(&s.ID, &catID, &s.CategoryName, &s.Domain, &name, &desc, &s.CreatedAt, &s.PageCount); err != nil {
			return nil, err
		}
		if catID.Valid {
			s.CategoryID = &catID.Int64
		}
		s.Name = name.String
		s.Description = desc.String
		sites = append(sites, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tags for each site
	for i := range sites {
		tags, err := r.GetSiteTags(sites[i].ID)
		if err != nil {
			return nil, err
		}
		sites[i].Tags = tags
	}

	return sites, nil
}

func (r *Repository) GetSite(id int64) (*models.Site, error) {
	var s models.Site
	var catID sql.NullInt64
	var name, desc sql.NullString
	err := r.db.QueryRow(`
		SELECT s.id, s.category_id, COALESCE(c.name, '') as category_name,
		       s.domain, s.name, s.description, s.created_at,
		       (SELECT COUNT(*) FROM pages WHERE site_id = s.id) as page_count
		FROM sites s
		LEFT JOIN categories c ON s.category_id = c.id
		WHERE s.id = ?
	`, id).Scan(&s.ID, &catID, &s.CategoryName, &s.Domain, &name, &desc, &s.CreatedAt, &s.PageCount)
	if err != nil {
		return nil, err
	}
	if catID.Valid {
		s.CategoryID = &catID.Int64
	}
	s.Name = name.String
	s.Description = desc.String

	tags, err := r.GetSiteTags(s.ID)
	if err != nil {
		return nil, err
	}
	s.Tags = tags

	return &s, nil
}

func (r *Repository) GetSiteByDomain(domain string) (*models.Site, error) {
	var s models.Site
	var catID sql.NullInt64
	var name, desc sql.NullString
	err := r.db.QueryRow(`
		SELECT s.id, s.category_id, COALESCE(c.name, '') as category_name,
		       s.domain, s.name, s.description, s.created_at,
		       (SELECT COUNT(*) FROM pages WHERE site_id = s.id) as page_count
		FROM sites s
		LEFT JOIN categories c ON s.category_id = c.id
		WHERE s.domain = ?
	`, domain).Scan(&s.ID, &catID, &s.CategoryName, &s.Domain, &name, &desc, &s.CreatedAt, &s.PageCount)
	if err != nil {
		return nil, err
	}
	if catID.Valid {
		s.CategoryID = &catID.Int64
	}
	s.Name = name.String
	s.Description = desc.String
	return &s, nil
}

func (r *Repository) CreateSite(categoryID *int64, domain, name, description string) (int64, error) {
	var catID interface{} = nil
	if categoryID != nil {
		catID = *categoryID
	}
	result, err := r.db.Exec(`INSERT INTO sites (category_id, domain, name, description) VALUES (?, ?, ?, ?)`,
		catID, domain, nullString(name), nullString(description))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) UpdateSite(id int64, categoryID *int64, domain, name, description string) error {
	var catID interface{} = nil
	if categoryID != nil {
		catID = *categoryID
	}
	_, err := r.db.Exec(`UPDATE sites SET category_id = ?, domain = ?, name = ?, description = ? WHERE id = ?`,
		catID, domain, nullString(name), nullString(description), id)
	return err
}

func (r *Repository) DeleteSite(id int64) error {
	_, err := r.db.Exec(`DELETE FROM sites WHERE id = ?`, id)
	return err
}

// Pages

func (r *Repository) GetPages(siteID *int64, categoryID *int64, tagID *int64) ([]models.Page, error) {
	query := `
		SELECT DISTINCT p.id, p.site_id, s.domain, p.path, p.title, p.description, p.created_at
		FROM pages p
		JOIN sites s ON p.site_id = s.id
		LEFT JOIN categories c ON s.category_id = c.id
	`
	args := []interface{}{}
	conditions := []string{}

	if siteID != nil {
		conditions = append(conditions, "p.site_id = ?")
		args = append(args, *siteID)
	}
	if categoryID != nil {
		conditions = append(conditions, "s.category_id = ?")
		args = append(args, *categoryID)
	}
	if tagID != nil {
		query += ` LEFT JOIN page_tags pt ON p.id = pt.page_id LEFT JOIN site_tags st ON s.id = st.site_id`
		conditions = append(conditions, "(pt.tag_id = ? OR st.tag_id = ?)")
		args = append(args, *tagID, *tagID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY p.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.Page
	for rows.Next() {
		var p models.Page
		var title, desc sql.NullString
		if err := rows.Scan(&p.ID, &p.SiteID, &p.SiteDomain, &p.Path, &title, &desc, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Title = title.String
		p.Description = desc.String
		pages = append(pages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tags for each page
	for i := range pages {
		tags, err := r.GetPageTags(pages[i].ID)
		if err != nil {
			return nil, err
		}
		pages[i].Tags = tags

		siteTags, err := r.GetSiteTags(pages[i].SiteID)
		if err != nil {
			return nil, err
		}
		pages[i].SiteTags = siteTags
	}

	return pages, nil
}

func (r *Repository) GetPage(id int64) (*models.Page, error) {
	var p models.Page
	var title, desc sql.NullString
	err := r.db.QueryRow(`
		SELECT p.id, p.site_id, s.domain, p.path, p.title, p.description, p.created_at
		FROM pages p
		JOIN sites s ON p.site_id = s.id
		WHERE p.id = ?
	`, id).Scan(&p.ID, &p.SiteID, &p.SiteDomain, &p.Path, &title, &desc, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.Title = title.String
	p.Description = desc.String

	tags, err := r.GetPageTags(p.ID)
	if err != nil {
		return nil, err
	}
	p.Tags = tags

	siteTags, err := r.GetSiteTags(p.SiteID)
	if err != nil {
		return nil, err
	}
	p.SiteTags = siteTags

	return &p, nil
}

func (r *Repository) CreatePage(siteID int64, path, title, description string) (int64, error) {
	result, err := r.db.Exec(`INSERT INTO pages (site_id, path, title, description) VALUES (?, ?, ?, ?)`,
		siteID, path, nullString(title), nullString(description))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) UpdatePage(id int64, siteID int64, path, title, description string) error {
	_, err := r.db.Exec(`UPDATE pages SET site_id = ?, path = ?, title = ?, description = ? WHERE id = ?`,
		siteID, path, nullString(title), nullString(description), id)
	return err
}

func (r *Repository) DeletePage(id int64) error {
	_, err := r.db.Exec(`DELETE FROM pages WHERE id = ?`, id)
	return err
}

// Tags

func (r *Repository) GetTags() ([]models.Tag, error) {
	rows, err := r.db.Query(`
		SELECT t.id, t.name,
		       (SELECT COUNT(*) FROM site_tags WHERE tag_id = t.id) as site_count,
		       (SELECT COUNT(*) FROM page_tags WHERE tag_id = t.id) as page_count
		FROM tags t
		ORDER BY t.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.SiteCount, &t.PageCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repository) GetTag(id int64) (*models.Tag, error) {
	var t models.Tag
	err := r.db.QueryRow(`
		SELECT t.id, t.name,
		       (SELECT COUNT(*) FROM site_tags WHERE tag_id = t.id) as site_count,
		       (SELECT COUNT(*) FROM page_tags WHERE tag_id = t.id) as page_count
		FROM tags t WHERE t.id = ?
	`, id).Scan(&t.ID, &t.Name, &t.SiteCount, &t.PageCount)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetOrCreateTag(name string) (int64, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	var id int64
	err := r.db.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	result, err := r.db.Exec(`INSERT INTO tags (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) CreateTag(name string) (int64, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	result, err := r.db.Exec(`INSERT INTO tags (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) DeleteTag(id int64) error {
	_, err := r.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	return err
}

func (r *Repository) GetSiteTags(siteID int64) ([]models.Tag, error) {
	rows, err := r.db.Query(`
		SELECT t.id, t.name FROM tags t
		JOIN site_tags st ON t.id = st.tag_id
		WHERE st.site_id = ?
		ORDER BY t.name
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repository) GetPageTags(pageID int64) ([]models.Tag, error) {
	rows, err := r.db.Query(`
		SELECT t.id, t.name FROM tags t
		JOIN page_tags pt ON t.id = pt.tag_id
		WHERE pt.page_id = ?
		ORDER BY t.name
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *Repository) SetSiteTags(siteID int64, tagIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM site_tags WHERE site_id = ?`, siteID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if _, err := tx.Exec(`INSERT INTO site_tags (site_id, tag_id) VALUES (?, ?)`, siteID, tagID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) SetPageTags(pageID int64, tagIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM page_tags WHERE page_id = ?`, pageID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if _, err := tx.Exec(`INSERT INTO page_tags (page_id, tag_id) VALUES (?, ?)`, pageID, tagID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) AddSiteTag(siteID, tagID int64) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO site_tags (site_id, tag_id) VALUES (?, ?)`, siteID, tagID)
	return err
}

func (r *Repository) RemoveSiteTag(siteID, tagID int64) error {
	_, err := r.db.Exec(`DELETE FROM site_tags WHERE site_id = ? AND tag_id = ?`, siteID, tagID)
	return err
}

func (r *Repository) AddPageTag(pageID, tagID int64) error {
	_, err := r.db.Exec(`INSERT OR IGNORE INTO page_tags (page_id, tag_id) VALUES (?, ?)`, pageID, tagID)
	return err
}

func (r *Repository) RemovePageTag(pageID, tagID int64) error {
	_, err := r.db.Exec(`DELETE FROM page_tags WHERE page_id = ? AND tag_id = ?`, pageID, tagID)
	return err
}

// Dashboard

func (r *Repository) GetDashboardStats() (*models.DashboardStats, error) {
	var stats models.DashboardStats

	r.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&stats.CategoryCount)
	r.db.QueryRow(`SELECT COUNT(*) FROM sites`).Scan(&stats.SiteCount)
	r.db.QueryRow(`SELECT COUNT(*) FROM pages`).Scan(&stats.PageCount)

	pages, err := r.GetPages(nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(pages) > 10 {
		pages = pages[:10]
	}
	stats.RecentPages = pages

	return &stats, nil
}

// Search

func (r *Repository) Search(query string) ([]models.Site, []models.Page, error) {
	query = "%" + query + "%"

	siteRows, err := r.db.Query(`
		SELECT s.id, s.category_id, COALESCE(c.name, '') as category_name,
		       s.domain, s.name, s.description, s.created_at,
		       (SELECT COUNT(*) FROM pages WHERE site_id = s.id) as page_count
		FROM sites s
		LEFT JOIN categories c ON s.category_id = c.id
		WHERE s.domain LIKE ? OR s.name LIKE ? OR s.description LIKE ?
		ORDER BY s.domain
		LIMIT 20
	`, query, query, query)
	if err != nil {
		return nil, nil, err
	}
	defer siteRows.Close()

	var sites []models.Site
	for siteRows.Next() {
		var s models.Site
		var catID sql.NullInt64
		var name, desc sql.NullString
		if err := siteRows.Scan(&s.ID, &catID, &s.CategoryName, &s.Domain, &name, &desc, &s.CreatedAt, &s.PageCount); err != nil {
			return nil, nil, err
		}
		if catID.Valid {
			s.CategoryID = &catID.Int64
		}
		s.Name = name.String
		s.Description = desc.String
		sites = append(sites, s)
	}

	pageRows, err := r.db.Query(`
		SELECT p.id, p.site_id, s.domain, p.path, p.title, p.description, p.created_at
		FROM pages p
		JOIN sites s ON p.site_id = s.id
		WHERE p.path LIKE ? OR p.title LIKE ? OR p.description LIKE ?
		ORDER BY p.created_at DESC
		LIMIT 20
	`, query, query, query)
	if err != nil {
		return nil, nil, err
	}
	defer pageRows.Close()

	var pages []models.Page
	for pageRows.Next() {
		var p models.Page
		var title, desc sql.NullString
		if err := pageRows.Scan(&p.ID, &p.SiteID, &p.SiteDomain, &p.Path, &title, &desc, &p.CreatedAt); err != nil {
			return nil, nil, err
		}
		p.Title = title.String
		p.Description = desc.String
		pages = append(pages, p)
	}

	return sites, pages, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
