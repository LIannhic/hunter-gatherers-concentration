package system

import (
	"fmt"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// ComboSystem gère la progression des combos et les récompenses associées.
type ComboSystem struct {
	world          *World
	comboCount     int
	lastMatchTurn  int
	matchThisTurn  bool
}

func NewComboSystem(world *World) *ComboSystem {
	s := &ComboSystem{
		world: world,
	}
	s.initListeners()
	return s
}

func (s *ComboSystem) Priority() int {
	return 10 // S'exécute après les autres systèmes logiques
}

func (s *ComboSystem) initListeners() {
	// S'abonne aux matchs réussis
	s.world.EventBus.SubscribeFunc(event.TileMatched, func(e event.Event) {
		s.onMatch(e)
	})

	// S'abonne aux fusions (compte comme un demi-combo ou un maintien)
	s.world.EventBus.SubscribeFunc(event.TileMerged, func(e event.Event) {
		s.onMerge(e)
	})

	// S'abonne aux dégâts pour réinitialiser en cas d'erreur
	s.world.EventBus.SubscribeFunc(event.PlayerDamaged, func(e event.Event) {
		reason, _ := e.Payload["reason"].(string)
		if reason == "invalid_match" || reason == "skipped_valid_match" {
			s.Reset()
		}
	})
}

func (s *ComboSystem) Update(world *World) {
	// Si le tour se termine et qu'on n'a pas fait de match, on perd le combo
	if world.Turn > s.lastMatchTurn && !s.matchThisTurn && s.comboCount > 0 {
		s.Reset()
	}
	s.matchThisTurn = false
}

func (s *ComboSystem) onMatch(e event.Event) {
	s.matchThisTurn = true
	s.lastMatchTurn = s.world.Turn
	s.comboCount++

	fmt.Printf("[COMBO] onMatch called! comboCount=%d, turn=%d, payload=%v\n", s.comboCount, s.world.Turn, e.Payload)

	// Récupère les types d'association pour le bonus de synergie
	assocTypes, _ := e.Payload["assoc_types"].([]string)
	isSynergy := len(assocTypes) > 1

	fmt.Printf("[COMBO] assocTypes=%v, isSynergy=%v\n", assocTypes, isSynergy)

	// Calcul de la "Juiciness" (1 à 5)
	juiciness := 1
	if s.comboCount > 2 {
		juiciness = 2
	}
	if s.comboCount > 4 {
		juiciness = 3
	}
	if s.comboCount > 7 {
		juiciness = 4
	}
	if s.comboCount > 10 {
		juiciness = 5
	}
	if isSynergy {
		juiciness++
		if juiciness > 5 {
			juiciness = 5
		}
	}

	// Message juicy
	msg := s.getJuicyMessage(s.comboCount, isSynergy)

	// Calcul du score bonus
	scoreBonus := s.comboCount * 5
	if isSynergy {
		scoreBonus += 50
	}

	s.world.Player.GainExperience(scoreBonus)

	// Publie l'événement de combo
	fmt.Printf("[COMBO] Publishing ComboTriggered: text=%q, count=%d, score=%d, juiciness=%d\n", msg, s.comboCount, scoreBonus, juiciness)
	s.world.EventBus.PublishImmediate(event.NewComboTriggeredEvent(msg, s.comboCount, scoreBonus, juiciness))
}

func (s *ComboSystem) onMerge(e event.Event) {
	s.matchThisTurn = true
	s.lastMatchTurn = s.world.Turn

	// Une fusion ne reset pas le combo, et peut même l'incrémenter légèrement
	msg := "MERGE!"
	s.world.EventBus.PublishImmediate(event.NewComboTriggeredEvent(msg, s.comboCount, 10, 2))
}

func (s *ComboSystem) Reset() {
	if s.comboCount > 0 {
		fmt.Printf("[COMBO] Combo brisé à %d\n", s.comboCount)
	}
	s.comboCount = 0
}

func (s *ComboSystem) getJuicyMessage(count int, isSynergy bool) string {
	if isSynergy {
		return "SYNERGY!"
	}

	switch count {
	case 1:
		return "GOOD!"
	case 2:
		return "NICE!"
	case 3:
		return "GREAT!"
	case 4:
		return "SUPER!"
	case 5:
		return "AWESOME!"
	case 6:
		return "EXCELLENT!"
	case 7:
		return "MARVELOUS!"
	case 8:
		return "INCREDIBLE!"
	case 9:
		return "UNSTOPPABLE!"
	default:
		return "GODLIKE!!!"
	}
}
