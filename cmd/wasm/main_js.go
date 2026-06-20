//go:build js && wasm

// Command wasm is the browser entrypoint for make-look-scanned.
//
// The browser rasterizes PDF pages (via PDF.js) and hands each page's raw RGBA
// pixels to this module; here we run the SAME effects pipeline the CLI uses and
// assemble the scanned PDF with gopdf, returning the bytes to JavaScript. Only
// rasterization differs between platforms — every effect is shared Go code.
//
// JS API (installed on the global `mls` object):
//
//	mls.reset(params)                                  -> {error}
//	mls.addPage(rgba, w, h, widthPt, heightPt)         -> {error}
//	mls.build()                                        -> {error, pdf: Uint8Array}
//
// `params` is a plain object whose keys mirror the CLI flags in camelCase:
// dpi, skew, grayscale, paperTone, noise, blur, edgeShadow, jpegQuality, and
// seed (a decimal string, since a 64-bit seed does not fit a JS number).
package main

import (
	"image"
	"maps"
	"math/rand"
	"strconv"
	"syscall/js"

	"github.com/overflowy/make-look-scanned/internal/assemble"
	"github.com/overflowy/make-look-scanned/internal/effects"
)

// doc is the document being built across reset -> addPage* -> build.
type document struct {
	params  effects.Params
	seed    int64
	page    int
	imgs    []*image.RGBA
	widths  []float64
	heights []float64
}

var doc document

func main() {
	js.Global().Set("mls", map[string]any{
		"reset":   js.FuncOf(reset),
		"addPage": js.FuncOf(addPage),
		"build":   js.FuncOf(build),
	})
	// Keep the Go runtime alive so the exported functions remain callable.
	select {}
}

// result builds the {error, ...} object the JS side inspects.
func result(err string, extra map[string]any) any {
	m := map[string]any{"error": nil}
	if err != "" {
		m["error"] = err
	}
	maps.Copy(m, extra)
	return m
}

// reset stores validated params + seed and clears any accumulated pages.
func reset(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return result("reset: missing params", nil)
	}
	p := args[0]

	params := effects.Defaults()
	params.DPI = numOr(p, "dpi", params.DPI)
	params.Skew = numOr(p, "skew", params.Skew)
	params.PaperTone = numOr(p, "paperTone", params.PaperTone)
	params.Noise = numOr(p, "noise", params.Noise)
	params.Blur = numOr(p, "blur", params.Blur)
	params.EdgeShadow = numOr(p, "edgeShadow", params.EdgeShadow)
	params.JPEGQuality = int(numOr(p, "jpegQuality", float64(params.JPEGQuality)))
	if g := p.Get("grayscale"); g.Type() == js.TypeBoolean {
		params.Grayscale = g.Bool()
	}
	if err := params.Validate(); err != nil {
		return result(err.Error(), nil)
	}

	seed := int64(0)
	if s := p.Get("seed"); s.Type() == js.TypeString {
		if v, err := strconv.ParseUint(s.String(), 10, 64); err == nil {
			seed = int64(v)
		}
	}

	doc = document{params: params, seed: seed}
	return result("", nil)
}

// addPage processes one rasterized page and appends it to the document.
func addPage(this js.Value, args []js.Value) any {
	if len(args) < 5 {
		return result("addPage: expected (rgba, w, h, widthPt, heightPt)", nil)
	}
	w := args[1].Int()
	h := args[2].Int()
	widthPt := args[3].Float()
	heightPt := args[4].Float()
	if w <= 0 || h <= 0 {
		return result("addPage: bad page dimensions", nil)
	}

	pix := make([]byte, w*h*4)
	if n := js.CopyBytesToGo(pix, args[0]); n != len(pix) {
		return result("addPage: rgba length does not match w*h*4", nil)
	}
	img := &image.RGBA{Pix: pix, Stride: w * 4, Rect: image.Rect(0, 0, w, h)}

	rng := rand.New(rand.NewSource(doc.seed ^ int64(doc.page)*0x9E3779B9))
	out := effects.Run(img, doc.params, rng)

	doc.imgs = append(doc.imgs, out)
	doc.widths = append(doc.widths, widthPt)
	doc.heights = append(doc.heights, heightPt)
	doc.page++
	return result("", nil)
}

// build assembles the accumulated pages into a PDF and returns its bytes.
func build(this js.Value, args []js.Value) any {
	if len(doc.imgs) == 0 {
		return result("build: no pages added", nil)
	}
	b, err := assemble.Bytes(doc.imgs, doc.widths, doc.heights, doc.params.JPEGQuality)
	if err != nil {
		return result(err.Error(), nil)
	}
	dst := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(dst, b)
	return result("", map[string]any{"pdf": dst})
}

// numOr reads a numeric field, falling back to def when absent or non-numeric.
func numOr(o js.Value, key string, def float64) float64 {
	if v := o.Get(key); v.Type() == js.TypeNumber {
		return v.Float()
	}
	return def
}
