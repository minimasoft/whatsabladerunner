package workflows

import (
	"context"
	"fmt"
	"whatsabladerunner/pkg/bot"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/pkg/workflow"
)

// BotProcessStep calls the bot to process the message.
type BotProcessStep struct {
	Bot         *bot.Bot
	Mode        string
	ContextMsgs []string
}

func (s *BotProcessStep) Name() string {
	return "Bot Process"
}

func (s *BotProcessStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	msg, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("expected string input")
	}

	response, err := s.Bot.Process(s.Mode, msg, s.ContextMsgs)
	if err != nil {
		return nil, fmt.Errorf("bot process failed: %w", err)
	}

	return response, nil
}

// ResponseHandlerStep removed as Bot handles it now.

type CommandWorkflow struct {
	Bot      *bot.Bot
	SendFunc func(string)
}

func NewCommandWorkflow(client *ollama.Client, sendFunc func(string), sendMasterFunc func(string)) *CommandWorkflow {
	// Assuming config is in "config" dir relative to CWD
	// Pass sendFunc to Bot so it can handle response actions.
	b := bot.NewBot(client, "config", sendFunc, sendMasterFunc)
	return &CommandWorkflow{
		Bot:      b,
		SendFunc: sendFunc,
	}
}

func (c *CommandWorkflow) Run(ctx context.Context, input string, contextMsgs []string) {
	processStep := &BotProcessStep{
		Bot:         c.Bot,
		Mode:        "command",
		ContextMsgs: contextMsgs,
	}

	// responseStep is no longer needed as Bot handles the response action directly.

	wf := workflow.NewWorkflow("Command Workflow", processStep)

	_, err := wf.Execute(ctx, input)
	if err != nil {
		fmt.Printf("Command Workflow failed: %v\n", err)
	}
}
