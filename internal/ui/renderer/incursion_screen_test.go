package renderer

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestIncursionScreenRender(t *testing.T) {
	// Crée une image off-screen 1280x720
	img := ebiten.NewImage(1280, 720)

	world := domain.NewWorld()
	world.CreateGrid("forest", 6, 6, domain.BiomeForest)
	world.SetCurrentGrid("forest")

	assetsMgr := assets.NewManager()
	boardRenderer := NewBoardRenderer(assetsMgr)
	boardRenderer.SetRenderConfig(377.5, 97.5, 87.5, "forest")
	defer boardRenderer.ResetRenderConfig()

	screen := NewIncursionScreen()
	screen.Render(img, world, boardRenderer)

	// Vérifie que l'image n'est pas vide (au moins un pixel a été dessiné)
	// On vérifie juste que ça ne panique pas
}
