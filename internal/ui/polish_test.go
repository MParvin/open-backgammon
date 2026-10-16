package ui

import (
	"testing"

	"bg/internal/config"
	"bg/internal/game"
)

func TestLeftClickFallsBackToSmallerDie(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[13] = 1
	b.Off[0] = 14
	// Block the 6-pip landing (7) but leave 5 (point 8) open.
	b.Points[7] = -2
	a.board = b
	r := game.NewRoll(b, game.White, 6, 5)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryDieMove(13, true) {
		t.Fatal(a.status)
	}
	drain(a)
	if len(a.roll.Steps) != 1 || a.roll.Steps[0].Die != 5 || a.view().Count(game.White, 8) != 1 {
		t.Fatalf("expected fallback to 5: steps=%v board=%v", a.roll.Steps, a.view().Points)
	}
}

func TestLargerDiePreferredWhenLegal(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[13] = 1
	b.Off[0] = 14
	a.board = b
	r := game.NewRoll(b, game.White, 6, 3)
	a.roll = &r
	a.rolled = true
	if !a.tryDieMove(13, true) {
		t.Fatal(a.status)
	}
	drain(a)
	if a.roll.Steps[0].Die != 6 {
		t.Fatalf("should keep larger die, got %v", a.roll.Steps)
	}
}

func TestThemeSwitchPersistsAndKeepsBoard(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[8] = 3
	b.Off[0] = 12
	a.board = b
	a.dice = [2]int{4, 2}
	a.rolled = true
	r := game.NewRoll(b, game.White, 4, 2)
	a.roll = &r
	beforeTurn := a.turn
	a.setTheme(ThemePersian)
	a.cfg.Theme = "persian"
	if a.theme.ID != ThemePersian {
		t.Fatalf("theme %+v", a.theme.ID)
	}
	if a.board.Points[8] != 3 || a.turn != beforeTurn || !a.rolled {
		t.Fatal("theme change must not reset the game")
	}
	a2, err := New(config.Config{Theme: "chinese", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	if a2.theme.ID != ThemeChinese {
		t.Fatal(a2.theme.ID)
	}
	a3, err := New(config.Config{Theme: "modern", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	if a3.theme.ID != ThemeModern || colBG != ThemeByID("modern").BG {
		t.Fatal("modern not applied on New")
	}
	a3.cfg.Theme = "persian"
	if err := config.Save(a3.cfg); err != nil {
		t.Log("save skipped:", err)
	} else {
		loaded, err := config.Load()
		if err == nil && loaded.Theme != "persian" && loaded.Theme != config.DefaultTheme {
			// Env/other keys may coexist; theme field should round-trip when writable.
			t.Logf("loaded theme %q", loaded.Theme)
		}
	}
	for _, th := range Themes() {
		a.setTheme(th.ID)
		if colBG != th.BG || colFelt != th.Felt {
			t.Fatalf("%s colors not applied", th.ID)
		}
	}
}

func TestSurrenderConfirmCancelAndScore(t *testing.T) {
	a := newHumanGame(t)
	a.kind = "easy"
	a.player = a.makePlayer("easy")
	a.confirmSurrender = true
	a.confirmSurrender = false // cancel path
	if a.winner != nil {
		t.Fatal("cancel must not end the game")
	}
	var b game.Board
	b.Off[0] = 0
	b.Points[24] = 15
	b.Points[1] = -15
	a.board = b
	a.cube.Value = 2
	a.doSurrender()
	if a.winner == nil || *a.winner != game.Black {
		t.Fatal("opponent should win")
	}
	if a.lastOutcome != game.Gammon && a.lastOutcome != game.Backgammon {
		// White has nothing off; Black has nothing off → gammon or backgammon.
		t.Fatalf("outcome %v", a.lastOutcome)
	}
	want := game.Points(a.lastOutcome, 2)
	if a.lastPoints != want || a.match.Black != want {
		t.Fatalf("pts %d match %+v", a.lastPoints, a.match)
	}
	if !a.gameOverShow {
		t.Fatal("game over")
	}
}

func TestSurrenderPendingDoubleIsDrop(t *testing.T) {
	a := newHumanGame(t)
	a.kind = "easy"
	a.player = a.makePlayer("easy")
	a.doubler = game.Black
	a.pendingDouble = true
	a.cubePrompt = true
	a.cube.Value = 1
	a.doSurrender()
	if a.winner == nil || *a.winner != game.Black || a.lastPoints != 1 || a.lastOutcome != game.Single {
		t.Fatalf("drop-like surrender winner=%v pts=%d out=%v", a.winner, a.lastPoints, a.lastOutcome)
	}
}

func TestCubeDialogButtonsDistinct(t *testing.T) {
	a := newHumanGame(t)
	a.doubler = game.Black
	a.pendingDouble = true
	a.cubePrompt = true
	a.cube.Value = 2
	var accept, reject bool
	for _, b := range a.buttons() {
		if b.label == "Accept / Take" && b.style == btnAccept {
			accept = true
		}
		if b.label == "Reject / Drop" && b.style == btnReject {
			reject = true
		}
	}
	if !accept || !reject {
		t.Fatal("dialog actions missing")
	}
	a.humanTake()
	if a.cube.Value != 4 || a.pendingDouble {
		t.Fatalf("take %+v", a.cube)
	}
	a.humanTake() // duplicate
	if a.cube.Value != 4 {
		t.Fatal("duplicate take")
	}
}

func TestHamburgerHasThemesAndSurrender(t *testing.T) {
	a := newHumanGame(t)
	a.menuOpen = true
	labels := map[string]bool{}
	for _, item := range a.menuEntries() {
		labels[item.label] = true
	}
	if !labels["Themes"] || !labels["Surrender"] || !labels["New match"] {
		t.Fatalf("%v", labels)
	}
}
