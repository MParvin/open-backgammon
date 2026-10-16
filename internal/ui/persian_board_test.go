package ui

import (
	"testing"

	"bg/internal/config"
	"bg/internal/game"
)

func TestPersianOrnateThemeUsesPlateAndKeepsPlay(t *testing.T) {
	a, err := New(config.Config{Theme: "persian", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	if a.theme.ID != ThemePersian {
		t.Fatal(a.theme.ID)
	}
	if a.layout.pointW < 10 || a.layout.feltW < 400 {
		t.Fatalf("persian layout not applied: %+v", a.layout)
	}
	a.selected = 1
	a.startGame()
	before := a.board
	a.setTheme(ThemeClassic)
	a.setTheme(ThemePersian)
	if a.board != before {
		t.Fatal("theme switch changed board state")
	}
	plate, err := loadPersianPlate()
	if err != nil || plate == nil {
		t.Fatalf("plate: %v", err)
	}
	if plate.Bounds().Dx() < 1000 || plate.Bounds().Dy() < 800 {
		t.Fatalf("plate size %v", plate.Bounds())
	}

	// Hit-test a top-left point should still resolve under the mapped layout.
	x, y := a.layout.checkerXY(13, 0, 1)
	if p, ok := a.layout.pointAt(x, y); !ok || p != 13 {
		t.Fatalf("pointAt(13) got %d ok=%v", p, ok)
	}
	// Opening position still has White's five on 13.
	if a.view().Count(game.White, 13) != 5 {
		t.Fatalf("opening white on 13: %d", a.view().Count(game.White, 13))
	}
}

func TestPersianDiceInLeftWell(t *testing.T) {
	a, err := New(config.Config{Theme: "persian", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.dice = [2]int{6, 3}
	a.showDice = true
	x, y, size, gap := a.dicePlacement()
	if x > a.layout.leftX {
		t.Fatalf("dice should sit in the left well, x=%.0f leftX=%.0f", x, a.layout.leftX)
	}
	if size < 30 || gap < 1 {
		t.Fatalf("dice size %.0f gap %.0f", size, gap)
	}
	rb := a.rollButton()
	if rb.x != x || rb.y != y {
		t.Fatalf("roll button %+v dice %.0f,%.0f", rb, x, y)
	}
}
