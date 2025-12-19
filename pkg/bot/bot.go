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
}

func NewBot(client *ollama.Client, configDir string, sendFunc func(string), sendMasterFunc func(string)) *Bot {
	return &Bot{
		Client:         client,
		PromptManager:  prompt.NewPromptManager(configDir),
		ConfigDir:      configDir,
		SendFunc:       sendFunc,
		SendMasterFunc: sendMasterFunc,
	}
}

type Action struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type BotResponse struct {
	Actions []Action `json:"actions"`
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
		Context:  strings.Join(context, "\n"),
		Message:  msg,
	}
	modePrompt, err := b.PromptManager.LoadModePrompt(mode, modeData)
	if err != nil {
		return nil, fmt.Errorf("failed to load mode prompt: %w", err)
	}

	// 4. Combine prompts and send to Ollama
	// Using system prompt as User message or System message?
	// Ollama usually supports system role. But here we can just concat or use roles.
	// The user prompt files seem to imply they are instructions.
	// Let's us System role for System Prompt if possible, but the `client.go` interactions are simple list.
	// We will send:
	// System: sysPrompt
	// User: modePrompt (which contains the current message and context embedded)

	// Wait, the mode prompt `command/00_main.txt` says: "Now you are in command mode... Current message: {{.Message}}... The response should be ONLY json..."
	// So `modePrompt` contains the instruction AND the message input.
	// So we can send it as User message.

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
	// The response is expected to be JSON.
	// Sometimes LLMs wrap JSON in ```json ... ``` blocks. We should handle that.
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
