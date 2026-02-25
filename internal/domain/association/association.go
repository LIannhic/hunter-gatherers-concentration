package association

import (
	"errors"
	"fmt"
)

// Type d'association
type Type int

const (
	Identical Type = iota // Paire identique (Memory classique)
	Logical               // Clé + Serrure, Marteau + Enclume
	Elemental             // Feu + Bois, Eau + Plante
	Narrative             // Indices d'histoire, symboles liés
)

func (t Type) String() string {
	switch t {
	case Identical:
		return "identical"
	case Logical:
		return "logical"
	case Elemental:
		return "elemental"
	case Narrative:
		return "narrative"
	}
	return "unknown"
}

// Resultat d'une association
type Result struct {
	Success    bool
	Type       Type
	ProducedID string // ID de l'entité créée (ressource, artefact)
	CapturedID string // ID de la créature capturée (si applicable)
	Effects    []Effect
	Message    string
}

type Effect struct {
	Type     string // "heal", "damage", "reveal", "transform"
	Target   string // "player", "creature", "board"
	Value    int
	Metadata map[string]interface{}
}

// Strategy interface pour les différents types d'association
type Strategy interface {
	Type() Type
	CanAssociate(a, b Matchable) bool
	Resolve(a, b Matchable) (Result, error)
}

// Matchable interface minimale pour l'association
type Matchable interface {
	GetMatchID() string
	GetLogicKey() string
	GetElement() string
	GetNarrativeTag() string
	GetMatchTypes() []string
}

// --- Implémentations concrètes des stratégies ---

// IdenticalStrategy : paires identiques (même ID)
type IdenticalStrategy struct{}

func (s *IdenticalStrategy) Type() Type { return Identical }

func (s *IdenticalStrategy) CanAssociate(a, b Matchable) bool {
	return a.GetMatchID() == b.GetMatchID() && a.GetMatchID() != ""
}

func (s *IdenticalStrategy) Resolve(a, b Matchable) (Result, error) {
	if !s.CanAssociate(a, b) {
		return Result{Success: false}, errors.New("pas une paire identique")
	}
	return Result{
		Success: true,
		Type:    Identical,
		Message: "Paire identique trouvée !",
		Effects: []Effect{
			{Type: "collect", Target: "player"},
		},
	}, nil
}

// LogicalStrategy : associations logiques (clé/serrure)
type LogicalStrategy struct {
	Pairs map[string]string // "key" -> "lock", "hammer" -> "anvil"
}

func NewLogicalStrategy() *LogicalStrategy {
	return &LogicalStrategy{
		Pairs: map[string]string{
			"key":         "lock",
			"lock":        "key",
			"hammer":      "anvil",
			"anvil":       "hammer",
			"lens":        "hidden_rune",
			"hidden_rune": "lens",
		},
	}
}

func (s *LogicalStrategy) Type() Type { return Logical }

func (s *LogicalStrategy) CanAssociate(a, b Matchable) bool {
	expected, ok := s.Pairs[a.GetLogicKey()]
	return ok && expected == b.GetLogicKey()
}

func (s *LogicalStrategy) Resolve(a, b Matchable) (Result, error) {
	if !s.CanAssociate(a, b) {
		return Result{Success: false}, errors.New("association logique invalide")
	}

	// Détermine quel côté est l'outil vs la cible
	tool, target := a.GetLogicKey(), b.GetLogicKey()
	if a.GetLogicKey() == "lock" || a.GetLogicKey() == "anvil" {
		tool, target = target, tool
	}

	return Result{
		Success: true,
		Type:    Logical,
		Message: fmt.Sprintf("Association logique: %s + %s", tool, target),
		Effects: []Effect{
			{Type: "unlock", Target: "board", Metadata: map[string]interface{}{"tool": tool}},
			{Type: "synthesize", Target: "player"},
		},
	}, nil
}

// ElementalStrategy : affinités élémentaires
type ElementalStrategy struct {
	Affinities map[string][]string // "fire" -> ["wood", "oil"], "water" -> ["fire", "plant"]
}

func NewElementalStrategy() *ElementalStrategy {
	return &ElementalStrategy{
		Affinities: map[string][]string{
			"fire":     {"wood", "oil", "ice"},
			"water":    {"fire", "lava", "salt"},
			"earth":    {"water", "air"},
			"air":      {"earth", "poison"},
			"life":     {"ethereal", "water"},
			"ethereal": {"life", "crystal"},
		},
	}
}

