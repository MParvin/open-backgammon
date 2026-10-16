// Package ui is the Ebitengine desktop client: menu, board, themes, and input.
package ui

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bg/assets"
	"bg/internal/bot"
	"bg/internal/config"
	"bg/internal/game"
	"bg/internal/llm"
	"bg/internal/ollama"
	"bg/internal/openrouter"
	"bg/internal/stats"
)

type scene int

const (
	sceneMenu scene = iota
	sceneGame
	sceneSettings
	sceneStats
)

type button struct {
	label  string
	x, y   float64
	w, h   float64
	action func()
	style  btnStyle
}

type anim struct {
	hide   int
	color  game.Color
	x0, y0 float64
	x1, y1 float64
	t      float64
}

type botResult struct {
	play game.Play
	note string
	err  error
}

// App is the Ebitengine game.
type App struct {
	title *text.GoTextFace
	body  *text.GoTextFace
	small *text.GoTextFace
	score *text.GoTextFace

	scene scene
	cfg   config.Config
	rng   *rand.Rand

	opponents []string
	selected  int

	keyField          string
	modelField        string
	ollamaURLField    string
	ollamaModelField  string
	ollamaModels      []string // installed models from Ollama /api/tags
	ollamaComboOpen   bool
	ollamaComboScroll int
	focus             int
	statusNote        string
	backspace         int
	targetIdx         int

	layout boardLayout
	board  game.Board
	turn   game.Color
	kind   string
	player bot.Player

	dice   [2]int
	rolled bool
	roll   *game.Roll
	winner *game.Color
	status string
	pause  int

	selFrom int
	hasFrom bool

	dragging bool
	dragFrom int
	dragX    int
	dragY    int
	pressX   int
	pressY   int
	pressed  bool

	anim     *anim
	queue    []game.Step
	thinking bool
	results  chan botResult
	quit     bool
	ateClick bool

	matchLive     bool
	recorded      bool
	history       stats.History
	match         game.Match
	cube          game.Cube
	pendingDouble bool
	cubePrompt    bool
	doubler       game.Color
	lastOutcome   game.Outcome
	lastPoints    int
	gameOverShow  bool
	matchOverShow bool

	undoStack []turnSnap
	dieFilter int  // when set, only highlight destinations for this die face
	showDice  bool // visual dice stay up for the whole turn

	voluntary    bool // human chose at least one move this turn
	awaitConfirm bool // sequence is complete; left-click or Space ends the turn
	passing      bool // no-move pass countdown already armed
	skipForce    bool // after Undo, do not immediately re-run forced follow-ups

	menuOpen         bool
	menuFocus        int
	confirmNew       bool
	confirmSurrender bool
	themePicker      bool
	theme            Theme
	art              artPlacement // screen placement of ornate board plate
}

func New(cfg config.Config) (*App, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(assets.Font))
	if err != nil {
		return nil, err
	}
	a := &App{
		title:            &text.GoTextFace{Source: src, Size: 54},
		body:             &text.GoTextFace{Source: src, Size: 22},
		small:            &text.GoTextFace{Source: src, Size: 16},
		score:            &text.GoTextFace{Source: src, Size: 42},
		cfg:              cfg,
		rng:              rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		opponents:        []string{"Human", "Easy", "Intermediate", "Hard", "AI"},
		selected:         1,
		keyField:         cfg.APIKey,
		modelField:       cfg.Model,
		ollamaURLField:   cfg.OllamaURL,
		ollamaModelField: cfg.OllamaModel,
		layout:           newLayout(),
		targetIdx:        indexOfTarget(cfg.MatchTarget),
	}
	a.setTheme(ThemeID(cfg.Theme))
	return a, nil
}

func indexOfTarget(n int) int {
	for i, t := range game.ValidTargets {
		if t == n {
			return i
		}
	}
	return 1 // 5
}

func (a *App) Update() error {
	if a.quit {
		return ebiten.Termination
	}
	a.shortcuts()
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && a.scene == sceneGame && a.winner == nil && !a.thinking {
		switch {
		case a.themePicker:
			a.themePicker = false
		case a.menuOpen:
			a.menuOpen = false
			a.confirmNew = false
			a.confirmSurrender = false
		default:
			a.scene = sceneMenu
		}
	}
	switch a.scene {
	case sceneMenu:
		a.clickButtons()
	case sceneSettings:
		a.updateSettings()
	case sceneStats:
		a.clickButtons()
	case sceneGame:
		a.updateGame()
	}
	a.ateClick = false
	if a.quit {
		return ebiten.Termination
	}
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, screenW, screenH, colBG, false)
	switch a.scene {
	case sceneMenu:
		a.drawMenu(screen)
	case sceneSettings:
		a.drawSettings(screen)
	case sceneStats:
		a.drawStats(screen)
	case sceneGame:
		a.drawGame(screen)
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenW, screenH
}

func (a *App) clickButtons() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	x, y := ebiten.CursorPosition()
	if a.scene == sceneMenu {
		for i := range a.opponents {
			r := a.opponentRect(i)
			if inRect(float64(x), float64(y), r) {
				a.selected = i
			}
		}
		if inRect(float64(x), float64(y), button{x: 440, y: 410, w: 400, h: 36}) {
			a.targetIdx = (a.targetIdx + 1) % len(game.ValidTargets)
			a.cfg.MatchTarget = game.ValidTargets[a.targetIdx]
		}
	}
	for _, b := range a.buttons() {
		if inRect(float64(x), float64(y), b) {
			a.ateClick = true
			b.action()
			return
		}
	}
}

func (a *App) shortcuts() {
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyN) && (a.scene == sceneGame || a.scene == sceneMenu) {
		a.startMatch()
	}
	if a.scene == sceneGame && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		a.onSpace()
	}
	if a.scene == sceneGame && inpututil.IsKeyJustPressed(ebiten.KeyD) && !a.menuOpen {
		a.clickOwnScore()
	}
	if a.scene == sceneGame {
		a.handleUndoShortcut()
		a.handleMenuKeys()
	}
}

