package effects

import "image"

// grayscale desaturates each pixel to its Rec.601 luminance.
func grayscale(img *image.RGBA) *image.RGBA {
	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		y := clamp8(0.299*float64(pix[i]) + 0.587*float64(pix[i+1]) + 0.114*float64(pix[i+2]))
		pix[i], pix[i+1], pix[i+2] = y, y, y
	}
	return img
}