func (s *ElementalStrategy) Type() Type { return Elemental }

func (s *ElementalStrategy) CanAssociate(a, b Matchable) bool {
	elemA, elemB := a.GetElement(), b.GetElement()
	if elemA == "" || elemB == "" {
		return false
	}

	// Vérifie si A réagit avec B ou vice versa
	if compat, ok := s.Affinities[elemA]; ok {
		for _, e := range compat {
			if e == elemB {
				return true
			}
		}
	}
	return false
}

func (s *ElementalStrategy) Resolve(a, b Matchable) (Result, error) {
	if !s.CanAssociate(a, b) {
		return Result{Success: false}, errors.New("affinité élémentaire inexistante")
	}

	return Result{
		Success: true,
		Type:    Elemental,
		Message: fmt.Sprintf("Réaction élémentaire: %s + %s", a.GetElement(), b.GetElement()),
		Effects: []Effect{
			{Type: "transform", Target: "board", Metadata: map[string]interface{}{
				"elements": []string{a.GetElement(), b.GetElement()},
			}},
			{Type: "create_resource", Target: "player"},
		},
	}, nil
}

// NarrativeStrategy : liens d'histoire et symboles
type NarrativeStrategy struct {
	Stories map[string][]string // "sun_ritual" -> ["dawn_symbol", "solar_disk", "chant"]
}

func NewNarrativeStrategy() *NarrativeStrategy {
	return &NarrativeStrategy{
		Stories: map[string][]string{
			"first_hunt": {"spear", "blood_trail", "moon"},
			"healing":    {"herb", "water", "prayer"},
			"prophecy":   {"star_map", "crystal", "whisper"},
		},
	}
}

func (s *NarrativeStrategy) Type() Type { return Narrative }

func (s *NarrativeStrategy) CanAssociate(a, b Matchable) bool {
	tagA, tagB := a.GetNarrativeTag(), b.GetNarrativeTag()
	if tagA == "" || tagB == "" {
		return false
	}

	// Vérifie si les deux tags font partie d'une même histoire
	for _, elements := range s.Stories {
		hasA, hasB := false, false
		for _, e := range elements {
			if e == tagA {
				hasA = true
			}
			if e == tagB {
				hasB = true
			}
		}
		if hasA && hasB && tagA != tagB {
			return true
		}
	}
	return false
}

func (s *NarrativeStrategy) Resolve(a, b Matchable) (Result, error) {
	if !s.CanAssociate(a, b) {
		return Result{Success: false}, errors.New("pas de lien narratif")
	}

	// Trouve l'histoire concernée
	storyName := "unknown"
	for name, elements := range s.Stories {
		hasA, hasB := false, false
		for _, e := range elements {
			if e == a.GetNarrativeTag() {
				hasA = true
			}
			if e == b.GetNarrativeTag() {
				hasB = true
			}
		}
		if hasA && hasB {
			storyName = name
			break
		}
	}

	return Result{
		Success: true,
		Type:    Narrative,
		Message: fmt.Sprintf("Fragment d'histoire découvert: %s", storyName),
		Effects: []Effect{
			{Type: "lore", Target: "player", Metadata: map[string]interface{}{"story": storyName}},
			{Type: "reveal_hidden", Target: "board"},
		},
	}, nil
}

// Engine orchestre les stratégies
type Engine struct {
	strategies []Strategy
}

func NewEngine() *Engine {
	return &Engine{
		strategies: []Strategy{
			&IdenticalStrategy{},
			NewLogicalStrategy(),
			NewElementalStrategy(),
			NewNarrativeStrategy(),
		},
	}
}

func (e *Engine) TryAssociate(a, b Matchable) (Result, error) {
	// Essaie chaque stratégie dans l'ordre de spécificité
	for _, strategy := range e.strategies {
		if strategy.CanAssociate(a, b) {
			return strategy.Resolve(a, b)
		}
	}
	return Result{Success: false}, errors.New("aucune association possible")
}

func (e *Engine) RegisterStrategy(s Strategy) {
	e.strategies = append([]Strategy{s}, e.strategies...) // Priorité aux nouvelles
}

func (e *Engine) GetStrategies() []Strategy {
	result := make([]Strategy, len(e.strategies))
	copy(result, e.strategies)
	return result
}