func (a *App) canRoll() bool {
	return a.winner == nil && a.humanTurn() && !a.rolled && a.anim == nil && len(a.queue) == 0 && !a.thinking && !a.cubePrompt && !a.gameOverShow
}

func (a *App) buttons() []button {
	switch a.scene {
	case sceneMenu:
		startY := 560.0
		settingsY := 626.0
		var btns []button
		if a.matchLive {
			btns = append(btns, button{label: "Resume", x: 440, y: 544, w: 400, h: 48, action: func() { a.scene = sceneGame }})
			startY = 600
			settingsY = 658
		}
		btns = append(btns,
			button{label: "Start match", x: 440, y: startY, w: 400, h: 48, action: a.startMatch},
			button{label: "Settings", x: 440, y: settingsY, w: 190, h: 44, action: func() { a.scene = sceneSettings; a.statusNote = "" }},
			button{label: "Stats", x: 650, y: settingsY, w: 190, h: 44, action: a.openStats},
			button{label: "Quit", x: 440, y: settingsY + 52, w: 400, h: 44, action: func() { a.quit = true }},
		)
		return btns
	case sceneSettings:
		return []button{
			{label: "Check", x: 280, y: 500, w: 200, h: 44, action: a.checkProvider},
			{label: "Save", x: 500, y: 500, w: 200, h: 44, action: a.saveSettings},
			{label: "Back", x: 720, y: 500, w: 200, h: 44, action: func() { a.scene = sceneMenu }},
		}
	case sceneStats:
		return []button{
			{label: "Back", x: 490, y: 720, w: 300, h: 48, action: func() { a.scene = sceneMenu }},
		}
	case sceneGame:
		btns := []button{}
		if a.matchOverShow {
			btns = append(btns, button{label: "Match over", x: 490, y: 480, w: 300, h: 48, action: func() {
				a.matchLive = false
				a.scene = sceneMenu
			}})
			return btns
		}
		if a.gameOverShow {
			btns = append(btns, button{label: "Next game", x: 490, y: 480, w: 300, h: 48, action: a.continueMatch})
			return btns
		}
		if a.humanMustAnswerCube() {
			btns = append(btns,
				button{label: "Accept / Take", x: 360, y: 470, w: 240, h: 56, action: a.humanTake, style: btnAccept},
				button{label: "Reject / Drop", x: 680, y: 470, w: 240, h: 56, action: a.humanDrop, style: btnReject},
			)
			return btns
		}
		if a.winner == nil && a.humanTurn() && a.anim == nil && len(a.queue) == 0 && !a.thinking && !a.cubePrompt {
			if a.canUndoHuman() {
				btns = append(btns, button{label: "Undo", x: 80, y: 16, w: 110, h: 42, action: a.undoHuman, style: btnSecondary})
			}
			if a.canRoll() {
				rb := a.rollButton()
				rb.style = btnPrimary
				btns = append(btns, rb)
			}
		}
		return btns
	default:
		return nil
	}
}

func (a *App) startGame() {
	// Kept for tests: single-game helper that also resets the match.
	a.kind = []string{"human", "easy", "intermediate", "hard", "ai"}[a.selected]
	a.startMatch()
}

func (a *App) openStats() {
	h, err := stats.Load()
	if err != nil {
		a.history = stats.History{}
	} else {
		a.history = h
	}
	a.scene = sceneStats
}

func (a *App) opponentName() string {
	switch a.kind {
	case "easy":
		return "Easy"
	case "intermediate":
		return "Intermediate"
	case "hard":
		return "Hard"
	case "ai":
		return "AI"
	default:
		return "Human"
	}
}

func (a *App) makePlayer(kind string) bot.Player {
	switch kind {
	case "easy":
		return bot.Easy{Rand: a.rng}
	case "intermediate":
		return bot.Intermediate{Rand: a.rng}
	case "hard":
		return bot.Hard{Rand: a.rng}
	case "ai":
		return &bot.AI{Completer: a.llmCompleter()}
	default:
		return nil
	}
}

func (a *App) llmCompleter() llm.Completer {
	switch a.cfg.Provider {
	case "ollama":
		return &ollama.Client{BaseURL: a.cfg.OllamaURL, Model: a.cfg.OllamaModel}
	default:
		if a.cfg.APIKey == "" {
			return nil
		}
		return &openrouter.Client{APIKey: a.cfg.APIKey, Model: a.cfg.Model}
	}
}

func (a *App) humanTurn() bool {
	if a.player == nil {
		return true
	}
	return a.turn == game.White
}

func (a *App) rollDice() {
	if a.rolled || a.winner != nil || a.thinking {
		return
	}
	a.dice[0] = a.rng.IntN(6) + 1
	a.dice[1] = a.rng.IntN(6) + 1
	r := game.NewRoll(a.board, a.turn, a.dice[0], a.dice[1])
	a.roll = &r
	a.rolled = true
	a.showDice = true
	a.clearUndo()
	a.clearSelection()
	a.dieFilter = 0
	a.pause = 0
	a.voluntary = false
	a.awaitConfirm = false
	a.passing = false
	a.skipForce = false
	if a.humanTurn() {
		a.status = fmt.Sprintf("%s rolled %d-%d", a.turn, a.dice[0], a.dice[1])
	} else {
		a.status = fmt.Sprintf("%s rolled %d-%d", a.turn, a.dice[0], a.dice[1])
	}
	a.maybeForce()
}

func (a *App) clearSelection() {
	a.hasFrom = false
	a.selFrom = 0
	a.dragging = false
	a.pressed = false
	a.dieFilter = 0
}

