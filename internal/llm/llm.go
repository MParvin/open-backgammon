// Package llm defines the chat completer used by the AI opponent.
package llm

import (
	"context"
	"encoding/json"
	"strings"
)

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Completer is a chat completion backend (OpenRouter, Ollama, ...).
type Completer interface {
	Complete(ctx context.Context, messages []Message) (string, error)
}

// Decision is the structured move selection from a model.
type Decision struct {
	PlayIndex int    `json:"play_index"`
	Reason    string `json:"reason"`
}

// ParseDecision extracts {"play_index": N, "reason": "..."} from model text.
func ParseDecision(text string) (Decision, error) {
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			text = text[i : j+1]
		}
	}
	var d Decision
	if err := json.Unmarshal([]byte(text), &d); err != nil {
		return Decision{}, err
	}
	return d, nil
}
