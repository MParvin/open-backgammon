// Package stats records finished matches for the Stats screen.
package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"bg/internal/config"
)

// Match is one finished game or a completed match summary.
// The tracked player is White (the human seat against a bot).
type Match struct {
	PlayedAt    time.Time `json:"played_at"`
	Opponent    string    `json:"opponent"`
	Winner      string    `json:"winner"`
	Outcome     string    `json:"outcome,omitempty"`
	Points      int       `json:"points,omitempty"`
	CubeValue   int       `json:"cube_value,omitempty"`
	MatchTarget int       `json:"match_target,omitempty"`
	MatchWhite  int       `json:"match_white,omitempty"`
	MatchBlack  int       `json:"match_black,omitempty"`
	MatchOver   bool      `json:"match_over,omitempty"`
}

// History is the saved match list. It starts empty; nothing is invented.
type History struct {
	Matches []Match `json:"matches"`
}

func Path() (string, error) {
	cfg, err := config.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfg), "history.json"), nil
}

func LoadFile(path string) (History, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return History{}, nil
		}
		return History{}, err
	}
	var h History
	if err := json.Unmarshal(raw, &h); err != nil {
		return History{}, err
	}
	return h, nil
}

func SaveFile(path string, h History) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func Append(m Match) error {
	path, err := Path()
	if err != nil {
		return err
	}
	h, err := LoadFile(path)
	if err != nil {
		return err
	}
	h.Matches = append(h.Matches, m)
	return SaveFile(path, h)
}

func Load() (History, error) {
	path, err := Path()
	if err != nil {
		return History{}, err
	}
	return LoadFile(path)
}

func (h History) scoredEntries() []Match {
	var matchOver []Match
	for _, m := range h.Matches {
		if m.MatchOver {
			matchOver = append(matchOver, m)
		}
	}
	if len(matchOver) > 0 {
		return matchOver
	}
	return h.Matches
}

func (h History) Wins() int {
	n := 0
	for _, m := range h.scoredEntries() {
		if m.Winner == "White" {
			n++
		}
	}
	return n
}

func (h History) Losses() int {
	return len(h.scoredEntries()) - h.Wins()
}

func (h History) WinRate() float64 {
	total := len(h.scoredEntries())
	if total == 0 {
		return 0
	}
	return float64(h.Wins()) / float64(total)
}

func (h History) CumulativeWinRate() []float64 {
	entries := h.scoredEntries()
	out := make([]float64, len(entries))
	wins := 0
	for i, m := range entries {
		if m.Winner == "White" {
			wins++
		}
		out[i] = float64(wins) / float64(i+1)
	}
	return out
}
