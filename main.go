package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game implémente l'interface ebiten.Game
type Game struct{}

// Update contient la logique (entrées clavier/manette, calculs)
func (g *Game) Update() error {
	return nil
}

// Draw contient le rendu (affichage des tuiles, texte)
func (g *Game) Draw(screen *ebiten.Image) {
}

// Layout définit la taille de la fenêtre
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480 // longueur, hauteur
}

func main() {
	game := &Game{}
	ebiten.SetWindowTitle("Hunter-Gatherers-concentration")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
