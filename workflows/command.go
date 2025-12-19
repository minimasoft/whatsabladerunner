package workflows

import (
	"context"
	"fmt"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/pkg/workflow"
)

// SenderStep sends a message using a provided function.
type SenderStep struct {
	SendFunc func(string)
}

func (s *SenderStep) Name() string {
	return "Sender"
}

func (s *SenderStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	msg, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}
	s.SendFunc(msg)
	return msg, nil
}

// PrefixStep adds the bot prefix to the message.
type PrefixStep struct {
	Prefix string
}

func (s *PrefixStep) Name() string {
	return "Prefix Message"
}

func (s *PrefixStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	msg, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}
	return s.Prefix + msg, nil
}

// ChatStep sends the input to Ollama with specific options.
type ChatStep struct {
	OllamaClient *ollama.Client
	Options      map[string]interface{}
}

func (s *ChatStep) Name() string {
	return "Chat with Options"
}

func (s *ChatStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	text, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}

	msgs := []ollama.Message{
		{Role: "user", Content: text},
	}

	resp, err := s.OllamaClient.Chat(msgs, s.Options)
	if err != nil {
		return nil, err
	}

	return resp.Content, nil
}

type CommandWorkflow struct {
	OllamaClient *ollama.Client
	SendFunc     func(string)
}

func NewCommandWorkflow(client *ollama.Client, sendFunc func(string)) *CommandWorkflow {
	return &CommandWorkflow{
		OllamaClient: client,
		SendFunc:     sendFunc,
	}
}

func (c *CommandWorkflow) Run(ctx context.Context, input string) {
	// Guard Step (using WatcherStep as defined in watcher.go)
	guardStep := workflow.NewWatcherStep(c.OllamaClient)

	// Chat Step
	chatStep := &ChatStep{
		OllamaClient: c.OllamaClient,
		Options:      nil, // Use defaults from client
	}

	prefixStep := &PrefixStep{Prefix: "[Blady] : "}
	sendStep := &SenderStep{SendFunc: c.SendFunc}

	wf := workflow.NewWorkflow("Command Workflow", guardStep, chatStep, prefixStep, sendStep)

	_, err := wf.Execute(ctx, input)
	if err != nil {
		fmt.Printf("Command Workflow failed: %v\n", err)
		// Optionally send an error message back to user?
		// c.SendFunc("[Blady] : Error: " + err.Error())
	}
}
