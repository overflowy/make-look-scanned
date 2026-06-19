package effects

import (
	"image"
	"math/rand"
)

// noise adds gaussian grain to every pixel. amount (0..1) scales the standard
// deviation up to a visible but believable level.
func noise(img *image.RGBA, amount float64, rng *rand.Rand) *image.RGBA {
	if amount > 1 {
		amount = 1
	}
	stddev := amount * 40.0
	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		n := rng.NormFloat64() * stddev
		pix[i] = clamp8(float64(pix[i]) + n)
		pix[i+1] = clamp8(float64(pix[i+1]) + n)
		pix[i+2] = clamp8(float64(pix[i+2]) + n)
	}
	return img
}
