// Command make-look-scanned degrades a PDF so it reads as a physical scan.
package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"image"
	"math/rand"
	"os"
	"strings"

	"github.com/overflowy/make-look-scanned/internal/assemble"
	"github.com/overflowy/make-look-scanned/internal/config"
	"github.com/overflowy/make-look-scanned/internal/effects"
	"github.com/overflowy/make-look-scanned/internal/render"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "make-look-scanned: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		outPath = flag.String("o", "", "output path (default: <input>.scanned.pdf)")
		preset  = flag.String("preset", "", "named preset from config.toml")
		seed    = flag.Int64("seed", 0, "override the content-derived random seed")
		force   = flag.Bool("force", false, "overwrite an existing output file")

		// Effect knobs. Defaults here are placeholders; the real defaults come
		// from effects.Defaults() via config.Resolve. Only flags the user
		// actually sets are applied, detected with flag.Visit below.
		dpi        = flag.Float64("dpi", 150, "render DPI")
		skew       = flag.Float64("skew", 0.6, "max rotation degrees (0 disables)")
		grayscale  = flag.Bool("grayscale", true, "desaturate to gray")
		paperTone  = flag.Float64("paper-tone", 0.6, "warm paper tint strength 0..1")
		noise      = flag.Float64("noise", 0.08, "scanner grain 0..1")
		blur       = flag.Float64("blur", 0.4, "defocus gaussian sigma")
		edgeShadow = flag.Float64("edge-shadow", 0.15, "border vignette 0..1")
		jpegQ      = flag.Int("jpeg-quality", 70, "JPEG quality 1..100")
	)
	flag.Usage = usage
	flag.Parse()

	// stdlib flag stops at the first positional, so flags after the input PDF
	// would be ignored. Pull out the input and re-parse the remaining args so
	// flags may appear on either side of the filename.
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return fmt.Errorf("expected exactly one input PDF")
	}
	inPath := args[0]
	if len(args) > 1 {
		if err := flag.CommandLine.Parse(args[1:]); err != nil {
			return err
		}
		if flag.NArg() > 0 {
			flag.Usage()
			return fmt.Errorf("unexpected extra arguments: %v", flag.Args())
		}
	}

	params, err := config.Resolve(*preset)
	if err != nil {
		return err
	}

	// Overlay only the CLI flags the user explicitly set (CLI wins over preset).
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "dpi":
			params.DPI = *dpi
		case "skew":
			params.Skew = *skew
		case "grayscale":
			params.Grayscale = *grayscale
		case "paper-tone":
			params.PaperTone = *paperTone
		case "noise":
			params.Noise = *noise
		case "blur":
			params.Blur = *blur
		case "edge-shadow":
			params.EdgeShadow = *edgeShadow
		case "jpeg-quality":
			params.JPEGQuality = *jpegQ
		}
	})

	if err := params.Validate(); err != nil {
		return err
	}

	out := *outPath
	if out == "" {
		out = defaultOutPath(inPath)
	}
	if !*force {
		if _, err := os.Stat(out); err == nil {
			return fmt.Errorf("output %s already exists (use --force to overwrite)", out)
		}
	}

	pdf, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	baseSeed := *seed
	if baseSeed == 0 {
		baseSeed = contentSeed(pdf)
	}

	pages, err := render.Pages(pdf, params.DPI)
	if err != nil {
		return err
	}

	processed := make([]*image.RGBA, len(pages))
	widthsPt := make([]float64, len(pages))
	heightsPt := make([]float64, len(pages))
	for i := range pages {
		rng := rand.New(rand.NewSource(baseSeed ^ int64(i)*0x9E3779B9))
		processed[i] = effects.Run(pages[i].Img, params, rng)
		widthsPt[i] = pages[i].WidthPt
		heightsPt[i] = pages[i].HeightPt
	}

	if err := assemble.Write(out, processed, widthsPt, heightsPt, params.JPEGQuality); err != nil {
		return err
	}

	fmt.Printf("wrote %s (%d pages)\n", out, len(pages))
	return nil
}

// contentSeed derives a stable seed from the input bytes for deterministic
// output: the same PDF always produces the same scan.
func contentSeed(b []byte) int64 {
	h := fnv.New64a()
	h.Write(b)
	return int64(h.Sum64())
}

func defaultOutPath(in string) string {
	if strings.HasSuffix(strings.ToLower(in), ".pdf") {
		return in[:len(in)-4] + ".scanned.pdf"
	}
	return in + ".scanned.pdf"
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: make-look-scanned [flags] input.pdf")
	fmt.Fprintln(os.Stderr, "\nFlags:")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nPresets are read from %s\n", config.Path())
}
