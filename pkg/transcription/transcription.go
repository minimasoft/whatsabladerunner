package transcription

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Transcribe sends audio data to an OpenAI-compatible transcription server
func Transcribe(serverURL, apiKey string, audioData []byte, filename string) (string, error) {
	if serverURL == "" {
		return "", fmt.Errorf("transcription server URL is empty")
	}

	// Heuristic to append endpoint if user provided base URL
	targetURL := serverURL
	if !strings.Contains(targetURL, "transcriptions") {
		targetURL = strings.TrimRight(targetURL, "/")
		if !strings.Contains(targetURL, "/v1/audio") {
			targetURL += "/v1/audio/transcriptions"
		} else {
			targetURL += "/transcriptions"
		}
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = part.Write(audioData)
	if err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	// Model is required by OpenAI API spec, usually 'whisper-1' is safe default compat
	_ = writer.WriteField("model", "whisper-1")
	// Request text format for simple string response
	_ = writer.WriteField("response_format", "text")

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest("POST", targetURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 120 * time.Second} // Allow generous timeout for long audio
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("transcription server error: %s - %s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}
