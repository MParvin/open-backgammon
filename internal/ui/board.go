package ui

import (
	"fmt"
	"image/color"

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
	trayY, trayH               float64
	r                          float64 // checker radius
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
		trayY:  feltY,
		trayH:  feltH,
		r:      20,
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
	colSelect  = color.RGBA{0xff, 0xd7, 0x6a, 0xff}
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
	return x >= l.trayX && x < l.trayX+l.trayW && y >= l.trayY && y < l.trayY+l.trayH
}

func (l boardLayout) checkerXY(point, index, total int) (float64, float64) {
	col, top := l.slot(point)
	cx := l.colX(col) + l.pointW/2
	r := l.r
	span := l.feltH/2 - r - 16
	gap := 2*r + 2
	if total > 1 && gap*float64(total-1) > span {
		gap = span / float64(total-1)
	}
	if top {
		return cx, l.feltY + r + 6 + float64(index)*gap
	}
	return cx, l.feltY + l.feltH - r - 6 - float64(index)*gap
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
		return cx, l.trayY + l.trayH - 14 - float64(index)*10
	}
	return cx, l.trayY + 14 + float64(index)*10
}

func (a *App) drawBoard(screen *ebiten.Image, b game.Board) {
	if a.kit != nil {
		a.drawKitBoard(screen, b)
		return
	}
	a.drawBoardClassic(screen, b)
}

func (a *App) drawBoardClassic(screen *ebiten.Image, b game.Board) {
	l := a.layout
	vector.DrawFilledRect(screen, 0, 0, screenW, screenH, colBG, false)
	vector.DrawFilledRect(screen, 28, 72, 1224, 700, colWood, false)
	vector.DrawFilledRect(screen, 36, 80, 1088, 684, colWoodHi, false)
	border := colGoldDk
	if a.theme.ID == ThemeModern {
		border = a.theme.Accent
	}
	vector.StrokeRect(screen, 42, 86, 1076, 672, 1.5, border, true)
	vector.DrawFilledRect(screen, float32(l.feltX), float32(l.feltY), float32(l.feltW), float32(l.feltH), colFelt, false)
	vector.DrawFilledRect(screen, float32(l.barX), float32(l.feltY), float32(l.barW), float32(l.feltH), colFeltDk, false)
	vector.DrawFilledRect(screen, float32(l.trayX), float32(l.trayY), float32(l.trayW), float32(l.trayH), color.RGBA{0x2a, 0x1c, 0x12, 0xff}, false)
	if a.hintOff() {
		vector.StrokeRect(screen, float32(l.trayX), float32(l.trayY), float32(l.trayW), float32(l.trayH), 3, colSelect, true)
	}

	for col := 0; col < 12; col++ {
		for _, top := range []bool{true, false} {
			a.drawPoint(screen, l, l.pointOf(col, top), top)
		}
	}
	a.drawCheckers(screen, b)
	a.drawDice(screen)
}

// pointTriangle returns the three corners of a point's triangle.
func (l boardLayout) pointTriangle(point int, top bool, length float64) (x0, baseY, x1, tipX, tipY float64) {
	col, _ := l.slot(point)
	x := l.colX(col)
	if length <= 0 {
		length = l.feltH/2 - 18
	}
	if top {
		baseY = l.feltY
		tipY = baseY + length
	} else {
		baseY = l.feltY + l.feltH
		tipY = baseY - length
	}
	return x + 3, baseY, x + l.pointW - 3, x + l.pointW/2, tipY
}

func (a *App) drawPoint(screen *ebiten.Image, l boardLayout, point int, top bool) {
	col, _ := l.slot(point)
	clr := colPointL
	if (col+boolInt(top))%2 == 1 {
		clr = colPointD
	}
	if a.hintPoint(point) {
		clr = color.RGBA{0xff, 0xe0, 0x8a, 0xff}
	}
	x0, baseY, x1, tipX, tipY := l.pointTriangle(point, top, 0)
	if top {
		baseY += 8
	} else {
		baseY -= 8
	}
	var path vector.Path
	path.MoveTo(float32(x0+1), float32(baseY))
	path.LineTo(float32(x1-1), float32(baseY))
	path.LineTo(float32(tipX), float32(tipY))
	path.Close()
	var draw vector.DrawPathOptions
	draw.AntiAlias = true
	draw.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, &draw)

	if a.hasFrom && a.selFrom == point {
		vector.StrokeLine(screen, float32(x0+3), float32(baseY), float32(tipX), float32(tipY), 3, colSelect, true)
		vector.StrokeLine(screen, float32(x1-3), float32(baseY), float32(tipX), float32(tipY), 3, colSelect, true)
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
		a.drawOffChecker(screen, x, y, game.White)
	}
	for i := 0; i < blackOff; i++ {
		x, y := l.offXY(game.Black, i)
		a.drawOffChecker(screen, x, y, game.Black)
	}
	if a.anim != nil {
		t := smoother(a.anim.t)
		x := a.anim.x0 + (a.anim.x1-a.anim.x0)*t
		y := a.anim.y0 + (a.anim.y1-a.anim.y0)*t
		a.drawChecker(screen, x, y, l.r, a.anim.color)
	}
	if a.dragging {
		a.drawChecker(screen, float64(a.dragX), float64(a.dragY), l.r, a.turn)
	}
}

