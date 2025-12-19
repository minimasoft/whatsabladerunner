package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/pkg/prompt"
)

type Bot struct {
	Client        *ollama.Client
	PromptManager *prompt.PromptManager
	ConfigDir     string

	SendFunc       func(string)
	SendMasterFunc func(string)
	Contacts       string // JSON formatted string of contacts
}

func NewBot(client *ollama.Client, configDir string, sendFunc func(string), sendMasterFunc func(string), contacts string) *Bot {
	return &Bot{
		Client:         client,
		PromptManager:  prompt.NewPromptManager(configDir),
		ConfigDir:      configDir,
		SendFunc:       sendFunc,
		SendMasterFunc: sendMasterFunc,
		Contacts:       contacts,
	}
}

type Action struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type BotResponse struct {
	Actions []Action `json:"actions"`
}

type WatcherResponse struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func (b *Bot) CheckMessage(proposedMsg string, context []string) (bool, string, error) {
	watcherData := prompt.WatcherData{
		ProposedMessage: proposedMsg,
		Context:         strings.Join(context, "\n"),
	}

	watcherPrompt, err := b.PromptManager.LoadWatcherPrompt(watcherData)
	if err != nil {
		return false, "", fmt.Errorf("failed to load watcher prompt: %w", err)
	}

	sysPrompt, err := b.PromptManager.LoadSystemPrompt("Spanish")
	if err != nil {
		return false, "", fmt.Errorf("failed to load system prompt: %w", err)
	}

	msgs := []ollama.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: watcherPrompt},
	}

	fmt.Printf("DEBUG: Sending to Watcher:\n--- System Prompt ---\n%s\n--- Watcher Prompt ---\n%s\n---------------------\n", sysPrompt, watcherPrompt)

	respMsg, err := b.Client.Chat(msgs, nil)
	if err != nil {
		return false, "", fmt.Errorf("watcher ollama chat failed: %w", err)
	}

	fmt.Printf("DEBUG: Watcher Response:\n%s\n---------------------\n", respMsg.Content)

	content := cleanJSON(respMsg.Content)
	var watcherResp WatcherResponse
	if err := json.Unmarshal([]byte(content), &watcherResp); err != nil {
		return false, "", fmt.Errorf("failed to parse watcher response json: %w", err)
	}

	if watcherResp.Action == "block" {
		return false, watcherResp.Reason, nil
	}

	return true, "", nil
}

func (b *Bot) Process(mode string, msg string, context []string) (*BotResponse, error) {
	// 1. Load System Prompt
	sysPrompt, err := b.PromptManager.LoadSystemPrompt("Spanish")
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	// 2. Load Memories
	memoriesPath := filepath.Join(b.ConfigDir, "memories.txt")
	memoriesContent, err := os.ReadFile(memoriesPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read memories: %w", err)
	}

	// 3. Load Mode Prompt
	modeData := prompt.ModeData{
		Memories: string(memoriesContent),
		Tasks:    "[]", // Default empty tasks
		Contacts: b.Contacts,
		Context:  strings.Join(context, "\n"),
		Message:  msg,
	}
	modePrompt, err := b.PromptManager.LoadModePrompt(mode, modeData)
	if err != nil {
		return nil, fmt.Errorf("failed to load mode prompt: %w", err)
	}

	msgs := []ollama.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: modePrompt},
	}

	fmt.Printf("DEBUG: Sending to Ollama:\n--- System Prompt ---\n%s\n--- Mode Prompt ---\n%s\n---------------------\n", sysPrompt, modePrompt)

	// Note: Client uses default options (Temperature 0.13, etc)
	respMsg, err := b.Client.Chat(msgs, nil)
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	// 5. Parse Response
	fmt.Printf("DEBUG: Raw Ollama Response:\n%s\n---------------------\n", respMsg.Content)

	content := respMsg.Content
	content = cleanJSON(content)

	var botResp BotResponse
	if err := json.Unmarshal([]byte(content), &botResp); err != nil {
		fmt.Printf("Raw response: %s\n", respMsg.Content)
		return nil, fmt.Errorf("failed to parse bot response json: %w", err)
	}

	// 6. Execute Actions
	for _, action := range botResp.Actions {
		if action.Type == "memory_update" {
			if err := os.WriteFile(memoriesPath, []byte(action.Content), 0644); err != nil {
				fmt.Printf("Failed to update memories: %v\n", err)
			} else {
				fmt.Println("Memories updated.")
			}
		} else if action.Type == "response" {
			// Watcher Check
			proceed, reason, err := b.CheckMessage(action.Content, context)
			if err != nil {
				fmt.Printf("Watcher check error: %v\n", err)
				if b.SendMasterFunc != nil {
					b.SendMasterFunc(fmt.Sprintf("[System] Watcher error: %v", err))
				}
				continue
			}

			if !proceed {
				fmt.Printf("Watcher BLOCKED message: %s. Reason: %s\n", action.Content, reason)
				if b.SendMasterFunc != nil {
					b.SendMasterFunc(fmt.Sprintf("Watcher stopped message %s. Reason: %s", action.Content, reason))
				}
				continue
			}

			if b.SendFunc != nil {
				finalMsg := "[Blady] : " + action.Content
				b.SendFunc(finalMsg)
			}
		} else if action.Type == "message_master" {
			if b.SendMasterFunc != nil {
				b.SendMasterFunc(action.Content)
			}
		}
	}

	return &botResp, nil
}

func cleanJSON(content string) string {
	content = strings.TrimSpace(content)
	// Find the start of the JSON object
	start := strings.Index(content, "{")
	// Find the end of the JSON object
	end := strings.LastIndex(content, "}")

	if start != -1 && end != -1 && end > start {
		return content[start : end+1]
	}

	// Fallback to previous logic if braces not found (unlikely for valid JSON)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}
	return strings.TrimSpace(content)
}
