package assets

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func generateLumiflySilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := LumiflyPalette

	wingColor := color.RGBA{255, 255, 255, 100}
	vector.DrawFilledRect(img, cx-25, cy-10, 18, 12, wingColor, true)
	vector.DrawFilledRect(img, cx-22, cy-5, 12, 15, wingColor, true)
	vector.DrawFilledRect(img, cx+7, cy-10, 18, 12, wingColor, true)
	vector.DrawFilledRect(img, cx+10, cy-5, 12, 15, wingColor, true)
	vector.DrawFilledCircle(img, cx, cy+8, 10, p.Body, true)
	vector.DrawFilledCircle(img, cx, cy+8, 6, p.Highlight, true)
	vector.DrawFilledCircle(img, cx-2, cy+6, 2, color.RGBA{255, 255, 255, 200}, true)
	vector.DrawFilledCircle(img, cx, cy-8, 8, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-3, cy-10, 2, p.Eye, true)
	vector.DrawFilledCircle(img, cx+3, cy-10, 2, p.Eye, true)
	vector.StrokeLine(img, cx-3, cy-14, cx-8, cy-22, 2, p.Shadow, true)
	vector.StrokeLine(img, cx+3, cy-14, cx+8, cy-22, 2, p.Shadow, true)
	return img
}

func generateShadowstalkerSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := ShadowstalkerPalette

	vector.DrawFilledRect(img, cx-15, cy-8, 30, 22, p.Body, true)
	vector.DrawFilledRect(img, cx-12, cy-18, 24, 12, p.Body, true)
	vector.DrawFilledRect(img, cx-18, cy-22, 6, 12, p.Shadow, true)
	vector.DrawFilledRect(img, cx+12, cy-22, 6, 12, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-6, cy-12, 4, p.Eye, true)
	vector.DrawFilledCircle(img, cx+6, cy-12, 4, p.Eye, true)
	vector.DrawFilledCircle(img, cx-7, cy-13, 1, color.RGBA{255, 150, 150, 255}, true)
	vector.DrawFilledCircle(img, cx+5, cy-13, 1, color.RGBA{255, 150, 150, 255}, true)
	vector.DrawFilledRect(img, cx-20, cy+8, 4, 12, p.Shadow, true)
	vector.DrawFilledRect(img, cx+16, cy+8, 4, 12, p.Shadow, true)
	auraColor := color.RGBA{60, 40, 70, 80}
	vector.DrawFilledCircle(img, cx, cy, 28, auraColor, true)
	return img
}

func generateBurrowerSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := BurrowerPalette

	vector.DrawFilledRect(img, cx-12, cy-15, 24, 35, p.Body, true)
	vector.DrawFilledRect(img, cx-10, cy-8, 20, 4, p.Highlight, true)
	vector.DrawFilledRect(img, cx-10, cy+5, 20, 4, p.Highlight, true)
	vector.DrawFilledRect(img, cx-6, cy-22, 12, 10, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-4, cy-12, 2, p.Eye, true)
	vector.DrawFilledCircle(img, cx+4, cy-12, 2, p.Eye, true)
	vector.DrawFilledRect(img, cx-20, cy-2, 8, 4, p.Shadow, true)
	vector.DrawFilledRect(img, cx+12, cy-2, 8, 4, p.Shadow, true)
	vector.DrawFilledRect(img, cx-22, cy+12, 10, 4, p.Shadow, true)
	vector.DrawFilledRect(img, cx+12, cy+12, 10, 4, p.Shadow, true)
	vector.DrawFilledRect(img, cx-22, cy, 3, 6, color.RGBA{80, 60, 40, 255}, true)
	vector.DrawFilledRect(img, cx+19, cy, 3, 6, color.RGBA{80, 60, 40, 255}, true)
	return img
}

func generateFlutterwingSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := FlutterwingPalette

	vector.DrawFilledCircle(img, cx-18, cy-5, 15, p.Body, true)
	vector.DrawFilledCircle(img, cx-15, cy-8, 8, p.Highlight, true)
	vector.DrawFilledCircle(img, cx+18, cy-5, 15, p.Body, true)
	vector.DrawFilledCircle(img, cx+15, cy-8, 8, p.Highlight, true)
	vector.DrawFilledRect(img, cx-2, cy-15, 4, 30, p.Shadow, true)
	vector.DrawFilledCircle(img, cx, cy-18, 6, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-2, cy-20, 2, p.Eye, true)
	vector.DrawFilledCircle(img, cx+2, cy-20, 2, p.Eye, true)
	vector.StrokeLine(img, cx-1, cy-23, cx-4, cy-28, 2, p.Shadow, true)
	vector.StrokeLine(img, cx+1, cy-23, cx+4, cy-28, 2, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-4, cy-28, 2, p.Highlight, true)
	vector.DrawFilledCircle(img, cx+4, cy-28, 2, p.Highlight, true)
	return img
}

func generateSpecterSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := SpecterPalette

	vector.DrawFilledCircle(img, cx, cy-5, 18, p.Body, true)
	vector.DrawFilledRect(img, cx-14, cy-5, 28, 20, p.Body, true)
	vector.DrawFilledCircle(img, cx-7, cy+15, 8, p.Body, true)
	vector.DrawFilledCircle(img, cx+7, cy+15, 8, p.Body, true)
	vector.DrawFilledCircle(img, cx-6, cy-8, 3, p.Eye, true)
	vector.DrawFilledCircle(img, cx+6, cy-8, 3, p.Eye, true)
	haloColor := color.RGBA{150, 255, 255, 50}
	vector.DrawFilledCircle(img, cx, cy-5, 25, haloColor, true)
	return img
}

func generateEchoHoundSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := EchoHoundPalette

	vector.DrawFilledRect(img, cx-20, cy-5, 40, 15, p.Body, true)
	vector.DrawFilledRect(img, cx-5, cy-20, 25, 18, p.Body, true)
	vector.DrawFilledRect(img, cx-2, cy-28, 6, 10, p.Shadow, true)
	vector.DrawFilledRect(img, cx+12, cy-28, 6, 10, p.Shadow, true)
	vector.DrawFilledCircle(img, cx+8, cy-14, 4, p.Eye, true)
	vector.DrawFilledCircle(img, cx+18, cy-14, 4, p.Eye, true)
	vector.DrawFilledRect(img, cx-15, cy+10, 4, 15, p.Shadow, true)
	vector.DrawFilledRect(img, cx-5, cy+10, 4, 15, p.Shadow, true)
	vector.DrawFilledRect(img, cx+5, cy+10, 4, 15, p.Shadow, true)
	vector.DrawFilledRect(img, cx+15, cy+10, 4, 15, p.Shadow, true)
	return img
}

func generateMossMonkeySilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := MossMonkeyPalette

	vector.DrawFilledCircle(img, cx, cy+5, 18, p.Body, true)
	vector.DrawFilledCircle(img, cx-10, cy-5, 8, p.Body, true)
	vector.DrawFilledCircle(img, cx+10, cy-5, 8, p.Body, true)
	vector.DrawFilledCircle(img, cx, cy-12, 10, p.Highlight, true)
	vector.DrawFilledCircle(img, cx, cy-5, 12, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-12, cy-8, 5, p.Shadow, true)
	vector.DrawFilledCircle(img, cx+12, cy-8, 5, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-4, cy-7, 3, p.Eye, true)
	vector.DrawFilledCircle(img, cx+4, cy-7, 3, p.Eye, true)
	vector.DrawFilledCircle(img, cx-4, cy-7, 1, color.Black, true)
	vector.DrawFilledCircle(img, cx+4, cy-7, 1, color.Black, true)
	vector.StrokeLine(img, cx-15, cy+5, cx-22, cy+20, 3, p.Body, true)
	vector.StrokeLine(img, cx+15, cy+5, cx+22, cy+20, 3, p.Body, true)
	return img
}

func generateStonewardenSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := StonewardenPalette

	vector.DrawFilledRect(img, cx-18, cy-10, 36, 30, p.Body, true)
	vector.DrawFilledRect(img, cx-18, cy-10, 36, 4, p.Highlight, true)
	vector.DrawFilledRect(img, cx-18, cy+16, 36, 4, p.Shadow, true)
	vector.DrawFilledRect(img, cx-24, cy-5, 12, 12, p.Shadow, true)
	vector.DrawFilledRect(img, cx+12, cy-5, 12, 12, p.Shadow, true)
	vector.DrawFilledRect(img, cx-10, cy-22, 20, 15, p.Body, true)
	vector.DrawFilledRect(img, cx-10, cy-22, 20, 2, p.Highlight, true)
	vector.DrawFilledRect(img, cx-6, cy-16, 4, 3, p.Eye, true)
	vector.DrawFilledRect(img, cx+2, cy-16, 4, 3, p.Eye, true)
	mossColor := color.RGBA{80, 120, 60, 200}
	vector.DrawFilledCircle(img, cx-10, cy+5, 4, mossColor, true)
	vector.DrawFilledCircle(img, cx+12, cy-2, 3, mossColor, true)
	vector.DrawFilledCircle(img, cx-5, cy+20, 5, mossColor, true)
	return img
}

func generateFleeingSpriteSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := FleeingSpritePalette

	for i := 0; i < 4; i++ {
		angle := float64(i) * math.Pi / 2
		dx := float32(math.Cos(angle)) * 20
		dy := float32(math.Sin(angle)) * 20
		vector.StrokeLine(img, cx-dx, cy-dy, cx+dx, cy+dy, 4, p.Body, true)
	}
	vector.DrawFilledCircle(img, cx, cy, 12, p.Highlight, true)
	vector.DrawFilledCircle(img, cx-15, cy+10, 6, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-25, cy+18, 4, p.Shadow, true)
	vector.DrawFilledCircle(img, cx-3, cy-2, 3, p.Eye, true)
	vector.DrawFilledCircle(img, cx+3, cy-2, 3, p.Eye, true)
	return img
}

func generateDreamberrySilhouette(size int, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := DreamberryPalette

	scale := float32(1.0)
	switch stage {
	case "bourgeon":
		scale = 0.8
	case "fleur":
		scale = 0.9
	}

	bodyColor := p.Primary
	switch stage {
	case "bourgeon":
		bodyColor = color.RGBA{100, 200, 100, 255}
	case "fleur":
		bodyColor = color.RGBA{255, 180, 220, 255}
	case "gâté":
		bodyColor = color.RGBA{80, 60, 70, 255}
	}

	leafColor := p.Secondary
	vector.DrawFilledCircle(img, cx-12*scale, cy-15*scale, 8*scale, leafColor, true)
	vector.DrawFilledCircle(img, cx+12*scale, cy-15*scale, 8*scale, leafColor, true)
	vector.DrawFilledCircle(img, cx, cy-20*scale, 10*scale, leafColor, true)
	berryRadius := float32(size/3) * scale
	vector.DrawFilledCircle(img, cx, cy+5*scale, berryRadius, bodyColor, true)
	vector.DrawFilledCircle(img, cx-8*scale, cy-2*scale, berryRadius/3, p.Accent, true)
	vector.DrawFilledCircle(img, cx-18*scale, cy+15*scale, 6*scale, p.Secondary, true)
	vector.DrawFilledCircle(img, cx+18*scale, cy+15*scale, 6*scale, p.Secondary, true)
	return img
}

func generateMoonstoneSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := MoonstonePalette

	stoneRadius := float32(size / 3)
	vector.DrawFilledCircle(img, cx, cy, stoneRadius, p.Primary, true)
	vector.DrawFilledRect(img, cx-8, cy-15, 16, 12, p.Secondary, true)
	vector.DrawFilledRect(img, cx-15, cy-3, 10, 10, p.Secondary, true)
	vector.DrawFilledRect(img, cx+5, cy-3, 10, 10, p.Secondary, true)
	vector.DrawFilledRect(img, cx-8, cy+8, 16, 8, p.Secondary, true)
	vector.DrawFilledCircle(img, cx-6, cy-6, 5, p.Accent, true)
	starColor := p.Accent
	vector.DrawFilledRect(img, cx-22, cy-18, 3, 3, starColor, true)
	vector.DrawFilledRect(img, cx+20, cy-20, 2, 2, starColor, true)
	vector.DrawFilledRect(img, cx+18, cy+18, 3, 3, starColor, true)
	return img
}

func generateWhisperingHerbSilhouette(size int, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	baseY := float32(size - 10)
	p := WhisperingHerbPalette

	scale := float32(1.0)
	switch stage {
	case "graine":
		scale = 0.8
	case "pousse":
		scale = 0.9
	}

	leafColor := p.Primary
	switch stage {
	case "graine":
		leafColor = color.RGBA{140, 100, 60, 255}
	case "pousse":
		leafColor = color.RGBA{150, 255, 150, 255}
	}

	stemColor := p.Secondary
	vector.DrawFilledRect(img, cx-2*scale, baseY-30*scale, 4*scale, 30*scale, stemColor, true)
	vector.DrawFilledCircle(img, cx-12*scale, baseY-20*scale, 10*scale, leafColor, true)
	vector.DrawFilledCircle(img, cx+12*scale, baseY-25*scale, 10*scale, leafColor, true)
	vector.DrawFilledCircle(img, cx, baseY-40*scale, 10*scale, leafColor, true)
	if stage == "mature" {
		soundColor := p.Accent
		vector.StrokeLine(img, cx+20, baseY-45, cx+28, baseY-50, 2, soundColor, true)
		vector.StrokeLine(img, cx+22, baseY-42, cx+30, baseY-45, 2, soundColor, true)
	}
	return img
}

func generateVoidBloomSilhouette(size int, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := VoidBloomPalette

	scale := float32(1.0)
	switch stage {
	case "graine":
		scale = 0.8
	case "éclosion":
		scale = 0.9
	}

	for i := 0; i < 6; i++ {
		angle := float64(i) * math.Pi * 2 / 6
		px := cx + float32(math.Cos(angle))*20*scale
		py := cy + float32(math.Sin(angle))*20*scale
		vector.DrawFilledCircle(img, px, py, 12*scale, p.Primary, true)
	}
	vector.DrawFilledCircle(img, cx, cy, 10*scale, p.Secondary, true)
	if stage == "pleine" {
		vector.DrawFilledCircle(img, cx, cy, 6*scale, p.Accent, true)
	}
	return img
}

