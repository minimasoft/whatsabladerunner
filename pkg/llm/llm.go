package llm

// Message represents a chat message with role and content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client is the common interface for LLM providers.
type Client interface {
	Chat(messages []Message, options map[string]interface{}) (*Message, error)
}
