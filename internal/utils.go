package internal

import (
	"bufio"
	"fmt"
	"highlights-anki/internal/models"
	"os"
)

func WriteHighlightsToFile(highlights []models.Highlight, filePath string) error {
	// Always persist highlights to a file for backup
	fmt.Println("Writing highlights to file:", filePath)
	file, err := os.Create(filePath)

	if err != nil {
		return err
	}
	defer file.Close()

	// Use a buffered writer for efficiency
	writer := bufio.NewWriter(file)

	for _, h := range highlights {
		_, err := writer.WriteString(h.Content + "\n")
		if err != nil {
			fmt.Println("Error writing highlight to file:", err)
			return err
		}
	}

	writer.Flush()

	fmt.Println("Highlights successfully written to", filePath)
	return nil
}
