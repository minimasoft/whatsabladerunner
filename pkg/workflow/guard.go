package workflow

import (
	"context"
	"fmt"
	"strings"
	"whatsabladerunner/pkg/ollama"
)

type GuardStep struct {
	OllamaClient *ollama.Client
}

func NewGuardStep(client *ollama.Client) *GuardStep {
	return &GuardStep{OllamaClient: client}
}

func (s *GuardStep) Name() string {
	return "Guard Check"
}

func (s *GuardStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}

	fmt.Printf("Running Guard on: %s\n", text)

	prompt := fmt.Sprintf(`Analyze the following text for insults or harmful content. 
If it is safe, reply with exactly "SAFE".
If it is unsafe, reply with exactly "UNSAFE".
Text: "%s"`, text)

	// We'll use a simple chat request or generate. Chat is fine.
	msgs := []ollama.Message{
		{Role: "user", Content: prompt},
	}

	resp, err := s.OllamaClient.Chat(msgs)
	if err != nil {
		return nil, err
	}

	verdict := strings.TrimSpace(strings.ToUpper(resp.Content))
	if strings.Contains(verdict, "SAFE") && !strings.Contains(verdict, "UNSAFE") {
		return text, nil
	}

	return "Blocked by Guard: Content was deemed unsafe.", nil // Or error if we want to stop workflow
}
