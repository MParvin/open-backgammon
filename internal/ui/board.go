package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bg/internal/game"
)

const (
	screenW = 1280
	screenH = 800
)

type boardLayout struct {
	feltX, feltY, feltW, feltH float64
	barX, barW                 float64
	leftX, rightX, halfW       float64
	pointW                     float64
	trayX, trayW               float64
}

func newLayout() boardLayout {
	feltX, feltY := 48.0, 86.0
	feltH := 640.0
	barW := 52.0
	trayW := 72.0
	halfW := 510.0
	feltW := halfW*2 + barW
	return boardLayout{
		feltX: feltX, feltY: feltY, feltW: feltW, feltH: feltH,
		barX:   feltX + halfW,
		barW:   barW,
		leftX:  feltX,
		rightX: feltX + halfW + barW,
		halfW:  halfW,
		pointW: halfW / 6,
		trayX:  feltX + feltW + 14,
		trayW:  trayW,
	}
}

var (
	colBG      = color.RGBA{0x14, 0x0e, 0x0a, 0xff}
	colWood    = color.RGBA{0x3d, 0x26, 0x17, 0xff}
	colWoodHi  = color.RGBA{0x6d, 0x48, 0x2a, 0xff}
	colFelt    = color.RGBA{0x14, 0x5a, 0x43, 0xff}
	colFeltDk  = color.RGBA{0x0e, 0x3f, 0x30, 0xff}
	colPointL  = color.RGBA{0xf3, 0xe6, 0xcc, 0xff}
	colPointD  = color.RGBA{0x8b, 0x2e, 0x2b, 0xff}
	colWhite   = color.RGBA{0xf6, 0xf0, 0xe4, 0xff}
	colWhiteLn = color.RGBA{0xc8, 0xb8, 0x98, 0xff}
	colBlack   = color.RGBA{0x1b, 0x1b, 0x1f, 0xff}
	colBlackLn = color.RGBA{0x4a, 0x4a, 0x52, 0xff}
	colGold    = color.RGBA{0xe8, 0xc3, 0x6a, 0xff}
	colGoldDk  = color.RGBA{0x8a, 0x68, 0x2a, 0xff}
	colCream   = color.RGBA{0xfb, 0xf6, 0xec, 0xff}
	colInk     = color.RGBA{0x2a, 0x1a, 0x10, 0xff}
	colMute    = color.RGBA{0xd7, 0xc7, 0xae, 0xff}
	colSelect = color.RGBA{0xff, 0xd7, 0x6a, 0xff}
)

func (l boardLayout) slot(point int) (col int, top bool) {
	if point >= 13 {
		return point - 13, true
	}
	return 12 - point, false
}

func (l boardLayout) pointOf(col int, top bool) int {
	if top {
		return 13 + col
	}
	return 12 - col
}

func (l boardLayout) colX(col int) float64 {
	if col < 6 {
		return l.leftX + float64(col)*l.pointW
	}
	return l.rightX + float64(col-6)*l.pointW
}

func (l boardLayout) colAt(x float64) int {
	if x >= l.leftX && x < l.leftX+l.halfW {
		return int((x - l.leftX) / l.pointW)
	}
	if x >= l.rightX && x < l.rightX+l.halfW {
		return 6 + int((x-l.rightX)/l.pointW)
	}
	return -1
}

func (l boardLayout) pointAt(x, y float64) (int, bool) {
	if y < l.feltY || y >= l.feltY+l.feltH {
		return 0, false
	}
	col := l.colAt(x)
	if col < 0 {
		return 0, false
	}
	return l.pointOf(col, y < l.feltY+l.feltH/2), true
}

func (l boardLayout) barAt(x, y float64) bool {
	return x >= l.barX && x < l.barX+l.barW && y >= l.feltY && y < l.feltY+l.feltH
}

func (l boardLayout) trayAt(x, y float64) bool {
	return x >= l.trayX && x < l.trayX+l.trayW && y >= l.feltY && y < l.feltY+l.feltH
}

