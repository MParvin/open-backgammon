package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bg/internal/game"
)

// Drawing for plate themes (Persian, Chinese): the composed plate goes down
// first, then move hints, checkers, dice and read-outs are drawn live on top.

func (a *App) drawKitBoard(screen *ebiten.Image, b game.Board) {
	k := a.kit
	k.gpu()
	screen.DrawImage(k.plateImg, nil)

	l := a.layout
	if a.hintOff() {
		t := k.geom.tray
		fillRoundRect(screen, t.x, t.y, t.w, t.h, 8, color.RGBA{0xff, 0xe0, 0x8a, 0x38})
		strokeRoundRect(screen, t.x, t.y, t.w, t.h, 8, 2.5, colSelect)
	}
	for col := 0; col < 12; col++ {
		for _, top := range []bool{true, false} {
			pt := l.pointOf(col, top)
			if a.hintPoint(pt) {
				a.drawHintPoint(screen, l, pt, top)
			}
			if a.hasFrom && a.selFrom == pt {
				a.drawSelectedPoint(screen, l, pt, top)
			}
		}
	}
	if a.hasFrom && a.selFrom == game.Bar {
		bar := k.geom.bar
		strokeRoundRect(screen, bar.x+2, bar.y+2, bar.w-4, bar.h-4, 6, 2.5, colSelect)
	}
	a.drawCheckers(screen, b)
	a.drawDice(screen)
}

func (a *App) drawHintPoint(screen *ebiten.Image, l boardLayout, point int, top bool) {
	x0, baseY, x1, tipX, tipY := l.pointTriangle(point, top, a.kit.geom.pointLen)
	var path vector.Path
	path.MoveTo(float32(x0), float32(baseY))
	path.LineTo(float32(x1), float32(baseY))
	path.LineTo(float32(tipX), float32(tipY))
	path.Close()
	var fill vector.DrawPathOptions
	fill.AntiAlias = true
	fill.ColorScale.ScaleWithColor(color.RGBA{0xff, 0xe0, 0x8a, 0x58})
	vector.FillPath(screen, &path, nil, &fill)
	var so vector.StrokeOptions
	so.Width = 2
	var stroke vector.DrawPathOptions
	stroke.AntiAlias = true
	stroke.ColorScale.ScaleWithColor(color.RGBA{0xff, 0xe8, 0xa0, 0xd0})
	vector.StrokePath(screen, &path, &so, &stroke)
}

func (a *App) drawSelectedPoint(screen *ebiten.Image, l boardLayout, point int, top bool) {
	x0, baseY, x1, tipX, tipY := l.pointTriangle(point, top, a.kit.geom.pointLen)
	vector.StrokeLine(screen, float32(x0), float32(baseY), float32(tipX), float32(tipY), 3, colSelect, true)
	vector.StrokeLine(screen, float32(x1), float32(baseY), float32(tipX), float32(tipY), 3, colSelect, true)
}

// drawChecker blits the sprite cut from the artwork, scaled to radius r,
// over a soft contact shadow (light from the top-left as in the reference).
func (k *boardKit) drawChecker(screen *ebiten.Image, x, y, r float64, c game.Color) {
	k.gpu()
	img := k.checkerImg[0]
	if c == game.Black {
		img = k.checkerImg[1]
	}
	if img == nil {
		drawChecker(screen, x, y, r, c)
		return
	}
	vector.DrawFilledCircle(screen, float32(x+r*0.1), float32(y+r*0.16), float32(r*1.02), color.RGBA{0, 0, 0, 90}, true)
	w := float64(img.Bounds().Dx())
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	s := 2 * r / w
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(x-r, y-r)
	screen.DrawImage(img, op)
}

// drawSlat shows a borne-off checker edge-on in the tray.
func (k *boardKit) drawSlat(screen *ebiten.Image, x, y float64, c game.Color) {
	w := k.geom.checkerR*2 - 6
	h := 7.0
	fill, line := colWhite, colWhiteLn
	if c == game.Black {
		fill, line = colBlack, colBlackLn
	}
	vector.DrawFilledRect(screen, float32(x-w/2+1), float32(y-h/2+2), float32(w), float32(h), color.RGBA{0, 0, 0, 90}, true)
	fillRoundRect(screen, x-w/2, y-h/2, w, h, 2.5, fill)
	vector.StrokeLine(screen, float32(x-w/2+3), float32(y-h/2+1.5), float32(x+w/2-3), float32(y-h/2+1.5), 1, line, true)
}

