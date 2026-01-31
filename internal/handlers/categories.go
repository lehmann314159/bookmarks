package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/lehmann314159/bookmarks/internal/repository"
)

type CategoryHandler struct {
	repo *repository.Repository
	tmpl *template.Template
}

func NewCategoryHandler(repo *repository.Repository, tmpl *template.Template) *CategoryHandler {
	return &CategoryHandler{repo: repo, tmpl: tmpl}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.GetCategories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Categories": categories,
	}

	if isHTMX(r) {
		h.tmpl.ExecuteTemplate(w, "category-list", data)
	} else {
		h.tmpl.ExecuteTemplate(w, "categories.html", data)
	}
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	id, err := h.repo.CreateCategory(name, description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		category, _ := h.repo.GetCategory(id)
		h.tmpl.ExecuteTemplate(w, "category-row", category)
	} else {
		http.Redirect(w, r, "/categories", http.StatusSeeOther)
	}
}

func (h *CategoryHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	category, err := h.repo.GetCategory(id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	h.tmpl.ExecuteTemplate(w, "category-edit-form", category)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateCategory(id, name, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		category, _ := h.repo.GetCategory(id)
		h.tmpl.ExecuteTemplate(w, "category-row", category)
	} else {
		http.Redirect(w, r, "/categories", http.StatusSeeOther)
	}
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteCategory(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/categories", http.StatusSeeOther)
	}
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
