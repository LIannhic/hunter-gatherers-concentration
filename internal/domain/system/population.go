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

// buildPlaytestPool construit le pool complet de toutes les entités pour le mode playtest.
func buildPlaytestPool() []gridEntity {
	var pool []gridEntity

	// Créatures globales
	for _, c := range []string{"lumifly", "shadowstalker", "fleeing_sprite", "flutterwing", "stonewarden"} {
		pool = append(pool, gridEntity{name: c, isCreature: true})
	}
	// Créatures exclusives (tous biomes)
	for _, c := range []string{"moss_monkey", "specter", "burrower", "echo_hound"} {
		pool = append(pool, gridEntity{name: c, isCreature: true})
	}
	// Ressources globales
	for _, r := range []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"} {
		pool = append(pool, gridEntity{name: r, isCreature: false})
	}
	// Ressources exclusives (tous biomes)
	for _, r := range []string{"moss_truffle", "void_bloom", "sand_core", "echo_crystal"} {
		pool = append(pool, gridEntity{name: r, isCreature: false})
	}
	// Pièges
	pool = append(pool, gridEntity{name: "trap", isTrap: true})

	return pool
}

// SpawnPairs place des paires d'entités sur les positions vides d'un grid.
// Retourne le nombre de paires effectivement spawnées.
func (w *World) SpawnPairs(gridID string, pairCount int) int {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return 0
	}

	// Collecte les positions vides
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

	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	pool := buildPlaytestPool()
	spawned := 0
	posIdx := 0

	for spawned < pairCount && posIdx+1 < len(positions) {
		if len(pool) == 0 {
			pool = buildPlaytestPool()
		}

		idx := rand.Intn(len(pool))
		e := pool[idx]
		pool = append(pool[:idx], pool[idx+1:]...)

		pos1 := positions[posIdx]
		pos2 := positions[posIdx+1]
		flipDir1 := grid.Plots[board.Position(pos1)].Tilt.ToFlipDirection()
		flipDir2 := grid.Plots[board.Position(pos2)].Tilt.ToFlipDirection()

		if e.isTrap {
			if _, err := w.SpawnTrap(gridID, pos1); err != nil {
				continue
			}
			if _, err := w.SpawnTrap(gridID, pos2); err != nil {
				continue
			}
		} else if e.isCreature {
			if _, err := w.SpawnCreature(gridID, e.name, pos1); err != nil {
				continue
			}
			if _, err := w.SpawnCreature(gridID, e.name, pos2); err != nil {
				continue
			}
		} else {
			if _, err := w.SpawnResource(gridID, e.name, pos1); err != nil {
				continue
			}
			if _, err := w.SpawnResource(gridID, e.name, pos2); err != nil {
				continue
			}
		}

		w.RevealTile(gridID, board.Position(pos1), flipDir1, "playtest_spawn")
		w.RevealTile(gridID, board.Position(pos2), flipDir2, "playtest_spawn")

		posIdx += 2
		spawned++
	}

	fmt.Printf("[PLAYTEST] SpawnPairs: %d paires (%d tuiles) sur %s\n", spawned, spawned*2, gridID)
	return spawned
}

// HasValidPair vérifie qu'il existe au moins une paire d'entités matchables sur le grid.
func (w *World) HasValidPair(gridID string) bool {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return false
	}

	// Compte les MatchID présents
	matchCounts := make(map[string]int)
	for _, plot := range grid.Plots {
		if len(plot.EntitiesID) == 0 {
			continue
		}
		topID := plot.EntitiesID[len(plot.EntitiesID)-1]
		ent, ok := w.Entities.Get(entity.ID(topID))
		if !ok {
			continue
		}
		if ent.GetType() != entity.TypeResource && ent.GetType() != entity.TypeCreature {
			continue
		}
		matchID := ent.GetMatchID()
		if matchID != "" {
			matchCounts[matchID]++
		}
	}

	for _, count := range matchCounts {
		if count >= 2 {
			return true
		}
	}
	return false
}
