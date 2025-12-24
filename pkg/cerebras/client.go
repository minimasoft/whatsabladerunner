package cerebras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"whatsabladerunner/pkg/llm"
)

type Client struct {
	APIKey       string
	Model        string
	Client       *http.Client
	BaseURL      string
	ErrorHandler func(error)
}

// NewClient creates a new Cerebras client.
// It reads the API key from the specified file path.
func NewClient(keyFilePath, model string) (*Client, error) {
	apiKey, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read API key from %s: %w", keyFilePath, err)
	}

	key := strings.TrimSpace(string(apiKey))
	if key == "" {
		return nil, fmt.Errorf("API key file %s is empty", keyFilePath)
	}

	if model == "" {
		model = "gpt-oss-120b"
	}

	return &Client{
		APIKey:  key,
		Model:   model,
		Client:  &http.Client{Timeout: 222 * time.Second},
		BaseURL: "https://api.cerebras.ai/v1/chat/completions",
	}, nil
}

// NewClientWithKey creates a new Cerebras client with the provided API key.
func NewClientWithKey(apiKey, model string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is empty")
	}

	if model == "" {
		model = "gpt-oss-120b"
	}

	return &Client{
		APIKey:  apiKey,
		Model:   model,
		Client:  &http.Client{Timeout: 222 * time.Second},
		BaseURL: "https://api.cerebras.ai/v1/chat/completions",
	}, nil
}

// ChatRequest represents the request body for Cerebras API
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []llm.Message `json:"messages"`
	Stream      bool          `json:"stream"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	TopP        float64       `json:"top_p"`
	//ReasoningEffort string        `json:"reasoning_effort"`
}

// ChatResponse represents the response from Cerebras API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat sends a chat request to the Cerebras API
func (c *Client) Chat(messages []llm.Message, options map[string]interface{}) (*llm.Message, error) {
	// Build request with defaults
	reqBody := ChatRequest{
		Model:       c.Model,
		Messages:    messages,
		Stream:      false,
		MaxTokens:   32768,
		Temperature: 0.2,
		TopP:        0.99,
		//		ReasoningEffort: "medium",
	}

	// Allow options to override defaults
	if options != nil {
		if v, ok := options["max_tokens"].(int); ok {
			reqBody.MaxTokens = v
		}
		if v, ok := options["temperature"].(float64); ok {
			reqBody.Temperature = v
		}
		if v, ok := options["top_p"].(float64); ok {
			reqBody.TopP = v
		}
		//if v, ok := options["reasoning_effort"].(string); ok {
		//	reqBody.ReasoningEffort = v
		//}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *http.Response
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("[Cerebras] Attempt %d/3: Sending request to model %s with %d messages...\n", attempt, c.Model, len(messages))

		req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		resp, lastErr = c.Client.Do(req)

		if lastErr == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			var chatResp ChatResponse
			if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}

			if len(chatResp.Choices) == 0 {
				return nil, fmt.Errorf("no choices in response")
			}

			fmt.Printf("[Cerebras] Success! Response received (length: %d chars).\n", len(chatResp.Choices[0].Message.Content))
			res := &llm.Message{
				Role:    chatResp.Choices[0].Message.Role,
				Content: chatResp.Choices[0].Message.Content,
			}
			logTag, _ := options["log_tag"].(string)
			llm.LogLLM("cerebras", logTag, messages, res)
			return res, nil
		}

		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %s: %s", resp.Status, string(body))
		}

		backoff := 0 * time.Second
		switch attempt {
		case 1:
			backoff = 7 * time.Second
		case 2:
			backoff = 29 * time.Second
		}

		if attempt < 3 {
			fmt.Printf("[Cerebras] Attempt %d failed: %v. Retrying in %v...\n", attempt, lastErr, backoff)
			time.Sleep(backoff)
		} else {
			fmt.Printf("[Cerebras] Attempt %d failed: %v.\n", attempt, lastErr)
		}
	}

	finalErr := fmt.Errorf("failed after 3 attempts. Last error: %w", lastErr)
	if c.ErrorHandler != nil {
		c.ErrorHandler(finalErr)
	}
	logTag, _ := options["log_tag"].(string)
	llm.LogLLM("cerebras", logTag, messages, nil)
	return nil, finalErr
}
