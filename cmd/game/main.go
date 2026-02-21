package main

import (
	"log"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/engine"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	game := &engine.Game{}
	ebiten.SetWindowTitle("Hunter-Gatherers-Concentration - Dev Build")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
