package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/lehmann314159/bookmarks/internal/repository"
)

type SiteHandler struct {
	repo *repository.Repository
	tmpl *template.Template
}

func NewSiteHandler(repo *repository.Repository, tmpl *template.Template) *SiteHandler {
	return &SiteHandler{repo: repo, tmpl: tmpl}
}

func (h *SiteHandler) List(w http.ResponseWriter, r *http.Request) {
	var categoryID *int64
	if catStr := r.URL.Query().Get("category"); catStr != "" {
		id, err := strconv.ParseInt(catStr, 10, 64)
		if err == nil {
			categoryID = &id
		}
	}

	sites, err := h.repo.GetSites(categoryID)
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
		"Sites":      sites,
		"Categories": categories,
		"Tags":       tags,
		"CategoryID": categoryID,
	}

	if isHTMX(r) {
		h.tmpl.ExecuteTemplate(w, "site-list", data)
	} else {
		h.tmpl.ExecuteTemplate(w, "sites.html", data)
	}
}

func (h *SiteHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(r.FormValue("domain"))
	name := r.FormValue("name")
	description := r.FormValue("description")

	var categoryID *int64
	if catStr := r.FormValue("category_id"); catStr != "" {
		id, err := strconv.ParseInt(catStr, 10, 64)
		if err == nil {
			categoryID = &id
		}
	}

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	id, err := h.repo.CreateSite(categoryID, domain, name, description)
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
			h.repo.AddSiteTag(id, tagID)
		}
	}

	if isHTMX(r) {
		site, _ := h.repo.GetSite(id)
		h.tmpl.ExecuteTemplate(w, "site-row", site)
	} else {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
	}
}

func (h *SiteHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	site, err := h.repo.GetSite(id)
	if err != nil {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	categories, err := h.repo.GetCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Site":       site,
		"Categories": categories,
	}

	h.tmpl.ExecuteTemplate(w, "site-edit-form", data)
}

func (h *SiteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(r.FormValue("domain"))
	name := r.FormValue("name")
	description := r.FormValue("description")

	var categoryID *int64
	if catStr := r.FormValue("category_id"); catStr != "" {
		cid, err := strconv.ParseInt(catStr, 10, 64)
		if err == nil {
			categoryID = &cid
		}
	}

	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateSite(id, categoryID, domain, name, description); err != nil {
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
	h.repo.SetSiteTags(id, tagIDs)

	if isHTMX(r) {
		site, _ := h.repo.GetSite(id)
		h.tmpl.ExecuteTemplate(w, "site-row", site)
	} else {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
	}
}

func (h *SiteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteSite(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
	}
}

func (h *SiteHandler) Pages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	pages, err := h.repo.GetPages(&id, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	site, err := h.repo.GetSite(id)
	if err != nil {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Site":  site,
		"Pages": pages,
	}

	h.tmpl.ExecuteTemplate(w, "site-pages", data)
}
