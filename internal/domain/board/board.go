package board

import (
	"errors"
	"fmt"
)

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

// Position représente une coordonnée sur le plateau
type Position struct {
	X, Y int
}

func (p Position) Add(other Position) Position {
	return Position{X: p.X + other.X, Y: p.Y + other.Y}
}

func (p Position) Distance(other Position) int {
	dx := p.X - other.X
	if dx < 0 {
		dx = -dx
	}
	dy := p.Y - other.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy // Distance de Manhattan
}

func (p Position) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

// Direction pour les déplacements
type Direction int

const (
	North Direction = iota
	South
	East
	West
)

func (d Direction) Vector() Position {
	switch d {
	case North:
		return Position{0, -1}
	case South:
		return Position{0, 1}
	case East:
		return Position{1, 0}
	case West:
		return Position{-1, 0}
	}
	return Position{0, 0}
}

// FlipDirection représente la direction de flip d'une tuile lors du reveal
// Cette information est purement visuelle et n'impacte pas la logique métier
type FlipDirection int

const (
	FlipTop FlipDirection = iota
	FlipTopRight
	FlipRight
	FlipBottomRight
	FlipBottom
	FlipBottomLeft
	FlipLeft
	FlipTopLeft
	FlipCenter // Flip direct (clic au centre)
)

func (f FlipDirection) String() string {
	switch f {
	case FlipTop:
		return "top"
	case FlipTopRight:
		return "top-right"
	case FlipRight:
		return "right"
	case FlipBottomRight:
		return "bottom-right"
	case FlipBottom:
		return "bottom"
	case FlipBottomLeft:
		return "bottom-left"
	case FlipLeft:
		return "left"
	case FlipTopLeft:
		return "top-left"
	case FlipCenter:
		return "center"
	}
	return "unknown"
}

// ToRotationAngles retourne les angles de rotation (X, Y) pour l'animation de flip
// en degrés, selon la direction. Utilisé par le renderer pour l'animation.
func (f FlipDirection) ToRotationAngles() (rotateX, rotateY float64) {
	switch f {
	case FlipTop:
		return -90, 0
	case FlipTopRight:
		return -45, 45
	case FlipRight:
		return 0, 90
	case FlipBottomRight:
		return 45, 45
	case FlipBottom:
		return 90, 0
	case FlipBottomLeft:
		return 45, -45
	case FlipLeft:
		return 0, -90
	case FlipTopLeft:
		return -45, -45
	case FlipCenter:
		return 0, 0
	}
	return 0, 0
}

// CalculateFlipDirection détermine la direction de flip basée sur la position
// du clic dans une tuile. tileSize est la taille de la tuile, localX et localY
// sont les coordonnées du clic relatives à la tuile (0,0 = coin supérieur gauche)
func CalculateFlipDirection(tileSize, localX, localY int) FlipDirection {
	// Définit les zones (en pourcentage de la taille de la tuile)
	// Centre : 40% au milieu
	// Bords : 30% de chaque côté
	centerStart := tileSize * 35 / 100
	centerEnd := tileSize * 65 / 100

	// Détermine la zone verticale
	var vertical int // 0 = top, 1 = center, 2 = bottom
	if localY < centerStart {
		vertical = 0 // top
	} else if localY > centerEnd {
		vertical = 2 // bottom
	} else {
		vertical = 1 // center
	}

	// Détermine la zone horizontale
	var horizontal int // 0 = left, 1 = center, 2 = right
	if localX < centerStart {
		horizontal = 0 // left
	} else if localX > centerEnd {
		horizontal = 2 // right
	} else {
		horizontal = 1 // center
	}

	// Combine pour obtenir la direction
	switch vertical {
	case 0: // top
		switch horizontal {
		case 0:
			return FlipTopLeft
		case 1:
			return FlipTop
		case 2:
			return FlipTopRight
		}
	case 1: // center
		switch horizontal {
		case 0:
			return FlipLeft
		case 1:
			return FlipCenter
		case 2:
			return FlipRight
		}
	case 2: // bottom
		switch horizontal {
		case 0:
			return FlipBottomLeft
		case 1:
			return FlipBottom
		case 2:
			return FlipBottomRight
		}
	}

	return FlipCenter
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
	return fmt.Sprintf("Plot[%s entities=%v]", p.Position.String(), p.EntitiesID)
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
		{0, -1}, {0, 1}, {1, 0}, {-1, 0}, // N, S, E, W
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1}, // Diagonales (NW, NE, SW, SE)
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
