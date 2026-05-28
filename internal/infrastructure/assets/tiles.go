// Package assets - Génération des tuiles avec motifs visuels
// 
// ⚠️  ASSETS TEMPORAIRES / PLACEHOLDERS
// Ces assets sont générés procéduralement et servent de placeholders temporaires
// pour le développement. Ils devront être remplacés par des assets finaux
// (sprites, pixel art, etc.) avant la release.
//
// Avantages des assets temporaires :
// - Pas de dépendances externes pendant le développement
// - Taille de repo réduite
// - Facilement modifiables via code
// - Libres de droit (CC0)
//
// TODO: Remplacer par des assets finaux avant release
package assets

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TileTheme représente un thème visuel pour les tuiles
type TileTheme struct {
	HiddenBg        color.Color
	HiddenPattern   color.Color
	HiddenBorder    color.Color
	RevealedBg      color.Color
	RevealedPattern color.Color
	MatchedBg       color.Color
	MatchedPattern  color.Color
}

// Themes disponibles (tous libres de droit - générés procéduralement)
var (
	// ThemeDefault - Thème classique bleu-violet
	ThemeDefault = TileTheme{
		HiddenBg:        color.RGBA{45, 40, 65, 255},
		HiddenPattern:   color.RGBA{65, 60, 90, 255},
		HiddenBorder:    color.RGBA{100, 95, 130, 255},
		RevealedBg:      color.RGBA{55, 55, 75, 255},
		RevealedPattern: color.RGBA{75, 75, 95, 255},
		MatchedBg:       color.RGBA{40, 90, 50, 255},
		MatchedPattern:  color.RGBA{60, 130, 70, 255},
	}

	// ThemeForest - Thème nature/forestier
	ThemeForest = TileTheme{
		HiddenBg:        color.RGBA{35, 55, 35, 255},
		HiddenPattern:   color.RGBA{50, 80, 50, 255},
		HiddenBorder:    color.RGBA{80, 120, 70, 255},
		RevealedBg:      color.RGBA{45, 70, 45, 255},
		RevealedPattern: color.RGBA{65, 100, 65, 255},
		MatchedBg:       color.RGBA{60, 100, 60, 255},
		MatchedPattern:  color.RGBA{90, 140, 80, 255},
	}

	// ThemeCave - Thème caverne/obscur
	ThemeCave = TileTheme{
		HiddenBg:        color.RGBA{40, 35, 45, 255},
		HiddenPattern:   color.RGBA{55, 50, 60, 255},
		HiddenBorder:    color.RGBA{90, 80, 100, 255},
		RevealedBg:      color.RGBA{50, 45, 55, 255},
		RevealedPattern: color.RGBA{70, 65, 80, 255},
		MatchedBg:       color.RGBA{70, 60, 80, 255},
		MatchedPattern:  color.RGBA{100, 90, 120, 255},
	}

	// ThemeSwamp - Thème marais/mystique
	ThemeSwamp = TileTheme{
		HiddenBg:        color.RGBA{35, 50, 40, 255},
		HiddenPattern:   color.RGBA{50, 70, 55, 255},
		HiddenBorder:    color.RGBA{70, 100, 75, 255},
		RevealedBg:      color.RGBA{45, 65, 50, 255},
		RevealedPattern: color.RGBA{65, 90, 70, 255},
		MatchedBg:       color.RGBA{55, 85, 60, 255},
		MatchedPattern:  color.RGBA{80, 120, 85, 255},
	}
)