func (l boardLayout) checkerXY(point, index, total int) (float64, float64) {
	col, top := l.slot(point)
	cx := l.colX(col) + l.pointW/2
	r := 20.0
	span := l.feltH/2 - 36
	gap := 42.0
	if total > 1 && gap*float64(total-1) > span {
		gap = span / float64(total-1)
	}
	if top {
		return cx, l.feltY + 26 + float64(index)*gap + r*0
	}
	return cx, l.feltY + l.feltH - 26 - float64(index)*gap
}

func (l boardLayout) barXY(c game.Color, index, total int) (float64, float64) {
	cx := l.barX + l.barW/2
	gap := 16.0
	if c == game.White {
		return cx, l.feltY + l.feltH/2 + 36 + float64(index)*gap
	}
	return cx, l.feltY + l.feltH/2 - 36 - float64(index)*gap
}

func (l boardLayout) offXY(c game.Color, index int) (float64, float64) {
	cx := l.trayX + l.trayW/2
	if c == game.White {
		return cx, l.feltY + l.feltH - 24 - float64(index)*10
	}
	return cx, l.feltY + 24 + float64(index)*10
}

func (a *App) drawBoard(screen *ebiten.Image, b game.Board) {
	if a.ornatePersian() {
		a.drawPersianBoard(screen, b)
		return
	}
	a.drawBoardClassic(screen, b)
}

func (a *App) drawBoardClassic(screen *ebiten.Image, b game.Board) {
	l := a.layout
	vector.DrawFilledRect(screen, 0, 0, screenW, screenH, colBG, false)
	vector.DrawFilledRect(screen, 28, 72, 1224, 700, colWood, false)
	vector.DrawFilledRect(screen, 36, 80, 1088, 684, colWoodHi, false)
	a.drawFrameMotif(screen)
	vector.DrawFilledRect(screen, float32(l.feltX), float32(l.feltY), float32(l.feltW), float32(l.feltH), colFelt, false)
	a.drawWoodGrain(screen)
	vector.DrawFilledRect(screen, float32(l.barX), float32(l.feltY), float32(l.barW), float32(l.feltH), colFeltDk, false)
	vector.DrawFilledRect(screen, float32(l.trayX), float32(l.feltY), float32(l.trayW), float32(l.feltH), color.RGBA{0x2a, 0x1c, 0x12, 0xff}, false)
	if a.hintOff() {
		vector.StrokeRect(screen, float32(l.trayX), float32(l.feltY), float32(l.trayW), float32(l.feltH), 3, colSelect, true)
	}

	for col := 0; col < 12; col++ {
		for _, top := range []bool{true, false} {
			a.drawPoint(screen, l, l.pointOf(col, top), top)
		}
	}
	a.drawBarOrnament(screen)
	a.drawCheckers(screen, b)
	a.drawDice(screen)
}

func (a *App) woodTheme() bool {
	return a.theme.ID == ThemePersian || a.theme.ID == ThemeChinese
}

// drawWoodGrain adds faint vertical planks on the light-wood themes.
func (a *App) drawWoodGrain(screen *ebiten.Image) {
	if !a.woodTheme() {
		return
	}
	l := a.layout
	grain := color.RGBA{colFelt.R - 12, colFelt.G - 10, colFelt.B - 8, 0xff}
	for x := l.feltX + 18; x < l.feltX+l.feltW; x += 36 {
		if x > l.barX-2 && x < l.barX+l.barW+2 {
			continue
		}
		vector.StrokeLine(screen, float32(x), float32(l.feltY+4), float32(x), float32(l.feltY+l.feltH-4), 1, grain, true)
	}
}

// drawFrameMotif paints the carved outer frame from the reference boards.
func (a *App) drawFrameMotif(screen *ebiten.Image) {
	accent := a.theme.Accent
	switch a.theme.ID {
	case ThemePersian:
		a.drawCarvedFrame(screen, accent, true)
	case ThemeChinese:
		a.drawCarvedFrame(screen, accent, false)
	case ThemeModern:
		vector.StrokeRect(screen, 40, 84, 1080, 676, 1.5, accent, true)
	default:
		vector.StrokeRect(screen, 42, 86, 1076, 672, 1.5, colGoldDk, true)
	}
}

