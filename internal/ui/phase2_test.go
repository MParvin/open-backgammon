package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bg/internal/config"
	"bg/internal/game"
	"bg/internal/ollama"
	"bg/internal/stats"
)

func TestUndoMultiDieIsAtomic(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Points[16] = 1
	b.Off[0] = 14
	a.board = b
	a.dice = [2]int{4, 4}
	r := game.NewRoll(b, game.White, 4, 4)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryMove(16, 8) {
		t.Fatal("two-die move")
	}
	for a.anim != nil || len(a.queue) > 0 {
		a.advanceAnim()
	}
	if len(a.roll.Steps) != 2 || len(a.undoStack) != 1 {
		t.Fatalf("steps %v stack %d", a.roll.Steps, len(a.undoStack))
	}
	a.undoHuman()
	if a.view().Count(game.White, 16) != 1 || len(a.roll.Steps) != 0 {
		t.Fatalf("undo should restore both dice: %+v steps %v", a.view(), a.roll.Steps)
	}
	if len(a.roll.RemainingDiceValues()) != 4 {
		t.Fatal(a.roll.RemainingDiceValues())
	}
}

func TestFullyForcedRollNotSeparatelyUndoable(t *testing.T) {
	a := newHumanGame(t)
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
	drain(a)
	if a.view().OnBar(game.White) != 0 {
		t.Fatalf("forced entry failed, bar=%d", a.view().OnBar(game.White))
	}
	if len(a.undoStack) != 0 {
		t.Fatal("a fully forced roll has no voluntary choice to undo")
	}
}

func TestUndoAfterVoluntaryThenForced(t *testing.T) {
	a := newHumanGame(t)
	a.player = nil
	// After playing the larger die from 6, only one continuation remains.
	var b game.Board
	b.Points[6] = 1
	b.Off[0] = 14
	for _, p := range []int{5, 4, 3, 2} {
		b.Points[p] = -2
	}
	// 6 with die 5 → 1; then only die 1 from 1 is forced (bear off).
	a.board = b
	r := game.NewRoll(b, game.White, 5, 1)
	a.roll = &r
	a.rolled = true
	a.showDice = true
	if !a.tryDieMove(6, true) {
		t.Fatal(a.status)
	}
	drain(a)
	a.skipForce = false
	a.maybeForce()
	drain(a)
	if !a.roll.Complete() || a.view().BorneOff(game.White) != 15 {
		t.Fatalf("expected forced bear-off, steps=%v off=%d", a.roll.Steps, a.view().BorneOff(game.White))
	}
	if len(a.undoStack) != 1 {
		t.Fatalf("one undo for the voluntary move, got %d", len(a.undoStack))
	}
	a.undoHuman()
	for i := 0; i < 5; i++ {
		a.maybeForce() // must not immediately re-force
	}
	if a.view().Count(game.White, 6) != 1 || len(a.roll.Steps) != 0 {
		t.Fatalf("undo should restore pre-move: %+v steps=%v", a.view().Points, a.roll.Steps)
	}
	if !a.skipForce {
		t.Fatal("skipForce should block re-applying the forced follow-up")
	}
}

func TestUndoRejectedAfterTurnEnds(t *testing.T) {
	a := newHumanGame(t)
	a.rollDice()
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
	a.endTurn()
	a.undoStack = []turnSnap{{snap: a.capture()}}
	a.undoHuman()
	if a.status != "Cannot undo after the AI has moved" {
		t.Fatalf("status %q", a.status)
	}
}

func TestHumanDoubleTakeAndUndo(t *testing.T) {
	a := newHumanGame(t)
	a.kind = "human"
	a.player = a.makePlayer("easy")
	if !a.canOfferDouble() {
		t.Fatal("should be able to double at the start")
	}
	before := a.cube.Value
	a.offerDouble()
	// Computer may take or drop; either is a complete transition.
	if a.winner != nil {
		if a.lastPoints != before {
			t.Fatalf("drop should award pre-offer cube %d, got %d", before, a.lastPoints)
		}
		if len(a.undoStack) != 0 {
			t.Fatal("finished game must clear undo")
		}
		return
	}
	if a.cube.Value != before*2 || a.cube.Owner == nil || *a.cube.Owner != game.Black {
		t.Fatalf("take cube %+v", a.cube)
	}
	a.undoHuman()
	if a.cube.Value != before || a.cube.Owner != nil || a.pendingDouble {
		t.Fatalf("undo double %+v pending %v", a.cube, a.pendingDouble)
	}
}

