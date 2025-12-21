package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Task status constants
const (
	StatusUnconfirmed = "unconfirmed"
	StatusPending     = "pending"
	StatusRunning     = "running"
	StatusPaused      = "paused"
	StatusFinished    = "finished"
)

// Task represents a task stored as a JSON file
type Task struct {
	ID                     int    `json:"id"`
	Objective              string `json:"objective"`
	OriginalOrders         string `json:"original_orders"`
	Contact                string `json:"contact"`
	ChatID                 string `json:"chat_id,omitempty"` // Chat JID where task is active (may differ from Contact for bots)
	Status                 string `json:"status"`
	LastProcessedTimestamp int64  `json:"last_processed_timestamp,omitempty"` // Unix timestamp of last processed message
}

// CreateTaskContent represents the content of a create_task action
type CreateTaskContent struct {
	Objective      string `json:"objective"`
	Contact        string `json:"contact"`
	OriginalOrders string `json:"original_orders"`
}

// TaskManager handles all task file operations
type TaskManager struct {
	TasksDir   string
	DeletedDir string
}

// NewTaskManager creates a new TaskManager for the given tasks directory
func NewTaskManager(tasksDir string) *TaskManager {
	return &TaskManager{
		TasksDir:   tasksDir,
		DeletedDir: filepath.Join(tasksDir, "deleted"),
	}
}

// getNextID checks the last id in '_last_id' file in the tasks directory (stored as string).
// If the file doesn't exist write 1, if it exists read, add 1, write and continue with the current value.
func (tm *TaskManager) getNextID() (int, error) {
	lastIDPath := filepath.Join(tm.TasksDir, "_last_id")

	// Ensure directory exists
	if err := os.MkdirAll(tm.TasksDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create tasks directory: %w", err)
	}

	data, err := os.ReadFile(lastIDPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, write "1" and return 1
			if err := os.WriteFile(lastIDPath, []byte("1"), 0644); err != nil {
				return 0, fmt.Errorf("failed to write initial _last_id: %w", err)
			}
			return 1, nil
		}
		return 0, fmt.Errorf("failed to read _last_id: %w", err)
	}

	// Read, increment, and write back
	idStr := strings.TrimSpace(string(data))
	lastID, err := strconv.Atoi(idStr)
	if err != nil {
		// Fallback: if file is corrupted, return error or try to recover?
		// Given it's a "fix", let's return error to be safe.
		return 0, fmt.Errorf("failed to parse _last_id '%s': %w", idStr, err)
	}

	nextID := lastID + 1
	if err := os.WriteFile(lastIDPath, []byte(strconv.Itoa(nextID)), 0644); err != nil {
		return 0, fmt.Errorf("failed to update _last_id: %w", err)
	}

	return nextID, nil
}

// taskPath returns the file path for a given task ID
func (tm *TaskManager) taskPath(id int) string {
	return filepath.Join(tm.TasksDir, fmt.Sprintf("%d.json", id))
}

// LoadActiveTasks loads all tasks with active statuses (unconfirmed, pending, running, paused)
func (tm *TaskManager) LoadActiveTasks() ([]Task, error) {
	entries, err := os.ReadDir(tm.TasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	var tasks []Task
	activeStatuses := map[string]bool{
		StatusUnconfirmed: true,
		StatusPending:     true,
		StatusRunning:     true,
		StatusPaused:      true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		// Skip sample file
		if name == "0_sample.json" {
			continue
		}

		filePath := filepath.Join(tm.TasksDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: failed to read task file %s: %v\n", name, err)
			continue
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			fmt.Printf("Warning: failed to parse task file %s: %v\n", name, err)
			continue
		}

		if activeStatuses[task.Status] {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// LoadTask loads a single task by ID
func (tm *TaskManager) LoadTask(id int) (*Task, error) {
	filePath := tm.taskPath(id)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task %d: %w", id, err)
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task %d: %w", id, err)
	}

	return &task, nil
}

// SaveTask saves a task to its file
func (tm *TaskManager) SaveTask(task *Task) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task %d: %w", task.ID, err)
	}

	filePath := tm.taskPath(task.ID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task %d: %w", task.ID, err)
	}

	return nil
}

// CreateTask creates a new task with an auto-incremented ID and unconfirmed status
func (tm *TaskManager) CreateTask(objective, contact, originalOrders string) (*Task, error) {
	nextID, err := tm.getNextID()
	if err != nil {
		return nil, fmt.Errorf("failed to get next ID: %w", err)
	}

	task := &Task{
		ID:             nextID,
		Objective:      objective,
		Contact:        contact,
		OriginalOrders: originalOrders,
		Status:         StatusUnconfirmed,
	}

	if err := tm.SaveTask(task); err != nil {
		return nil, err
	}

	fmt.Printf("[TaskManager] Created task %d: %s (status: %s)\n", task.ID, task.Objective, task.Status)
	return task, nil
}

// DeleteTask moves a task file to the deleted directory
func (tm *TaskManager) DeleteTask(id int) error {
	srcPath := tm.taskPath(id)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("task %d not found", id)
	}

	// Ensure deleted directory exists
	if err := os.MkdirAll(tm.DeletedDir, 0755); err != nil {
		return fmt.Errorf("failed to create deleted directory: %w", err)
	}

	dstPath := filepath.Join(tm.DeletedDir, fmt.Sprintf("%d.json", id))
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move task %d to deleted: %w", id, err)
	}

	fmt.Printf("[TaskManager] Deleted task %d (moved to deleted/)\n", id)
	return nil
}

