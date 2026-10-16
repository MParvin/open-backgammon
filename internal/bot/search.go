package bot

import (
	"math"
	"math/rand/v2"

	"bg/internal/game"
)

func mathExp(x float64) float64 { return math.Exp(x) }

// Choose implements Player for a Level profile.
func (l Level) Choose(b game.Board, c game.Color, d1, d2 int) (game.Play, error) {
	plays := game.LegalPlays(b, c, d1, d2)
	if len(plays) == 0 {
		return game.Play{}, nil
	}
	r := l.Rand
	if r == nil {
		r = rand.New(rand.NewPCG(1, 2))
	}
	p := l.Profile
	n := p.Candidates
	if n < 1 {
		n = 1
	}
	cands := scorePlays(b, c, plays, p, r)
	if len(cands) > n {
		cands = cands[:n]
	}
	if p.BlunderProb > 0 && r.Float64() < p.BlunderProb && len(cands) > 1 {
		weak := cands[len(cands)/2:]
		return weak[r.IntN(len(weak))].play, nil
	}
	return pickSoftmax(cands, p.SoftmaxTemp, r), nil
}

type scoredPlay struct {
	play  game.Play
	score float64
}

func scorePlays(b game.Board, c game.Color, plays []game.Play, p Profile, r *rand.Rand) []scoredPlay {
	// Always rank by static eval first; deeper search only touches the strongest few.
	out := make([]scoredPlay, len(plays))
	for i, pl := range plays {
		score := Eval(b.ApplyPlay(c, pl), c)
		if p.NoiseStd > 0 {
			score += r.NormFloat64() * p.NoiseStd
		}
		out[i] = scoredPlay{play: pl, score: score}
	}
	sortScored(out)
	if p.Lookahead {
		opp := c.Opponent()
		limit := p.Candidates
		if limit < 8 {
			limit = 8
		}
		if limit > len(out) {
			limit = len(out)
		}
		for i := 0; i < limit; i++ {
			after := b.ApplyPlay(c, out[i].play)
			out[i].score = expectReply(after, opp, c)
		}
		sortScored(out[:limit])
	}
	return out
}

func sortScored(out []scoredPlay) {
	for i := 0; i < len(out); i++ {
		best := i
		for j := i + 1; j < len(out); j++ {
			if out[j].score > out[best].score {
				best = j
			}
		}
		out[i], out[best] = out[best], out[i]
	}
}

func expectReply(after game.Board, opp, perspective game.Color) float64 {
	if after.Winner() != nil {
		return Eval(after, perspective)
	}
	var total float64
	var w float64
	// Unique dice pairs with multiplicity (21 instead of 36).
	for d1 := 1; d1 <= 6; d1++ {
		for d2 := d1; d2 <= 6; d2++ {
			weight := 2.0
			if d1 == d2 {
				weight = 1
			}
			reply := BestByEval(after, opp, d1, d2)
			total += weight * Eval(after.ApplyPlay(opp, reply), perspective)
			w += weight
		}
	}
	return total / w
}

// BestByEval returns the play that maximizes static Eval for c.
func BestByEval(b game.Board, c game.Color, d1, d2 int) game.Play {
	plays := game.LegalPlays(b, c, d1, d2)
	if len(plays) == 0 {
		return game.Play{}
	}
	best := plays[0]
	bestScore := Eval(b.ApplyPlay(c, best), c)
	for _, p := range plays[1:] {
		score := Eval(b.ApplyPlay(c, p), c)
		if score > bestScore {
			best = p
			bestScore = score
		}
	}
	return best
}

func pickSoftmax(cands []scoredPlay, temp float64, r *rand.Rand) game.Play {
	if len(cands) == 0 {
		return game.Play{}
	}
	if temp <= 0 {
		return cands[0].play
	}
	max := cands[0].score
	var weights []float64
	var sum float64
	for _, c := range cands {
		w := math.Exp((c.score - max) / temp)
		weights = append(weights, w)
		sum += w
	}
	x := r.Float64() * sum
	for i, w := range weights {
		x -= w
		if x <= 0 {
			return cands[i].play
		}
	}
	return cands[len(cands)-1].play
}
