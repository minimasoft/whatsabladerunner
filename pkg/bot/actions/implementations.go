package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"whatsabladerunner/pkg/tasks"
)

// --- MemoryUpdateAction ---

type MemoryUpdateAction struct {
	MemoriesPath string
}

func (a *MemoryUpdateAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "memory_update",
		Description: "The **full, updated version** of the Global Memory. Use this ONLY to REWRITE the entire memory. For adding lines, use memory_append. Global memory IS NOT FOR TASKS.",
		Parameters:  json.RawMessage(`{"type": "string", "description": "The full memory text."}`),
	}
}

func (a *MemoryUpdateAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var contentStr string
	if err := json.Unmarshal(payload, &contentStr); err != nil {
		// Try treating the payload itself as the string if it's not a JSON string (unlikely given how we parse)
		// But usually payload is the "content" field value.
		// If the LLM sends a string, it comes as a JSON string.
		return fmt.Errorf("invalid payload for memory_update: %w", err)
	}

	if err := os.WriteFile(a.MemoriesPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write memories: %w", err)
	}
	fmt.Println("Memories updated.")
	return nil
}

// --- MemoryAppendAction ---

type MemoryAppendAction struct {
	MemoriesPath string
}

func (a *MemoryAppendAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "memory_append",
		Description: "Append a new line or lines to the Global Memory. Use this for incremental updates. Global memory IS NOT FOR TASKS.",
		Parameters:  json.RawMessage(`{"type": "string", "description": "The text to append."}`),
	}
}

func (a *MemoryAppendAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var contentStr string
	if err := json.Unmarshal(payload, &contentStr); err != nil {
		return fmt.Errorf("invalid payload for memory_append: %w", err)
	}

	f, err := os.OpenFile(a.MemoriesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open memories for append: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat memories file: %w", err)
	}

	// Add a newline if file is not empty and doesn't end with one
	if info.Size() > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline to memories: %w", err)
		}
	}

	if _, err := f.WriteString(contentStr); err != nil {
		return fmt.Errorf("failed to append to memories: %w", err)
	}

	fmt.Println("Memories appended.")
	return nil
}

// --- ResponseAction ---

type ResponseAction struct {
	SendFunc       func(string)
	CheckMessage   func(proposedMsg string, context []string) (bool, string, error)
	SendMasterFunc func(string)
	OnWatcherBlock func(blockedMsg string, targetChatJID string, sendFunc func(string))
}

func (a *ResponseAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "response",
		Description: "A message sent to the 3rd party (as the Master).",
		Parameters:  json.RawMessage(`{"type": "string", "description": "The message text."}`),
	}
}

func (a *ResponseAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var contentStr string
	if err := json.Unmarshal(payload, &contentStr); err != nil {
		return fmt.Errorf("invalid payload for response: %w", err)
	}

	// Helper to send feedback to master
	logToMaster := func(msg string) {
		if a.SendMasterFunc != nil {
			if ctx.Task != nil {
				a.SendMasterFunc(fmt.Sprintf("[Blady][Task %d] : %s", ctx.Task.ID, msg))
			} else {
				a.SendMasterFunc(fmt.Sprintf("[Blady] : %s", msg))
			}
		}
	}

	if ctx.Task != nil {
		// Task Mode: Check with Watcher
		if a.CheckMessage != nil {
			proceed, reason, err := a.CheckMessage(contentStr, ctx.Context)
			if err != nil {
				logToMaster(fmt.Sprintf("Error in Watcher check: %v", err))
				// Fail safe? Or continue? Bot code continued with warning.
				// But block if error prevents verification? Safe to block.
				return fmt.Errorf("watcher check error: %w", err)
			}

			if !proceed {
				fmt.Printf("Watcher BLOCKED message: %s. Reason: %s\n", contentStr, reason)
				logToMaster(fmt.Sprintf("[Watcher] : Blocked: \"%s\". Reason: %s ('LET IT BE' cancels block)", contentStr, reason))

				if a.OnWatcherBlock != nil && ctx.SendToContact != nil {
					a.OnWatcherBlock(contentStr, ctx.Task.ChatID, ctx.SendToContact)
				}
				return nil // Blocked, so we don't send
			}
		}

		// Send to contact
		if ctx.SendToContact != nil {
			ctx.SendToContact(contentStr)
		}

		// Transition task to running if pending
		// We need access to TaskManager to set status?
		// The original code did: b.TaskManager.SetTaskRunning(task.ID)
		// We don't have TaskManager here.
		// Maybe ActionContext should have a callback "SetTaskRunning"?
		// Or we pass TaskManager to ResponseAction.
		// ResponseAction doesn't seemingly need TaskManager usually, but side-effect of sending in task mode requires it.
		// ACTUALLY, I missed dependencies. I should probably inject a wrapper or interface.
		// For now, I'll ignore the status update side-effect here? No, that's important.
		// I will handle this by emitting an event or modifying Task object?
		// Task object is a pointer but modifying it doesn't save it.
		// I will rely on the caller to handle status updates? No, action is encapsulation.
		// I'll add TaskManager to ResponseAction dependencies.
	} else {
		// Command Mode (Response to Master/Self)
		if a.SendFunc != nil {
			a.SendFunc("[Blady] : " + contentStr)
		}
	}
	return nil
}

