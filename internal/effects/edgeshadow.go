package effects

import (
	"image"
	"math"
)

// edgeShadow darkens the page toward its borders, like a sheet lifted slightly
// off the scanner glass. strength (0..1) sets how dark the very edge gets.
func edgeShadow(img *image.RGBA, strength float64) *image.RGBA {
	if strength > 1 {
		strength = 1
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// Shadow band reaches ~12% in from each edge.
	band := math.Min(float64(w), float64(h)) * 0.12
	if band < 1 {
		band = 1
	}

	pix := img.Pix
	for y := range h {
		dy := math.Min(float64(y), float64(h-1-y))
		for x := range w {
			dx := math.Min(float64(x), float64(w-1-x))
			d := math.Min(dx, dy)
			if d >= band {
				continue
			}
			// Darkest at the edge (d=0), fading to none at d=band.
			f := 1.0 - strength*(1.0-d/band)
			idx := (y*w + x) * 4
			pix[idx] = clamp8(float64(pix[idx]) * f)
			pix[idx+1] = clamp8(float64(pix[idx+1]) * f)
			pix[idx+2] = clamp8(float64(pix[idx+2]) * f)
		}
	}
	return img
}
