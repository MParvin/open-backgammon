package bot

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"bg/internal/game"
	"bg/internal/llm"
	"bg/internal/openrouter"
)

func TestBotsReturnLegalPlays(t *testing.T) {
	b := game.NewBoard()
	r := rand.New(rand.NewPCG(1, 2))
	players := []Player{Easy{Rand: r}, Intermediate{Rand: r}, Hard{Rand: r}}
	for _, p := range players {
		play, err := p.Choose(b, game.White, 3, 1)
		if err != nil {
			t.Fatalf("%s: %v", p.Name(), err)
		}
		if !isLegal(b, game.White, 3, 1, play) {
			t.Fatalf("%s illegal play %s", p.Name(), play)
		}
	}
}

func TestProfilesAreDistinct(t *testing.T) {
	e, i, h := EasyProfile(), IntermediateProfile(), HardProfile()
	if e.Lookahead || !h.Lookahead {
		t.Fatal("lookahead flags")
	}
	if e.BlunderProb <= i.BlunderProb || i.BlunderProb <= h.BlunderProb {
		t.Fatal("blunder ordering")
	}
	if h.Candidates < i.Candidates {
		t.Fatal("Hard should consider at least as many candidates as Intermediate")
	}
	if e.NoiseStd <= i.NoiseStd || i.NoiseStd <= h.NoiseStd {
		t.Fatal("noise should fall Easy > Intermediate > Hard")
	}
	if e.Name == i.Name || i.Name == h.Name {
		t.Fatal("names")
	}
}

func TestAIUsesIndexedPlay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"play_index\":0,\"reason\":\"test\"}"}}]}`))
	}))
	defer srv.Close()
	ai := &AI{Completer: &openrouter.Client{APIKey: "k", Model: "m", Endpoint: srv.URL, HTTP: srv.Client()}}
	b := game.NewBoard()
	play, err := ai.Choose(b, game.White, 6, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !isLegal(b, game.White, 6, 1, play) {
		t.Fatalf("play %s", play)
	}
	if ai.Note != "test" {
		t.Fatalf("note %q", ai.Note)
	}
}

type badCompleter struct{}

func (badCompleter) Complete(context.Context, []llm.Message) (string, error) {
	return `{"play_index":999,"reason":"bad"}`, nil
}

func TestAIFallsBackOnIllegal(t *testing.T) {
	ai := &AI{Completer: badCompleter{}}
	b := game.NewBoard()
	play, err := ai.Choose(b, game.Black, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !isLegal(b, game.Black, 2, 2, play) {
		t.Fatalf("play %s", play)
	}
	if ai.Note == "" {
		t.Fatal("expected fallback note")
	}
}

func TestAIFallsBackWhenUnconfigured(t *testing.T) {
	ai := &AI{}
	b := game.NewBoard()
	play, err := ai.Choose(b, game.Black, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !isLegal(b, game.Black, 2, 2, play) {
		t.Fatalf("play %s", play)
	}
}

func isLegal(b game.Board, c game.Color, d1, d2 int, play game.Play) bool {
	for _, legal := range game.LegalPlays(b, c, d1, d2) {
		if legal.String() == play.String() {
			return true
		}
	}
	return false
}