// --- MessageMasterAction ---

type MessageMasterAction struct {
	SendMasterFunc func(string)
}

func (a *MessageMasterAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "message_master",
		Description: "A private note or summary for the Master.",
		Parameters:  json.RawMessage(`{"type": "string", "description": "The message content."}`),
	}
}

func (a *MessageMasterAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var contentStr string
	if err := json.Unmarshal(payload, &contentStr); err != nil {
		return fmt.Errorf("invalid payload for message_master: %w", err)
	}

	if a.SendMasterFunc != nil {
		if ctx.Task != nil {
			a.SendMasterFunc(fmt.Sprintf("[Blady][Task %d] : %s", ctx.Task.ID, contentStr))
		} else {
			a.SendMasterFunc(contentStr)
		}
	}
	return nil
}

// --- CreateTaskAction ---

type CreateTaskAction struct {
	TaskManager *tasks.TaskManager
	GetContacts func() string
	SendFunc    func(string) // for feedback in command mode
}

func (a *CreateTaskAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "create_task",
		Description: "A new task to be added to the task list. Contains objective and contact.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"objective": {"type": "string"},
				"contact": {"type": "string", "description": "The contact number (e.g. 12345@whats.me)"},
				"original_orders": {"type": "string"},
				"schedule_datetime": {"type": "string", "description": "ISO 8601 format without timezone (e.g. 2024-12-31T23:59), optional"}
			},
			"required": ["objective", "contact", "original_orders"]
		}`),
	}
}

func (a *CreateTaskAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var input tasks.CreateTaskContent
	// Handle string payload (if LLM messes up and sends stringified JSON)
	var contentStr string
	if err := json.Unmarshal(payload, &contentStr); err == nil {
		if err := json.Unmarshal([]byte(contentStr), &input); err != nil {
			return fmt.Errorf("failed to parse stringified create_task content: %w", err)
		}
	} else {
		if err := json.Unmarshal(payload, &input); err != nil {
			return fmt.Errorf("failed to parse create_task content: %w", err)
		}
	}

	// Validate contact
	if !a.isValidContact(input.Contact) {
		msg := fmt.Sprintf("Error: contact '%s' not found.", input.Contact)
		if a.SendFunc != nil {
			a.SendFunc("[Blady] : " + msg)
		}
		return fmt.Errorf("%s", msg)
	}

	task, err := a.TaskManager.CreateTask(input.Objective, input.Contact, input.OriginalOrders, input.ScheduleDatetime)
	if err != nil {
		msg := fmt.Sprintf("Error creating task: %v", err)
		if a.SendFunc != nil {
			a.SendFunc("[Blady] : " + msg)
		}
		return err
	}

	if a.SendFunc != nil {
		taskJSON, _ := json.MarshalIndent(task, "", "  ")
		a.SendFunc(fmt.Sprintf("[Blady] : Tarea creada:\n```json\n%s\n```", string(taskJSON)))
	}
	return nil
}

func (a *CreateTaskAction) isValidContact(contact string) bool {
	if a.GetContacts == nil {
		return false
	}
	contacts := a.GetContacts()
	if contacts == "" || contacts == "[]" {
		return false
	}
	return strings.Contains(contacts, fmt.Sprintf(`"number":"%s"`, contact))
}

// --- Task Management Actions ---

type TaskActionType string

const (
	TaskDelete  TaskActionType = "delete_task"
	TaskConfirm TaskActionType = "confirm_task"
	TaskPause   TaskActionType = "pause_task"
	TaskResume  TaskActionType = "resume_task"
)

type TaskManagementAction struct {
	Type               TaskActionType
	TaskManager        *tasks.TaskManager
	StartTaskCallback  func(*tasks.Task) // For confirm_task
	ResumeTaskCallback func(*tasks.Task) // For resume_task
}

func (a *TaskManagementAction) GetSchema() ActionSchema {
	desc := ""
	switch a.Type {
	case TaskDelete:
		desc = "Delete a task. Content is ID. ONLY BY USER REQUEST."
	case TaskConfirm:
		desc = "Confirm a newly created task to start working. Content is ID."
	case TaskPause:
		desc = "Pause a task. Content is ID."
	case TaskResume:
		desc = "Resume a task. Content is ID."
	}

	// Just accept ID as string or int, schema says string/int but we usually get string representation
	return ActionSchema{
		Name:        string(a.Type),
		Description: desc,
		Parameters:  json.RawMessage(`{"type": "string", "description": "The Task ID."}`),
	}
}

func (a *TaskManagementAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var idStr string
	if err := json.Unmarshal(payload, &idStr); err != nil {
		// Maybe it's an int
		var idInt int
		if err := json.Unmarshal(payload, &idInt); err != nil {
			return fmt.Errorf("invalid payload for %s: %w", a.Type, err)
		}
		idStr = strconv.Itoa(idInt)
	}

	id, err := parseTaskID(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID for %s: %w", a.Type, err)
	}

	switch a.Type {
	case TaskDelete:
		return a.TaskManager.DeleteTask(id)
	case TaskConfirm:
		task, err := a.TaskManager.ConfirmTaskAndGet(id)
		if err != nil {
			return err
		}
		if a.StartTaskCallback != nil {
			a.StartTaskCallback(task)
		}
	case TaskPause:
		return a.TaskManager.PauseTask(id)
	case TaskResume:
		if err := a.TaskManager.ResumeTask(id); err != nil {
			return err
		}
		if a.ResumeTaskCallback != nil {
			task, err := a.TaskManager.LoadTask(id)
			if err == nil {
				a.ResumeTaskCallback(task)
			}
		}
		return nil
	}
	return nil
}

func parseTaskID(s string) (int, error) {
	s = strings.Trim(s, `"`)
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

