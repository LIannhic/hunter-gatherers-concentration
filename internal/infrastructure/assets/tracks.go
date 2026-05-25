// Package assets - Indices de traces (mud, griffes, etc.)
//
// ⚠️  ASSETS TEMPORAIRES / PLACEHOLDERS
// Ces indices sont générés procéduralement avec fonds transparents
// pour se superposer aux autres assets. À remplacer par des sprites
// finaux avant la release.
//
// TODO: Remplacer par des assets finaux avant release
package assets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TrackPalette contient les couleurs pour chaque type de trace
type TrackPalette struct {
	Primary   color.Color
	Secondary color.Color
	Shadow    color.Color
	Alpha     uint8 // Transparence
}

// Palettes des traces
var (
	// Mud - Boue / empreintes dans la terre
	MudPalette = TrackPalette{
		Primary:   color.RGBA{120, 100, 80, 180},  // Brun terreux semi-transparent
		Secondary: color.RGBA{150, 130, 110, 150}, // Brun clair
		Shadow:    color.RGBA{80, 60, 40, 200},    // Brun foncé
		Alpha:     180,
	}

	// Claws - Griffes / marques de griffes
	ClawsPalette = TrackPalette{
		Primary:   color.RGBA{200, 100, 100, 150}, // Rouge-brun semi-transparent
		Secondary: color.RGBA{255, 150, 150, 120}, // Rose clair
		Shadow:    color.RGBA{150, 50, 50, 200},   // Rouge foncé
		Alpha:     150,
	}

	// BrokenGrass - Herbe cassée / végétation écrasée
	BrokenGrassPalette = TrackPalette{
		Primary:   color.RGBA{100, 140, 60, 140},  // Vert herbe semi-transparent
		Secondary: color.RGBA{150, 180, 100, 110}, // Vert clair
		Shadow:    color.RGBA{70, 100, 40, 180},   // Vert foncé
		Alpha:     140,
	}

	// Footprints - Empreintes de pas
	FootprintsPalette = TrackPalette{
		Primary:   color.RGBA{100, 100, 100, 130}, // Gris semi-transparent
		Secondary: color.RGBA{150, 150, 150, 100}, // Gris clair
		Shadow:    color.RGBA{60, 60, 60, 170},    // Gris foncé
		Alpha:     130,
	}

	// IntentBeam - Rayon d'intention d'attaque
	IntentBeamPalette = TrackPalette{
		Primary:   color.RGBA{255, 100, 100, 180}, // Rouge lumineux
		Secondary: color.RGBA{255, 200, 200, 150}, // Rose très clair
		Shadow:    color.RGBA{200, 50, 50, 220},   // Rouge intense
		Alpha:     180,
	}
)

// generateMudTrack crée l'indice de trace dans la boue
func generateMudTrack(size int, p TrackPalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	// Fond transparent
	img.Fill(color.RGBA{0, 0, 0, 0})

	centerX := float32(size / 2)
	centerY := float32(size/2) + 8 // Décalé vers le bas

	// Empreinte principale (irrégulière)
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/5), p.Primary, true)
	// Ombres pour donner de la profondeur
	vector.DrawFilledCircle(img, centerX-3, centerY+2, float32(size/6), p.Shadow, true)
	vector.DrawFilledCircle(img, centerX+2, centerY-2, float32(size/7), p.Secondary, true)

	// Détails de boue éclaboussée
	vector.DrawFilledCircle(img, centerX-8, centerY-8, 2, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX+7, centerY+7, 2, p.Secondary, true)

	return img
}

// generateClawsTrack crée l'indice de griffes (3 petites encoches discrètes)
func generateClawsTrack(size int, p TrackPalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	// Fond transparent
	img.Fill(color.RGBA{0, 0, 0, 0})

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Trois petites encoches discrètes en haut
	encokeWidth := float32(1.5)
	encokeLengths := []float32{8, 6, 8} // Longueurs variables pour plus de discrétion

	// Gauche
	vector.StrokeLine(img, centerX-6, centerY-8, centerX-6, centerY-8-encokeLengths[0], encokeWidth, p.Shadow, true)
	// Centre
	vector.StrokeLine(img, centerX, centerY-12, centerX, centerY-12-encokeLengths[1], encokeWidth, p.Shadow, true)
	// Droite
	vector.StrokeLine(img, centerX+6, centerY-8, centerX+6, centerY-8-encokeLengths[2], encokeWidth, p.Shadow, true)

	return img
}

// generateBrokenGrassTrack crée l'indice d'herbe cassée
func generateBrokenGrassTrack(size int, p TrackPalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	// Fond transparent
	img.Fill(color.RGBA{0, 0, 0, 0})

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Herbe cassée - plusieurs traits croisés
	// Trait principal
	vector.StrokeLine(img, centerX-10, centerY-8, centerX+10, centerY+8, 2, p.Primary, true)
	// Trait secondaire
	vector.StrokeLine(img, centerX-8, centerY+10, centerX+12, centerY-6, 2, p.Primary, true)
	// Petits brins
	vector.StrokeLine(img, centerX-5, centerY-12, centerX-2, centerY-6, 1, p.Secondary, true)
	vector.StrokeLine(img, centerX+6, centerY+10, centerX+8, centerY+4, 1, p.Secondary, true)

	// Tache d'humidité ou d'écrasement
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/6), p.Shadow, true)

	return img
}

// generateFootprintTrack crée l'indice d'empreinte de pas
func generateFootprintTrack(size int, p TrackPalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	// Fond transparent
	img.Fill(color.RGBA{0, 0, 0, 0})

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Empreinte de pied - ellipse
	vector.DrawFilledCircle(img, centerX, centerY+2, float32(size/4), p.Primary, true)
	// Doigts/orteils
	vector.DrawFilledCircle(img, centerX-5, centerY-6, 2, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX-2, centerY-8, 2, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX+2, centerY-8, 2, p.Secondary, true)
	vector.DrawFilledCircle(img, centerX+5, centerY-6, 2, p.Secondary, true)

	// Talon plus prononcé
	vector.DrawFilledCircle(img, centerX, centerY+8, float32(size/5), p.Shadow, true)

	return img
}

// generateIntentBeam crée le rayon d'intention d'attaque
func generateIntentBeam(size int, p TrackPalette) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	// Fond transparent
	img.Fill(color.RGBA{0, 0, 0, 0})

	centerX := float32(size / 2)
	centerY := float32(size / 2)

	// Faisceau principal (gradient simulé par des couches)
	vector.DrawFilledRect(img, centerX-3, 0, 6, float32(size), p.Primary, true)
	vector.DrawFilledRect(img, centerX-2, 2, 4, float32(size-4), p.Secondary, true)

	// Pointe du rayon (arête)
	vector.DrawFilledCircle(img, centerX, 2, 3, color.RGBA{255, 255, 255, 220}, true)

	// Halo/aura
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/3),
		color.RGBA{255, 100, 100, 80}, true)

	return img
}

// GenerateTrackAsset génère l'asset visual d'une trace selon son type
func GenerateTrackAsset(kind string, size int) *ebiten.Image {
	switch kind {
	case "mud":
		return generateMudTrack(size, MudPalette)
	case "claws":
		return generateClawsTrack(size, ClawsPalette)
	case "broken_grass":
		return generateBrokenGrassTrack(size, BrokenGrassPalette)
	case "footprints":
		return generateFootprintTrack(size, FootprintsPalette)
	case "intent_beam":
		return generateIntentBeam(size, IntentBeamPalette)
	default:
		// Fallback : retourne une trace générique
		return generateMudTrack(size, MudPalette)
	}
}
