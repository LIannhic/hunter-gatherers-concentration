package renderer

import (
	_ "embed"
	"image/color"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	//go:embed shader/scanner.kage
	scannerShaderSource []byte
	//go:embed shader/blur.kage
	blurShaderSource []byte
	//go:embed shader/bubble.kage
	bubbleShaderSource []byte
	//go:embed shader/heat.kage
	heatShaderSource []byte
	//go:embed shader/wave.kage
	waveShaderSource []byte
	//go:embed shader/vortex.kage
	vortexShaderSource []byte
	//go:embed shader/quake.kage
	quakeShaderSource []byte
)

type EffectRenderer struct {
	shaders map[string]*ebiten.Shader
	bufferA *ebiten.Image
	bufferB *ebiten.Image
	count   int
}

func NewEffectRenderer() (*EffectRenderer, error) {
	r := &EffectRenderer{
		shaders: make(map[string]*ebiten.Shader),
		bufferA: ebiten.NewImage(ui.ScreenWidth, ui.ScreenHeight),
		bufferB: ebiten.NewImage(ui.ScreenWidth, ui.ScreenHeight),
	}

	sources := map[string][]byte{
		"scanner": scannerShaderSource,
		"blur":    blurShaderSource,
		"bubble":  bubbleShaderSource,
		"heat":    heatShaderSource,
		"wave":    waveShaderSource,
		"vortex":  vortexShaderSource,
		"quake":   quakeShaderSource,
	}

	for name, src := range sources {
		s, err := ebiten.NewShader(src)
		if err != nil {
			return nil, err
		}
		r.shaders[name] = s
	}

	return r, nil
}

func (r *EffectRenderer) Update() {
	r.count++
}

// DrawScannerEffect dessine l'effet de balayage du scanner.
func (r *EffectRenderer) DrawScannerEffect(dst, src *ebiten.Image, x, y int, progress, erase, thickness float32, clr color.Color) {
	shader, ok := r.shaders["scanner"]
	if !ok {
		return
	}

	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	cr, cg, cb, ca := clr.RGBA()

	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.Images[0] = src
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

	dst.DrawRectShader(w, h, shader, op)
}

// DrawQuakeEffect dessine l'effet de séisme de rotation du Stonewarden.
// currentSrc/ghostSrc = le playmat ghost (990×990)
// Le shader tourne le ghost et le recadre en 700×700 (taille de resolution)
func (r *EffectRenderer) DrawQuakeEffect(dst, currentSrc, ghostSrc *ebiten.Image, x, y int, progress, rotationAngle, intensity float32, center []float32, resolution []float32, ghostSize []float32) {
	shader, ok := r.shaders["quake"]
	if !ok {
		return
	}

	w, h := int(resolution[0]), int(resolution[1])

	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.Images[0] = currentSrc
	op.Images[1] = ghostSrc
	op.Uniforms = map[string]interface{}{
		"Time":           float32(r.count) / 60.0,
		"Intensity":      intensity,
		"Resolution":     resolution,
		"Center":         center,
		"RotationAngle":  rotationAngle,
		"Progress":       progress,
		"GhostSize":      ghostSize,
	}

	dst.DrawRectShader(w, h, shader, op)
}

type GlobalEffectParams struct {
	SanityRatio float32 // 0.0 (vide) to 1.0 (plein)
	Biome       string
	UseBlur     bool
	UseBubble   bool
	PortalPos   []float32 // [x, y] normalized, nil if none
	MousePos    []float32 // [x, y] normalized
	ScreenSize  []float32 // [width, height] pixels
}

// ProcessGlobalEffects applique les effets cumulatifs sur l'image entière.
func (r *EffectRenderer) ProcessGlobalEffects(screen *ebiten.Image, params GlobalEffectParams) {
	// L'intensité est nulle à Sanity 1.0 et max à 0.0.
	// On utilise une atténuation carrée pour le cumul comme suggéré.
	intensity := float32(math.Pow(float64(1.0-params.SanityRatio), 2))

	if intensity <= 0 && !params.UseBlur && !params.UseBubble && params.Biome == "" {
		return
	}

	// Redimensionnement dynamique des buffers si nécessaire
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if r.bufferA.Bounds().Dx() != sw || r.bufferA.Bounds().Dy() != sh {
		r.bufferA = ebiten.NewImage(sw, sh)
		r.bufferB = ebiten.NewImage(sw, sh)
	}

	// Ping-pong buffers initialization
	r.bufferA.Clear()
	r.bufferA.DrawImage(screen, nil)
	src := r.bufferA
	dst := r.bufferB

	anyApplied := false

	// 1. Biome Effects (Wave for Swamp, Heat for Desert)
	if params.Biome == "swamp" {
		r.applyShader(src, dst, "wave", intensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	} else if params.Biome == "desert" {
		r.applyShader(src, dst, "heat", intensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	// 2. Creature Attack Effects
	if params.UseBubble {
		r.applyShader(src, dst, "bubble", intensity, params.MousePos, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseBlur {
		// Blur is often applied last to smooth things out
		r.applyShader(src, dst, "blur", intensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	// 3. Special Static Effects (Vortex for Portal)
	if params.PortalPos != nil {
		// Le vortex est toujours à intensité max (ou peut être couplé à Sanity si on veut)
		vIntensity := float32(1.0)
		r.applyShader(src, dst, "vortex", vIntensity, params.PortalPos, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if anyApplied {
		screen.DrawImage(src, nil)
	}
}

func (r *EffectRenderer) applyShader(src, dst *ebiten.Image, name string, intensity float32, center []float32, resolution []float32) {
	shader, ok := r.shaders[name]
	if !ok {
		return
	}

	dst.Clear()
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	op.Uniforms = map[string]interface{}{
		"Time":       float32(r.count) / 60.0,
		"Intensity":  intensity,
		"Resolution": resolution,
	}

	if center != nil {
		op.Uniforms["Center"] = center
	}

	// Important : DrawRectShader doit utiliser les dimensions réelles de l'image source
	// pour que les coordonnées (dstPos, srcPos) soient correctes.
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst.DrawRectShader(sw, sh, shader, op)
}
