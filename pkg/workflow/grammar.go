package workflow

import (
	"context"
	"fmt"
	"whatsabladerunner/pkg/ollama"
)

type GrammarStep struct {
	OllamaClient *ollama.Client
}

func NewGrammarStep(client *ollama.Client) *GrammarStep {
	return &GrammarStep{OllamaClient: client}
}

func (s *GrammarStep) Name() string {
	return "Grammar Check"
}

func (s *GrammarStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}

	prompt := fmt.Sprintf(`Check the grammar of the following text. 
If there are errors, correct them and return ONLY the corrected text.
If there are no errors, return the original text.
Do not add any explanations.
Text: "%s"`, text)

	msgs := []ollama.Message{
		{Role: "user", Content: prompt},
	}

	resp, err := s.OllamaClient.Chat(msgs)
	if err != nil {
		return nil, err
	}

	return resp.Content, nil
}
