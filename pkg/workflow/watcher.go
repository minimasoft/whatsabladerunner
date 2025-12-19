package workflow

import (
	"context"
	"fmt"
	"strings"
	"whatsabladerunner/pkg/ollama"
)

type WatcherStep struct {
	OllamaClient *ollama.Client
}

func NewWatcherStep(client *ollama.Client) *WatcherStep {
	return &WatcherStep{OllamaClient: client}
}

func (s *WatcherStep) Name() string {
	return "Watcher Check"
}

func (s *WatcherStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}

	fmt.Printf("Running Watcher on: %s\n", text)

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

	return "Blocked by Watcher: Content was deemed unsafe.", nil // Or error if we want to stop workflow
}
