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

// Render dessine le HUD compact en haut à gauche
func (h *HUD) Render(screen *ebiten.Image) {
	x, y := 10, 15
	
	// Ligne 1: Titre + Tour sur la même ligne
	title := fmt.Sprintf("HUNTER-GATHERERS - Tour %d", h.world.Turn)
	text.Draw(screen, title, basicfont.Face7x13, x, y, color.RGBA{255, 200, 100, 255})
	y += 16
	
	// Ligne 2: Compteurs côte à côte
	resources := h.world.Entities.GetByType(domain.TypeResource)
	creatures := h.world.Entities.GetByType(domain.TypeCreature)
	
	counters := fmt.Sprintf("R:%d C:%d | Diff: %s", len(resources), len(creatures), h.world.Difficulty.Level)
	text.Draw(screen, counters, basicfont.Face7x13, x, y, color.White)
	y += 18
	
	// Ligne 3-6: Contrôles essentiels (compact)
	text.Draw(screen, "Click:Reveler M:Matcher", basicfont.Face7x13, x, y, color.Gray{150})
	y += 13
	text.Draw(screen, "F1-F4:Diff F5:Révéler Tout", basicfont.Face7x13, x, y, color.RGBA{100, 200, 255, 255})
	y += 13
	text.Draw(screen, "S:Spawn Shift+S:Toutes", basicfont.Face7x13, x, y, color.Gray{150})
	y += 13
	text.Draw(screen, "SPACE:Fin ESC:Reset", basicfont.Face7x13, x, y, color.Gray{150})
}

// RenderEntityList affiche la liste des entités à droite (utilisé par le menu debug)
func (h *HUD) RenderEntityList(screen *ebiten.Image, startX, startY int) {
	x, y := startX, startY
	
	text.Draw(screen, "=== ENTITES ===", basicfont.Face7x13, x, y, color.RGBA{100, 255, 100, 255})
	y += 20
	
	// Récupère les listes une seule fois pour éviter les changements pendant l'affichage
	resources := h.world.Entities.GetByType(domain.TypeResource)
	creatures := h.world.Entities.GetByType(domain.TypeCreature)
	
	// Ressources - groupe par type pour stabilité
	if len(resources) > 0 {
		text.Draw(screen, "Ressources:", basicfont.Face7x13, x, y, color.White)
		y += 15
		
		for _, e := range resources {
			if res, ok := e.(*domain.Resource); ok {
				info := fmt.Sprintf("  %s", res.ResourceType)
				text.Draw(screen, info, basicfont.Face7x13, x, y, color.Gray{180})
				y += 12
			}
		}
		y += 5
	}
	
	// Creatures - limite pour éviter débordement
	if len(creatures) > 0 {
		text.Draw(screen, "Creatures:", basicfont.Face7x13, x, y, color.White)
		y += 15
		
		maxToShow := 15
		shown := 0
		for _, e := range creatures {
			if shown >= maxToShow {
				text.Draw(screen, "  ...", basicfont.Face7x13, x, y, color.Gray{128})
				break
			}
			if c, ok := e.(*domain.Creature); ok {
				info := fmt.Sprintf("  %s", c.Species)
				text.Draw(screen, info, basicfont.Face7x13, x, y, color.Gray{180})
				y += 12
				shown++
			}
		}
	}
}