// ConfirmTask changes task status from unconfirmed to pending
func (tm *TaskManager) ConfirmTask(id int) error {
	task, err := tm.LoadTask(id)
	if err != nil {
		return err
	}

	if task.Status != StatusUnconfirmed {
		return fmt.Errorf("task %d is not unconfirmed (current status: %s)", id, task.Status)
	}

	task.Status = StatusPending
	if err := tm.SaveTask(task); err != nil {
		return err
	}

	fmt.Printf("[TaskManager] Confirmed task %d: status changed to %s\n", id, task.Status)
	return nil
}

// PauseTask changes task status to paused
func (tm *TaskManager) PauseTask(id int) error {
	task, err := tm.LoadTask(id)
	if err != nil {
		return err
	}

	if task.Status != StatusRunning && task.Status != StatusPending {
		return fmt.Errorf("task %d cannot be paused (current status: %s)", id, task.Status)
	}

	task.Status = StatusPaused
	if err := tm.SaveTask(task); err != nil {
		return err
	}

	fmt.Printf("[TaskManager] Paused task %d: status changed to %s\n", id, task.Status)
	return nil
}

// ResumeTask changes task status from paused to running
func (tm *TaskManager) ResumeTask(id int) error {
	task, err := tm.LoadTask(id)
	if err != nil {
		return err
	}

	if task.Status != StatusPaused {
		return fmt.Errorf("task %d is not paused (current status: %s)", id, task.Status)
	}

	task.Status = StatusRunning
	if err := tm.SaveTask(task); err != nil {
		return err
	}

	fmt.Printf("[TaskManager] Resumed task %d: status changed to %s\n", id, task.Status)
	return nil
}

// GetTaskByContact finds an active (running or pending) task for the given contact or chat ID
func (tm *TaskManager) GetTaskByContact(contactOrChatID string) (*Task, error) {
	entries, err := os.ReadDir(tm.TasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || name == "0_sample.json" {
			continue
		}

		filePath := filepath.Join(tm.TasksDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}

		// Check if this task matches by Contact or ChatID and is active (running or pending)
		if (task.Contact == contactOrChatID || task.ChatID == contactOrChatID) && (task.Status == StatusRunning || task.Status == StatusPending) {
			return &task, nil
		}
	}

	return nil, nil
}

// SetTaskRunning changes task status from pending to running
func (tm *TaskManager) SetTaskRunning(id int) error {
	task, err := tm.LoadTask(id)
	if err != nil {
		return err
	}

	if task.Status != StatusPending {
		// Already running or in another state, just return
		if task.Status == StatusRunning {
			return nil
		}
		return fmt.Errorf("task %d is not pending (current status: %s)", id, task.Status)
	}

	task.Status = StatusRunning
	if err := tm.SaveTask(task); err != nil {
		return err
	}

	fmt.Printf("[TaskManager] Task %d now running\n", id)
	return nil
}

// ConfirmTaskAndGet changes status from unconfirmed to pending and returns the task
func (tm *TaskManager) ConfirmTaskAndGet(id int) (*Task, error) {
	task, err := tm.LoadTask(id)
	if err != nil {
		return nil, err
	}

	if task.Status != StatusUnconfirmed {
		return nil, fmt.Errorf("task %d is not unconfirmed (current status: %s)", id, task.Status)
	}

	task.Status = StatusPending
	if err := tm.SaveTask(task); err != nil {
		return nil, err
	}

	fmt.Printf("[TaskManager] Confirmed task %d: status changed to %s\n", id, task.Status)
	return task, nil
}

// SetTaskChatID sets the chat ID for a task (for bots that respond from different JID)
func (tm *TaskManager) SetTaskChatID(id int, chatID string) error {
	task, err := tm.LoadTask(id)
	if err != nil {
		return err
	}

	oldChatID := task.ChatID
	task.ChatID = chatID
	if err := tm.SaveTask(task); err != nil {
		return err
	}

	fmt.Printf("[TaskManager] Task %d chat ID updated: '%s' -> '%s'\n", id, oldChatID, chatID)
	return nil
}

// SetTaskProcessedTimestamp updates the last processed timestamp for a task
func (tm *TaskManager) SetTaskProcessedTimestamp(id int, timestamp int64) error {
	task, err := tm.LoadTask(id)
	if err != nil {
		return err
	}

	task.LastProcessedTimestamp = timestamp
	if err := tm.SaveTask(task); err != nil {
		return err
	}

	// fmt.Printf("[TaskManager] Task %d processed timestamp updated to %d\n", id, timestamp)
	return nil
}
