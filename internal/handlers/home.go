package handlers

import (
	"html/template"
	"net/http"

	"github.com/lehmann314159/bookmarks/internal/repository"
)

type HomeHandler struct {
	repo *repository.Repository
	tmpl *template.Template
}

func NewHomeHandler(repo *repository.Repository, tmpl *template.Template) *HomeHandler {
	return &HomeHandler{repo: repo, tmpl: tmpl}
}

func (h *HomeHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetDashboardStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Stats": stats,
	}

	h.tmpl.ExecuteTemplate(w, "index.html", data)
}

func (h *HomeHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query required", http.StatusBadRequest)
		return
	}

	sites, pages, err := h.repo.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Query": query,
		"Sites": sites,
		"Pages": pages,
	}

	h.tmpl.ExecuteTemplate(w, "search-results", data)
}
