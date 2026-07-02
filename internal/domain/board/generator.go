package board

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
)

// LayoutGenerator est un service de domaine responsable de la création structurelle du monde.
type LayoutGenerator struct{}

// NewLayoutGenerator crée une nouvelle instance du générateur.
func NewLayoutGenerator() *LayoutGenerator {
	return &LayoutGenerator{}
}

// GenerateDreamPlane génère un réseau de zones interconnectées selon la difficulté choisie.
func (g *LayoutGenerator) GenerateDreamPlane(id string, level meta.DifficultyLevel, worldsCleared int) *DreamPlane {
	plane := NewDreamPlane(id)

	// 1. Détermination de l'échelle de l'expédition
	pathLength := g.calculatePathLength(level, worldsCleared)
	minSize, maxSize := g.getZoneSizeRange(level)

	getRandomSize := func() int {
		return rand.Intn(maxSize-minSize+1) + minSize
	}

	var pathZoneIDs []string

	// 2. Création de la zone de départ (Toujours 6x6 pour accueillir le portail de base)
	startGrid := NewGrid("zone_start", 6, 6, BiomeDefault)
	plane.AddZone(startGrid)
	plane.StartZoneID = startGrid.ID
	plane.Coords[startGrid.ID] = Position{4, 4} // Position centrale sur la minimap 9x9
	pathZoneIDs = append(pathZoneIDs, startGrid.ID)

	// 3. Expansion du chemin principal
	biomes := []BiomeType{BiomeForest, BiomeCave, BiomeDesert, BiomeSwamp}
	for i := 0; i < pathLength; i++ {
		biome := biomes[rand.Intn(len(biomes))]
		gridID := fmt.Sprintf("zone_%d", i+1)
		size := getRandomSize()
		grid := NewGrid(gridID, size, size, biome)

		prevID := pathZoneIDs[len(pathZoneIDs)-1]
		dir, coords, ok := g.findAvailableDirectionAndCoords(plane, prevID)
		if !ok {
			break // Arrêt du chemin si l'espace est saturé
		}

		plane.AddZone(grid)
		plane.Coords[gridID] = coords
		plane.Connect(prevID, gridID, dir)
		pathZoneIDs = append(pathZoneIDs, grid.ID)
	}

	// 4. Création de la zone finale
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
		plane.EndZoneID = lastID
	}

	// 5. Ajout de chemins secondaires (impasses)
	g.addDeadEnds(plane, pathZoneIDs, biomes, getRandomSize)

	// 6. Configuration des zones de sécurité (Portails)
	g.SetupPortalArea(plane.Zones[plane.StartZoneID], true)
	if _, ok := plane.Zones[plane.EndZoneID]; ok {
		g.SetupPortalArea(plane.Zones[plane.EndZoneID], false)
	}

	return plane
}

// GeneratePlaytestPlane génère une grille 6x6 simple pour le mode playtest.
func (g *LayoutGenerator) GeneratePlaytestPlane(id string) *DreamPlane {
	plane := NewDreamPlane(id)

	startGrid := NewGrid("zone_playtest", 6, 6, BiomeDefault)
	plane.AddZone(startGrid)
	plane.Coords[startGrid.ID] = Position{4, 4}

	return plane
}

// calculatePathLength définit le nombre de zones à traverser.
func (g *LayoutGenerator) calculatePathLength(level meta.DifficultyLevel, cleared int) int {
	switch level {
	case meta.LevelEasy:
		return 4 + cleared
	case meta.LevelNormal:
		return 5 + cleared*2
	default:
		return 6 + cleared*2
	}
}

// getZoneSizeRange définit les dimensions min/max d'une grille.
func (g *LayoutGenerator) getZoneSizeRange(level meta.DifficultyLevel) (int, int) {
	switch level {
	case meta.LevelEasy:
		return 3, 4
	case meta.LevelNormal:
		return 4, 5
	default:
		return 5, 6
	}
}

// findAvailableDirectionAndCoords localise un emplacement adjacent libre pour une nouvelle zone.
func (g *LayoutGenerator) findAvailableDirectionAndCoords(plane *DreamPlane, zoneID string) (Direction, Position, bool) {
	currentPos := plane.Coords[zoneID]
	allDirs := []Direction{North, South, East, West}
	rand.Shuffle(len(allDirs), func(i, j int) {
		allDirs[i], allDirs[j] = allDirs[j], allDirs[i]
	})

	for _, d := range allDirs {
		if _, exists := plane.GetConnectedZone(zoneID, d); exists {
			continue
		}

		vec := DirectionVector(d)
		newPos := Position{X: currentPos.X + vec.X, Y: currentPos.Y + vec.Y}

		if newPos.X < 0 || newPos.X > 8 || newPos.Y < 0 || newPos.Y > 8 {
			continue
		}

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

// addDeadEnds injecte des zones optionnelles pour l'exploration.
func (g *LayoutGenerator) addDeadEnds(plane *DreamPlane, path []string, biomes []BiomeType, sizeFunc func() int) {
	numDeadEnds := rand.Intn(2) + 1
	for i := 0; i < numDeadEnds; i++ {
		parentID := path[rand.Intn(len(path))]
		dir, coords, ok := g.findAvailableDirectionAndCoords(plane, parentID)
		if !ok {
			continue
		}

		deadID := fmt.Sprintf("deadend_%d", i+1)
		biome := biomes[rand.Intn(len(biomes))]
		grid := NewGrid(deadID, sizeFunc(), sizeFunc(), biome)
		plane.AddZone(grid)
		plane.Coords[deadID] = coords
		plane.Connect(parentID, deadID, dir)
	}
}

// SetupPortalArea configure les dolmens/obélisques autour du portail dans une zone 6x6.
func (g *LayoutGenerator) SetupPortalArea(grid *Grid, isStart bool) {
	structureType := "obelisk"
	if isStart {
		structureType = "dolmen"
	}

	corners := map[Position]bool{
		{1, 1}: true, {1, 4}: true, {4, 1}: true, {4, 4}: true,
	}
	portals := map[Position]bool{
		{2, 2}: true, {2, 3}: true, {3, 2}: true, {3, 3}: true,
	}

	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			pos := Position{X: x, Y: y}
			plot, err := grid.Get(pos)
			if err != nil {
				continue
			}

			if corners[pos] {
				plot.StructureID = fmt.Sprintf("struct_%s_%d_%d", structureType, pos.X, pos.Y)
				plot.Empty = false
			} else if portals[pos] {
				if isStart {
					plot.StructureID = "start_portal"
				} else {
					plot.StructureID = "finish_portal"
				}
				plot.Empty = false
			} else {
				plot.Empty = true
				plot.StructureID = ""
			}
		}
	}
}
