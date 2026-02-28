// Package assets - Icônes des créatures
//
// ⚠️  ASSETS TEMPORAIRES / PLACEHOLDERS
// Ces icônes de créatures sont générées procéduralement et servent de
// placeholders temporaires. Elles devront être remplacées par des sprites
// animés finaux avant la release.
//
// TODO: Remplacer par des assets finaux avant release
package assets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// CreaturePalette contient les couleurs pour chaque type de créature
type CreaturePalette struct {
	Body      color.Color
	Highlight color.Color
	Shadow    color.Color
	Eye       color.Color
	Bg        color.Color
}

// Palettes des créatures
var (
	// Lumifly - Luciole lumineuse
	LumiflyPalette = CreaturePalette{
		Body:      color.RGBA{255, 220, 100, 255}, // Jaune doré
		Highlight: color.RGBA{255, 255, 200, 255}, // Jaune brillant
		Shadow:    color.RGBA{200, 160, 50, 255},  // Doré foncé
		Eye:       color.RGBA{50, 30, 0, 255},     // Noir/brun
		Bg:        color.RGBA{60, 55, 70, 255},
	}

	// Shadowstalker - Rôdeur des ombres
	ShadowstalkerPalette = CreaturePalette{
		Body:      color.RGBA{80, 60, 90, 255},    // Violet sombre
		Highlight: color.RGBA{120, 90, 140, 255},  // Violet clair
		Shadow:    color.RGBA{50, 35, 60, 255},    // Très sombre
		Eye:       color.RGBA{255, 80, 80, 255},   // Rouge lumineux
		Bg:        color.RGBA{45, 35, 50, 255},
	}

	// Burrower - Fouisseur
	BurrowerPalette = CreaturePalette{
		Body:      color.RGBA{160, 130, 100, 255}, // Brun terreux
		Highlight: color.RGBA{200, 170, 140, 255}, // Beige clair
		Shadow:    color.RGBA{120, 90, 60, 255},   // Brun foncé
		Eye:       color.RGBA{60, 100, 60, 255},   // Vert sombre
		Bg:        color.RGBA{60, 55, 50, 255},
	}

	// Flutterwing - Ailevoltige
	FlutterwingPalette = CreaturePalette{
		Body:      color.RGBA{150, 200, 220, 255}, // Bleu ciel
		Highlight: color.RGBA{200, 240, 255, 255}, // Bleu très clair
		Shadow:    color.RGBA{100, 150, 180, 255}, // Bleu gris
		Eye:       color.RGBA{80, 60, 120, 255},   // Violet foncé
		Bg:        color.RGBA{50, 65, 80, 255},
	}
)

// generateLumifly crée l'icône d'une luciole
func generateLumifly(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Ailes (translucides)
	wingColor := color.RGBA{255, 255, 255, 100}
	// Aile gauche
	vector.DrawFilledRect(img, centerX-25, centerY-10, 18, 12, wingColor, true)
	vector.DrawFilledRect(img, centerX-22, centerY-5, 12, 15, wingColor, true)
	// Aile droite
	vector.DrawFilledRect(img, centerX+7, centerY-10, 18, 12, wingColor, true)
	vector.DrawFilledRect(img, centerX+10, centerY-5, 12, 15, wingColor, true)

	// Corps (abdomen lumineux)
	bodyColor := p.Body
	vector.DrawFilledCircle(img, centerX, centerY+8, 10, bodyColor, true)
	// Brillance magique
	vector.DrawFilledCircle(img, centerX, centerY+8, 6, p.Highlight, true)
	vector.DrawFilledCircle(img, centerX-2, centerY+6, 2, color.RGBA{255, 255, 255, 200}, true)

	// Tête
	vector.DrawFilledCircle(img, centerX, centerY-8, 8, p.Shadow, true)
	// Yeux
	vector.DrawFilledCircle(img, centerX-3, centerY-10, 2, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+3, centerY-10, 2, p.Eye, true)

	// Antennes
	vector.StrokeLine(img, centerX-3, centerY-14, centerX-8, centerY-22, 2, p.Shadow, true)
	vector.StrokeLine(img, centerX+3, centerY-14, centerX+8, centerY-22, 2, p.Shadow, true)

	return img
}

