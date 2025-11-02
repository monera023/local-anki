package internal

import (
	"bufio"
	"encoding/base64"
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

func EncodeToBase64(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

func DecodeFromBase64(encoded string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		fmt.Println("[utils.go] Error decoding from base64:", err)
		return "", err
	}

	return string(decodedBytes), nil
}

func ParseSourceNameFromFileName(fileName string) string {
	// Example: "Title_book_highlights.txt" -> "Title"
	// Example : "Title_podcast_highlights.txt" -> "Title"
	underscoreIndex := -1
	for i := 0; i < len(fileName); i++ {
		if fileName[i] == '_' {
			underscoreIndex = i
			break
		}
	}

	if underscoreIndex == -1 {
		return fileName // No underscore found, return the whole filename
	}

	return fileName[:underscoreIndex]
}
