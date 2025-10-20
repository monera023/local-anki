package handlers

import (
	"fmt"
	"highlights-anki/internal/database"
	"highlights-anki/internal/models"
	"io"
	"net/http"
	"strings"
)

type Handlers struct {
	DB *database.Db
}

func NewHandlers(db *database.Db) *Handlers {
	return &Handlers{DB: db}
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

	lines := strings.Split(string(content), "\n")

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

	response := fmt.Sprintf(`
		<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded relative" role="alert">
			<strong class="font-bold">Success!</strong>
			<span class="block sm:inline">Uploaded %d highlights for "%s"</span>
		</div>
	`, count, sourceName)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(response))
}
