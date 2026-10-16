package ui

import (
	"fmt"
	"time"

	"bg/internal/bot"
	"bg/internal/game"
	"bg/internal/stats"
)

func (a *App) botProfile() bot.Profile {
	switch a.kind {
	case "easy":
		return bot.EasyProfile()
	case "hard":
		return bot.HardProfile()
	default:
		return bot.IntermediateProfile()
	}
}

func (a *App) startMatch() {
	a.kind = []string{"human", "easy", "intermediate", "hard", "ai"}[a.selected]
	a.cfg.MatchTarget = game.ValidTargets[a.targetIdx]
	a.match = game.NewMatch(a.cfg.MatchTarget)
	a.matchLive = true
	a.recorded = false
	a.player = a.makePlayer(a.kind)
	a.menuOpen = false
	a.confirmNew = false
	a.confirmSurrender = false
	a.themePicker = false
	a.resetBoardForGame()
	a.scene = sceneGame
}

func (a *App) resetBoardForGame() {
	a.board = game.NewBoard()
	a.cube = game.NewCube()
	a.pendingDouble = false
	a.cubePrompt = false
	a.doubler = 0
	a.turn = game.White
	a.dice = [2]int{}
	a.rolled = false
	a.roll = nil
	a.showDice = false
	a.winner = nil
	a.lastOutcome = 0
	a.lastPoints = 0
	a.gameOverShow = false
	a.matchOverShow = false
	a.pause = 0
	a.clearUndo()
	a.clearSelection()
	a.anim = nil
	a.queue = nil
	a.thinking = false
	a.results = nil
	if a.match.Crawford {
		a.status = fmt.Sprintf("Crawford game %d — White to play — click the board to roll", a.match.GameNum)
	} else {
		a.status = fmt.Sprintf("Game %d — White to play — click the board to roll", a.match.GameNum)
	}
}

func (a *App) canOfferDouble() bool {
	if !a.humanTurn() || a.rolled || a.thinking || a.anim != nil || len(a.queue) > 0 {
		return false
	}
	if a.winner != nil || a.pendingDouble || a.cubePrompt || a.gameOverShow || a.matchOverShow {
		return false
	}
	return game.CanOffer(a.cube, a.turn, a.match.Crawford, false, a.match.Finished())
}

func (a *App) whyNoDouble() string {
	switch {
	case a.match.Crawford:
		return "Crawford game — the cube is off"
	case a.rolled:
		return "Double before you roll"
	case !a.humanTurn():
		return "Wait for your turn to double"
	case a.cube.Owner != nil && *a.cube.Owner != a.turn:
		return "The opponent owns the cube"
	case a.cube.Value >= game.MaxCube:
		return "The cube is already at 32"
	case a.pendingDouble || a.cubePrompt:
		return "A double is already pending"
	default:
		return "Cannot double now"
	}
}

func (a *App) offerDouble() {
	if !a.canOfferDouble() {
		a.status = a.whyNoDouble()
		return
	}
	a.pushUndo()
	a.doubler = a.turn
	a.pendingDouble = true
	a.cubePrompt = true
	offered := a.cube.OfferedValue()
	if a.player == nil {
		a.status = fmt.Sprintf("Double to %d — opponent must take or drop", offered)
		return
	}
	responder := a.turn.Opponent()
	if bot.ShouldTake(a.board, responder, a.cube, a.match, a.botProfile()) {
		a.cube = a.cube.Take(responder)
		a.pendingDouble = false
		a.cubePrompt = false
		a.status = fmt.Sprintf("Computer takes. Cube ×%d", a.cube.Value)
		return
	}
	a.pendingDouble = false
	a.cubePrompt = false
	a.finishByDrop(a.doubler)
}

// humanMustAnswerCube is true while a human still has to Take or Drop.
func (a *App) humanMustAnswerCube() bool {
	if !a.cubePrompt || !a.pendingDouble || a.doubler == 0 {
		return false
	}
	if a.player == nil {
		return true
	}
	return a.doubler == game.Black
}

func (a *App) maybeComputerDouble() {
	if a.player == nil || a.humanTurn() || a.rolled || a.pendingDouble || a.winner != nil {
		return
	}
	if a.gameOverShow || a.matchOverShow || a.thinking {
		return
	}
	if !game.CanOffer(a.cube, a.turn, a.match.Crawford, false, a.match.Finished()) {
		return
	}
	if !bot.ShouldOffer(a.board, a.turn, a.cube, a.match, a.botProfile()) {
		return
	}
	a.doubler = a.turn
	a.pendingDouble = true
	a.cubePrompt = true
	a.status = fmt.Sprintf("Computer doubles to %d — Take or Drop?", a.cube.OfferedValue())
}