func (a *App) drawCarvedFrame(screen *ebiten.Image, accent color.RGBA, diamonds bool) {
	vector.StrokeRect(screen, 40, 82, 1080, 680, 2, colGoldDk, true)
	vector.StrokeRect(screen, 46, 88, 1068, 668, 1, accent, true)
	for i := 0; i < 14; i++ {
		x := 70 + float64(i)*74
		if diamonds {
			drawDiamond(screen, float32(x), 96, 10, 8, colGold)
			drawDiamond(screen, float32(x), 742, 10, 8, colGold)
		} else {
			vector.StrokeLine(screen, float32(x-8), 94, float32(x+8), 94, 1.5, colGold, true)
			vector.StrokeLine(screen, float32(x-8), 744, float32(x+8), 744, 1.5, colGold, true)
		}
	}
}

func drawDiamond(screen *ebiten.Image, cx, cy, rx, ry float32, clr color.RGBA) {
	var path vector.Path
	path.MoveTo(cx, cy-ry)
	path.LineTo(cx+rx, cy)
	path.LineTo(cx, cy+ry)
	path.LineTo(cx-rx, cy)
	path.Close()
	var opt vector.DrawPathOptions
	opt.AntiAlias = true
	opt.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, &opt)
}

// drawBarOrnament paints the center motif: Persian star medallion or Chinese dragon.
func (a *App) drawBarOrnament(screen *ebiten.Image) {
	l := a.layout
	cx := float32(l.barX + l.barW/2)
	cy := float32(l.feltY + l.feltH/2)
	switch a.theme.ID {
	case ThemePersian:
		drawPersianMedallion(screen, cx, cy)
	case ThemeChinese:
		drawChineseDragon(screen, cx, cy)
	}
}

func drawPersianMedallion(screen *ebiten.Image, cx, cy float32) {
	red := color.RGBA{0x8a, 0x1c, 0x18, 0xff}
	gold := color.RGBA{0xe8, 0xc0, 0x48, 0xff}
	cream := color.RGBA{0xf6, 0xe6, 0xc0, 0xff}
	vector.DrawFilledCircle(screen, cx, cy, 22, red, true)
	vector.StrokeCircle(screen, cx, cy, 22, 1.6, gold, true)
	drawStar(screen, cx, cy, 18, 8, 8, gold)
	vector.DrawFilledCircle(screen, cx, cy, 7, cream, true)
	vector.StrokeCircle(screen, cx, cy, 7, 1.2, gold, true)
	drawStar(screen, cx, cy, 5, 2.2, 8, red)
	vector.DrawFilledCircle(screen, cx, cy, 2.2, gold, true)
}

func drawStar(screen *ebiten.Image, cx, cy, outer, inner float32, n int, clr color.RGBA) {
	var path vector.Path
	for i := 0; i < n*2; i++ {
		ang := float64(i)*math.Pi/float64(n) - math.Pi/2
		r := outer
		if i%2 == 1 {
			r = inner
		}
		x := cx + float32(math.Cos(ang))*r
		y := cy + float32(math.Sin(ang))*r
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	path.Close()
	var opt vector.DrawPathOptions
	opt.AntiAlias = true
	opt.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, &opt)
}

