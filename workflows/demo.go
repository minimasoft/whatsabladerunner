package workflows

import (
	"context"
	"fmt"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/pkg/workflow"
)

type DemoWorkflow struct {
	OllamaClient *ollama.Client
}

func NewDemoWorkflow(client *ollama.Client) *DemoWorkflow {
	return &DemoWorkflow{OllamaClient: client}
}

// SendMessageStep is a final step to simulate sending the message.
type SendMessageStep struct{}

func (s *SendMessageStep) Name() string {
	return "Send Message"
}

func (s *SendMessageStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	msg, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}
	fmt.Printf(">>> SENDING MESSAGE: %s\n", msg)
	return msg, nil
}

func (d *DemoWorkflow) Run(ctx context.Context, input string) {
	watcherStep := workflow.NewWatcherStep(d.OllamaClient)
	sendStep := &SendMessageStep{}

	wf := workflow.NewWorkflow("Demo Workflow", watcherStep, sendStep)

	_, err := wf.Execute(ctx, input)
	if err != nil {
		fmt.Printf("Workflow failed: %v\n", err)
	}
}