// generateTileHidden crée une tuile face cachée avec motif décoratif
func generateTileHidden(size int, theme TileTheme) *ebiten.Image {
	img := ebiten.NewImage(size, size)

	// Fond avec dégradé subtil
	img.Fill(theme.HiddenBg)

	// Motif géométrique au centre (style "dos de carte")
	centerX := float32(size / 2)
	centerY := float32(size / 2)
	patternColor := theme.HiddenPattern

	// Cercle central principal
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/4), patternColor, true)

	// Motif en losange autour
	diamondSize := float32(size / 5)
	vector.DrawFilledRect(img, centerX-diamondSize/2, 8, diamondSize, diamondSize, patternColor, true)
	vector.DrawFilledRect(img, centerX-diamondSize/2, float32(size)-8-diamondSize, diamondSize, diamondSize, patternColor, true)
	vector.DrawFilledRect(img, 8, centerY-diamondSize/2, diamondSize, diamondSize, patternColor, true)
	vector.DrawFilledRect(img, float32(size)-8-diamondSize, centerY-diamondSize/2, diamondSize, diamondSize, patternColor, true)

	// Petits cercles aux coins
	cornerOffset := float32(size / 6)
	cornerRadius := float32(size / 12)
	vector.DrawFilledCircle(img, cornerOffset, cornerOffset, cornerRadius, patternColor, true)
	vector.DrawFilledCircle(img, float32(size)-cornerOffset, cornerOffset, cornerRadius, patternColor, true)
	vector.DrawFilledCircle(img, cornerOffset, float32(size)-cornerOffset, cornerRadius, patternColor, true)
	vector.DrawFilledCircle(img, float32(size)-cornerOffset, float32(size)-cornerOffset, cornerRadius, patternColor, true)

	// Bordure ornementale
	borderWidth := float32(3)
	vector.StrokeLine(img, 0, borderWidth/2, float32(size), borderWidth/2, borderWidth, theme.HiddenBorder, true)
	vector.StrokeLine(img, borderWidth/2, 0, borderWidth/2, float32(size), borderWidth, theme.HiddenBorder, true)
	vector.StrokeLine(img, float32(size)-borderWidth/2, 0, float32(size)-borderWidth/2, float32(size), borderWidth, theme.HiddenBorder, true)
	vector.StrokeLine(img, 0, float32(size)-borderWidth/2, float32(size), float32(size)-borderWidth/2, borderWidth, theme.HiddenBorder, true)

	// Coins ornementaux
	cornerSize := float32(8)
	vector.DrawFilledCircle(img, cornerSize, cornerSize, cornerSize/2, theme.HiddenBorder, true)
	vector.DrawFilledCircle(img, float32(size)-cornerSize, cornerSize, cornerSize/2, theme.HiddenBorder, true)
	vector.DrawFilledCircle(img, cornerSize, float32(size)-cornerSize, cornerSize/2, theme.HiddenBorder, true)
	vector.DrawFilledCircle(img, float32(size)-cornerSize, float32(size)-cornerSize, cornerSize/2, theme.HiddenBorder, true)

	return img
}

// generateTileRevealed crée une tuile révélée avec motif subtil
func generateTileRevealed(size int, theme TileTheme) *ebiten.Image {
	img := ebiten.NewImage(size, size)

	// Fond
	img.Fill(theme.RevealedBg)

	// Motif de grille subtil
	patternColor := theme.RevealedPattern
	gridSpacing := float32(size / 4)

	for i := float32(1); i < 4; i++ {
		// Lignes verticales
		vector.StrokeLine(img, i*gridSpacing, 0, i*gridSpacing, float32(size), 1, patternColor, true)
		// Lignes horizontales
		vector.StrokeLine(img, 0, i*gridSpacing, float32(size), i*gridSpacing, 1, patternColor, true)
	}

	// Bordure fine
	borderColor := theme.HiddenBorder
	vector.StrokeRect(img, 1, 1, float32(size-2), float32(size-2), 2, borderColor, true)

	return img
}

// generateTileMatched crée une tuile appairée avec effet de réussite
func generateTileMatched(size int, theme TileTheme) *ebiten.Image {
	img := ebiten.NewImage(size, size)

	// Fond vert de succès
	img.Fill(theme.MatchedBg)

	// Motif de rayons émanant du centre
	centerX := float32(size / 2)
	centerY := float32(size / 2)
	patternColor := theme.MatchedPattern

	// Rayons
	for angle := 0; angle < 360; angle += 30 {
		rad := float64(angle) * math.Pi / 180
		x2 := centerX + float32(math.Cos(rad))*float32(size/2)
		y2 := centerY + float32(math.Sin(rad))*float32(size/2)
		vector.StrokeLine(img, centerX, centerY, x2, y2, 2, patternColor, true)
	}

	// Cercle central
	vector.DrawFilledCircle(img, centerX, centerY, float32(size/6), patternColor, true)

	// Bordure épaisse de succès
	vector.StrokeRect(img, 2, 2, float32(size-4), float32(size-4), 3, color.RGBA{100, 200, 100, 255}, true)

	return img
}

// generateTileBlocked crée une tuile bloquée (Transparent + Croix rouge)
func generateTileBlocked(size int, theme TileTheme) *ebiten.Image {
	img := ebiten.NewImage(size, size) // Transparent par défaut

	padding := float32(size) * 0.25
	crossColor := color.RGBA{200, 0, 0, 255}
	vector.StrokeLine(img, padding, padding, float32(size)-padding, float32(size)-padding, 4, crossColor, true)
	vector.StrokeLine(img, float32(size)-padding, padding, padding, float32(size)-padding, 4, crossColor, true)

	return img
}

