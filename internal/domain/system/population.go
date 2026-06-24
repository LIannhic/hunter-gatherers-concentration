package system

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

type gridEntity struct {
	name       string
	isCreature bool
	isTrap     bool
	exclusive  bool
	drawCount  int
}

// FillGridRandomly remplit un grid avec des paires d'entités.
// Pool commun créatures + ressources + pièges, tirage aléatoire, retrait après 2 paires (4 tuiles).
// Priorité aux exclusives du biome.
func (w *World) FillGridRandomly(gridID string) {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return
	}

	fmt.Printf("[DOMAIN-POP] Filling grid %s (Biome: %s)...\n", gridID, grid.Biome)

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

	totalSlots := len(positions)
	if totalSlots == 0 {
		return
	}

	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// 2. Construire le pool commun créatures + ressources + pièges
	var pool []gridEntity

	// Créatures
	globalCreatures := []string{"lumifly", "shadowstalker", "fleeing_sprite", "flutterwing", "stonewarden"}
	for _, c := range globalCreatures {
		pool = append(pool, gridEntity{name: c, isCreature: true})
	}
	var exclusiveCreature string
	switch grid.Biome {
	case board.BiomeForest:
		exclusiveCreature = "moss_monkey"
	case board.BiomeCave:
		exclusiveCreature = "specter"
	case board.BiomeDesert:
		exclusiveCreature = "burrower"
	case board.BiomeSwamp:
		exclusiveCreature = "echo_hound"
	}
	if exclusiveCreature != "" {
		pool = append(pool, gridEntity{name: exclusiveCreature, isCreature: true, exclusive: true})
	}

	// Ressources
	globalResources := []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"}
	for _, r := range globalResources {
		pool = append(pool, gridEntity{name: r, isCreature: false})
	}
	var exclusiveResource string
	switch grid.Biome {
	case board.BiomeForest:
		exclusiveResource = "moss_truffle"
	case board.BiomeCave:
		exclusiveResource = "void_bloom"
	case board.BiomeDesert:
		exclusiveResource = "sand_core"
	case board.BiomeSwamp:
		exclusiveResource = "echo_crystal"
	}
	if exclusiveResource != "" {
		pool = append(pool, gridEntity{name: exclusiveResource, isCreature: false, exclusive: true})
	}

	// Pièges (dans le pool global, tirés aléatoirement)
	pool = append(pool, gridEntity{name: "trap", isTrap: true})

	// 3. Séparer exclusives et non-exclusives, mélanger, puis concaténer
	var exclusives, commons []gridEntity
	for _, e := range pool {
		if e.exclusive {
			exclusives = append(exclusives, e)
		} else {
			commons = append(commons, e)
		}
	}
	rand.Shuffle(len(exclusives), func(i, j int) { exclusives[i], exclusives[j] = exclusives[j], exclusives[i] })
	rand.Shuffle(len(commons), func(i, j int) { commons[i], commons[j] = commons[j], commons[i] })
	pool = append(exclusives, commons...)

	// 4. Tirage aléatoire du pool, spawn 2 tuiles à chaque tirage, retrait après 2 paires
	remainingSlots := totalSlots
	posIdx := 0
	for remainingSlots >= 2 && len(pool) > 0 {
		idx := rand.Intn(len(pool))
		e := pool[idx]

		if e.isTrap {
			w.SpawnTrap(gridID, positions[posIdx])
			w.SpawnTrap(gridID, positions[posIdx+1])
		} else if e.isCreature {
			w.SpawnCreature(gridID, e.name, positions[posIdx])
			w.SpawnCreature(gridID, e.name, positions[posIdx+1])
		} else {
			w.SpawnResource(gridID, e.name, positions[posIdx])
			w.SpawnResource(gridID, e.name, positions[posIdx+1])
		}
		posIdx += 2
		remainingSlots -= 2

		e.drawCount++
		if e.drawCount >= 2 {
			pool = append(pool[:idx], pool[idx+1:]...)
		} else {
			pool[idx] = e
		}
	}

	// 5. Tuile orpheline si nombre impair → toujours un piège
	if posIdx < totalSlots {
		w.SpawnTrap(gridID, positions[posIdx])
	}

	fmt.Printf("[DEBUG-POP] Grid %s population terminee. Cibles (Matchable): %d\n",
		gridID, grid.InitialMatchableCount)
}
