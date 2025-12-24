package actions

import (
	"encoding/json"
	"testing"
)

func TestRegistry_RegistrationAndRetrieval(t *testing.T) {
	r := NewRegistry()
	action := &MemoryUpdateAction{MemoriesPath: "/tmp/mem"}
	r.Register(action)

	// Test Get
	retrieved, ok := r.Get("memory_update")
	if !ok {
		t.Error("Failed to retrieve registered action 'memory_update'")
	}
	if retrieved != action {
		t.Error("Retrieved action does not match registered action")
	}

	// Test Validate
	if err := r.Validate("memory_update"); err != nil {
		t.Errorf("Validate failed for existing action: %v", err)
	}
	if err := r.Validate("non_existent"); err == nil {
		t.Error("Validate should fail for non-existent action")
	}
}

func TestRegistry_GetSchemas(t *testing.T) {
	r := NewRegistry()
	r.Register(&ResponseAction{})
	r.Register(&MemoryUpdateAction{})
	r.Register(&CreateTaskAction{})

	schemas := r.GetSchemas()
	if len(schemas) != 3 {
		t.Errorf("Expected 3 schemas, got %d", len(schemas))
	}

	// Verify one schema content
	var responseSchema ActionSchema
	found := false
	for _, s := range schemas {
		if s.Name == "response" {
			responseSchema = s
			found = true
			break
		}
	}

	if !found {
		t.Fatal("Action 'response' not found in schemas")
	}

	if responseSchema.Description == "" {
		t.Error("Schema description should not be empty")
	}

	// Verify JSON serialization
	data, err := json.Marshal(schemas)
	if err != nil {
		t.Fatalf("Failed to marshal schemas: %v", err)
	}

	jsonStr := string(data)
	if len(jsonStr) < 10 {
		t.Error("JSON output too short")
	}
}

func TestActionExecution_Response(t *testing.T) {
	sentMsg := ""
	sendFunc := func(msg string) {
		sentMsg = msg
	}

	act := &ResponseAction{
		SendFunc: sendFunc,
	}

	ctx := ActionContext{}
	payload := json.RawMessage(`"Hello World"`)

	if err := act.Execute(ctx, payload); err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	expected := "[Blady] : Hello World"
	if sentMsg != expected {
		t.Errorf("Expected message '%s', got '%s'", expected, sentMsg)
	}
}
