// Package hud affiche les informations de l'interface
package hud

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// HUD affiche les informations de jeu
type HUD struct {
	world       *domain.World
	showDetails bool
}

// NewHUD crée un nouveau HUD
func NewHUD(world *domain.World) *HUD {
	return &HUD{
		world:       world,
		showDetails: false,
	}
}

// ToggleDetails bascule l'affichage de la fenêtre de détails
func (h *HUD) ToggleDetails() {
	h.showDetails = !h.showDetails
}

// Render dessine le HUD complet
func (h *HUD) Render(screen *ebiten.Image) {
	h.renderPortrait(screen)
	h.renderInventory(screen)
	h.renderGauges(screen)
	h.renderMiniMap(screen)

	if h.showDetails {
		h.renderDetailWindow(screen)
	}
}

// getGridDetailedCounts retourne le nombre d'entités par type/espèce pour une grille donnée
func (h *HUD) getGridDetailedCounts(gridID string) map[string]int {
	counts := make(map[string]int)

	// Ressources
	for _, e := range h.world.Entities.GetByType(entity.TypeResource) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			if r, ok := e.(*domain.Resource); ok {
				counts[r.ResourceType]++
			}
		}
	}
	// Créatures
	for _, e := range h.world.Entities.GetByType(entity.TypeCreature) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			if c, ok := e.(*domain.Creature); ok {
				counts[c.Species]++
			}
		}
	}
	// Structures & Portails
	for _, e := range h.world.Entities.GetByType(entity.TypeStructure) {
		if e.GetGridID() == gridID {
			label := "structure"
			if e.HasTag("commencement_portal") {
				label = "portail_entree"
			} else if e.HasTag("finish_portal") {
				label = "portail_sortie"
			} else if e.HasTag("dolmen") {
				label = "dolmen"
			} else if e.HasTag("obelisk") {
				label = "obelisque"
			}
			counts[label]++
		}
	}
	return counts
}

// getGridEntityCounts retourne le nombre total de ressources, créatures et structures sur un grid
func (h *HUD) getGridEntityCounts(gridID string) (resCount, creCount, structCount int) {
	for _, e := range h.world.Entities.GetByType(entity.TypeResource) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			resCount++
		}
	}
	for _, e := range h.world.Entities.GetByType(entity.TypeCreature) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			creCount++
		}
	}
	for _, e := range h.world.Entities.GetByType(entity.TypeStructure) {
		if e.GetGridID() == gridID {
			structCount++
		}
	}
	return
}

func (h *HUD) renderPortrait(screen *ebiten.Image) {
	// Portrait Holder
	vector.StrokeRect(screen, ui.PortraitX, ui.PortraitY, ui.PortraitW, ui.PortraitH, 1, color.RGBA{100, 100, 100, 255}, true)

	// Turn and Difficulty (aligned)
	infoX := int(ui.PortraitX) + 60
	infoY := int(ui.PortraitY) + 25
	text.Draw(screen, fmt.Sprintf("T:%d", h.world.Turn), basicfont.Face7x13, infoX, infoY, color.White)
	text.Draw(screen, fmt.Sprintf("D:%s", h.world.Difficulty.Level), basicfont.Face7x13, infoX+60, infoY, color.RGBA{255, 200, 100, 255})

	// --- COLUMN LEFT: CONTROLS ---
	y := int(ui.PortraitY) + 85
	text.Draw(screen, "ACTION:", basicfont.Face7x13, int(ui.PortraitX)+10, y-15, color.RGBA{100, 200, 255, 255})

	controls := []string{
		"CLIC: Ouvrir",
		"M: Matcher",
		"I: Zones",
		"ESPACE: Fin",
		"F1-F4: Diff",
		"F5/F6: R/H All",
		"ESC: Menu",
	}
	for _, c := range controls {
		text.Draw(screen, c, basicfont.Face7x13, int(ui.PortraitX)+10, y, color.RGBA{160, 160, 160, 255})
		y += 18
	}

	// --- COLUMN RIGHT: CURRENT GRID DETAILS ---
	rx := int(ui.PortraitX) + 125
	ry := int(ui.PortraitY) + 85
	text.Draw(screen, "SUR ZONE:", basicfont.Face7x13, rx, ry-15, color.RGBA{100, 200, 255, 255})

	counts := h.getGridDetailedCounts(h.world.CurrentGridID)

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		label := strings.Title(strings.ReplaceAll(k, "_", " "))
		if len(label) > 12 {
			label = label[:12]
		}
		info := fmt.Sprintf("%s:%d", label, counts[k])
		text.Draw(screen, info, basicfont.Face7x13, rx, ry, color.White)
		ry += 16
		if ry > int(ui.PortraitY)+int(ui.PortraitH)-20 {
			break
		}
	}

	if len(keys) == 0 {
		text.Draw(screen, "(Zone vide)", basicfont.Face7x13, rx, ry, color.RGBA{150, 150, 150, 255})
	}

	// Menu Icon
	mx := ui.PortraitX + ui.MenuIconRelativeX
	my := ui.PortraitY + ui.MenuIconRelativeY
	vector.DrawFilledRect(screen, float32(mx), float32(my), float32(ui.MenuIconSize), float32(ui.MenuIconSize), color.RGBA{150, 150, 150, 255}, true)
	text.Draw(screen, "M", basicfont.Face7x13, int(mx)+15, int(my)+25, color.Black)
}

