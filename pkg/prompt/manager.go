package prompt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type SystemData struct {
	Language string
	Date     string
}

type ModeData struct {
	Memories string
	Tasks    string // JSON string
	Context  string
	Message  string
}

type PromptManager struct {
	ConfigDir string
}

func NewPromptManager(configDir string) *PromptManager {
	return &PromptManager{
		ConfigDir: configDir,
	}
}

func (pm *PromptManager) LoadSystemPrompt(lang string) (string, error) {
	dir := filepath.Join(pm.ConfigDir, "system")
	data := SystemData{
		Language: lang,
		Date:     time.Now().Format("2006-01-02 15:04:05"),
	}
	return pm.renderTemplates(dir, data)
}

func (pm *PromptManager) LoadModePrompt(mode string, data ModeData) (string, error) {
	dir := filepath.Join(pm.ConfigDir, "modes", mode)
	return pm.renderTemplates(dir, data)
}

func (pm *PromptManager) renderTemplates(dir string, data interface{}) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read dir %s: %w", dir, err)
	}

	var sb strings.Builder

	// Ensure predictable order is usually good, ReadDir returns sorted by filename.
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return "", fmt.Errorf("failed to read file %s: %w", entry.Name(), err)
			}

			// Create a new template and parse the content
			tmpl, err := template.New(entry.Name()).Parse(string(content))
			if err != nil {
				return "", fmt.Errorf("failed to parse template %s: %w", entry.Name(), err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("failed to execute template %s: %w", entry.Name(), err)
			}

			sb.WriteString(buf.String())
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
