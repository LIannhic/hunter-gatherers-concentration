package board

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
)

type LayoutGenerator struct {
}

func NewLayoutGenerator() *LayoutGenerator {
	return &LayoutGenerator{}
}

// GenerateDreamPlane génère un nouveau plan de rêve avec sa structure de zones
func (g *LayoutGenerator) GenerateDreamPlane(id string, level meta.DifficultyLevel, worldsCleared int) *DreamPlane {
	plane := NewDreamPlane(id)

	// 1. Déterminer la longueur du chemin principal
	var pathLength int
	switch level {
	case meta.LevelEasy:
		pathLength = 4 + worldsCleared*1
	case meta.LevelNormal:
		pathLength = 5 + worldsCleared*2
	default: // Hard, Insane
		pathLength = 6 + worldsCleared*2
	}

	// 2. Déterminer la taille des grilles (zones)
	minSize, maxSize := 2, 6
	switch level {
	case meta.LevelEasy:
		minSize, maxSize = 3, 4
	case meta.LevelNormal:
		minSize, maxSize = 4, 5
	case meta.LevelHard, meta.LevelInsane:
		minSize, maxSize = 5, 6
	}

	getRandomSize := func() int {
		return rand.Intn(maxSize-minSize+1) + minSize
	}

	// 3. Créer les zones du chemin principal
	var pathZoneIDs []string

	// Zone de départ (Toujours 6x6 pour la config Portail)
	startGrid := NewGrid("zone_start", 6, 6, BiomeDefault)
	plane.AddZone(startGrid)
	plane.StartZoneID = startGrid.ID
	plane.Coords[startGrid.ID] = Position{4, 4} // Centre du 9x9 (0..8)
	pathZoneIDs = append(pathZoneIDs, startGrid.ID)

	// Zones intermédiaires
	biomes := []BiomeType{BiomeForest, BiomeCave, BiomeDesert, BiomeSwamp}
	for i := 0; i < pathLength; i++ {
		biome := biomes[rand.Intn(len(biomes))]
		gridID := fmt.Sprintf("zone_%d", i+1)
		size := getRandomSize()
		grid := NewGrid(gridID, size, size, biome)

		// Trouve une position libre adjacente à la dernière zone
		prevID := pathZoneIDs[len(pathZoneIDs)-1]
		dir, coords, ok := g.findAvailableDirectionAndCoords(plane, prevID)
		if !ok {
			// Backtrack ou abandon si bloqué (simplifié ici: on arrête le chemin)
			break
		}

		plane.AddZone(grid)
		plane.Coords[gridID] = coords
		plane.Connect(prevID, gridID, dir)
		pathZoneIDs = append(pathZoneIDs, grid.ID)
	}

	// Zone de fin (Toujours 6x6 pour la config Portail)
	lastID := pathZoneIDs[len(pathZoneIDs)-1]
	dir, coords, ok := g.findAvailableDirectionAndCoords(plane, lastID)
	if ok {
		endGrid := NewGrid("zone_end", 6, 6, BiomeDefault)
		plane.AddZone(endGrid)
		plane.EndZoneID = endGrid.ID
		plane.Coords[endGrid.ID] = coords
		plane.Connect(lastID, endGrid.ID, dir)
		pathZoneIDs = append(pathZoneIDs, endGrid.ID)
	} else {
		// Si bloqué, on prend la dernière zone comme fin
		plane.EndZoneID = lastID
	}

	// 5. Ajouter 1-2 impasses (deadends)
	numDeadEnds := rand.Intn(2) + 1
	for i := 0; i < numDeadEnds; i++ {
		parentIdx := rand.Intn(len(pathZoneIDs))
		parentID := pathZoneIDs[parentIdx]

		dir, coords, ok := g.findAvailableDirectionAndCoords(plane, parentID)
		if !ok {
			continue
		}

		deadID := fmt.Sprintf("deadend_%d", i+1)
		biome := biomes[rand.Intn(len(biomes))]
		size := getRandomSize()
		deadGrid := NewGrid(deadID, size, size, biome)
		plane.AddZone(deadGrid)
		plane.Coords[deadID] = coords
		plane.Connect(parentID, deadID, dir)
	}

	// 6. Appliquer les configurations spéciales (Portails)
	g.SetupPortalArea(plane.Zones[plane.StartZoneID], true)
	if _, ok := plane.Zones[plane.EndZoneID]; ok {
		g.SetupPortalArea(plane.Zones[plane.EndZoneID], false)
	}

	return plane
}

