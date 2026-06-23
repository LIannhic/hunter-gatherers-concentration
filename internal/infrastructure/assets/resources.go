// Package assets - Icônes des ressources
//
// ⚠️  ASSETS TEMPORAIRES / PLACEHOLDERS
// Ces icônes de ressources sont générées procéduralement et servent de 
// placeholders temporaires. Elles devront être remplacées par des sprites
// finaux (pixel art ou illustrations) avant la release.
//
// TODO: Remplacer par des assets finaux avant release
package assets

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ResourcePalette contient les couleurs pour chaque type de ressource
type ResourcePalette struct {
	Primary   color.Color
	Secondary color.Color
	Accent    color.Color
	Bg        color.Color
}

// Palettes des ressources
var (
	// Dreamberry - Baie onirique violette
	DreamberryPalette = ResourcePalette{
		Primary:   color.RGBA{180, 120, 220, 255}, // Violet clair
		Secondary: color.RGBA{140, 80, 180, 255},  // Violet foncé
		Accent:    color.RGBA{220, 200, 255, 255}, // Blanc violet
		Bg:        color.RGBA{60, 50, 80, 255},
	}

	// Moonstone - Pierre de lune bleutée
	MoonstonePalette = ResourcePalette{
		Primary:   color.RGBA{150, 180, 220, 255}, // Bleu clair
		Secondary: color.RGBA{100, 130, 180, 255}, // Bleu foncé
		Accent:    color.RGBA{220, 230, 255, 255}, // Blanc bleuté
		Bg:        color.RGBA{50, 60, 80, 255},
	}

	// WhisperingHerb - Herbe murmurante verte
	WhisperingHerbPalette = ResourcePalette{
		Primary:   color.RGBA{120, 200, 120, 255}, // Vert vif
		Secondary: color.RGBA{60, 140, 60, 255},   // Vert foncé
		Accent:    color.RGBA{200, 255, 200, 255}, // Blanc vert
		Bg:        color.RGBA{50, 70, 50, 255},
	}

	// ShadowEssence - Essence d'ombre
	ShadowEssencePalette = ResourcePalette{
		Primary:   color.RGBA{100, 80, 120, 255},
		Secondary: color.RGBA{60, 40, 80, 255},
		Accent:    color.RGBA{180, 160, 200, 255},
		Bg:        color.RGBA{40, 30, 50, 255},
	}

	// CrystalShard - Éclat de cristal
	CrystalShardPalette = ResourcePalette{
		Primary:   color.RGBA{180, 220, 240, 255},
		Secondary: color.RGBA{120, 180, 220, 255},
		Accent:    color.RGBA{240, 250, 255, 255},
		Bg:        color.RGBA{50, 70, 90, 255},
	}

	// MossTruffle - Truffe de mousse (Forêt)
	MossTrufflePalette = ResourcePalette{
		Primary:   color.RGBA{100, 140, 60, 255},  // Vert mousse
		Secondary: color.RGBA{70, 100, 40, 255},   // Vert sombre
		Accent:    color.RGBA{200, 220, 150, 255}, // Vert clair brillant
		Bg:        color.RGBA{45, 55, 40, 255},
	}

	// VoidBloom - Fleur du vide (Grotte)
	VoidBloomPalette = ResourcePalette{
		Primary:   color.RGBA{100, 60, 140, 255},  // Indigo/Violet
		Secondary: color.RGBA{50, 30, 80, 255},    // Violet profond
		Accent:    color.RGBA{200, 150, 255, 255}, // Violet néon
		Bg:        color.RGBA{30, 25, 45, 255},
	}

	// EchoCrystal - Cristal d'écho (Marais)
	EchoCrystalPalette = ResourcePalette{
		Primary:   color.RGBA{80, 200, 180, 255},  // Turquoise
		Secondary: color.RGBA{40, 120, 100, 255},  // Sarcelle sombre
		Accent:    color.RGBA{150, 255, 220, 255}, // Cyan clair
		Bg:        color.RGBA{20, 50, 45, 255},
	}

	// SandCore - Noyau de sable (Désert)
	SandCorePalette = ResourcePalette{
		Primary:   color.RGBA{240, 200, 100, 255}, // Sable doré
		Secondary: color.RGBA{180, 140, 60, 255},  // Ocre
		Accent:    color.RGBA{255, 250, 200, 255}, // Jaune pâle brillant
		Bg:        color.RGBA{60, 50, 30, 255},
	}
)

