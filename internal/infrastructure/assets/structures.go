package assets

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// generateTilePortal crée une tuile portail avec un motif en spirale
func generateTilePortal(size int, theme TileTheme) *ebiten.Image {
	img := ebiten.NewImage(size, size)

	// Fond sombre pour le portail
	img.Fill(color.RGBA{20, 10, 30, 255})

	centerX := float32(size) / 2
	centerY := float32(size) / 2

	// Paramètres de la spirale
	const steps = 300
	const revolutions = 6
	const maxTheta = revolutions * 2 * math.Pi
	maxRadius := float64(size) * 0.45

	// On dessine plusieurs bras de spirale pour un effet plus dense
	colors := []color.Color{
		theme.RevealedPattern,
		color.RGBA{150, 100, 255, 255}, // Violet magique
		color.RGBA{100, 200, 255, 255}, // Bleu onirique
	}

	for arm := 0; arm < len(colors); arm++ {
		offsetTheta := float64(arm) * (2 * math.Pi / float64(len(colors)))
		prevX, prevY := centerX, centerY

		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			theta := t * maxTheta
			r := t * maxRadius

			x := centerX + float32(math.Cos(theta+offsetTheta)*r)
			y := centerY + float32(math.Sin(theta+offsetTheta)*r)

			// Épaisseur variable pour un effet organique
			width := float32(1 + (1-t)*2)
			vector.StrokeLine(img, prevX, prevY, x, y, width, colors[arm], true)

			prevX, prevY = x, y
		}
	}

	// Halo central
	vector.DrawFilledCircle(img, centerX, centerY, 5, color.White, true)

	// Bordure ornementale (utilisant le thème)
	vector.StrokeRect(img, 2, 2, float32(size-4), float32(size-4), 3, theme.HiddenBorder, true)

	return img
}

// generateTileDolmen crée une tuile dolmen avec un motif de pierre mégalithique
func generateTileDolmen(size int, theme TileTheme) *ebiten.Image {
	img := generateTileRevealed(size, theme)

	centerX := float32(size) / 2
	centerY := float32(size) / 2
	patternColor := theme.RevealedPattern

	// On dessine un dolmen stylisé (2 piliers, 1 linteau)
	pillarW := float32(size) * 0.15
	pillarH := float32(size) * 0.4
	lintelW := float32(size) * 0.6
	lintelH := float32(size) * 0.15

	// Piliers
	vector.DrawFilledRect(img, centerX-pillarW*1.5, centerY-pillarH/2+lintelH/2, pillarW, pillarH, patternColor, true)
	vector.DrawFilledRect(img, centerX+pillarW*0.5, centerY-pillarH/2+lintelH/2, pillarW, pillarH, patternColor, true)

	// Linteau (sommet)
	vector.DrawFilledRect(img, centerX-lintelW/2, centerY-pillarH/2-lintelH/2, lintelW, lintelH, patternColor, true)

	return img
}

// generateTileObelisk crée une tuile obélisque avec un motif de pilier pointu
func generateTileObelisk(size int, theme TileTheme) *ebiten.Image {
	img := generateTileRevealed(size, theme)

	centerX := float32(size) / 2
	centerY := float32(size) / 2
	patternColor := theme.RevealedPattern

	// On dessine un obélisque stylisé (un rectangle fin avec un sommet en pointe)
	w := float32(size) * 0.2
	h := float32(size) * 0.5

	// Corps
	vector.DrawFilledRect(img, centerX-w/2, centerY-h/2, w, h, patternColor, true)

	// Sommet (Pyramidion)
	p3x, p3y := centerX, centerY-h/2-w/2
	vector.DrawFilledCircle(img, p3x, p3y, 2, patternColor, true) // Placeholder pointu

	// On dessine un triangle manuellement avec des lignes car vector n'a pas DrawFilledTriangle simple
	for i := float32(0); i <= w/2; i += 0.5 {
		vector.StrokeLine(img, centerX-i, centerY-h/2, centerX+i, centerY-h/2, 1, patternColor, true)
	}
	// On utilise plutôt DrawFilledCircle pour le sommet pour rester simple et propre
	vector.DrawFilledCircle(img, centerX, centerY-h/2, w/2, patternColor, true)

	return img
}
