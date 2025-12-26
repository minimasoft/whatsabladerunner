package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBehaviorPrompt(t *testing.T) {
	// Setup temporary config directory
	tmpDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create necessary files
	modesDir := filepath.Join(tmpDir, "modes")
	behaviorDir := filepath.Join(modesDir, "behavior")
	if err := os.MkdirAll(behaviorDir, 0755); err != nil {
		t.Fatalf("failed to create modes/behavior dir: %v", err)
	}

	contextContent := "CONTEXT_START\n{{.Context}}\nCONTEXT_END"
	if err := os.WriteFile(filepath.Join(modesDir, "context.txt"), []byte(contextContent), 0644); err != nil {
		t.Fatalf("failed to write context.txt: %v", err)
	}

	baseContent := "BEHAVIOR_START\n{{.EnabledBehaviors}}\nBEHAVIOR_END"
	if err := os.WriteFile(filepath.Join(behaviorDir, "__base.txt"), []byte(baseContent), 0644); err != nil {
		t.Fatalf("failed to write __base.txt: %v", err)
	}

	protocolContent := "PROTOCOL_START\nPROTOCOL_END"
	if err := os.WriteFile(filepath.Join(modesDir, "protocol.txt"), []byte(protocolContent), 0644); err != nil {
		t.Fatalf("failed to write protocol.txt: %v", err)
	}

	pm := NewPromptManager(tmpDir)
	data := BehaviorData{
		ModeData: ModeData{
			Context: "MY_CONTEXT",
		},
		EnabledBehaviors: "MY_BEHAVIOR",
	}

	prompt, err := pm.LoadBehaviorPrompt(data)
	if err != nil {
		t.Fatalf("LoadBehaviorPrompt failed: %v", err)
	}

	// Verify order: Behavior -> Context -> Protocol
	behaviorIdx := strings.Index(prompt, "BEHAVIOR_START")
	contextIdx := strings.Index(prompt, "CONTEXT_START")
	protocolIdx := strings.Index(prompt, "PROTOCOL_START")

	if behaviorIdx == -1 || contextIdx == -1 || protocolIdx == -1 {
		t.Fatalf("Prompt missing sections: behavior=%d, context=%d, protocol=%d", behaviorIdx, contextIdx, protocolIdx)
	}

	if !(behaviorIdx < contextIdx && contextIdx < protocolIdx) {
		t.Errorf("Incorrect prompt order. behaviorIdx=%d, contextIdx=%d, protocolIdx=%d", behaviorIdx, contextIdx, protocolIdx)
	}

	// Verify content replacement
	if !strings.Contains(prompt, "MY_BEHAVIOR") {
		t.Error("Prompt does not contain MY_BEHAVIOR")
	}
	if !strings.Contains(prompt, "MY_CONTEXT") {
		t.Error("Prompt does not contain MY_CONTEXT")
	}
}
