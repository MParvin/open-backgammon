package game

// Snapshot is an immutable copy of match-play state.
// Slices and cube ownership are cloned so later mutations cannot
// change a stored snapshot.
type Snapshot struct {
	Board         Board
	Turn          Color
	Dice          [2]int
	Rolled        bool
	ShowDice      bool
	Roll          *Roll
	Cube          Cube
	PendingDouble bool
	Doubler       Color
	Match         Match
	Winner        *Color
	GameOver      bool
	MatchOver     bool
	LastOutcome   Outcome
	LastPoints    int
	Status        string
}

// CloneRoll copies steps so the snapshot does not share the live slice.
func CloneRoll(r Roll) Roll {
	r.Steps = append([]Step(nil), r.Steps...)
	return r
}

// CloneCube copies ownership so the snapshot does not share the live pointer.
func CloneCube(c Cube) Cube {
	if c.Owner == nil {
		return c
	}
	own := *c.Owner
	c.Owner = &own
	return c
}

// Clone returns a deep copy of the snapshot.
func (s Snapshot) Clone() Snapshot {
	out := s
	out.Cube = CloneCube(s.Cube)
	if s.Roll != nil {
		r := CloneRoll(*s.Roll)
		out.Roll = &r
	}
	if s.Winner != nil {
		w := *s.Winner
		out.Winner = &w
	}
	return out
}