func drawChineseDragon(screen *ebiten.Image, cx, cy float32) {
	gold := color.RGBA{0xe8, 0xc0, 0x48, 0xff}
	// Vertical S-curve body along the bar.
	var path vector.Path
	const steps = 28
	for i := 0; i <= steps; i++ {
		t := float64(i) / steps
		y := cy - 70 + float32(t)*140
		x := cx + float32(math.Sin(t*math.Pi*2))*10
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	var stroke vector.StrokeOptions
	stroke.Width = 3.2
	var opt vector.DrawPathOptions
	opt.AntiAlias = true
	opt.ColorScale.ScaleWithColor(gold)
	vector.StrokePath(screen, &path, &stroke, &opt)
	// Head at the top of the curve, tail curl at the bottom.
	vector.DrawFilledCircle(screen, cx, cy-74, 6, gold, true)
	vector.DrawFilledCircle(screen, cx-2, cy-76, 1.4, color.RGBA{0x3a, 0x18, 0x08, 0xff}, true)
	vector.StrokeCircle(screen, cx+8, cy+72, 6, 1.6, gold, true)
}

func (a *App) drawPoint(screen *ebiten.Image, l boardLayout, point int, top bool) {
	col, _ := l.slot(point)
	x := l.colX(col)
	clr := colPointL
	if (col+boolInt(top))%2 == 1 {
		clr = colPointD
	}
	if a.hintPoint(point) {
		clr = color.RGBA{0xff, 0xe0, 0x8a, 0xff}
	}
	h := l.feltH/2 - 18
	tipY := l.feltY + h
	baseY := l.feltY + 8
	if !top {
		baseY = l.feltY + l.feltH - 8
		tipY = baseY - h
	}
	var path vector.Path
	path.MoveTo(float32(x+4), float32(baseY))
	path.LineTo(float32(x+l.pointW-4), float32(baseY))
	path.LineTo(float32(x+l.pointW/2), float32(tipY))
	path.Close()
	var draw vector.DrawPathOptions
	draw.AntiAlias = true
	draw.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, &draw)

	if a.hasFrom && a.selFrom == point {
		vector.StrokeLine(screen, float32(x+6), float32(baseY), float32(x+l.pointW/2), float32(tipY), 3, colSelect, true)
		vector.StrokeLine(screen, float32(x+l.pointW-6), float32(baseY), float32(x+l.pointW/2), float32(tipY), 3, colSelect, true)
	}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (a *App) hintPoint(point int) bool {
	return a.hints(point)
}

func (a *App) hintOff() bool {
	return a.hints(game.Off)
}

func (a *App) hints(to int) bool {
	if a.roll == nil || !a.hasFrom {
		return false
	}
	for _, chain := range a.roll.CheckerChains(a.selFrom) {
		if a.dieFilter != 0 && (len(chain) == 0 || chain[0].Die != a.dieFilter) {
			continue
		}
		for _, s := range chain {
			if s.To == to {
				return true
			}
		}
	}
	return false
}

func (a *App) drawCheckers(screen *ebiten.Image, b game.Board) {
	l := a.layout
	hideFrom, hideColor, hide := a.hiddenSource()
	for p := 1; p <= 24; p++ {
		a.stack(screen, game.White, b.Count(game.White, p), func(i, n int) (float64, float64) {
			return l.checkerXY(p, i, n)
		}, hide && hideColor == game.White && hideFrom == p)
		a.stack(screen, game.Black, b.Count(game.Black, p), func(i, n int) (float64, float64) {
			return l.checkerXY(p, i, n)
		}, hide && hideColor == game.Black && hideFrom == p)
	}
	a.stack(screen, game.White, b.OnBar(game.White), func(i, n int) (float64, float64) {
		return l.barXY(game.White, i, n)
	}, hide && hideColor == game.White && hideFrom == game.Bar)
	a.stack(screen, game.Black, b.OnBar(game.Black), func(i, n int) (float64, float64) {
		return l.barXY(game.Black, i, n)
	}, hide && hideColor == game.Black && hideFrom == game.Bar)

	whiteOff := b.BorneOff(game.White) - a.skipOff(game.White)
	blackOff := b.BorneOff(game.Black) - a.skipOff(game.Black)
	for i := 0; i < whiteOff; i++ {
		x, y := l.offXY(game.White, i)
		drawChecker(screen, x, y, 16, game.White)
	}
	for i := 0; i < blackOff; i++ {
		x, y := l.offXY(game.Black, i)
		drawChecker(screen, x, y, 16, game.Black)
	}
	if a.anim != nil {
		t := smoother(a.anim.t)
		x := a.anim.x0 + (a.anim.x1-a.anim.x0)*t
		y := a.anim.y0 + (a.anim.y1-a.anim.y0)*t
		drawChecker(screen, x, y, 20, a.anim.color)
	}
	if a.dragging {
		drawChecker(screen, float64(a.dragX), float64(a.dragY), 20, a.turn)
	}
}

func (a *App) stack(screen *ebiten.Image, c game.Color, n int, xy func(i, n int) (float64, float64), hideOne bool) {
	drawN := n
	if hideOne && drawN > 0 {
		drawN--
	}
	for i := 0; i < drawN; i++ {
		x, y := xy(i, n)
		drawChecker(screen, x, y, 20, c)
	}
}

func (a *App) hiddenSource() (from int, c game.Color, ok bool) {
	if a.anim != nil && a.anim.hide != game.Off {
		return a.anim.hide, a.anim.color, true
	}
	if a.dragging {
		return a.dragFrom, a.turn, true
	}
	return 0, 0, false
}

func (a *App) skipOff(c game.Color) int {
	if a.anim != nil && a.anim.hide == game.Off && a.anim.color == c {
		return 1
	}
	return 0
}

func drawChecker(screen *ebiten.Image, x, y, r float64, c game.Color) {
	fill, line := colWhite, colWhiteLn
	if c == game.Black {
		fill, line = colBlack, colBlackLn
	}
	// Soft drop shadow (light from top-left, matching the reference plate).
	vector.DrawFilledCircle(screen, float32(x+2.4), float32(y+3.4), float32(r+0.8), color.RGBA{0, 0, 0, 80}, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r+1.4), line, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r-0.4), fill, true)
	// Recessed dish like the ivory/ebony checkers in the photo.
	if c == game.White {
		vector.DrawFilledCircle(screen, float32(x), float32(y+0.6), float32(r*0.52), color.RGBA{0x90, 0x78, 0x50, 0x40}, true)
		vector.DrawFilledCircle(screen, float32(x-r*0.28), float32(y-r*0.32), float32(r*0.2), color.RGBA{255, 255, 255, 55}, true)
	} else {
		vector.DrawFilledCircle(screen, float32(x), float32(y+0.6), float32(r*0.5), color.RGBA{0, 0, 0, 50}, true)
		vector.StrokeCircle(screen, float32(x), float32(y), float32(r*0.58), 1.1, color.RGBA{0x70, 0x6a, 0x62, 0x55}, true)
		vector.DrawFilledCircle(screen, float32(x-r*0.22), float32(y-r*0.28), float32(r*0.14), color.RGBA{255, 255, 255, 22}, true)
	}
}

