package ui

import (
	"image"
	"image/color"
	"math"

	"bg/assets"
)

// Chinese kit — recreates assets/board/chinese1.jpg.
//
// The reference is a perspective render (the near edge is ~20% wider than
// the far edge). The board plane is un-warped with a homography so that the
// frame, trays, silver brackets, carved bar and hinges become a flat plate
// at screen resolution. Teal-and-gold point inlays, the dragon and phoenix
// checkers and the clean walnut field are cut from that flat plate; the
// fields are then rebuilt with six points per quadrant on a centred bar.

// Un-warped ("U") coordinate system: the field plane quad of the source maps
// to the rectangle uField; the board's outer edge lands on uBoard.
var (
	chnQuad  = [4]f2{{212, 68}, {800, 68}, {852, 514}, {150, 514}} // TL TR BR BL in source px
	chnField = frect{200, 120, 702, 446}
	chnBoard = frect{93.6, 71.4, 911.3, 513.7}
)

const (
	chnGutter = 40.0 // minimum side margin on screen
)

func buildChineseKit() (*boardKit, error) {
	src, err := decodeRGBA(assets.ChineseBoard)
	if err != nil {
		return nil, err
	}
	bw := float64(screenW) - 2*chnGutter
	k := bw / chnBoard.w
	bh := chnBoard.h * k
	board := frect{chnGutter, math.Round((float64(screenH) - bh) / 2), bw, bh}
	X := func(u float64) float64 { return board.x + (u-chnBoard.x)*k }
	Y := func(u float64) float64 { return board.y + (u-chnBoard.y)*k }
	S := func(u float64) float64 { return u * k }
	R := func(x0, y0, x1, y1 float64) frect { return frect{X(x0), Y(y0), S(x1 - x0), S(y1 - y0)} }

	field := R(chnField.x, chnField.y, chnField.x+chnField.w, chnField.y+chnField.h)
	halfW := math.Round(field.w * 0.46)
	barW := field.w - 2*halfW
	g := kitGeom{
		board:        board,
		fieldL:       frect{field.x, field.y, halfW, field.h},
		bar:          frect{field.x + halfW, field.y, barW, field.h},
		fieldR:       frect{field.x + halfW + barW, field.y, halfW, field.h},
		tray:         R(939, 102, 989, 560),
		trayVertical: true,
		pipWhite:     R(137, 299, 173, 321),
		pipBlack:     R(137, 327, 173, 349),
		statusTop:    true,
		checkerR:     math.Round(20.5 * k),
	}
	g.pointLen = math.Round(0.39 * field.h)
	die := math.Round(g.fieldL.w / 6 * 0.74)
	g.dice = frect{g.fieldR.cx() - die - 6, g.fieldR.cy() - die/2, die, die}
	g.cube = frect{g.tray.cx() - 26, g.tray.cy() - 26, 52, 52}
	g.hud = kitHUD{
		status:    f2{76, 18},
		match:     f2{130, 757},
		hint:      f2{720, 757},
		undo:      frect{16, 752, 96, 32},
		hamburger: frect{16, 9, 44, 44},
	}

	// Homography: screen field rect → source quad.
	dstQuad := [4]f2{{field.x, field.y}, {field.x + field.w, field.y}, {field.x + field.w, field.y + field.h}, {field.x, field.y + field.h}}
	h := homography(dstQuad, chnQuad)

	// Frame layer: warp the board plane straight into screen pixels.
	frame := image.NewRGBA(board.ir())
	warpPerspective(frame, h, src)
	unsharp(frame, 0.35)
	// Soften the outer edge (the painted box has a bevel, not a hard cut).
	fb := frame.Bounds()
	edge := blurAlpha(polygonMask(fb.Dx(), fb.Dy(), roundRectPoly(1, 1, float64(fb.Dx())-1, float64(fb.Dy())-1, S(14))), 1)
	frame = applyMask(frame, edge) // now origin-based; frameAt restores screen placement
	frameAt := fb.Min
	F := func(x0, y0, x1, y1 float64) image.Rectangle { // U-space → frame-local
		return image.Rect(int(math.Round(X(x0)-float64(frameAt.X))), int(math.Round(Y(y0)-float64(frameAt.Y))), int(math.Round(X(x1)-float64(frameAt.X))), int(math.Round(Y(y1)-float64(frameAt.Y))))
	}

	// Sprites, cut before the fields are rebuilt.
	d := int(2 * g.checkerR)
	red := checkerSprite(frame, X(232)-float64(frameAt.X), Y(510)-float64(frameAt.Y), S(20.5), d)
	black := checkerSprite(frame, X(232)-float64(frameAt.X), Y(387)-float64(frameAt.Y), S(20.5), d)
	bar := cut(frame, F(524, 71.4, 598, 585.1))
	woodR := cut(frame, F(700, 305, 900, 390))
	wood := hcat(woodR, flipH(woodR))
	point := func(x0, y0, x1, y1, bl, br, by, tx, ty float64) *image.RGBA {
		r := F(x0, y0, x1, y1)
		ox, oy := float64(r.Min.X), float64(r.Min.Y)
		m := triangleMask(r.Dx(), r.Dy(), X(bl)-float64(frameAt.X)-ox, X(br)-float64(frameAt.X)-ox, Y(by)-float64(frameAt.Y)-oy,
			X(tx)-float64(frameAt.X)-ox, Y(ty)-float64(frameAt.Y)-oy, S(1.5), 0, 0)
		return cutMasked(frame, r, m)
	}
	topPts := []*image.RGBA{
		point(260, 118, 321, 310, 263, 318, 120, 290, 302),
		point(315, 118, 375, 310, 318, 372, 120, 345, 302),
		point(371, 118, 430, 310, 374, 427, 120, 400, 302),
	}
	botPts := []*image.RGBA{
		point(260, 388, 321, 568, 263, 318, 566, 291, 395),
		point(315, 388, 375, 568, 318, 372, 566, 345, 395),
		point(371, 388, 430, 568, 374, 427, 566, 400, 395),
	}

	// Patches on the frame: the painted cube in the right tray and the
	// PIPS read-outs are replaced by clean tray wood / dark glass.
	trayPatch := cut(frame, F(936, 150, 992, 240))
	scaleDraw(frame, F(936, 272, 992, 362), trayPatch, trayPatch.Bounds())
	glass := avgColor(frame, F(150, 303, 165, 318))
	for _, p := range []frect{g.pipWhite, g.pipBlack} {
		ir := p.ir().Sub(frameAt)
		fillMasked(frame, ir.Min, blurAlpha(polygonMask(ir.Dx(), ir.Dy(), roundRectPoly(0.5, 0.5, p.w-0.5, p.h-0.5, 3)), 1), glass)
	}

	plate := newRGBA(screenW, screenH)
	backgroundPlate(plate, color.RGBA{0x15, 0x10, 0x0c, 0xff}, board)
	drawOver(plate, frameAt, frame)

	spec := fieldSpec{
		wood:      wood,
		pointsTop: [][]*image.RGBA{topPts},
		pointsBot: [][]*image.RGBA{botPts},
		pointGap:  0,
		pointLen:  g.pointLen,
		shadow:    [4]float64{0.5, 0.4, 0.35, 0.45},
		shadowPx:  S(14),
		vignette:  0.08,
	}
	composeField(plate, g.fieldL.ir(), spec)
	composeField(plate, g.fieldR.ir(), spec)
	// Centred bar with its carved panel and hinges, full board height.
	bb := bar.Bounds()
	barX := int(math.Round(g.bar.cx() - float64(bb.Dx())/2))
	drawOver(plate, image.Pt(barX, int(math.Round(board.y))), bar)

	kit := &boardKit{id: ThemeChinese, geom: g, plate: plate}
	kit.checker[0] = red
	kit.checker[1] = black
	return kit, nil
}
