// Package openrouter talks to OpenRouter chat completions for the AI opponent.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bg/internal/llm"
)

const defaultEndpoint = "https://openrouter.ai/api/v1/chat/completions"

// Client calls the OpenRouter chat completions API.
type Client struct {
	APIKey   string
	Model    string
	Endpoint string
	// ModelsURL overrides the models endpoint used by Check. Tests set this.
	ModelsURL string
	HTTP      *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []llm.Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends a chat request and returns the assistant text.
func (c *Client) Complete(ctx context.Context, messages []llm.Message) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("openrouter API key is empty")
	}
	model := c.Model
	if model == "" {
		model = "openai/gpt-4o-mini"
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	body, err := json.Marshal(chatRequest{Model: model, Messages: messages})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/mparvin/backgammon")
	req.Header.Set("X-Title", "Backgammon")

	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 45 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openrouter status %d", resp.StatusCode)
	}
	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("openrouter: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openrouter returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// Check verifies the API key against the models endpoint.
// Errors never include the key.
func (c *Client) Check(ctx context.Context) error {
	if c.APIKey == "" {
		return fmt.Errorf("openrouter API key is empty")
	}
	endpoint := c.ModelsURL
	if endpoint == "" {
		endpoint = "https://openrouter.ai/api/v1/models"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("openrouter unreachable")
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("openrouter rejected the API key")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("openrouter status %d", resp.StatusCode)
	}
	return nil
}

// ParseDecision is kept for older call sites.
func ParseDecision(text string) (llm.Decision, error) {
	return llm.ParseDecision(text)
}
