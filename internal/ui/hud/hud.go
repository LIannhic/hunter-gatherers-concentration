// Package hud affiche les informations de l'interface
package hud

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

// HUD affiche les informations de jeu
type HUD struct {
	world *domain.World
}

// NewHUD crée un nouveau HUD
func NewHUD(world *domain.World) *HUD {
	return &HUD{
		world: world,
	}
}

// Render dessine le HUD
func (h *HUD) Render(screen *ebiten.Image) {
	y := 20
	
	// Titre
	title := "=== HUNTER-GATHERERS ==="
	text.Draw(screen, title, basicfont.Face7x13, 10, y, color.RGBA{255, 200, 100, 255})
	y += 25
	
	// Tour actuel
	text.Draw(screen, fmt.Sprintf("Tour: %d", h.world.Turn), basicfont.Face7x13, 10, y, color.White)
	y += 20
	
	// Entités
	resources := h.world.Entities.GetByType(domain.TypeResource)
	creatures := h.world.Entities.GetByType(domain.TypeCreature)
	
	text.Draw(screen, fmt.Sprintf("Ressources: %d", len(resources)), basicfont.Face7x13, 10, y, color.White)
	y += 15
	text.Draw(screen, fmt.Sprintf("Creatures: %d", len(creatures)), basicfont.Face7x13, 10, y, color.White)
	y += 25
	
	// Contrôles
	text.Draw(screen, "CONTROLES:", basicfont.Face7x13, 10, y, color.RGBA{150, 150, 255, 255})
	y += 15
	text.Draw(screen, "Click: Reveler/Selectionner", basicfont.Face7x13, 10, y, color.Gray{150})
	y += 12
	text.Draw(screen, "M: Matcher selection", basicfont.Face7x13, 10, y, color.Gray{150})
	y += 12
	text.Draw(screen, "S: Spawn test", basicfont.Face7x13, 10, y, color.Gray{150})
	y += 12
	text.Draw(screen, "C: Nettoyer plateau", basicfont.Face7x13, 10, y, color.Gray{150})
	y += 12
	text.Draw(screen, "SPACE: Fin tour", basicfont.Face7x13, 10, y, color.Gray{150})
	y += 12
	text.Draw(screen, "ESC: Deselectionner", basicfont.Face7x13, 10, y, color.Gray{150})
}

// RenderEntityList affiche la liste des entités à droite
func (h *HUD) RenderEntityList(screen *ebiten.Image, startX, startY int) {
	x, y := startX, startY
	
	text.Draw(screen, "=== ENTITES ===", basicfont.Face7x13, x, y, color.RGBA{100, 255, 100, 255})
	y += 20
	
	// Ressources
	resources := h.world.Entities.GetByType(domain.TypeResource)
	if len(resources) > 0 {
		text.Draw(screen, "Ressources:", basicfont.Face7x13, x, y, color.White)
		y += 15
		
		for _, e := range resources {
			if res, ok := e.(*domain.Resource); ok {
				info := fmt.Sprintf("  %s (%s)", res.ResourceType, res.Lifecycle.GetCurrentStageName())
				text.Draw(screen, info, basicfont.Face7x13, x, y, color.Gray{180})
				y += 12
			}
		}
		y += 5
	}
	
	// Creatures
	creatures := h.world.Entities.GetByType(domain.TypeCreature)
	if len(creatures) > 0 {
		text.Draw(screen, "Creatures:", basicfont.Face7x13, x, y, color.White)
		y += 15
		
		for _, e := range creatures {
			if c, ok := e.(*domain.Creature); ok {
				info := fmt.Sprintf("  %s [%s]", c.Species, c.Behavior.State)
				text.Draw(screen, info, basicfont.Face7x13, x, y, color.Gray{180})
				y += 12
			}
		}
	}
}
