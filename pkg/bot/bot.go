package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"whatsabladerunner/pkg/llm"
	"whatsabladerunner/pkg/prompt"
	"whatsabladerunner/pkg/tasks"
)

type Bot struct {
	Client        llm.Client
	PromptManager *prompt.PromptManager
	TaskManager   *tasks.TaskManager
	ConfigDir     string

	SendFunc          func(string)
	SendMasterFunc    func(string)
	Contacts          string            // JSON formatted string of contacts
	StartTaskCallback func(*tasks.Task) // Called when a task is confirmed to start it
}

func NewBot(client llm.Client, configDir string, sendFunc func(string), sendMasterFunc func(string), contacts string) *Bot {
	return &Bot{
		Client:         client,
		PromptManager:  prompt.NewPromptManager(configDir),
		TaskManager:    tasks.NewTaskManager(filepath.Join(configDir, "tasks")),
		ConfigDir:      configDir,
		SendFunc:       sendFunc,
		SendMasterFunc: sendMasterFunc,
		Contacts:       contacts,
	}
}

// RawAction is used for initial parsing to handle flexible content types
type RawAction struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type RawBotResponse struct {
	Actions []RawAction `json:"actions"`
}

// Action represents a processed action with string content
type Action struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// BotResponse is the processed response returned to callers
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

	msgs := []llm.Message{
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

	// 3. Load Tasks
	activeTasks, err := b.TaskManager.LoadActiveTasks()
	if err != nil {
		fmt.Printf("Warning: failed to load tasks: %v\n", err)
		activeTasks = []tasks.Task{}
	}
	tasksJSON, err := json.Marshal(activeTasks)
	if err != nil {
		fmt.Printf("Warning: failed to marshal tasks: %v\n", err)
		tasksJSON = []byte("[]")
	}

	// 4. Load Mode Prompt
	modeData := prompt.ModeData{
		Memories: string(memoriesContent),
		Tasks:    string(tasksJSON),
		Contacts: b.Contacts,
		Context:  strings.Join(context, "\n"),
		Message:  msg,
	}
	modePrompt, err := b.PromptManager.LoadModePrompt(mode, modeData)
	if err != nil {
		return nil, fmt.Errorf("failed to load mode prompt: %w", err)
	}

	msgs := []llm.Message{
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

	var rawResp RawBotResponse
	if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
		fmt.Printf("Raw response: %s\n", respMsg.Content)
		return nil, fmt.Errorf("failed to parse bot response json: %w", err)
	}

	// Build the returned BotResponse with processed actions
	botResp := &BotResponse{}

	// 6. Execute Actions
	for _, rawAction := range rawResp.Actions {
		// Parse content as string for most actions
		var contentStr string
		if rawAction.Content != nil {
			// Try to unmarshal as string first
			if err := json.Unmarshal(rawAction.Content, &contentStr); err != nil {
				// Not a string, keep raw for special handling
				contentStr = string(rawAction.Content)
			}
		}

		switch rawAction.Type {
		case "memory_update":
			if err := os.WriteFile(memoriesPath, []byte(contentStr), 0644); err != nil {
				fmt.Printf("Failed to update memories: %v\n", err)
			} else {
				fmt.Println("Memories updated.")
			}

		case "response":
			// In command mode, no watcher check - just send directly
			if b.SendFunc != nil {
				finalMsg := "[Blady] : " + contentStr
				b.SendFunc(finalMsg)
			}

		case "message_master":
			if b.SendMasterFunc != nil {
				b.SendMasterFunc(contentStr)
			}

		case "create_task":
			var taskContent tasks.CreateTaskContent
			if err := json.Unmarshal(rawAction.Content, &taskContent); err != nil {
				fmt.Printf("Failed to parse create_task content: %v\n", err)
				if b.SendFunc != nil {
					b.SendFunc("[Blady] : Error: No pude procesar la creación de tarea.")
				}
				continue
			}

			// Validate contact is in the contacts list
			if !b.isValidContact(taskContent.Contact) {
				fmt.Printf("Failed to create task: contact %s not in contacts list\n", taskContent.Contact)
				if b.SendFunc != nil {
					b.SendFunc(fmt.Sprintf("[Blady] : Error: El contacto '%s' no está en la lista de contactos. No se puede crear la tarea.", taskContent.Contact))
				}
				continue
			}

			task, err := b.TaskManager.CreateTask(taskContent.Objective, taskContent.Contact, taskContent.OriginalOrders)
			if err != nil {
				fmt.Printf("Failed to create task: %v\n", err)
				if b.SendFunc != nil {
					b.SendFunc(fmt.Sprintf("[Blady] : Error al crear tarea: %v", err))
				}
			} else {
				fmt.Printf("Task %d created successfully\n", task.ID)
				// Print task JSON to self-chat
				if b.SendFunc != nil {
					taskJSON, _ := json.MarshalIndent(task, "", "  ")
					b.SendFunc(fmt.Sprintf("[Blady] : Tarea creada:\n```json\n%s\n```", string(taskJSON)))
				}
			}

		case "delete_task":
			id, err := parseTaskID(contentStr)
			if err != nil {
				fmt.Printf("Failed to parse delete_task ID: %v\n", err)
				continue
			}
			if err := b.TaskManager.DeleteTask(id); err != nil {
				fmt.Printf("Failed to delete task %d: %v\n", id, err)
			}

		case "confirm_task":
			id, err := parseTaskID(contentStr)
			if err != nil {
				fmt.Printf("Failed to parse confirm_task ID: %v\n", err)
				continue
			}
			task, err := b.TaskManager.ConfirmTaskAndGet(id)
			if err != nil {
				fmt.Printf("Failed to confirm task %d: %v\n", id, err)
			} else if b.StartTaskCallback != nil {
				// Trigger task start callback
				b.StartTaskCallback(task)
			}

		case "pause_task":
			id, err := parseTaskID(contentStr)
			if err != nil {
				fmt.Printf("Failed to parse pause_task ID: %v\n", err)
				continue
			}
			if err := b.TaskManager.PauseTask(id); err != nil {
				fmt.Printf("Failed to pause task %d: %v\n", id, err)
			}

		case "resume_task":
			id, err := parseTaskID(contentStr)
			if err != nil {
				fmt.Printf("Failed to parse resume_task ID: %v\n", err)
				continue
			}
			if err := b.TaskManager.ResumeTask(id); err != nil {
				fmt.Printf("Failed to resume task %d: %v\n", id, err)
			}

		default:
			fmt.Printf("Unknown action type: %s\n", rawAction.Type)
		}
	}

	return botResp, nil
}

