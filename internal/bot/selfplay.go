package bot

import (
	"math"
	"math/rand/v2"
	"time"

	"bg/internal/game"
)

// MatchResult is one self-play game between two players.
type MatchResult struct {
	Winner         game.Color
	Plies          int
	WhiteMs        float64
	BlackMs        float64
	WhiteDecisions int
	BlackDecisions int
}

// PlayGame plays a single game with deterministic dice from rng.
// White is player W, Black is player B. No cube.
func PlayGame(w, b Player, rng *rand.Rand) MatchResult {
	board := game.NewBoard()
	turn := game.White
	var res MatchResult
	for {
		if win := board.Winner(); win != nil {
			res.Winner = *win
			return res
		}
		d1 := rng.IntN(6) + 1
		d2 := rng.IntN(6) + 1
		player := w
		if turn == game.Black {
			player = b
		}
		start := time.Now()
		play, err := player.Choose(board, turn, d1, d2)
		elapsed := time.Since(start).Seconds() * 1000
		if err != nil {
			play = BestByEval(board, turn, d1, d2)
		}
		if turn == game.White {
			res.WhiteMs += elapsed
			res.WhiteDecisions++
		} else {
			res.BlackMs += elapsed
			res.BlackDecisions++
		}
		board = board.ApplyPlay(turn, play)
		res.Plies++
		if res.Plies > 800 {
			// Force a finish by pip count if something stalled.
			if board.Pips(game.White) <= board.Pips(game.Black) {
				res.Winner = game.White
			} else {
				res.Winner = game.Black
			}
			return res
		}
		turn = turn.Opponent()
	}
}

// SeriesStats summarizes a batch of games.
type SeriesStats struct {
	Games        int
	FirstWins    int // wins for the first player argument
	SecondWins   int
	FirstWinRate float64
	CILow        float64
	CIHigh       float64
	AvgFirstMs   float64
	AvgSecondMs  float64
}

// RunSeries plays n games, alternating colors each game.
// odd games: a is White; even games: b is White.
func RunSeries(a, b Player, n int, seed uint64) SeriesStats {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var s SeriesStats
	s.Games = n
	var firstMs, secondMs float64
	var firstDec, secondDec int
	for i := 0; i < n; i++ {
		var res MatchResult
		aIsWhite := i%2 == 0
		if aIsWhite {
			res = PlayGame(a, b, rng)
			firstMs += res.WhiteMs
			secondMs += res.BlackMs
			firstDec += res.WhiteDecisions
			secondDec += res.BlackDecisions
			if res.Winner == game.White {
				s.FirstWins++
			} else {
				s.SecondWins++
			}
		} else {
			res = PlayGame(b, a, rng)
			firstMs += res.BlackMs
			secondMs += res.WhiteMs
			firstDec += res.BlackDecisions
			secondDec += res.WhiteDecisions
			if res.Winner == game.Black {
				s.FirstWins++
			} else {
				s.SecondWins++
			}
		}
	}
	s.FirstWinRate = float64(s.FirstWins) / float64(n)
	s.CILow, s.CIHigh = wilson(s.FirstWins, n)
	if firstDec > 0 {
		s.AvgFirstMs = firstMs / float64(firstDec)
	}
	if secondDec > 0 {
		s.AvgSecondMs = secondMs / float64(secondDec)
	}
	return s
}

func wilson(wins, n int) (low, high float64) {
	if n == 0 {
		return 0, 0
	}
	z := 1.96
	p := float64(wins) / float64(n)
	zn := z * z / float64(n)
	center := p + zn/2
	margin := z * math.Sqrt(p*(1-p)/float64(n)+zn*zn/4)
	den := 1 + zn
	low = (center - margin) / den
	high = (center + margin) / den
	if low < 0 {
		low = 0
	}
	if high > 1 {
		high = 1
	}
	return low, high
}
