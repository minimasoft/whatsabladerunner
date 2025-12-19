package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL        string
	Model          string
	Client         *http.Client
	DefaultOptions map[string]interface{}
}

func NewClient(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		Client:  &http.Client{Timeout: 90 * time.Second},
		DefaultOptions: map[string]interface{}{
			"temperature": 0.13,
			"num_ctx":     9000,
			"think":       false,
		},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string                 `json:"model"`
	Messages []Message              `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ChatResponse struct {
	Model   string  `json:"model"`
	Created string  `json:"created_at"`
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

func (c *Client) Chat(messages []Message, options map[string]interface{}) (*Message, error) {
	finalOptions := make(map[string]interface{})
	for k, v := range c.DefaultOptions {
		finalOptions[k] = v
	}
	for k, v := range options {
		finalOptions[k] = v
	}

	reqBody := ChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
		Options:  finalOptions,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *http.Response
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("[Ollama] Attempt %d/3: Sending request to model %s with %d messages...\n", attempt, c.Model, len(messages))

		resp, lastErr = c.Client.Post(c.BaseURL+"/api/chat", "application/json", bytes.NewBuffer(jsonData))

		if lastErr == nil && resp.StatusCode == http.StatusOK {
			// Success
			defer resp.Body.Close()

			var chatResp ChatResponse
			if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("[Ollama] Success! Response received (length: %d chars).\n", len(chatResp.Message.Content))
			return &chatResp.Message, nil
		}

		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %s: %s", resp.Status, string(body))
		}

		fmt.Printf("[Ollama] Attempt %d failed: %v. Retrying in 2s...\n", attempt, lastErr)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed after 3 attempts. Last error: %w", lastErr)
}