func (h *HUD) renderInventory(screen *ebiten.Image) {
	// Inventory Panel
	vector.StrokeRect(screen, ui.InventoryX, ui.InventoryY, ui.InventoryW, ui.InventoryH, 1, color.RGBA{100, 100, 100, 255}, true)
	text.Draw(screen, "INVENTORY", basicfont.Face7x13, ui.InventoryX+10, ui.InventoryY+20, color.RGBA{100, 200, 255, 255})

	// Loot Slots (Grid 3x4)
	for i := 0; i < 12; i++ {
		row := i / ui.LootSlotsPerRow
		col := i % ui.LootSlotsPerRow
		sx := ui.InventoryX + float64(col)*(ui.LootSlotSize+ui.LootSlotPadding) + 5
		sy := ui.InventoryY + 40 + float64(row)*(ui.LootSlotSize+ui.LootSlotPadding)
		vector.StrokeRect(screen, float32(sx), float32(sy), float32(ui.LootSlotSize), float32(ui.LootSlotSize), 1, color.RGBA{50, 50, 50, 255}, true)
	}

	// Loot counter
	lcx := ui.InventoryX + ui.LootCounterRelativeX
	lcy := ui.InventoryY + ui.LootCounterRelativeY
	vector.DrawFilledRect(screen, float32(lcx), float32(lcy), float32(ui.LootCounterSize), float32(ui.LootCounterSize), color.RGBA{50, 50, 50, 255}, true)
	text.Draw(screen, "0", basicfont.Face7x13, int(lcx)+15, int(lcy)+25, color.White)

	// Delete Loot icon
	dlx := ui.InventoryX + ui.DeleteLootRelativeX
	dly := ui.InventoryY + ui.DeleteLootRelativeY
	vector.DrawFilledRect(screen, float32(dlx), float32(dly), float32(ui.DeleteLootSize), float32(ui.DeleteLootSize), color.RGBA{150, 50, 50, 255}, true)
	text.Draw(screen, "X", basicfont.Face7x13, int(dlx)+15, int(dly)+25, color.White)
}

func (h *HUD) renderGauges(screen *ebiten.Image) {
	// Gauges Holder
	vector.StrokeRect(screen, ui.GaugesX, ui.GaugesY, ui.GaugesW, ui.GaugesH, 1, color.RGBA{100, 100, 100, 255}, true)

	p := h.world.Player
	// Health gauge
	h.drawVerticalGauge(screen, ui.GaugesX+ui.HealthGaugeRelativeX, ui.GaugesY+ui.HealthGaugeRelativeY, "HP", p.Stats.Health, p.Stats.MaxHealth, color.RGBA{255, 50, 50, 255})
	// Mana gauge
	h.drawVerticalGauge(screen, ui.GaugesX+ui.ManaGaugeRelativeX, ui.GaugesY+ui.ManaGaugeRelativeY, "MN", p.Stats.Mana, p.Stats.MaxMana, color.RGBA{50, 50, 255, 255})
	// Sanity gauge
	h.drawVerticalGauge(screen, ui.GaugesX+ui.SanityGaugeRelativeX, ui.GaugesY+ui.SanityGaugeRelativeY, "SN", p.Stats.Sanity, p.Stats.MaxSanity, color.RGBA{50, 255, 50, 255})
}