// generateDreamberry crée l'icône d'une baie onirique
func generateDreamberry(size int, p ResourcePalette, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Échelle selon le stade
	scale := float32(1.0)
	switch stage {
	case "bourgeon":
		scale = 0.8
	case "fleur":
		scale = 0.9
	}

	// Ajuste la couleur selon le stade
	bodyColor := p.Primary
	switch stage {
	case "bourgeon":
		bodyColor = color.RGBA{100, 200, 100, 255} // Vert (pas encore mûr)
	case "fleur":
		bodyColor = color.RGBA{255, 180, 220, 255} // Rose
	case "gâté":
		bodyColor = color.RGBA{80, 60, 70, 255}    // Grisâtre/Sombre
	}

	// Feuilles
	leafColor := p.Secondary
	vector.DrawFilledCircle(img, centerX-12*scale, centerY-15*scale, 8*scale, leafColor, true)
	vector.DrawFilledCircle(img, centerX+12*scale, centerY-15*scale, 8*scale, leafColor, true)
	vector.DrawFilledCircle(img, centerX, centerY-20*scale, 10*scale, leafColor, true)

	// Baie principale
	berryRadius := float32(size/3) * scale
	vector.DrawFilledCircle(img, centerX, centerY+5*scale, berryRadius, bodyColor, true)

	// Reflet brillant
	vector.DrawFilledCircle(img, centerX-8*scale, centerY-2*scale, berryRadius/3, p.Accent, true)

	// Petites baies secondaires
	vector.DrawFilledCircle(img, centerX-18*scale, centerY+15*scale, 6*scale, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX+18*scale, centerY+15*scale, 6*scale, p.Secondary, true)

	return img
}

// generateMoonstone crée l'icône d'une pierre de lune
func generateMoonstone(size int, p ResourcePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Pierre principale (forme hexagonale approximée)
	stoneRadius := float32(size / 3)
	vector.DrawFilledCircle(img, centerX, centerY, stoneRadius, p.Primary, true)

	// Facettes
	facetColor := p.Secondary
	vector.DrawFilledRect(img, centerX-8, centerY-15, 16, 12, facetColor, true)
	vector.DrawFilledRect(img, centerX-15, centerY-3, 10, 10, facetColor, true)
	vector.DrawFilledRect(img, centerX+5, centerY-3, 10, 10, facetColor, true)
	vector.DrawFilledRect(img, centerX-8, centerY+8, 16, 8, facetColor, true)

	// Brillance magique
	vector.DrawFilledCircle(img, centerX-6, centerY-6, 5, p.Accent, true)

	// Étoiles scintillantes autour
	starColor := p.Accent
	vector.DrawFilledRect(img, centerX-22, centerY-18, 3, 3, starColor, true)
	vector.DrawFilledRect(img, centerX+20, centerY-20, 2, 2, starColor, true)
	vector.DrawFilledRect(img, centerX+18, centerY+18, 3, 3, starColor, true)

	return img
}

