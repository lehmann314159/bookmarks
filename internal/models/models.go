package models

import "time"

type Category struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	SiteCount   int // computed field
}

type Site struct {
	ID           int64
	CategoryID   *int64
	CategoryName string // computed field
	Domain       string
	Name         string
	Description  string
	CreatedAt    time.Time
	PageCount    int    // computed field
	Tags         []Tag  // computed field
}

type Page struct {
	ID          int64
	SiteID      int64
	SiteDomain  string // computed field
	Path        string
	Title       string
	Description string
	CreatedAt   time.Time
	Tags        []Tag  // computed field - page's own tags
	SiteTags    []Tag  // computed field - inherited from site
}

type Tag struct {
	ID        int64
	Name      string
	SiteCount int // computed field
	PageCount int // computed field
}

type DashboardStats struct {
	CategoryCount int
	SiteCount     int
	PageCount     int
	RecentPages   []Page
}