// drawDie paints a die in the kit's material: ivory for the Persian board,
// red and black lacquer (first and second die) for the Chinese board.
func (k *boardKit) drawDie(screen *ebiten.Image, x, y, size float64, n int, used bool, index int) {
	var fill, edge, pip, hi color.RGBA
	switch {
	case k.id == ThemeChinese && index == 0:
		fill, edge, pip, hi = color.RGBA{0x8c, 0x16, 0x1c, 0xff}, color.RGBA{0x4a, 0x08, 0x0c, 0xff}, color.RGBA{0xe6, 0xc0, 0x5e, 0xff}, color.RGBA{0xff, 0x80, 0x80, 0x40}
	case k.id == ThemeChinese:
		fill, edge, pip, hi = color.RGBA{0x1a, 0x18, 0x18, 0xff}, color.RGBA{0x05, 0x05, 0x05, 0xff}, color.RGBA{0xe6, 0xc0, 0x5e, 0xff}, color.RGBA{0xff, 0xff, 0xff, 0x30}
	default:
		fill, edge, pip, hi = color.RGBA{0xe9, 0xdc, 0xbe, 0xff}, color.RGBA{0xa8, 0x90, 0x64, 0xff}, color.RGBA{0x24, 0x1c, 0x16, 0xff}, color.RGBA{0xff, 0xff, 0xff, 0x70}
	}
	rad := size * 0.17
	// Shadow, body, bevel.
	fillRoundRect(screen, x+size*0.05, y+size*0.08, size, size, rad, color.RGBA{0, 0, 0, 100})
	fillRoundRect(screen, x, y, size, size, rad, edge)
	fillRoundRect(screen, x+1, y+1, size-2, size-2, rad-1, fill)
	// Top-left sheen, bottom-right shade.
	fillRoundRect(screen, x+2, y+2, size-4, size*0.45, rad-2, hi)
	fillRoundRect(screen, x+2, y+size*0.6, size-4, size*0.4-2, rad-2, color.RGBA{0, 0, 0, 0x22})
	scale := size / 28
	for _, p := range diePips(n) {
		px, py := float32(x+p[0]*scale), float32(y+p[1]*scale)
		pr := float32(2.7 * scale)
		vector.DrawFilledCircle(screen, px+pr*0.15, py+pr*0.2, pr, color.RGBA{0, 0, 0, 0x50}, true)
		vector.DrawFilledCircle(screen, px, py, pr, pip, true)
		vector.DrawFilledCircle(screen, px-pr*0.3, py-pr*0.3, pr*0.3, color.RGBA{0xff, 0xff, 0xff, 0x50}, true)
	}
	if used {
		fillRoundRect(screen, x, y, size, size, rad, color.RGBA{0x10, 0x0a, 0x06, 0x80})
	}
}

// drawKitCube paints the doubling cube as an ivory block in the kit's cube slot.
func (a *App) drawKitCube(screen *ebiten.Image, r button, hover bool) {
	x, y, s := r.x, r.y, r.w
	if hover && a.canOfferDouble() {
		strokeRoundRect(screen, x-4, y-4, s+8, s+8, 8, 2, colSelect)
	}
	fillRoundRect(screen, x+2, y+3, s, s, s*0.16, color.RGBA{0, 0, 0, 110})
	fillRoundRect(screen, x, y, s, s, s*0.16, color.RGBA{0xa8, 0x90, 0x64, 0xff})
	fillRoundRect(screen, x+1, y+1, s-2, s-2, s*0.16-1, color.RGBA{0xe9, 0xdc, 0xbe, 0xff})
	fillRoundRect(screen, x+2, y+2, s-4, s*0.42, s*0.14, color.RGBA{0xff, 0xff, 0xff, 0x60})
	face := fmt.Sprintf("%d", a.cube.Value)
	tw, th := text.Measure(face, a.score, 0)
	a.drawText(screen, face, a.score, x+(s-tw)/2, y+(s-th)/2, colInk)
	owner := "center"
	if a.cube.Owner != nil {
		owner = a.cube.Owner.String()
	}
	if a.match.Crawford {
		owner = "off"
	}
	ow, _ := text.Measure(owner, a.small, 0)
	a.drawText(screen, owner, a.small, x+(s-ow)/2, y+s+4, colMute)
	if hover && a.canOfferDouble() {
		dw, _ := text.Measure("Double", a.small, 0)
		a.drawText(screen, "Double", a.small, x+(s-dw)/2, y-22, colGold)
	}
}