// generateTileTrap crée une tuile piège (Style Hidden mais avec une teinte différente)
func generateTileTrap(size int, theme TileTheme) *ebiten.Image {
	// Version légèrement plus sombre ou désaturée pour le piège révélé
	trapTheme := theme
	trapTheme.HiddenBg = color.RGBA{50, 45, 45, 255} // Teinte plus rougeâtre/sombre
	trapTheme.HiddenPattern = color.RGBA{70, 60, 60, 255}

	img := generateTileHidden(size, trapTheme)

	// On ajoute un petit indicateur de danger (subtile) au centre
	centerX, centerY := float32(size)/2, float32(size)/2
	vector.StrokeLine(img, centerX-5, centerY-5, centerX+5, centerY+5, 2, color.RGBA{150, 50, 50, 100}, true)
	vector.StrokeLine(img, centerX+5, centerY-5, centerX-5, centerY+5, 2, color.RGBA{150, 50, 50, 100}, true)

	return img
}

// generateTileSealed crée une tuile scellée (Transparent + Cadenas)
func generateTileSealed(size int, theme TileTheme) *ebiten.Image {
	img := ebiten.NewImage(size, size) // Transparent par défaut

	centerX := float32(size) / 2
	centerY := float32(size) / 2
	padlockColor := color.RGBA{220, 200, 100, 255} // Doré pour le cadenas

	// Corps du cadenas
	bodyW := float32(size) * 0.3
	bodyH := float32(size) * 0.25
	vector.DrawFilledRect(img, centerX-bodyW/2, centerY-2, bodyW, bodyH, padlockColor, true)

	// Anse du cadenas
	shackleRadius := float32(size) * 0.12
	vector.StrokeCircle(img, centerX, centerY-2, shackleRadius, 4, padlockColor, true)

	// Trou de serrure
	vector.DrawFilledCircle(img, centerX, centerY+bodyH/2-2, 3, color.RGBA{40, 40, 40, 255}, true)

	return img
}

// generateEmptySquare crée une image pour case vide (sol nu avec quadrillage)
func generateEmptySquare(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	// Fond très sombre
	img.Fill(color.RGBA{15, 15, 20, 255})

	// Quadrillage et bordures
	gridColor := color.RGBA{40, 40, 50, 255}
	vector.StrokeRect(img, 0, 0, float32(size), float32(size), 1, gridColor, true)

	// Lignes diagonales subtiles pour marquer le centre
	vector.StrokeLine(img, 0, 0, float32(size), float32(size), 0.5, gridColor, true)
	vector.StrokeLine(img, float32(size), 0, 0, float32(size), 0.5, gridColor, true)

	return img
}

