package ui

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// A board kit recreates one of the reference boards as separate layers:
// a frame cut from the artwork, a wood field tiled from a clean sample,
// point inlays placed per column by the layout, and checker sprites. The
// plate (frame + fields) is composed once on the CPU and uploaded as a
// single texture; interactive elements are drawn on top every frame.

type frect struct{ x, y, w, h float64 }

func (r frect) ir() image.Rectangle {
	return image.Rect(int(math.Round(r.x)), int(math.Round(r.y)), int(math.Round(r.x+r.w)), int(math.Round(r.y+r.h)))
}

func (r frect) btn(label string) button {
	return button{x: r.x, y: r.y, w: r.w, h: r.h, label: label}
}

func (r frect) contains(x, y float64) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

func (r frect) cx() float64 { return r.x + r.w/2 }
func (r frect) cy() float64 { return r.y + r.h/2 }

// kitGeom is the screen geometry of a composed board.
type kitGeom struct {
	board          frect
	fieldL, fieldR frect
	bar            frect
	tray           frect // bear-off target
	dice           frect // dice origin + size (w is one die)
	diceVertical   bool
	cube           frect
	pipWhite       frect // click-to-double / pip display for White
	pipBlack       frect
	plaques        bool // pip read-outs are titled plaques (else bare windows)
	statusTop      bool // status text in the strip above the board
	checkerR       float64
	pointLen       float64
	trayVertical   bool // borne-off checkers stack along y
	hud            kitHUD
}

type kitHUD struct {
	status, match, hint f2 // text anchors
	undo                frect
	hamburger           frect
}

type boardKit struct {
	id      ThemeID
	geom    kitGeom
	plate   *image.RGBA
	checker [2]*image.RGBA // White, Black at diameter 2·checkerR

	once       sync.Once
	plateImg   *ebiten.Image
	checkerImg [2]*ebiten.Image
	preview    *ebiten.Image
}

// gpu uploads the composed layers on first use.
func (k *boardKit) gpu() {
	k.once.Do(func() {
		k.plateImg = ebiten.NewImageFromImage(k.plate)
		for i := range k.checker {
			if k.checker[i] != nil {
				k.checkerImg[i] = ebiten.NewImageFromImage(k.checker[i])
			}
		}
		pv := newRGBA(200, 125)
		scaleDrawSrc(pv, pv.Bounds(), k.plate, k.plate.Bounds())
		k.preview = ebiten.NewImageFromImage(pv)
	})
}

var (
	kitMu    sync.Mutex
	kitCache = map[ThemeID]*boardKit{}
	kitErr   = map[ThemeID]error{}
)

// kitFor builds (once) and returns the kit for a plate theme, or nil for
// vector themes and on asset failure.
func kitFor(id ThemeID) *boardKit {
	kitMu.Lock()
	defer kitMu.Unlock()
	if k, ok := kitCache[id]; ok {
		return k
	}
	if _, failed := kitErr[id]; failed {
		return nil
	}
	var k *boardKit
	var err error
	switch id {
	case ThemePersian:
		k, err = buildPersianKit()
	case ThemeChinese:
		k, err = buildChineseKit()
	default:
		return nil
	}
	if err != nil {
		kitErr[id] = err
		return nil
	}
	kitCache[id] = k
	return k
}

// layout derives the interactive geometry from the composed board.
func (k *boardKit) layout() boardLayout {
	g := k.geom
	return boardLayout{
		feltX:  g.fieldL.x,
		feltY:  g.fieldL.y,
		feltW:  g.fieldR.x + g.fieldR.w - g.fieldL.x,
		feltH:  g.fieldL.h,
		barX:   g.bar.x,
		barW:   g.bar.w,
		leftX:  g.fieldL.x,
		rightX: g.fieldR.x,
		halfW:  g.fieldL.w,
		pointW: g.fieldL.w / 6,
		trayX:  g.tray.x,
		trayW:  g.tray.w,
		trayY:  g.tray.y,
		trayH:  g.tray.h,
		r:      g.checkerR,
	}
}

// fieldSpec describes how to fill one half of the playing surface.
type fieldSpec struct {
	wood      *image.RGBA     // clean wood sample, tiled with mirroring
	pointsTop [][]*image.RGBA // [colour][variant], base at the top edge
	pointsBot [][]*image.RGBA // [colour][variant], base at the bottom edge
	pointGap  float64         // horizontal gap between neighbouring points (screen px)
	pointLen  float64         // point length incl. ornament (screen px)
	medallion *image.RGBA     // optional centre inlay
	medalW    float64         // medallion width on screen
	shadow    [4]float64      // top, bottom, left, right inner-shadow strength
	shadowPx  float64
	vignette  float64
	// colourOf picks the point colour index for a column (0..5) and row.
	colourOf func(col int, top bool) int
}