// drawKitPips writes live pip counts into the board's own read-outs: the
// "You"/"Computer" plaques on the Persian board, the PIPS windows on the
// Chinese tray.
func (a *App) drawKitPips(screen *ebiten.Image) {
	g := a.kit.geom
	for _, c := range []game.Color{game.White, game.Black} {
		r := g.pipWhite
		title := "You"
		if c == game.Black {
			r = g.pipBlack
			title = a.opponentName()
			if a.player == nil {
				title = "Black"
			}
		}
		hover := a.hoverOwnScore(c)
		if hover {
			strokeRoundRect(screen, r.x-2, r.y-2, r.w+4, r.h+4, 6, 2, colGold)
		}
		label := fmt.Sprintf("%d", a.view().Pips(c))
		if g.plaques {
			tw, _ := text.Measure(title, a.small, 0)
			a.drawText(screen, title, a.small, r.x+(r.w-tw)/2, r.y+r.h*0.1, colGold)
			nw, _ := text.Measure(label, a.score, 0)
			a.drawText(screen, label, a.score, r.x+(r.w-nw)/2, r.y+r.h*0.34, colCream)
		} else {
			nw, nh := text.Measure(label, a.body, 0)
			a.drawText(screen, label, a.body, r.x+(r.w-nw)/2, r.y+(r.h-nh)/2, colCream)
		}
		if hover && a.canOfferDouble() {
			dw, _ := text.Measure("Double", a.small, 0)
			a.drawText(screen, "Double", a.small, r.x+(r.w-dw)/2, r.y+r.h+4, colGold)
		}
	}
}

// drawKitPreview shows the composed plate in the theme picker.
func (k *boardKit) drawPreview(screen *ebiten.Image, r button) {
	k.gpu()
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	pw, ph := float64(k.preview.Bounds().Dx()), float64(k.preview.Bounds().Dy())
	s := math.Min(r.w/pw, r.h/ph)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(r.x+(r.w-pw*s)/2, r.y+(r.h-ph*s)/2)
	screen.DrawImage(k.preview, op)
}

func roundRectPath(x, y, w, h, r float64) vector.Path {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	var p vector.Path
	X, Y, W, H, R := float32(x), float32(y), float32(w), float32(h), float32(r)
	p.MoveTo(X+R, Y)
	p.LineTo(X+W-R, Y)
	p.ArcTo(X+W, Y, X+W, Y+R, R)
	p.LineTo(X+W, Y+H-R)
	p.ArcTo(X+W, Y+H, X+W-R, Y+H, R)
	p.LineTo(X+R, Y+H)
	p.ArcTo(X, Y+H, X, Y+H-R, R)
	p.LineTo(X, Y+R)
	p.ArcTo(X, Y, X+R, Y, R)
	p.Close()
	return p
}

func fillRoundRect(screen *ebiten.Image, x, y, w, h, r float64, c color.RGBA) {
	p := roundRectPath(x, y, w, h, r)
	var op vector.DrawPathOptions
	op.AntiAlias = true
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(screen, &p, nil, &op)
}

func strokeRoundRect(screen *ebiten.Image, x, y, w, h, r, width float64, c color.RGBA) {
	p := roundRectPath(x, y, w, h, r)
	var so vector.StrokeOptions
	so.Width = float32(width)
	var op vector.DrawPathOptions
	op.AntiAlias = true
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(screen, &p, &so, &op)
}