const dieSize = 72.0

func (l boardLayout) diceOrigin() (float64, float64) {
	cx := l.rightX + l.halfW/2
	cy := l.feltY + l.feltH/2
	total := dieSize*2 + 16
	return cx - total/2, cy - dieSize/2
}

func (a *App) dicePlacement() (x, y, size, gap float64) {
	if a.ornatePersian() {
		x, y, size = a.persianDiceOrigin()
		gap = size * 0.18
		return x, y, size, gap
	}
	x, y = a.layout.diceOrigin()
	return x, y, dieSize, 16
}

func (a *App) rollButton() button {
	x, y, size, gap := a.dicePlacement()
	return button{label: "Roll", x: x, y: y, w: size*2 + gap, h: size, action: a.rollDice}
}

func (a *App) drawDice(screen *ebiten.Image) {
	// Dice stay visible for the whole turn (human or AI), including while
	// moves are in progress. They clear only when endTurn sets showDice false.
	if !a.showDice || a.dice[0] == 0 {
		return
	}
	x, y, size, gap := a.dicePlacement()
	used0, used1 := a.dieUsedFlags()
	if a.ornatePersian() {
		// Stack dice vertically in the left recessed well.
		drawIvoryDie(screen, x, y, size, a.dice[0], used0)
		drawIvoryDie(screen, x, y+size+gap, size, a.dice[1], used1)
		if a.dice[0] == a.dice[1] {
			rem := 0
			if a.roll != nil {
				rem = len(a.roll.RemainingDiceValues())
			}
			a.drawText(screen, fmt.Sprintf("x%d", rem), a.small, x+size+6, y+size, colGold)
		}
		return
	}
	drawDie(screen, x, y, a.dice[0], used0)
	drawDie(screen, x+size+gap, y, a.dice[1], used1)
	if a.dice[0] == a.dice[1] {
		rem := 0
		if a.roll != nil {
			rem = len(a.roll.RemainingDiceValues())
		}
		a.drawText(screen, fmt.Sprintf("x%d", rem), a.small, x+size*2+gap+8, y+size/2-8, colGold)
	}
}

