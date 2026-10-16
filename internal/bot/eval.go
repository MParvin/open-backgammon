package bot

import "bg/internal/game"

// Eval scores a position for perspective. Terminal wins are large positives.
func Eval(b game.Board, perspective game.Color) float64 {
	if w := b.Winner(); w != nil {
		outcome := game.Classify(b, *w)
		pts := float64(game.Points(outcome, 1))
		if *w == perspective {
			return 100000 * pts
		}
		return -100000 * pts
	}
	return static(b, perspective) - static(b, perspective.Opponent())
}

func static(b game.Board, c game.Color) float64 {
	s := -float64(b.Pips(c)) * 1.15
	s += float64(b.BorneOff(c)) * 14
	s -= float64(b.OnBar(c)) * 32
	opp := c.Opponent()
	var run int
	anchors := 0
	home := 0
	for p := 1; p <= 24; p++ {
		n := b.Count(c, p)
		switch {
		case n == 1:
			s -= 7
			if canBeHit(b, opp, p) {
				s -= 9
			}
			run = 0
		case n >= 2:
			s += 4
			run++
			if run >= 2 {
				s += float64(run) * 1.5 // primes
			}
			if inHome(c, p) {
				anchors++
				home += n
			}
		default:
			run = 0
		}
	}
	s += float64(anchors) * 3
	s += float64(home) * 0.5
	if b.AllInHome(c) {
		s += 8
	}
	return s
}

func inHome(c game.Color, p int) bool {
	if c == game.White {
		return p >= 1 && p <= 6
	}
	return p >= 19 && p <= 24
}

func canBeHit(b game.Board, attacker game.Color, point int) bool {
	if b.OnBar(attacker) > 0 {
		for die := 1; die <= 6; die++ {
			var entry int
			if attacker == game.White {
				entry = 25 - die
			} else {
				entry = die
			}
			if entry == point {
				return true
			}
		}
		return false
	}
	for from := 1; from <= 24; from++ {
		if b.Count(attacker, from) == 0 {
			continue
		}
		dist := point - from
		if attacker == game.White {
			dist = from - point
		}
		if dist >= 1 && dist <= 6 {
			return true
		}
	}
	return false
}

// WinProb is a crude [0,1] estimate from Eval for cube decisions.
func WinProb(b game.Board, c game.Color) float64 {
	if w := b.Winner(); w != nil {
		if *w == c {
			return 1
		}
		return 0
	}
	e := Eval(b, c)
	// Squash: ±80 maps near 0/1.
	x := e / 80
	if x > 8 {
		x = 8
	}
	if x < -8 {
		x = -8
	}
	return 1 / (1 + expNeg(x))
}

func expNeg(x float64) float64 {
	// e^{-x} without importing math for a tiny helper; use math.Exp below.
	return mathExp(-x)
}
