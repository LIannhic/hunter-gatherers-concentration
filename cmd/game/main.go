package main

import (
	"log"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Crée le jeu
	g, err := game.NewGame()
	if err != nil {
		log.Fatal(err)
	}

	// Configure la fenêtre
	ebiten.SetWindowTitle("Hunter-Gatherers Concentration - Dev Build")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	
	// Lance la boucle de jeu
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
