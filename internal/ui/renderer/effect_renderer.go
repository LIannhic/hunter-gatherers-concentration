package renderer

import (
	_ "embed"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shader/scanner.kage
var ScannerEffectShaderSource []byte

type EffectRenderer struct {
	shader *ebiten.Shader
}

func NewEffectRenderer() (*EffectRenderer, error) {
	shader, err := ebiten.NewShader(ScannerEffectShaderSource)
	if err != nil {
		return nil, err
	}
	return &EffectRenderer{shader: shader}, nil
}

// DrawScannerEffect dessine l'effet de balayage du scanner.
// x, y : position de destination sur l'image dst.
// progress : position X (en pixels écran) de l'avant de la vague.
// erase : position X (en pixels écran) de l'arrière de la vague.
// thickness : épaisseur de la ligne de balayage.
// clr : couleur de révélation utilisée par le shader.
func (r *EffectRenderer) DrawScannerEffect(dst, src *ebiten.Image, x, y int, progress, erase, thickness float32, clr color.Color) {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()

	// Conversion de la couleur en float32 pour le shader (vec4)
	cr, cg, cb, ca := clr.RGBA()

	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(float64(x), float64(y))

	// On passe l'image source au shader (imageSrc0At)
	op.Images[0] = src

	// Définition des uniforms requis par scanner.kage
	op.Uniforms = map[string]interface{}{
		"WaveProgress":  progress,
		"WaveErase":     erase,
		"WaveThickness": thickness,
		"RevealColor": []float32{
			float32(cr) / 0xffff,
			float32(cg) / 0xffff,
			float32(cb) / 0xffff,
			float32(ca) / 0xffff,
		},
	}

	dst.DrawRectShader(w, h, r.shader, op)
}
