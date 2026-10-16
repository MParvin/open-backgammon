package game

import "fmt"

// Step is one checker move using a single die.
type Step struct {
	From int
	To   int
	Die  int
	Hit  bool
}

func (s Step) String() string {
	from := fmt.Sprintf("%d", s.From)
	if s.From == Bar {
		from = "bar"
	}
	to := fmt.Sprintf("%d", s.To)
	if s.To == Off {
		to = "off"
	}
	mark := ""
	if s.Hit {
		mark = "*"
	}
	return from + "/" + to + mark
}

// Play is a complete legal use of a dice roll.
type Play struct {
	Steps []Step
}

func (p Play) String() string {
	if len(p.Steps) == 0 {
		return "(no move)"
	}
	out := p.Steps[0].String()
	for _, s := range p.Steps[1:] {
		out += " " + s.String()
	}
	return out
}

// Apply performs one legal step. The caller must only pass steps from Steps or LegalPlays.
func (b Board) Apply(c Color, s Step) Board {
	if s.From == Bar {
		b.Bar[colorIndex(c)]--
	} else {
		b = b.remove(c, s.From)
	}
	if s.To == Off {
		b.Off[colorIndex(c)]++
		return b
	}
	if s.Hit {
		opp := c.Opponent()
		b = b.remove(opp, s.To)
		b.Bar[colorIndex(opp)]++
	}
	return b.add(c, s.To)
}

func (b Board) ApplyPlay(c Color, p Play) Board {
	for _, s := range p.Steps {
		b = b.Apply(c, s)
	}
	return b
}

// Steps lists individual checker moves that use die, ignoring the
// "must use both dice" rule. That rule is applied by LegalPlays.
func (b Board) Steps(c Color, die int) []Step {
	if die < 1 || die > 6 {
		return nil
	}
	var out []Step
	if b.OnBar(c) > 0 {
		to := entryPoint(c, die)
		if s, ok := b.land(c, Bar, to, die); ok {
			out = append(out, s)
		}
		return out
	}
	for p := 1; p <= 24; p++ {
		if b.Count(c, p) == 0 {
			continue
		}
		to, bear := destination(c, p, die)
		if bear {
			if b.canBear(c, p, die) {
				out = append(out, Step{From: p, To: Off, Die: die})
			}
			continue
		}
		if s, ok := b.land(c, p, to, die); ok {
			out = append(out, s)
		}
	}
	return out
}

func entryPoint(c Color, die int) int {
	if c == White {
		return 25 - die
	}
	return die
}

func destination(c Color, from, die int) (to int, bear bool) {
	if c == White {
		to = from - die
		if to < 1 {
			return Off, true
		}
		return to, false
	}
	to = from + die
	if to > 24 {
		return Off, true
	}
	return to, false
}

func distanceToOff(c Color, point int) int {
	if c == White {
		return point
	}
	return 25 - point
}

func (b Board) canBear(c Color, from, die int) bool {
	if !b.AllInHome(c) {
		return false
	}
	dist := distanceToOff(c, from)
	if die < dist {
		return false
	}
	if die == dist {
		return true
	}
	// A larger die may bear off this checker only when no checker is farther from off.
	if c == White {
		for p := from + 1; p <= 6; p++ {
			if b.Count(White, p) > 0 {
				return false
			}
		}
		return true
	}
	for p := from - 1; p >= 19; p-- {
		if b.Count(Black, p) > 0 {
			return false
		}
	}
	return true
}

func (b Board) land(c Color, from, to, die int) (Step, bool) {
	opp := b.Count(c.Opponent(), to)
	if opp >= 2 {
		return Step{}, false
	}
	return Step{From: from, To: to, Die: die, Hit: opp == 1}, true
}

