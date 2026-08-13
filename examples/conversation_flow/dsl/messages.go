package dsl

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Message represents a single user prompt loaded from a CSV dataset.
type Message struct {
	ID             string
	Text           string
	Category       string
	ExpectedTokens int
}

// LoadMessages reads user prompts from a CSV file with header: message_id,user_message,category,expected_tokens.
// Rows with empty message text are silently skipped. Unparseable token counts default to 15.
func LoadMessages(filePath string) ([]Message, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open messages file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse messages CSV: %w", err)
	}

	if len(records) <= 1 {
		return nil, nil
	}

	var messages []Message
	for _, row := range records[1:] {
		if len(row) < 2 {
			continue
		}

		text := strings.Trim(row[1], `"`)
		if text == "" {
			continue
		}

		tokens := 15
		if len(row) >= 4 && row[3] != "" {
			if parsed, parseErr := strconv.Atoi(strings.TrimSpace(row[3])); parseErr == nil {
				tokens = parsed
			}
		}

		category := ""
		if len(row) >= 3 {
			category = strings.TrimSpace(row[2])
		}

		messages = append(messages, Message{
			ID:             strings.TrimSpace(row[0]),
			Text:           text,
			Category:       category,
			ExpectedTokens: tokens,
		})
	}

	return messages, nil
}
