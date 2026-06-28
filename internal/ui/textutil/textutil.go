package textutil

import (
	"bytes"
	_ "embed"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed fonts/GoMono-Regular.ttf
var fontData []byte

var (
	fontSource *text.GoTextFaceSource
	// Face is the shared GoTextFace for all text rendering.
	Face *text.GoTextFace
	// charWidth is the fixed pixel width of a single character (monospace).
	charWidth float64
)

func init() {
	fontSource, _ = text.NewGoTextFaceSource(bytes.NewReader(fontData))
	Face = &text.GoTextFace{
		Source: fontSource,
		Size:   13,
	}
	// Measure a single character to get the exact monospace width.
	charWidth, _ = text.Measure("M", Face, 0)
}

// Draw draws text at (x, y) with the given color.
// y is treated as the baseline position (like text v1), not the top of the text.
var sharedDrawOp = &text.DrawOptions{}

func Draw(dst *ebiten.Image, str string, x, y int, clr color.Color) {
	sharedDrawOp.GeoM.Reset()
	// text v1 y was the baseline; text/v2 y is the top of the rendering region.
	// Subtract ascent to convert baseline -> top-left.
	sharedDrawOp.GeoM.Translate(float64(x), float64(y)-Face.Metrics().HAscent)
	sharedDrawOp.ColorScale.Reset()
	sharedDrawOp.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, str, Face, sharedDrawOp)
}

// MeasureWidth returns the width in pixels of the given text.
// Optimized for Go Mono (monospace): len * charWidth.
func MeasureWidth(str string) int {
	return int(float64(len(str)) * charWidth)
}
