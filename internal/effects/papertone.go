package effects

import "image"

// paperTone tints the page toward a warm off-white and lifts the blacks
// slightly, so pure white reads as paper and pure black as scanner-gray.
// strength (0..1) blends between the original and the toned version.
func paperTone(img *image.RGBA, strength float64) *image.RGBA {
	if strength > 1 {
		strength = 1
	}
	// Per-channel scale that maps white(255) onto the paper color.
	sr := float64(paperColor.R) / 255.0
	sg := float64(paperColor.G) / 255.0
	sb := float64(paperColor.B) / 255.0
	// Lift blacks so the darkest ink is dark gray, not pure black.
	const blackLift = 18.0

	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		r := blackLift + float64(pix[i])*sr*(1-blackLift/255.0)
		g := blackLift + float64(pix[i+1])*sg*(1-blackLift/255.0)
		b := blackLift + float64(pix[i+2])*sb*(1-blackLift/255.0)
		pix[i] = clamp8(lerp(float64(pix[i]), r, strength))
		pix[i+1] = clamp8(lerp(float64(pix[i+1]), g, strength))
		pix[i+2] = clamp8(lerp(float64(pix[i+2]), b, strength))
	}
	return img
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }
