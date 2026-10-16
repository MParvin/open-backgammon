package ui

import (
	"image"
	"image/color"
	"math"

	"bg/assets"
)

// Persian kit — recreates assets/board/persian_ornate.jpg (1536×1024).
//
// The artwork is used as a sprite atlas: the frame with its inlaid bands,
// bird panels, hinges, dice well and score plaques is cut as one layer; the
// two playing fields are rebuilt from a clean wood sample, the central
// medallion, and six point inlays per quadrant (the painting itself shows
// seven irregular columns per half, so points are re-laid on the layout).

const (
	prsW, prsH = 1536.0, 1024.0
	prsTop     = 32.0 // status strip above the board
	prsBottom  = 32.0 // match / hint strip below
)

func buildPersianKit() (*boardKit, error) {
	src, err := decodeRGBA(assets.PersianOrnateBoard)
	if err != nil {
		return nil, err
	}
	// Fit the 3:2 plate between the HUD strips, centred horizontally.
	bh := float64(screenH) - prsTop - prsBottom
	bw := bh * prsW / prsH
	board := frect{(float64(screenW) - bw) / 2, prsTop, bw, bh}
	s := bh / prsH
	X := func(v float64) float64 { return board.x + v*s }
	Y := func(v float64) float64 { return board.y + v*s }
	S := func(v float64) float64 { return v * s }
	R := func(x0, y0, x1, y1 float64) frect { return frect{X(x0), Y(y0), S(x1 - x0), S(y1 - y0)} }

	g := kitGeom{
		board:        board,
		fieldL:       R(195, 60, 735, 962),
		fieldR:       R(800, 60, 1340, 962),
		bar:          R(735, 60, 800, 962),
		tray:         R(1398, 218, 1508, 988),
		trayVertical: true,
		diceVertical: true,
		pipWhite:     R(32, 850, 130, 935),
		pipBlack:     R(1400, 105, 1505, 205),
		plaques:      true,
		statusTop:    true,
		checkerR:     math.Round(29.5 * s),
	}
	well := R(36, 130, 124, 365)
	die := math.Round(well.w * 0.78)
	gap := math.Round(die * 0.18)
	g.dice = frect{well.cx() - die/2, well.cy() - die - gap/2, die, die}
	gutter := board.x
	g.cube = frect{board.x + board.w + (gutter-56)/2, 372, 56, 56}
	g.pointLen = math.Round(0.35 * g.fieldL.h)
	g.hud = kitHUD{
		status:    f2{76, 3},
		match:     f2{200, 775},
		hint:      f2{720, 775},
		undo:      frect{board.x, 769, 96, 28},
		hamburger: frect{(gutter - 44) / 2, 40, 44, 44},
	}

	plate := newRGBA(screenW, screenH)
	backgroundPlate(plate, color.RGBA{0x16, 0x0d, 0x08, 0xff}, board)
	// 1. Frame layer: the whole painting, uniformly scaled (fields are
	//    overdrawn below, so the checkers in the artwork never show).
	scaleDraw(plate, board.ir(), src, src.Bounds())

	// 2. Clear the dice well and the two score plaques of their baked-in
	//    glyphs so live values can be drawn there.
	wellCol := avgColor(src, rect(44, 290, 70, 350))
	fillMasked(plate, well.ir().Min, blurAlpha(polygonMask(well.ir().Dx(), well.ir().Dy(), roundRectPoly(1.5, 1.5, well.w-1.5, well.h-1.5, S(12))), 1), wellCol)
	plaqueCol := avgColor(src, rect(38, 858, 60, 880))
	for _, p := range []frect{g.pipWhite, g.pipBlack} {
		ir := p.ir()
		fillMasked(plate, ir.Min, blurAlpha(polygonMask(ir.Dx(), ir.Dy(), octagon(1, 1, p.w-1, p.h-1, S(9))), 1), plaqueCol)
	}

	// 3. Fields.
	woodL := cut(src, rect(205, 380, 335, 650))
	woodR := cut(src, rect(595, 380, 725, 650))
	wood := hcat(woodL, flipH(woodL), flipH(woodR), woodR)

	medR := rect(322, 342, 608, 668)
	medMask := blurAlpha(polygonMask(medR.Dx(), medR.Dy(), roundRectPoly(10, 10, float64(medR.Dx())-10, float64(medR.Dy())-10, 70)), 4)
	medallion := cutMasked(src, medR, medMask)

	// Point inlays: red and dark variants for each orientation.
	redTop := persianPoint(src, rect(330, 60, 414, 380), 338, 405, 60, 372, 336, 14, 374)
	darkTop := persianPoint(src, rect(398, 60, 480, 380), 405, 471, 60, 439, 330, 14, 366)
	redBot := persianPoint(src, rect(330, 600, 414, 962), 338, 405, 962, 373, 650, 14, 612)
	darkBot := persianPoint(src, rect(398, 600, 480, 962), 405, 473, 962, 440, 650, 14, 615)

	spec := fieldSpec{
		wood:      wood,
		pointsTop: [][]*image.RGBA{{redTop}, {darkTop}},
		pointsBot: [][]*image.RGBA{{redBot}, {darkBot}},
		pointGap:  math.Round(g.fieldL.w / 6 * 0.09),
		pointLen:  g.pointLen,
		medallion: medallion,
		medalW:    S(float64(medR.Dx())),
		shadow:    [4]float64{0.55, 0.35, 0.3, 0.6},
		shadowPx:  S(22),
		vignette:  0.12,
		// Reference: in both rows the column beside the frame is dark,
		// then red, alternating towards the bar.
		colourOf: func(col int, _ bool) int { return (col + 1) % 2 },
	}
	composeField(plate, g.fieldL.ir(), spec)
	composeField(plate, g.fieldR.ir(), spec)

	k := &boardKit{id: ThemePersian, geom: g, plate: plate}
	d := int(2 * g.checkerR)
	k.checker[0] = checkerSprite(src, 235, 100, 29.5, d)
	k.checker[1] = checkerSprite(src, 650, 100, 29.5, d)
	return k, nil
}

// persianPoint cuts one inlaid point (triangle plus the drop ornament past
// its tip) out of the painting. Coordinates are in source pixels.
func persianPoint(src *image.RGBA, r image.Rectangle, bl, br, by, tx, ty, spikeW, spikeEnd float64) *image.RGBA {
	ox, oy := float64(r.Min.X), float64(r.Min.Y)
	m := triangleMask(r.Dx(), r.Dy(), bl-ox, br-ox, by-oy, tx-ox, ty-oy, 3, spikeW, spikeEnd-oy)
	return cutMasked(src, r, m)
}

func octagon(x0, y0, x1, y1, c float64) []f2 {
	return []f2{{x0 + c, y0}, {x1 - c, y0}, {x1, y0 + c}, {x1, y1 - c}, {x1 - c, y1}, {x0 + c, y1}, {x0, y1 - c}, {x0, y0 + c}}
}