// generateShadowstalker crée l'icône d'un rôdeur des ombres
func generateShadowstalker(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps sombre
	bodyColor := p.Body
	vector.DrawFilledRect(img, centerX-15, centerY-8, 30, 22, bodyColor, true)

	// Tête triangulaire
	vector.DrawFilledRect(img, centerX-12, centerY-18, 24, 12, bodyColor, true)

	// Oreilles pointues
	vector.DrawFilledRect(img, centerX-18, centerY-22, 6, 12, p.Shadow, true)
	vector.DrawFilledRect(img, centerX+12, centerY-22, 6, 12, p.Shadow, true)

	// Yeux rouges lumineux
	eyeColor := p.Eye
	vector.DrawFilledCircle(img, centerX-6, centerY-12, 4, eyeColor, true)
	vector.DrawFilledCircle(img, centerX+6, centerY-12, 4, eyeColor, true)
	// Brillance des yeux
	vector.DrawFilledCircle(img, centerX-7, centerY-13, 1, color.RGBA{255, 150, 150, 255}, true)
	vector.DrawFilledCircle(img, centerX+5, centerY-13, 1, color.RGBA{255, 150, 150, 255}, true)

	// Griffes
	clawColor := p.Shadow
	vector.DrawFilledRect(img, centerX-20, centerY+8, 4, 12, clawColor, true)
	vector.DrawFilledRect(img, centerX+16, centerY+8, 4, 12, clawColor, true)

	// Aura sombre
	auraColor := color.RGBA{60, 40, 70, 80}
	vector.DrawFilledCircle(img, centerX, centerY, 28, auraColor, true)

	return img
}

// generateBurrower crée l'icône d'un fouisseur
func generateBurrower(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps cylindrique
	bodyColor := p.Body
	vector.DrawFilledRect(img, centerX-12, centerY-15, 24, 35, bodyColor, true)

	// Bandes sur le corps
	stripeColor := p.Highlight
	vector.DrawFilledRect(img, centerX-10, centerY-8, 20, 4, stripeColor, true)
	vector.DrawFilledRect(img, centerX-10, centerY+5, 20, 4, stripeColor, true)

	// Museau pointu
	vector.DrawFilledRect(img, centerX-6, centerY-22, 12, 10, p.Shadow, true)

	// Yeux petits
	vector.DrawFilledCircle(img, centerX-4, centerY-12, 2, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+4, centerY-12, 2, p.Eye, true)

	// Pattes de fouisseur
	legColor := p.Shadow
	// Pattes avant
	vector.DrawFilledRect(img, centerX-20, centerY-2, 8, 4, legColor, true)
	vector.DrawFilledRect(img, centerX+12, centerY-2, 8, 4, legColor, true)
	// Pattes arrière
	vector.DrawFilledRect(img, centerX-22, centerY+12, 10, 4, legColor, true)
	vector.DrawFilledRect(img, centerX+12, centerY+12, 10, 4, legColor, true)

	// Griffes
	vector.DrawFilledRect(img, centerX-22, centerY, 3, 6, color.RGBA{80, 60, 40, 255}, true)
	vector.DrawFilledRect(img, centerX+19, centerY, 3, 6, color.RGBA{80, 60, 40, 255}, true)

	return img
}

// generateFlutterwing crée l'icône d'une ailevoltige
func generateFlutterwing(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Grandes ailes papillon
	wingColor := p.Body
	highlightColor := p.Highlight

	// Aile gauche
	vector.DrawFilledCircle(img, centerX-18, centerY-5, 15, wingColor, true)
	vector.DrawFilledCircle(img, centerX-15, centerY-8, 8, highlightColor, true)
	// Aile droite
	vector.DrawFilledCircle(img, centerX+18, centerY-5, 15, wingColor, true)
	vector.DrawFilledCircle(img, centerX+15, centerY-8, 8, highlightColor, true)

	// Corps fin
	vector.DrawFilledRect(img, centerX-2, centerY-15, 4, 30, p.Shadow, true)

	// Tête
	vector.DrawFilledCircle(img, centerX, centerY-18, 6, p.Shadow, true)

	// Yeux
	vector.DrawFilledCircle(img, centerX-2, centerY-20, 2, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+2, centerY-20, 2, p.Eye, true)

	// Antennes
	vector.StrokeLine(img, centerX-1, centerY-23, centerX-4, centerY-28, 2, p.Shadow, true)
	vector.StrokeLine(img, centerX+1, centerY-23, centerX+4, centerY-28, 2, p.Shadow, true)
	// Boules aux antennes
	vector.DrawFilledCircle(img, centerX-4, centerY-28, 2, p.Highlight, true)
	vector.DrawFilledCircle(img, centerX+4, centerY-28, 2, p.Highlight, true)

	return img
}

// generateGenericCreature crée une icône de créature générique
func generateGenericCreature(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps simple
	radius := float32(size / 3)
	vector.DrawFilledCircle(img, centerX, centerY, radius, p.Body, true)
	vector.DrawFilledCircle(img, centerX-5, centerY-5, 5, p.Highlight, true)

	// Yeux
	vector.DrawFilledCircle(img, centerX-8, centerY-3, 4, color.White, true)
	vector.DrawFilledCircle(img, centerX+8, centerY-3, 4, color.White, true)
	vector.DrawFilledCircle(img, centerX-8, centerY-3, 2, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+8, centerY-3, 2, p.Eye, true)

	return img
}


