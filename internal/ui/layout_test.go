package ui

import (
	"testing"

	"bg/internal/config"
	"bg/internal/game"
)

func TestPointAtOpeningClicks(t *testing.T) {
	l := newLayout()
	cases := []struct {
		x, y float64
		want int
	}{
		{1078, 227, 24},
		{823, 227, 21},
		{90, 227, 13},
		{90, 585, 12},
		{652, 585, 6},
	}
	for _, c := range cases {
		got, ok := l.pointAt(c.x, c.y)
		if !ok || got != c.want {
			t.Errorf("pointAt(%v,%v)=%d,%v want %d", c.x, c.y, got, ok, c.want)
		}
	}
}

func TestClickMovesChecker(t *testing.T) {
	a, err := New(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.dice = [2]int{5, 1}
	r := game.NewRoll(a.board, game.White, 5, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true

	// Left-click uses the larger die (5): 13/8 on the opening position.
	x13, y13 := a.layout.checkerXY(13, 0, 5)
	a.onClick(x13, y13)
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if a.view().Count(game.White, 13) != 4 || a.view().Count(game.White, 8) != 4 {
		t.Fatalf("expected 13/8 with die 5, points=%v steps=%v", a.view().Points, a.roll.Steps)
	}
	if len(a.roll.Steps) != 1 || a.roll.Steps[0].Die != 5 {
		t.Fatalf("steps %v", a.roll.Steps)
	}
}

func TestEmptyPointDoesNotSelect(t *testing.T) {
	a, err := New(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.dice = [2]int{5, 1}
	r := game.NewRoll(a.board, game.White, 5, 1)
	a.roll = &r
	a.rolled = true

	x, y := a.layout.checkerXY(23, 0, 1)
	a.onClick(x, y)
	if a.hasFrom || a.hints(18) || a.hints(22) {
		t.Fatalf("empty point must not select or highlight, has=%v", a.hasFrom)
	}
}

func TestDoubleClickConsumesTwoDice(t *testing.T) {
	a, err := New(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 0
	a.startGame()
	var b game.Board
	b.Points[16] = 1
	b.Off[0] = 14
	a.board = b
	a.dice = [2]int{4, 4}
	r := game.NewRoll(b, game.White, 4, 4)
	a.roll = &r
	a.rolled = true
	a.hasFrom = true
	a.selFrom = 16
	if !a.tryMove(16, 8) {
		t.Fatal("expected two-die move to 8")
	}
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if a.view().Count(game.White, 8) != 1 || a.view().Count(game.White, 16) != 0 {
		t.Fatalf("board after doubles jump: %+v", a.view())
	}
	if len(a.roll.Steps) != 2 {
		t.Fatalf("consumed %d dice", len(a.roll.Steps))
	}
}

func TestResumeKeepsTheGame(t *testing.T) {
	a, err := New(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.dice = [2]int{6, 1}
	r := game.NewRoll(a.board, game.White, 6, 1)
	a.roll = &r
	a.rolled = true
	a.scene = sceneMenu
	resumed := false
	for _, b := range a.buttons() {
		if b.label == "Resume" {
			b.action()
			resumed = true
		}
	}
	if !resumed || a.scene != sceneGame || !a.rolled || a.dice != [2]int{6, 1} {
		t.Fatalf("resume scene=%d rolled=%v dice=%v", a.scene, a.rolled, a.dice)
	}
}
