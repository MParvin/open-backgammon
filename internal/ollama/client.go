// Package ollama talks to a local Ollama daemon for the AI opponent.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bg/internal/llm"
)

const DefaultBaseURL = "http://localhost:11434"

// Client talks to Ollama's OpenAI-compatible chat API.
type Client struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []llm.Message `json:"messages"`
	Stream   bool          `json:"stream"`
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

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// NormalizeBaseURL adds http:// when the user typed host:port with no scheme.
func NormalizeBaseURL(raw string) string {
	base := strings.TrimSpace(strings.TrimRight(raw, "/"))
	if base == "" {
		return DefaultBaseURL
	}
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	return base
}

func (c *Client) base() string {
	return NormalizeBaseURL(c.BaseURL)
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// Models lists installed model names from GET /api/tags.
func (c *Client) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama server unavailable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var parsed tagsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("ollama returned a malformed response")
	}
	names := make([]string, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		if m.Name != "" {
			names = append(names, m.Name)
		}
	}
	return names, nil
}

// Check verifies the server is reachable and the configured model is installed.
func (c *Client) Check(ctx context.Context) error {
	if c.Model == "" {
		return fmt.Errorf("ollama model is empty")
	}
	names, err := c.Models(ctx)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == c.Model || strings.HasPrefix(n, c.Model+":") {
			return nil
		}
	}
	return fmt.Errorf("ollama model %q is not installed", c.Model)
}

func (c *Client) Complete(ctx context.Context, messages []llm.Message) (string, error) {
	if c.Model == "" {
		return "", fmt.Errorf("ollama model is empty")
	}
	base := c.base()
	body, err := json.Marshal(chatRequest{Model: c.Model, Messages: messages, Stream: false})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("ollama returned a malformed response")
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("ollama: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ollama returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