// generateWhisperingHerb crée l'icône d'une herbe murmurante
func generateWhisperingHerb(size int, p ResourcePalette, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	baseY := float32(size - 10)

	// Échelle selon le stade
	scale := float32(1.0)
	switch stage {
	case "graine":
		scale = 0.8
	case "pousse":
		scale = 0.9
	}

	// Ajuste la couleur selon le stade
	leafColor := p.Primary
	switch stage {
	case "graine":
		leafColor = color.RGBA{140, 100, 60, 255} // Marron graine
	case "pousse":
		leafColor = color.RGBA{150, 255, 150, 255} // Vert très clair
	}

	// Tige principale
	stemColor := p.Secondary
	vector.DrawFilledRect(img, centerX-2*scale, baseY-30*scale, 4*scale, 30*scale, stemColor, true)

	// Feuilles ondulantes
	// Feuille gauche
	vector.DrawFilledCircle(img, centerX-12*scale, baseY-20*scale, 10*scale, leafColor, true)
	vector.DrawFilledCircle(img, centerX-8*scale, baseY-20*scale, 6*scale, p.Bg, true) // Masque
	// Feuille droite
	vector.DrawFilledCircle(img, centerX+12*scale, baseY-25*scale, 10*scale, leafColor, true)
	vector.DrawFilledCircle(img, centerX+8*scale, baseY-25*scale, 6*scale, p.Bg, true) // Masque
	// Feuille haute
	vector.DrawFilledCircle(img, centerX, baseY-40*scale, 10*scale, leafColor, true)
	vector.DrawFilledCircle(img, centerX, baseY-35*scale, 6*scale, p.Bg, true) // Masque

	// Effet de "murmure" (ondes sonores)
	if stage == "mature" {
		soundColor := p.Accent
		vector.StrokeLine(img, centerX+20, baseY-45, centerX+28, baseY-50, 2, soundColor, true)
		vector.StrokeLine(img, centerX+22, baseY-42, centerX+30, baseY-45, 2, soundColor, true)
	}

	return img
}

// generateShadowEssence crée l'icône d'une essence d'ombre
func generateShadowEssence(size int, p ResourcePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Flamme/essence sombre
	flameColor := p.Primary
	vector.DrawFilledCircle(img, centerX, centerY+5, 14, flameColor, true)
	vector.DrawFilledCircle(img, centerX, centerY-5, 12, flameColor, true)
	vector.DrawFilledCircle(img, centerX, centerY-15, 8, flameColor, true)

	// Cœur sombre
	vector.DrawFilledCircle(img, centerX, centerY+5, 8, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX, centerY-5, 6, p.Secondary, true)

	// Particules magiques
	particleColor := p.Accent
	vector.DrawFilledCircle(img, centerX-20, centerY-10, 3, particleColor, true)
	vector.DrawFilledCircle(img, centerX+22, centerY+8, 2, particleColor, true)
	vector.DrawFilledCircle(img, centerX-15, centerY+20, 2, particleColor, true)

	return img
}

// generateCrystalShard crée l'icône d'un éclat de cristal
func generateCrystalShard(size int, p ResourcePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Cristal principal (forme de diamant)
	crystalColor := p.Primary
	// Partie supérieure
	vector.DrawFilledRect(img, centerX-10, centerY-18, 20, 18, crystalColor, true)
	// Partie inférieure pointue
	vector.DrawFilledRect(img, centerX-8, centerY, 16, 18, crystalColor, true)

	// Facettes de cristal
	facetColor := p.Secondary
	vector.DrawFilledRect(img, centerX-2, centerY-15, 4, 12, facetColor, true)
	vector.DrawFilledRect(img, centerX-6, centerY-5, 4, 8, facetColor, true)
	vector.DrawFilledRect(img, centerX+2, centerY-5, 4, 8, facetColor, true)

	// Reflets
	sparkleColor := p.Accent
	vector.DrawFilledCircle(img, centerX-5, centerY-10, 3, sparkleColor, true)
	vector.DrawFilledCircle(img, centerX+3, centerY-5, 2, sparkleColor, true)

	return img
}

