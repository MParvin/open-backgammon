package ui

import (
	"testing"

	"bg/internal/config"
	"bg/internal/game"
)

func TestLeftClickUsesLargerDie(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	var b game.Board
	b.Points[13] = 1
	b.Off[0] = 14
	a.board = b
	a.dice = [2]int{6, 3}
	r := game.NewRoll(b, game.White, 6, 3)
	a.roll = &r
	a.rolled = true
	a.showDice = true

	x, y := a.layout.checkerXY(13, 0, 1)
	a.onClick(x, y)
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if a.view().Count(game.White, 7) != 1 {
		t.Fatalf("expected 13/7 with larger die 6, board=%v steps=%v", a.view().Points, a.roll.Steps)
	}
	if len(a.roll.Steps) != 1 || a.roll.Steps[0].Die != 6 {
		t.Fatalf("steps %v", a.roll.Steps)
	}
}

func TestRightClickUsesSmallerDie(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	var b game.Board
	b.Points[13] = 1
	b.Off[0] = 14
	a.board = b
	a.dice = [2]int{6, 3}
	r := game.NewRoll(b, game.White, 6, 3)
	a.roll = &r
	a.rolled = true
	a.showDice = true

	if !a.tryDieMove(13, false) {
		t.Fatal("smaller die move")
	}
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if a.view().Count(game.White, 10) != 1 || a.roll.Steps[0].Die != 3 {
		t.Fatalf("expected 13/10 with die 3, got %v", a.roll.Steps)
	}
}

func TestUndoRestoresDiceAndBoard(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	var b game.Board
	b.Points[13] = 1
	b.Off[0] = 14
	a.board = b
	a.dice = [2]int{5, 1}
	r := game.NewRoll(b, game.White, 5, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	a.tryDieMove(13, true)
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if len(a.undoStack) != 1 {
		t.Fatalf("undo stack %d", len(a.undoStack))
	}
	a.undoHuman()
	if a.view().Count(game.White, 13) != 1 || len(a.roll.Steps) != 0 {
		t.Fatalf("undo failed: points=%v steps=%v", a.view().Points, a.roll.Steps)
	}
	if !a.showDice || len(a.roll.RemainingDiceValues()) != 2 {
		t.Fatal("dice should remain for the turn")
	}
}

func TestUndoRestoresHit(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	var b game.Board
	b.Points[13] = 1
	b.Points[8] = -1 // blot
	b.Off[0] = 14
	b.Off[1] = 14
	a.board = b
	a.dice = [2]int{5, 2}
	r := game.NewRoll(b, game.White, 5, 2)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryDieMove(13, true) {
		t.Fatal("hit with larger die")
	}
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if a.view().OnBar(game.Black) != 1 {
		t.Fatalf("expected hit, bar=%d", a.view().OnBar(game.Black))
	}
	a.undoHuman()
	if a.view().OnBar(game.Black) != 0 || a.view().Count(game.Black, 8) != 1 {
		t.Fatalf("undo must restore blot: %+v", a.view())
	}
	if len(a.roll.RemainingDiceValues()) != 2 {
		t.Fatal("dice restored")
	}
}

func TestUndoRejectedAfterAITurn(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.turn = game.Black
	a.rolled = true
	a.undoStack = []turnSnap{{}}
	a.undoHuman()
	if a.status != "Cannot undo after the AI has moved" {
		t.Fatalf("status %q", a.status)
	}
}

func TestForcedMoveAutoPlays(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	var b game.Board
	b.Bar[0] = 1
	b.Off[0] = 14
	b.Points[19] = -2
	a.board = b
	a.dice = [2]int{5, 6}
	r := game.NewRoll(b, game.White, 5, 6)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	a.maybeForce()
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	// Full forced sequence: bar/20 then 20/14.
	if a.view().OnBar(game.White) != 0 || a.view().Count(game.White, 14) != 1 {
		t.Fatalf("forced sequence %+v steps=%v", a.view(), a.roll.Steps)
	}
}

func TestDiceStayVisibleUntilTurnEnds(t *testing.T) {
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	a.rollDice()
	if !a.showDice || !a.rolled {
		t.Fatal("dice should show after roll")
	}
	// Simulate completing the roll without a winner.
	if a.roll != nil {
		for !a.roll.Complete() {
			steps := a.roll.NextSteps()
			if len(steps) == 0 {
				break
			}
			next, ok := a.roll.Apply(steps[0])
			if !ok {
				break
			}
			a.roll = &next
		}
	}
	if !a.showDice {
		t.Fatal("dice must remain while the turn is active")
	}
	a.endTurn()
	if a.showDice {
		t.Fatal("dice must clear after the turn ends")
	}
}