func (h *HUD) drawVerticalGauge(screen *ebiten.Image, x, y float64, label string, val, max int, clr color.Color) {
	// Gauge Holder background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(ui.GaugeW), float32(ui.GaugeH), color.RGBA{30, 30, 30, 255}, true)

	// Rule: 100 hp = 200 px recalculate if over 200 hp for always = 400 px.
	var totalPx float32
	if max <= 200 {
		totalPx = float32(max) * 2
	} else {
		totalPx = 400
	}

	var fillHeight float32
	if max > 0 {
		fillHeight = (float32(val) / float32(max)) * totalPx
	}

	// Draw background of the actual gauge
	vector.DrawFilledRect(screen, float32(x), float32(y+ui.GaugeH-float64(totalPx)), float32(ui.GaugeW), totalPx, color.RGBA{50, 50, 50, 255}, true)

	// Fill from bottom
	vector.DrawFilledRect(screen, float32(x), float32(y+ui.GaugeH-float64(fillHeight)), float32(ui.GaugeW), fillHeight, clr, true)

	// Label
	text.Draw(screen, label, basicfont.Face7x13, int(x)+25, int(y)+int(ui.GaugeH)+15, color.White)
}

func (h *HUD) renderMiniMap(screen *ebiten.Image) {
	// Minimap Holder
	vector.StrokeRect(screen, ui.MinimapX, ui.MinimapY, ui.MinimapW, ui.MinimapH, 1, color.RGBA{100, 100, 100, 255}, true)
	text.Draw(screen, "MINIMAP", basicfont.Face7x13, ui.MinimapX+10, ui.MinimapY+20, color.RGBA{100, 200, 255, 255})

	if h.world.DreamPlane == nil {
		return
	}

	// Drawing the map as we go along
	const padding = 10
	mapW := float32(ui.MinimapW - 2*padding)
	cellSize := mapW / 9 // Assuming 9x9 grid for minimap

	plane := h.world.DreamPlane

	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			sx := float32(ui.MinimapX+padding) + float32(col)*cellSize
			sy := float32(ui.MinimapY+padding+20) + float32(row)*cellSize

			var zoneID string
			for id, coords := range plane.Coords {
				if coords.X == col && coords.Y == row {
					zoneID = id
					break
				}
			}

			if zoneID != "" {
				var rectColor color.Color = color.RGBA{150, 150, 150, 255}
				if zoneID == h.world.CurrentGridID {
					rectColor = color.RGBA{255, 255, 0, 255}
				}
				vector.DrawFilledRect(screen, sx, sy, cellSize-1, cellSize-1, rectColor, true)
			}
		}
	}
}

func (h *HUD) renderDetailWindow(screen *ebiten.Image) {
	// Position et taille de la fenêtre
	winW, winH := 320, 450
	winX := (ui.ScreenWidth - winW) / 2
	winY := (ui.ScreenHeight - winH) / 2

	// Fond translucide
	vector.DrawFilledRect(screen, float32(winX), float32(winY), float32(winW), float32(winH), color.RGBA{10, 10, 20, 230}, true)
	vector.StrokeRect(screen, float32(winX), float32(winY), float32(winW), float32(winH), 2, color.RGBA{100, 100, 150, 255}, true)

	// Titre
	text.Draw(screen, "STATISTIQUES DES ZONES", basicfont.Face7x13, winX+20, winY+30, color.RGBA{100, 200, 255, 255})

	// Icone fermer (X)
	closeX := winX + winW - 30
	closeY := winY + 10
	vector.DrawFilledRect(screen, float32(closeX), float32(closeY), 20, 20, color.RGBA{150, 50, 50, 255}, true)
	text.Draw(screen, "X", basicfont.Face7x13, closeX+6, closeY+15, color.White)

	// Liste des zones (L'ancienne liste du portrait)
	dy := winY + 70
	for _, gridID := range h.world.GridOrder {
		resCount, creCount, structCount := h.getGridEntityCounts(gridID)

		info := fmt.Sprintf("%-15s R:%d C:%d S:%d", gridID, resCount, creCount, structCount)
		var clr color.Color = color.White
		if gridID == h.world.CurrentGridID {
			clr = color.RGBA{255, 255, 0, 255}
		}
		text.Draw(screen, info, basicfont.Face7x13, winX+30, dy, clr)
		dy += 20
	}
}

// HandleClick gère les clics sur les éléments de l'HUD
func (h *HUD) HandleClick(x, y int) bool {
	if h.showDetails {
		// Vérifie le bouton fermer
		winW, winH := 320, 450
		winX := (ui.ScreenWidth - winW) / 2
		winY := (ui.ScreenHeight - winH) / 2
		closeX := winX + winW - 30
		closeY := winY + 10

		if x >= closeX && x <= closeX+20 && y >= closeY && y <= closeY+20 {
			h.showDetails = false
			return true
		}
	}
	return false
}
