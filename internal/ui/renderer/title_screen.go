// Package renderer - Écran titre du jeu
package renderer

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// TitleScreen gère l'affichage de l'écran titre
type TitleScreen struct {
	width      int
	height     int
	buttonRect Rect
}

// Rect représente un rectangle
type Rect struct {
	X, Y, W, H int
}

// NewTitleScreen crée un nouvel écran titre
func NewTitleScreen() *TitleScreen {
	return &TitleScreen{
		width:  800,
		height: 600,
		buttonRect: Rect{
			X: 300,
			Y: 350,
			W: 200,
			H: 50,
		},
	}
}

// Render dessine l'écran titre
func (t *TitleScreen) Render(screen *ebiten.Image) {
	// Fond noir
	screen.Fill(color.Black)

	// Titre du jeu
	title := "Hunter Gatherers Concentration"
	titleX := t.width/2 - len(title)*4 // Approximation du centrage
	text.Draw(screen, title, basicfont.Face7x13, titleX, 200, color.RGBA{200, 180, 100, 255})

	// Sous-titre
	subtitle := "Memory Game"
	subX := t.width/2 - len(subtitle)*3
	text.Draw(screen, subtitle, basicfont.Face7x13, subX, 240, color.RGBA{150, 150, 150, 255})

	// Bouton Démarrer
	t.drawButton(screen)

	// Instructions
	hint := "En jeu: Appuyez sur * pour retourner au menu"
	hintX := t.width/2 - len(hint)*3
	text.Draw(screen, hint, basicfont.Face7x13, hintX, 450, color.RGBA{100, 100, 100, 255})
}

// drawButton dessine le bouton "Démarrer"
func (t *TitleScreen) drawButton(screen *ebiten.Image) {
	// Fond du bouton
	vector.DrawFilledRect(
		screen,
		float32(t.buttonRect.X),
		float32(t.buttonRect.Y),
		float32(t.buttonRect.W),
		float32(t.buttonRect.H),
		color.RGBA{60, 100, 60, 255},
		true,
	)

	// Bordure du bouton
	vector.StrokeRect(
		screen,
		float32(t.buttonRect.X),
		float32(t.buttonRect.Y),
		float32(t.buttonRect.W),
		float32(t.buttonRect.H),
		2,
		color.RGBA{100, 180, 100, 255},
		true,
	)

	// Texte du bouton
	btnText := "DEMARRER"
	btnX := t.buttonRect.X + t.buttonRect.W/2 - len(btnText)*3
	btnY := t.buttonRect.Y + t.buttonRect.H/2 + 4
	text.Draw(screen, btnText, basicfont.Face7x13, btnX, btnY, color.White)
}

// IsStartButtonClicked vérifie si le bouton démarrer a été cliqué
func (t *TitleScreen) IsStartButtonClicked(x, y int) bool {
	return x >= t.buttonRect.X &&
		x <= t.buttonRect.X+t.buttonRect.W &&
		y >= t.buttonRect.Y &&
		y <= t.buttonRect.Y+t.buttonRect.H
}

// Layout retourne la taille de l'écran titre
func (t *TitleScreen) Layout() (int, int) {
	return t.width, t.height
}
