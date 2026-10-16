package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

// CPU-side raster helpers used to cut sprites out of the reference artwork
// and compose a board plate. Everything here works on premultiplied
// *image.RGBA and *image.Alpha so the result can be uploaded once to the GPU.

type f2 struct{ x, y float64 }

func rect(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }

func decodeRGBA(data []byte) (*image.RGBA, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	return out, nil
}

func newRGBA(w, h int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, w, h)) }

// cut copies r from src into a new image with origin (0,0).
func cut(src *image.RGBA, r image.Rectangle) *image.RGBA {
	out := newRGBA(r.Dx(), r.Dy())
	draw.Draw(out, out.Bounds(), src, r.Min, draw.Src)
	return out
}

// applyMask multiplies img's premultiplied channels by m.
func applyMask(img *image.RGBA, m *image.Alpha) *image.RGBA {
	b := img.Bounds()
	out := newRGBA(b.Dx(), b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			a := uint32(m.AlphaAt(x, y).A)
			i := img.PixOffset(b.Min.X+x, b.Min.Y+y)
			o := out.PixOffset(x, y)
			p := img.Pix[i : i+4 : i+4]
			q := out.Pix[o : o+4 : o+4]
			q[0] = uint8(uint32(p[0]) * a / 255)
			q[1] = uint8(uint32(p[1]) * a / 255)
			q[2] = uint8(uint32(p[2]) * a / 255)
			q[3] = uint8(uint32(p[3]) * a / 255)
		}
	}
	return out
}

// cutMasked cuts r out of src and applies a mask given in r-local coordinates.
func cutMasked(src *image.RGBA, r image.Rectangle, m *image.Alpha) *image.RGBA {
	return applyMask(cut(src, r), m)
}

// polygonMask rasterizes an anti-aliased polygon (local coords) into an alpha mask.
func polygonMask(w, h int, pts []f2) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	if len(pts) < 3 {
		return m
	}
	rz := vector.NewRasterizer(w, h)
	rz.MoveTo(float32(pts[0].x), float32(pts[0].y))
	for _, p := range pts[1:] {
		rz.LineTo(float32(p.x), float32(p.y))
	}
	rz.ClosePath()
	rz.Draw(m, m.Bounds(), image.Opaque, image.Point{})
	return m
}

func unionMask(a, b *image.Alpha) *image.Alpha {
	out := image.NewAlpha(a.Bounds())
	for i := range a.Pix {
		v := int(a.Pix[i]) + int(b.Pix[i])
		if v > 255 {
			v = 255
		}
		out.Pix[i] = uint8(v)
	}
	return out
}

func ellipsePoly(cx, cy, rx, ry float64, n int) []f2 {
	pts := make([]f2, 0, n)
	for i := 0; i < n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		pts = append(pts, f2{cx + rx*math.Cos(t), cy + ry*math.Sin(t)})
	}
	return pts
}

func circleMask(w, h int, cx, cy, r float64) *image.Alpha {
	return polygonMask(w, h, ellipsePoly(cx, cy, r, r, 96))
}

// roundRectPoly returns a rounded rectangle outline.
func roundRectPoly(x0, y0, x1, y1, rad float64) []f2 {
	const seg = 10
	var pts []f2
	arc := func(cx, cy, a0 float64) {
		for i := 0; i <= seg; i++ {
			t := a0 + (math.Pi/2)*float64(i)/seg
			pts = append(pts, f2{cx + rad*math.Cos(t), cy + rad*math.Sin(t)})
		}
	}
	arc(x1-rad, y0+rad, -math.Pi/2)
	arc(x1-rad, y1-rad, 0)
	arc(x0+rad, y1-rad, math.Pi/2)
	arc(x0+rad, y0+rad, math.Pi)
	return pts
}

