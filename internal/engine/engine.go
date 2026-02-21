package engine

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Prochainement : screen.DrawImage(...)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 640, 480
}
