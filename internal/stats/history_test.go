package stats

import (
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryStartsEmptyAndPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")
	h, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Matches) != 0 || h.WinRate() != 0 {
		t.Fatalf("empty history %+v", h)
	}
	stamp := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	h.Matches = append(h.Matches,
		Match{PlayedAt: stamp, Opponent: "Easy", Winner: "White"},
		Match{PlayedAt: stamp.Add(time.Hour), Opponent: "Hard", Winner: "Black"},
		Match{PlayedAt: stamp.Add(2 * time.Hour), Opponent: "Easy", Winner: "White"},
	)
	if err := SaveFile(path, h); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Wins() != 2 || got.Losses() != 1 {
		t.Fatalf("wins %d losses %d", got.Wins(), got.Losses())
	}
	if got.WinRate() < 0.66 || got.WinRate() > 0.67 {
		t.Fatalf("rate %v", got.WinRate())
	}
	rates := got.CumulativeWinRate()
	if len(rates) != 3 || rates[0] != 1 || rates[1] != 0.5 || rates[2] < 0.66 {
		t.Fatalf("rates %v", rates)
	}
}