// composeField paints wood, shading, medallion and twelve points into r.
func composeField(dst *image.RGBA, r image.Rectangle, s fieldSpec) {
	tileWood(dst, r, s.wood)
	if s.vignette > 0 {
		vignette(dst, r, s.vignette)
	}
	if s.medallion != nil && s.medalW > 0 {
		mb := s.medallion.Bounds()
		mh := s.medalW * float64(mb.Dy()) / float64(mb.Dx())
		cx, cy := float64(r.Min.X)+float64(r.Dx())/2, float64(r.Min.Y)+float64(r.Dy())/2
		dr := image.Rect(int(cx-s.medalW/2), int(cy-mh/2), int(cx+s.medalW/2), int(cy+mh/2))
		scaleDraw(dst, dr, s.medallion, mb)
	}
	pitch := float64(r.Dx()) / 6
	pw := pitch - s.pointGap
	for col := 0; col < 6; col++ {
		x0 := float64(r.Min.X) + pitch*float64(col) + s.pointGap/2
		for _, top := range []bool{true, false} {
			ci := 0
			if s.colourOf != nil {
				ci = s.colourOf(col, top)
			}
			set := s.pointsBot
			if top {
				set = s.pointsTop
			}
			if ci >= len(set) || len(set[ci]) == 0 {
				continue
			}
			spr := set[ci][col%len(set[ci])]
			var dr image.Rectangle
			if top {
				dr = image.Rect(int(math.Round(x0)), r.Min.Y, int(math.Round(x0+pw)), int(math.Round(float64(r.Min.Y)+s.pointLen)))
			} else {
				dr = image.Rect(int(math.Round(x0)), int(math.Round(float64(r.Max.Y)-s.pointLen)), int(math.Round(x0+pw)), r.Max.Y)
			}
			scaleDraw(dst, dr, spr, spr.Bounds())
		}
	}
	if s.shadowPx > 0 {
		innerShadow(dst, r, s.shadow[0], s.shadow[1], s.shadow[2], s.shadow[3], s.shadowPx)
	}
}

// tileWood fills r with the sample scaled to r's width and mirror-tiled
// vertically, centred so one unbroken band sits across the middle.
func tileWood(dst *image.RGBA, r image.Rectangle, wood *image.RGBA) {
	wb := wood.Bounds()
	bandH := int(math.Round(float64(r.Dx()) * float64(wb.Dy()) / float64(wb.Dx())))
	if bandH < 1 {
		bandH = 1
	}
	band := newRGBA(r.Dx(), bandH)
	scaleDrawSrc(band, band.Bounds(), wood, wb)
	flipped := flipV(band)
	cy := r.Min.Y + r.Dy()/2
	start := cy - bandH/2
	// rows above
	for i, y := 0, start; y+bandH > r.Min.Y; i, y = i+1, y-bandH {
		src := band
		if i%2 == 1 {
			src = flipped
		}
		blit(dst, r, image.Pt(r.Min.X, y), src)
	}
	for i, y := 1, start+bandH; y < r.Max.Y; i, y = i+1, y+bandH {
		src := band
		if i%2 == 1 {
			src = flipped
		}
		blit(dst, r, image.Pt(r.Min.X, y), src)
	}
}

// blit copies src at `at`, clipped to clip.
func blit(dst *image.RGBA, clip image.Rectangle, at image.Point, src *image.RGBA) {
	dr := image.Rectangle{Min: at, Max: at.Add(src.Bounds().Size())}.Intersect(clip)
	if dr.Empty() {
		return
	}
	sp := src.Bounds().Min.Add(dr.Min.Sub(at))
	scaleDrawSrc(dst, dr, src, image.Rectangle{Min: sp, Max: sp.Add(dr.Size())})
}

// checkerSprite cuts a circular checker out of the artwork and resamples it
// to the layout's diameter.
func checkerSprite(src *image.RGBA, cx, cy, r float64, diameter int) *image.RGBA {
	x0, y0 := int(math.Floor(cx-r-1)), int(math.Floor(cy-r-1))
	x1, y1 := int(math.Ceil(cx+r+1)), int(math.Ceil(cy+r+1))
	sr := image.Rect(x0, y0, x1, y1)
	m := blurAlpha(circleMask(sr.Dx(), sr.Dy(), cx-float64(x0), cy-float64(y0), r-0.5), 1)
	spr := cutMasked(src, sr, m)
	out := newRGBA(diameter, diameter)
	scaleDrawSrc(out, out.Bounds(), spr, spr.Bounds())
	return out
}

// backgroundPlate paints the table around the board: a dark surround with a
// soft pool of light under the board, as in both reference photos.
func backgroundPlate(dst *image.RGBA, base color.RGBA, board frect) {
	fillRGBA(dst, dst.Bounds(), base)
	b := dst.Bounds()
	cx, cy := board.cx(), board.cy()
	rx, ry := board.w*0.75, board.h*0.75
	shade(dst, b, 0.35, func(x, y float64) float64 {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		d := math.Sqrt(dx*dx + dy*dy)
		if d > 1 {
			return 0
		}
		return (1 - d) * (1 - d)
	})
	// Soft drop shadow around the board edge.
	pad := 18.0
	shade(dst, image.Rect(int(board.x-pad), int(board.y-pad), int(board.x+board.w+pad), int(board.y+board.h+pad)), -0.6, func(x, y float64) float64 {
		w, h := board.w+2*pad, board.h+2*pad
		dx := math.Min(x, w-x)
		dy := math.Min(y, h-y)
		d := math.Min(dx, dy)
		if d >= pad {
			return 1
		}
		t := d / pad
		return t * t
	})
}
