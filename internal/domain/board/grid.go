package board

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Grid représente un plateau de jeu individuel avec sa géométrie et ses entités.
type Grid struct {
	ID             string
	Width, Height  int
	Biome          BiomeType
	Climate        Climate
	CurrentSeason  Season
	SeasonProgress int
	TimeDilation   float64
	GlobalStage    SuccessionStage
	MainBearing    Bearing
	Plots          map[Position]*Plot

	InitialMatchableCount int
	ExitsState            map[Direction][2]entity.TileState
	ExitsTransform        map[Direction][2]entity.Transformation
	NavigationForcedOpen  bool
	MatchedTargetsCount   int
	LastNavigationOpen    bool // NOUVEAU: État mémorisé pour détecter les changements (Sealing/Unsealing)
}

// NewGrid instancie et configure une nouvelle grille avec ses pentes initiales selon le biome.
func NewGrid(id string, width, height int, biome BiomeType) *Grid {
	g := &Grid{
		ID:             id,
		Width:          width,
		Height:         height,
		Biome:          biome,
		CurrentSeason:  SeasonAwakening,
		GlobalStage:    StagePreliminary,
		Plots:          make(map[Position]*Plot),
		ExitsState:     make(map[Direction][2]entity.TileState),
		ExitsTransform: make(map[Direction][2]entity.Transformation),
	}

	for d := entity.DirNorth; d <= entity.DirWest; d++ {
		g.ExitsState[d] = [2]entity.TileState{entity.Hidden | entity.Blocked, entity.Hidden | entity.Blocked}
		g.ExitsTransform[d] = [2]entity.Transformation{entity.TransIdentity, entity.TransIdentity}
	}

	// 1. Initialisation des parcelles à plat
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pos := Position{X: x, Y: y}
			g.Plots[pos] = &Plot{
				Position:   pos,
				EntitiesID: []string{},
				LocalStage: StagePreliminary,
				Tilt:       SlopeFlat,
			}
		}
	}

	if biome == BiomeDefault {
		return g
	}

	// 2. Application des règles de relief (Slopes)
	targetPos := Position{X: rand.Intn(width), Y: rand.Intn(height)}

	switch biome {
	case BiomeForest:
		slope := ChooseRandomGlobalSlope()
		for _, plot := range g.Plots {
			plot.Tilt = slope
		}
	case BiomeCave:
		for pos, plot := range g.Plots {
			plot.Tilt = CalculateSlopeDirectionCardinal(pos, targetPos)
		}
	case BiomeDesert:
		for pos, plot := range g.Plots {
			plot.Tilt = InvertSlope(CalculateSlopeDirectionCardinal(pos, targetPos))
		}
	case BiomeSwamp:
		ApplySpiralVortex(g.Plots, targetPos, g.Width, g.Height, true, true)
	}

	return g
}

// IsValid vérifie si une position est dans les limites de la grille.
func (g *Grid) IsValid(pos Position) bool {
	return pos.X >= 0 && pos.X < g.Width && pos.Y >= 0 && pos.Y < g.Height
}

// Get retourne la parcelle à la position donnée.
func (g *Grid) Get(pos Position) (*Plot, error) {
	if !g.IsValid(pos) {
		return nil, fmt.Errorf("position %v hors limites", pos)
	}
	p, ok := g.Plots[pos]
	if !ok {
		return nil, errors.New("parcelle inexistante")
	}
	return p, nil
}

// PlaceEntity ajoute une entité à une position donnée.
func (g *Grid) PlaceEntity(pos Position, entityID string) error {
	plot, err := g.Get(pos)
	if err != nil {
		return err
	}
	plot.PushEntity(entityID)
	return nil
}

// PlaceEntityAtBottom ajoute une entité au bas de la pile d'une parcelle.
func (g *Grid) PlaceEntityAtBottom(pos Position, entityID string) error {
	plot, err := g.Get(pos)
	if err != nil {
		return err
	}
	plot.PushEntityToBottom(entityID)
	return nil
}

// RemoveEntity retire une entité spécifique d'une position.
func (g *Grid) RemoveEntity(pos Position, entityID string) (string, error) {
	plot, err := g.Get(pos)
	if err != nil {
		return "", err
	}

	foundIdx := -1
	for i, id := range plot.EntitiesID {
		if id == entityID {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return "", fmt.Errorf("entité %s non trouvée à la position %v", entityID, pos)
	}

	plot.EntitiesID = append(plot.EntitiesID[:foundIdx], plot.EntitiesID[foundIdx+1:]...)
	return entityID, nil
}

// GetNeighbors retourne les parcelles adjacentes (8 directions).
func (g *Grid) GetNeighbors(pos Position) []*Plot {
	var neighbors []*Plot
	dirs := []Position{
		{0, -1}, {0, 1}, {1, 0}, {-1, 0},
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
	}
	for _, d := range dirs {
		targetPos := pos.Add(d)
		if plot, err := g.Get(targetPos); err == nil {
			neighbors = append(neighbors, plot)
		}
	}
	return neighbors
}

// GetTileAt est un alias pour Get utilisant des entiers.
func (g *Grid) GetTileAt(x, y int) (*Plot, error) {
	return g.Get(Position{X: x, Y: y})
}

// RotateClockwise fait pivoter la grille de 90 degrés dans le sens horaire.
func (g *Grid) RotateClockwise() {
	oldHeight := g.Height
	g.MainBearing = Bearing((int(g.MainBearing) + 1) % 4)

	newPlots := make(map[Position]*Plot)
	for oldPos, plot := range g.Plots {
		// On utilise oldHeight pour la transformation car g.Height n'a pas encore changé
		newPos := Position{
			X: oldHeight - 1 - oldPos.Y,
			Y: oldPos.X,
		}
		plot.Position = newPos
		// Rotation de la pente (Slope) de 90° (+2 pas de 45°)
		plot.Tilt = RotateSlope(plot.Tilt, 2)
		newPlots[newPos] = plot
	}
	g.Plots = newPlots

	// Permutation des dimensions
	g.Width, g.Height = g.Height, g.Width

	// Rotation des sorties
	newExitsState := make(map[Direction][2]entity.TileState)
	newExitsTransform := make(map[Direction][2]entity.Transformation)

	directions := []Direction{North, East, South, West}
	rotatedDirs := []Direction{East, South, West, North}

	for i, dir := range directions {
		newDir := rotatedDirs[i]
		newExitsState[newDir] = g.ExitsState[dir]
		newExitsTransform[newDir] = g.ExitsTransform[dir]
	}

	g.ExitsState = newExitsState
	g.ExitsTransform = newExitsTransform
}

// TransformPosition calcule la nouvelle position après une rotation de 90°.
// Note: Cette fonction doit être utilisée avec précaution car elle dépend de la hauteur de la grille.
func (g *Grid) TransformPosition(pos Position, heightBeforeRotation int) Position {
	return Position{
		X: heightBeforeRotation - 1 - pos.Y,
		Y: pos.X,
	}
}
