package game

// Outcome is how a game ends for the winner.
type Outcome int

const (
	Single     Outcome = 1
	Gammon     Outcome = 2
	Backgammon Outcome = 3
)

func (o Outcome) String() string {
	switch o {
	case Gammon:
		return "gammon"
	case Backgammon:
		return "backgammon"
	default:
		return "single"
	}
}

// Classify returns the outcome for winner against the finished board.
// A gammon is when the loser has not borne off any checkers.
// A backgammon is a gammon where the loser still has a checker on the bar
// or in the winner's home board.
func Classify(b Board, winner Color) Outcome {
	loser := winner.Opponent()
	if b.BorneOff(loser) > 0 {
		return Single
	}
	if b.OnBar(loser) > 0 {
		return Backgammon
	}
	if winner == White {
		for p := 1; p <= 6; p++ {
			if b.Count(Black, p) > 0 {
				return Backgammon
			}
		}
	} else {
		for p := 19; p <= 24; p++ {
			if b.Count(White, p) > 0 {
				return Backgammon
			}
		}
	}
	return Gammon
}

// Points awards the game value: outcome times cube value.
func Points(outcome Outcome, cubeValue int) int {
	if cubeValue < 1 {
		cubeValue = 1
	}
	return int(outcome) * cubeValue
}
