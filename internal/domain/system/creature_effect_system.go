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
		hitTarget, hasHit := e.Payload["hit_target"].(*entity.Position)
		if !hasHit || hitTarget == nil {
			return
		}
		fmt.Println("[STONEWARDEN] L'attaque touche et provoque un séisme de rotation !")
		s.world.RotateGrid(c.GetGridID())
		s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Vous êtes déboussolé."))

	case "shadowstalker":
		// Applique un effet visuel de flou
		if s.world.Player != nil {
			s.world.Player.VisualEffects["blur"] = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Votre vision se trouble..."))
		}

	case "echo_hound":
		if s.world.Player != nil {
			s.world.Player.AphasiaTurns = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Les mots perdent leur sens..."))
		}

	case "burrower":
		if s.world.Player != nil {
			s.world.Player.AtaxiaTurns = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Vos mains ne vous obéissent plus !"))
		}

	case "moss_monkey":
		if s.world.Player != nil {
			s.world.Player.AgnosiaTurns = 3
			s.world.Player.VisualEffects["moss_drip"] = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Une mousse envahissante vous aveugle !"))
		}

	case "specter":
		hitTarget, hasHit := e.Payload["hit_target"].(*entity.Position)
		if !hasHit || hitTarget == nil {
			return
		}
		fmt.Printf("[%s] L'attaque touche le joueur et provoque une amnésie temporaire !\n", c.Species)
		flipDir := entity.FlipTop
		if s.world.Player != nil {
			flipDir = s.world.Player.GetAnchor().GetFlipDirection()
		}
		s.world.HideInventory(flipDir)
		s.world.Player.AmnesiaTurns = 5
		s.world.EventBus.PublishImmediate(event.NewAmnesiaStartedEvent(string(c.GetID()), 5))
		s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Vous avez du mal à vous souvenir."))

	case "lumifly":
		if s.world.Player != nil {
			s.world.Player.VisualEffects["bubble"] = 2
		}
	}
}
