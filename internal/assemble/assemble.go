// Package assemble re-encodes processed page rasters as JPEG and packs them
// into an image-only PDF, preserving each page's original point dimensions.
//
// JPEG encoding lives here on purpose: the low-quality JPEG IS the final
// "scanned" artifact, and embedding the encoded bytes directly avoids a
// redundant re-encode by the PDF writer.
//
// gopdf is used (rather than fpdf) because it serializes pages and images in a
// deterministic order and stamps no timestamp, so identical input + seed yields
// byte-identical output. fpdf orders embedded images via Go map iteration,
// which is randomized per process.
//
// The package takes plain images plus page dimensions (not a render.Page) so it
// carries no dependency on the rasterizer — which lets it compile to js/wasm,
// where rendering is done in the browser instead of by go-fitz.
package assemble

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"

	"github.com/signintech/gopdf"
)

// Bytes encodes each processed page to JPEG at the given quality (1..100) and
// returns the assembled PDF, one full-page image per page. widthsPt/heightsPt
// give each page's size in points and must be parallel to imgs.
func Bytes(imgs []*image.RGBA, widthsPt, heightsPt []float64, quality int) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeLetter})

	for i, img := range imgs {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode page %d: %w", i+1, err)
		}

		size := &gopdf.Rect{W: widthsPt[i], H: heightsPt[i]}
		pdf.AddPageWithOption(gopdf.PageOption{PageSize: size})

		holder, err := gopdf.ImageHolderByBytes(buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("embed page %d: %w", i+1, err)
		}
		if err := pdf.ImageByHolder(holder, 0, 0, size); err != nil {
			return nil, fmt.Errorf("place page %d: %w", i+1, err)
		}
	}

	return pdf.GetBytesPdf(), nil
}

// Write assembles the PDF and writes it to outPath.
func Write(outPath string, imgs []*image.RGBA, widthsPt, heightsPt []float64, quality int) error {
	b, err := Bytes(imgs, widthsPt, heightsPt, quality)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	return nil
}
