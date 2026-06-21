// Package effects degrades a page raster so it reads as a physical scan.
//
// Each effect is an independent stage that maps one CLI flag to one
// transformation. A numeric knob of 0 disables its stage, so no separate
// enable/disable booleans are needed (grayscale is the lone bool).
package effects

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
)

// Params holds every effect knob. Zero on a numeric knob disables that effect.
type Params struct {
	DPI         float64
	Skew        float64 // max absolute rotation in degrees
	Grayscale   bool    // desaturate to luminance
	PaperTone   float64 // 0..1 strength of warm off-white tint
	Noise       float64 // 0..1 scanner grain
	Blur        float64 // gaussian sigma (defocus)
	EdgeShadow  float64 // 0..1 vignette/border darkening
	JPEGQuality int     // 1..100, applied at assembly (the low quality IS the tell)
}

// Defaults returns the built-in parameter set, tuned for a believable scan.
func Defaults() Params {
	return Params{
		DPI:         150,
		Skew:        0.6,
		Grayscale:   true,
		PaperTone:   0.6,
		Noise:       0.08,
		Blur:        0.4,
		EdgeShadow:  0.15,
		JPEGQuality: 70,
	}
}

// Validate rejects out-of-range parameters before they reach rendering or
// image allocation, so bad CLI flags or preset values fail fast with a clear
// message instead of panicking or burning CPU/memory.
func (p Params) Validate() error {
	switch {
	case p.DPI <= 0 || p.DPI > 1200:
		return fmt.Errorf("dpi must be in (0, 1200], got %g", p.DPI)
	case p.Skew < 0 || p.Skew > 45:
		return fmt.Errorf("skew must be in [0, 45] degrees, got %g", p.Skew)
	case p.PaperTone < 0 || p.PaperTone > 1:
		return fmt.Errorf("paper-tone must be in [0, 1], got %g", p.PaperTone)
	case p.Noise < 0 || p.Noise > 1:
		return fmt.Errorf("noise must be in [0, 1], got %g", p.Noise)
	case p.Blur < 0 || p.Blur > 50:
		return fmt.Errorf("blur must be in [0, 50], got %g", p.Blur)
	case p.EdgeShadow < 0 || p.EdgeShadow > 1:
		return fmt.Errorf("edge-shadow must be in [0, 1], got %g", p.EdgeShadow)
	case p.JPEGQuality < 1 || p.JPEGQuality > 100:
		return fmt.Errorf("jpeg-quality must be in [1, 100], got %d", p.JPEGQuality)
	}
	return nil
}

// paperColor is the warm off-white that pure white becomes, and the fill for
// corners exposed by skew rotation.
var paperColor = color.RGBA{R: 250, G: 247, B: 236, A: 255}

// Run applies the effect stages in a fixed order. Order matters: the page is
// toned and roughed up before being rotated, so every prior effect rotates
// with the page. The JPEG pass happens later, in the assemble package.
//
// Each stochastic effect gets its own rng, seeded up front from the shared rng
// in a fixed order. This keeps a given effect's randomness independent of the
// others: noise draws one value per pixel, so a different DPI (more pixels)
// would otherwise leave the shared rng in a different state by the time skew
// reads its angle — making skew drift with DPI. Seeding per effect also makes
// each effect's look independent of whether the others are enabled.
func Run(img *image.RGBA, p Params, rng *rand.Rand) *image.RGBA {
	noiseRng := rand.New(rand.NewSource(rng.Int63()))
	skewRng := rand.New(rand.NewSource(rng.Int63()))

	if p.Grayscale {
		img = grayscale(img)
	}
	if p.PaperTone > 0 {
		img = paperTone(img, p.PaperTone)
	}
	if p.Blur > 0 {
		img = gaussianBlur(img, p.Blur)
	}
	if p.Noise > 0 {
		img = noise(img, p.Noise, noiseRng)
	}
	if p.EdgeShadow > 0 {
		img = edgeShadow(img, p.EdgeShadow)
	}
	if p.Skew > 0 {
		img = skew(img, p.Skew, skewRng)
	}
	return img
}

// clamp8 bounds v to a valid uint8.
func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