func TestComputerDoubleTakeDrop(t *testing.T) {
	a := newHumanGame(t)
	a.turn = game.Black
	a.player = a.makePlayer("hard")
	a.kind = "human"
	a.doubler = game.Black
	a.pendingDouble = true
	a.cubePrompt = true
	a.humanTake()
	if a.cube.Value != 2 || a.cube.Owner == nil || *a.cube.Owner != game.White || a.pendingDouble {
		t.Fatalf("take %+v pending %v", a.cube, a.pendingDouble)
	}
	a.doubler = game.Black
	a.pendingDouble = true
	a.cubePrompt = true
	a.humanDrop()
	if a.winner == nil || *a.winner != game.Black || a.lastPoints != 2 {
		t.Fatalf("drop winner %v pts %d", a.winner, a.lastPoints)
	}
	if a.match.Black != 2 {
		t.Fatalf("score %+v", a.match)
	}
}

func TestCrawfordDisablesDouble(t *testing.T) {
	a := newHumanGame(t)
	a.match.Crawford = true
	if a.canOfferDouble() {
		t.Fatal("crawford")
	}
}

func TestMatchProgressAndHistory(t *testing.T) {
	a := newHumanGame(t)
	var b game.Board
	b.Off[0] = 15
	b.Off[1] = 1
	b.Points[24] = -1
	a.board = b
	a.kind = "human"
	dir := t.TempDir()
	path := dir + "/history.json"
	a.finishByBoard(game.White)
	// Single at cube 1 is 1 point; game number advances if the match continues.
	if a.match.White != 1 {
		t.Fatalf("score %+v", a.match)
	}
	a.continueMatch()
	if a.gameOverShow || a.board != game.NewBoard() || a.match.White != 1 {
		t.Fatalf("reset failed %+v over %v", a.match, a.gameOverShow)
	}
	// Interrupted match is not a finished-match record until the target is reached.
	h := stats.History{Matches: []stats.Match{{
		Opponent: "Easy", Winner: "White", Points: 1, MatchTarget: a.match.Target, MatchOver: false,
	}}}
	if err := stats.SaveFile(path, h); err != nil {
		t.Fatal(err)
	}
	loaded, err := stats.LoadFile(path)
	if err != nil || len(loaded.Matches) != 1 || loaded.Matches[0].MatchOver {
		t.Fatalf("%+v %v", loaded, err)
	}
}

func TestProviderSwitch(t *testing.T) {
	a, err := New(config.Config{Provider: "openrouter", MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.scene = sceneSettings
	a.cfg.Provider = "ollama"
	a.ollamaURLField = "http://127.0.0.1:9"
	a.ollamaModelField = "missing"
	a.checkProvider()
	if a.statusNote == "" || a.ollamaModels != nil {
		t.Fatalf("expected offline error, got %q models=%v", a.statusNote, a.ollamaModels)
	}
	p := a.makePlayer("ai")
	if p == nil || p.Name() != "AI" {
		t.Fatal("ai player")
	}
	a.cfg.Provider = "openrouter"
	if a.makePlayer("easy").Name() != "Easy" {
		t.Fatal("built-in easy must not need a provider")
	}
}

func TestOllamaCheckListsAndComboSelects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"},{"name":"qwen2.5:7b"}]}`))
	}))
	defer srv.Close()

	a, err := New(config.Config{Provider: "ollama", MatchTarget: 5, OllamaURL: srv.URL, OllamaModel: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	a.scene = sceneSettings
	a.cfg.Provider = "ollama"
	a.ollamaURLField = srv.URL
	a.ollamaModelField = "missing"
	a.checkProvider()
	if len(a.ollamaModels) != 2 {
		t.Fatalf("models %v note %q", a.ollamaModels, a.statusNote)
	}
	if a.ollamaModelField != "llama3.2:latest" {
		t.Fatalf("should auto-select first installed model, got %q", a.ollamaModelField)
	}
	if !a.ollamaComboOpen {
		t.Fatal("check should open the model combo")
	}
	// Simulate picking the second row in the dropdown.
	y := ollamaComboY + ollamaComboH + ollamaComboItemH + 4
	if !a.handleOllamaComboClick(ollamaComboX+20, y) {
		t.Fatal("combo click should select")
	}
	if a.ollamaModelField != "qwen2.5:7b" || a.ollamaComboOpen {
		t.Fatalf("combo select got %q open=%v", a.ollamaModelField, a.ollamaComboOpen)
	}
	c := a.llmCompleter()
	oc, ok := c.(*ollama.Client)
	if !ok || oc.Model != "qwen2.5:7b" {
		t.Fatalf("completer %+v", c)
	}
}

func newHumanGame(t *testing.T) *App {
	t.Helper()
	a, err := New(config.Config{MatchTarget: 5})
	if err != nil {
		t.Fatal(err)
	}
	a.selected = 1
	a.startGame()
	return a
}