func (a *App) updateGame() {
	a.clickButtons()
	if a.handleGameChrome() {
		a.ateClick = true
	}
	a.updateCursor()
	if a.gameOverShow || a.matchOverShow {
		return
	}
	if a.cubePrompt {
		return
	}
	if a.winner != nil {
		return
	}
	if !a.humanTurn() && !a.rolled && a.anim == nil && !a.thinking && len(a.queue) == 0 {
		a.maybeComputerDouble()
		if a.cubePrompt {
			return
		}
		a.rollDice()
		a.startBot()
	}
	a.pollBot()
	a.advanceAnim()
	if a.humanTurn() && a.anim == nil && len(a.queue) == 0 && !a.thinking && !a.ateClick {
		a.handlePointerControls()
		a.updateHumanInput()
	}
	if a.humanTurn() && a.rolled && a.roll != nil && a.anim == nil && len(a.queue) == 0 && !a.thinking {
		a.maybeForce()
	}
	if a.rolled && a.roll != nil && a.anim == nil && len(a.queue) == 0 && !a.thinking && a.roll.Complete() {
		// A finished voluntary sequence waits for a left-click or Space.
		// Passes and fully forced sequences finalize on their own.
		if a.humanTurn() && a.voluntary && len(a.roll.Steps) > 0 {
			if !a.awaitConfirm {
				a.awaitConfirm = true
				a.status = "Left-click or Space to confirm"
			}
			return
		}
		if a.pause > 0 {
			a.pause--
			if a.pause > 0 {
				return
			}
		}
		a.endTurn()
	}
}

func (a *App) startBot() {
	if a.player == nil || a.roll == nil {
		return
	}
	a.thinking = true
	a.status = a.player.Name() + " is thinking..."
	b := a.board
	c := a.turn
	d1, d2 := a.dice[0], a.dice[1]
	player := a.player
	ch := make(chan botResult, 1)
	a.results = ch
	go func() {
		play, err := player.Choose(b, c, d1, d2)
		note := ""
		if ai, ok := player.(*bot.AI); ok {
			note = ai.Note
		}
		ch <- botResult{play: play, note: note, err: err}
	}()
}

func (a *App) pollBot() {
	if !a.thinking || a.results == nil {
		return
	}
	select {
	case res := <-a.results:
		a.thinking = false
		a.results = nil
		if res.note != "" {
			a.status = res.note
		} else if res.err != nil {
			a.status = "Bot error"
		} else {
			a.status = res.play.String()
		}
		if len(res.play.Steps) == 0 {
			a.pause = 40
			if a.status == "" || a.status == a.player.Name()+" is thinking..." {
				a.status = "No legal move"
			}
			return
		}
		a.queue = append([]game.Step{}, res.play.Steps...)
		a.pump()
	default:
	}
}

func (a *App) pump() {
	if a.anim != nil || len(a.queue) == 0 || a.roll == nil {
		return
	}
	s := a.queue[0]
	a.queue = a.queue[1:]
	fromB := a.roll.Board()
	x0, y0 := a.anchor(fromB, a.turn, s.From)
	next, ok := a.roll.Apply(s)
	if !ok {
		a.queue = nil
		a.anim = nil
		a.status = "Could not play that move"
		a.endTurn()
		return
	}
	x1, y1 := a.anchor(next.Board(), a.turn, s.To)
	a.roll = &next
	_ = fromB
	a.anim = &anim{hide: s.To, color: a.turn, x0: x0, y0: y0, x1: x1, y1: y1}
}

func (a *App) advanceAnim() {
	if a.anim == nil {
		return
	}
	a.anim.t += (1.0 / 60.0) / 0.22
	if a.anim.t >= 1 {
		a.anim = nil
		a.pump()
	}
}

func (a *App) endTurn() {
	if a.roll != nil {
		a.board = a.roll.Board()
	}
	a.showDice = false
	a.clearUndo()
	a.voluntary = false
	a.awaitConfirm = false
	a.passing = false
	a.skipForce = false
	if w := a.board.Winner(); w != nil {
		a.rolled = false
		a.roll = nil
		a.clearSelection()
		a.finishByBoard(*w)
		return
	}
	a.turn = a.turn.Opponent()
	a.rolled = false
	a.roll = nil
	a.queue = nil
	a.pause = 0
	a.clearSelection()
	a.status = a.turn.String() + " to play"
}

func (a *App) updateHumanInput() {
	x, y := ebiten.CursorPosition()
	pressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	released := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	down := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if pressed {
		if a.hitButton(float64(x), float64(y)) {
			a.pressed = false
			return
		}
		a.pressed = true
		a.pressX, a.pressY = x, y
		a.dragX, a.dragY = x, y
		a.dragging = false
		if a.awaitConfirm || a.cubePrompt || a.anim != nil || len(a.queue) > 0 {
			a.dragFrom = -1
			return
		}
		if from, ok := a.fromAt(float64(x), float64(y)); ok {
			a.dragFrom = from
		} else {
			a.dragFrom = -1
		}
		// Do not move on press — wait for drag or tap-on-release.
		return
	}
	if a.pressed && down && a.dragFrom >= 0 && (abs(x-a.pressX) > 6 || abs(y-a.pressY) > 6) {
		a.dragging = true
		a.hasFrom = true
		a.selFrom = a.dragFrom
		a.dieFilter = 0
		a.dragX, a.dragY = x, y
	}
	if a.pressed && released {
		if a.dragging && a.dragFrom >= 0 {
			if to, ok := a.toAt(float64(x), float64(y)); ok {
				if !a.tryMove(a.dragFrom, to) {
					a.clearSelection()
					a.status = "Illegal drop"
				}
			} else {
				a.clearSelection()
			}
		} else {
			// Tap: same click path as before (die move, empty combined dest, confirm).
			a.onClick(float64(x), float64(y))
		}
		a.pressed = false
		a.dragging = false
		a.dragFrom = -1
	}
}

func (a *App) hitButton(x, y float64) bool {
	for _, b := range a.buttons() {
		if inRect(x, y, b) {
			return true
		}
	}
	return false
}

