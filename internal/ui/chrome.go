package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bg/internal/config"
	"bg/internal/game"
)

func (a *App) hamburgerRect() button {
	if a.kit != nil {
		return a.kit.geom.hud.hamburger.btn("Menu")
	}
	return button{x: 16, y: 12, w: 44, h: 44, label: "Menu"}
}

func (a *App) pipRect(c game.Color) button {
	if a.kit != nil {
		if c == game.White {
			return a.kit.geom.pipWhite.btn("White score")
		}
		return a.kit.geom.pipBlack.btn("Black score")
	}
	if c == game.White {
		return button{x: 1048, y: 4, w: 120, h: 46, label: "White score"}
	}
	return button{x: 1048, y: 50, w: 120, h: 46, label: "Black score"}
}

func (a *App) hoverOwnScore(c game.Color) bool {
	if a.menuOpen || a.themePicker || !a.humanTurn() || c != a.turn || a.cubePrompt || a.gameOverShow {
		return false
	}
	x, y := ebiten.CursorPosition()
	return inRect(float64(x), float64(y), a.pipRect(c))
}

func (a *App) scoreClick(c game.Color) {
	if c != a.turn {
		a.status = "Cannot double for the opponent"
		return
	}
	a.clickOwnScore()
}

func (a *App) clickOwnScore() {
	if a.pendingDouble || a.cubePrompt {
		return
	}
	a.offerDouble()
}

func (a *App) menuEntries() []button {
	panel := a.menuPanel()
	y := panel.y + 8
	row := func(i int) float64 { return y + float64(i)*48 }
	if a.confirmNew {
		y = panel.y + 36
		return []button{
			{label: "Yes, new match", x: panel.x + 8, y: y, w: panel.w - 16, h: 40, action: func() {
				a.menuOpen = false
				a.confirmNew = false
				a.startMatch()
			}, style: btnDanger},
			{label: "Cancel", x: panel.x + 8, y: y + 48, w: panel.w - 16, h: 40, action: func() {
				a.confirmNew = false
				a.menuFocus = 0
			}, style: btnSecondary},
		}
	}
	if a.confirmSurrender {
		y = panel.y + 36
		return []button{
			{label: "Yes, surrender", x: panel.x + 8, y: y, w: panel.w - 16, h: 40, action: a.doSurrender, style: btnDanger},
			{label: "Cancel", x: panel.x + 8, y: y + 48, w: panel.w - 16, h: 40, action: func() {
				a.confirmSurrender = false
				a.menuFocus = 0
			}, style: btnSecondary},
		}
	}
	return []button{
		{label: "Main menu", x: panel.x + 8, y: row(0), w: panel.w - 16, h: 40, action: func() {
			if a.thinking || a.cubePrompt {
				return
			}
			a.menuOpen = false
			a.scene = sceneMenu
		}, style: btnSecondary},
		{label: "Themes", x: panel.x + 8, y: row(1), w: panel.w - 16, h: 40, action: func() {
			a.menuOpen = false
			a.themePicker = true
			a.menuFocus = 0
		}, style: btnSecondary},
		{label: "Surrender", x: panel.x + 8, y: row(2), w: panel.w - 16, h: 40, action: func() {
			if !a.canSurrender() {
				a.status = "Cannot surrender now"
				return
			}
			a.confirmSurrender = true
			a.menuFocus = 0
		}, style: btnDanger},
		{label: "New match", x: panel.x + 8, y: row(3), w: panel.w - 16, h: 40, action: func() {
			if a.matchLive && !a.matchOverShow {
				a.confirmNew = true
				a.menuFocus = 0
				return
			}
			a.menuOpen = false
			a.startMatch()
		}, style: btnPrimary},
	}
}

func (a *App) menuPanel() button {
	h := 220.0
	if a.confirmNew || a.confirmSurrender {
		h = 148
	}
	if h > screenH-70 {
		h = screenH - 70
	}
	w := 240.0
	if w > screenW-32 {
		w = screenW - 32
	}
	hb := a.hamburgerRect()
	return button{x: hb.x, y: hb.y + hb.h + 8, w: w, h: h, label: "Game menu"}
}

func (a *App) handleMenuKeys() {
	if a.themePicker {
		a.handleThemeKeys()
		return
	}
	if !a.menuOpen {
		return
	}
	items := a.menuEntries()
	if len(items) == 0 {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		a.menuFocus = (a.menuFocus + 1) % len(items)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		a.menuFocus = (a.menuFocus - 1 + len(items)) % len(items)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		if a.menuFocus < 0 || a.menuFocus >= len(items) {
			a.menuFocus = 0
		}
		items[a.menuFocus].action()
	}
}

func (a *App) handleThemeKeys() {
	themes := Themes()
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		a.menuFocus = (a.menuFocus + 1) % len(themes)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		a.menuFocus = (a.menuFocus - 1 + len(themes)) % len(themes)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		a.applyThemeChoice(themes[a.menuFocus].ID)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.themePicker = false
	}
}

func (a *App) applyThemeChoice(id ThemeID) {
	a.setTheme(id)
	_ = config.Save(a.cfg)
	a.themePicker = false
	a.status = "Theme: " + a.theme.Name
}

