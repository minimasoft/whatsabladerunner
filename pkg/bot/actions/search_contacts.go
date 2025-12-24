package actions

import (
	"encoding/json"
	"fmt"
)

type SearchContactsAction struct {
	SearchFunc func(query string) string
}

func (a *SearchContactsAction) GetSchema() ActionSchema {
	return ActionSchema{
		Name:        "search_contacts",
		Description: "Search for a contact's JID by name (fuzzy match) or number. Use this to find the JID before sending messages or creating tasks if the exact JID is unknown.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "The name or number term to search for (case-insensitive)"
				}
			},
			"required": ["query"]
		}`),
	}
}

func (a *SearchContactsAction) Execute(ctx ActionContext, payload json.RawMessage) error {
	var input struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(payload, &input); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	if a.SearchFunc == nil {
		return fmt.Errorf("SearchFunc not initialized")
	}

	result := a.SearchFunc(input.Query)

	output := fmt.Sprintf("[search_contacts] Results for '%s':\n%s", input.Query, result)

	if ctx.ToolOutputs != nil {
		*ctx.ToolOutputs = append(*ctx.ToolOutputs, output)
	}

	return nil
}