func generateMossTruffleSilhouette(size int, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := MossTrufflePalette

	scale := float32(1.0)
	switch stage {
	case "bourgeon":
		scale = 0.8
	case "pousse":
		scale = 0.9
	}

	vector.DrawFilledCircle(img, cx, cy, float32(size/4)*scale, p.Primary, true)
	vector.DrawFilledCircle(img, cx-10*scale, cy-8*scale, float32(size/6)*scale, p.Primary, true)
	vector.DrawFilledCircle(img, cx+8*scale, cy+5*scale, float32(size/6)*scale, p.Primary, true)
	vector.DrawFilledCircle(img, cx-5*scale, cy+10*scale, 4*scale, p.Secondary, true)
	vector.DrawFilledCircle(img, cx+12*scale, cy-5*scale, 3*scale, p.Accent, true)
	return img
}

func generateEchoCrystalSilhouette(size int, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := EchoCrystalPalette

	scale := float32(1.0)
	if stage == "vibrant" {
		scale = 0.85
	}

	vector.DrawFilledRect(img, cx-10*scale, cy-20*scale, 20*scale, 40*scale, p.Primary, true)
	vector.DrawFilledRect(img, cx-22*scale, cy-5*scale, 15*scale, 25*scale, p.Secondary, true)
	vector.DrawFilledRect(img, cx+7*scale, cy-15*scale, 15*scale, 30*scale, p.Secondary, true)
	vector.DrawFilledCircle(img, cx, cy-10*scale, 5*scale, p.Accent, true)
	if stage == "résonnant" {
		vector.StrokeCircle(img, cx, cy, 30*scale, 2*scale, p.Accent, true)
	}
	return img
}

func generateSandCoreSilhouette(size int, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := SandCorePalette

	scale := float32(1.0)
	if stage == "instable" {
		scale = 0.85
	}

	vector.DrawFilledCircle(img, cx, cy, float32(size/5)*scale, p.Primary, true)
	vector.StrokeCircle(img, cx, cy, 25*scale, 4*scale, p.Secondary, true)
	if stage == "stable" {
		vector.DrawFilledRect(img, cx-25*scale, cy-25*scale, 4*scale, 4*scale, p.Accent, true)
		vector.DrawFilledRect(img, cx+20*scale, cy+20*scale, 4*scale, 4*scale, p.Accent, true)
	}
	return img
}

func generateShadowEssenceSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := ShadowEssencePalette

	vector.DrawFilledCircle(img, cx, cy+5, 14, p.Primary, true)
	vector.DrawFilledCircle(img, cx, cy-5, 12, p.Primary, true)
	vector.DrawFilledCircle(img, cx, cy-15, 8, p.Primary, true)
	vector.DrawFilledCircle(img, cx, cy+5, 8, p.Secondary, true)
	vector.DrawFilledCircle(img, cx, cy-5, 6, p.Secondary, true)
	particleColor := p.Accent
	vector.DrawFilledCircle(img, cx-20, cy-10, 3, particleColor, true)
	vector.DrawFilledCircle(img, cx+22, cy+8, 2, particleColor, true)
	vector.DrawFilledCircle(img, cx-15, cy+20, 2, particleColor, true)
	return img
}

func generateCrystalShardSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	cx := float32(size / 2)
	cy := float32(size / 2)
	p := CrystalShardPalette

	crystalColor := p.Primary
	vector.DrawFilledRect(img, cx-10, cy-18, 20, 18, crystalColor, true)
	vector.DrawFilledRect(img, cx-8, cy, 16, 18, crystalColor, true)
	facetColor := p.Secondary
	vector.DrawFilledRect(img, cx-2, cy-15, 4, 12, facetColor, true)
	vector.DrawFilledRect(img, cx-6, cy-5, 4, 8, facetColor, true)
	vector.DrawFilledRect(img, cx+2, cy-5, 4, 8, facetColor, true)
	sparkleColor := p.Accent
	vector.DrawFilledCircle(img, cx-5, cy-10, 3, sparkleColor, true)
	vector.DrawFilledCircle(img, cx+3, cy-5, 2, sparkleColor, true)
	return img
}

func generateTrapSilhouette(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	centerX, centerY := float32(size)/2, float32(size)/2
	trapColor := color.RGBA{180, 50, 50, 255}

	// Un X stylisé et épais
	vector.StrokeLine(img, centerX-15, centerY-15, centerX+15, centerY+15, 4, trapColor, true)
	vector.StrokeLine(img, centerX+15, centerY-15, centerX-15, centerY+15, 4, trapColor, true)

	return img
}
