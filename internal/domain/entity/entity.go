package entity

import (
	"fmt"

	"github.com/google/uuid"
)

// ID unique pour les entités
type ID string

func NewID() ID {
	return ID(uuid.New().String())
}

// Type d'entité
type Type int

const (
	TypeResource Type = iota
	TypeCreature
	TypeStructure
	TypeArtefact
)

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

// Entity est l'interface de base pour tous les éléments du jeu
type Entity interface {
	GetID() ID
	GetType() Type
	GetPosition() Position
	SetPosition(Position)
	IsActive() bool
	Deactivate()
}

// BaseEntity implémentation commune
type BaseEntity struct {
	ID       ID
	EType    Type
	Pos      Position
	Active   bool
	Tags     []string
	Metadata map[string]interface{}
}

func NewBaseEntity(etype Type) BaseEntity {
	return BaseEntity{
		ID:       NewID(),
		EType:    etype,
		Active:   true,
		Tags:     make([]string, 0),
		Metadata: make(map[string]interface{}),
	}
}

func (e *BaseEntity) GetID() ID              { return e.ID }
func (e *BaseEntity) GetType() Type          { return e.EType }
func (e *BaseEntity) GetPosition() Position  { return e.Pos }
func (e *BaseEntity) SetPosition(p Position) { e.Pos = p }
func (e *BaseEntity) IsActive() bool         { return e.Active }
func (e *BaseEntity) Deactivate()            { e.Active = false }

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
	result := make([]Entity, 0)
	for _, e := range m.byType[t] {
		result = append(result, e)
	}
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
		if be, ok := e.(*BaseEntity); ok {
			if be.HasTag(tag) && e.IsActive() {
				result = append(result, e)
			}
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
