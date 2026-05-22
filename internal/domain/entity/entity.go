package entity

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// ID unique pour les entités
type ID string

func NewID() ID {
	return ID(uuid.New().String())
}

// TileState représente l'état visuel d'une tuile (entité)
type TileState int

const (
	Hidden   TileState = 1 << iota // 1
	Revealed                       // 2
	Matched                        // 4
	Blocked                        // 8
)

func (s TileState) String() string {
	res := ""
	if s&Hidden != 0 {
		res += "hidden "
	}
	if s&Revealed != 0 {
		res += "revealed "
	}
	if s&Matched != 0 {
		res += "matched "
	}
	if s&Blocked != 0 {
		res += "blocked "
	}
	if res == "" {
		return "unknown"
	}
	return res[:len(res)-1]
}

// Type d'entité
type Type int

const (
	TypeResource Type = iota
	TypeCreature
	TypeStructure
	TypeArtefact
	TypeTrap // Changé de TypeEmptyTile à TypeTrap
	TypeLoot
)

// Direction représente les orientations cardinales
type Direction int

const (
	DirNorth Direction = iota
	DirEast
	DirSouth
	DirWest
)

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

func (t Type) String() string {
	switch t {
	case TypeResource:
		return "resource"
	case TypeCreature:
		return "creature"
	case TypeStructure:
		return "structure"
	case TypeArtefact:
		return "artefact"
	case TypeTrap:
		return "trap"
	case TypeLoot:
		return "loot"
	}
	return "unknown"
}

// Position utilitaire
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

// Entity est l'interface de base pour tous les éléments du jeu
type Entity interface {
	GetID() ID
	GetType() Type
	GetPosition() Position
	SetPosition(Position)
	GetGridID() string
	SetGridID(string)
	IsActive() bool
	Deactivate()
	GetState() TileState
	SetState(TileState)
	GetOrientation() Direction
	SetOrientation(Direction)
	AddTag(string)
	HasTag(string) bool
	RemoveTag(string)
}

// BaseEntity implémentation commune
type BaseEntity struct {
	ID       ID
	EType    Type
	Pos      Position
	GridID   string // ID du grid sur lequel se trouve l'entité
	Active   bool
	State    TileState // L'état appartient maintenant à l'entité
	Orientation Direction
	Tags     []string
	Metadata map[string]interface{}
}

func NewBaseEntity(etype Type) BaseEntity {
	return BaseEntity{
		ID:       NewID(),
		EType:    etype,
		GridID:   "", // Doit être défini après création
		Active:   true,
		State:    Hidden, // Par défaut caché
		Tags:     make([]string, 0),
		Metadata: make(map[string]interface{}),
	}
}

// GetGridID retourne l'ID du grid de l'entité
func (e *BaseEntity) GetGridID() string {
	return e.GridID
}

// SetGridID définit l'ID du grid de l'entité
func (e *BaseEntity) SetGridID(gridID string) {
	e.GridID = gridID
}

func (e *BaseEntity) GetID() ID              { return e.ID }
func (e *BaseEntity) GetType() Type          { return e.EType }
func (e *BaseEntity) GetPosition() Position  { return e.Pos }
func (e *BaseEntity) SetPosition(p Position) { e.Pos = p }
func (e *BaseEntity) IsActive() bool         { return e.Active }
func (e *BaseEntity) Deactivate()            { e.Active = false }
func (e *BaseEntity) GetState() TileState    { return e.State }
func (e *BaseEntity) SetState(s TileState)   { e.State = s }

func (e *BaseEntity) GetOrientation() Direction  { return e.Orientation }
func (e *BaseEntity) SetOrientation(o Direction) { e.Orientation = o }

func (e *BaseEntity) AddTag(tag string) {
	for _, t := range e.Tags {
		if t == tag {
			return
		}
	}
	e.Tags = append(e.Tags, tag)
}

func (e *BaseEntity) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (e *BaseEntity) RemoveTag(tag string) {
	for i, t := range e.Tags {
		if t == tag {
			e.Tags = append(e.Tags[:i], e.Tags[i+1:]...)
			return
		}
	}
}

// Manager gère toutes les entités
type Manager struct {
	entities map[ID]Entity
	byType   map[Type]map[ID]Entity
	byPos    map[Position]ID
}

func NewManager() *Manager {
	return &Manager{
		entities: make(map[ID]Entity),
		byType:   make(map[Type]map[ID]Entity),
		byPos:    make(map[Position]ID),
	}
}

func (m *Manager) Register(e Entity) {
	m.entities[e.GetID()] = e
	if m.byType[e.GetType()] == nil {
		m.byType[e.GetType()] = make(map[ID]Entity)
	}
	m.byType[e.GetType()][e.GetID()] = e
	m.byPos[e.GetPosition()] = e.GetID()
}

func (m *Manager) Remove(id ID) {
	e, ok := m.entities[id]
	if !ok {
		return
	}
	delete(m.entities, id)
	delete(m.byType[e.GetType()], id)
	delete(m.byPos, e.GetPosition())
}

func (m *Manager) Get(id ID) (Entity, bool) {
	e, ok := m.entities[id]
	return e, ok
}

func (m *Manager) GetByPosition(pos Position) (Entity, bool) {
	id, ok := m.byPos[pos]
	if !ok {
		return nil, false
	}
	return m.Get(id)
}

func (m *Manager) UpdatePosition(id ID, newPos Position) error {
	e, ok := m.entities[id]
	if !ok {
		return fmt.Errorf("entité %s non trouvée", id)
	}
	delete(m.byPos, e.GetPosition())
	e.SetPosition(newPos)
	m.byPos[newPos] = id
	return nil
}

func (m *Manager) GetByType(t Type) []Entity {
	result := make([]Entity, 0, len(m.byType[t]))
	for _, e := range m.byType[t] {
		result = append(result, e)
	}
	// Trie par ID pour avoir un ordre stable entre les frames
	sort.Slice(result, func(i, j int) bool {
		return result[i].GetID() < result[j].GetID()
	})
	return result
}

func (m *Manager) GetAllActive() []Entity {
	result := make([]Entity, 0)
	for _, e := range m.entities {
		if e.IsActive() {
			result = append(result, e)
		}
	}
	return result
}

func (m *Manager) QueryByTag(tag string) []Entity {
	result := make([]Entity, 0)
	for _, e := range m.entities {
		if e.HasTag(tag) && e.IsActive() {
			result = append(result, e)
		}
	}
	return result
}

func (m *Manager) Count() int {
	return len(m.entities)
}

func (m *Manager) CountByType(t Type) int {
	return len(m.byType[t])
}
