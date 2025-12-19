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

// ResponseHandlerStep handles the bot response actions (specifically sending messages).
type ResponseHandlerStep struct {
	SendFunc func(string)
}

func (s *ResponseHandlerStep) Name() string {
	return "handle Response"
}

func (s *ResponseHandlerStep) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	resp, ok := input.(*bot.BotResponse)
	if !ok {
		return nil, fmt.Errorf("expected *bot.BotResponse input")
	}

	for _, action := range resp.Actions {
		if action.Type == "response" {
			// Prefix logic could be here or in the prompt.
			// The previous logic added "[Blady] : ".
			// The user said: "Supported action response is the current to send message back".
			// And "Replies with the prefix [Blady] : ".
			// I should probably keep the prefix logic here or rely on the LLM to include it?
			// The prompt says "You can respond...". It doesn't explicitly enforce prefix in the prompt template provided.
			// The user requirement in Task 0 was "The answer from ollama should be sent back with [Blady] : prefix".
			// So I should append it here.
			finalMsg := "[Blady] : " + action.Response
			s.SendFunc(finalMsg)
		}
	}

	return resp, nil
}

type CommandWorkflow struct {
	Bot      *bot.Bot
	SendFunc func(string)
}

func NewCommandWorkflow(client *ollama.Client, sendFunc func(string)) *CommandWorkflow {
	// Assuming config is in "config" dir relative to CWD
	b := bot.NewBot(client, "config")
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

	responseStep := &ResponseHandlerStep{
		SendFunc: c.SendFunc,
	}

	wf := workflow.NewWorkflow("Command Workflow", processStep, responseStep)

	_, err := wf.Execute(ctx, input)
	if err != nil {
		fmt.Printf("Command Workflow failed: %v\n", err)
	}
}
