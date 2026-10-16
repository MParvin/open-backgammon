package game

// ValidTargets are the supported match lengths.
var ValidTargets = []int{3, 5, 7, 9, 11, 13, 15, 17, 19, 21}

func ValidTarget(n int) bool {
	for _, t := range ValidTargets {
		if t == n {
			return true
		}
	}
	return false
}

// Match tracks match score and Crawford state.
type Match struct {
	Target   int
	White    int
	Black    int
	GameNum  int
	Crawford bool
	// PostCrawford is true after the Crawford game has been played.
	PostCrawford bool
}

func NewMatch(target int) Match {
	if !ValidTarget(target) {
		target = 5
	}
	return Match{Target: target, GameNum: 1}
}

func (m Match) Score(c Color) int {
	if c == White {
		return m.White
	}
	return m.Black
}

func (m Match) Winner() *Color {
	if m.White >= m.Target {
		c := White
		return &c
	}
	if m.Black >= m.Target {
		c := Black
		return &c
	}
	return nil
}

func (m Match) Finished() bool {
	return m.Winner() != nil
}

// ApplyGame awards points for a finished game and advances Crawford/match state.
// Points are not capped beyond reaching the target; the match ends when either
// side first reaches Target.
func (m Match) ApplyGame(winner Color, outcome Outcome, cubeValue int) Match {
	pts := Points(outcome, cubeValue)
	if winner == White {
		m.White += pts
	} else {
		m.Black += pts
	}
	if m.Crawford {
		m.Crawford = false
		m.PostCrawford = true
	}
	if m.Finished() {
		return m
	}
	// Crawford: a player reaches match-1, then the next game is Crawford.
	if !m.PostCrawford && !m.Crawford {
		if m.White == m.Target-1 || m.Black == m.Target-1 {
			m.Crawford = true
		}
	}
	m.GameNum++
	return m
}