// ProcessTask processes a message in task mode for a specific task
// It sets CurrentTask in the mode data and transitions task to running on first response
func (b *Bot) ProcessTask(task *tasks.Task, msg string, context []string, sendToContact func(string)) (*BotResponse, error) {
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

	// 3. Load Tasks
	activeTasks, err := b.TaskManager.LoadActiveTasks()
	if err != nil {
		fmt.Printf("Warning: failed to load tasks: %v\n", err)
		activeTasks = []tasks.Task{}
	}
	tasksJSON, err := json.Marshal(activeTasks)
	if err != nil {
		fmt.Printf("Warning: failed to marshal tasks: %v\n", err)
		tasksJSON = []byte("[]")
	}

	// 4. Marshal current task for template
	currentTaskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		fmt.Printf("Warning: failed to marshal current task: %v\n", err)
		currentTaskJSON = []byte("{}")
	}

	// 5. Load Mode Prompt (task mode)
	modeData := prompt.ModeData{
		Memories:    string(memoriesContent),
		Tasks:       string(tasksJSON),
		Contacts:    b.Contacts,
		Context:     strings.Join(context, "\n"),
		Message:     msg,
		CurrentTask: string(currentTaskJSON),
	}
	modePrompt, err := b.PromptManager.LoadModePrompt("task", modeData)
	if err != nil {
		return nil, fmt.Errorf("failed to load task mode prompt: %w", err)
	}

	msgs := []llm.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: modePrompt},
	}

	fmt.Printf("DEBUG: Sending to Ollama (Task Mode):\n--- System Prompt ---\n%s\n--- Mode Prompt ---\n%s\n---------------------\n", sysPrompt, modePrompt)

	respMsg, err := b.Client.Chat(msgs, nil)
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	// 6. Parse Response
	fmt.Printf("DEBUG: Raw Ollama Response (Task Mode):\n%s\n---------------------\n", respMsg.Content)

	content := respMsg.Content
	content = cleanJSON(content)

	var rawResp RawBotResponse
	if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
		fmt.Printf("Raw response: %s\n", respMsg.Content)
		return nil, fmt.Errorf("failed to parse bot response json: %w", err)
	}

	botResp := &BotResponse{}

	// 7. Execute Actions
	for _, rawAction := range rawResp.Actions {
		var contentStr string
		if rawAction.Content != nil {
			if err := json.Unmarshal(rawAction.Content, &contentStr); err != nil {
				contentStr = string(rawAction.Content)
			}
		}

		switch rawAction.Type {
		case "memory_update":
			if err := os.WriteFile(memoriesPath, []byte(contentStr), 0644); err != nil {
				fmt.Printf("Failed to update memories: %v\n", err)
			} else {
				fmt.Println("Memories updated.")
			}

		case "response":
			// Watcher Check
			proceed, reason, err := b.CheckMessage(contentStr, context)
			if err != nil {
				fmt.Printf("Watcher check error: %v\n", err)
				if b.SendMasterFunc != nil {
					b.SendMasterFunc(fmt.Sprintf("[System] Watcher error: %v", err))
				}
				continue
			}

			if !proceed {
				fmt.Printf("Watcher BLOCKED message: %s. Reason: %s\n", contentStr, reason)
				if b.SendMasterFunc != nil {
					b.SendMasterFunc(fmt.Sprintf("Watcher stopped message %s. Reason: %s", contentStr, reason))
				}
				continue
			}

			// Send to contact (no [Blady] prefix for task conversations)
			if sendToContact != nil {
				sendToContact(contentStr)
			}

			// Transition task to running if pending
			if task.Status == tasks.StatusPending {
				if err := b.TaskManager.SetTaskRunning(task.ID); err != nil {
					fmt.Printf("Failed to set task running: %v\n", err)
				}
			}

		case "message_master":
			if b.SendMasterFunc != nil {
				b.SendMasterFunc(contentStr)
			}

		case "pause_task":
			if err := b.TaskManager.PauseTask(task.ID); err != nil {
				fmt.Printf("Failed to pause task %d: %v\n", task.ID, err)
			}

		default:
			fmt.Printf("Unknown action type in task mode: %s\n", rawAction.Type)
		}
	}

	return botResp, nil
}

// parseTaskID parses a task ID from a string that might be quoted or a number
func parseTaskID(s string) (int, error) {
	// Remove quotes if present
	s = strings.Trim(s, `"`)
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
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

// isValidContact checks if a contact number exists in the bot's contacts list
func (b *Bot) isValidContact(contact string) bool {
	if b.Contacts == "" || b.Contacts == "[]" {
		return false
	}
	// Simple string check - the contact should appear as a "number" field value
	return strings.Contains(b.Contacts, fmt.Sprintf(`"number":"%s"`, contact))
}
