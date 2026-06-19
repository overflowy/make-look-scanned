// Package config loads user-defined presets from
// $XDG_CONFIG_HOME/make-look-scanned/config.toml and overlays a selected
// preset onto the built-in effect defaults.
//
// Precedence is: built-in defaults -> selected preset -> explicit CLI flags.
// This package handles the first two; the caller (main) applies CLI flags on
// top, since only it knows which flags the user actually set.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/overflowy/make-look-scanned/internal/effects"
)

// presetParams mirrors effects.Params with pointer fields so an absent TOML key
// is distinguishable from a zero value and leaves the default untouched.
type presetParams struct {
	DPI         *float64 `toml:"dpi"`
	Skew        *float64 `toml:"skew"`
	Grayscale   *bool    `toml:"grayscale"`
	PaperTone   *float64 `toml:"paper_tone"`
	Noise       *float64 `toml:"noise"`
	Blur        *float64 `toml:"blur"`
	EdgeShadow  *float64 `toml:"edge_shadow"`
	JPEGQuality *int     `toml:"jpeg_quality"`
}

type configFile struct {
	Presets map[string]presetParams `toml:"presets"`
}

// Path returns the location of the config file. If XDG_CONFIG_HOME is set it is
// honored ($XDG_CONFIG_HOME/make-look-scanned/config.toml); otherwise the file
// lives in a dotted directory directly under the home dir
// ($HOME/.make-look-scanned/config.toml).
func Path() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "make-look-scanned", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".make-look-scanned", "config.toml")
}

// Resolve returns the built-in defaults with the named preset overlaid. An
// empty preset name returns the plain defaults. A named preset that cannot be
// found (missing file or missing table) is an error.
func Resolve(preset string) (effects.Params, error) {
	base := effects.Defaults()
	if preset == "" {
		return base, nil
	}

	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return base, fmt.Errorf("preset %q requested but no config file at %s", preset, path)
		}
		return base, fmt.Errorf("read config %s: %w", path, err)
	}

	var cf configFile
	if err := toml.Unmarshal(data, &cf); err != nil {
		return base, fmt.Errorf("parse config %s: %w", path, err)
	}

	pp, ok := cf.Presets[preset]
	if !ok {
		return base, fmt.Errorf("preset %q not found in %s", preset, path)
	}
	pp.applyTo(&base)
	return base, nil
}

// applyTo overlays the present fields of pp onto p.
func (pp presetParams) applyTo(p *effects.Params) {
	if pp.DPI != nil {
		p.DPI = *pp.DPI
	}
	if pp.Skew != nil {
		p.Skew = *pp.Skew
	}
	if pp.Grayscale != nil {
		p.Grayscale = *pp.Grayscale
	}
	if pp.PaperTone != nil {
		p.PaperTone = *pp.PaperTone
	}
	if pp.Noise != nil {
		p.Noise = *pp.Noise
	}
	if pp.Blur != nil {
		p.Blur = *pp.Blur
	}
	if pp.EdgeShadow != nil {
		p.EdgeShadow = *pp.EdgeShadow
	}
	if pp.JPEGQuality != nil {
		p.JPEGQuality = *pp.JPEGQuality
	}
}
