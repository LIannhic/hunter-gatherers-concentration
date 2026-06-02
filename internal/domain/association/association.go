package association

import (
	"errors"
	"fmt"
)

// Type d'association
type Type int

const (
	Identical   Type = iota // Paire identique (Memory classique)
	Logical                 // Clé + Serrure, Marteau + Enclume
	Elemental               // Feu + Bois, Eau + Plante
	Narrative               // Indices d'histoire, symboles liés
	Orientation             // Moitiés de flèches de navigation
	Creature                // Capture de créature
	Trap                    // Neutralisation de piège
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
	case Orientation:
		return "orientation"
	case Creature:
		return "creature_capture"
	case Trap:
		return "trap_neutralization"
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
	GetCumulationLevel() int
	IsCumulated() bool
	GetCategory() string
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

	matchType := Identical
	if a.GetCategory() == "creature" {
		matchType = Creature
	} else if a.GetCategory() == "trap" {
		matchType = Trap
	}

	return Result{
		Success: true,
		Type:    matchType,
		Message: fmt.Sprintf("Paire identique de %s trouvée !", a.GetCategory()),
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

// OrientationStrategy : Moitiés de flèches (Nord/Nord, etc.)
type OrientationStrategy struct{}

func (s *OrientationStrategy) Type() Type { return Orientation }

func (s *OrientationStrategy) CanAssociate(a, b Matchable) bool {
	// 1. On vérifie si ce sont des éléments de navigation (Sorties)
	if a.GetCategory() == "navigation" && b.GetCategory() == "navigation" {
		tagA, tagB := a.GetMatchID(), b.GetMatchID()
		// On attend des tags comme "exit_north_0" et "exit_north_1"
		if len(tagA) < 11 || len(tagB) < 11 {
			return false
		}
		prefixA := tagA[:len(tagA)-1]
		prefixB := tagB[:len(tagB)-1]

		// Même direction mais index différent (0 et 1)
		if prefixA == prefixB && tagA != tagB && (prefixA == "exit_north_" || prefixA == "exit_south_" || prefixA == "exit_east_" || prefixA == "exit_west_") {
			// On vérifie que les deux tuiles ont une orientation cohérente vers cette direction
			// Pour les sorties, on attend que le "Haut" de l'asset pointe vers l'extérieur du plateau.
			// TODO: Implémenter la vérification stricte de GetOrientation() == targetDir.
			return true
		}
		return false
	}

	// 2. TODO : Implémenter la logique générique pour toutes les tuiles
	// basées sur leur GetOrientation() réelle et la direction de la sortie
	return false
}

func (s *OrientationStrategy) Resolve(a, b Matchable) (Result, error) {
	if !s.CanAssociate(a, b) {
		return Result{Success: false}, errors.New("orientation non complémentaire")
	}

	direction := a.GetMatchID()[5 : len(a.GetMatchID())-2]

	return Result{
		Success: true,
		Type:    Orientation,
		Message: fmt.Sprintf("Passage ouvert vers le %s", direction),
		Effects: []Effect{
			{Type: "unlock_navigation", Target: "board", Metadata: map[string]interface{}{"direction": direction}},
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
			// Note : Les stratégies Logique, Élémentaire et Narrative sont désactivées
			// pour le moment. Toutes les tuiles s'associent par similarité.
			// NewLogicalStrategy(),
			// NewElementalStrategy(),
			// NewNarrativeStrategy(),
			&OrientationStrategy{},
		},
	}
}

func (e *Engine) TryAssociate(a, b Matchable) (Result, error) {
	// Centralisation du contrôle de cumul : seules les tuiles de même niveau s'associent
	if a.GetCumulationLevel() != b.GetCumulationLevel() {
		return Result{Success: false}, fmt.Errorf("niveaux de cumul incompatibles (%d vs %d)", a.GetCumulationLevel(), b.GetCumulationLevel())
	}

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
