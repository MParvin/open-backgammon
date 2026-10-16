package ui

import (
	"testing"

	"bg/internal/game"
)

func drain(a *App) {
	for i := 0; i < 400; i++ {
		if a.anim == nil && len(a.queue) > 0 {
			a.pump()
		}
		if a.anim == nil && len(a.queue) == 0 {
			return
		}
		a.advanceAnim()
	}
}

func TestBlockedBarDoesNotStick(t *testing.T) {
	for _, c := range []game.Color{game.White, game.Black} {
		a := newHumanGame(t)
		a.player = nil
		var b game.Board
		b.Bar[gameColorIndex(c)] = 1
		b.Off[gameColorIndex(c)] = 14
		for p := 1; p <= 24; p++ {
			if c == game.White {
				b.Points[p] = -2
			} else {
				b.Points[p] = 2
			}
		}
		a.board = b
		a.turn = c
		r := game.NewRoll(b, c, 2, 5)
		a.roll = &r
		a.rolled = true
		a.showDice = true
		for i := 0; i < 50; i++ {
			a.updateGame()
		}
		if a.turn == c || a.rolled {
			t.Fatalf("%s still to move rolled=%v status=%q", c, a.rolled, a.status)
		}
	}
}

func gameColorIndex(c game.Color) int {
	if c == game.White {
		return 0
	}
	return 1
}

func TestConfirmCompletesOnce(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[8] = 1
	b.Off[0] = 14
	a.board = b
	r := game.NewRoll(b, game.White, 3, 2)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	plays := game.LegalPlays(b, game.White, 3, 2)
	if len(plays) == 0 || len(plays[0].Steps) < 2 {
		t.Fatal(plays)
	}
	if !a.applySteps(plays[0].Steps) {
		t.Fatal("apply")
	}
	drain(a)
	for i := 0; i < 5; i++ {
		a.updateGame()
	}
	if !a.awaitConfirm || a.turn != game.White {
		t.Fatalf("await %v turn %v steps %v", a.awaitConfirm, a.turn, a.roll.Steps)
	}
	a.onSpace()
	if a.turn != game.Black || a.awaitConfirm || a.rolled {
		t.Fatalf("after space turn %v confirm %v rolled %v", a.turn, a.awaitConfirm, a.rolled)
	}
	a.onSpace()
	if a.turn != game.Black {
		t.Fatal("second space changed the turn")
	}

	a.turn = game.White
	a.board = b
	a.winner = nil
	a.gameOverShow = false
	a.rolled = false
	a.roll = nil
	r = game.NewRoll(b, game.White, 3, 2)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	a.player = nil
	plays = game.LegalPlays(b, game.White, 3, 2)
	if !a.applySteps(plays[0].Steps) {
		t.Fatal("apply")
	}
	drain(a)
	a.updateGame()
	if !a.awaitConfirm {
		t.Fatal("expected confirm")
	}
	a.onClick(0, 0)
	if a.turn != game.Black {
		t.Fatalf("left-click confirm turn %v", a.turn)
	}
	a.confirmTurn()
	if a.turn != game.Black {
		t.Fatal("duplicate confirm")
	}
}

