// Package hud affiche les informations de l'interface
package hud

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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

// Render dessine le HUD dans la barre latérale
func (h *HUD) Render(screen *ebiten.Image, x int) {
	y := 30
	
	// --- SECTION: TITRE & TOUR ---
	title := fmt.Sprintf("TOUR %d", h.world.Turn)
	text.Draw(screen, "HUNTER-GATHERERS", basicfont.Face7x13, x, y, color.RGBA{255, 200, 100, 255})
	y += 16
	text.Draw(screen, title, basicfont.Face7x13, x, y, color.White)
	y += 30

	// --- SECTION: STATISTIQUES ---
	text.Draw(screen, "-- STATS --", basicfont.Face7x13, x, y, color.RGBA{100, 200, 255, 255})
	y += 18

	p := h.world.Player
	h.drawStat(screen, x, y, "HP", p.Stats.Health, p.Stats.MaxHealth)
	y += 14
	h.drawStat(screen, x, y, "MN", p.Stats.Mana, p.Stats.MaxMana)
	y += 14
	h.drawStat(screen, x, y, "SN", p.Stats.Sanity, p.Stats.MaxSanity)
	y += 30

	// --- SECTION: MONDE & DIFFICULTÉ ---
	text.Draw(screen, "-- WORLD --", basicfont.Face7x13, x, y, color.RGBA{100, 255, 100, 255})
	y += 18
	text.Draw(screen, fmt.Sprintf("Diff: %s", h.world.Difficulty.Level), basicfont.Face7x13, x, y, color.White)
	y += 14
	text.Draw(screen, fmt.Sprintf("Active: %s", h.world.CurrentGridID), basicfont.Face7x13, x, y, color.RGBA{255, 255, 0, 255})
	y += 30

	// --- SECTION: ENTITÉS ---
	h.RenderEntityList(screen, x, y)
	
	// --- SECTION: CONTRÔLES (en bas) ---
	y = 480 // Remonté un peu pour laisser de la place
	text.Draw(screen, "-- CONTROLS --", basicfont.Face7x13, x, y, color.RGBA{150, 150, 150, 255})
	y += 18
	controls := []string{
		"Click : Reveler",
		"M : Matcher",
		"SPACE : Fin de tour",
		"F1-F4 : Difficulte",
		"F5/F6 : Reveal/Hide All",
		"S : Spawn paires",
		"ESC : Abandonner",
	}
	for _, c := range controls {
		text.Draw(screen, c, basicfont.Face7x13, x, y, color.RGBA{180, 180, 180, 255})
		y += 14
	}
}

func (h *HUD) drawStat(screen *ebiten.Image, x, y int, label string, val, max int) {
	var c color.Color = color.White
	if val < 20 {
		c = color.RGBA{255, 100, 100, 255}
	}
	text.Draw(screen, fmt.Sprintf("%s: %d/%d", label, val, max), basicfont.Face7x13, x, y, c)
}

// RenderEntityList affiche le décompte des entités par grille
func (h *HUD) RenderEntityList(screen *ebiten.Image, x, y int) {
	text.Draw(screen, "-- ENTITIES --", basicfont.Face7x13, x, y, color.RGBA{100, 255, 100, 255})
	y += 18

	for _, gridID := range h.world.GridOrder {
		resCount := 0
		creCount := 0
		
		for _, e := range h.world.Entities.GetByType(entity.TypeResource) {
			if e.GetGridID() == gridID {
				resCount++
			}
		}
		for _, e := range h.world.Entities.GetByType(entity.TypeCreature) {
			if e.GetGridID() == gridID {
				creCount++
			}
		}

		info := fmt.Sprintf("[%s] R:%d C:%d", gridID, resCount, creCount)
		text.Draw(screen, info, basicfont.Face7x13, x, y, color.White)
		y += 14
	}
}
