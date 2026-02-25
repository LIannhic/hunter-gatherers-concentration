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

// TileState représente l'état visuel d'une tuile
type TileState int

const (
	Hidden TileState = iota
	Revealed
	Matched
	Blocked
)

func (s TileState) String() string {
	switch s {
	case Hidden:
		return "hidden"
	case Revealed:
		return "revealed"
	case Matched:
		return "matched"
	case Blocked:
		return "blocked"
	}
	return "unknown"
}

// Tile représente une case du plateau de jeu
type Tile struct {
	Position    Position
	State       TileState
	EntityID    string // Référence vers l'entité présente (si existante)
	StructureID string // Référence vers une structure (terrier, etc.)
	Modifier    TileModifier
}

func (t *Tile) String() string {
	return fmt.Sprintf("Tile[%s state=%s entity=%s]", t.Position.String(), t.State.String(), t.EntityID)
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
				State:    Hidden,
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

// GetEmptyTiles retourne toutes les tuiles vides (sans entité)
func (g *Grid) GetEmptyTiles() []*Tile {
	var empty []*Tile
	for _, tile := range g.Tiles {
		if tile.EntityID == "" && !tile.Modifier.Obstructed {
			empty = append(empty, tile)
		}
	}
	return empty
}

// Reveal retourne une tuile (action de dévoiler)
func (g *Grid) Reveal(pos Position) (*Tile, error) {
	tile, err := g.Get(pos)
	if err != nil {
		return nil, err
	}
	if tile.State != Hidden {
		return nil, errors.New("tuile déjà révélée ou appairée")
	}
	tile.State = Revealed
	return tile, nil
}

// Hide remet une tuile face cachée (pour certains effets)
func (g *Grid) Hide(pos Position) error {
	tile, err := g.Get(pos)
	if err != nil {
		return err
	}
	if tile.State == Matched {
		return errors.New("impossible de cacher une tuile appairée")
	}
	tile.State = Hidden
	return nil
}

// Match marque une tuile comme appairée avec succès
func (g *Grid) Match(pos Position) error {
	tile, err := g.Get(pos)
	if err != nil {
		return err
	}
	if tile.State != Revealed {
		return errors.New("seule une tuile révélée peut être appairée")
	}
	tile.State = Matched
	return nil
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

// CountByState compte les tuiles par état
func (g *Grid) CountByState(state TileState) int {
	count := 0
	for _, tile := range g.Tiles {
		if tile.State == state {
			count++
		}
	}
	return count
}
