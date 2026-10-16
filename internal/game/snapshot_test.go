package game

import "testing"

func TestSnapshotIsolatedFromLaterMutation(t *testing.T) {
	b := NewBoard()
	r := NewRoll(b, White, 6, 1)
	next, ok := r.Apply(r.NextSteps()[0])
	if !ok {
		t.Fatal("apply")
	}
	s := Snapshot{Board: next.Board(), Turn: White, Rolled: true, Cube: NewCube()}
	s.Roll = &next
	s.Cube = s.Cube.Take(Black)
	saved := s.Clone()

	next.Steps[0].Die = 99
	if saved.Roll.Steps[0].Die == 99 {
		t.Fatal("roll steps aliased")
	}
	own := White
	s.Cube.Owner = &own
	if saved.Cube.Owner == nil || *saved.Cube.Owner != Black {
		t.Fatal("cube owner aliased")
	}
}

func TestAllMatchTargetsAndOvershoot(t *testing.T) {
	for _, target := range ValidTargets {
		m := NewMatch(target)
		if m.Target != target || m.GameNum != 1 {
			t.Fatalf("%+v", m)
		}
	}
	m := NewMatch(5)
	m = m.ApplyGame(White, Backgammon, 4) // 12 points, not capped
	if m.White != 12 || !m.Finished() || *m.Winner() != White {
		t.Fatalf("%+v", m)
	}
}

func TestCrawfordThenCubeReturns(t *testing.T) {
	m := NewMatch(3)
	m = m.ApplyGame(White, Single, 1) // 1-0
	m = m.ApplyGame(White, Single, 1) // 2-0, next is Crawford
	if !m.Crawford || m.Finished() {
		t.Fatalf("%+v", m)
	}
	if CanOffer(NewCube(), White, m.Crawford, false, false) {
		t.Fatal("crawford blocks cube")
	}
	m = m.ApplyGame(Black, Single, 1) // 2-1, post-crawford
	if m.Crawford || !m.PostCrawford || m.Finished() {
		t.Fatalf("%+v", m)
	}
	if !CanOffer(NewCube(), White, m.Crawford, false, false) {
		t.Fatal("cube returns after crawford")
	}
}
