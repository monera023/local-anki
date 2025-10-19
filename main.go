package main

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Highlight struct {
	ID         int
	Source     string
	SourceType string
	Content    string
}

type Source struct {
	Name string
	Type string
}

var templatesFS embed.FS

var db *sql.DB
var tmpl *template.Template

func initDb() error {
	var err error
	db, err = sql.Open("sqlite3", "./highlights.db")
	if err != nil {
		return err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS highlights (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT,
		source_type TEXT,
		content TEXT
	);
	`
	_, err = db.Exec(createTableQuery)
	return err
}

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

	if err := initDb(); err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	// Load templates from files
	var err error
	tmpl, err = template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Failed to parse templates:", err)
	}

	http.HandleFunc("/", loggingMiddleware(homeHandler))
	http.HandleFunc("/random", loggingMiddleware(randomHighlightsHandler))
	http.HandleFunc("/sources", loggingMiddleware(sourcesHandler))
	http.HandleFunc("/source/", loggingMiddleware(sourceHighlightsHandler))
	http.HandleFunc("/admin", loggingMiddleware(adminHandler))
	http.HandleFunc("/admin/upload", loggingMiddleware(uploadHandler))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

func randomHighlightsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, source, source_type, content 
		FROM highlights 
		ORDER BY RANDOM() 
		LIMIT 10
	`)
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

func sourcesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT DISTINCT source, source_type 
		FROM highlights 
		ORDER BY source
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		err := rows.Scan(&s.Name, &s.Type)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sources = append(sources, s)
	}

	tmpl.ExecuteTemplate(w, "sources.html", sources)
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

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	sourceName := r.FormValue("source_name")
	sourceType := r.FormValue("source_type")

	if sourceName == "" || sourceType == "" {
		http.Error(w, "Source name and type are required", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("highlights_file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content", http.StatusInternalServerError)
		return
	}

	// Split content by newlines - each line is a separate highlight
	lines := strings.Split(string(content), "\n")

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare("INSERT INTO highlights (source, source_type, content) VALUES (?, ?, ?)")
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to prepare statement", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Each non-empty line is a highlight
		_, err = stmt.Exec(sourceName, sourceType, line)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to insert highlight", http.StatusInternalServerError)
			return
		}
		count++
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	response := fmt.Sprintf(`
		<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded relative" role="alert">
			<strong class="font-bold">Success!</strong>
			<span class="block sm:inline">Uploaded %d highlights for "%s"</span>
		</div>
	`, count, sourceName)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(response))
}
