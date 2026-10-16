package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bg/internal/game"
	"bg/internal/llm"
)

const maxLLMRetries = 2

// AI asks an LLM Completer to pick a numbered legal play.
// Invalid answers retry once, then fall back to Intermediate.
type AI struct {
	Completer llm.Completer
	Fallback  Player
	Note      string
}

func (a *AI) Name() string { return "AI" }

func (a *AI) Choose(b game.Board, c game.Color, d1, d2 int) (game.Play, error) {
	a.Note = ""
	plays := game.LegalPlays(b, c, d1, d2)
	if len(plays) == 0 {
		return game.Play{}, nil
	}
	if a.Completer == nil {
		return a.fallback(b, c, d1, d2, "AI is not configured")
	}
	msgs := []llm.Message{
		{Role: "system", Content: "You are a backgammon player. Reply with JSON only: {\"play_index\": <int>, \"reason\": \"<short>\"}. play_index must be one of the listed plays."},
		{Role: "user", Content: prompt(b, c, d1, d2, plays)},
	}
	var lastErr string
	for attempt := 0; attempt <= maxLLMRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		text, err := a.Completer.Complete(ctx, msgs)
		cancel()
		if err != nil {
			lastErr = "AI request failed"
			continue
		}
		d, err := llm.ParseDecision(text)
		if err != nil || d.PlayIndex < 0 || d.PlayIndex >= len(plays) {
			lastErr = "AI returned an illegal play"
			msgs = append(msgs,
				llm.Message{Role: "assistant", Content: text},
				llm.Message{Role: "user", Content: fmt.Sprintf("Invalid. play_index must be an integer from 0 to %d inclusive. Reply with JSON only.", len(plays)-1)},
			)
			continue
		}
		a.Note = d.Reason
		return plays[d.PlayIndex], nil
	}
	return a.fallback(b, c, d1, d2, lastErr)
}

func (a *AI) fallback(b game.Board, c game.Color, d1, d2 int, why string) (game.Play, error) {
	fb := a.Fallback
	if fb == nil {
		fb = Intermediate{}
	}
	play, err := fb.Choose(b, c, d1, d2)
	a.Note = why + "; used " + fb.Name()
	return play, err
}

func prompt(b game.Board, c game.Color, d1, d2 int, plays []game.Play) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "You are %s. Dice: %d and %d.\n", c, d1, d2)
	fmt.Fprintf(&sb, "Pips white=%d black=%d. White bar=%d off=%d. Black bar=%d off=%d.\n",
		b.Pips(game.White), b.Pips(game.Black),
		b.OnBar(game.White), b.BorneOff(game.White),
		b.OnBar(game.Black), b.BorneOff(game.Black))
	sb.WriteString("Points (positive white, negative black):\n")
	for p := 1; p <= 24; p++ {
		if n := b.Points[p]; n != 0 {
			fmt.Fprintf(&sb, "  %d: %d\n", p, n)
		}
	}
	sb.WriteString("Legal plays:\n")
	for i, p := range plays {
		fmt.Fprintf(&sb, "%d: %s\n", i, p)
	}
	sb.WriteString("Choose the strongest play_index.")
	return sb.String()
}
