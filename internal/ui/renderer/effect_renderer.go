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
	//go:embed shader/lumifly.kage
	lumiflyShaderSource []byte
	//go:embed shader/rain.kage
	rainShaderSource []byte
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
		"lumifly": lumiflyShaderSource,
		"rain": rainShaderSource,
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

// DrawLumiflyEffect dessine l'onde lumineuse circulaire du Lumifly.
// src contient les icons des entités rendues sur le dos des tuiles.
// centerXY = position du centre en pixels (coords srcImg), radius = rayon actuel, progress = 0..1
func (r *EffectRenderer) DrawLumiflyEffect(dst, src *ebiten.Image, x, y int, centerX, centerY, radius, progress, duration float32, clr color.Color) {
	shader, ok := r.shaders["lumifly"]
	if !ok {
		return
	}

	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	cr, cg, cb, ca := clr.RGBA()

	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.Images[0] = src
	op.Uniforms = map[string]interface{}{
		"CenterX":  centerX,
		"CenterY":  centerY,
		"Radius":   radius,
		"Progress": progress,
		"Duration": duration,
		"GlowColor": []float32{
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
	UseRain     bool
	PortalPos   []float32 // [x, y] normalized, nil if none
	MousePos    []float32 // [x, y] normalized
	ScreenSize  []float32 // [width, height] pixels
}

// ProcessGlobalEffects applique les effets cumulatifs sur l'image entière.
func (r *EffectRenderer) ProcessGlobalEffects(screen *ebiten.Image, params GlobalEffectParams) {
	// L'intensité est nulle à Sanity 1.0 et max à 0.0.
	intensity := float32(math.Pow(float64(1.0-params.SanityRatio), 2))

	// On vérifie si on doit appliquer des shaders même sans intensité de folie
	shouldApply := intensity > 0 || params.UseBlur || params.UseBubble || params.UseRain || params.Biome != "" || params.PortalPos != nil

	if !shouldApply {
		return
	}

	// On s'assure d'une intensité minimale pour que les shaders biome soient visibles
	// même si la santé mentale est haute.
	shaderIntensity := intensity
	if shaderIntensity < 0.15 {
		shaderIntensity = 0.15
	}

	// Redimensionnement dynamique des buffers si nécessaire
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if r.bufferA.Bounds().Dx() != sw || r.bufferA.Bounds().Dy() != sh {
		r.bufferA = ebiten.NewImage(sw, sh)
		r.bufferB = ebiten.NewImage(sw, sh)
	}

	// Initialisation : on copie l'écran actuel dans Buffer A (Source)
	r.bufferA.Clear()
	r.bufferA.DrawImage(screen, nil)
	src := r.bufferA
	dst := r.bufferB

	anyApplied := false

	// 1. Biome Effects (Wave for Swamp, Heat for Desert, Rain for Forest)
	if params.Biome == "swamp" {
		r.applyShader(src, dst, "wave", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src // Ping-pong
		anyApplied = true
	} else if params.Biome == "desert" {
		r.applyShader(src, dst, "heat", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src // Ping-pong
		anyApplied = true
	} else if params.Biome == "forest" {
		r.applyShader(src, dst, "rain", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src // Ping-pong
		anyApplied = true
	}

	// 2. Creature Attack Effects
	if params.UseBubble {
		r.applyShader(src, dst, "bubble", shaderIntensity, params.MousePos, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseBlur {
		r.applyShader(src, dst, "blur", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseRain {
		r.applyShader(src, dst, "rain", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	// 3. Special Static Effects (Vortex for Portal)
	if params.PortalPos != nil {
		r.applyShader(src, dst, "vortex", 1.0, params.PortalPos, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	// Rendu final : On recopie le dernier Buffer (src) sur l'écran
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
