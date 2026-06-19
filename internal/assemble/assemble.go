// Package assemble re-encodes processed page rasters as JPEG and writes them
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
package assemble

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/signintech/gopdf"

	"github.com/overflowy/make-look-scanned/internal/render"
)

// Write encodes each page to JPEG at the given quality (1..100) and assembles
// them into a PDF at outPath, one full-page image per page.
func Write(outPath string, pages []render.Page, processed []*image.RGBA, quality int) error {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeLetter})

	for i, page := range pages {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, processed[i], &jpeg.Options{Quality: quality}); err != nil {
			return fmt.Errorf("encode page %d: %w", i+1, err)
		}

		size := &gopdf.Rect{W: page.WidthPt, H: page.HeightPt}
		pdf.AddPageWithOption(gopdf.PageOption{PageSize: size})

		holder, err := gopdf.ImageHolderByBytes(buf.Bytes())
		if err != nil {
			return fmt.Errorf("embed page %d: %w", i+1, err)
		}
		if err := pdf.ImageByHolder(holder, 0, 0, size); err != nil {
			return fmt.Errorf("place page %d: %w", i+1, err)
		}
	}

	if err := pdf.WritePdf(outPath); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	return nil
}