func TestDoublesWaitForConfirm(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[8] = 1
	b.Off[0] = 14
	a.board = b
	r := game.NewRoll(b, game.White, 1, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	plays := game.LegalPlays(b, game.White, 1, 1)
	if len(plays) == 0 {
		t.Fatal(plays)
	}
	// Use a maximal play (four 1s when the path is open).
	best := plays[0]
	for _, p := range plays[1:] {
		if len(p.Steps) > len(best.Steps) {
			best = p
		}
	}
	if !a.applySteps(best.Steps) {
		t.Fatal("apply")
	}
	drain(a)
	a.updateGame()
	if !a.roll.Complete() || !a.awaitConfirm || a.turn != game.White {
		t.Fatalf("doubles should wait, complete %v confirm %v turn %v steps %d", a.roll.Complete(), a.awaitConfirm, a.turn, len(a.roll.Steps))
	}
	a.onSpace()
	a.onSpace()
	if a.turn != game.Black || a.rolled {
		t.Fatalf("turn %v rolled %v", a.turn, a.rolled)
	}
}

func TestIncompleteCannotConfirm(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[13] = 2
	b.Off[0] = 13
	a.board = b
	r := game.NewRoll(b, game.White, 3, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryDieMove(13, true) {
		t.Fatal(a.status)
	}
	drain(a)
	a.updateGame()
	if a.awaitConfirm || a.roll.Complete() {
		t.Fatalf("incomplete should not confirm steps=%v", a.roll.Steps)
	}
	a.confirmTurn()
	if a.status != "Dice still to play" || a.turn != game.White {
		t.Fatalf("status %q turn %v", a.status, a.turn)
	}
}

func TestCombinedClickAndUndo(t *testing.T) {
	a := newHumanGame(t)
	a.player = nil
	var b game.Board
	b.Points[2] = -1
	b.Points[5] = -1
	b.Points[6] = 2
	b.Points[8] = 2
	b.Points[12] = 2
	b.Off[1] = 13
	a.board = b
	a.turn = game.Black
	r := game.NewRoll(b, game.Black, 3, 4)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryCombinedDest(9) {
		t.Fatal(a.status)
	}
	drain(a)
	if a.view().Count(game.Black, 9) != 1 || len(a.undoStack) != 1 {
		t.Fatalf("board %v stack %d", a.view().Points, len(a.undoStack))
	}
	a.undoHuman()
	if a.view().Count(game.Black, 2) != 1 || a.view().Count(game.Black, 5) != 1 || len(a.roll.Steps) != 0 {
		t.Fatalf("undo %+v steps %v", a.view().Points, a.roll.Steps)
	}
	if a.tryCombinedDest(3) {
		t.Fatal("unreachable dest should not move")
	}
}

func TestAmbiguousDestinationDoesNotMove(t *testing.T) {
	a := newHumanGame(t)
	a.player = nil
	var b game.Board
	b.Points[2] = -1
	b.Points[5] = -1
	b.Off[1] = 13
	a.board = b
	a.turn = game.Black
	r := game.NewRoll(b, game.Black, 3, 4)
	a.roll = &r
	a.rolled = true
	before := a.view()
	if a.tryCombinedDest(9) {
		t.Fatal("ambiguous play executed")
	}
	if a.view() != before || len(a.roll.Steps) != 0 || len(a.status) < 9 || a.status[:9] != "Ambiguous" {
		t.Fatalf("status %q", a.status)
	}
}

func TestClickOccupiedOrEmptyPoint7MakesOpening61(t *testing.T) {
	a := newHumanGame(t)
	a.player = nil
	a.board = game.NewBoard()
	r := game.NewRoll(a.board, game.White, 6, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	x, y := a.layout.checkerXY(7, 0, 1)
	a.onClick(x, y)
	drain(a)
	if a.view().Count(game.White, 7) != 2 {
		t.Fatalf("click empty 7 should make the point, status=%q board7=%d steps=%v",
			a.status, a.view().Count(game.White, 7), a.roll.Steps)
	}
	if a.view().Count(game.White, 8) != 2 || a.view().Count(game.White, 13) != 4 {
		t.Fatalf("empty 7 should play 8/7+13/7, board8=%d board13=%d",
			a.view().Count(game.White, 8), a.view().Count(game.White, 13))
	}

	// Occupied point 7: click must move that checker, not auto-make with others.
	a = newHumanGame(t)
	a.player = nil
	moved := game.NewBoard()
	moved.Points[7] = 1
	moved.Points[13] = 4
	a.board = moved
	r = game.NewRoll(moved, game.White, 6, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	before8 := a.view().Count(game.White, 8)
	before13 := a.view().Count(game.White, 13)
	x, y = a.layout.checkerXY(7, 0, 1)
	a.onClick(x, y)
	drain(a)
	if a.view().Count(game.White, 7) != 0 {
		t.Fatalf("click occupied 7 should move that checker, status=%q count7=%d steps=%v",
			a.status, a.view().Count(game.White, 7), a.roll.Steps)
	}
	if a.view().Count(game.White, 8) != before8 || a.view().Count(game.White, 13) != before13 {
		t.Fatalf("occupied 7 must not steal 8/13: 8=%d 13=%d steps=%v",
			a.view().Count(game.White, 8), a.view().Count(game.White, 13), a.roll.Steps)
	}
	if len(a.roll.Steps) != 1 || a.roll.Steps[0].From != 7 {
		t.Fatalf("expected single step from 7, got %v", a.roll.Steps)
	}
}

func TestDragDropMovesChecker(t *testing.T) {
	a := newHumanGame(t)
	a.player = nil
	a.board = game.NewBoard()
	r := game.NewRoll(a.board, game.White, 6, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	a.dragFrom = 13
	a.hasFrom = true
	a.selFrom = 13
	if !a.tryMove(13, 7) {
		t.Fatalf("drag drop 13→7 should succeed, status=%q", a.status)
	}
	drain(a)
	if a.view().Count(game.White, 13) != 4 || a.view().Count(game.White, 7) != 1 {
		t.Fatalf("after drag 13→7 got 13=%d 7=%d steps=%v",
			a.view().Count(game.White, 13), a.view().Count(game.White, 7), a.roll.Steps)
	}
	if len(a.roll.Steps) != 1 || a.roll.Steps[0].From != 13 || a.roll.Steps[0].To != 7 {
		t.Fatalf("steps %v", a.roll.Steps)
	}
}

func TestClickDestMakesPointWithBothDice(t *testing.T) {
	a := newHumanGame(t)
	a.player = nil
	var b game.Board
	b.Points[13] = 2
	b.Points[8] = 2
	b.Points[6] = 2
	b.Off[0] = 9
	a.board = b
	r := game.NewRoll(b, game.White, 6, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryCombinedDest(7) {
		t.Fatalf("expected auto make-point, status=%q", a.status)
	}
	drain(a)
	if a.view().Count(game.White, 7) != 2 {
		t.Fatalf("point 7 not made: %+v steps=%v", a.view().Points, a.roll.Steps)
	}
	if len(a.roll.Steps) != 2 {
		t.Fatalf("steps %v", a.roll.Steps)
	}
}

func TestScoreDoubleAndOpponentIgnored(t *testing.T) {
	a := newHumanGame(t)
	a.kind = "human"
	a.player = a.makePlayer("easy")
	before := a.cube.Value
	a.clickOwnScore()
	if a.cube.Value == before && a.winner == nil && a.status == a.whyNoDouble() && !a.canOfferDouble() {
		t.Fatal("opening double should be legal")
	}
	if a.winner == nil && a.cube.Value == before*2 {
		a.clickOwnScore()
		if a.cube.Value != before*2 {
			t.Fatalf("repeat click changed cube to %d", a.cube.Value)
		}
	}
	a.match.Crawford = true
	a.cube = game.NewCube()
	a.pendingDouble = false
	a.cubePrompt = false
	a.winner = nil
	a.gameOverShow = false
	a.rolled = false
	a.turn = game.White
	a.clickOwnScore()
	if a.cube.Value != 1 || a.status != "Crawford game — the cube is off" {
		t.Fatalf("crawford cube %d status %q", a.cube.Value, a.status)
	}
	a.match.Crawford = false
	a.turn = game.White
	a.cube = game.NewCube()
	a.scoreClick(game.Black)
	if a.status != "Cannot double for the opponent" || a.cube.Value != 1 || a.pendingDouble {
		t.Fatalf("opponent click status %q cube %+v", a.status, a.cube)
	}
}

func TestHamburgerNewConfirms(t *testing.T) {
	a := newHumanGame(t)
	a.menuOpen = true
	var b game.Board
	b.Points[1] = 15
	a.board = b
	items := a.menuEntries()
	var found bool
	for _, item := range items {
		if item.label == "New match" {
			item.action()
			found = true
		}
	}
	if !found || !a.confirmNew || a.view().Count(game.White, 1) != 15 {
		t.Fatalf("confirm %v board changed", a.confirmNew)
	}
	for _, item := range a.menuEntries() {
		if item.label == "Yes, new match" {
			item.action()
		}
	}
	if a.menuOpen || a.view().Count(game.White, 24) != 2 {
		t.Fatalf("new match menu %v board %v", a.menuOpen, a.view().Points)
	}
	labels := map[string]bool{}
	for _, item := range a.menuEntries() {
		labels[item.label] = true
	}
	if !labels["Main menu"] || !labels["New match"] {
		t.Fatalf("menu %v", labels)
	}
}
