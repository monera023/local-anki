package database

import (
	"database/sql"
	"highlights-anki/internal/models"
)

type Db struct {
	*sql.DB
}

func InitDb(dbUri string) (*Db, error) {
	println("Initializing database at:", dbUri)
	db, err := sql.Open("sqlite3", "./highlights.db")

	if err != nil {
		println("Error opening database:", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		println("Error pinging database:", err)
		return nil, err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS highlights (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT,
		source_type TEXT,
		content TEXT
	);`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		println("Error creating table:", err)
		return nil, err
	}

	return &Db{db}, nil
}

func (db *Db) InsertHighlights(highlights []models.Highlight) (count int, err error) {
	tx, err := db.Begin()

	if err != nil {
		println("Error beginning transaction:", err)
		return 0, err
	}

	stmt, err := tx.Prepare("INSERT INTO highlights (source, source_type, content) VALUES (?, ?, ?)")
	if err != nil {
		println("Error preparing statement:", err)
		return 0, err
	}

	defer stmt.Close()
	count = 0

	for _, highlight := range highlights {
		_, err := stmt.Exec(highlight.Source, highlight.SourceType, highlight.Content)
		if err != nil {
			println("Error inserting highlight:", err)
			tx.Rollback()
			return count, err
		}
		count++
	}

	err = tx.Commit()
	if err != nil {
		println("Error committing transaction:", err)
		return 0, err
	}

	return count, nil
}

func (db *Db) GetRandomHighlights(limit int) ([]models.Highlight, error) {
	rows, err := db.Query("SELECT source, source_type, content FROM highlights ORDER BY RANDOM() LIMIT ?", limit)
	if err != nil {
		println("Error querying highlights:", err)
		return nil, err
	}
	defer rows.Close()
	var randomHighlights []models.Highlight

	for rows.Next() {
		var highlight models.Highlight
		err := rows.Scan(&highlight.Source, &highlight.SourceType, &highlight.Content)
		if err != nil {
			println("Error scanning highlight:", err)
			return nil, err
		}

		randomHighlights = append(randomHighlights, highlight)
	}
	return randomHighlights, nil
}