func (a *App) stack(screen *ebiten.Image, c game.Color, n int, xy func(i, n int) (float64, float64), hideOne bool) {
	drawN := n
	if hideOne && drawN > 0 {
		drawN--
	}
	for i := 0; i < drawN; i++ {
		x, y := xy(i, n)
		a.drawChecker(screen, x, y, a.layout.r, c)
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

// drawChecker paints one checker centred at (x, y): a sprite cut from the
// reference artwork on plate themes, a shaded disc otherwise.
func (a *App) drawChecker(screen *ebiten.Image, x, y, r float64, c game.Color) {
	if a.kit != nil {
		a.kit.drawChecker(screen, x, y, r, c)
		return
	}
	drawChecker(screen, x, y, r, c)
}

// drawOffChecker draws a borne-off checker. Plate themes show them edge-on
// as slats in the tray; the classic board keeps small discs.
func (a *App) drawOffChecker(screen *ebiten.Image, x, y float64, c game.Color) {
	if a.kit != nil {
		a.kit.drawSlat(screen, x, y, c)
		return
	}
	drawChecker(screen, x, y, 16, c)
}

func drawChecker(screen *ebiten.Image, x, y, r float64, c game.Color) {
	fill, line := colWhite, colWhiteLn
	if c == game.Black {
		fill, line = colBlack, colBlackLn
	}
	vector.DrawFilledCircle(screen, float32(x+2.4), float32(y+3.4), float32(r+0.8), color.RGBA{0, 0, 0, 80}, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r+1.4), line, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r-0.4), fill, true)
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

// dicePlacement returns the first die's origin, the die size, and the gap
// to the second die (which follows horizontally, or vertically on kits
// that stack dice in a well).
func (a *App) dicePlacement() (x, y, size, gap float64) {
	if a.kit != nil {
		d := a.kit.geom.dice
		return d.x, d.y, d.w, d.w * 0.18
	}
	x, y = a.layout.diceOrigin()
	return x, y, dieSize, 16
}

func (a *App) onBoard(x, y float64) bool {
	l := a.layout
	if x >= l.feltX && x < l.feltX+l.feltW && y >= l.feltY && y < l.feltY+l.feltH {
		return true
	}
	if l.trayAt(x, y) {
		return true
	}
	if a.kit != nil {
		return a.kit.geom.board.contains(x, y)
	}
	return false
}

func (a *App) drawDice(screen *ebiten.Image) {
	// Dice stay visible for the whole turn (human or AI), including while
	// moves are in progress. They clear only when endTurn sets showDice false.
	if !a.showDice || a.dice[0] == 0 {
		return
	}
	x, y, size, gap := a.dicePlacement()
	used0, used1 := a.dieUsedFlags()
	rem := 0
	if a.roll != nil {
		rem = len(a.roll.RemainingDiceValues())
	}
	if a.kit != nil {
		x2, y2 := x+size+gap, y
		if a.kit.geom.diceVertical {
			x2, y2 = x, y+size+gap
		}
		a.kit.drawDie(screen, x, y, size, a.dice[0], used0, 0)
		a.kit.drawDie(screen, x2, y2, size, a.dice[1], used1, 1)
		if a.dice[0] == a.dice[1] {
			if a.kit.geom.diceVertical {
				a.drawText(screen, fmt.Sprintf("x%d", rem), a.small, x+size+4, y+size, colGold)
			} else {
				a.drawText(screen, fmt.Sprintf("x%d", rem), a.small, x2+size+8, y+size/2-8, colGold)
			}
		}
		return
	}
	drawDie(screen, x, y, a.dice[0], used0)
	drawDie(screen, x+size+gap, y, a.dice[1], used1)
	if a.dice[0] == a.dice[1] {
		a.drawText(screen, fmt.Sprintf("x%d", rem), a.small, x+size*2+gap+8, y+size/2-8, colGold)
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