func (a *App) onClick(x, y float64) {
	if a.awaitConfirm {
		a.confirmTurn()
		return
	}
	if !a.rolled || a.roll == nil || a.anim != nil || len(a.queue) > 0 {
		return
	}
	if to, ok := a.toAt(x, y); ok && a.hasFrom {
		if a.tryMove(a.selFrom, to) {
			return
		}
	}
	// Own checker: always move that checker. Never steal a CombinedPlays sequence.
	if from, ok := a.ownCheckerAt(x, y); ok {
		if a.hasFrom && a.selFrom == from && a.dieFilter == 0 {
			a.clearSelection()
			return
		}
		a.tryDieMove(from, true)
		return
	}
	// Empty point (or opponent blot): GNU-style combined destination only.
	if p, ok := a.layout.pointAt(x, y); ok && a.view().Count(a.turn, p) == 0 {
		a.tryCombinedDest(p)
		return
	}
	a.clearSelection()
}

func (a *App) tryMove(from, to int) bool {
	if a.roll == nil {
		return false
	}
	chains := a.roll.CheckerChains(from)
	if a.dieFilter != 0 {
		var filtered [][]game.Step
		for _, ch := range chains {
			if len(ch) > 0 && ch[0].Die == a.dieFilter {
				filtered = append(filtered, ch)
			}
		}
		chains = filtered
	}
	chain := game.ShortestChainTo(chains, to)
	if len(chain) == 0 {
		return false
	}
	return a.applySteps(chain)
}

func (a *App) fromAt(x, y float64) (int, bool) {
	if a.roll == nil {
		return 0, false
	}
	cands := map[int]bool{}
	for _, s := range a.roll.NextSteps() {
		cands[s.From] = true
	}
	if a.layout.barAt(x, y) && cands[game.Bar] && a.view().OnBar(a.turn) > 0 {
		return game.Bar, true
	}
	if p, ok := a.layout.pointAt(x, y); ok && cands[p] && a.view().Count(a.turn, p) > 0 {
		return p, true
	}
	return 0, false
}

func (a *App) toAt(x, y float64) (int, bool) {
	if a.layout.trayAt(x, y) {
		return game.Off, true
	}
	if p, ok := a.layout.pointAt(x, y); ok {
		return p, true
	}
	return 0, false
}

func (a *App) view() game.Board {
	if a.roll != nil {
		return a.roll.Board()
	}
	return a.board
}

func (a *App) anchor(b game.Board, c game.Color, place int) (float64, float64) {
	l := a.layout
	switch place {
	case game.Bar:
		n := b.OnBar(c)
		if n < 1 {
			n = 1
		}
		return l.barXY(c, n-1, n)
	case game.Off:
		n := b.BorneOff(c)
		if n < 1 {
			n = 1
		}
		return l.offXY(c, n-1)
	default:
		n := b.Count(c, place)
		if n < 1 {
			n = 1
		}
		return l.checkerXY(place, n-1, n)
	}
}

func (a *App) opponentRect(i int) button {
	return button{x: 440, y: 160 + float64(i)*48, w: 400, h: 42}
}

func (a *App) drawMenu(screen *ebiten.Image) {
	a.centerText(screen, "Backgammon", a.title, 70, colGold)
	a.centerText(screen, "Opponent", a.small, 140, colMute)
	for i, name := range a.opponents {
		r := a.opponentRect(i)
		fill := color.RGBA{0x2a, 0x1c, 0x12, 0xff}
		if i == a.selected {
			fill = color.RGBA{0x5a, 0x3e, 0x18, 0xff}
		}
		vector.DrawFilledRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), fill, true)
		if i == a.selected {
			vector.StrokeRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), 2, colGold, true)
		}
		a.drawText(screen, name, a.body, r.x+24, r.y+8, colCream)
	}
	a.drawText(screen, fmt.Sprintf("Match to %d  (click to cycle)", game.ValidTargets[a.targetIdx]), a.small, 440, 420, colMute)
	if a.selected == 4 { // AI opponent
		hint := "AI uses OpenRouter — configure in Settings"
		if a.cfg.Provider == "ollama" {
			model := a.cfg.OllamaModel
			if model == "" {
				model = a.ollamaModelField
			}
			hint = "AI uses Ollama (" + model + ") — configure in Settings"
		}
		a.drawText(screen, hint, a.small, 440, 448, colGold)
	}
	a.drawButtons(screen)
}

func (a *App) drawGame(screen *ebiten.Image) {
	a.drawBoard(screen, a.view())
	label := a.status
	if a.thinking && a.player != nil {
		label = a.player.Name() + " is thinking..."
	}
	statusX, statusY := 280.0, 18.0
	matchY := 48.0
	if a.ornatePersian() {
		// Keep chrome in the letterbox / top frame so it does not cover medallions.
		statusX, statusY = 56, 8
		matchY = 34
	}
	a.drawText(screen, label, a.body, statusX, statusY, colCream)
	matchLine := fmt.Sprintf("Match %d–%d / %d   Game %d", a.match.White, a.match.Black, a.match.Target, a.match.GameNum)
	if a.match.Crawford {
		matchLine += "  (Crawford)"
	}
	a.drawText(screen, matchLine, a.small, statusX, matchY, colMute)
	if a.ornatePersian() {
		a.drawPersianPipScores(screen)
	} else {
		a.drawPipScore(screen, "White", a.view().Pips(game.White), 980, 12, colCream, game.White)
		a.drawPipScore(screen, "Black", a.view().Pips(game.Black), 980, 58, colMute, game.Black)
	}
	a.drawCube(screen)
	if a.gameOverShow || a.matchOverShow {
		vector.DrawFilledRect(screen, 340, 300, 600, 200, color.RGBA{0x14, 0x0e, 0x0a, 0xee}, true)
		vector.StrokeRect(screen, 340, 300, 600, 200, 2, colGold, true)
		msg := fmt.Sprintf("%s — %s for %d", a.winner, a.lastOutcome, a.lastPoints)
		if a.winner != nil {
			a.centerText(screen, msg, a.body, 330, colGold)
		}
		a.centerText(screen, fmt.Sprintf("Match %d–%d", a.match.White, a.match.Black), a.body, 370, colCream)
		if a.matchOverShow {
			a.centerText(screen, "Match complete", a.score, 410, colGold)
		}
	}
	if a.humanMustAnswerCube() {
		a.drawCubeOfferDialog(screen)
	}
	a.drawButtons(screen)
	a.drawHamburger(screen)
	if a.themePicker {
		a.drawThemePicker(screen)
	}
	if a.humanTurn() && a.rolled && a.roll != nil && a.hasFrom && !a.cubePrompt {
		a.drawText(screen, "Choose a highlighted point", a.small, 470, 748, colMute)
	}
	if a.awaitConfirm {
		a.drawText(screen, "Move ready — left-click or Space", a.small, 470, 748, colGold)
	}
}

