package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// btnStyle distinguishes control hierarchy.
type btnStyle int

const (
	btnSecondary btnStyle = iota
	btnPrimary
	btnDanger
	btnAccept
	btnReject
	btnGhost
)

func styleColors(s btnStyle) (fill, border, label color.RGBA) {
	switch s {
	case btnPrimary:
		return color.RGBA{0x6a, 0x48, 0x24, 0xff}, colGold, colCream
	case btnDanger:
		return color.RGBA{0x6a, 0x28, 0x28, 0xff}, color.RGBA{0xe8, 0x80, 0x70, 0xff}, colCream
	case btnAccept:
		return color.RGBA{0x2a, 0x6a, 0x3a, 0xff}, color.RGBA{0x70, 0xe0, 0x90, 0xff}, colCream
	case btnReject:
		return color.RGBA{0x7a, 0x28, 0x28, 0xff}, color.RGBA{0xf0, 0x90, 0x80, 0xff}, colCream
	case btnGhost:
		return color.RGBA{0x28, 0x1c, 0x14, 0xaa}, colMute, colMute
	default:
		return color.RGBA{0x48, 0x32, 0x1c, 0xff}, colGoldDk, colCream
	}
}

func (a *App) drawStyledButton(screen *ebiten.Image, b button, focused bool) {
	fill, border, label := styleColors(b.style)
	x, y := ebiten.CursorPosition()
	hover := inRect(float64(x), float64(y), b)
	if hover {
		fill = lighten(fill, 18)
	}
	if focused {
		border = colSelect
	}
	r := float32(10)
	drawRoundRect(screen, float32(b.x), float32(b.y), float32(b.w), float32(b.h), r, fill)
	vector.StrokeRect(screen, float32(b.x), float32(b.y), float32(b.w), float32(b.h), 2, border, true)

	face := a.body
	if b.label == "Roll" {
		face = a.score
	}
	tw, th := text.Measure(b.label, face, 0)
	iconW := 0.0
	if b.style == btnAccept || b.style == btnReject || b.style == btnDanger {
		iconW = 28
	}
	tx := b.x + (b.w-tw-iconW)/2 + iconW
	ty := b.y + (b.h-th)/2
	if b.style == btnAccept {
		drawCheckIcon(screen, b.x+14, b.y+b.h/2, 10)
	}
	if b.style == btnReject || b.style == btnDanger {
		drawFlagIcon(screen, b.x+14, b.y+b.h/2-2, 12)
	}
	a.drawText(screen, b.label, face, tx, ty, label)
}

func lighten(c color.RGBA, n uint8) color.RGBA {
	add := func(v, d uint8) uint8 {
		if int(v)+int(d) > 255 {
			return 255
		}
		return v + d
	}
	return color.RGBA{add(c.R, n), add(c.G, n), add(c.B, n), c.A}
}

func drawRoundRect(screen *ebiten.Image, x, y, w, h, _ float32, fill color.RGBA) {
	vector.DrawFilledRect(screen, x, y, w, h, fill, true)
}

// drawCheckIcon draws a white checkmark centered near (cx, cy).
func drawCheckIcon(screen *ebiten.Image, cx, cy, s float64) {
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	vector.StrokeLine(screen, float32(cx-s*0.4), float32(cy), float32(cx-s*0.05), float32(cy+s*0.4), 3, white, true)
	vector.StrokeLine(screen, float32(cx-s*0.05), float32(cy+s*0.4), float32(cx+s*0.55), float32(cy-s*0.45), 3, white, true)
}

// drawFlagIcon draws a small white flag (pole + cloth), not a text glyph.
func drawFlagIcon(screen *ebiten.Image, cx, cy, s float64) {
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	vector.StrokeLine(screen, float32(cx-s*0.35), float32(cy-s*0.55), float32(cx-s*0.35), float32(cy+s*0.55), 2.5, white, true)
	var path vector.Path
	path.MoveTo(float32(cx-s*0.3), float32(cy-s*0.5))
	path.LineTo(float32(cx+s*0.55), float32(cy-s*0.25))
	path.LineTo(float32(cx-s*0.3), float32(cy))
	path.Close()
	var opts vector.DrawPathOptions
	opts.AntiAlias = true
	opts.ColorScale.ScaleWithColor(white)
	vector.FillPath(screen, &path, nil, &opts)
}
