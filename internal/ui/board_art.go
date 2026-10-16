package ui

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"bg/assets"
	"bg/internal/game"
)

// Reference plate is 1536×1024. These UVs place interactive regions on the
// artwork measured from Ornate Persian Backgammon Board.png.
const (
	refW = 1536.0
	refH = 1024.0

	uvFeltY  = 58.0 / refH
	uvFeltH  = 900.0 / refH
	uvLeftX  = 168.0 / refW
	uvHalfW  = 570.0 / refW
	uvBarX   = 738.0 / refW
	uvBarW   = 50.0 / refW
	uvRightX = 788.0 / refW
	uvTrayX  = 1392.0 / refW
	uvTrayW  = 118.0 / refW

	uvDiceX = 42.0 / refW
	uvDiceY = 168.0 / refH
	uvDiceW = 90.0 / refW
	uvDiceH = 200.0 / refH

	uvYouX = 38.0 / refW
	uvYouY = 872.0 / refH
	uvYouW = 140.0 / refW
	uvYouH = 100.0 / refH

	uvCompX = 1360.0 / refW
	uvCompY = 48.0 / refH
	uvCompW = 145.0 / refW
	uvCompH = 115.0 / refH
)

var (
	persianPlateOnce sync.Once
	persianPlate     *ebiten.Image
	persianPlateErr  error
)

func loadPersianPlate() (*ebiten.Image, error) {
	persianPlateOnce.Do(func() {
		img, _, err := image.Decode(bytes.NewReader(assets.PersianOrnateBoard))
		if err != nil {
			persianPlateErr = err
			return
		}
		persianPlate = ebiten.NewImageFromImage(img)
	})
	return persianPlate, persianPlateErr
}

// artPlacement is where the reference plate is drawn on the logical screen.
type artPlacement struct {
	x, y, w, h float64
}

func (p artPlacement) mapX(uv float64) float64 { return p.x + uv*p.w }
func (p artPlacement) mapY(uv float64) float64 { return p.y + uv*p.h }
func (p artPlacement) mapW(uv float64) float64 { return uv * p.w }
func (p artPlacement) mapH(uv float64) float64 { return uv * p.h }

func (a *App) persianPlacement() artPlacement {
	// Fit the 3:2 plate into 1280×800, letterboxing on the sides.
	const aspect = refW / refH
	h := float64(screenH)
	w := h * aspect
	if w > float64(screenW) {
		w = float64(screenW)
		h = w / aspect
	}
	return artPlacement{
		x: (float64(screenW) - w) / 2,
		y: (float64(screenH) - h) / 2,
		w: w, h: h,
	}
}

func (a *App) applyPersianLayout() {
	p := a.persianPlacement()
	a.layout = boardLayout{
		feltX:  p.mapX(uvLeftX),
		feltY:  p.mapY(uvFeltY),
		feltW:  p.mapW(uvHalfW*2 + uvBarW),
		feltH:  p.mapH(uvFeltH),
		barX:   p.mapX(uvBarX),
		barW:   p.mapW(uvBarW),
		leftX:  p.mapX(uvLeftX),
		rightX: p.mapX(uvRightX),
		halfW:  p.mapW(uvHalfW),
		pointW: p.mapW(uvHalfW) / 6,
		trayX:  p.mapX(uvTrayX),
		trayW:  p.mapW(uvTrayW),
	}
	a.art = p
}

func (a *App) resetDefaultLayout() {
	a.layout = newLayout()
	a.art = artPlacement{}
}

func (a *App) ornatePersian() bool {
	return a.theme.ID == ThemePersian
}

func (a *App) drawPersianBoard(screen *ebiten.Image, b game.Board) {
	plate, err := loadPersianPlate()
	if err != nil || plate == nil {
		a.drawBoardFallback(screen, b)
		return
	}
	p := a.persianPlacement()
	a.art = p
	vector.DrawFilledRect(screen, 0, 0, screenW, screenH, color.RGBA{0x12, 0x0a, 0x06, 0xff}, false)
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	sw, sh := plate.Bounds().Dx(), plate.Bounds().Dy()
	op.GeoM.Scale(p.w/float64(sw), p.h/float64(sh))
	op.GeoM.Translate(p.x, p.y)
	screen.DrawImage(plate, op)

	// Soft recess shading so checkers read above the wood.
	l := a.layout
	vector.StrokeRect(screen, float32(l.feltX), float32(l.feltY), float32(l.feltW), float32(l.feltH), 1.2, color.RGBA{0, 0, 0, 55}, true)

	if a.hintOff() {
		vector.StrokeRect(screen, float32(l.trayX), float32(l.feltY), float32(l.trayW), float32(l.feltH), 3, colSelect, true)
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
	a.drawCheckers(screen, b)
	a.drawDice(screen)
}

func (a *App) drawBoardFallback(screen *ebiten.Image, b game.Board) {
	a.drawBoardClassic(screen, b)
}

func (a *App) drawHintPoint(screen *ebiten.Image, l boardLayout, point int, top bool) {
	col, _ := l.slot(point)
	x := l.colX(col)
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
	draw.ColorScale.ScaleWithColor(color.RGBA{0xff, 0xe0, 0x8a, 0x66})
	vector.FillPath(screen, &path, nil, &draw)
}

func (a *App) drawSelectedPoint(screen *ebiten.Image, l boardLayout, point int, top bool) {
	col, _ := l.slot(point)
	x := l.colX(col)
	h := l.feltH/2 - 18
	tipY := l.feltY + h
	baseY := l.feltY + 8
	if !top {
		baseY = l.feltY + l.feltH - 8
		tipY = baseY - h
	}
	vector.StrokeLine(screen, float32(x+6), float32(baseY), float32(x+l.pointW/2), float32(tipY), 3, colSelect, true)
	vector.StrokeLine(screen, float32(x+l.pointW-6), float32(baseY), float32(x+l.pointW/2), float32(tipY), 3, colSelect, true)
}

func (a *App) persianDiceOrigin() (float64, float64, float64) {
	p := a.art
	if p.w == 0 {
		p = a.persianPlacement()
	}
	x := p.mapX(uvDiceX)
	y := p.mapY(uvDiceY)
	size := p.mapW(uvDiceW)
	if size < 36 {
		size = 36
	}
	return x, y, size
}

func (a *App) persianScoreRect(c game.Color) button {
	p := a.art
	if p.w == 0 {
		p = a.persianPlacement()
	}
	if c == game.White {
		return button{
			x: p.mapX(uvYouX), y: p.mapY(uvYouY),
			w: p.mapW(uvYouW), h: p.mapH(uvYouH),
			label: "White score",
		}
	}
	return button{
		x: p.mapX(uvCompX), y: p.mapY(uvCompY),
		w: p.mapW(uvCompW), h: p.mapH(uvCompH),
		label: "Black score",
	}
}
