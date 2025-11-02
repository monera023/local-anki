package handlers

import (
	"fmt"
	"highlights-anki/internal"
	"highlights-anki/internal/database"
	"highlights-anki/internal/models"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

type Handlers struct {
	DB     *database.Db
	tmpl   *template.Template
	Search *database.Search
}

func NewHandlers(db *database.Db, search *database.Search) *Handlers {
	tmpl, err := template.ParseGlob("templates/*.html")

	// Debug: List all parsed templates
	// for _, t := range tmpl.Templates() {
	// 	fmt.Println("Found template:", t.Name())
	// }
	if err != nil {
		panic(err)
	}
	return &Handlers{DB: db, tmpl: tmpl, Search: search}
}

func (h *Handlers) GetRandomHighlights(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching random highlights...")
	randomHighlights, err := h.DB.GetRandomHighlights(10)
	if err != nil {
		http.Error(w, "Failed to fetch random highlights", http.StatusInternalServerError)
		return
	}
	log.Println("Random highlights fetched:", len(randomHighlights))

	err = h.tmpl.ExecuteTemplate(w, "highlights.html", randomHighlights)
	if err != nil {
		log.Fatal("Error executing template:", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) SourcesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Fetching sources...")
	sources, err := h.DB.GetSources()

	if err != nil {
		log.Fatal("Error fetching sources: %v", err)
		http.Error(w, "Failed to fetch sources", http.StatusInternalServerError)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "sources.html", sources)

	if err != nil {
		log.Fatal("Error executing template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) SourceHighlightsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[handler.go] SourceHighlightsHandler called")
	sourceName := strings.TrimPrefix(r.URL.Path, "/source/")
	if sourceName == "" {
		http.Error(w, "Source Name not passed", http.StatusBadRequest)
		return
	}

	highlights, err := h.DB.GetSourceHighlights(sourceName)
	if err != nil {
		log.Fatal("Error fetching source highlights: %v", err)
		http.Error(w, "Failed to fetch source highlights", http.StatusInternalServerError)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "highlights.html", highlights)

	if err != nil {
		log.Fatal("Error executing template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}

}

func (h *Handlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("SearchHandler called")

	err := h.tmpl.ExecuteTemplate(w, "search.html", nil)
	if err != nil {
		log.Fatal("Error executing template:", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) SearchResultsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("SearchResultsHandler called")
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	results, err := h.Search.GetSearchResults(query, 5)
	if err != nil {
		log.Println("Error fetching search results:", err)
		http.Error(w, "Failed to fetch search results", http.StatusInternalServerError)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "search-results.html", results)
	if err != nil {
		log.Println("Error executing template:", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) AddHighlights(w http.ResponseWriter, r *http.Request) {
	println("AddHighlights called")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form .. Size limit exceeded", http.StatusBadRequest)
		return
	}

	sourceName := r.FormValue("source_name")
	sourceType := r.FormValue("source_type")
	highlightsText := r.FormValue("highlights_text")

	if sourceName == "" || sourceType == "" {
		http.Error(w, "Source name and type are required", http.StatusBadRequest)
		return
	}

	var lines []string

	fmt.Println("Len of highlights text area:", len(strings.TrimSpace(highlightsText)))

	if strings.TrimSpace(highlightsText) != "" {
		fmt.Println("Processing highlights from text area")
		lines = strings.Split(highlightsText, "\n")
	} else {
		fmt.Println("Processing highlights from uploaded file")
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

		lines = strings.Split(string(content), "\n")
	}

	var highlights []models.Highlight

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		trimmed_line := strings.TrimSpace(line)
		highlight := models.Highlight{
			Source:     sourceName,
			SourceType: sourceType,
			Content:    trimmed_line,
		}
		highlights = append(highlights, highlight)
	}

	// Insert highlights into the database
	count, err := h.DB.InsertHighlights(highlights)
	if err != nil {
		http.Error(w, "Failed to insert highlights into database", http.StatusInternalServerError)
		return
	}

	// Insert highlights into the FTS table
	log.Println("Inserting highlights into FTS table...")
	err = h.Search.InsertToFTS(highlights, sourceName)
	if err != nil {
		http.Error(w, "Failed to insert highlights into search index", http.StatusInternalServerError)
		return
	}
	log.Println("Highlights successfully inserted into FTS table.")

	// Write highlights to a file for backup
	backupFilePath := fmt.Sprintf("backups/%s/%s_highlights.txt", sourceType, sourceName)
	err = internal.WriteHighlightsToFile(highlights, backupFilePath)

	if err != nil {
		log.Println("Error writing highlights to backup file:", err)
		http.Error(w, "Failed to write highlights to backup file", http.StatusInternalServerError)
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
