package actions

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomAction(t *testing.T) {
	// 1. Setup Mock Server
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// 2. Setup Config File
	tmpDir := t.TempDir()
	configContent := `{
		"name": "test_action",
		"description": "A test action",
		"url": "` + server.URL + `",
		"response_to_llm": true,
		"parameters": {"type": "object"}
	}`
	configPath := filepath.Join(tmpDir, "test_action.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 3. Load Actions
	registry := NewRegistry()
	if err := LoadCustomActions(tmpDir, registry); err != nil {
		t.Fatalf("LoadCustomActions failed: %v", err)
	}

	// 4. Verify Registration
	action, ok := registry.Get("test_action")
	if !ok {
		t.Fatalf("Action 'test_action' not registered")
	}

	// 5. Execute Action
	ctx := ActionContext{
		ToolOutputs: &[]string{},
	}
	payload := json.RawMessage(`{"foo": "bar"}`)
	if err := action.Execute(ctx, payload); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// 6. Verify Request
	expected := `{"foo":"bar"}`
	if string(receivedBody) != expected {
		t.Errorf("Expected body %s, got %s", expected, string(receivedBody))
	}

	// 7. Verify Output
	if len(*ctx.ToolOutputs) != 1 {
		t.Errorf("Expected 1 tool output, got %d", len(*ctx.ToolOutputs))
	}
}

func TestCustomAction_GET(t *testing.T) {
	// 1. Setup Mock Server
	var receivedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// 2. Setup Config
	config := CustomActionConfig{
		Name:        "get_action",
		Description: "A get action",
		Method:      "GET",
		URL:         server.URL + "/test",
	}
	action := &CustomAction{Config: config}

	// 3. Execute
	ctx := ActionContext{ToolOutputs: &[]string{}}
	payload := json.RawMessage(`{"id": "123", "foo": "bar"}`)
	if err := action.Execute(ctx, payload); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// 4. Verify URL (should have query params)
	if !strings.Contains(receivedURL, "id=123") || !strings.Contains(receivedURL, "foo=bar") {
		t.Errorf("URL %s does not contain expected query params", receivedURL)
	}
}

func TestCustomAction_Templating(t *testing.T) {
	// 1. Setup Mock Server
	var receivedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 2. Setup Config with template
	config := CustomActionConfig{
		Name:   "template_action",
		Method: "GET",
		URL:    server.URL + "/docs/{doc_id}",
	}
	action := &CustomAction{Config: config}

	// 3. Execute
	ctx := ActionContext{ToolOutputs: &[]string{}}
	payload := json.RawMessage(`{"doc_id": "456", "extra": "val"}`)
	if err := action.Execute(ctx, payload); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// 4. Verify URL
	expectedPath := "/docs/456"
	if !strings.Contains(receivedURL, expectedPath) {
		t.Errorf("Expected path %s in URL %s", expectedPath, receivedURL)
	}
	if !strings.Contains(receivedURL, "extra=val") {
		t.Errorf("Expected query param extra=val in URL %s", receivedURL)
	}
}
