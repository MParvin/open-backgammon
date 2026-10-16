package ui

import (
	"image/color"
	"testing"

	"bg/internal/config"
	"bg/internal/game"
)

func TestPersianOrnateThemeUsesPlateAndKeepsPlay(t *testing.T) {
	a, err := New(config.Config{Theme: "persian", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	if a.theme.ID != ThemePersian || a.kit == nil {
		t.Fatalf("theme %s kit=%v", a.theme.ID, a.kit != nil)
	}
	if a.layout.pointW < 10 || a.layout.feltW < 400 {
		t.Fatalf("persian layout not applied: %+v", a.layout)
	}
	a.selected = 1
	a.startGame()
	before := a.board
	a.setTheme(ThemeClassic)
	if a.kit != nil {
		t.Fatal("classic theme must not use a kit")
	}
	a.setTheme(ThemePersian)
	if a.board != before {
		t.Fatal("theme switch changed board state")
	}
	if a.kit.plate.Bounds().Dx() != screenW || a.kit.plate.Bounds().Dy() != screenH {
		t.Fatalf("plate size %v", a.kit.plate.Bounds())
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
	x, _, size, gap := a.dicePlacement()
	if x > a.layout.leftX {
		t.Fatalf("dice should sit in the left well, x=%.0f leftX=%.0f", x, a.layout.leftX)
	}
	if size < 30 || gap < 1 {
		t.Fatalf("dice size %.0f gap %.0f", size, gap)
	}
}

func TestLeftClickBoardRollsDice(t *testing.T) {
	a := newHumanGame(t)
	if a.rolled || !a.canRoll() {
		t.Fatalf("expected unrolled human turn, rolled=%v", a.rolled)
	}
	for _, b := range a.buttons() {
		if b.label == "Roll" {
			t.Fatal("Roll button must not appear")
		}
	}
	cx := a.layout.feltX + a.layout.feltW/2
	cy := a.layout.feltY + a.layout.feltH/2
	a.onClick(cx, cy)
	if !a.rolled || a.roll == nil || a.dice[0] == 0 {
		t.Fatalf("board click should roll, rolled=%v dice=%v", a.rolled, a.dice)
	}
}

// Both plate kits must compose: the fields are rebuilt from the artwork, so
// a pixel in the middle of a point must be an inlay colour, not bare wood,
// and the checker sprites must be opaque discs.
func TestPlateKitsCompose(t *testing.T) {
	for _, id := range []ThemeID{ThemePersian, ThemeChinese} {
		k := kitFor(id)
		if k == nil {
			t.Fatalf("%s kit failed to build", id)
		}
		l := k.layout()
		if l.pointW < 40 || l.feltH < 300 || l.r < 15 {
			t.Fatalf("%s layout %+v", id, l)
		}
		// Centre of the first top-left point, a third of the way down.
		x := int(l.colX(0) + l.pointW/2)
		y := int(l.feltY + k.geom.pointLen*0.3)
		c := k.plate.RGBAAt(x, y)
		// Wood sample from the middle of the left field, away from the medallion.
		w := k.plate.RGBAAt(int(l.leftX+8), int(l.feltY+l.feltH/2))
		if colorDist(c, w) < 30 {
			t.Fatalf("%s: point pixel %v too close to wood %v — inlay missing", id, c, w)
		}
		for i, spr := range k.checker {
			if spr == nil {
				t.Fatalf("%s checker %d missing", id, i)
			}
			b := spr.Bounds()
			if spr.RGBAAt(b.Dx()/2, b.Dy()/2).A < 250 {
				t.Fatalf("%s checker %d centre not opaque", id, i)
			}
			if spr.RGBAAt(0, 0).A > 10 {
				t.Fatalf("%s checker %d corner not transparent", id, i)
			}
		}
		if !k.geom.board.contains(l.feltX+1, l.feltY+1) || !k.geom.board.contains(k.geom.tray.cx(), k.geom.tray.cy()) {
			t.Fatalf("%s fields/tray outside board", id)
		}
	}
}

func colorDist(a, b color.RGBA) int {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) + d(a.G, b.G) + d(a.B, b.B)
}
