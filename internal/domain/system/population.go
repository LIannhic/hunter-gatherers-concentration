package system

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// FillGridRandomly remplit un grid avec des paires d'entités et des pièges
// en respectant un équilibre strict par grille : 40% Ressources, 40% Créatures, 20% Pièges.
func (w *World) FillGridRandomly(gridID string) {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return
	}

	fmt.Printf("[DOMAIN-POP] Filling grid %s (Biome: %s) with strict ratios...\n", gridID, grid.Biome)

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
	totalPairs := totalSlots / 2
	if totalPairs == 0 {
		return
	}

	// 2. Calcul des quotas par grille (Ratios 40/40/20 avec ajustement pour 3x3)
	resourcePairsQuota := (totalPairs * 40) / 100
	creaturePairsQuota := (totalPairs * 40) / 100

	// AJUSTEMENT POUR PETITES GRILLES (ex: 3x3 = 4 paires max) :
	// On veut au moins 3 ou 4 paires pour ne pas avoir une grille vide.
	if totalPairs <= 4 {
		resourcePairsQuota = 2
		creaturePairsQuota = 2
		if totalPairs < 4 { // Cas 3x3 standard sans obstacle = 4 paires. Si obstacle, on réduit.
			resourcePairsQuota = 1
			creaturePairsQuota = 1
		}
	}

	trapPairsQuota := totalPairs - resourcePairsQuota - creaturePairsQuota
	if trapPairsQuota < 0 {
		trapPairsQuota = 0
	}

	fmt.Printf("  - Quotas for %d pairs: Resources=%d, Creatures=%d, Traps=%d\n",
		totalPairs, resourcePairsQuota, creaturePairsQuota, trapPairsQuota)

	// 3. Configuration des pools d'entités (incluant l'exclusivité par biome)
	globalResources := []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"}

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

	resourcePool := append([]string{}, globalResources...)

	// Créatures globales (partout)
	globalCreatures := []string{"lumifly", "shadowstalker", "fleeing_sprite", "flutterwing", "stonewarden"}

	// Créature exclusive selon le biome
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

	// Pool final pour cette grille
	creaturePool := append([]string{}, globalCreatures...)
	if exclusiveCreature != "" {
		creaturePool = append(creaturePool, exclusiveCreature)
	}

	// 4. Mélange les positions pour une répartition spatiale aléatoire
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	posIdx := 0

	// 5. Spawn des Ressources
	exclusiveResSpawned := false
	for i := 0; i < resourcePairsQuota; i++ {
		var resType string
		// Limite à UNE SEULE paire de ressource exclusive par grille
		if exclusiveResource != "" && !exclusiveResSpawned && rand.Float32() < 0.5 {
			resType = exclusiveResource
			exclusiveResSpawned = true
		} else {
			resType = resourcePool[rand.Intn(len(resourcePool))]
		}
		w.SpawnResource(gridID, resType, positions[posIdx])
		w.SpawnResource(gridID, resType, positions[posIdx+1])
		posIdx += 2
	}

	// 6. Spawn des Créatures
	for i := 0; i < creaturePairsQuota; i++ {
		// On donne une plus grande chance de spawn à la créature exclusive (50% de chance si elle existe)
		var creType string
		if exclusiveCreature != "" && rand.Float32() < 0.5 {
			creType = exclusiveCreature
		} else {
			creType = creaturePool[rand.Intn(len(creaturePool))]
		}

		w.SpawnCreature(gridID, creType, positions[posIdx])
		w.SpawnCreature(gridID, creType, positions[posIdx+1])
		posIdx += 2
	}

	// 7. Spawn des Pièges
	for i := 0; i < trapPairsQuota; i++ {
		w.SpawnTrap(gridID, positions[posIdx])
		w.SpawnTrap(gridID, positions[posIdx+1])
		posIdx += 2
	}

	// 8. Gestion de la tuile orpheline si nombre impair
	if posIdx < totalSlots {
		w.SpawnTrap(gridID, positions[posIdx])
	}

	fmt.Printf("[DEBUG-POP] Grid %s population terminee. Cibles (Matchable): %d\n",
		gridID, grid.InitialMatchableCount)
}