func (a *App) humanTake() {
	if !a.cubePrompt || !a.pendingDouble || a.doubler == 0 {
		return
	}
	taker := a.doubler.Opponent()
	a.cube = a.cube.Take(taker)
	a.pendingDouble = false
	a.cubePrompt = false
	a.clearUndo()
	a.status = fmt.Sprintf("Taken. Cube ×%d", a.cube.Value)
}

func (a *App) humanDrop() {
	if !a.cubePrompt || !a.pendingDouble || a.doubler == 0 {
		return
	}
	winner := a.doubler
	a.pendingDouble = false
	a.cubePrompt = false
	a.finishByDrop(winner)
}

func (a *App) canSurrender() bool {
	if a.player == nil || !a.humanTurn() {
		return false
	}
	if a.winner != nil || a.gameOverShow || a.matchOverShow || a.thinking {
		return false
	}
	return true
}

// doSurrender resigns for White (human seat). Pending computer doubles are
// treated as a Drop (current cube value). Otherwise the opponent is awarded
// Classify(board) × cube, matching a played-out finish from the current position.
func (a *App) doSurrender() {
	if !a.canSurrender() {
		a.confirmSurrender = false
		return
	}
	a.menuOpen = false
	a.confirmSurrender = false
	a.themePicker = false
	a.awaitConfirm = false
	a.queue = nil
	a.anim = nil
	if a.cubePrompt && a.pendingDouble && a.doubler == game.Black {
		a.pendingDouble = false
		a.cubePrompt = false
		a.finishByDrop(a.doubler)
		return
	}
	a.pendingDouble = false
	a.cubePrompt = false
	opp := game.Black
	outcome := game.Classify(a.board, opp)
	pts := game.Points(outcome, a.cube.Value)
	a.lastOutcome = outcome
	a.lastPoints = pts
	w := opp
	a.winner = &w
	a.status = fmt.Sprintf("You surrendered — %s wins a %s for %d", opp, outcome, pts)
	a.applyMatchResult(opp, outcome, a.cube.Value)
}

func (a *App) finishByDrop(winner game.Color) {
	pts := game.Points(game.Single, a.cube.Value)
	a.lastOutcome = game.Single
	a.lastPoints = pts
	w := winner
	a.winner = &w
	a.status = fmt.Sprintf("%s wins by drop (%d)", winner, pts)
	a.applyMatchResult(winner, game.Single, a.cube.Value)
}

func (a *App) finishByBoard(winner game.Color) {
	outcome := game.Classify(a.board, winner)
	pts := game.Points(outcome, a.cube.Value)
	a.lastOutcome = outcome
	a.lastPoints = pts
	w := winner
	a.winner = &w
	a.status = fmt.Sprintf("%s wins a %s for %d", winner, outcome, pts)
	a.applyMatchResult(winner, outcome, a.cube.Value)
}

func (a *App) applyMatchResult(winner game.Color, outcome game.Outcome, cubeValue int) {
	a.clearUndo()
	a.match = a.match.ApplyGame(winner, outcome, cubeValue)
	a.gameOverShow = true
	a.recordGame(winner, outcome, cubeValue)
	if a.match.Finished() {
		a.matchOverShow = true
		a.recordMatchOver()
	}
}

func (a *App) recordGame(winner game.Color, outcome game.Outcome, cubeValue int) {
	if a.kind == "human" {
		return
	}
	_ = stats.Append(stats.Match{
		PlayedAt:    time.Now(),
		Opponent:    a.opponentName(),
		Winner:      winner.String(),
		Outcome:     outcome.String(),
		Points:      game.Points(outcome, cubeValue),
		CubeValue:   cubeValue,
		MatchTarget: a.match.Target,
		MatchWhite:  a.match.White,
		MatchBlack:  a.match.Black,
	})
}

func (a *App) recordMatchOver() {
	if a.kind == "human" || a.recorded {
		return
	}
	a.recorded = true
	w := a.match.Winner()
	if w == nil {
		return
	}
	_ = stats.Append(stats.Match{
		PlayedAt:    time.Now(),
		Opponent:    a.opponentName(),
		Winner:      w.String(),
		MatchTarget: a.match.Target,
		MatchWhite:  a.match.White,
		MatchBlack:  a.match.Black,
		MatchOver:   true,
	})
}

func (a *App) continueMatch() {
	if a.match.Finished() {
		a.scene = sceneMenu
		a.matchLive = false
		return
	}
	a.resetBoardForGame()
}