// blurAlpha softens a mask with repeated box blurs (≈ Gaussian).
func blurAlpha(m *image.Alpha, radius int) *image.Alpha {
	if radius <= 0 {
		return m
	}
	w, h := m.Bounds().Dx(), m.Bounds().Dy()
	src := m
	for pass := 0; pass < 3; pass++ {
		tmp := image.NewAlpha(m.Bounds())
		// horizontal
		for y := 0; y < h; y++ {
			sum := 0
			for x := -radius; x <= radius; x++ {
				sum += int(alphaAt(src, x, y, w, h))
			}
			for x := 0; x < w; x++ {
				tmp.Pix[y*tmp.Stride+x] = uint8(sum / (2*radius + 1))
				sum += int(alphaAt(src, x+radius+1, y, w, h)) - int(alphaAt(src, x-radius, y, w, h))
			}
		}
		out := image.NewAlpha(m.Bounds())
		for x := 0; x < w; x++ {
			sum := 0
			for y := -radius; y <= radius; y++ {
				sum += int(alphaAt(tmp, x, y, w, h))
			}
			for y := 0; y < h; y++ {
				out.Pix[y*out.Stride+x] = uint8(sum / (2*radius + 1))
				sum += int(alphaAt(tmp, x, y+radius+1, w, h)) - int(alphaAt(tmp, x, y-radius, w, h))
			}
		}
		src = out
	}
	return src
}

func alphaAt(m *image.Alpha, x, y, w, h int) uint8 {
	if x < 0 || y < 0 || x >= w || y >= h {
		return 0
	}
	return m.Pix[y*m.Stride+x]
}

// shrinkPoly pulls each vertex toward (cx, cy) by `by` pixels so a later
// blur/feather stays inside the original outline.
func shrinkPoly(pts []f2, cx, cy, by float64) []f2 {
	out := make([]f2, len(pts))
	for i, p := range pts {
		dx, dy := p.x-cx, p.y-cy
		d := math.Hypot(dx, dy)
		if d < 1e-6 {
			out[i] = p
			continue
		}
		k := (d - by) / d
		if k < 0 {
			k = 0
		}
		out[i] = f2{cx + dx*k, cy + dy*k}
	}
	return out
}

// scaleDraw resamples sr of src into dr of dst (Catmull-Rom, source-over).
func scaleDraw(dst *image.RGBA, dr image.Rectangle, src image.Image, sr image.Rectangle) {
	xdraw.CatmullRom.Scale(dst, dr, src, sr, xdraw.Over, nil)
}

// scaleDrawSrc is scaleDraw with a copy (no blending).
func scaleDrawSrc(dst *image.RGBA, dr image.Rectangle, src image.Image, sr image.Rectangle) {
	xdraw.CatmullRom.Scale(dst, dr, src, sr, xdraw.Src, nil)
}

func flipH(img *image.RGBA) *image.RGBA {
	b := img.Bounds()
	out := newRGBA(b.Dx(), b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.SetRGBA(x, y, img.RGBAAt(b.Min.X+b.Dx()-1-x, b.Min.Y+y))
		}
	}
	return out
}

func flipV(img *image.RGBA) *image.RGBA {
	b := img.Bounds()
	out := newRGBA(b.Dx(), b.Dy())
	for y := 0; y < b.Dy(); y++ {
		row := img.Pix[img.PixOffset(b.Min.X, b.Min.Y+b.Dy()-1-y):][:b.Dx()*4]
		copy(out.Pix[out.PixOffset(0, y):], row)
	}
	return out
}

// hcat joins images of equal height left to right.
func hcat(parts ...*image.RGBA) *image.RGBA {
	w, h := 0, parts[0].Bounds().Dy()
	for _, p := range parts {
		w += p.Bounds().Dx()
	}
	out := newRGBA(w, h)
	x := 0
	for _, p := range parts {
		draw.Draw(out, image.Rect(x, 0, x+p.Bounds().Dx(), h), p, p.Bounds().Min, draw.Src)
		x += p.Bounds().Dx()
	}
	return out
}

// fillRGBA fills r with a flat colour (source-over).
func fillRGBA(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Over)
}

// fillMasked paints c through m at r.Min.
func fillMasked(dst *image.RGBA, at image.Point, m *image.Alpha, c color.RGBA) {
	draw.DrawMask(dst, image.Rectangle{Min: at, Max: at.Add(m.Bounds().Size())}, image.NewUniform(c), image.Point{}, m, image.Point{}, draw.Over)
}

