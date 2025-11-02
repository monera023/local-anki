package main

import (
	"highlights-anki/internal"
	"highlights-anki/internal/database"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Lshortfile)
	log.Println("In operations main.go")

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

	op := internal.NewOperations(db, search_db)

	tablesToFlush := []string{}

	if len(os.Args) > 1 {
		if os.Args[1] == "flush" {
			for _, arg := range os.Args[2:] {
				tablesToFlush = append(tablesToFlush, arg)
			}
			err := op.FlushTables(tablesToFlush)
			if err != nil {
				log.Fatal("Failed to flush tables:", err)
			}
			log.Println("Flushed tables:", tablesToFlush)
			return
		}

		if os.Args[1] == "index" {
			folder := os.Args[2]
			err := op.IndexFolder(folder)
			if err != nil {
				log.Fatal("Failed to index folder:", err)
			}
			log.Println("Indexed folder:", folder)
			return
		}
	}

	// folderToIndex := "podcast"
	// err := op.IndexFolder(folderToIndex)
	// if err != nil {
	// 	log.Fatal("Failed to index folder:", err)
	// }
}
