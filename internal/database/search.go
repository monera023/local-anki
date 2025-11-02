package database

import (
	"database/sql"
	"highlights-anki/internal/models"
	"log"

	_ "modernc.org/sqlite"
)

type Search struct {
	*sql.DB
}

func InitSearch(dbUri string) (*Search, error) {
	log.Println("Initializing search database at:", dbUri)
	db, err := sql.Open("sqlite", dbUri)

	if err != nil {
		log.Println("Error opening search database:", err)
		return nil, err
	}

	createSearchTableQuery := `
	CREATE VIRTUAL TABLE IF NOT EXISTS highlights_fts USING fts5(
    title, content, tokenize='porter unicode61' );`

	_, err = db.Exec(createSearchTableQuery)
	if err != nil {
		log.Println("Error creating FTS table:", err)
		return nil, err
	}
	return &Search{db}, nil
}

func (search *Search) InsertToFTS(highlights []models.Highlight, title string) error {

	log.Println("Inserting highlights into FTS table with title: and size: ", title, len(highlights))
	tx, err := search.Begin()

	if err != nil {
		log.Println("Error beginning FTS transaction:", err)
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO highlights_fts (title, content) VALUES (?, ?)")
	if err != nil {
		log.Println("Error preparing FTS statement:", err)
		return err
	}

	defer stmt.Close()

	for _, highlight := range highlights {
		_, err = stmt.Exec(title, highlight.Content)
		if err != nil {
			log.Println("Error inserting highlight into FTS:", err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Error committing FTS transaction:", err)
		return err
	}

	log.Println("Successfully inserted highlights into FTS table")
	return nil
}

func (search *Search) GetSearchResults(query string, limit int) ([]models.Highlight, error) {
	log.Println("Searching FTS table with query:", query)
	rows, err := search.Query("SELECT title, content FROM highlights_fts WHERE content MATCH ? LIMIT ?", query, limit)
	if err != nil {
		log.Println("Error querying FTS table:", err)
		return nil, err
	}

	defer rows.Close()
	var results []models.Highlight

	for rows.Next() {
		var title, content string
		err := rows.Scan(&title, &content)
		if err != nil {
			log.Println("Error scanning FTS result row:", err)
			return nil, err
		}
		highlight := models.Highlight{
			Source:  title,
			Content: content,
		}
		results = append(results, highlight)
	}
	return results, nil
}
