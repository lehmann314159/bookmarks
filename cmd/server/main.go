package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lehmann314159/bookmarks/internal/database"
	"github.com/lehmann314159/bookmarks/internal/handlers"
	"github.com/lehmann314159/bookmarks/internal/repository"
)

func main() {
	// Get data directory from environment or use default
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Initialize database
	db, err := database.New(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	repo := repository.New(db)

	// Parse templates
	tmpl, err := parseTemplates()
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Initialize handlers
	homeHandler := handlers.NewHomeHandler(repo, tmpl)
	categoryHandler := handlers.NewCategoryHandler(repo, tmpl)
	siteHandler := handlers.NewSiteHandler(repo, tmpl)
	pageHandler := handlers.NewPageHandler(repo, tmpl)
	tagHandler := handlers.NewTagHandler(repo, tmpl)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Home
	mux.HandleFunc("GET /{$}", homeHandler.Dashboard)
	mux.HandleFunc("GET /search", homeHandler.Search)

	// Categories
	mux.HandleFunc("GET /categories", categoryHandler.List)
	mux.HandleFunc("POST /categories", categoryHandler.Create)
	mux.HandleFunc("GET /categories/{id}/edit", categoryHandler.Edit)
	mux.HandleFunc("PUT /categories/{id}", categoryHandler.Update)
	mux.HandleFunc("DELETE /categories/{id}", categoryHandler.Delete)

	// Sites
	mux.HandleFunc("GET /sites", siteHandler.List)
	mux.HandleFunc("POST /sites", siteHandler.Create)
	mux.HandleFunc("GET /sites/{id}/edit", siteHandler.Edit)
	mux.HandleFunc("PUT /sites/{id}", siteHandler.Update)
	mux.HandleFunc("DELETE /sites/{id}", siteHandler.Delete)
	mux.HandleFunc("GET /sites/{id}/pages", siteHandler.Pages)

	// Pages
	mux.HandleFunc("GET /pages", pageHandler.List)
	mux.HandleFunc("POST /pages", pageHandler.Create)
	mux.HandleFunc("GET /pages/{id}/edit", pageHandler.Edit)
	mux.HandleFunc("PUT /pages/{id}", pageHandler.Update)
	mux.HandleFunc("DELETE /pages/{id}", pageHandler.Delete)
	mux.HandleFunc("POST /pages/quick-add", pageHandler.QuickAdd)

	// Tags
	mux.HandleFunc("GET /tags", tagHandler.List)
	mux.HandleFunc("POST /tags", tagHandler.Create)
	mux.HandleFunc("DELETE /tags/{id}", tagHandler.Delete)
	mux.HandleFunc("GET /tags/{id}/items", tagHandler.Items)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func parseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"join": func(tags interface{}, sep string) string {
			switch t := tags.(type) {
			case []string:
				return strings.Join(t, sep)
			default:
				return ""
			}
		},
		"tagNames": func(tags interface{}) string {
			// Helper to extract tag names as comma-separated string
			switch t := tags.(type) {
			case []interface{}:
				var names []string
				for _, tag := range t {
					if m, ok := tag.(map[string]interface{}); ok {
						if name, ok := m["Name"].(string); ok {
							names = append(names, name)
						}
					}
				}
				return strings.Join(names, ", ")
			default:
				return ""
			}
		},
	}

	return template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
}