func (a *App) cubeRect() button {
	return button{x: 1164, y: 348, w: 88, h: 108, label: "Doubling cube"}
}

func (a *App) drawCube(screen *ebiten.Image) {
	r := a.cubeRect()
	x, y := r.x+16, r.y+8
	mx, my := ebiten.CursorPosition()
	hover := inRect(float64(mx), float64(my), r) && a.humanTurn() && !a.menuOpen && !a.themePicker && !a.cubePrompt
	if hover && a.canOfferDouble() {
		vector.StrokeRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), 2, colSelect, true)
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), 56, 56, colGold, true)
	vector.StrokeRect(screen, float32(x), float32(y), 56, 56, 2, colGoldDk, true)
	face := fmt.Sprintf("%d", a.cube.Value)
	tw, th := text.Measure(face, a.score, 0)
	a.drawText(screen, face, a.score, x+(56-tw)/2, y+(56-th)/2, colInk)
	owner := "center"
	if a.cube.Owner != nil {
		owner = a.cube.Owner.String()
	}
	if a.match.Crawford {
		owner = "off"
	}
	a.drawText(screen, owner, a.small, x-4, y+60, colMute)
	if hover && a.canOfferDouble() {
		a.drawText(screen, "Double", a.small, x-4, y+78, colGold)
	}
}

func (a *App) drawPipScore(screen *ebiten.Image, name string, pips int, x, y float64, col color.Color, c game.Color) {
	a.drawText(screen, name, a.small, x, y, colMute)
	rect := a.pipRect(c)
	hover := a.hoverOwnScore(c)
	if hover {
		vector.StrokeRect(screen, float32(rect.x), float32(rect.y), float32(rect.w), float32(rect.h), 2, colGold, true)
	}
	label := fmt.Sprintf("%d", pips)
	a.drawText(screen, label, a.score, x+70, y-6, col)
	if hover && a.canOfferDouble() {
		a.drawText(screen, "Double", a.small, x+70, y+36, colGold)
	}
}

func (a *App) drawPersianPipScores(screen *ebiten.Image) {
	a.drawPersianPlaque(screen, game.White, "You", a.view().Pips(game.White))
	a.drawPersianPlaque(screen, game.Black, "Computer", a.view().Pips(game.Black))
}

func (a *App) drawPersianPlaque(screen *ebiten.Image, c game.Color, title string, pips int) {
	r := a.persianScoreRect(c)
	hover := a.hoverOwnScore(c)
	if hover {
		vector.StrokeRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), 2, colGold, true)
	}
	tw, _ := text.Measure(title, a.small, 0)
	a.drawText(screen, title, a.small, r.x+(r.w-tw)/2, r.y+10, colGold)
	label := fmt.Sprintf("%d", pips)
	nw, _ := text.Measure(label, a.score, 0)
	a.drawText(screen, label, a.score, r.x+(r.w-nw)/2, r.y+r.h*0.38, colCream)
	if hover && a.canOfferDouble() {
		a.drawText(screen, "Double", a.small, r.x+12, r.y+r.h-22, colGold)
	}
}

func (a *App) drawSettings(screen *ebiten.Image) {
	a.centerText(screen, "Settings", a.title, 60, colGold)
	a.drawText(screen, "AI provider (used when Opponent is AI)", a.small, 280, 118, colMute)
	a.drawProviderChip(screen, 280, 148, 200, 40, "OpenRouter", a.cfg.Provider != "ollama")
	a.drawProviderChip(screen, 500, 148, 200, 40, "Ollama", a.cfg.Provider == "ollama")
	if a.cfg.Provider == "ollama" {
		a.drawText(screen, "Ollama base URL", a.small, 280, 210, colMute)
		a.drawField(screen, 280, 234, 720, 44, a.ollamaURLField, a.focus == 2, false)
		a.drawText(screen, "Ollama model", a.small, 280, 300, colMute)
		a.drawOllamaModelCombo(screen)
		if !a.ollamaComboOpen {
			a.drawText(screen, "Requires a local Ollama daemon (ollama serve). Check refreshes the model list.", a.small, 280, 384, colMute)
		}
	} else {
		a.drawText(screen, "OpenRouter API key", a.small, 280, 210, colMute)
		a.drawField(screen, 280, 234, 720, 44, a.keyField, a.focus == 0, true)
		a.drawText(screen, "OpenRouter model", a.small, 280, 300, colMute)
		a.drawField(screen, 280, 324, 720, 44, a.modelField, a.focus == 1, false)
	}
	if a.statusNote != "" && !a.ollamaComboOpen {
		a.drawText(screen, a.statusNote, a.small, 280, 420, colGold)
	}
	if !a.ollamaComboOpen {
		a.drawText(screen, "Saved to ~/.config/backgammon/config.json", a.small, 280, 448, colMute)
	}
	a.drawButtons(screen)
	if a.cfg.Provider == "ollama" && a.ollamaComboOpen {
		a.drawOllamaModelDropdown(screen)
	}
}

const (
	ollamaComboX      = 280.0
	ollamaComboY      = 324.0
	ollamaComboW      = 720.0
	ollamaComboH      = 44.0
	ollamaComboItemH  = 36.0
	ollamaComboMaxVis = 6
)

func (a *App) ollamaComboRect() button {
	return button{x: ollamaComboX, y: ollamaComboY, w: ollamaComboW, h: ollamaComboH}
}

