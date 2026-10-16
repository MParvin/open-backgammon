package bot

import (
	"math/rand/v2"
	"testing"
)

func TestDifficultyOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("self-play benchmark")
	}
	easy := Easy{Rand: rand.New(rand.NewPCG(42, 7))}
	mid := Intermediate{Rand: rand.New(rand.NewPCG(42, 8))}
	hard := Hard{Rand: rand.New(rand.NewPCG(42, 9))}

	const n = 24
	ei := RunSeries(easy, mid, n, 1001)
	t.Logf("Easy vs Intermediate: easy_wins=%d/%d rate=%.2f CI[%.2f,%.2f] avg_ms easy=%.1f mid=%.1f",
		ei.FirstWins, ei.Games, ei.FirstWinRate, ei.CILow, ei.CIHigh, ei.AvgFirstMs, ei.AvgSecondMs)
	if ei.FirstWinRate > 0.58 {
		t.Errorf("Easy should not outplay Intermediate (rate=%.2f)", ei.FirstWinRate)
	}

	ih := RunSeries(mid, hard, n, 2002)
	t.Logf("Intermediate vs Hard: mid_wins=%d/%d rate=%.2f CI[%.2f,%.2f] avg_ms mid=%.1f hard=%.1f",
		ih.FirstWins, ih.Games, ih.FirstWinRate, ih.CILow, ih.CIHigh, ih.AvgFirstMs, ih.AvgSecondMs)
	if ih.FirstWinRate > 0.62 {
		t.Errorf("Intermediate should not clearly outplay Hard (rate=%.2f)", ih.FirstWinRate)
	}

	eh := RunSeries(easy, hard, n, 3003)
	t.Logf("Easy vs Hard: easy_wins=%d/%d rate=%.2f CI[%.2f,%.2f] avg_ms easy=%.1f hard=%.1f",
		eh.FirstWins, eh.Games, eh.FirstWinRate, eh.CILow, eh.CIHigh, eh.AvgFirstMs, eh.AvgSecondMs)
	if eh.FirstWinRate > 0.48 {
		t.Errorf("Easy should lose more often to Hard (rate=%.2f)", eh.FirstWinRate)
	}
}

func TestProfilesUsedByLevels(t *testing.T) {
	e, i, h := (Easy{}).Profile(), (Intermediate{}).Profile(), (Hard{}).Profile()
	if e.Name != "Easy" || i.Name != "Intermediate" || h.Name != "Hard" {
		t.Fatal("profile names")
	}
	if !h.Lookahead || e.Lookahead {
		t.Fatal("lookahead assignment")
	}
}
