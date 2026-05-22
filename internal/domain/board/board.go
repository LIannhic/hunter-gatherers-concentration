package board

import (
	"errors"
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Position est maintenant définie dans internal/domain/entity/entity.go
type Position = entity.Position

// Direction est maintenant définie dans internal/domain/entity/entity.go
type Direction = entity.Direction

const (
	North = entity.DirNorth
	East  = entity.DirEast
	South = entity.DirSouth
	West  = entity.DirWest
)

// FlipDirection est maintenant définie dans internal/domain/entity/entity.go
type FlipDirection = entity.FlipDirection

const (
	FlipTop         = entity.FlipTop
	FlipTopRight    = entity.FlipTopRight
	FlipRight       = entity.FlipRight
	FlipBottomRight = entity.FlipBottomRight
	FlipBottom      = entity.FlipBottom
	FlipBottomLeft  = entity.FlipBottomLeft
	FlipLeft        = entity.FlipLeft
	FlipTopLeft     = entity.FlipTopLeft
	FlipCenter      = entity.FlipCenter
)

func CalculateFlipDirection(tileSize, localX, localY int) FlipDirection {
	return entity.CalculateFlipDirection(tileSize, localX, localY)
}

// Bearing représente l'orientation de la Grille (Cardinaux)
type Bearing int

const (
	BearingNorth Bearing = iota
	BearingEast
	BearingSouth
	BearingWest
	BearingMirror
)

// Slope représente l'inclinaison de la Parcelle (Topographie)
// C'est cette pente qui dicte comment la tuile se "recouche" en mode caché.
type Slope int

const (
	SlopeTop Slope = iota
	SlopeTopRight
	SlopeRight
	SlopeBottomRight
	SlopeBottom
	SlopeBottomLeft
	SlopeLeft
	SlopeTopLeft
	SlopeFlat // État neutre
)

// --- SYSTÈME ENVIRONNEMENTAL ---

type BiomeType string

const (
	BiomeForest BiomeType = "forest"
	BiomeCave   BiomeType = "cave"
	BiomeDesert BiomeType = "desert"
)

type Climate string

const (
	ClimateTemperate Climate = "temperate"
	ClimateHumid     Climate = "humid"
	ClimateArid      Climate = "arid"
)

type Season int

const (
	SeasonAwakening Season = iota
	SeasonZenith
	SeasonDecay
	SeasonSlumber
)

type SuccessionStage int

const (
	StagePreliminary SuccessionStage = iota
	StagePioneer
	StageClimax
)

func DirectionVector(d Direction) Position {
	switch d {
	case North:
		return Position{X: 0, Y: -1}
	case South:
		return Position{X: 0, Y: 1}
	case East:
		return Position{X: 1, Y: 0}
	case West:
		return Position{X: -1, Y: 0}
	}
	return Position{X: 0, Y: 0}
}

// Plot représente une case du plateau de jeu
// Elle ne porte plus d'état, car l'état appartient à l'entité posée dessus
type Plot struct {
	Position    Position
	EntitiesID  []string
	StructureID string
	LocalStage  SuccessionStage
	Tilt        Slope
	Modifier    PlotModifier
}

func (p *Plot) PushEntity(id string) {
	p.EntitiesID = append(p.EntitiesID, id)
}

func (p *Plot) PushEntityToBottom(id string) {
	p.EntitiesID = append([]string{id}, p.EntitiesID...)
}

func (p *Plot) PopEntity() (string, bool) {
	if len(p.EntitiesID) == 0 {
		return "", false
	}
	lastIdx := len(p.EntitiesID) - 1
	id := p.EntitiesID[lastIdx]
	p.EntitiesID = p.EntitiesID[:lastIdx]
	return id, true
}

func (p *Plot) String() string {
	return fmt.Sprintf("Plot[%v entities=%v]", p.Position, p.EntitiesID)
}

type PlotModifier struct {
	Concealed    bool // Dissimulation (hautes herbes)
	Obstructed   bool // Entrave (ronces)
	LuminousHint bool // Rayonner (indices visuels)
}

// Grid est le plateau de jeu
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

	// V0.2: Navigation
	InitialMatchableCount int
	ExitsState            map[Direction][2]entity.TileState
	NavigationForcedOpen  bool
}

func NewGrid(id string, width, height int, biome BiomeType) *Grid {
	g := &Grid{
		ID:            id,
		Width:         width,
		Height:        height,
		Biome:         biome,
		CurrentSeason: SeasonAwakening,
		GlobalStage:   StagePreliminary,
		Plots:         make(map[Position]*Plot),
		ExitsState:    make(map[Direction][2]entity.TileState),
	}

	for d := entity.DirNorth; d <= entity.DirWest; d++ {
		g.ExitsState[d] = [2]entity.TileState{entity.Hidden | entity.Blocked, entity.Hidden | entity.Blocked}
	}

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
	return g
}

func (g *Grid) IsValid(pos Position) bool {
	return pos.X >= 0 && pos.X < g.Width && pos.Y >= 0 && pos.Y < g.Height
}

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

// PlaceEntity ajoute l'entité au sommet de la pile
func (g *Grid) PlaceEntity(pos Position, entityID string) error {
	plot, err := g.Get(pos)
	if err != nil {
		return err
	}
	plot.PushEntity(entityID)
	return nil
}

// PlaceEntityAtBottom ajoute l'entité à la base de la pile
func (g *Grid) PlaceEntityAtBottom(pos Position, entityID string) error {
	plot, err := g.Get(pos)
	if err != nil {
		return err
	}
	plot.PushEntityToBottom(entityID)
	return nil
}

// RemoveEntity retire une entité spécifique de la pile à une position donnée
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

func (g *Grid) GetNeighbors(pos Position) []*Plot {
	var neighbors []*Plot

	dirs := []Position{
		{X: 0, Y: -1}, {X: 0, Y: 1}, {X: 1, Y: 0}, {X: -1, Y: 0}, // N, S, E, W
		{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1}, // Diagonales (NW, NE, SW, SE)
	}

	for _, d := range dirs {
		targetPos := pos.Add(d)
		if plot, err := g.Get(targetPos); err == nil {
			neighbors = append(neighbors, plot)
		}
	}

	return neighbors
}

func (g *Grid) GetTileAt(x, y int) (*Plot, error) {
	return g.Get(Position{X: x, Y: y})
}

// RotateClockwise effectue une rotation à 90° dans le sens horaire du plateau
func (g *Grid) RotateClockwise() {
	// 1. Mise à jour du Bearing (Orientation globale)
	g.MainBearing = Bearing((int(g.MainBearing) + 1) % 4)

	// 2. Transformation des coordonnées des parcelles (Plots)
	newPlots := make(map[Position]*Plot)
	for oldPos, plot := range g.Plots {
		newPos := g.TransformPosition(oldPos)
		plot.Position = newPos
		newPlots[newPos] = plot
	}
	g.Plots = newPlots

	// 3. Rotation des sorties (ExitsState)
	newExitsState := make(map[Direction][2]entity.TileState)
	// Nord -> Est -> Sud -> West -> Nord
	newExitsState[East] = g.ExitsState[North]
	newExitsState[South] = g.ExitsState[East]
	newExitsState[West] = g.ExitsState[South]
	newExitsState[North] = g.ExitsState[West]
	g.ExitsState = newExitsState
}

// TransformPosition transforme une position locale lors d'une rotation horaire de 90°
func (g *Grid) TransformPosition(pos Position) Position {
	return Position{
		X: g.Height - 1 - pos.Y,
		Y: pos.X,
	}
}