// LegalPlays returns every maximal legal play for the dice.
// Both numbers must be used when possible. If only one number can be
// played, it must be the larger one. Doubles may be played up to four times.
// An empty play is included when the roll cannot be used.
func LegalPlays(b Board, c Color, d1, d2 int) []Play {
	dice := []int{d1, d2}
	if d1 == d2 {
		dice = []int{d1, d1, d1, d1}
	}
	var found []Play
	var walk func(cur Board, rem []int, steps []Step)
	walk = func(cur Board, rem []int, steps []Step) {
		if len(rem) == 0 {
			found = append(found, Play{cloneSteps(steps)})
			return
		}
		tried := map[int]bool{}
		progress := false
		for i, die := range rem {
			if tried[die] {
				continue
			}
			tried[die] = true
			next := make([]int, 0, len(rem)-1)
			next = append(next, rem[:i]...)
			next = append(next, rem[i+1:]...)
			for _, st := range cur.Steps(c, die) {
				progress = true
				nextSteps := make([]Step, len(steps)+1)
				copy(nextSteps, steps)
				nextSteps[len(steps)] = st
				walk(cur.Apply(c, st), next, nextSteps)
			}
		}
		if !progress {
			found = append(found, Play{cloneSteps(steps)})
		}
	}
	walk(b, dice, nil)

	maxDice := 0
	for _, p := range found {
		if n := len(p.Steps); n > maxDice {
			maxDice = n
		}
	}
	out := make([]Play, 0, len(found))
	for _, p := range found {
		if len(p.Steps) == maxDice {
			out = append(out, p)
		}
	}
	if d1 != d2 && maxDice == 1 {
		higher := d1
		if d2 > higher {
			higher = d2
		}
		canHigher := false
		for _, p := range out {
			if p.Steps[0].Die == higher {
				canHigher = true
				break
			}
		}
		if canHigher {
			filtered := out[:0]
			for _, p := range out {
				if p.Steps[0].Die == higher {
					filtered = append(filtered, p)
				}
			}
			out = filtered
		}
	}
	return dedupePlays(out)
}

func cloneSteps(steps []Step) []Step {
	if len(steps) == 0 {
		return nil
	}
	out := make([]Step, len(steps))
	copy(out, steps)
	return out
}

