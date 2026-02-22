package component

import (
	"time"
)

// Component est l'interface marqueur pour l'ECS
type Component interface {
	Type() string
}

// Store stocke les composants par entité
type Store struct {
	components map[string]map[string]Component // entityID -> componentType -> Component
}

func NewStore() *Store {
	return &Store{
		components: make(map[string]map[string]Component),
	}
}

func (s *Store) Add(entityID string, c Component) {
	if s.components[entityID] == nil {
		s.components[entityID] = make(map[string]Component)
	}
	s.components[entityID][c.Type()] = c
}

func (s *Store) Get(entityID string, componentType string) (Component, bool) {
	if s.components[entityID] == nil {
		return nil, false
	}
	c, ok := s.components[entityID][componentType]
	return c, ok
}

func (s *Store) Remove(entityID string, componentType string) {
	if s.components[entityID] != nil {
		delete(s.components[entityID], componentType)
	}
}

func (s *Store) Has(entityID string, componentType string) bool {
	_, ok := s.Get(entityID, componentType)
	return ok
}

func (s *Store) GetAll(entityID string) []Component {
	if s.components[entityID] == nil {
		return nil
	}
	result := make([]Component, 0, len(s.components[entityID]))
	for _, c := range s.components[entityID] {
		result = append(result, c)
	}
	return result
}

func (s *Store) QueryByComponent(componentType string) []string {
	var result []string
	for entityID, comps := range s.components {
		if _, ok := comps[componentType]; ok {
			result = append(result, entityID)
		}
	}
	return result
}

func (s *Store) RemoveEntity(entityID string) {
	delete(s.components, entityID)
}

// --- Composants concrets ---

// Lifecycle gère les stades de maturation/dégradation
type Lifecycle struct {
	CurrentStage int
	MaxStages    int
	StageNames   []string // ex: ["bourgeon", "fruit", "gâté"]
	TurnsInStage int
	TurnsToNext  int // -1 pour infini
	CanPropagate bool
}

func (l Lifecycle) Type() string { return "lifecycle" }

func (l *Lifecycle) Progress() bool {
	l.TurnsInStage++
	if l.TurnsToNext > 0 && l.TurnsInStage >= l.TurnsToNext {
		if l.CurrentStage < l.MaxStages-1 {
			l.CurrentStage++
			l.TurnsInStage = 0
			return true // Stage changé
		}
	}
	return false
}

func (l *Lifecycle) GetCurrentStageName() string {
	if l.CurrentStage < len(l.StageNames) {
		return l.StageNames[l.CurrentStage]
	}
	return "unknown"
}

func (l *Lifecycle) IsMature() bool {
	return l.CurrentStage >= l.MaxStages/2
}

func (l *Lifecycle) IsDecayed() bool {
	return l.CurrentStage == l.MaxStages-1
}

// Visual indique comment la tuile apparaît (face cachée vs révélée)
type Visual struct {
	HiddenSprite   string
	RevealedSprite string
	BackHint       string // Indice sur le verso (rayonner)
	HasBackHint    bool
}

func (v Visual) Type() string { return "visual" }

// Matchable définit les propriétés d'association
type Matchable struct {
	MatchID      string   // ID de correspondance (pour paires identiques)
	MatchTypes   []string // Types d'association possibles
	LogicKey     string   // Pour associations logiques (clé/serrure)
	Element      string   // Pour associations élémentaires
	NarrativeTag string   // Pour associations narratives
}

func (m Matchable) Type() string { return "matchable" }

func (m Matchable) GetMatchID() string       { return m.MatchID }
func (m Matchable) GetLogicKey() string      { return m.LogicKey }
func (m Matchable) GetElement() string       { return m.Element }
func (m Matchable) GetNarrativeTag() string  { return m.NarrativeTag }
func (m Matchable) GetMatchTypes() []string { return m.MatchTypes }

// Mobility pour créatures
type Mobility struct {
	CanMove      bool
	MovePattern  string // "static", "random", "hunter", "flee"
	Speed        int    // Tuiles par tour
	LastMoveTime time.Time
}

func (m Mobility) Type() string { return "mobility" }

// Behavior pour IA des créatures
type Behavior struct {
	State          string // "idle", "hunting", "fleeing", "pollinating"
	Aggression     int    // 0-100
	Territorial    bool
	Transformation string // ex: "pollinize", "break", "fertilize"
	LeavesTraces   bool
}

func (b Behavior) Type() string { return "behavior" }

// Inventory pour stockage
type Inventory struct {
	Slots    []string // IDs des entités contenues
	MaxSlots int
}

func (i Inventory) Type() string { return "inventory" }

func (i *Inventory) Add(itemID string) bool {
	if len(i.Slots) >= i.MaxSlots {
		return false
	}
	i.Slots = append(i.Slots, itemID)
	return true
}

func (i *Inventory) Remove(itemID string) bool {
	for idx, id := range i.Slots {
		if id == itemID {
			i.Slots = append(i.Slots[:idx], i.Slots[idx+1:]...)
			return true
		}
	}
	return false
}

func (i *Inventory) Has(itemID string) bool {
	for _, id := range i.Slots {
		if id == itemID {
			return true
		}
	}
	return false
}

func (i *Inventory) Count() int {
	return len(i.Slots)
}

// Value pour ressources et score
type Value struct {
	BaseValue    int
	CurrentValue int
	DegradeRate  int // Valeur perdue par tour si non récolté
}

func (v Value) Type() string { return "value" }

// Trigger pour structures interactives
type Trigger struct {
	Condition string // ex: "reveal_with_creature"
	Action    string // ex: "creature_flee"
	Consumed  bool   // Si le trigger se désactive après usage
}

func (t Trigger) Type() string { return "trigger" }

// Concealment pour dissimulation
type Concealment struct {
	Concealed       bool
	ConcealmentType string   // "grass", "fog", "burrow"
	RevealedBy      []string // Tags qui révèlent cette dissimulation
}

func (c Concealment) Type() string { return "concealment" }
