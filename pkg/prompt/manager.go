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
	Memories         string
	Tasks            string // JSON string of active tasks
	Contacts         string // JSON string
	Context          string
	Message          string
	CurrentTask      string // JSON of current task for task mode
	AvailableActions string // JSON schema of available actions
	Behaviors        string // List of available behaviors
	ActiveBehaviors  string // JSON string of all active behaviors
}

type BehaviorData struct {
	ModeData
	EnabledBehaviors string // Content of enabled behaviors
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
		Date:     time.Now().Format("Monday, 2006-01-02 15:04:05"),
	}
	return pm.renderTemplates(dir, data)
}

type WatcherData struct {
	ProposedMessage string
	Context         string
}

func (pm *PromptManager) LoadModePrompt(mode string, data ModeData) (string, error) {
	var sb strings.Builder

	// 1. Load context.txt
	contextPath := filepath.Join(pm.ConfigDir, "modes", "context.txt")
	contextContent, err := pm.renderFile(contextPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load context.txt: %w", err)
	}
	sb.WriteString(contextContent)
	sb.WriteString("\n")

	// 2. Load Mode Directory Files
	dir := filepath.Join(pm.ConfigDir, "modes", mode)
	modeContent, err := pm.renderTemplates(dir, data)
	if err != nil {
		return "", fmt.Errorf("failed to load mode directory %s: %w", mode, err)
	}
	sb.WriteString(modeContent)

	// 3. Load protocol.txt
	protocolPath := filepath.Join(pm.ConfigDir, "modes", "protocol.txt")
	protocolContent, err := pm.renderFile(protocolPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load protocol.txt: %w", err)
	}
	sb.WriteString(protocolContent)
	sb.WriteString("\n")

	return sb.String(), nil
}

func (pm *PromptManager) LoadWatcherPrompt(data WatcherData) (string, error) {
	var sb strings.Builder

	// 1. Load context.txt
	contextPath := filepath.Join(pm.ConfigDir, "watcher", "context.txt")
	contextContent, err := pm.renderFile(contextPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load watcher context.txt: %w", err)
	}
	sb.WriteString(contextContent)
	sb.WriteString("\n")

	// 2. Load Rules Directory Files
	dir := filepath.Join(pm.ConfigDir, "watcher", "rules")
	rulesContent, err := pm.renderTemplates(dir, data)
	if err != nil {
		return "", fmt.Errorf("failed to load watcher rules directory: %w", err)
	}
	sb.WriteString(rulesContent)

	// 3. Load protocol.txt
	protocolPath := filepath.Join(pm.ConfigDir, "watcher", "protocol.txt")
	protocolContent, err := pm.renderFile(protocolPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load watcher protocol.txt: %w", err)
	}
	sb.WriteString(protocolContent)
	sb.WriteString("\n")

	return sb.String(), nil
}

func (pm *PromptManager) LoadBehaviorPrompt(data BehaviorData) (string, error) {
	var sb strings.Builder

	// 1. Load __base.txt templated with enabled behaviors
	basePath := filepath.Join(pm.ConfigDir, "modes", "behavior", "__base.txt")
	baseContent, err := pm.renderFile(basePath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load behavior __base.txt: %w", err)
	}
	sb.WriteString(baseContent)
	sb.WriteString("\n")

	// 2. Load context.txt
	contextPath := filepath.Join(pm.ConfigDir, "modes", "context.txt")
	contextContent, err := pm.renderFile(contextPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load behavior context.txt: %w", err)
	}
	sb.WriteString(contextContent)
	sb.WriteString("\n")

	// 3. Load protocol.txt
	protocolPath := filepath.Join(pm.ConfigDir, "modes", "protocol.txt")
	protocolContent, err := pm.renderFile(protocolPath, data)
	if err != nil {
		return "", fmt.Errorf("failed to load behavior protocol.txt: %w", err)
	}
	sb.WriteString(protocolContent)
	sb.WriteString("\n")

	return sb.String(), nil
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
			content, err := pm.renderFile(filepath.Join(dir, entry.Name()), data)
			if err != nil {
				return "", err
			}
			sb.WriteString(content)
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

func (pm *PromptManager) renderFile(path string, data interface{}) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Create a new template and parse the content
	tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", path, err)
	}

	return buf.String(), nil
}
