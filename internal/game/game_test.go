package game

import "testing"

func TestOpeningCounts(t *testing.T) {
	b := NewBoard()
	if b.Count(White, 24) != 2 || b.Count(White, 13) != 5 || b.Count(White, 8) != 3 || b.Count(White, 6) != 5 {
		t.Fatalf("white opening: %+v", b.Points)
	}
	if b.Count(Black, 1) != 2 || b.Count(Black, 12) != 5 || b.Count(Black, 17) != 3 || b.Count(Black, 19) != 5 {
		t.Fatalf("black opening: %+v", b.Points)
	}
	if b.Pips(White) != 167 || b.Pips(Black) != 167 {
		t.Fatalf("pips white=%d black=%d", b.Pips(White), b.Pips(Black))
	}
}

func TestHitAndEnterFromBar(t *testing.T) {
	var b Board
	b.Points[8] = 1
	b.Points[5] = -1
	b.Off[colorIndex(White)] = 14
	b.Off[colorIndex(Black)] = 14

	plays := LegalPlays(b, White, 3, 1)
	var hit Step
	found := false
	for _, p := range plays {
		for _, s := range p.Steps {
			if s.From == 8 && s.To == 5 && s.Hit {
				hit = s
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected a hit, plays=%v", plays)
	}
	after := b.Apply(White, hit)
	if after.Count(White, 5) != 1 || after.Count(Black, 5) != 0 || after.OnBar(Black) != 1 {
		t.Fatalf("after hit: %+v", after)
	}

	// Black must enter before any other checker moves. Die 3 is blocked.
	// Entering on 1 cannot continue with die 3, because point 4 is blocked.
	var enter Board
	enter.Bar[colorIndex(Black)] = 1
	enter.Off[colorIndex(Black)] = 14
	enter.Points[3] = 2
	enter.Points[4] = 2
	plays = LegalPlays(enter, Black, 1, 3)
	if len(plays) != 1 || len(plays[0].Steps) != 1 || plays[0].Steps[0].From != Bar || plays[0].Steps[0].To != 1 {
		t.Fatalf("enter plays=%v", plays)
	}
	entered := enter.ApplyPlay(Black, plays[0])
	if entered.OnBar(Black) != 0 || entered.Count(Black, 1) != 1 {
		t.Fatalf("entered: %+v", entered)
	}
}

func TestBearingOff(t *testing.T) {
	var b Board
	b.Points[1] = 2
	b.Off[colorIndex(White)] = 13
	plays := LegalPlays(b, White, 2, 1)
	if len(plays) == 0 {
		t.Fatal("no plays")
	}
	for _, p := range plays {
		if len(p.Steps) != 2 {
			t.Fatalf("plays=%v", plays)
		}
		for _, s := range p.Steps {
			if s.To != Off {
				t.Fatalf("expected bear off, got %s", s)
			}
		}
	}
	done := b.ApplyPlay(White, plays[0])
	if done.Winner() == nil || *done.Winner() != White {
		t.Fatal("expected white win")
	}

	var one Board
	one.Points[1] = 1
	one.Off[colorIndex(White)] = 14
	higher := LegalPlays(one, White, 6, 5)
	if len(higher) != 1 || higher[0].Steps[0].Die != 6 || higher[0].Steps[0].To != Off {
		t.Fatalf("must bear off with the larger die, got %v", higher)
	}
}

func TestDoubles(t *testing.T) {
	var b Board
	b.Points[1] = 4
	b.Off[colorIndex(White)] = 11
	plays := LegalPlays(b, White, 1, 1)
	if len(plays) == 0 {
		t.Fatal("no plays")
	}
	for _, p := range plays {
		if len(p.Steps) != 4 {
			t.Fatalf("doubles should use four dice, got %s", p)
		}
		for _, s := range p.Steps {
			if s.Die != 1 || s.To != Off {
				t.Fatalf("unexpected step %s", s)
			}
		}
	}
}

func TestMustUseBothDiceWhenPossible(t *testing.T) {
	var b Board
	b.Points[8] = 1
	b.Points[6] = 1
	b.Off[colorIndex(White)] = 13
	plays := LegalPlays(b, White, 3, 1)
	if len(plays) == 0 {
		t.Fatal("no plays")
	}
	for _, p := range plays {
		if len(p.Steps) != 2 {
			t.Fatalf("must use both dice, got %s", p)
		}
	}
}

func TestMustPlayLargerDie(t *testing.T) {
	var b Board
	b.Bar[colorIndex(White)] = 1
	b.Off[colorIndex(White)] = 14
	// Either die can enter, but the second die is blocked on point 14,
	// so only one number can be played and it must be the larger.
	b.Points[14] = -2
	plays := LegalPlays(b, White, 5, 6)
	if len(plays) != 1 || len(plays[0].Steps) != 1 || plays[0].Steps[0].Die != 6 || plays[0].Steps[0].To != 19 {
		t.Fatalf("plays=%v", plays)
	}

	// Larger entry is blocked, so the smaller die is allowed.
	b.Points[19] = -2
	smaller := LegalPlays(b, White, 5, 6)
	if len(smaller) != 1 || smaller[0].Steps[0].Die != 5 || smaller[0].Steps[0].To != 20 {
		t.Fatalf("smaller=%v", smaller)
	}
}

func TestRollContinuation(t *testing.T) {
	b := NewBoard()
	r := NewRoll(b, White, 3, 1)
	next := r.NextSteps()
	if len(next) == 0 {
		t.Fatal("opening 3-1 should have moves")
	}
	r, ok := r.Apply(next[0])
	if !ok {
		t.Fatal("apply")
	}
	if r.Complete() {
		t.Fatal("3-1 from the opening should have a second move")
	}
	r, ok = r.Undo()
	if !ok || len(r.Steps) != 0 {
		t.Fatal("undo")
	}
}

func TestChainUsesSmallerDieWhenLargerIsBlocked(t *testing.T) {
	var b Board
	b.Points[13] = 1
	b.Off[colorIndex(White)] = 14
	b.Points[7] = -2 // 13-6 is blocked, so 6 then 3 cannot start
	r := NewRoll(b, White, 6, 3)
	chains := r.CheckerChains(13)
	dest := map[int]bool{}
	for _, chain := range chains {
		if chain[0].To == 7 {
			t.Fatalf("blocked point offered: %v", chain)
		}
		for _, s := range chain {
			dest[s.To] = true
		}
	}
	if !dest[10] || !dest[4] {
		t.Fatalf("want 3-then-6 through 10 and 4, got %v", chains)
	}
	far := ShortestChainTo(chains, 4)
	if len(far) != 2 || far[0].To != 10 || far[1].To != 4 {
		t.Fatalf("far chain %v", far)
	}
	near := ShortestChainTo(chains, 10)
	if len(near) != 1 {
		t.Fatalf("one-die chain %v", near)
	}
}

func TestDoublesChainToFarPoint(t *testing.T) {
	var b Board
	b.Points[16] = 1
	b.Off[colorIndex(White)] = 14
	r := NewRoll(b, White, 4, 4)
	chains := r.CheckerChains(16)
	far := ShortestChainTo(chains, 8)
	if len(far) != 2 || far[0].Die != 4 || far[1].To != 8 {
		t.Fatalf("two dice to 8: %v", far)
	}
	if len(ShortestChainTo(chains, 12)) != 1 {
		t.Fatal("point 12 is one die")
	}
}

func TestForcedSequence(t *testing.T) {
	var b Board
	b.Bar[colorIndex(White)] = 1
	b.Off[colorIndex(White)] = 14
	b.Points[19] = -2 // die 6 entry blocked
	r := NewRoll(b, White, 5, 6)
	forced, ok := r.ForcedSequence()
	// Only one complete play: enter with 5 then play 6 from 20.
	if !ok || len(forced.Steps) != 2 ||
		forced.Steps[0].Die != 5 || forced.Steps[0].To != 20 ||
		forced.Steps[1].Die != 6 || forced.Steps[1].To != 14 {
		t.Fatalf("forced=%v ok=%v", forced, ok)
	}
	// Two legal entry options: not forced.
	open := NewRoll(b, White, 5, 4)
	if _, ok := open.ForcedSequence(); ok {
		t.Fatal("two entries should not force")
	}
}

func TestForcedSequenceSingleDie(t *testing.T) {
	// Both dice blocked except one entry die that leaves no further play.
	var b Board
	b.Bar[colorIndex(White)] = 1
	b.Off[colorIndex(White)] = 14
	for p := 1; p <= 24; p++ {
		b.Points[p] = -2
	}
	b.Points[20] = 0 // only 5-entry open; 6-entry (19) still blocked
	r := NewRoll(b, White, 5, 6)
	forced, ok := r.ForcedSequence()
	if !ok || len(forced.Steps) != 1 || forced.Steps[0].Die != 5 || forced.Steps[0].To != 20 {
		t.Fatalf("forced=%v ok=%v", forced, ok)
	}
}

func TestForcedBearOff(t *testing.T) {
	// Single checker on the 1-point: either die bears off, so the higher die must be used.
	var b Board
	b.Points[1] = 1
	b.Off[colorIndex(White)] = 14
	r := NewRoll(b, White, 2, 3)
	forced, ok := r.ForcedSequence()
	if !ok || len(forced.Steps) != 1 || forced.Steps[0].To != Off || forced.Steps[0].Die != 3 {
		t.Fatalf("forced bear-off=%v ok=%v", forced, ok)
	}
}

func TestForcedHit(t *testing.T) {
	var b Board
	b.Points[13] = 1
	b.Points[10] = -1
	b.Off[colorIndex(White)] = 14
	b.Off[colorIndex(Black)] = 14
	// Die 6 cannot start from 13 (7 blocked); only 13/10 with 3 then 10/4 with 6.
	b.Points[7] = -2
	r := NewRoll(b, White, 3, 6)
	forced, ok := r.ForcedSequence()
	if !ok || len(forced.Steps) != 2 ||
		forced.Steps[0].To != 10 || forced.Steps[0].Die != 3 ||
		forced.Steps[1].To != 4 || forced.Steps[1].Die != 6 {
		t.Fatalf("forced hit=%v ok=%v", forced, ok)
	}
	after := b.ApplyPlay(White, forced)
	if after.OnBar(Black) != 1 {
		t.Fatal("hit should send blot to bar")
	}
}

func TestForcedDoubles(t *testing.T) {
	// Only 8→7 is open; further 1-pip landings are blocked, so one die of the double is forced.
	var b Board
	b.Points[8] = 1
	b.Off[colorIndex(White)] = 14
	for _, p := range []int{6, 5, 4, 3, 2, 1} {
		b.Points[p] = -2
	}
	r := NewRoll(b, White, 1, 1)
	forced, ok := r.ForcedSequence()
	if !ok || len(forced.Steps) != 1 || forced.Steps[0].From != 8 || forced.Steps[0].To != 7 {
		t.Fatalf("one double step forced=%v ok=%v", forced, ok)
	}
}

func TestRemainingDiceValues(t *testing.T) {
	b := NewBoard()
	r := NewRoll(b, White, 6, 1)
	if got := r.RemainingDiceValues(); len(got) != 2 || got[0] != 6 || got[1] != 1 {
		t.Fatalf("%v", got)
	}
	steps := r.NextSteps()
	if len(steps) == 0 {
		t.Fatal("no steps")
	}
	next, ok := r.Apply(steps[0])
	if !ok {
		t.Fatal("apply")
	}
	rem := next.RemainingDiceValues()
	if len(rem) != 1 {
		t.Fatalf("after one die: %v", rem)
	}
	if next.LargerRemainingDie() != rem[0] || next.SmallerRemainingDie() != rem[0] {
		t.Fatal("single remaining")
	}
}

func TestBlockedBarEntryPasses(t *testing.T) {
	for _, c := range []Color{White, Black} {
		var b Board
		b.Bar[colorIndex(c)] = 1
		b.Off[colorIndex(c)] = 14
		opp := c.Opponent()
		for p := 1; p <= 24; p++ {
			if opp == White {
				b.Points[p] = 2
			} else {
				b.Points[p] = -2
			}
		}
		plays := LegalPlays(b, c, 3, 4)
		if len(plays) != 1 || len(plays[0].Steps) != 0 {
			t.Fatalf("%s blocked entry plays=%v", c, plays)
		}
		r := NewRoll(b, c, 3, 4)
		if !r.Complete() {
			t.Fatalf("%s should have no legal continuation", c)
		}
		if _, ok := r.ForcedSequence(); ok {
			t.Fatal("a pass is not a forced step sequence")
		}
		doubles := LegalPlays(b, c, 2, 2)
		if len(doubles) != 1 || len(doubles[0].Steps) != 0 {
			t.Fatalf("%s doubles %v", c, doubles)
		}
	}
}

func TestPartialBarEntryAndHit(t *testing.T) {
	var b Board
	b.Bar[colorIndex(White)] = 1
	b.Off[colorIndex(White)] = 14
	for p := 1; p <= 24; p++ {
		b.Points[p] = -2
	}
	b.Points[20] = -1 // blot: die 5 hits
	b.Points[14] = -2
	plays := LegalPlays(b, White, 5, 6)
	if len(plays) != 1 || len(plays[0].Steps) != 1 || !plays[0].Steps[0].Hit || plays[0].Steps[0].To != 20 {
		t.Fatalf("only the hitting entry %v", plays)
	}
	after := b.ApplyPlay(White, plays[0])
	if after.OnBar(White) != 0 || after.OnBar(Black) != 1 || after.Count(White, 20) != 1 {
		t.Fatalf("after hit entry %+v", after)
	}
}

func TestCombinedDestination(t *testing.T) {
	// Black moves upward: 5/9 with 4 and 2/5 with 3.
	var b Board
	b.Points[2] = -1
	b.Points[5] = -1
	b.Off[colorIndex(Black)] = 13
	// Other routes (via 6, 8, or 12) are blocked, leaving one result.
	b.Points[6] = 2
	b.Points[8] = 2
	b.Points[12] = 2
	r := NewRoll(b, Black, 3, 4)
	got := r.CombinedPlays(9)
	if len(got) != 1 || len(got[0].Steps) != 2 {
		t.Fatalf("plays %v", got)
	}
	after := b.ApplyPlay(Black, got[0])
	if after.Count(Black, 9) != 1 || after.Count(Black, 5) != 1 || after.Count(Black, 2) != 0 {
		t.Fatalf("after %+v steps %v", after.Points, got[0])
	}
	open := NewRoll(Board{Points: [26]int8{2: -1, 5: -1}, Off: [2]int8{1: 13}}, Black, 3, 4)
	if len(open.CombinedPlays(9)) < 2 {
		t.Fatalf("unblocked 2 and 5 should be ambiguous: %v", open.CombinedPlays(9))
	}
	if len(r.CombinedPlays(20)) != 0 {
		t.Fatal("20 is not a combined destination")
	}
}

func TestOpening61Click7MakesThePoint(t *testing.T) {
	b := NewBoard()
	r := NewRoll(b, White, 6, 1)
	got := r.CombinedPlays(7)
	if len(got) != 1 {
		t.Fatalf("opening 6-1 click 7 should uniquely make the point, got %v legal=%v", got, LegalPlays(b, White, 6, 1))
	}
	n := 0
	from8, from13 := false, false
	for _, s := range got[0].Steps {
		if s.To == 7 {
			n++
		}
		if s.From == 8 && s.To == 7 {
			from8 = true
		}
		if s.From == 13 && s.To == 7 {
			from13 = true
		}
	}
	if n != 2 || !from8 || !from13 {
		t.Fatalf("want 8/7 and 13/7, got %v", got[0])
	}
	after := b.ApplyPlay(White, got[0])
	if after.Count(White, 7) != 2 {
		t.Fatalf("point 7: %+v", after.Points)
	}
}

func TestMakePointPrefersBothDiceOnDest(t *testing.T) {
	// White 6-1: several plays touch 7, but only 8/7 + 13/7 puts two checkers there.
	var b Board
	b.Points[13] = 2
	b.Points[8] = 2
	b.Points[6] = 2
	b.Off[colorIndex(White)] = 9
	r := NewRoll(b, White, 6, 1)
	got := r.CombinedPlays(7)
	if len(got) != 1 {
		t.Fatalf("want unique make-point play, got %v", got)
	}
	n := 0
	for _, s := range got[0].Steps {
		if s.To == 7 {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("both dice should land on 7: %v", got[0])
	}
	after := b.ApplyPlay(White, got[0])
	if after.Count(White, 7) != 2 {
		t.Fatalf("point 7 should be made: %+v", after.Points)
	}
}

func TestMakePointDoesNotUseMoreThanTwo(t *testing.T) {
	// Three checkers on 8 and a blocked 6: double-1s can only stack 8/7 three
	// times. Closing a point needs two, so this must not auto-play.
	var stacked Board
	stacked.Points[8] = 3
	stacked.Points[6] = -2
	stacked.Off[colorIndex(White)] = 10
	stacked.Off[colorIndex(Black)] = 13
	over := NewRoll(stacked, White, 1, 1)
	if got := over.CombinedPlays(7); len(got) != 0 {
		t.Fatalf("must not auto-stack more than two on 7, got %v", got)
	}

	// Two on 8 and one on 10: making 7 with exactly two checkers is preferred
	// over any line that leaves a different count.
	var b Board
	b.Points[8] = 2
	b.Points[10] = 1
	b.Off[colorIndex(White)] = 12
	r := NewRoll(b, White, 1, 1)
	got := r.CombinedPlays(7)
	if len(got) == 0 {
		t.Fatal("expected a two-checker close of point 7")
	}
	for _, p := range got {
		if n := b.ApplyPlay(White, p).Count(White, 7); n != 2 {
			t.Fatalf("play leaves %d on 7, want 2: %v", n, p)
		}
	}
}

func TestHomeBoardPips(t *testing.T) {
	var white Board
	white.Points[6] = 1
	white.Points[4] = 1
	white.Off[colorIndex(White)] = 13
	if white.Pips(White) != 10 {
		t.Fatalf("white pips %d", white.Pips(White))
	}
	var ones Board
	ones.Points[1] = 2
	ones.Off[colorIndex(White)] = 13
	if ones.Pips(White) != 2 {
		t.Fatalf("two on 1: %d", ones.Pips(White))
	}
	var done Board
	done.Off[colorIndex(White)] = 15
	if done.Pips(White) != 0 {
		t.Fatalf("borne off %d", done.Pips(White))
	}
	var black Board
	black.Points[19] = -1 // 6 from off
	black.Points[21] = -1 // 4 from off
	black.Off[colorIndex(Black)] = 13
	if black.Pips(Black) != 10 {
		t.Fatalf("black pips %d", black.Pips(Black))
	}
}
