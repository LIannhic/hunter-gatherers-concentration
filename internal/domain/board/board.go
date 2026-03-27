package board

import (
	"errors"
	"fmt"
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

// Tile représente une case du plateau de jeu
// Elle ne porte plus d'état, car l'état appartient à l'entité posée dessus
type Tile struct {
	Position    Position
	EntityID    string // Référence vers l'entité présente (si existante)
	StructureID string // Référence vers une structure (terrier, etc.)
	Modifier    TileModifier
}

func (t *Tile) String() string {
	return fmt.Sprintf("Tile[%s entity=%s]", t.Position.String(), t.EntityID)
}

type TileModifier struct {
	Concealed    bool // Dissimulation (hautes herbes)
	Obstructed   bool // Entrave (ronces)
	LuminousHint bool // Rayonner (indices visuels)
}

// Grid est le plateau de jeu
type Grid struct {
	ID            string
	Width, Height int
	Tiles         map[Position]*Tile
}

func NewGrid(id string, width, height int) *Grid {
	g := &Grid{
		ID:     id,
		Width:  width,
		Height: height,
		Tiles:  make(map[Position]*Tile),
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pos := Position{X: x, Y: y}
			g.Tiles[pos] = &Tile{
				Position: pos,
			}
		}
	}
	return g
}

func (g *Grid) Get(pos Position) (*Tile, error) {
	if !g.IsValid(pos) {
		return nil, errors.New("position hors limites")
	}
	tile, ok := g.Tiles[pos]
	if !ok {
		return nil, errors.New("tuile inexistante")
	}
	return tile, nil
}

func (g *Grid) IsValid(pos Position) bool {
	return pos.X >= 0 && pos.X < g.Width && pos.Y >= 0 && pos.Y < g.Height
}

func (g *Grid) GetNeighbors(pos Position) []*Tile {
	var neighbors []*Tile
	directions := []Direction{North, South, East, West}

	for _, dir := range directions {
		newPos := pos.Add(dir.Vector())
		if tile, err := g.Get(newPos); err == nil {
			neighbors = append(neighbors, tile)
		}
	}
	return neighbors
}

// PlaceEntity place une entité sur une tuile
func (g *Grid) PlaceEntity(pos Position, entityID string) error {
	tile, err := g.Get(pos)
	if err != nil {
		return err
	}
	if tile.EntityID != "" {
		return errors.New("tuile déjà occupée")
	}
	tile.EntityID = entityID
	return nil
}

// RemoveEntity retire une entité d'une tuile
func (g *Grid) RemoveEntity(pos Position) error {
	tile, err := g.Get(pos)
	if err != nil {
		return err
	}
	tile.EntityID = ""
	return nil
}

// GetTileAt retourne la tuile à une position donnée (alias pour Get)
func (g *Grid) GetTileAt(x, y int) (*Tile, error) {
	return g.Get(Position{X: x, Y: y})
}
