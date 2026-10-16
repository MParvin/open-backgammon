package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"bg/internal/game"
)

// turnSnap stores a full game snapshot. Restoring it rebuilds checkers,
// dice, cube, match score, and turn phase together.
type turnSnap struct {
	snap      game.Snapshot
	voluntary bool
}

func (a *App) capture() game.Snapshot {
	s := game.Snapshot{
		Board:         a.view(),
		Turn:          a.turn,
		Dice:          a.dice,
		Rolled:        a.rolled,
		ShowDice:      a.showDice,
		Cube:          game.CloneCube(a.cube),
		PendingDouble: a.pendingDouble,
		Doubler:       a.doubler,
		Match:         a.match,
		GameOver:      a.gameOverShow,
		MatchOver:     a.matchOverShow,
		LastOutcome:   a.lastOutcome,
		LastPoints:    a.lastPoints,
		Status:        a.status,
	}
	if a.roll != nil {
		r := game.CloneRoll(*a.roll)
		s.Roll = &r
	}
	if a.winner != nil {
		w := *a.winner
		s.Winner = &w
	}
	return s.Clone()
}

func (a *App) restore(s game.Snapshot) {
	s = s.Clone()
	a.turn = s.Turn
	a.dice = s.Dice
	a.rolled = s.Rolled
	a.showDice = s.ShowDice
	a.cube = s.Cube
	a.pendingDouble = s.PendingDouble
	a.cubePrompt = s.PendingDouble
	a.doubler = s.Doubler
	a.match = s.Match
	a.winner = s.Winner
	a.gameOverShow = s.GameOver
	a.matchOverShow = s.MatchOver
	a.lastOutcome = s.LastOutcome
	a.lastPoints = s.LastPoints
	a.status = s.Status
	a.roll = s.Roll
	if a.roll != nil {
		a.board = a.roll.Origin
	} else {
		a.board = s.Board
	}
	a.anim = nil
	a.queue = nil
	a.pause = 0
	a.clearSelection()
	a.dieFilter = 0
}

func (a *App) pushUndo() {
	if !a.humanTurn() || a.winner != nil || a.gameOverShow {
		return
	}
	a.undoStack = append(a.undoStack, turnSnap{snap: a.capture(), voluntary: a.voluntary})
}

func (a *App) clearUndo() {
	a.undoStack = nil
}

// canUndoHuman reports whether the human may reverse a move this turn.
// Undo is refused after the turn has ended or the AI is acting.
func (a *App) canUndoHuman() bool {
	if a.scene != sceneGame || a.winner != nil || a.gameOverShow || a.matchOverShow {
		return false
	}
	if !a.humanTurn() || a.thinking || a.cubePrompt || a.anim != nil || len(a.queue) > 0 {
		return false
	}
	return len(a.undoStack) > 0
}

// undoHuman restores the last snapshot. It does not cross turn boundaries
// or rewind AI moves. Ctrl+Z and middle-click share this path.
func (a *App) undoHuman() {
	if !a.canUndoHuman() {
		if a.scene == sceneGame && a.humanTurn() && len(a.undoStack) == 0 {
			a.status = "Nothing to undo"
		} else if a.scene == sceneGame && !a.humanTurn() {
			a.status = "Cannot undo after the AI has moved"
		}
		return
	}
	snap := a.undoStack[len(a.undoStack)-1]
	a.undoStack = a.undoStack[:len(a.undoStack)-1]
	a.restore(snap.snap)
	a.voluntary = snap.voluntary
	a.passing = false
	a.skipForce = true // do not immediately re-apply a forced follow-up
	a.awaitConfirm = a.voluntary && a.roll != nil && a.roll.Complete() && len(a.roll.Steps) > 0
	if a.awaitConfirm {
		a.status = "Left-click or Space to confirm"
	} else {
		a.status = "Undid last move"
	}
}

// tryDieMove attempts a single-die move from from.
// When preferLarger is true (left-click), the larger remaining die is tried
// first; if that checker cannot use it, the smaller die is tried automatically.
// Right-click (preferLarger false) always uses the smaller die only.
func (a *App) tryDieMove(from int, preferLarger bool) bool {
	if a.roll == nil || !a.rolled {
		return false
	}
	larger := a.roll.LargerRemainingDie()
	smaller := a.roll.SmallerRemainingDie()
	if larger == 0 {
		a.status = "No dice remaining"
		return false
	}
	order := []int{smaller}
	if preferLarger {
		if larger == smaller {
			order = []int{larger}
		} else {
			order = []int{larger, smaller}
		}
	}
	for i, want := range order {
		opts := stepsFromDie(a.roll, from, want)
		if len(opts) == 0 {
			continue
		}
		if len(opts) > 1 {
			a.hasFrom = true
			a.selFrom = from
			a.dieFilter = want
			a.status = fmt.Sprintf("Choose a destination for die %d", want)
			return false
		}
		if preferLarger && i > 0 {
			a.status = fmt.Sprintf("Used smaller die (%d)", want)
		}
		return a.applySteps([]game.Step{opts[0]})
	}
	a.hasFrom = true
	a.selFrom = from
	a.dieFilter = order[0]
	a.status = fmt.Sprintf("No legal move from there with %v", order)
	return false
}