func (a *App) drawOllamaModelCombo(screen *ebiten.Image) {
	r := a.ollamaComboRect()
	vector.DrawFilledRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), color.RGBA{0x2a, 0x1c, 0x12, 0xff}, true)
	border := colWoodHi
	if a.focus == 3 || a.ollamaComboOpen {
		border = colGold
	}
	vector.StrokeRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), 2, border, true)
	label := a.ollamaModelField
	if label == "" {
		if len(a.ollamaModels) == 0 {
			label = "Check to load models from Ollama"
		} else {
			label = "Select a model"
		}
	}
	a.drawText(screen, label, a.body, r.x+12, r.y+10, colCream)
	// Chevron
	cx, cy := r.x+r.w-28, r.y+r.h/2
	if a.ollamaComboOpen {
		vector.StrokeLine(screen, float32(cx-6), float32(cy+3), float32(cx), float32(cy-3), 2, colGold, true)
		vector.StrokeLine(screen, float32(cx), float32(cy-3), float32(cx+6), float32(cy+3), 2, colGold, true)
	} else {
		vector.StrokeLine(screen, float32(cx-6), float32(cy-3), float32(cx), float32(cy+3), 2, colGold, true)
		vector.StrokeLine(screen, float32(cx), float32(cy+3), float32(cx+6), float32(cy-3), 2, colGold, true)
	}
}

func (a *App) drawOllamaModelDropdown(screen *ebiten.Image) {
	items := a.ollamaModels
	if len(items) == 0 {
		vector.DrawFilledRect(screen, ollamaComboX, ollamaComboY+ollamaComboH, ollamaComboW, ollamaComboItemH, color.RGBA{0x1a, 0x12, 0x0c, 0xfa}, true)
		vector.StrokeRect(screen, ollamaComboX, ollamaComboY+ollamaComboH, ollamaComboW, ollamaComboItemH, 2, colGold, true)
		a.drawText(screen, "No models — run Check or ollama pull …", a.small, ollamaComboX+12, ollamaComboY+ollamaComboH+10, colMute)
		return
	}
	vis := len(items)
	if vis > ollamaComboMaxVis {
		vis = ollamaComboMaxVis
	}
	h := float64(vis) * ollamaComboItemH
	vector.DrawFilledRect(screen, ollamaComboX, ollamaComboY+ollamaComboH, ollamaComboW, float32(h), color.RGBA{0x1a, 0x12, 0x0c, 0xfa}, true)
	vector.StrokeRect(screen, ollamaComboX, ollamaComboY+ollamaComboH, ollamaComboW, float32(h), 2, colGold, true)
	start := a.ollamaComboScroll
	if start < 0 {
		start = 0
	}
	if start > len(items)-vis {
		start = len(items) - vis
	}
	a.ollamaComboScroll = start
	mx, my := ebiten.CursorPosition()
	for i := 0; i < vis; i++ {
		idx := start + i
		y := ollamaComboY + ollamaComboH + float64(i)*ollamaComboItemH
		hover := inRect(float64(mx), float64(my), button{x: ollamaComboX, y: y, w: ollamaComboW, h: ollamaComboItemH})
		sel := items[idx] == a.ollamaModelField
		if hover || sel {
			fill := color.RGBA{0x5a, 0x3e, 0x18, 0xff}
			if hover && !sel {
				fill = color.RGBA{0x3a, 0x28, 0x18, 0xff}
			}
			vector.DrawFilledRect(screen, ollamaComboX+2, float32(y+1), ollamaComboW-4, ollamaComboItemH-2, fill, true)
		}
		a.drawText(screen, items[idx], a.body, ollamaComboX+12, y+8, colCream)
	}
}

func (a *App) drawProviderChip(screen *ebiten.Image, x, y, w, h float64, label string, active bool) {
	fill := color.RGBA{0x2a, 0x1c, 0x12, 0xff}
	border := colWoodHi
	if active {
		fill = color.RGBA{0x5a, 0x3e, 0x18, 0xff}
		border = colGold
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), fill, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, border, true)
	a.drawText(screen, label, a.body, x+24, y+8, colCream)
}

func (a *App) drawField(screen *ebiten.Image, x, y, w, h float64, value string, focused, secret bool) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), color.RGBA{0x2a, 0x1c, 0x12, 0xff}, true)
	border := colWoodHi
	if focused {
		border = colGold
	}
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, border, true)
	shown := value
	if secret && !focused && value != "" {
		shown = mask(value)
	}
	a.drawText(screen, shown, a.body, x+12, y+10, colCream)
}

func mask(s string) string {
	if len(s) <= 4 {
		return "••••"
	}
	return "••••" + s[len(s)-4:]
}

