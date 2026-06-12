package system

import (
	"errors"
	"math/rand"
	"sort"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

// FindAvailable3x3DeploymentArea cherche une zone 3x3 libre sur un grid
func (w *World) FindAvailable3x3DeploymentArea(gridID string) (board.Position, bool) {
	grid, ok := w.GetGrid(gridID)
	if !ok || grid.Width < 3 || grid.Height < 3 {
		return board.Position{}, false
	}

	for y := 0; y <= grid.Height-3; y++ {
		for x := 0; x <= grid.Width-3; x++ {
			okArea := true
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					plot, err := grid.Get(board.Position{X: x + dx, Y: y + dy})
					if err != nil {
						okArea = false
						break
					}
					if plot.Modifier.Obstructed {
						okArea = false
						break
					}
					for _, entityID := range plot.EntitiesID {
						if e, ok := w.Entities.Get(entity.ID(entityID)); ok {
							if e.GetType() == entity.TypeStructure {
								okArea = false
								break
							}
						}
					}
					if !okArea {
						break
					}
				}
				if !okArea {
					break
				}
			}
			if okArea {
				return board.Position{X: x + 1, Y: y + 1}, true
			}
		}
	}
	return board.Position{}, false
}

// findBest3x3DeploymentArea cherche une zone 3x3 avec un score optimal (moins d'obstructions)
func (w *World) findBest3x3DeploymentArea(gridID string) (board.Position, bool) {
	grid, ok := w.GetGrid(gridID)
	if !ok || grid.Width < 3 || grid.Height < 3 {
		return board.Position{}, false
	}

	bestScore := 1<<31 - 1
	bestPos := board.Position{}
	for y := 0; y <= grid.Height-3; y++ {
		for x := 0; x <= grid.Width-3; x++ {
			score := 0
			hasStructure := false
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					plot, err := grid.Get(board.Position{X: x + dx, Y: y + dy})
					if err != nil {
						hasStructure = true
						break
					}
					if plot.Modifier.Obstructed {
						score += 10
					}
					for _, entityID := range plot.EntitiesID {
						if e, ok := w.Entities.Get(entity.ID(entityID)); ok {
							if e.GetType() == entity.TypeStructure {
								hasStructure = true
								break
							}
						}
					}
					if hasStructure {
						break
					}
					if len(plot.EntitiesID) > 0 {
						score += 1
					}
				}
				if hasStructure {
					break
				}
			}
			if hasStructure {
				continue
			}
			if score < bestScore {
				bestScore = score
				bestPos = board.Position{X: x + 1, Y: y + 1}
			}
		}
	}
	if bestScore == 1<<31-1 {
		return board.Position{}, false
	}
	return bestPos, true
}

func (w *World) isValid3x3DeploymentCenter(grid *board.Grid, center board.Position) bool {
	return center.X >= 1 && center.Y >= 1 && center.X <= grid.Width-2 && center.Y <= grid.Height-2
}

func (w *World) is3x3DeploymentAreaClear(grid *board.Grid, center board.Position) bool {
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			plot, err := grid.Get(board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy})
			if err != nil {
				return false
			}
			if plot.Modifier.Obstructed {
				return false
			}
			for _, entityID := range plot.EntitiesID {
				if e, ok := w.Entities.Get(entity.ID(entityID)); ok {
					if e.GetType() == entity.TypeStructure {
						return false
					}
				}
			}
		}
	}
	return true
}

func (w *World) clear3x3DeploymentArea(grid *board.Grid, center board.Position) {
	// 1. On crée une liste pour collecter TOUTES les entités de la zone 3x3
	idsToRemove := make([]string, 0)

	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			pos := board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy}

			// plot est un *Plot (pointeur)
			plot, err := grid.Get(pos)
			if err != nil {
				continue
			}

			// On accumule les IDs à supprimer du monde
			for _, id := range plot.EntitiesID {
				idsToRemove = append(idsToRemove, id)
			}

			// 2. On nettoie DIRECTEMENT la tuile dans la grille (via le pointeur)
			plot.Modifier.Obstructed = false
			plot.StructureID = ""
			plot.EntitiesID = nil // 'nil' réinitialise proprement la slice
		}
	}

	// 3. On détruit définitivement les entités auprès du gestionnaire du World
	for _, id := range idsToRemove {
		w.RemoveEntity(entity.ID(id))
	}
}

