package player

// CognitiveImpairment représente un trouble cognitif affectant l'interface
type CognitiveImpairment int

const (
	ImpairmentNone CognitiveImpairment = iota
	ImpairmentAphasia   // Difficulté avec les symboles / labels
	ImpairmentAgnosia   // Difficulté de reconnaissance visuelle
	ImpairmentAtaxia    // Perturbation de la coordination / position
	ImpairmentAmnesia   // Perturbation de la mémoire / séquence
	ImpairmentVertigo   // Vertige / distorsion visuelle (Fleeing Sprite)
)

func (c CognitiveImpairment) String() string {
	switch c {
	case ImpairmentAphasia:
		return "aphasia"
	case ImpairmentAgnosia:
		return "agnosia"
	case ImpairmentAtaxia:
		return "ataxia"
	case ImpairmentAmnesia:
		return "amnesia"
	case ImpairmentVertigo:
		return "vertigo"
	default:
		return "none"
	}
}

// StatusEffects regroupe les altérations d'état du joueur
type StatusEffects struct {
	ActiveImpairments map[CognitiveImpairment]bool
}

// NewStatusEffects crée un nouveau gestionnaire d'effets
func NewStatusEffects() *StatusEffects {
	return &StatusEffects{
		ActiveImpairments: make(map[CognitiveImpairment]bool),
	}
}

// AddImpairment active un trouble cognitif
func (s *StatusEffects) AddImpairment(imp CognitiveImpairment) {
	s.ActiveImpairments[imp] = true
}

// RemoveImpairment désactive un trouble cognitif
func (s *StatusEffects) RemoveImpairment(imp CognitiveImpairment) {
	delete(s.ActiveImpairments, imp)
}

// HasImpairment vérifie si un trouble est actif
func (s *StatusEffects) HasImpairment(imp CognitiveImpairment) bool {
	return s.ActiveImpairments[imp]
}

// HasAnyImpairment retourne true si au moins un trouble cognitif est actif
func (s *StatusEffects) HasAnyImpairment() bool {
	return len(s.ActiveImpairments) > 0
}

// Clear supprime tous les troubles
func (s *StatusEffects) Clear() {
	s.ActiveImpairments = make(map[CognitiveImpairment]bool)
}

// GetActiveImpairments retourne la liste des troubles actifs
func (s *StatusEffects) GetActiveImpairments() []CognitiveImpairment {
	result := make([]CognitiveImpairment, 0, len(s.ActiveImpairments))
	for imp := range s.ActiveImpairments {
		result = append(result, imp)
	}
	return result
}
