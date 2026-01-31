package handlers

import (
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lehmann314159/bookmarks/internal/repository"
)

type PageHandler struct {
	repo *repository.Repository
	tmpl *template.Template
}

func NewPageHandler(repo *repository.Repository, tmpl *template.Template) *PageHandler {
	return &PageHandler{repo: repo, tmpl: tmpl}
}

func (h *PageHandler) List(w http.ResponseWriter, r *http.Request) {
	var siteID, categoryID, tagID *int64

	if str := r.URL.Query().Get("site"); str != "" {
		id, err := strconv.ParseInt(str, 10, 64)
		if err == nil {
			siteID = &id
		}
	}
	if str := r.URL.Query().Get("category"); str != "" {
		id, err := strconv.ParseInt(str, 10, 64)
		if err == nil {
			categoryID = &id
		}
	}
	if str := r.URL.Query().Get("tag"); str != "" {
		id, err := strconv.ParseInt(str, 10, 64)
		if err == nil {
			tagID = &id
		}
	}

	pages, err := h.repo.GetPages(siteID, categoryID, tagID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sites, err := h.repo.GetSites(nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories, err := h.repo.GetCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tags, err := h.repo.GetTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Pages":      pages,
		"Sites":      sites,
		"Categories": categories,
		"Tags":       tags,
		"SiteID":     siteID,
		"CategoryID": categoryID,
		"TagID":      tagID,
	}

	if isHTMX(r) {
		h.tmpl.ExecuteTemplate(w, "page-list", data)
	} else {
		h.tmpl.ExecuteTemplate(w, "pages.html", data)
	}
}

func (h *PageHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawURL := strings.TrimSpace(r.FormValue("url"))
	title := r.FormValue("title")
	description := r.FormValue("description")

	if rawURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Parse the URL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	domain := parsedURL.Host
	path := parsedURL.Path
	if parsedURL.RawQuery != "" {
		path += "?" + parsedURL.RawQuery
	}
	if path == "" {
		path = "/"
	}

	// Check if this is a root domain (no specific page)
	isRootDomain := path == "/" || path == ""

	// Fetch title from page if not provided
	if title == "" {
		title = fetchPageTitle(rawURL)
	}

	// Find or create site
	site, err := h.repo.GetSiteByDomain(domain)
	if err != nil {
		// Create new site - use title as site name if this is root domain
		siteName := ""
		if isRootDomain {
			siteName = title
		}
		siteID, err := h.repo.CreateSite(nil, domain, siteName, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		site, _ = h.repo.GetSite(siteID)
	}

	// If root domain, just create/update site, don't create a page
	if isRootDomain {
		// Handle tags - apply to site instead of page
		if tagStr := r.FormValue("tags"); tagStr != "" {
			tagNames := strings.Split(tagStr, ",")
			for _, tagName := range tagNames {
				tagName = strings.TrimSpace(tagName)
				if tagName == "" {
					continue
				}
				tagID, err := h.repo.GetOrCreateTag(tagName)
				if err != nil {
					continue
				}
				h.repo.AddSiteTag(site.ID, tagID)
			}
		}

		if isHTMX(r) {
			// Return empty - the page will refresh the sites list
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, "/sites", http.StatusSeeOther)
		}
		return
	}

	// Create page
	id, err := h.repo.CreatePage(site.ID, path, title, description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle tags
	if tagStr := r.FormValue("tags"); tagStr != "" {
		tagNames := strings.Split(tagStr, ",")
		for _, tagName := range tagNames {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagID, err := h.repo.GetOrCreateTag(tagName)
			if err != nil {
				continue
			}
			h.repo.AddPageTag(id, tagID)
		}
	}

	if isHTMX(r) {
		page, _ := h.repo.GetPage(id)
		h.tmpl.ExecuteTemplate(w, "page-row", page)
	} else {
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
	}
}

func (h *PageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	page, err := h.repo.GetPage(id)
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	sites, err := h.repo.GetSites(nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Page":  page,
		"Sites": sites,
	}

	h.tmpl.ExecuteTemplate(w, "page-edit-form", data)
}

func (h *PageHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	siteID, err := strconv.ParseInt(r.FormValue("site_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	path := strings.TrimSpace(r.FormValue("path"))
	title := r.FormValue("title")
	description := r.FormValue("description")

	if path == "" {
		path = "/"
	}

	if err := h.repo.UpdatePage(id, siteID, path, title, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle tags
	tagStr := r.FormValue("tags")
	var tagIDs []int64
	if tagStr != "" {
		tagNames := strings.Split(tagStr, ",")
		for _, tagName := range tagNames {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagID, err := h.repo.GetOrCreateTag(tagName)
			if err != nil {
				continue
			}
			tagIDs = append(tagIDs, tagID)
		}
	}
	h.repo.SetPageTags(id, tagIDs)

	if isHTMX(r) {
		page, _ := h.repo.GetPage(id)
		h.tmpl.ExecuteTemplate(w, "page-row", page)
	} else {
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
	}
}

func (h *PageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeletePage(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
	}
}

// QuickAdd handles adding pages from the dashboard
func (h *PageHandler) QuickAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawURL := strings.TrimSpace(r.FormValue("url"))
	title := r.FormValue("title")

	if rawURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Parse the URL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	domain := parsedURL.Host
	path := parsedURL.Path
	if parsedURL.RawQuery != "" {
		path += "?" + parsedURL.RawQuery
	}
	if path == "" {
		path = "/"
	}

	// Check if this is a root domain (no specific page)
	isRootDomain := path == "/" || path == ""

	// Fetch title from page if not provided
	if title == "" {
		title = fetchPageTitle(rawURL)
	}

	// Find or create site
	site, err := h.repo.GetSiteByDomain(domain)
	if err != nil {
		// Create new site - use title as site name if this is root domain
		siteName := ""
		if isRootDomain {
			siteName = title
		}
		siteID, err := h.repo.CreateSite(nil, domain, siteName, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		site, _ = h.repo.GetSite(siteID)
	}

	// If root domain, just create site, don't create a page
	if isRootDomain {
		if isHTMX(r) {
			// Return a row showing the site was added
			h.tmpl.ExecuteTemplate(w, "recent-site-row", site)
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
		return
	}

	// Create page
	id, err := h.repo.CreatePage(site.ID, path, title, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		page, _ := h.repo.GetPage(id)
		h.tmpl.ExecuteTemplate(w, "recent-page-row", page)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// fetchPageTitle fetches a URL and extracts the <title> tag content
func fetchPageTitle(rawURL string) string {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Read first 64KB to find title
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return ""
	}

	// Extract title using regex
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := titleRegex.FindSubmatch(body)
	if len(matches) >= 2 {
		title := strings.TrimSpace(string(matches[1]))
		// Decode HTML entities
		title = strings.ReplaceAll(title, "&amp;", "&")
		title = strings.ReplaceAll(title, "&lt;", "<")
		title = strings.ReplaceAll(title, "&gt;", ">")
		title = strings.ReplaceAll(title, "&quot;", "\"")
		title = strings.ReplaceAll(title, "&#39;", "'")
		title = strings.ReplaceAll(title, "&ndash;", "-")
		title = strings.ReplaceAll(title, "&mdash;", "-")
		return title
	}

	return ""
}