func (a *App) updateSettings() {
	if a.cfg.Provider == "ollama" && a.ollamaComboOpen {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			a.ollamaComboScroll -= int(dy)
			if a.ollamaComboScroll < 0 {
				a.ollamaComboScroll = 0
			}
			maxScroll := len(a.ollamaModels) - ollamaComboMaxVis
			if maxScroll < 0 {
				maxScroll = 0
			}
			if a.ollamaComboScroll > maxScroll {
				a.ollamaComboScroll = maxScroll
			}
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		fx, fy := float64(x), float64(y)
		if a.cfg.Provider == "ollama" && a.ollamaComboOpen {
			if a.handleOllamaComboClick(fx, fy) {
				return
			}
		}
		switch {
		case inRect(fx, fy, button{x: 280, y: 148, w: 200, h: 40}):
			a.cfg.Provider = "openrouter"
			a.focus = 0
			a.ollamaComboOpen = false
			a.statusNote = ""
		case inRect(fx, fy, button{x: 500, y: 148, w: 200, h: 40}):
			a.cfg.Provider = "ollama"
			a.focus = 2
			a.ollamaComboOpen = false
			a.statusNote = ""
			a.refreshOllamaModels(false)
		case inRect(fx, fy, button{x: 280, y: 234, w: 720, h: 44}):
			a.ollamaComboOpen = false
			if a.cfg.Provider == "ollama" {
				a.focus = 2
			} else {
				a.focus = 0
			}
		case a.cfg.Provider == "ollama" && inRect(fx, fy, a.ollamaComboRect()):
			a.focus = 3
			if len(a.ollamaModels) == 0 {
				a.refreshOllamaModels(true)
			}
			a.ollamaComboOpen = !a.ollamaComboOpen
			a.ollamaComboScroll = 0
		case a.cfg.Provider != "ollama" && inRect(fx, fy, button{x: 280, y: 324, w: 720, h: 44}):
			a.focus = 1
		default:
			a.ollamaComboOpen = false
			a.clickButtons()
		}
	}
	if a.cfg.Provider == "ollama" && a.focus == 3 {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.ollamaComboOpen = false
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			if len(a.ollamaModels) == 0 {
				a.refreshOllamaModels(true)
			}
			a.ollamaComboOpen = !a.ollamaComboOpen
		}
		if a.ollamaComboOpen && len(a.ollamaModels) > 0 {
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
				a.selectOllamaModelByDelta(1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
				a.selectOllamaModelByDelta(-1)
			}
		}
	}
	field := a.settingsField()
	if field != nil {
		for _, r := range ebiten.AppendInputChars(nil) {
			if r == '\n' || r == '\r' || len(*field) > 300 {
				continue
			}
			*field += string(r)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || repeating(ebiten.KeyBackspace, &a.backspace) {
			runes := []rune(*field)
			if len(runes) > 0 {
				*field = string(runes[:len(runes)-1])
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		a.ollamaComboOpen = false
		if a.cfg.Provider == "ollama" {
			if a.focus == 2 {
				a.focus = 3
			} else {
				a.focus = 2
			}
		} else {
			a.focus = 1 - a.focus
			if a.focus < 0 {
				a.focus = 0
			}
		}
	}
}

func (a *App) handleOllamaComboClick(fx, fy float64) bool {
	items := a.ollamaModels
	vis := len(items)
	if vis == 0 {
		empty := button{x: ollamaComboX, y: ollamaComboY + ollamaComboH, w: ollamaComboW, h: ollamaComboItemH}
		if inRect(fx, fy, empty) || inRect(fx, fy, a.ollamaComboRect()) {
			a.ollamaComboOpen = false
			return true
		}
		a.ollamaComboOpen = false
		return false
	}
	if vis > ollamaComboMaxVis {
		vis = ollamaComboMaxVis
	}
	for i := 0; i < vis; i++ {
		idx := a.ollamaComboScroll + i
		if idx < 0 || idx >= len(items) {
			continue
		}
		y := ollamaComboY + ollamaComboH + float64(i)*ollamaComboItemH
		if inRect(fx, fy, button{x: ollamaComboX, y: y, w: ollamaComboW, h: ollamaComboItemH}) {
			a.ollamaModelField = items[idx]
			a.cfg.OllamaModel = items[idx]
			a.ollamaComboOpen = false
			a.statusNote = "Model: " + items[idx]
			return true
		}
	}
	if inRect(fx, fy, a.ollamaComboRect()) {
		a.ollamaComboOpen = false
		return true
	}
	a.ollamaComboOpen = false
	return false
}

func (a *App) selectOllamaModelByDelta(delta int) {
	if len(a.ollamaModels) == 0 {
		return
	}
	idx := 0
	for i, n := range a.ollamaModels {
		if n == a.ollamaModelField {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(a.ollamaModels)) % len(a.ollamaModels)
	a.ollamaModelField = a.ollamaModels[idx]
	a.cfg.OllamaModel = a.ollamaModelField
	// Keep selection visible in the scroll window.
	if idx < a.ollamaComboScroll {
		a.ollamaComboScroll = idx
	}
	if idx >= a.ollamaComboScroll+ollamaComboMaxVis {
		a.ollamaComboScroll = idx - ollamaComboMaxVis + 1
	}
}

func (a *App) refreshOllamaModels(announce bool) {
	a.ollamaURLField = ollama.NormalizeBaseURL(a.ollamaURLField)
	a.cfg.OllamaURL = a.ollamaURLField
	c := &ollama.Client{BaseURL: a.cfg.OllamaURL, Model: a.ollamaModelField}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	names, err := c.Models(ctx)
	if err != nil {
		a.ollamaModels = nil
		if announce {
			host := a.cfg.OllamaURL
			if host == "" {
				host = ollama.DefaultBaseURL
			}
			if strings.Contains(err.Error(), "unavailable") {
				a.statusNote = "Ollama is not running at " + host + " — start it with: ollama serve"
			} else {
				a.statusNote = err.Error()
			}
		}
		return
	}
	a.ollamaModels = names
	if len(names) == 0 {
		if announce {
			a.statusNote = "Ollama is up, but no models are installed (try: ollama pull llama3.2)"
		}
		return
	}
	a.ensureOllamaModelSelected()
	if announce {
		a.statusNote = fmt.Sprintf("Ollama connected — %d models", len(names))
	}
}

func (a *App) ensureOllamaModelSelected() {
	if len(a.ollamaModels) == 0 {
		return
	}
	for _, n := range a.ollamaModels {
		if n == a.ollamaModelField || strings.HasPrefix(n, a.ollamaModelField+":") {
			a.ollamaModelField = n
			a.cfg.OllamaModel = n
			return
		}
	}
	a.ollamaModelField = a.ollamaModels[0]
	a.cfg.OllamaModel = a.ollamaModels[0]
}

func (a *App) settingsField() *string {
	switch a.focus {
	case 0:
		return &a.keyField
	case 1:
		return &a.modelField
	case 2:
		return &a.ollamaURLField
	case 3:
		// Model is chosen from the combo, not typed.
		return nil
	default:
		return &a.modelField
	}
}

func repeating(k ebiten.Key, hold *int) bool {
	if !ebiten.IsKeyPressed(k) {
		*hold = 0
		return false
	}
	*hold++
	return *hold > 20 && *hold%2 == 0
}

func (a *App) checkProvider() {
	a.cfg.APIKey = a.keyField
	a.cfg.Model = a.modelField
	a.cfg.OllamaURL = a.ollamaURLField
	a.cfg.OllamaModel = a.ollamaModelField
	if a.cfg.Provider == "ollama" {
		a.refreshOllamaModels(true)
		if len(a.ollamaModels) > 0 {
			a.ollamaComboOpen = true
			a.ollamaComboScroll = 0
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	c := &openrouter.Client{APIKey: a.cfg.APIKey, Model: a.cfg.Model}
	if err := c.Check(ctx); err != nil {
		a.statusNote = err.Error()
		return
	}
	a.statusNote = "OpenRouter connected"
}

func (a *App) saveSettings() {
	a.cfg.APIKey = a.keyField
	a.cfg.Model = a.modelField
	a.ollamaURLField = ollama.NormalizeBaseURL(a.ollamaURLField)
	a.cfg.OllamaURL = a.ollamaURLField
	a.cfg.OllamaModel = a.ollamaModelField
	a.cfg.MatchTarget = game.ValidTargets[a.targetIdx]
	if a.cfg.Model == "" {
		a.cfg.Model = config.DefaultModel
		a.modelField = a.cfg.Model
	}
	if a.cfg.OllamaModel == "" {
		a.cfg.OllamaModel = config.DefaultOllamaModel
		a.ollamaModelField = a.cfg.OllamaModel
	}
	if a.cfg.Provider == "" {
		a.cfg.Provider = config.DefaultProvider
	}
	if err := config.Save(a.cfg); err != nil {
		a.statusNote = "Could not save settings"
		return
	}
	a.statusNote = "Saved"
}

func (a *App) drawButtons(screen *ebiten.Image) {
	for i, b := range a.buttons() {
		if b.label == "Roll" {
			vector.DrawFilledRect(screen, float32(b.x-4), float32(b.y-4), float32(b.w+8), float32(b.h+8), color.RGBA{0x1a, 0x12, 0x0c, 0x88}, true)
		}
		a.drawStyledButton(screen, b, false)
		_ = i
	}
}

func (a *App) drawCubeOfferDialog(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 300, 250, 680, 320, color.RGBA{0x12, 0x0c, 0x08, 0xf2}, true)
	vector.StrokeRect(screen, 300, 250, 680, 320, 2.5, colGold, true)
	a.centerText(screen, "Your opponent offers to double the stakes.", a.body, 280, colGold)
	a.centerText(screen, "Accept to continue at double stakes, or reject to", a.small, 320, colMute)
	a.centerText(screen, "end the game and concede the current cube value.", a.small, 344, colMute)
	a.centerText(screen, fmt.Sprintf("Current cube: %d", a.cube.Value), a.body, 390, colCream)
	a.centerText(screen, fmt.Sprintf("Proposed cube: %d", a.cube.OfferedValue()), a.body, 424, colSelect)
}

func (a *App) drawStats(screen *ebiten.Image) {
	a.centerText(screen, "Performance", a.title, 70, colGold)
	h := a.history
	if len(h.Matches) == 0 {
		a.centerText(screen, "No matches recorded yet.", a.body, 280, colMute)
		a.centerText(screen, "Finish a game against a bot to start the chart.", a.small, 330, colMute)
		a.drawButtons(screen)
		return
	}
	summary := fmt.Sprintf("Matches %d   Wins %d   Losses %d   Win rate %.0f%%",
		len(h.Matches), h.Wins(), h.Losses(), h.WinRate()*100)
	a.centerText(screen, summary, a.body, 150, colCream)
	a.drawWinRateChart(screen, 180, 210, 920, 360, h.CumulativeWinRate())
	a.drawRecentMatches(screen, h)
	a.drawButtons(screen)
}

func (a *App) drawWinRateChart(screen *ebiten.Image, x, y, w, h float64, rates []float64) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), color.RGBA{0x2a, 0x1c, 0x12, 0xff}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, colGold, true)
	a.drawText(screen, "Win rate over time", a.small, x+16, y+12, colMute)
	a.drawText(screen, "100%", a.small, x+8, y+36, colMute)
	a.drawText(screen, "0%", a.small, x+16, y+h-28, colMute)
	if len(rates) == 0 {
		return
	}
	padL, padT, padR, padB := 56.0, 48.0, 24.0, 36.0
	plotW := w - padL - padR
	plotH := h - padT - padB
	originX := x + padL
	originY := y + padT
	for i := 1; i < len(rates); i++ {
		x0 := originX + plotW*float64(i-1)/float64(max(len(rates)-1, 1))
		x1 := originX + plotW*float64(i)/float64(max(len(rates)-1, 1))
		y0 := originY + plotH*(1-rates[i-1])
		y1 := originY + plotH*(1-rates[i])
		vector.StrokeLine(screen, float32(x0), float32(y0), float32(x1), float32(y1), 3, colGold, true)
	}
	for i, r := range rates {
		cx := originX + plotW*float64(i)/float64(max(len(rates)-1, 1))
		cy := originY + plotH*(1-r)
		vector.DrawFilledCircle(screen, float32(cx), float32(cy), 4, colCream, true)
	}
}

func (a *App) drawRecentMatches(screen *ebiten.Image, h stats.History) {
	a.drawText(screen, "Recent matches", a.small, 180, 590, colMute)
	start := 0
	if len(h.Matches) > 4 {
		start = len(h.Matches) - 4
	}
	y := 620.0
	for _, m := range h.Matches[start:] {
		line := fmt.Sprintf("%s  vs %s  —  %s", m.PlayedAt.Local().Format("2006-01-02 15:04"), m.Opponent, m.Winner)
		a.drawText(screen, line, a.small, 180, y, colCream)
		y += 22
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *App) drawText(dst *ebiten.Image, s string, face *text.GoTextFace, x, y float64, col color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(col)
	text.Draw(dst, s, face, op)
}

func (a *App) centerText(dst *ebiten.Image, s string, face *text.GoTextFace, y float64, col color.Color) {
	w, _ := text.Measure(s, face, 0)
	a.drawText(dst, s, face, (screenW-w)/2, y, col)
}

func inRect(x, y float64, b button) bool {
	return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