// lighten augmente la luminosité d'une couleur
func lighten(c color.Color, amount int) color.Color {
	r, g, b, a := c.RGBA()
	// RGBA() retourne des valeurs sur 16 bits (0-65535)
	// On convertit en 8 bits (0-255)
	r8 := int(r >> 8)
	g8 := int(g >> 8)
	b8 := int(b >> 8)
	a8 := int(a >> 8)

	r8 = min(255, r8+amount)
	g8 = min(255, g8+amount)
	b8 = min(255, b8+amount)

	return color.RGBA{uint8(r8), uint8(g8), uint8(b8), uint8(a8)}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateFlipEffectOverlay crée une image pour l'effet de flip visuel
func generateFlipEffectOverlay(size int, direction string) *ebiten.Image {
	img := ebiten.NewImage(size, size)

	switch direction {
	case "top":
		// Ombre en haut
		for i := 0; i < size/3; i++ {
			alpha := uint8(60 - i*2)
			vector.StrokeLine(img, 0, float32(i), float32(size), float32(i), 1, color.RGBA{0, 0, 0, alpha}, true)
		}
	case "bottom":
		// Ombre en bas
		for i := 0; i < size/3; i++ {
			alpha := uint8(60 - i*2)
			vector.StrokeLine(img, 0, float32(size-i), float32(size), float32(size-i), 1, color.RGBA{0, 0, 0, alpha}, true)
		}
	case "left":
		// Ombre à gauche
		for i := 0; i < size/3; i++ {
			alpha := uint8(60 - i*2)
			vector.StrokeLine(img, float32(i), 0, float32(i), float32(size), 1, color.RGBA{0, 0, 0, alpha}, true)
		}
	case "right":
		// Ombre à droite
		for i := 0; i < size/3; i++ {
			alpha := uint8(60 - i*2)
			vector.StrokeLine(img, float32(size-i), 0, float32(size-i), float32(size), 1, color.RGBA{0, 0, 0, alpha}, true)
		}
	default:
		// Effet de brillance central
		vector.DrawFilledCircle(img, float32(size/2), float32(size/2), float32(size/3), color.RGBA{255, 255, 255, 40}, true)
	}

	return img
}

// generateDirectionIndicator crée un indicateur visuel de direction pour le flip
func generateDirectionIndicator(size int, dir string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	centerX := float32(size / 2)
	centerY := float32(size / 2)

	arrowColor := color.RGBA{255, 255, 100, 200}
	arrowSize := float32(size / 4)

	switch dir {
	case "top":
		vector.DrawFilledRect(img, centerX-2, centerY-arrowSize, 4, arrowSize, arrowColor, true)
		vector.DrawFilledRect(img, centerX-arrowSize/2, centerY-arrowSize, arrowSize, 4, arrowColor, true)
	case "bottom":
		vector.DrawFilledRect(img, centerX-2, centerY, 4, arrowSize, arrowColor, true)
		vector.DrawFilledRect(img, centerX-arrowSize/2, centerY+arrowSize-4, arrowSize, 4, arrowColor, true)
	case "left":
		vector.DrawFilledRect(img, centerX-arrowSize, centerY-2, arrowSize, 4, arrowColor, true)
		vector.DrawFilledRect(img, centerX-arrowSize, centerY-arrowSize/2, 4, arrowSize, arrowColor, true)
	case "right":
		vector.DrawFilledRect(img, centerX, centerY-2, arrowSize, 4, arrowColor, true)
		vector.DrawFilledRect(img, centerX+arrowSize-4, centerY-arrowSize/2, 4, arrowSize, arrowColor, true)
	}

	return img
}

// generateBiomeIcon crée une icône simplifiée pour représenter un biome sur la minimap (26x26)
func generateBiomeIcon(size int, biome string) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	centerX := float32(size) / 2
	centerY := float32(size) / 2
	iconColor := color.White

	switch biome {
	case "forest":
		// Silhouette de sapin simplifiée
		vector.DrawFilledRect(img, centerX-2, centerY+4, 4, 4, iconColor, true) // Tronc
		// Triangle du bas
		vector.StrokeLine(img, centerX-8, centerY+4, centerX+8, centerY+4, 2, iconColor, true)
		vector.StrokeLine(img, centerX-8, centerY+4, centerX, centerY-2, 2, iconColor, true)
		vector.StrokeLine(img, centerX+8, centerY+4, centerX, centerY-2, 2, iconColor, true)
		// Triangle du haut
		vector.StrokeLine(img, centerX-6, centerY-2, centerX+6, centerY-2, 2, iconColor, true)
		vector.StrokeLine(img, centerX-6, centerY-2, centerX, centerY-8, 2, iconColor, true)
		vector.StrokeLine(img, centerX+6, centerY-2, centerX, centerY-8, 2, iconColor, true)
	case "cave":
		// Stalactite / Entrée de grotte
		vector.StrokeLine(img, centerX-8, centerY+6, centerX-4, centerY-6, 2, iconColor, true)
		vector.StrokeLine(img, centerX-4, centerY-6, centerX+4, centerY-6, 2, iconColor, true)
		vector.StrokeLine(img, centerX+4, centerY-6, centerX+8, centerY+6, 2, iconColor, true)
		// Petit pic
		vector.DrawFilledRect(img, centerX-1, centerY-2, 2, 6, iconColor, true)
	case "desert":
		// Soleil
		vector.DrawFilledCircle(img, centerX, centerY, 5, iconColor, true)
		for i := 0; i < 8; i++ {
			angle := float64(i) * math.Pi / 4
			x1 := centerX + float32(math.Cos(angle))*7
			y1 := centerY + float32(math.Sin(angle))*7
			x2 := centerX + float32(math.Cos(angle))*10
			y2 := centerY + float32(math.Sin(angle))*10
			vector.StrokeLine(img, x1, y1, x2, y2, 1.5, iconColor, true)
		}
	case "swamp":
		// Nénuphar / Bulles
		vector.StrokeCircle(img, centerX, centerY+2, 8, 1.5, iconColor, true)
		vector.StrokeLine(img, centerX-4, centerY+2, centerX+4, centerY+2, 1, iconColor, true)
		vector.DrawFilledCircle(img, centerX-4, centerY-4, 2, iconColor, true)
		vector.DrawFilledCircle(img, centerX+3, centerY-6, 1.5, iconColor, true)
	default:
		// Losange pour Default
		vector.StrokeLine(img, centerX, centerY-8, centerX+8, centerY, 2, iconColor, true)
		vector.StrokeLine(img, centerX+8, centerY, centerX, centerY+8, 2, iconColor, true)
		vector.StrokeLine(img, centerX, centerY+8, centerX-8, centerY, 2, iconColor, true)
		vector.StrokeLine(img, centerX-8, centerY, centerX, centerY-8, 2, iconColor, true)
	}

	return img
}
