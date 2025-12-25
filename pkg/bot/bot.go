package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whatsabladerunner/pkg/behaviors"
	"whatsabladerunner/pkg/bot/actions"
	"whatsabladerunner/pkg/llm"
	"whatsabladerunner/pkg/prompt"
	"whatsabladerunner/pkg/tasks"
)

type Bot struct {
	Client          llm.Client
	PromptManager   *prompt.PromptManager
	TaskManager     *tasks.TaskManager
	BehaviorManager *behaviors.BehaviorManager
	ActionRegistry  *actions.Registry
	ConfigDir       string

	SendFunc               func(string)
	SendMasterFunc         func(string)
	SendButtonResponseFunc func(displayText, buttonID string)  // For sending button responses
	SendMediaFunc          func(chatJID string, mediaID int64) // For sending media back
	Contacts               string                              // JSON formatted string of contacts
	StartTaskCallback      func(*tasks.Task)                   // Called when a task is confirmed to start it
	ResumeTaskCallback     func(*tasks.Task)                   // Called when a task is resumed

	// OnWatcherBlock is called when watcher blocks a message, allowing the caller to store it for potential override
	OnWatcherBlock func(blockedMsg string, targetChatJID string, sendFunc func(string))

	// CurrentTaskID is set during ProcessTask to enable proper message tagging
	CurrentTaskID int
}

func NewBot(client llm.Client, configDir string, sendFunc func(string), sendMasterFunc func(string), contacts string) *Bot {
	b := &Bot{
		Client:          client,
		PromptManager:   prompt.NewPromptManager(configDir),
		TaskManager:     tasks.NewTaskManager(filepath.Join(configDir, "tasks")),
		BehaviorManager: behaviors.NewBehaviorManager(filepath.Join(configDir, "behaviors")),
		ActionRegistry:  actions.NewRegistry(),
		ConfigDir:       configDir,
		SendFunc:        sendFunc,
		SendMasterFunc:  sendMasterFunc,
		Contacts:        contacts,
	}

	b.registerActions()
	return b
}