// GeneratePlaytestPlane génère un monde fixe et dense pour le test
func (g *LayoutGenerator) GeneratePlaytestPlane(id string) *DreamPlane {
	plane := NewDreamPlane(id)

	// Une seule zone 6x6 dense
	startGrid := NewGrid("zone_playtest", 6, 6, BiomeForest)
	plane.AddZone(startGrid)
	plane.StartZoneID = startGrid.ID
	plane.EndZoneID = startGrid.ID // Départ = Fin pour le test simple
	plane.Coords[startGrid.ID] = Position{4, 4}

	// Setup structures
	g.SetupPortalArea(startGrid, true)

	return plane
}

func (g *LayoutGenerator) findAvailableDirectionAndCoords(plane *DreamPlane, zoneID string) (Direction, Position, bool) {
	currentPos := plane.Coords[zoneID]
	allDirs := []Direction{North, South, East, West}
	rand.Shuffle(len(allDirs), func(i, j int) {
		allDirs[i], allDirs[j] = allDirs[j], allDirs[i]
	})

	for _, d := range allDirs {
		// Vérifie si la direction est déjà prise sur cette zone
		if _, exists := plane.GetConnectedZone(zoneID, d); exists {
			continue
		}

		// Calcule les nouvelles coordonnées
		vec := DirectionVector(d)
		newPos := Position{X: currentPos.X + vec.X, Y: currentPos.Y + vec.Y}

		// Vérifie les limites 9x9 (0..8)
		if newPos.X < 0 || newPos.X > 8 || newPos.Y < 0 || newPos.Y > 8 {
			continue
		}

		// Vérifie si les coordonnées sont déjà occupées par une autre zone
		occupied := false
		for _, pos := range plane.Coords {
			if pos == newPos {
				occupied = true
				break
			}
		}

		if !occupied {
			return d, newPos, true
		}
	}

	return -1, Position{}, false
}

// SetupPortalArea configure les dolmens/obélisques autour du portail
func (g *LayoutGenerator) SetupPortalArea(grid *Grid, isStart bool) {
	// Structure en 2x2 (4 coins) dans un grid 6x6
	// Coordonnées B2, B5, E2, E5 -> (1,1), (1,4), (4,1), (4,4)
	corners := []Position{
		{1, 1}, {1, 4}, {4, 1}, {4, 4},
	}

	structureType := "obelisk"
	if isStart {
		structureType = "dolmen"
	}

	for _, pos := range corners {
		if plot, err := grid.Get(pos); err == nil {
			// On marque la parcelle avec l'ID de structure
			// Note: On utilisera World pour spawner les entités réelles plus tard
			// ou on peut mettre un tag/métadonnée ici.
			plot.StructureID = fmt.Sprintf("struct_%s_%d_%d", structureType, pos.X, pos.Y)
			plot.Modifier.Obstructed = true // Les structures sont infranchissables
		}
	}

	// Portail au centre (couvrant 2x2) : positions (2,2), (2,3), (3,2), (3,3)
	portalPositions := []Position{
		{2, 2}, {2, 3}, {3, 2}, {3, 3},
	}
	for _, portalPos := range portalPositions {
		if plot, err := grid.Get(portalPos); err == nil {
			if isStart {
				plot.StructureID = "commencement_portal"
			} else {
				plot.StructureID = "finish_portal"
			}
		}
	}
	// Ajouter des parcelles vides dans les autres positions de la zone
	otherPositions := []Position{
		{0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5},
		{1, 0}, {1, 2}, {1, 3}, {1, 5},
		{2, 0}, {2, 1}, {2, 4}, {2, 5},
		{3, 0}, {3, 1}, {3, 4}, {3, 5},
		{4, 0}, {4, 2}, {4, 3}, {4, 5},
		{5, 0}, {5, 1}, {5, 2}, {5, 3}, {5, 4}, {5, 5},
	}
	for _, pos := range otherPositions {
		if plot, err := grid.Get(pos); err != nil {
			plot.Empty = true
		}
	}
}
