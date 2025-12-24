package actions

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMemoryActions(t *testing.T) {
	tempFile, err := os.CreateTemp("", "memories_test.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	updateAction := &MemoryUpdateAction{MemoriesPath: tempFile.Name()}
	appendAction := &MemoryAppendAction{MemoriesPath: tempFile.Name()}

	// Test Update
	updatePayload := json.RawMessage(`"Initial memory line"`)
	if err := updateAction.Execute(ActionContext{}, updatePayload); err != nil {
		t.Errorf("MemoryUpdateAction.Execute failed: %v", err)
	}

	content, _ := os.ReadFile(tempFile.Name())
	if string(content) != "Initial memory line" {
		t.Errorf("Expected 'Initial memory line', got '%s'", string(content))
	}

	// Test Append
	appendPayload := json.RawMessage(`"Appended line"`)
	if err := appendAction.Execute(ActionContext{}, appendPayload); err != nil {
		t.Errorf("MemoryAppendAction.Execute failed: %v", err)
	}

	content, _ = os.ReadFile(tempFile.Name())
	expected := "Initial memory line\nAppended line"
	if string(content) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(content))
	}

	// Test Update Overwrite
	updatePayload2 := json.RawMessage(`"New content only"`)
	if err := updateAction.Execute(ActionContext{}, updatePayload2); err != nil {
		t.Errorf("MemoryUpdateAction.Execute failed: %v", err)
	}

	content, _ = os.ReadFile(tempFile.Name())
	if string(content) != "New content only" {
		t.Errorf("Expected 'New content only', got '%s'", string(content))
	}
}
