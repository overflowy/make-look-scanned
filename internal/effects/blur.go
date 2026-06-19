package effects

import (
	"image"
	"math"
)

// gaussianBlur applies a separable gaussian blur of the given sigma,
// mimicking the slight defocus of a scanner's optics.
func gaussianBlur(img *image.RGBA, sigma float64) *image.RGBA {
	kernel := gaussianKernel(sigma)
	radius := len(kernel) / 2
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Horizontal pass into a temporary buffer, then vertical pass back.
	tmp := image.NewRGBA(b)
	blurPass(img.Pix, tmp.Pix, w, h, kernel, radius, true)
	out := image.NewRGBA(b)
	blurPass(tmp.Pix, out.Pix, w, h, kernel, radius, false)
	return out
}

// blurPass convolves one axis. horizontal=true blurs along x, else along y.
func blurPass(src, dst []uint8, w, h int, kernel []float64, radius int, horizontal bool) {
	for y := range h {
		for x := range w {
			var r, g, bl float64
			for k := -radius; k <= radius; k++ {
				sx, sy := x, y
				if horizontal {
					sx = clampInt(x+k, 0, w-1)
				} else {
					sy = clampInt(y+k, 0, h-1)
				}
				idx := (sy*w + sx) * 4
				wt := kernel[k+radius]
				r += float64(src[idx]) * wt
				g += float64(src[idx+1]) * wt
				bl += float64(src[idx+2]) * wt
			}
			idx := (y*w + x) * 4
			dst[idx] = clamp8(r)
			dst[idx+1] = clamp8(g)
			dst[idx+2] = clamp8(bl)
			dst[idx+3] = 255
		}
	}
}

func gaussianKernel(sigma float64) []float64 {
	radius := max(int(math.Ceil(sigma*3)), 1)
	kernel := make([]float64, 2*radius+1)
	var sum float64
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		kernel[i+radius] = v
		sum += v
	}
	for i := range kernel {
		kernel[i] /= sum
	}
	return kernel
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
