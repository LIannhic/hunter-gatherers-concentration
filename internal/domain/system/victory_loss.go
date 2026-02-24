package system

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// VictorySystem vérifie les conditions de victoire et de défaite
type VictorySystem struct{}

// Priority définit l'ordre d'exécution du système
func (s *VictorySystem) Priority() int { return 10 }

// Update exécute la vérification sur le World
func (s *VictorySystem) Update(world *World) {
	// Condition de défaite : dépassement du nombre max de tours
	if world.Turn >= world.MaxTurns {
		world.EventBus.Publish(event.NewGameOverEvent(world.Turn, "Exceeded max turns"))
		return
	}

	// Condition de victoire : toutes les tuiles appariées
	allMatched := true
	hasMatchableContent := false

	for _, tile := range world.Grid.Tiles {
		if tile.EntityID != "" {
			hasMatchableContent = true
			if tile.State != board.Matched {
				allMatched = false
				break
			}
		}
	}

	if hasMatchableContent && allMatched {
		world.EventBus.Publish(event.NewVictoryEvent(world.Turn))
	}
}