func stepsFromDie(r *game.Roll, from, die int) []game.Step {
	var opts []game.Step
	for _, s := range r.NextSteps() {
		if s.From == from && s.Die == die {
			opts = append(opts, s)
		}
	}
	return opts
}

func (a *App) applySteps(steps []game.Step) bool {
	if a.roll == nil || len(steps) == 0 {
		return false
	}
	a.pushUndo()
	a.clearSelection()
	a.dieFilter = 0
	a.voluntary = true
	a.skipForce = false
	a.queue = append([]game.Step{}, steps...)
	a.pump()
	return true
}

// maybeForce plays the only remaining legal sequence, or notes a forced pass.
func (a *App) maybeForce() {
	if a.roll == nil || !a.rolled || a.anim != nil || len(a.queue) > 0 || a.thinking || a.cubePrompt {
		return
	}
	if a.winner != nil || a.gameOverShow {
		return
	}
	if a.skipForce {
		return
	}
	if forced, ok := a.roll.ForcedSequence(); ok {
		// Do not push a separate undo entry. Forced follow-ups after a
		// voluntary move undo together with that move. A fully forced roll
		// has no player choice to reverse.
		a.status = "Forced move"
		a.queue = append([]game.Step{}, forced.Steps...)
		a.pump()
		if a.roll != nil && a.roll.Complete() && a.anim == nil && len(a.queue) == 0 {
			a.pause = 8
		}
		return
	}
	if a.roll.Complete() && len(a.roll.Steps) == 0 && !a.passing {
		a.passing = true
		a.status = "No legal move"
		a.pause = 40
	}
}

// confirmTurn finalizes a completed voluntary sequence once.
func (a *App) confirmTurn() {
	if !a.awaitConfirm || a.roll == nil || !a.roll.Complete() || len(a.roll.NextSteps()) > 0 {
		if a.rolled && a.roll != nil && !a.roll.Complete() {
			a.status = "Dice still to play"
		}
		return
	}
	a.awaitConfirm = false
	a.endTurn()
}

func (a *App) onSpace() {
	if a.awaitConfirm {
		a.confirmTurn()
		return
	}
	if a.canRoll() {
		a.rollDice()
	}
}

// tryCombinedDest plays the only multi-step legal sequence that lands on dest.
func (a *App) tryCombinedDest(dest int) bool {
	if a.roll == nil || !a.rolled || a.awaitConfirm {
		return false
	}
	plays := a.roll.CombinedPlays(dest)
	if len(plays) == 0 {
		a.status = "No combined move reaches that point"
		a.clearSelection()
		return false
	}
	if len(plays) > 1 {
		msg := "Ambiguous — pick a checker"
		for i, p := range plays {
			if i >= 3 {
				break
			}
			msg += " · " + p.String()
		}
		a.status = msg
		a.clearSelection()
		return false
	}
	return a.applySteps(plays[0].Steps)
}

func (a *App) ownCheckerAt(x, y float64) (int, bool) {
	if a.roll == nil {
		return 0, false
	}
	b := a.view()
	if a.layout.barAt(x, y) && b.OnBar(a.turn) > 0 {
		return game.Bar, true
	}
	if p, ok := a.layout.pointAt(x, y); ok && b.Count(a.turn, p) > 0 {
		return p, true
	}
	return 0, false
}

func (a *App) handlePointerControls() {
	if !a.humanTurn() || a.thinking || a.cubePrompt || a.gameOverShow || a.matchOverShow {
		return
	}
	x, y := ebiten.CursorPosition()
	fx, fy := float64(x), float64(y)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		a.undoHuman()
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if a.hitButton(fx, fy) {
			return
		}
		if !a.rolled || a.roll == nil || a.anim != nil || len(a.queue) > 0 {
			return
		}
		if from, ok := a.ownCheckerAt(fx, fy); ok {
			a.tryDieMove(from, false)
		}
		return
	}
}

func (a *App) handleUndoShortcut() {
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		a.undoHuman()
	}
}
