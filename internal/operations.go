package internal

import (
	"bufio"
	"highlights-anki/internal/database"
	"highlights-anki/internal/models"
	"log"
	"os"
	"strings"
)

type Operations struct {
	DB     *database.Db
	Search *database.Search
}

func NewOperations(db *database.Db, search *database.Search) *Operations {
	return &Operations{DB: db, Search: search}
}

func (op *Operations) FlushTables(tables []string) error {
	for _, table := range tables {
		err := op.DB.FlushTable(table)
		if err != nil {
			return err
		}
	}
	return nil
}

func (op *Operations) IndexFolder(folder string) error {
	log.Println("Indexing folder:", folder)

	folderPath := "backups/" + folder
	files, err := os.ReadDir(folderPath)
	if err != nil {
		log.Fatal("Failed to read directory:", err)
		return err
	}

	for _, file_name := range files {
		if !file_name.IsDir() {
			sourceName := ParseSourceNameFromFileName(file_name.Name())
			log.Println("Processing file:", file_name.Name())
			file, err := os.Open(folderPath + "/" + file_name.Name())
			if err != nil {
				log.Println("Failed to open file:", err)
				continue
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			var highlights []models.Highlight
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) == "" {
					continue
				}
				highlight := models.Highlight{
					Source:     sourceName,
					SourceType: folder,
					Content:    line,
				}
				highlights = append(highlights, highlight)
			}

			// Insert highlights into the database
			count, err := op.DB.InsertHighlights(highlights)
			if err != nil {
				log.Println("Failed to insert highlights into database:", err)
				continue
			}
			log.Printf("Inserted %d highlights from file %s into database.\n", count, file_name.Name())

			// Insert highlights into the FTS table
			err = op.Search.InsertToFTS(highlights, sourceName)
			if err != nil {
				log.Println("Failed to insert highlights into search index:", err)
				continue
			}
			log.Printf("Inserted %d highlights from file %s into FTS table.\n", len(highlights), file_name.Name())

		}
	}
	return nil
}
