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
)

// generateDreamberry crée l'icône d'une baie onirique
func generateDreamberry(size int, p ResourcePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Feuilles
	leafColor := p.Secondary
	vector.DrawFilledCircle(img, centerX-12, centerY-15, 8, leafColor, true)
	vector.DrawFilledCircle(img, centerX+12, centerY-15, 8, leafColor, true)
	vector.DrawFilledCircle(img, centerX, centerY-20, 10, leafColor, true)

	// Baie principale
	berryRadius := float32(size / 3)
	vector.DrawFilledCircle(img, centerX, centerY+5, berryRadius, p.Primary, true)

	// Reflet brillant
	vector.DrawFilledCircle(img, centerX-8, centerY-2, berryRadius/3, p.Accent, true)

	// Petites baies secondaires
	vector.DrawFilledCircle(img, centerX-18, centerY+15, 6, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX+18, centerY+15, 6, p.Secondary, true)

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
func generateWhisperingHerb(size int, p ResourcePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	baseY := float32(size - 10)

	// Tige principale
	stemColor := p.Secondary
	vector.DrawFilledRect(img, centerX-2, baseY-30, 4, 30, stemColor, true)

	// Feuilles ondulantes
	leafColor := p.Primary
	// Feuille gauche
	vector.DrawFilledCircle(img, centerX-12, baseY-20, 10, leafColor, true)
	vector.DrawFilledCircle(img, centerX-8, baseY-20, 6, p.Bg, true) // Masque
	// Feuille droite
	vector.DrawFilledCircle(img, centerX+12, baseY-25, 10, leafColor, true)
	vector.DrawFilledCircle(img, centerX+8, baseY-25, 6, p.Bg, true) // Masque
	// Feuille haute
	vector.DrawFilledCircle(img, centerX, baseY-40, 10, leafColor, true)
	vector.DrawFilledCircle(img, centerX, baseY-35, 6, p.Bg, true) // Masque

	// Effet de "murmure" (ondes sonores)
	soundColor := p.Accent
	vector.StrokeLine(img, centerX+20, baseY-45, centerX+28, baseY-50, 2, soundColor, true)
	vector.StrokeLine(img, centerX+22, baseY-42, centerX+30, baseY-45, 2, soundColor, true)

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
