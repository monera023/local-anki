package database

import (
	"database/sql"
	"highlights-anki/internal/models"
	"log"

	_ "modernc.org/sqlite"
)

type Db struct {
	*sql.DB
}

func InitDb(dbUri string) (*Db, error) {
	println("Initializing database at:", dbUri)
	db, err := sql.Open("sqlite", dbUri)

	if err != nil {
		log.Fatalln("Error opening database:", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Fatalln("Error pinging database:", err)
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
		log.Println("[db.go] Error querying highlights:", err)
		return nil, err
	}
	defer rows.Close()
	var randomHighlights []models.Highlight

	for rows.Next() {
		var highlight models.Highlight
		err := rows.Scan(&highlight.Source, &highlight.SourceType, &highlight.Content)
		if err != nil {
			log.Println("[db.go] Error scanning highlight:", err)
			return nil, err
		}

		randomHighlights = append(randomHighlights, highlight)
	}
	return randomHighlights, nil
}

func (db *Db) GetSources() ([]models.Source, error) {
	rows, err := db.Query("SELECT DISTINCT source, source_type FROM highlights")
	if err != nil {
		log.Println("[db.go] Error querying sources:", err)
		return nil, err
	}

	defer rows.Close()

	var sources []models.Source

	for rows.Next() {
		var source models.Source
		err := rows.Scan(&source.Name, &source.Type)
		if err != nil {
			log.Println("[db.go] Error scanning source:", err)
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil

}

func (db *Db) GetSourceHighlights(source string) ([]models.Highlight, error) {
	rows, err := db.Query("SELECT source, source_type, content FROM highlights WHERE source = ?", source)
	if err != nil {
		log.Fatalf("Error querying highlights for source %s: %v", source, err)
		return nil, err
	}

	defer rows.Close()

	var highlights []models.Highlight

	for rows.Next() {
		var highlight models.Highlight
		err := rows.Scan(&highlight.Source, &highlight.SourceType, &highlight.Content)
		if err != nil {
			log.Fatal("Error scanning source highlights:", err)
			return nil, err
		}
		highlights = append(highlights, highlight)
	}
	return highlights, nil

}

func (db *Db) FlushTable(table_name string) error {

	_, err := db.Exec("DELETE FROM " + table_name)
	if err != nil {
		log.Fatal("Error flushing highlights table:", err)
		return err
	}

	log.Println("Flushed table:", table_name)

	return nil
}
