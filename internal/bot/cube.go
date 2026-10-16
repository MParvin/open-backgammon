package bot

import "bg/internal/game"

// CubeDecision is a Take/Drop or whether to offer a double.
type CubeDecision int

const (
	CubeHold CubeDecision = iota
	CubeOffer
	CubeTake
	CubeDrop
)

// PointEquity is the expected point swing for c if the game is played out
// at cubeValue with no further doubles. It uses win probability plus a
// separate gammon/backgammon estimate. Checker search does not call this.
func PointEquity(b game.Board, c game.Color, cubeValue int) float64 {
	if cubeValue < 1 {
		cubeValue = 1
	}
	if w := b.Winner(); w != nil {
		pts := float64(game.Points(game.Classify(b, *w), cubeValue))
		if *w == c {
			return pts
		}
		return -pts
	}
	p := WinProb(b, c)
	g, bg := gammonRates(b, c)
	// g is P(gammon or backgammon | win), bg is P(backgammon | win).
	avgWin := (1 - g) + 2*(g-bg) + 3*bg
	return float64(cubeValue) * (p*avgWin - (1 - p))
}

// gammonRates estimates conditional gammon and backgammon rates for c.
// A checker already borne off by the opponent makes both rates zero.
func gammonRates(b game.Board, c game.Color) (gammon, backgammon float64) {
	opp := c.Opponent()
	if b.BorneOff(opp) > 0 {
		return 0, 0
	}
	lead := float64(b.Pips(opp) - b.Pips(c))
	g := 0.12
	if b.BorneOff(c) >= 8 {
		g += 0.25
	} else if b.BorneOff(c) >= 4 {
		g += 0.1
	}
	if lead > 30 {
		g += 0.2
	} else if lead > 12 {
		g += 0.08
	}
	if g > 0.9 {
		g = 0.9
	}
	if g < 0 {
		g = 0
	}
	bg := 0.0
	if b.OnBar(opp) > 0 {
		bg = g * 0.45
	} else if loserInWinnerHome(b, c) {
		bg = g * 0.3
	}
	if bg > g {
		bg = g
	}
	return g, bg
}

func loserInWinnerHome(b game.Board, winner game.Color) bool {
	if winner == game.White {
		for p := 1; p <= 6; p++ {
			if b.Count(game.Black, p) > 0 {
				return true
			}
		}
		return false
	}
	for p := 19; p <= 24; p++ {
		if b.Count(game.White, p) > 0 {
			return true
		}
	}
	return false
}

// ShouldOffer reports whether the side to move should offer a double.
// It compares playing on, an opponent drop, and an opponent take.
// A large gammon equity can make a double incorrect (too good to double).
func ShouldOffer(b game.Board, c game.Color, cube game.Cube, match game.Match, p Profile) bool {
	if !game.CanOffer(cube, c, match.Crawford, false, match.Finished()) {
		return false
	}
	if w := b.Winner(); w != nil {
		return false
	}
	eq := PointEquity(b, c, cube.Value)
	eqDrop := float64(cube.Value)
	eqTake := PointEquity(b, c, cube.OfferedValue())
	if takes(b, c.Opponent(), cube, match, p) {
		// Opponent takes: offer when the doubled game is worth more than playing on.
		margin := 0.15 * float64(cube.Value)
		if match.Score(c) < match.Score(c.Opponent()) {
			margin *= 0.7
		}
		return eqTake > eq+margin && eq > 0
	}
	// Opponent drops. Offer only when the drop beats playing for a gammon.
	return eqDrop > eq+0.05
}

// ShouldTake reports whether the responder should accept the pending double.
func ShouldTake(b game.Board, c game.Color, cube game.Cube, match game.Match, p Profile) bool {
	return takes(b, c, cube, match, p)
}

func takes(b game.Board, responder game.Color, cube game.Cube, match game.Match, p Profile) bool {
	if w := b.Winner(); w != nil {
		return *w == responder
	}
	// Drop loses the current cube. Take plays on at the offered value.
	eqTake := PointEquity(b, responder, cube.OfferedValue())
	eqDrop := -float64(cube.Value)
	// Match score nudges the dead-cube money decision.
	need := match.Target - match.Score(responder)
	oppNeed := match.Target - match.Score(responder.Opponent())
	if need > 0 && oppNeed > 0 && need < oppNeed {
		eqDrop -= 0.15 * float64(cube.Value) // trailing: drop is worse
	}
	if need > 0 && oppNeed > 0 && need > oppNeed && cube.OfferedValue() >= oppNeed {
		eqTake -= 0.1 * float64(cube.Value) // a loss ends the match
	}
	_ = p
	return eqTake >= eqDrop
}
