package main

import (
	"fmt"
	"image"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const (
	speakerBody = `<path fill="currentColor" d="M3 9v6h4l5 5V4L7 9H3z"/>`
	waveLow     = `<path fill="currentColor" d="M16.5 12A4.5 4.5 0 0 0 14 8v8a4.5 4.5 0 0 0 2.5-4z"/>`
	waveHigh    = `<path fill="currentColor" d="M14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>`
	cross       = `<path fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" d="M16 9.5l5 5M21 9.5l-5 5"/>`
	bars        = `<path fill="currentColor" d="M4 10h2.4v4H4z M8.4 6h2.4v12H8.4z M12.8 3h2.4v18h-2.4z M17.2 8h2.4v8h-2.4z"/>`
)

func glyph(size int, argb uint32, paths ...string) image.Image {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">` + strings.Join(paths, "") + `</svg>`
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	icon, err := oksvg.ReadReplacingCurrentColor(strings.NewReader(svg), fmt.Sprintf("#%06x", argb&0xFFFFFF))
	if err != nil {
		return img
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	for i := range icon.SVGPaths {
		icon.SVGPaths[i].LineWidth *= float64(size) / icon.ViewBox.W
	}
	icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, img, img.Bounds())), 1)
	return img
}

func speakerGlyph(size int, argb uint32, volume float32, muted bool) image.Image {
	paths := []string{speakerBody}
	if muted {
		paths = append(paths, cross)
	} else {
		if volume > 0.01 {
			paths = append(paths, waveLow)
		}
		if volume > 0.5 {
			paths = append(paths, waveHigh)
		}
	}
	return glyph(size, argb, paths...)
}

func appGlyph(size int, argb uint32) image.Image {
	return glyph(size, argb, bars)
}
