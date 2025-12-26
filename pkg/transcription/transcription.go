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
func Transcribe(serverURL, apiKey, model string, audioData []byte, filename string) (string, error) {
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
	if model == "" {
		model = "whisper-1"
	}
	_ = writer.WriteField("model", model)
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
		if resp.StatusCode == http.StatusNotFound && strings.Contains(string(respBody), "not installed locally") {
			// Trigger download in background
			go func() {
				fmt.Printf("[Transcription] Triggering automatic download for model: %s\n", model)
				if err := EnsureModelDownloaded(serverURL, apiKey, model); err != nil {
					fmt.Printf("[Transcription] Auto-download failed: %v\n", err)
				} else {
					fmt.Printf("[Transcription] Auto-download triggered successfully for: %s\n", model)
				}
			}()
			return "", fmt.Errorf("transcription model '%s' is being downloaded. Please try again in a few moments", model)
		}
		return "", fmt.Errorf("transcription server error: %s - %s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}

// EnsureModelDownloaded triggers a model download on the faster-whisper-server
func EnsureModelDownloaded(serverURL, apiKey, model string) error {
	if serverURL == "" {
		return fmt.Errorf("transcription server URL is empty")
	}

	// Heuristic to build the download URL
	// Format should be: {serverURL}/v1/models/{model}/download
	targetURL := strings.TrimRight(serverURL, "/")
	if !strings.Contains(targetURL, "/v1/models") {
		targetURL += "/v1/models"
	}

	if model == "" {
		model = "whisper-1"
	}

	// Add model and download suffix
	if !strings.HasSuffix(targetURL, "/download") {
		// If the model is already in the URL (unlikely but possible if user provided it), don't duplicate
		if !strings.Contains(targetURL, model) {
			targetURL += "/" + model
		}
		//targetURL += "/download"
	}

	req, err := http.NewRequest("POST", targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// Increase timeout to 10 minutes as requested
	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("download trigger error: %s - %s (URL: %s)", resp.Status, string(respBody), targetURL)
	}

	return nil
}