func (a *App) handleGameChrome() bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	x, y := ebiten.CursorPosition()
	fx, fy := float64(x), float64(y)
	if a.themePicker {
		for i, r := range a.themePreviewRects() {
			if inRect(fx, fy, r) {
				a.applyThemeChoice(Themes()[i].ID)
				return true
			}
		}
		if inRect(fx, fy, a.themeCloseRect()) {
			a.themePicker = false
			return true
		}
		return true
	}
	if inRect(fx, fy, a.hamburgerRect()) {
		if a.thinking {
			return true
		}
		a.menuOpen = !a.menuOpen
		a.confirmNew = false
		a.confirmSurrender = false
		a.menuFocus = 0
		return true
	}
	if a.menuOpen {
		for i, item := range a.menuEntries() {
			if inRect(fx, fy, item) {
				a.menuFocus = i
				item.action()
				return true
			}
		}
		if !inRect(fx, fy, a.menuPanel()) {
			a.menuOpen = false
			a.confirmNew = false
			a.confirmSurrender = false
		}
		return true
	}
	if inRect(fx, fy, a.cubeRect()) && a.humanTurn() && !a.cubePrompt {
		a.clickOwnScore()
		return true
	}
	if inRect(fx, fy, a.pipRect(a.turn)) && a.humanTurn() {
		a.scoreClick(a.turn)
		return true
	}
	opp := a.turn.Opponent()
	if inRect(fx, fy, a.pipRect(opp)) {
		a.scoreClick(opp)
		return true
	}
	return false
}

func (a *App) updateCursor() {
	x, y := ebiten.CursorPosition()
	fx, fy := float64(x), float64(y)
	hover := inRect(fx, fy, a.hamburgerRect()) || a.hoverOwnScore(a.turn) || (inRect(fx, fy, a.cubeRect()) && a.humanTurn() && !a.cubePrompt)
	if a.menuOpen {
		for _, item := range a.menuEntries() {
			if inRect(fx, fy, item) {
				hover = true
			}
		}
	}
	if a.themePicker {
		for _, r := range a.themePreviewRects() {
			if inRect(fx, fy, r) {
				hover = true
			}
		}
	}
	if hover {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
		return
	}
	ebiten.SetCursorShape(ebiten.CursorShapeDefault)
}

func (a *App) drawHamburger(screen *ebiten.Image) {
	r := a.hamburgerRect()
	vector.DrawFilledRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), color.RGBA{0x3a, 0x26, 0x14, 0xff}, true)
	vector.StrokeRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), 2, colGold, true)
	for i := 0; i < 3; i++ {
		yy := r.y + 12 + float64(i)*10
		vector.DrawFilledRect(screen, float32(r.x+10), float32(yy), 24, 3, colCream, true)
	}
	if !a.menuOpen {
		return
	}
	p := a.menuPanel()
	vector.DrawFilledRect(screen, float32(p.x), float32(p.y), float32(p.w), float32(p.h), color.RGBA{0x1c, 0x12, 0x0c, 0xf5}, true)
	vector.StrokeRect(screen, float32(p.x), float32(p.y), float32(p.w), float32(p.h), 2, colGold, true)
	if a.confirmNew {
		a.drawText(screen, "Discard this match?", a.small, p.x+12, p.y+8, colGold)
	}
	if a.confirmSurrender {
		a.drawText(screen, "Resign this game?", a.small, p.x+12, p.y+8, colGold)
	}
	for i, item := range a.menuEntries() {
		a.drawStyledButton(screen, item, i == a.menuFocus)
	}
}

func (a *App) themePreviewRects() []button {
	const (
		ox = 220.0
		oy = 180.0
		w  = 200.0
		h  = 120.0
		g  = 24.0
	)
	var out []button
	for i := range Themes() {
		col := i % 2
		row := i / 2
		out = append(out, button{
			x: ox + float64(col)*(w+g),
			y: oy + float64(row)*(h+g),
			w: w, h: h,
			label: Themes()[i].Name,
		})
	}
	return out
}

func (a *App) themeCloseRect() button {
	return button{x: 520, y: 520, w: 240, h: 48, label: "Close", style: btnSecondary}
}

func (a *App) drawThemePicker(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 160, 120, 960, 480, color.RGBA{0x10, 0x0c, 0x08, 0xf4}, true)
	vector.StrokeRect(screen, 160, 120, 960, 480, 2, colGold, true)
	a.centerText(screen, "Board themes", a.body, 140, colGold)
	a.centerText(screen, "Applied immediately — game state is kept", a.small, 168, colMute)
	for i, r := range a.themePreviewRects() {
		t := Themes()[i]
		if k := kitFor(t.ID); k != nil {
			k.drawPreview(screen, r)
		} else {
			vector.DrawFilledRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), t.Wood, true)
			vector.DrawFilledRect(screen, float32(r.x+10), float32(r.y+28), float32(r.w-20), 54, t.Felt, true)
			vector.DrawFilledRect(screen, float32(r.x+18), float32(r.y+34), 28, 42, t.PointL, true)
			vector.DrawFilledRect(screen, float32(r.x+50), float32(r.y+34), 28, 42, t.PointD, true)
			vector.DrawFilledCircle(screen, float32(r.x+40), float32(r.y+55), 10, t.White, true)
			vector.DrawFilledCircle(screen, float32(r.x+78), float32(r.y+55), 10, t.Black, true)
		}
		sel := a.theme.ID == t.ID || a.menuFocus == i
		border := t.GoldDk
		if sel {
			border = t.Gold
			vector.StrokeRect(screen, float32(r.x-2), float32(r.y-2), float32(r.w+4), float32(r.h+4), 3, border, true)
		} else {
			vector.StrokeRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), 2, border, true)
		}
		a.drawText(screen, t.Name, a.small, r.x+12, r.y+8, t.Cream)
		if a.theme.ID == t.ID {
			a.drawText(screen, "Selected", a.small, r.x+12, r.y+r.h-22, t.Gold)
		}
	}
	close := a.themeCloseRect()
	a.drawStyledButton(screen, close, false)
}