func drawIvoryDie(screen *ebiten.Image, x, y, size float64, n int, used bool) {
	fill := color.RGBA{0xed, 0xe0, 0xc8, 0xff}
	edge := color.RGBA{0xa8, 0x90, 0x68, 0xff}
	pip := color.RGBA{0x1a, 0x14, 0x10, 0xff}
	if used {
		fill = color.RGBA{0x9a, 0x88, 0x68, 0xff}
		pip = color.RGBA{0x40, 0x30, 0x20, 0xff}
	}
	vector.DrawFilledCircle(screen, float32(x+2), float32(y+3), float32(size*0.52), color.RGBA{0, 0, 0, 60}, true)
	r := float32(size * 0.18)
	drawRoundRect(screen, float32(x), float32(y), float32(size), float32(size), r, fill)
	vector.StrokeRect(screen, float32(x), float32(y), float32(size), float32(size), 1.5, edge, true)
	scale := size / 28
	for _, p := range diePips(n) {
		vector.DrawFilledCircle(screen, float32(x+p[0]*scale), float32(y+p[1]*scale), float32(2.8*scale), pip, true)
	}
}

func (a *App) dieUsedFlags() (bool, bool) {
	if a.roll == nil {
		return false, false
	}
	rem := a.roll.RemainingDiceValues()
	if a.dice[0] == a.dice[1] {
		// Doubles: dim both once any face is spent; remaining count is in the label.
		return len(rem) < 4, len(rem) < 4
	}
	has := map[int]int{}
	for _, d := range rem {
		has[d]++
	}
	return has[a.dice[0]] == 0, has[a.dice[1]] == 0
}

func drawDie(screen *ebiten.Image, x, y float64, n int, used bool) {
	fill := colGold
	if used {
		fill = color.RGBA{0x6a, 0x58, 0x38, 0xff}
	}
	vector.DrawFilledRect(screen, float32(x), float32(y), dieSize, dieSize, fill, true)
	vector.StrokeRect(screen, float32(x), float32(y), dieSize, dieSize, 3, colGoldDk, true)
	scale := dieSize / 28
	pip := colInk
	if used {
		pip = color.RGBA{0x40, 0x30, 0x20, 0xff}
	}
	for _, p := range diePips(n) {
		vector.DrawFilledCircle(screen, float32(x+p[0]*scale), float32(y+p[1]*scale), float32(3.1*scale), pip, true)
	}
}

func diePips(n int) [][2]float64 {
	const c, a, b = 14.0, 7.0, 21.0
	switch n {
	case 1:
		return [][2]float64{{c, c}}
	case 2:
		return [][2]float64{{a, a}, {b, b}}
	case 3:
		return [][2]float64{{a, a}, {c, c}, {b, b}}
	case 4:
		return [][2]float64{{a, a}, {b, a}, {a, b}, {b, b}}
	case 5:
		return [][2]float64{{a, a}, {b, a}, {c, c}, {a, b}, {b, b}}
	default:
		return [][2]float64{{a, a}, {b, a}, {a, c}, {b, c}, {a, b}, {b, b}}
	}
}

func smoother(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}
