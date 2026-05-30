package domain

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// FillGridRandomly remplit un grid avec des paires d'entités et des pièges
// Cette méthode a été déplacée du package app vers le domaine pour une architecture plus propre.
func (w *World) FillGridRandomly(gridID string) {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return
	}

	fmt.Printf("[DOMAIN-POP] Filling grid %s randomly...\n", gridID)

	// 1. Liste toutes les positions libres
	var positions []entity.Position
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			plot, _ := grid.Get(pos)
			if len(plot.EntitiesID) == 0 && !plot.Modifier.Obstructed {
				positions = append(positions, entity.Position{X: x, Y: y})
			}
		}
	}

	// 2. Mélange les positions
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// 3. Types disponibles
	resourceTypes := []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"}
	// Ajout de "stonewarden" et "flutterwing" à la liste des créatures spawnables
	creatureTypes := []string{"lumifly", "shadowstalker", "burrower", "specter", "echo_hound", "fleeing_sprite", "moss_monkey", "stonewarden", "flutterwing"}

	posIdx := 0
	totalTiles := len(positions)

	// On remplit par paires tant qu'on a de la place
	for posIdx < totalTiles-1 {
		// Choisit aléatoirement entre Ressource, Créature ou Piège
		choice := rand.Float32()

		if choice < 0.4 {
			// Paire de Ressources (40% de chance)
			resType := resourceTypes[rand.Intn(len(resourceTypes))]
			fmt.Printf("  - [%s] Spawning resource pair: %s at %v and %v\n", gridID, resType, positions[posIdx], positions[posIdx+1])
			w.SpawnResource(gridID, resType, positions[posIdx])
			w.SpawnResource(gridID, resType, positions[posIdx+1])
			posIdx += 2
		} else if choice < 0.8 {
			// Paire de Créatures (40% de chance)
			creType := creatureTypes[rand.Intn(len(creatureTypes))]
			fmt.Printf("  - [%s] Spawning creature pair: %s at %v and %v\n", gridID, creType, positions[posIdx], positions[posIdx+1])
			w.SpawnCreature(gridID, creType, positions[posIdx])
			w.SpawnCreature(gridID, creType, positions[posIdx+1])
			posIdx += 2
		} else {
			// Paire de Pièges (20% de chance)
			fmt.Printf("  - [%s] Spawning trap pair at %v and %v\n", gridID, positions[posIdx], positions[posIdx+1])
			w.SpawnTrap(gridID, positions[posIdx])
			w.SpawnTrap(gridID, positions[posIdx+1])
			posIdx += 2
		}
	}

	// Si le nombre de cases était impair, on met un dernier piège
	if posIdx < totalTiles {
		fmt.Printf("  - [%s] Spawning lone trap at %v\n", gridID, positions[posIdx])
		w.SpawnTrap(gridID, positions[posIdx])
	}

	fmt.Printf("[DEBUG-POP] Grid %s population terminee. Cibles (Matchable): %d | Pieges: %d\n",
		gridID, grid.InitialMatchableCount, totalTiles-grid.InitialMatchableCount)
}
