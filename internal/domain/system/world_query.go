package system

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// GetGridForEntity retourne le grid sur lequel se trouve une entité
func (w *World) GetGridForEntity(entityID string) (*board.Grid, bool) {
	e, ok := w.Entities.Get(entity.ID(entityID))
	if !ok {
		return nil, false
	}
	return w.GetGrid(e.GetGridID())
}

// HasResourceAt vérifie s'il y a déjà une ressource à une position donnée
func (w *World) HasResourceAt(gridID string, pos board.Position) bool {
	grid, ok := w.Grids[gridID]
	if !ok {
		return false
	}
	plot, err := grid.Get(pos)
	if err != nil {
		return false
	}
	for _, id := range plot.EntitiesID {
		if e, ok := w.Entities.Get(entity.ID(id)); ok {
			if e.GetType() == entity.TypeResource {
				return true
			}
		}
	}
	return false
}

// SetPlayerPosition définit la position logique du joueur sur la grille
func (w *World) SetPlayerPosition(pos entity.Position) {
	w.playerPosition = pos
}

// GetPlayerPosition retourne la position logique du joueur
func (w *World) GetPlayerPosition() entity.Position {
	return w.playerPosition
}

// MoveSpeciesOneStepTowards est un wrapper pratique pour appeler le CreatureMovementSystem
// depuis d'autres couches (usecases, UI) sans exposer directement le movementSystem.
func (w *World) MoveSpeciesOneStepTowards(species string, target entity.Position) {
	if w.Engine != nil && w.Engine.movementSystem != nil {
		w.Engine.movementSystem.MoveSpeciesOneStepTowards(species, target, w)
	}
}
