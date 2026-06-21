package system

import (
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// CreatureAttackEffectSystem gère les conséquences logiques mondiales des attaques de créatures.
type CreatureAttackEffectSystem struct {
	world *World
}

func NewCreatureAttackEffectSystem(world *World) *CreatureAttackEffectSystem {
	s := &CreatureAttackEffectSystem{world: world}
	world.EventBus.SubscribeFunc(event.CreatureAttacked, s.onCreatureAttacked)
	return s
}

func (s *CreatureAttackEffectSystem) Priority() int { return 10 } // S'exécute après les calculs d'agressivité

func (s *CreatureAttackEffectSystem) Update(world *World) {
	// Pas de logique de mise à jour par frame nécessaire pour l'instant
}

func (s *CreatureAttackEffectSystem) onCreatureAttacked(e event.Event) {
	entID := entity.ID(e.SourceID)
	ent, ok := s.world.Entities.Get(entID)
	if !ok || ent.GetType() != entity.TypeCreature {
		return
	}

	c := ent.(*creature.Creature)

	// Logique d'effets spécifiques par espèce
	switch c.Species {
	case "stonewarden":
		fmt.Println("[STONEWARDEN] L'attaque provoque un séisme de rotation !")
		s.world.RotateGrid(c.GetGridID())

	case "shadowstalker":
		// Les effets de shader sont gérés par le AggressionSystem (immédiat)
		// et le Renderer (visuel). On pourra ajouter ici des effets de statut
		// comme la réduction de sanité maximale ou le vol d'objets.

	case "lumifly":
		// Effets logiques futurs pour le Lumifly (ex: étourdissement, mana drain)
	}
}
