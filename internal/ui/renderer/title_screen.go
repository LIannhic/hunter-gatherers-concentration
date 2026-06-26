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
	width              int
	height             int
	buttonRect         Rect
	playtestButtonRect Rect
	profileButtonRect  Rect
	ButtonText         string // Texte dynamique du bouton
}

// Rect représente un rectangle
type Rect struct {
	X, Y, W, H int
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H
}

// NewTitleScreen crée un nouvel écran titre
func NewTitleScreen() *TitleScreen {
	return &TitleScreen{
		width:  1280,
		height: 720,
		buttonRect: Rect{
			X: 540,
			Y: 300,
			W: 200,
			H: 50,
		},
		profileButtonRect: Rect{
			X: 540,
			Y: 360,
			W: 200,
			H: 50,
		},
		playtestButtonRect: Rect{
			X: 540,
			Y: 420,
			W: 200,
			H: 50,
		},
		ButtonText: "DEMARRER", // Texte par défaut
	}
}

// Render dessine l'écran titre
func (t *TitleScreen) Render(screen *ebiten.Image, hasSaves bool) {
	// Fond noir
	screen.Fill(color.Black)

	// Titre du jeu
	title := "Hunter Gatherers Concentration"
	titleX := t.width/2 - len(title)*4 // Approximation du centrage
	text.Draw(screen, title, basicfont.Face7x13, titleX, 150, color.RGBA{200, 180, 100, 255})

	// Sous-titre
	subtitle := "Memory Game"
	subX := t.width/2 - len(subtitle)*3
	text.Draw(screen, subtitle, basicfont.Face7x13, subX, 190, color.RGBA{150, 150, 255, 255})

	// Bouton principal
	t.drawButton(screen)

	// Bouton Profil (si existant)
	if hasSaves {
		t.drawProfileButton(screen)
	}

	// Bouton Playtest
	t.drawPlaytestButton(screen)

	// Instructions
	hint := "En jeu: Appuyez sur echap pour retourner au menu"
	hintX := t.width/2 - len(hint)*3
	text.Draw(screen, hint, basicfont.Face7x13, hintX, 500, color.RGBA{100, 100, 100, 255})
}

// drawButton dessine le bouton principal
func (t *TitleScreen) drawButton(screen *ebiten.Image) {
	// Fond du bouton
	vector.FillRect(
		screen,
		float32(t.buttonRect.X),
		float32(t.buttonRect.Y),
		float32(t.buttonRect.W),
		float32(t.buttonRect.H),
		color.RGBA{30, 50, 30, 255},
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
	btnText := t.ButtonText
	if btnText == "" {
		btnText = "DEMARRER"
	}
	btnX := t.buttonRect.X + t.buttonRect.W/2 - len(btnText)*3
	btnY := t.buttonRect.Y + t.buttonRect.H/2 + 4
	text.Draw(screen, btnText, basicfont.Face7x13, btnX, btnY, color.White)
}

// IsStartButtonClicked vérifie si le bouton démarrer a été cliqué
func (t *TitleScreen) IsStartButtonClicked(x, y int) bool {
	return t.buttonRect.Contains(x, y)
}

// IsPlaytestButtonClicked vérifie si le bouton playtest a été cliqué
func (t *TitleScreen) IsPlaytestButtonClicked(x, y int) bool {
	return t.playtestButtonRect.Contains(x, y)
}

// IsProfileButtonClicked vérifie si le bouton profil a été cliqué
func (t *TitleScreen) IsProfileButtonClicked(x, y int) bool {
	return t.profileButtonRect.Contains(x, y)
}

// drawPlaytestButton dessine le bouton de playtest
func (t *TitleScreen) drawPlaytestButton(screen *ebiten.Image) {
	// Fond du bouton
	vector.FillRect(
		screen,
		float32(t.playtestButtonRect.X),
		float32(t.playtestButtonRect.Y),
		float32(t.playtestButtonRect.W),
		float32(t.playtestButtonRect.H),
		color.RGBA{50, 30, 30, 255}, // Teinte rougeâtre pour test
		true,
	)

	// Bordure du bouton
	vector.StrokeRect(
		screen,
		float32(t.playtestButtonRect.X),
		float32(t.playtestButtonRect.Y),
		float32(t.playtestButtonRect.W),
		float32(t.playtestButtonRect.H),
		2,
		color.RGBA{180, 100, 100, 255},
		true,
	)

	// Texte du bouton
	btnText := "PLAYTEST"
	btnX := t.playtestButtonRect.X + t.playtestButtonRect.W/2 - len(btnText)*3
	btnY := t.playtestButtonRect.Y + t.playtestButtonRect.H/2 + 4
	text.Draw(screen, btnText, basicfont.Face7x13, btnX, btnY, color.White)
}

// drawProfileButton dessine le bouton de changement de profil
func (t *TitleScreen) drawProfileButton(screen *ebiten.Image) {
	// Fond du bouton
	vector.FillRect(
		screen,
		float32(t.profileButtonRect.X),
		float32(t.profileButtonRect.Y),
		float32(t.profileButtonRect.W),
		float32(t.profileButtonRect.H),
		color.RGBA{40, 40, 60, 255}, // Teinte bleutée
		true,
	)

	// Bordure du bouton
	vector.StrokeRect(
		screen,
		float32(t.profileButtonRect.X),
		float32(t.profileButtonRect.Y),
		float32(t.profileButtonRect.W),
		float32(t.profileButtonRect.H),
		2,
		color.RGBA{100, 100, 180, 255},
		true,
	)

	// Texte du bouton
	btnText := "PROFILS"
	btnX := t.profileButtonRect.X + t.profileButtonRect.W/2 - len(btnText)*3
	btnY := t.profileButtonRect.Y + t.profileButtonRect.H/2 + 4
	text.Draw(screen, btnText, basicfont.Face7x13, btnX, btnY, color.White)
}

// Layout retourne la taille de l'écran titre
func (t *TitleScreen) Layout() (int, int) {
	return t.width, t.height
}