// generateMossTruffle crée l'icône d'une truffe de mousse
func generateMossTruffle(size int, p ResourcePalette, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)
	centerX, centerY := float32(size/2), float32(size/2)

	scale := float32(1.0)
	switch stage {
	case "bourgeon":
		scale = 0.8
	case "pousse":
		scale = 0.9
	}

	// Forme bosselée
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/4)*scale, p.Primary, true)
	vector.DrawFilledCircle(img, centerX-10*scale, centerY-8*scale, float32(size/6)*scale, p.Primary, true)
	vector.DrawFilledCircle(img, centerX+8*scale, centerY+5*scale, float32(size/6)*scale, p.Primary, true)
	// Points de mousse
	vector.DrawFilledCircle(img, centerX-5*scale, centerY+10*scale, 4*scale, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX+12*scale, centerY-5*scale, 3*scale, p.Accent, true)
	return img
}

// generateVoidBloom crée l'icône d'une fleur du vide
func generateVoidBloom(size int, p ResourcePalette, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)
	centerX, centerY := float32(size/2), float32(size/2)

	scale := float32(1.0)
	switch stage {
	case "graine":
		scale = 0.8
	case "éclosion":
		scale = 0.9
	}

	// Pétales éthérés
	for i := 0; i < 6; i++ {
		angle := float64(i) * 3.14159 * 2 / 6
		px := centerX + float32(math.Cos(angle))*20*scale
		py := centerY + float32(math.Sin(angle))*20*scale
		vector.DrawFilledCircle(img, px, py, 12*scale, p.Primary, true)
	}
	// Noyau sombre
	vector.DrawFilledCircle(img, centerX, centerY, 10*scale, p.Secondary, true)
	// Aura
	if stage == "pleine" {
		vector.DrawFilledCircle(img, centerX, centerY, 6*scale, p.Accent, true)
	}
	return img
}

// generateEchoCrystal crée l'icône d'un cristal d'écho
func generateEchoCrystal(size int, p ResourcePalette, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)
	centerX, centerY := float32(size/2), float32(size/2)

	scale := float32(1.0)
	if stage == "vibrant" {
		scale = 0.85
	}

	// Prismes
	vector.DrawFilledRect(img, centerX-10*scale, centerY-20*scale, 20*scale, 40*scale, p.Primary, true)
	vector.DrawFilledRect(img, centerX-22*scale, centerY-5*scale, 15*scale, 25*scale, p.Secondary, true)
	vector.DrawFilledRect(img, centerX+7*scale, centerY-15*scale, 15*scale, 30*scale, p.Secondary, true)
	// Brillance
	vector.DrawFilledCircle(img, centerX, centerY-10*scale, 5*scale, p.Accent, true)
	// Ondes
	if stage == "résonnant" {
		vector.StrokeCircle(img, centerX, centerY, 30*scale, 2*scale, p.Accent, true)
	}
	return img
}

// generateSandCore crée l'icône d'un noyau de sable
func generateSandCore(size int, p ResourcePalette, stage string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)
	centerX, centerY := float32(size/2), float32(size/2)

	scale := float32(1.0)
	if stage == "instable" {
		scale = 0.85
	}

	// Sphère centrale
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/5)*scale, p.Primary, true)
	// Anneau de sable
	vector.StrokeCircle(img, centerX, centerY, 25*scale, 4*scale, p.Secondary, true)
	// Étincelles de chaleur
	if stage == "stable" {
		vector.DrawFilledRect(img, centerX-25*scale, centerY-25*scale, 4*scale, 4*scale, p.Accent, true)
		vector.DrawFilledRect(img, centerX+20*scale, centerY+20*scale, 4*scale, 4*scale, p.Accent, true)
	}
	return img
}

// generateGenericResource crée une icône de ressource générique
func generateGenericResource(size int, p ResourcePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Cercle principal
	radius := float32(size / 3)
	vector.DrawFilledCircle(img, centerX, centerY, radius, p.Primary, true)
	vector.DrawFilledCircle(img, centerX, centerY, radius-5, p.Secondary, true)

	// Point central
	vector.DrawFilledCircle(img, centerX, centerY, 8, p.Accent, true)

	return img
}