// shade darkens (k<0) or lightens (k>0) every pixel in r by |k|·strength(x,y).
// strength returns 0..1 for a pixel in r-local coordinates.
func shade(dst *image.RGBA, r image.Rectangle, k float64, strength func(x, y float64) float64) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			s := strength(float64(x-r.Min.X)+0.5, float64(y-r.Min.Y)+0.5)
			if s <= 0 {
				continue
			}
			f := 1 + k*s
			i := dst.PixOffset(x, y)
			p := dst.Pix[i : i+3 : i+3]
			for c := 0; c < 3; c++ {
				v := float64(p[c]) * f
				if v < 0 {
					v = 0
				}
				if v > 255 {
					v = 255
				}
				p[c] = uint8(v)
			}
		}
	}
}

// innerShadow darkens the edges of r like a recess: light from the top-left,
// so the right and top walls throw the deepest shadow.
func innerShadow(dst *image.RGBA, r image.Rectangle, top, bottom, left, right, depth float64) {
	w, h := float64(r.Dx()), float64(r.Dy())
	shade(dst, r, -1, func(x, y float64) float64 {
		s := 0.0
		if y < depth {
			s = math.Max(s, top*(1-y/depth))
		}
		if h-y < depth {
			s = math.Max(s, bottom*(1-(h-y)/depth))
		}
		if x < depth {
			s = math.Max(s, left*(1-x/depth))
		}
		if w-x < depth {
			s = math.Max(s, right*(1-(w-x)/depth))
		}
		return s * s * (3 - 2*s) // smoothstep
	})
}

// vignette darkens toward the top and bottom of r.
func vignette(dst *image.RGBA, r image.Rectangle, amount float64) {
	h := float64(r.Dy())
	shade(dst, r, -amount, func(_, y float64) float64 {
		t := math.Abs(y/h*2 - 1)
		return t * t
	})
}

// avgColor averages the opaque pixels of r.
func avgColor(src *image.RGBA, r image.Rectangle) color.RGBA {
	var rs, gs, bs, n uint64
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := src.RGBAAt(x, y)
			rs += uint64(c.R)
			gs += uint64(c.G)
			bs += uint64(c.B)
			n++
		}
	}
	if n == 0 {
		return color.RGBA{A: 255}
	}
	return color.RGBA{uint8(rs / n), uint8(gs / n), uint8(bs / n), 255}
}

// homography maps four points p (destination) to q (source) and returns the
// 3×3 matrix M with q ~ M·p (row-major).
func homography(p, q [4]f2) [9]float64 {
	// Build the 8×8 linear system A·h = b.
	var A [8][8]float64
	var b [8]float64
	for i := 0; i < 4; i++ {
		x, y := p[i].x, p[i].y
		u, v := q[i].x, q[i].y
		A[2*i] = [8]float64{x, y, 1, 0, 0, 0, -u * x, -u * y}
		b[2*i] = u
		A[2*i+1] = [8]float64{0, 0, 0, x, y, 1, -v * x, -v * y}
		b[2*i+1] = v
	}
	// Gaussian elimination with partial pivoting.
	for col := 0; col < 8; col++ {
		piv := col
		for r := col + 1; r < 8; r++ {
			if math.Abs(A[r][col]) > math.Abs(A[piv][col]) {
				piv = r
			}
		}
		A[col], A[piv] = A[piv], A[col]
		b[col], b[piv] = b[piv], b[col]
		for r := 0; r < 8; r++ {
			if r == col || A[col][col] == 0 {
				continue
			}
			f := A[r][col] / A[col][col]
			for c := col; c < 8; c++ {
				A[r][c] -= f * A[col][c]
			}
			b[r] -= f * b[col]
		}
	}
	var h [9]float64
	for i := 0; i < 8; i++ {
		h[i] = b[i] / A[i][i]
	}
	h[8] = 1
	return h
}

func applyH(h [9]float64, p f2) f2 {
	w := h[6]*p.x + h[7]*p.y + h[8]
	return f2{(h[0]*p.x + h[1]*p.y + h[2]) / w, (h[3]*p.x + h[4]*p.y + h[5]) / w}
}

// warpPerspective fills dst (whole image) by mapping each destination pixel
// through h into src and sampling it bicubically. Pixels that land outside
// src stay transparent.
func warpPerspective(dst *image.RGBA, h [9]float64, src *image.RGBA) {
	b := dst.Bounds()
	sb := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			s := applyH(h, f2{float64(x) + 0.5, float64(y) + 0.5})
			sx, sy := s.x-0.5, s.y-0.5
			if sx < 1 || sy < 1 || sx >= float64(sb.Max.X-2) || sy >= float64(sb.Max.Y-2) {
				continue
			}
			c := bicubicSample(src, sx, sy)
			i := dst.PixOffset(x, y)
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = c.R, c.G, c.B, 255
		}
	}
}

