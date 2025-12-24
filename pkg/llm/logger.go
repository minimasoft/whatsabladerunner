package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLLM handles logging of LLM prompts and responses.
// Filename convention: ISO8601 date with ms + llm engine [+ tag] + prompt|response.
func LogLLM(engine string, tag string, messages []Message, response *Message) {
	now := time.Now().Format("2006-01-02T15:04:05.000")
	logDir := "logs/llm"

	// Ensure directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("[Logger] Failed to create log directory: %v\n", err)
		return
	}

	tagSuffix := ""
	if tag != "" {
		tagSuffix = "-" + tag
	}

	// Log Prompt
	promptFilename := fmt.Sprintf("%s-%s%s-prompt.txt", now, engine, tagSuffix)
	promptPath := filepath.Join(logDir, promptFilename)

	var promptContent string
	for _, m := range messages {
		promptContent += fmt.Sprintf("[%s]: %s\n\n", m.Role, m.Content)
		promptContent += "------------------------------------------------\n\n"
	}

	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		fmt.Printf("[Logger] Failed to write prompt log: %v\n", err)
	}

	// Log Response
	if response != nil {
		responseFilename := fmt.Sprintf("%s-%s%s-response.txt", now, engine, tagSuffix)
		responsePath := filepath.Join(logDir, responseFilename)
		if err := os.WriteFile(responsePath, []byte(response.Content), 0644); err != nil {
			fmt.Printf("[Logger] Failed to write response log: %v\n", err)
		}
	}
}