func dedupePlays(plays []Play) []Play {
	seen := make(map[string]struct{}, len(plays))
	out := make([]Play, 0, len(plays))
	for _, p := range plays {
		key := playKey(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

// CombinedPlays returns one maximal remaining sequence per distinct resulting
// position, for sequences of at least two steps that land on dest.
// Several step orders that reach the same position collapse to one play.
//
// Closing an empty point takes two checkers. Plays that leave more than two
// of our checkers on dest are dropped. When any play leaves exactly two, only
// those are kept. If that still leaves more than one result, the click is
// ambiguous.
func (r Roll) CombinedPlays(dest int) []Play {
	var out []Play
	seen := map[string]struct{}{}
	cur := r.Board()
	for _, p := range r.RemainingSequences() {
		if len(p.Steps) < 2 {
			continue
		}
		if landCount(p, dest) == 0 {
			continue
		}
		key := boardKey(cur.ApplyPlay(r.Color, p))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	if len(out) <= 1 {
		if len(out) == 1 && cur.ApplyPlay(r.Color, out[0]).Count(r.Color, dest) > 2 {
			return nil
		}
		return out
	}
	var made, fewer []Play
	for _, p := range out {
		n := cur.ApplyPlay(r.Color, p).Count(r.Color, dest)
		switch {
		case n == 2:
			made = append(made, p)
		case n < 2:
			fewer = append(fewer, p)
		}
	}
	if len(made) > 0 {
		return made
	}
	return fewer
}

func landCount(p Play, dest int) int {
	n := 0
	for _, s := range p.Steps {
		if s.To == dest {
			n++
		}
	}
	return n
}

func boardKey(b Board) string {
	return fmt.Sprintf("%v|%v|%v", b.Points, b.Bar, b.Off)
}

func playKey(p Play) string {
	if len(p.Steps) == 0 {
		return ""
	}
	out := fmt.Sprintf("%d:%d-%d", p.Steps[0].Die, p.Steps[0].From, p.Steps[0].To)
	for _, s := range p.Steps[1:] {
		out += fmt.Sprintf("|%d:%d-%d", s.Die, s.From, s.To)
	}
	return out
}

// Roll is one player's turn with a fixed dice pair.
type Roll struct {
	Origin Board
	Color  Color
	D1, D2 int
	Steps  []Step
}

func NewRoll(b Board, c Color, d1, d2 int) Roll {
	return Roll{Origin: b, Color: c, D1: d1, D2: d2}
}

func (r Roll) Board() Board {
	return r.Origin.ApplyPlay(r.Color, Play{Steps: r.Steps})
}

// NextSteps are the legal one-step continuations of the moves already taken.
func (r Roll) NextSteps() []Step {
	plays := LegalPlays(r.Origin, r.Color, r.D1, r.D2)
	seen := map[Step]struct{}{}
	var out []Step
	for _, p := range plays {
		if len(p.Steps) <= len(r.Steps) || !hasPrefix(p.Steps, r.Steps) {
			continue
		}
		s := p.Steps[len(r.Steps)]
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// Complete reports whether no further checker may be moved this roll.
func (r Roll) Complete() bool {
	return len(r.NextSteps()) == 0
}

// RemainingSequences are the legal suffixes that continue the steps already taken.
// An empty list means the roll is finished or there was never a legal play.
func (r Roll) RemainingSequences() []Play {
	plays := LegalPlays(r.Origin, r.Color, r.D1, r.D2)
	seen := map[string]struct{}{}
	var out []Play
	for _, p := range plays {
		if len(p.Steps) < len(r.Steps) || !hasPrefix(p.Steps, r.Steps) {
			continue
		}
		rest := cloneSteps(p.Steps[len(r.Steps):])
		key := playKey(Play{Steps: rest})
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Play{Steps: rest})
	}
	return out
}

// ForcedSequence returns the only remaining legal suffix when it is unambiguous.
// A nil play means either nothing remains or the player still has a choice.
func (r Roll) ForcedSequence() (Play, bool) {
	rest := r.RemainingSequences()
	if len(rest) != 1 {
		return Play{}, false
	}
	if len(rest[0].Steps) == 0 {
		return Play{}, false
	}
	return rest[0], true
}

// RemainingDiceValues lists unused die faces for this roll (order preserved).
// Doubles contribute up to four equal faces.
func (r Roll) RemainingDiceValues() []int {
	var pool []int
	if r.D1 == r.D2 {
		pool = []int{r.D1, r.D1, r.D1, r.D1}
	} else {
		pool = []int{r.D1, r.D2}
	}
	used := append([]int{}, pool...)
	for _, s := range r.Steps {
		for i, d := range used {
			if d == s.Die {
				used = append(used[:i], used[i+1:]...)
				break
			}
		}
	}
	return used
}

// LargerRemainingDie is the max unused face, or 0 if none remain.
func (r Roll) LargerRemainingDie() int {
	rem := r.RemainingDiceValues()
	best := 0
	for _, d := range rem {
		if d > best {
			best = d
		}
	}
	return best
}

// SmallerRemainingDie is the min unused face, or 0 if none remain.
func (r Roll) SmallerRemainingDie() int {
	rem := r.RemainingDiceValues()
	if len(rem) == 0 {
		return 0
	}
	best := rem[0]
	for _, d := range rem[1:] {
		if d < best {
			best = d
		}
	}
	return best
}

// Apply appends a step when it is a legal continuation.
func (r Roll) Apply(s Step) (Roll, bool) {
	for _, n := range r.NextSteps() {
		if n == s {
			r.Steps = append(append([]Step{}, r.Steps...), s)
			return r, true
		}
	}
	return r, false
}

func (r Roll) Undo() (Roll, bool) {
	if len(r.Steps) == 0 {
		return r, false
	}
	r.Steps = append([]Step{}, r.Steps[:len(r.Steps)-1]...)
	return r, true
}

// CheckerChains are the consecutive steps one checker can take from from,
// including every legal die order. A later step continues the same checker
// when it starts where the previous step landed.
func (r Roll) CheckerChains(from int) [][]Step {
	plays := LegalPlays(r.Origin, r.Color, r.D1, r.D2)
	seen := map[string]struct{}{}
	var out [][]Step
	for _, p := range plays {
		if len(p.Steps) <= len(r.Steps) || !hasPrefix(p.Steps, r.Steps) {
			continue
		}
		rest := p.Steps[len(r.Steps):]
		if rest[0].From != from {
			continue
		}
		pos := from
		var chain []Step
		for _, s := range rest {
			if s.From != pos {
				break
			}
			chain = append(chain, s)
			pos = s.To
			if s.To == Off {
				break
			}
		}
		if len(chain) == 0 {
			continue
		}
		key := playKey(Play{Steps: chain})
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, chain)
	}
	return out
}

// ShortestChainTo returns the fewest steps that land on to.
// A point two dice away is returned as both steps; a one-die point is a single step.
func ShortestChainTo(chains [][]Step, to int) []Step {
	var best []Step
	for _, chain := range chains {
		for i, s := range chain {
			if s.To != to {
				continue
			}
			prefix := append([]Step{}, chain[:i+1]...)
			if best == nil || len(prefix) < len(best) {
				best = prefix
			}
			break
		}
	}
	return best
}

func hasPrefix(steps, prefix []Step) bool {
	if len(prefix) > len(steps) {
		return false
	}
	for i := range prefix {
		if steps[i] != prefix[i] {
			return false
		}
	}
	return true
}