func cubicW(t float64) float64 {
	// Catmull-Rom kernel.
	t = math.Abs(t)
	if t < 1 {
		return 1.5*t*t*t - 2.5*t*t + 1
	}
	if t < 2 {
		return -0.5*t*t*t + 2.5*t*t - 4*t + 2
	}
	return 0
}

func bicubicSample(src *image.RGBA, x, y float64) color.RGBA {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	var acc [3]float64
	wsum := 0.0
	for j := -1; j <= 2; j++ {
		wy := cubicW(float64(j) - fy)
		for i := -1; i <= 2; i++ {
			w := cubicW(float64(i)-fx) * wy
			c := src.RGBAAt(int(x0)+i, int(y0)+j)
			acc[0] += float64(c.R) * w
			acc[1] += float64(c.G) * w
			acc[2] += float64(c.B) * w
			wsum += w
		}
	}
	cl := func(v float64) uint8 {
		v /= wsum
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v + 0.5)
	}
	return color.RGBA{cl(acc[0]), cl(acc[1]), cl(acc[2]), 255}
}

// unsharp sharpens img in place: img += amount·(img − blur3(img)).
func unsharp(img *image.RGBA, amount float64) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	blur := make([]float64, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var s [3]float64
			n := 0.0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= w || yy >= h {
						continue
					}
					i := img.PixOffset(b.Min.X+xx, b.Min.Y+yy)
					s[0] += float64(img.Pix[i])
					s[1] += float64(img.Pix[i+1])
					s[2] += float64(img.Pix[i+2])
					n++
				}
			}
			o := (y*w + x) * 3
			blur[o], blur[o+1], blur[o+2] = s[0]/n, s[1]/n, s[2]/n
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := img.PixOffset(b.Min.X+x, b.Min.Y+y)
			if img.Pix[i+3] == 0 {
				continue
			}
			o := (y*w + x) * 3
			for c := 0; c < 3; c++ {
				v := float64(img.Pix[i+c])
				v += amount * (v - blur[o+c])
				if v < 0 {
					v = 0
				}
				if v > float64(img.Pix[i+3]) {
					v = float64(img.Pix[i+3])
				}
				img.Pix[i+c] = uint8(v)
			}
		}
	}
}

// drawOver composites src onto dst at the given point (source-over).
func drawOver(dst *image.RGBA, at image.Point, src *image.RGBA) {
	draw.Draw(dst, image.Rectangle{Min: at, Max: at.Add(src.Bounds().Size())}, src, src.Bounds().Min, draw.Over)
}

// triangleMask returns a feathered mask for a backgammon point (local
// coords): base from (bl,by) to (br,by), tip at (tx,ty), grown by `grow`
// so the inlay's outline survives, with an optional spike ornament past the
// tip of width spikeW ending at spikeEnd.
func triangleMask(w, h int, bl, br, by, tx, ty, grow, spikeW, spikeEnd float64) *image.Alpha {
	dir := 1.0 // base above tip
	if ty < by {
		dir = -1
	}
	pts := []f2{
		{bl - grow, by - dir*grow},
		{br + grow, by - dir*grow},
		{tx + grow*0.7, ty + dir*grow},
		{tx - grow*0.7, ty + dir*grow},
	}
	// Grow, then shrink toward the triangle centre so blurAlpha does not
	// bleed the inlay past the painted outline.
	m := polygonMask(w, h, shrinkPoly(pts, tx, (by+ty)/2, 0.6))
	if spikeW > 0 {
		sp := []f2{
			{tx - spikeW/2, ty - dir*grow*2},
			{tx + spikeW/2, ty - dir*grow*2},
			{tx + spikeW/2, spikeEnd},
			{tx - spikeW/2, spikeEnd},
		}
		m = unionMask(m, polygonMask(w, h, shrinkPoly(sp, tx, (ty+spikeEnd)/2, 0.4)))
	}
	return blurAlpha(m, 1)
}
