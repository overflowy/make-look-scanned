package effects

import (
	"image"
	"math"
	"math/rand"
)

// skew rotates the page by a small random angle within ±maxDeg, filling the
// corners exposed by the rotation with the paper color. Output keeps the input
// dimensions. The rotated page edge is anti-aliased (see bilinear) so it does
// not look jagged against the fill.
func skew(img *image.RGBA, maxDeg float64, rng *rand.Rand) *image.RGBA {
	angle := (rng.Float64()*2 - 1) * maxDeg * math.Pi / 180.0
	if angle == 0 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(b)

	cx, cy := float64(w)/2, float64(h)/2
	sin, cos := math.Sin(angle), math.Cos(angle)

	for y := range h {
		for x := range w {
			// Inverse-rotate the destination point back into source space.
			ox, oy := float64(x)-cx, float64(y)-cy
			sx := cos*ox + sin*oy + cx
			sy := -sin*ox + cos*oy + cy
			idx := (y*w + x) * 4
			r, g, bl := bilinear(img, w, h, sx, sy)
			out.Pix[idx] = r
			out.Pix[idx+1] = g
			out.Pix[idx+2] = bl
			out.Pix[idx+3] = 255
		}
	}
	return out
}

// bilinear samples the source RGBA at fractional (sx, sy). Neighbors that fall
// outside the image contribute the paper color rather than being clamped, so a
// pixel straddling the rotated page edge blends content into the fill by
// coverage — anti-aliasing the boundary instead of producing a jagged seam.
func bilinear(img *image.RGBA, w, h int, sx, sy float64) (uint8, uint8, uint8) {
	x0, y0 := int(math.Floor(sx)), int(math.Floor(sy))
	fx, fy := sx-float64(x0), sy-float64(y0)
	x1, y1 := x0+1, y0+1

	at := func(x, y, ch int) float64 {
		if x < 0 || x >= w || y < 0 || y >= h {
			switch ch {
			case 0:
				return float64(paperColor.R)
			case 1:
				return float64(paperColor.G)
			default:
				return float64(paperColor.B)
			}
		}
		return float64(img.Pix[(y*w+x)*4+ch])
	}
	sample := func(ch int) uint8 {
		top := at(x0, y0, ch)*(1-fx) + at(x1, y0, ch)*fx
		bot := at(x0, y1, ch)*(1-fx) + at(x1, y1, ch)*fx
		return clamp8(top*(1-fy) + bot*fy)
	}
	return sample(0), sample(1), sample(2)
}
