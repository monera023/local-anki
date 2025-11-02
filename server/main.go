package main

import (
	"database/sql"
	"highlights-anki/internal/database"
	"highlights-anki/internal/handlers"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	_ "modernc.org/sqlite"
)

type Highlight struct {
	ID         int
	Source     string
	SourceType string
	Content    string
}

type Source struct {
	Name       string
	Type       string
	Identifier string
}

var db *sql.DB
var tmpl *template.Template

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs each HTTP request with method, path, status, and duration
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200
		}

		// Log the incoming request
		log.Printf("→ %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Call the actual handler
		next(wrapped, r)

		// Log the completion with duration and status
		duration := time.Since(start)
		log.Printf("← %s %s [%d] completed in %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	}
}

func main() {
	log.SetFlags(log.Lshortfile)
	db, db_err := database.InitDb("./highlights.db")
	if db_err != nil {
		log.Fatal("Failed to initialize database:", db_err)
	}

	search_db, search_err := database.InitSearch("./highlights.db")
	if search_err != nil {
		log.Fatal("Failed to initialize search database:", search_err)
	}

	defer db.Close()
	defer search_db.Close()

	h := handlers.NewHandlers(db, search_db)

	http.HandleFunc("/admin/upload", loggingMiddleware(h.AddHighlights))
	http.HandleFunc("/random", loggingMiddleware(h.GetRandomHighlights))
	http.HandleFunc("/sources", loggingMiddleware(h.SourcesHandler))
	http.HandleFunc("/source/", loggingMiddleware(h.SourceHighlightsHandler))
	http.HandleFunc("/search", loggingMiddleware(h.SearchHandler))
	http.HandleFunc("/searchResults", loggingMiddleware(h.SearchResultsHandler))

	// if err := initDb(); err != nil {
	// 	log.Fatal(err)
	// }

	// defer db.Close()

	// Load templates from files
	var err error
	tmpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Failed to parse templates:", err)
	}

	http.HandleFunc("/", loggingMiddleware(homeHandler))
	// http.HandleFunc("/source/", loggingMiddleware(sourceHighlightsHandler))
	http.HandleFunc("/admin", loggingMiddleware(adminHandler))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

func sourceHighlightsHandler(w http.ResponseWriter, r *http.Request) {
	sourceName := strings.TrimPrefix(r.URL.Path, "/source/")
	if sourceName == "" {
		http.Error(w, "Source name required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT id, source, source_type, content 
		FROM highlights 
		WHERE source = ?
		ORDER BY id
	`, sourceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var highlights []Highlight
	for rows.Next() {
		var h Highlight
		err := rows.Scan(&h.ID, &h.Source, &h.SourceType, &h.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		highlights = append(highlights, h)
	}

	tmpl.ExecuteTemplate(w, "highlights.html", highlights)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "admin.html", nil)
}
