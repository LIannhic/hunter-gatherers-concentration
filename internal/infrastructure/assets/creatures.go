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
	"math"

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

	// Specter - Spectre
	SpecterPalette = CreaturePalette{
		Body:      color.RGBA{200, 200, 255, 180}, // Bleu spectral translucide
		Highlight: color.RGBA{255, 255, 255, 220}, // Blanc brillant
		Shadow:    color.RGBA{100, 100, 150, 150}, // Bleu profond
		Eye:       color.RGBA{0, 255, 255, 255},   // Cyan électrique
		Bg:        color.RGBA{40, 40, 60, 255},
	}

	// Echo Hound - Chien d'écho
	EchoHoundPalette = CreaturePalette{
		Body:      color.RGBA{70, 70, 70, 255},    // Gris foncé (béton/ombre)
		Highlight: color.RGBA{180, 180, 180, 255}, // Argenté
		Shadow:    color.RGBA{30, 30, 30, 255},    // Noir
		Eye:       color.RGBA{0, 255, 0, 255},     // Vert radioactif
		Bg:        color.RGBA{50, 50, 50, 255},
	}

	// Moss Monkey - Singe mousse
	MossMonkeyPalette = CreaturePalette{
		Body:      color.RGBA{100, 140, 60, 255},  // Vert mousse
		Highlight: color.RGBA{160, 200, 100, 255}, // Vert clair (mousse fraîche)
		Shadow:    color.RGBA{60, 80, 40, 255},    // Vert forêt profond
		Eye:       color.RGBA{255, 200, 0, 255},   // Ambre/Jaune
		Bg:        color.RGBA{50, 45, 40, 255},    // Brun terreux
	}

	// Stonewarden - Gardien de pierre
	StonewardenPalette = CreaturePalette{
		Body:      color.RGBA{120, 120, 130, 255}, // Gris pierre
		Highlight: color.RGBA{180, 180, 190, 255}, // Gris clair
		Shadow:    color.RGBA{70, 70, 80, 255},    // Gris sombre
		Eye:       color.RGBA{100, 200, 255, 255}, // Bleu éthéré
		Bg:        color.RGBA{45, 45, 50, 255},
	}

	// Fleeing Sprite - Esprit fuyant
	FleeingSpritePalette = CreaturePalette{
		Body:      color.RGBA{200, 255, 255, 200}, // Cyan clair translucide
		Highlight: color.RGBA{255, 255, 255, 255}, // Blanc pur
		Shadow:    color.RGBA{100, 200, 255, 150}, // Bleu ciel
		Eye:       color.RGBA{255, 150, 0, 255},   // Orange (énergie)
		Bg:        color.RGBA{35, 40, 50, 255},
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

// generateSpecter crée l'icône d'un spectre
func generateSpecter(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps vaporeux (translucide)
	bodyColor := p.Body
	vector.DrawFilledCircle(img, centerX, centerY-5, 18, bodyColor, true)

	// Traîne vaporeuse
	vector.DrawFilledRect(img, centerX-14, centerY-5, 28, 20, bodyColor, true)
	vector.DrawFilledCircle(img, centerX-7, centerY+15, 8, bodyColor, true)
	vector.DrawFilledCircle(img, centerX+7, centerY+15, 8, bodyColor, true)

	// Yeux cyan électriques
	vector.DrawFilledCircle(img, centerX-6, centerY-8, 3, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+6, centerY-8, 3, p.Eye, true)

	// Halo de lumière
	haloColor := color.RGBA{150, 255, 255, 50}
	vector.DrawFilledCircle(img, centerX, centerY-5, 25, haloColor, true)

	return img
}

// generateEchoHound crée l'icône d'un chien d'écho
func generateEchoHound(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps rectangulaire (plus "canin")
	vector.DrawFilledRect(img, centerX-20, centerY-5, 40, 15, p.Body, true)

	// Tête
	vector.DrawFilledRect(img, centerX-5, centerY-20, 25, 18, p.Body, true)

	// Oreilles triangulaires (hautes)
	vector.DrawFilledRect(img, centerX-2, centerY-28, 6, 10, p.Shadow, true)
	vector.DrawFilledRect(img, centerX+12, centerY-28, 6, 10, p.Shadow, true)

	// Yeux verts radioactifs
	vector.DrawFilledCircle(img, centerX+8, centerY-14, 4, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+18, centerY-14, 4, p.Eye, true)

	// Pattes fines (indiquant la vitesse)
	vector.DrawFilledRect(img, centerX-15, centerY+10, 4, 15, p.Shadow, true)
	vector.DrawFilledRect(img, centerX-5, centerY+10, 4, 15, p.Shadow, true)
	vector.DrawFilledRect(img, centerX+5, centerY+10, 4, 15, p.Shadow, true)
	vector.DrawFilledRect(img, centerX+15, centerY+10, 4, 15, p.Shadow, true)

	return img
}

// generateMossMonkey crée l'icône d'un singe mousse
func generateMossMonkey(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps (boule de mousse)
	vector.DrawFilledCircle(img, centerX, centerY+5, 18, p.Body, true)
	// Touffes de mousse
	vector.DrawFilledCircle(img, centerX-10, centerY-5, 8, p.Body, true)
	vector.DrawFilledCircle(img, centerX+10, centerY-5, 8, p.Body, true)
	vector.DrawFilledCircle(img, centerX, centerY-12, 10, p.Highlight, true)

	// Tête (plus sombre)
	vector.DrawFilledCircle(img, centerX, centerY-5, 12, p.Shadow, true)

	// Oreilles
	vector.DrawFilledCircle(img, centerX-12, centerY-8, 5, p.Shadow, true)
	vector.DrawFilledCircle(img, centerX+12, centerY-8, 5, p.Shadow, true)

	// Yeux ambre
	vector.DrawFilledCircle(img, centerX-4, centerY-7, 3, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+4, centerY-7, 3, p.Eye, true)
	// Pupilles
	vector.DrawFilledCircle(img, centerX-4, centerY-7, 1, color.Black, true)
	vector.DrawFilledCircle(img, centerX+4, centerY-7, 1, color.Black, true)

	// Bras (longs pour un singe)
	vector.StrokeLine(img, centerX-15, centerY+5, centerX-22, centerY+20, 3, p.Body, true)
	vector.StrokeLine(img, centerX+15, centerY+5, centerX+22, centerY+20, 3, p.Body, true)

	return img
}

// generateStonewarden crée l'icône d'un gardien de pierre
func generateStonewarden(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Corps (forme de bloc monolithique)
	vector.DrawFilledRect(img, centerX-18, centerY-10, 36, 30, p.Body, true)
	// Facettes de pierre
	vector.DrawFilledRect(img, centerX-18, centerY-10, 36, 4, p.Highlight, true)
	vector.DrawFilledRect(img, centerX-18, centerY+16, 36, 4, p.Shadow, true)

	// Épaules
	vector.DrawFilledRect(img, centerX-24, centerY-5, 12, 12, p.Shadow, true)
	vector.DrawFilledRect(img, centerX+12, centerY-5, 12, 12, p.Shadow, true)

	// Tête (bloc plus petit posé dessus)
	vector.DrawFilledRect(img, centerX-10, centerY-22, 20, 15, p.Body, true)
	// Bordure supérieure tête
	vector.DrawFilledRect(img, centerX-10, centerY-22, 20, 2, p.Highlight, true)

	// Yeux (fentes éthérées)
	vector.DrawFilledRect(img, centerX-6, centerY-16, 4, 3, p.Eye, true)
	vector.DrawFilledRect(img, centerX+2, centerY-16, 4, 3, p.Eye, true)

	// Lichen/Mousse (touches de vert sur le corps)
	mossColor := color.RGBA{80, 120, 60, 200}
	vector.DrawFilledCircle(img, centerX-10, centerY+5, 4, mossColor, true)
	vector.DrawFilledCircle(img, centerX+12, centerY-2, 3, mossColor, true)
	vector.DrawFilledCircle(img, centerX-5, centerY+20, 5, mossColor, true)

	return img
}

// generateFleeingSprite crée l'icône d'un esprit fuyant (étincelle vive)
func generateFleeingSprite(size int, p CreaturePalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(p.Bg)

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Éclats d'énergie (forme en étoile)
	sparkColor := p.Body
	for i := 0; i < 4; i++ {
		angle := float64(i) * 3.14159 / 2
		dx := float32(math.Cos(angle)) * 20
		dy := float32(math.Sin(angle)) * 20
		vector.StrokeLine(img, centerX-dx, centerY-dy, centerX+dx, centerY+dy, 4, sparkColor, true)
	}

	// Noyau central
	vector.DrawFilledCircle(img, centerX, centerY, 12, p.Highlight, true)

	// Traînée de mouvement (vitesse)
	vector.DrawFilledCircle(img, centerX-15, centerY+10, 6, p.Shadow, true)
	vector.DrawFilledCircle(img, centerX-25, centerY+18, 4, p.Shadow, true)

	// Yeux expressifs
	vector.DrawFilledCircle(img, centerX-3, centerY-2, 3, p.Eye, true)
	vector.DrawFilledCircle(img, centerX+3, centerY-2, 3, p.Eye, true)

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
