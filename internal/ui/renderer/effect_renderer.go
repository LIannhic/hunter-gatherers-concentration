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
	//go:embed shader/cave.kage
	caveShaderSource []byte
	//go:embed shader/vertige.kage
	vertigeShaderSource []byte
	//go:embed shader/invert.kage
	invertShaderSource []byte
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
		"rain":    rainShaderSource,
		"cave":    caveShaderSource,
		"vertige": vertigeShaderSource,
		"invert":  invertShaderSource,
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
	UseWave     bool
	UseHeat     bool
	UseCave     bool
	UseVortex   bool
	UseVertige  bool
	UseInvert   bool
	PortalPos   []float32 // [x, y] normalized, nil if none
	MousePos    []float32 // [x, y] normalized
	ScreenSize  []float32 // [width, height] pixels
}

// ProcessCreatureAttackEffects applique les effets visuels d'attaque de créatures
// (blur, bubble, vertige) sur l'image entière.
// Doit être appelé AVANT le rendu du HUD pour ne pas affecter les fenêtres UI.
func (r *EffectRenderer) ProcessCreatureAttackEffects(screen *ebiten.Image, params GlobalEffectParams) {
	intensity := float32(math.Pow(float64(1.0-params.SanityRatio), 2))

	if !params.UseBlur && !params.UseBubble && !params.UseVertige && !params.UseInvert {
		return
	}

	r.ensureBuffers(screen)

	r.bufferA.Clear()
	r.bufferA.DrawImage(screen, nil)
	src := r.bufferA
	dst := r.bufferB

	anyApplied := false

	if params.UseBubble {
		bubbleMin := float32(0.45)
		bubbleIntensity := bubbleMin + (1.0-bubbleMin)*intensity
		r.applyShader(src, dst, "bubble", bubbleIntensity, params.MousePos, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseBlur {
		blurMin := float32(0.4)
		blurIntensity := blurMin + (1.0-blurMin)*intensity
		r.applyShader(src, dst, "blur", blurIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseVertige {
		vertigeMin := float32(0.45)
		vertigeIntensity := vertigeMin + (1.0-vertigeMin)*intensity
		r.applyShader(src, dst, "vertige", vertigeIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseInvert {
		invertMin := float32(0.5)
		invertIntensity := invertMin + (1.0-invertMin)*intensity
		r.applyShader(src, dst, "invert", invertIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if anyApplied {
		op := &ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendCopy
		screen.DrawImage(src, op)
	}
}

// ProcessBiomeEffects applique les effets de biome et environnementaux
// (wave, heat, rain, cave, vortex) sur l'image entière.
func (r *EffectRenderer) ProcessBiomeEffects(screen *ebiten.Image, params GlobalEffectParams) {
	intensity := float32(math.Pow(float64(1.0-params.SanityRatio), 2))

	shouldApply := intensity > 0 || params.UseRain || params.Biome != "" || params.PortalPos != nil || params.UseVortex
	if !shouldApply {
		return
	}

	shaderIntensity := intensity
	if shaderIntensity < 0.15 {
		shaderIntensity = 0.15
	}

	r.ensureBuffers(screen)

	r.bufferA.Clear()
	r.bufferA.DrawImage(screen, nil)
	src := r.bufferA
	dst := r.bufferB

	anyApplied := false
	applyCave := params.Biome == "cave" || params.UseCave

	if params.Biome == "swamp" || params.UseWave {
		r.applyShader(src, dst, "wave", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}
	if params.Biome == "desert" || params.UseHeat {
		r.applyShader(src, dst, "heat", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}
	if params.Biome == "forest" || params.UseRain {
		r.applyShader(src, dst, "rain", shaderIntensity, nil, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if params.UseVortex || params.PortalPos != nil {
		vortexCenter := params.PortalPos
		if vortexCenter == nil && params.UseVortex {
			vortexCenter = []float32{
				float32(ui.PlaymatX+ui.PlaymatW/2) / params.ScreenSize[0],
				float32(ui.PlaymatY+ui.PlaymatH/2) / params.ScreenSize[1],
			}
		}
		r.applyShader(src, dst, "vortex", 1.0, vortexCenter, params.ScreenSize)
		src, dst = dst, src
		anyApplied = true
	}

	if applyCave {
		hudLights := r.buildHudLights()
		r.applyCaveShader(src, dst, shaderIntensity, params.ScreenSize, hudLights)
		src, dst = dst, src
		anyApplied = true
	}

	if anyApplied {
		op := &ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendCopy
		screen.DrawImage(src, op)
	}
}

// ensureBuffers redimensionne les buffers ping-pong si nécessaire.
func (r *EffectRenderer) ensureBuffers(screen *ebiten.Image) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	if r.bufferA.Bounds().Dx() != sw || r.bufferA.Bounds().Dy() != sh {
		r.bufferA = ebiten.NewImage(sw, sh)
		r.bufferB = ebiten.NewImage(sw, sh)
	}
}

// buildHudLights retourne les cercles de lumière pour les panneaux HUD.
// Format plat [cx, cy, radius, 0, ...] × 4 éléments (pixels écran).
func (r *EffectRenderer) buildHudLights() []float32 {
	// Portrait : centre (145,145), rayon ~190 (couvre le carré 270×270)
	// Inventaire : centre (145,525), rayon ~230 (couvre 270×370)
	// Jauges : centre (1135,220), rayon ~210 (couvre 270×320)
	// Minimap : centre (1135,575), rayon ~190 (couvre 270×270)
	return []float32{
		145, 145, 190, 0,
		145, 525, 230, 0,
		1135, 220, 210, 0,
		1135, 575, 190, 0,
	}
}

func (r *EffectRenderer) applyCaveShader(src, dst *ebiten.Image, intensity float32, resolution, hudLights []float32) {
	shader, ok := r.shaders["cave"]
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
		"HudLights":  hudLights,
	}

	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst.DrawRectShader(sw, sh, shader, op)
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