func (b *Bot) registerActions() {
	// Behaviors
	b.ActionRegistry.Register(&actions.EnableBehaviorAction{})
	b.ActionRegistry.Register(&actions.DisableBehaviorAction{})

	// Memory Update & Append
	b.ActionRegistry.Register(&actions.MemoryUpdateAction{
		MemoriesPath: filepath.Join(b.ConfigDir, "memories.txt"),
	})
	b.ActionRegistry.Register(&actions.MemoryAppendAction{
		MemoriesPath: filepath.Join(b.ConfigDir, "memories.txt"),
	})

	// Response
	b.ActionRegistry.Register(&actions.ResponseAction{
		SendFunc:       b.SendFunc,
		CheckMessage:   b.CheckMessage,
		SendMasterFunc: b.SendMasterFunc,
		OnWatcherBlock: func(blockedMsg string, targetChatJID string, sendFunc func(string)) {
			if b.OnWatcherBlock != nil {
				b.OnWatcherBlock(blockedMsg, targetChatJID, sendFunc)
			}
		},
	})

	// Message Master
	b.ActionRegistry.Register(&actions.MessageMasterAction{
		SendMasterFunc: b.SendMasterFunc,
	})

	// Create Task
	b.ActionRegistry.Register(&actions.CreateTaskAction{
		TaskManager: b.TaskManager,
		GetContacts: func() string { return b.Contacts },
		SendFunc:    b.SendFunc,
	})

	// Task Management
	taskActions := []actions.TaskActionType{
		actions.TaskDelete,
		actions.TaskConfirm,
		actions.TaskPause,
		actions.TaskResume,
	}
	for _, t := range taskActions {
		b.ActionRegistry.Register(&actions.TaskManagementAction{
			Type:        t,
			TaskManager: b.TaskManager,
			StartTaskCallback: func(task *tasks.Task) {
				if b.StartTaskCallback != nil {
					b.StartTaskCallback(task)
				}
			},
			ResumeTaskCallback: func(task *tasks.Task) {
				if b.ResumeTaskCallback != nil {
					b.ResumeTaskCallback(task)
				}
			},
		})
	}

	// Send Media
	b.ActionRegistry.Register(&actions.SendMediaAction{
		SendMediaFunc: b.SendMediaFunc,
		ToMaster:      false,
	})
	b.ActionRegistry.Register(&actions.SendMediaAction{
		SendMediaFunc: b.SendMediaFunc,
		ToMaster:      true,
	})

	// Button Response
	b.ActionRegistry.Register(&actions.ButtonResponseAction{
		SendButtonResponseFunc: b.SendButtonResponseFunc,
		TaskManager:            b.TaskManager,
	})

	// Custom Actions
	actionsDir := filepath.Join(b.ConfigDir, "actions")
	if err := actions.LoadCustomActions(actionsDir, b.ActionRegistry); err != nil {
		fmt.Printf("Error loading custom actions: %v\n", err)
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

	///fmt.Printf("DEBUG: Sending to Watcher:\n--- System Prompt ---\n%s\n--- Watcher Prompt ---\n%s\n---------------------\n", sysPrompt, watcherPrompt)

	respMsg, err := b.Client.Chat(msgs, map[string]interface{}{"log_tag": "watcher"})
	if err != nil {
		return false, "", fmt.Errorf("watcher ollama chat failed: %w", err)
	}

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

func (b *Bot) getAvailableActionsJSON() string {
	schemas := b.ActionRegistry.GetSchemas()
	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
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
		Memories:         string(memoriesContent),
		Tasks:            string(tasksJSON),
		Contacts:         b.Contacts,
		Context:          strings.Join(context, "\n"),
		Message:          msg,
		AvailableActions: b.getAvailableActionsJSON(),
	}
	modePrompt, err := b.PromptManager.LoadModePrompt(mode, modeData)
	if err != nil {
		return nil, fmt.Errorf("failed to load mode prompt: %w", err)
	}

	msgs := []llm.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: modePrompt},
	}

	//fmt.Printf("DEBUG: Sending to Ollama:\n--- System Prompt ---\n%s\n--- Mode Prompt ---\n%s\n---------------------\n", sysPrompt, modePrompt)

	// Note: Client uses default options (Temperature 0.13, etc)
	respMsg, err := b.Client.Chat(msgs, map[string]interface{}{"log_tag": "mode-" + mode})
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	// 5. Parse Response

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
	// Loop for tool use recursion
	maxRecursion := 5
	recursionDepth := 0
	currentMsg := msg

	for {
		// Calculate recursion depth
		if recursionDepth > maxRecursion {
			fmt.Println("Warning: Max recursion depth reached in Process")
			break
		}

		// Update message in modeData if recursion happened
		if recursionDepth > 0 {
			// Update the Mode Prompt with the new combined message
			modeData.Message = currentMsg
			modePrompt, err = b.PromptManager.LoadModePrompt(mode, modeData)
			if err != nil {
				return nil, fmt.Errorf("failed to reload mode prompt: %w", err)
			}
			msgs[1].Content = modePrompt

			// Re-run Chat
			respMsg, err = b.Client.Chat(msgs, map[string]interface{}{"log_tag": "mode-" + mode})
			if err != nil {
				return nil, fmt.Errorf("ollama chat failed in recursion: %w", err)
			}
			content = cleanJSON(respMsg.Content)
			if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
				fmt.Printf("Raw response: %s\n", respMsg.Content)
				return nil, fmt.Errorf("failed to parse bot response json: %w", err)
			}
		}

		toolOutputs := []string{}

		for i, rawAction := range rawResp.Actions {
			fmt.Printf("[Bot] Processing action %d: type=%s\n", i+1, rawAction.Type)

			act, ok := b.ActionRegistry.Get(rawAction.Type)
			if !ok {
				fmt.Printf("Warning: received unknown action '%s'\n", rawAction.Type)
				continue
			}

			ctx := actions.ActionContext{
				Context:         context,
				BehaviorManager: b.BehaviorManager,
				ToolOutputs:     &toolOutputs,
			}

			if err := act.Execute(ctx, rawAction.Content); err != nil {
				fmt.Printf("Error executing action %s: %v\n", rawAction.Type, err)
			}

			// Parse content to string
			var contentStr string
			if rawAction.Content != nil {
				if err := json.Unmarshal(rawAction.Content, &contentStr); err != nil {
					contentStr = string(rawAction.Content)
				}
			}
			botResp.Actions = append(botResp.Actions, Action{Type: rawAction.Type, Content: contentStr})
		}

		// Check if we have tool outputs to feed back
		if len(toolOutputs) > 0 {
			recursionDepth++
			combinedOutputs := strings.Join(toolOutputs, "\n")
			currentMsg = fmt.Sprintf("%s\n\n[System: Tool Results]\n%s", currentMsg, combinedOutputs)
			fmt.Printf("[Bot] Tool outputs received, recursing (depth %d)...\n", recursionDepth)
			continue
		}

		// No tool outputs, we are done
		break
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

	// 3. Marshal current task for template (no other tasks needed in task mode)
	currentTaskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		fmt.Printf("Warning: failed to marshal current task: %v\n", err)
		currentTaskJSON = []byte("{}")
	}

	// 5. Load Mode Prompt (task mode)
	// Send empty tasks and contacts to focus on current task
	modeData := prompt.ModeData{
		Memories:         string(memoriesContent),
		Tasks:            "[]", // Empty to focus on current task
		Contacts:         "[]", // Empty to focus on conversation
		Context:          strings.Join(context, "\n"),
		Message:          msg,
		CurrentTask:      string(currentTaskJSON),
		AvailableActions: b.getAvailableActionsJSON(),
	}
	modePrompt, err := b.PromptManager.LoadModePrompt("task", modeData)
	if err != nil {
		return nil, fmt.Errorf("failed to load task mode prompt: %w", err)
	}

	msgs := []llm.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: modePrompt},
	}

	//fmt.Printf("DEBUG: Sending to Ollama (Task Mode):\n--- System Prompt ---\n%s\n--- Mode Prompt ---\n%s\n---------------------\n", sysPrompt, modePrompt)

	respMsg, err := b.Client.Chat(msgs, map[string]interface{}{"log_tag": "task"})
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	// 6. Parse Response

	content := respMsg.Content
	content = cleanJSON(content)

	var rawResp RawBotResponse
	if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
		fmt.Printf("Raw response: %s\n", respMsg.Content)
		return nil, fmt.Errorf("failed to parse bot response json: %w", err)
	}

	botResp := &BotResponse{}

	// 7. Execute Actions
	// Loop for tool use recursion
	maxRecursion := 5
	recursionDepth := 0
	currentMsg := msg

	for {
		// Calculate recursion depth
		if recursionDepth > maxRecursion {
			fmt.Println("Warning: Max recursion depth reached in ProcessTask")
			break
		}

		// Update message in modeData if recursion happened
		if recursionDepth > 0 {
			modeData.Message = currentMsg
			modePrompt, err = b.PromptManager.LoadModePrompt("task", modeData)
			if err != nil {
				return nil, fmt.Errorf("failed to reload task mode prompt: %w", err)
			}
			msgs[1].Content = modePrompt

			respMsg, err = b.Client.Chat(msgs, map[string]interface{}{"log_tag": "task"})
			if err != nil {
				return nil, fmt.Errorf("ollama chat failed in recursion: %w", err)
			}
			content = cleanJSON(respMsg.Content)
			if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
				fmt.Printf("Raw response: %s\n", respMsg.Content)
				return nil, fmt.Errorf("failed to parse bot response json: %w", err)
			}
		}

		toolOutputs := []string{}

		for i, rawAction := range rawResp.Actions {
			fmt.Printf("[Bot/Task] Processing action %d: type=%s\n", i+1, rawAction.Type)

			act, ok := b.ActionRegistry.Get(rawAction.Type)
			if !ok {
				fmt.Printf("Warning: received unknown action '%s' in task mode\n", rawAction.Type)
				continue
			}

			ctx := actions.ActionContext{
				Context:         context,
				Task:            task,
				BehaviorManager: b.BehaviorManager,
				SendToContact:   sendToContact,
				ToolOutputs:     &toolOutputs,
			}

			if err := act.Execute(ctx, rawAction.Content); err != nil {
				fmt.Printf("Error executing action %s: %v\n", rawAction.Type, err)
			}

			var contentStr string
			if rawAction.Content != nil {
				if err := json.Unmarshal(rawAction.Content, &contentStr); err != nil {
					contentStr = string(rawAction.Content)
				}
			}
			botResp.Actions = append(botResp.Actions, Action{Type: rawAction.Type, Content: contentStr})
		}

		if len(toolOutputs) > 0 {
			recursionDepth++
			combinedOutputs := strings.Join(toolOutputs, "\n")
			currentMsg = fmt.Sprintf("%s\n\n[System: Tool Results]\n%s", currentMsg, combinedOutputs)
			fmt.Printf("[Bot/Task] Tool outputs received, recursing (depth %d)...\n", recursionDepth)
			continue
		}

		break
	}

	return botResp, nil
}

