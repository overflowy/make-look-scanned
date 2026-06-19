// Package render rasterizes PDF pages to bitmaps using MuPDF (go-fitz).
package render

import (
	"fmt"
	"image"
	"image/draw"

	"github.com/gen2brain/go-fitz"
)

// Page is a single rasterized PDF page together with the physical page size
// (in PostScript points, 1/72 inch) so the output PDF can preserve dimensions.
type Page struct {
	Img      *image.RGBA
	WidthPt  float64
	HeightPt float64
}

// Pages renders every page of the given PDF to an RGBA bitmap at dpi.
func Pages(pdf []byte, dpi float64) ([]Page, error) {
	doc, err := fitz.NewFromMemory(pdf)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer doc.Close()

	n := doc.NumPage()
	if n == 0 {
		return nil, fmt.Errorf("pdf has no pages")
	}

	pages := make([]Page, 0, n)
	for i := range n {
		img, err := doc.ImageDPI(i, dpi)
		if err != nil {
			return nil, fmt.Errorf("render page %d: %w", i+1, err)
		}
		// Take page size from the PDF's own page box (in points), not from the
		// rounded raster dimensions, so non-standard sizes stay exact.
		bound, err := doc.Bound(i)
		if err != nil {
			return nil, fmt.Errorf("page %d bounds: %w", i+1, err)
		}
		pages = append(pages, Page{
			Img:      toRGBA(img),
			WidthPt:  float64(bound.Dx()),
			HeightPt: float64(bound.Dy()),
		})
	}
	return pages, nil
}

// toRGBA returns img as *image.RGBA, copying only if it isn't already one.
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba
}
