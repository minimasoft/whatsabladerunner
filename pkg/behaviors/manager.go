package behaviors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Behavior status constants
const (
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

// Behavior represents a behavior instance stored as a JSON file
type Behavior struct {
	ID        int    `json:"id"`
	Contact   string `json:"contact"`
	Name      string `json:"name"` // Matches filename in config/modes/behavior/ (without extension)
	Comments  string `json:"comments"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"` // Unix timestamp of creation
}

// BehaviorManager handles all behavior file operations
type BehaviorManager struct {
	BehaviorsDir string
}

// NewBehaviorManager creates a new BehaviorManager for the given behaviors directory
func NewBehaviorManager(behaviorsDir string) *BehaviorManager {
	return &BehaviorManager{
		BehaviorsDir: behaviorsDir,
	}
}

// getNextID gets the next auto-increment ID
func (bm *BehaviorManager) getNextID() (int, error) {
	lastIDPath := filepath.Join(bm.BehaviorsDir, "_last_id")

	// Ensure directory exists
	if err := os.MkdirAll(bm.BehaviorsDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create behaviors directory: %w", err)
	}

	data, err := os.ReadFile(lastIDPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.WriteFile(lastIDPath, []byte("1"), 0644); err != nil {
				return 0, fmt.Errorf("failed to write initial _last_id: %w", err)
			}
			return 1, nil
		}
		return 0, fmt.Errorf("failed to read _last_id: %w", err)
	}

	idStr := strings.TrimSpace(string(data))
	lastID, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse _last_id '%s': %w", idStr, err)
	}

	nextID := lastID + 1
	if err := os.WriteFile(lastIDPath, []byte(strconv.Itoa(nextID)), 0644); err != nil {
		return 0, fmt.Errorf("failed to update _last_id: %w", err)
	}

	return nextID, nil
}

func (bm *BehaviorManager) behaviorPath(id int) string {
	return filepath.Join(bm.BehaviorsDir, fmt.Sprintf("%d.json", id))
}

// EnableBehavior creates a new enabled behavior
func (bm *BehaviorManager) EnableBehavior(contact, name, comments string) (*Behavior, error) {
	// Validate that the behavior template exists in config/modes/behavior/
	// We need to know where config/modes/behavior is.
	// Actually, the Manager only knows about storage. The validation might belong higher up?
	// But it's good to prevent enabling non-existent behaviors.
	// For now, we'll assume the caller (Action) validates or we trust the user.

	nextID, err := bm.getNextID()
	if err != nil {
		return nil, fmt.Errorf("failed to get next ID: %w", err)
	}

	behavior := &Behavior{
		ID:        nextID,
		Contact:   contact,
		Name:      name,
		Comments:  comments,
		Status:    StatusEnabled,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.MarshalIndent(behavior, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal behavior: %w", err)
	}

	if err := os.WriteFile(bm.behaviorPath(behavior.ID), data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write behavior file: %w", err)
	}

	fmt.Printf("[BehaviorManager] Enabled behavior %d: %s for %s\n", behavior.ID, behavior.Name, behavior.Contact)
	return behavior, nil
}

// DisableBehavior removes the behavior file
func (bm *BehaviorManager) DisableBehavior(id int) error {
	path := bm.behaviorPath(id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("behavior %d not found", id)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove behavior file: %w", err)
	}

	fmt.Printf("[BehaviorManager] Disabled (removed) behavior %d\n", id)
	return nil
}

// GetActiveBehaviors returns all enabled behaviors for a specific contact
func (bm *BehaviorManager) GetActiveBehaviors(contact string) ([]Behavior, error) {
	entries, err := os.ReadDir(bm.BehaviorsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Behavior{}, nil
		}
		return nil, fmt.Errorf("failed to read behaviors directory: %w", err)
	}

	var matchBehaviors []Behavior

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == "_last_id" {
			continue
		}

		path := filepath.Join(bm.BehaviorsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var b Behavior
		if err := json.Unmarshal(data, &b); err != nil {
			continue
		}

		if b.Status == StatusEnabled && b.Contact == contact {
			matchBehaviors = append(matchBehaviors, b)
		}
	}

	return matchBehaviors, nil
}

// GetAllActiveBehaviors returns all enabled behaviors across all contacts
func (bm *BehaviorManager) GetAllActiveBehaviors() ([]Behavior, error) {
	entries, err := os.ReadDir(bm.BehaviorsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Behavior{}, nil
		}
		return nil, fmt.Errorf("failed to read behaviors directory: %w", err)
	}

	var matchBehaviors []Behavior

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == "_last_id" {
			continue
		}

		path := filepath.Join(bm.BehaviorsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var b Behavior
		if err := json.Unmarshal(data, &b); err != nil {
			continue
		}

		if b.Status == StatusEnabled {
			matchBehaviors = append(matchBehaviors, b)
		}
	}

	return matchBehaviors, nil
}