// ProcessBehaviors processes a message with active behaviors enabled
func (b *Bot) ProcessBehaviors(activeBehaviors []behaviors.Behavior, msg string, context []string, sendToContact func(string)) (*BotResponse, error) {
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

	// 3. Prepare Behaviors Content
	var behaviorsContent strings.Builder
	for _, behavior := range activeBehaviors {
		// Verify file exists in config/modes/behavior/<Name>.txt
		path := filepath.Join(b.ConfigDir, "modes", "behavior", behavior.Name+".txt")
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Warning: failed to read behavior file %s: %v\n", behavior.Name, err)
			continue
		}
		behaviorsContent.WriteString(fmt.Sprintf("\n--- Behavior: %s (Comments: %s) ---\n", behavior.Name, behavior.Comments))
		behaviorsContent.WriteString(string(content))
		behaviorsContent.WriteString("\n-------------------------------------\n")
	}

	// 4. Load Behavior Prompt
	behaviorData := prompt.BehaviorData{
		ModeData: prompt.ModeData{
			Memories:         string(memoriesContent),
			Tasks:            "[]",
			Contacts:         "[]",
			Context:          strings.Join(context, "\n"),
			Message:          msg,
			AvailableActions: b.getAvailableActionsJSON(),
		},
		EnabledBehaviors: behaviorsContent.String(),
	}

	behaviorPrompt, err := b.PromptManager.LoadBehaviorPrompt(behaviorData)
	if err != nil {
		return nil, fmt.Errorf("failed to load behavior prompt: %w", err)
	}

	msgs := []llm.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: behaviorPrompt},
	}

	respMsg, err := b.Client.Chat(msgs, map[string]interface{}{"log_tag": "behavior"})
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	// 5. Parse Response
	content := respMsg.Content
	content = cleanJSON(content)

	var rawResp RawBotResponse
	if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
		fmt.Printf("Raw response: %s\n", respMsg.Content)
		// Don't fail hard on JSON parse if it's just chatter, but behaviors should return actions usually.
		// If fails, maybe the LLM just talked?
		// We'll treat it as error for now to match other modes.
		return nil, fmt.Errorf("failed to parse bot response json: %w", err)
	}

	botResp := &BotResponse{}

	// 6. Execute Actions (with recursion support similar to tasks)
	maxRecursion := 5
	recursionDepth := 0
	currentMsg := msg

	for {
		if recursionDepth > maxRecursion {
			break
		}

		if recursionDepth > 0 {
			behaviorData.Message = currentMsg
			behaviorPrompt, err = b.PromptManager.LoadBehaviorPrompt(behaviorData)
			if err != nil {
				return nil, fmt.Errorf("failed to reload behavior prompt: %w", err)
			}
			msgs[1].Content = behaviorPrompt

			respMsg, err = b.Client.Chat(msgs, map[string]interface{}{"log_tag": "behavior"})
			if err != nil {
				return nil, fmt.Errorf("ollama chat failed in recursion: %w", err)
			}
			content = cleanJSON(respMsg.Content)
			if err := json.Unmarshal([]byte(content), &rawResp); err != nil {
				return nil, fmt.Errorf("failed to parse bot response json: %w", err)
			}
		}

		toolOutputs := []string{}

		for i, rawAction := range rawResp.Actions {
			fmt.Printf("[Bot/Behavior] Processing action %d: type=%s\n", i+1, rawAction.Type)

			act, ok := b.ActionRegistry.Get(rawAction.Type)
			if !ok {
				fmt.Printf("Warning: received unknown action '%s' in behavior mode\n", rawAction.Type)
				continue
			}

			// Note: Task is nil for behaviors
			ctx := actions.ActionContext{
				Context:         context,
				BehaviorManager: b.BehaviorManager,
				SendToContact:   sendToContact,
				ToolOutputs:     &toolOutputs,
			}

			if err := act.Execute(ctx, rawAction.Content); err != nil {
				fmt.Printf("Error executing action %s: %v\n", rawAction.Type, err)
			}

			var contentStr string
			if rawAction.Content != nil {
				if err := json.Unmarshal(rawAction.Content, &contentStr); err != nil {
					contentStr = string(rawAction.Content)
				}
			}
			botResp.Actions = append(botResp.Actions, Action{Type: rawAction.Type, Content: contentStr})
		}

		if len(toolOutputs) > 0 {
			recursionDepth++
			combinedOutputs := strings.Join(toolOutputs, "\n")
			currentMsg = fmt.Sprintf("%s\n\n[System: Tool Results]\n%s", currentMsg, combinedOutputs)
			continue
		}
		break
	}

	return botResp, nil
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
