package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/lehmann314159/bookmarks/internal/repository"
)

type TagHandler struct {
	repo *repository.Repository
	tmpl *template.Template
}

func NewTagHandler(repo *repository.Repository, tmpl *template.Template) *TagHandler {
	return &TagHandler{repo: repo, tmpl: tmpl}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.repo.GetTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Tags": tags,
	}

	if isHTMX(r) {
		h.tmpl.ExecuteTemplate(w, "tag-list", data)
	} else {
		h.tmpl.ExecuteTemplate(w, "tags.html", data)
	}
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	id, err := h.repo.CreateTag(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		tag, _ := h.repo.GetTag(id)
		h.tmpl.ExecuteTemplate(w, "tag-pill", tag)
	} else {
		http.Redirect(w, r, "/tags", http.StatusSeeOther)
	}
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteTag(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/tags", http.StatusSeeOther)
	}
}

func (h *TagHandler) Items(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	tag, err := h.repo.GetTag(id)
	if err != nil {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return
	}

	sites, err := h.repo.GetSites(nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter sites with this tag
	var taggedSites []interface{}
	for _, site := range sites {
		for _, t := range site.Tags {
			if t.ID == id {
				taggedSites = append(taggedSites, site)
				break
			}
		}
	}

	pages, err := h.repo.GetPages(nil, nil, &id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Tag":   tag,
		"Sites": taggedSites,
		"Pages": pages,
	}

	h.tmpl.ExecuteTemplate(w, "tag-items", data)
}