// HasPortablePortal vérifie si le joueur possède un portail portable dans son inventaire
func (w *World) HasPortablePortal() bool {
	for _, item := range w.Player.Inventory.Items {
		if item.SourceID == player.PortablePortalItemSourceID {
			return true
		}
	}
	return false
}

// RemovePortablePortal retire un portail portable de l'inventaire
func (w *World) RemovePortablePortal() bool {
	for idx, item := range w.Player.Inventory.Items {
		if item.SourceID == player.PortablePortalItemSourceID {
			_ = w.RemoveLootItem(idx)
			return true
		}
	}
	return false
}

func (w *World) applyPortablePortalLootTax() {
	taxAmount := int(float64(len(w.Player.Inventory.Items)) * float64(player.PortablePortalLootTaxPercent) / 100.0)
	if taxAmount <= 0 {
		return
	}

	indices := make([]int, 0, len(w.Player.Inventory.Items))
	for idx := range w.Player.Inventory.Items {
		indices = append(indices, idx)
	}

	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	indices = indices[:taxAmount]
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))
	for _, idx := range indices {
		_ = w.RemoveLootItem(idx)
	}
}

func (w *World) applyDreamBreachPenalty() {
	w.Player.TakeDamage(5, "dream_breach")
	w.Player.ConsumeSanity(0)
	w.EventBus.PublishImmediate(event.Event{
		Type:     event.Type("dream_breach"),
		SourceID: "portable_portal",
		Payload: map[string]interface{}{
			"damage": 5,
			"sanity": 0,
			"reason": "Forced portable portal deployment",
		},
	})
}

// DeployPortablePortal déploie le portail portable à une position automatique
func (w *World) DeployPortablePortal(gridID string) (entity.Entity, error) {
	return w.DeployPortablePortalAt(gridID, board.Position{X: -1, Y: -1})
}

// DeployPortablePortalAt déploie le portail portable à une position précise
func (w *World) DeployPortablePortalAt(gridID string, center board.Position) (entity.Entity, error) {
	if !w.HasPortablePortal() {
		return nil, errors.New("aucun portail portable disponible")
	}

	grid, ok := w.GetGrid(gridID)
	if !ok {
		return nil, ErrGridNotFound
	}

	forced := false
	if center.X < 0 || center.Y < 0 {
		center, ok = w.FindAvailable3x3DeploymentArea(gridID)
		if !ok {
			forced = true
			center, ok = w.findBest3x3DeploymentArea(gridID)
			if !ok {
				return nil, errors.New("impossible de trouver une zone 3x3 pour le portail portable")
			}
		}
	} else {
		if !w.isValid3x3DeploymentCenter(grid, center) {
			return nil, errors.New("zone de déploiement invalide")
		}
		if !w.is3x3DeploymentAreaClear(grid, center) {
			forced = true
		}
	}

	// On vide systématiquement la zone 3x3 autour du portail pour libérer de l'espace (Effet séisme)
	w.clear3x3DeploymentArea(grid, center)

	portal, err := w.SpawnStructure(gridID, "portable_portal", entity.Position{X: center.X, Y: center.Y})
	if err != nil {
		return nil, err
	}

	w.RemovePortablePortal()
	w.applyPortablePortalLootTax()

	if forced {
		w.applyDreamBreachPenalty()
	}

	w.EventBus.PublishImmediate(event.Event{
		Type:     event.Type("portable_portal_deployed"),
		SourceID: string(portal.GetID()),
		Payload: map[string]interface{}{
			"grid_id":  gridID,
			"forced":   forced,
			"position": portal.GetPosition(),
		},
	})

	return portal, nil
}