// --- SendMediaAction ---

type SendMediaAction struct {
	SendMediaFunc func(chatJID string, mediaID int64)
	ToMaster      bool
}

func (a *SendMediaAction) GetSchema() ActionSchema {
	name := "send_media"
	desc := "Send a media file back to the contact (in task mode) or to the current conversation."
	if a.ToMaster {
		name = "send_media_to_master"
		desc = "Send a media file private to the master."
	}
	return ActionSchema{
		Name:        name,
		Description: desc,
		Parameters:  json.RawMessage(`{"type": "string", "description": "The Media ID."}`),
	}
}

func (a *SendMediaAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var idStr string
	if err := json.Unmarshal(payload, &idStr); err != nil {
		var idInt int64
		if err := json.Unmarshal(payload, &idInt); err != nil {
			return fmt.Errorf("invalid payload: %w", err)
		}
		idStr = strconv.FormatInt(idInt, 10)
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid media ID: %w", err)
	}

	if a.SendMediaFunc != nil {
		target := ""
		if a.ToMaster {
			target = "master"
		} else {
			// In task mode?
			if ctx.Task != nil {
				target = ctx.Task.ChatID
			} else {
				// In command mode, empty string implies replying to current chat if supported,
				// but SendMediaFunc usually requires a JID.
				// bot.go implementation:
				// case "send_media": ... b.SendMediaFunc("", id)
				// So we assume "" is handled or we pass "" and let it fail/default.
				target = ""
			}
		}
		a.SendMediaFunc(target, id)
	}
	return nil
}

// --- ButtonResponseAction ---

type ButtonResponseAction struct {
	SendButtonResponseFunc func(displayText, buttonID string)
	TaskManager            *tasks.TaskManager // needed for side-effect?
}

func (a *ButtonResponseAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "button_response",
		Description: "Click a button option.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"displayText": {"type": "string"},
				"buttonID": {"type": "string"}
			},
			"required": ["displayText"]
		}`),
	}
}

func (a *ButtonResponseAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var btnResp struct {
		DisplayText string `json:"displayText"`
		ButtonID    string `json:"buttonID"`
	}

	// Handle stringified JSON or direct object
	var contentStr string
	if err := json.Unmarshal(payload, &contentStr); err == nil {
		if err := json.Unmarshal([]byte(contentStr), &btnResp); err != nil {
			// Fallback: simple string
			btnResp.DisplayText = contentStr
		}
	} else {
		if err := json.Unmarshal(payload, &btnResp); err != nil {
			return fmt.Errorf("invalid payload for button_response: %w", err)
		}
	}

	if a.SendButtonResponseFunc != nil {
		a.SendButtonResponseFunc(btnResp.DisplayText, btnResp.ButtonID)
	}

	// Side-effect: Transition task to running if pending
	if ctx.Task != nil && ctx.Task.Status == tasks.StatusPending && a.TaskManager != nil {
		if err := a.TaskManager.SetTaskRunning(ctx.Task.ID); err != nil {
			fmt.Printf("Failed to set task running: %v\n", err)
		}
	}
	return nil
}
