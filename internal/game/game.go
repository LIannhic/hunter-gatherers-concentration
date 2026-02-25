// Package game implémente la boucle de jeu Ebiten
package game

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/app"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game implémente l'interface ebiten.Game
type Game struct {
	app *app.Application
}

// NewGame crée une nouvelle instance du jeu
func NewGame() (*Game, error) {
	application, err := app.NewApplication()
	if err != nil {
		return nil, err
	}
	
	return &Game{
		app: application,
	}, nil
}

// Update met à jour la logique du jeu
// Appelé 60 fois par seconde par défaut
func (g *Game) Update() error {
	return g.app.Update()
}

// Draw dessine le jeu
// Appelé autant de fois que possible (dépend du refresh rate)
func (g *Game) Draw(screen *ebiten.Image) {
	g.app.Draw(screen)
}

// Layout définit la taille de l'écran logique
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.app.Layout(outsideWidth, outsideHeight)
}
