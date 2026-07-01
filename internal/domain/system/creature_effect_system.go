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

	// SI L'ATTAQUE TOUCHE LE JOUEUR ALORS EN PLUS DES DEGATS IL Y A UN EFFET NEGATIF !
	// SI L'ATTAQUE NE TOUCHE PAS, LE JOUEUR N'A RIEN !
	hitTarget, hasHit := e.Payload["hit_target"].(*entity.Position)
	if !hasHit || hitTarget == nil {
		return
	}

	// Logique d'effets spécifiques par espèce
	switch c.Species {
	case "stonewarden":
		fmt.Println("[STONEWARDEN] L'attaque touche et provoque un séisme de rotation !")
		s.world.RotateGrid(c.GetGridID())
		s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Vous êtes déboussolé."))

	case "shadowstalker":
		if s.world.Debug.DisabledEffects["blur"] {
			fmt.Printf("[DEBUG] Effet blur bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.VisualEffects["blur"] = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Votre vision se trouble..."))
		}

	case "echo_hound":
		if s.world.Debug.DisabledEffects["aphasia"] {
			fmt.Printf("[DEBUG] Effet aphasia bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.AphasiaTurns = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Les mots perdent leur sens..."))
		}

	case "burrower":
		if s.world.Debug.DisabledEffects["ataxia"] {
			fmt.Printf("[DEBUG] Effet ataxia bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.AtaxiaTurns = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Vos mains ne vous obéissent plus !"))
		}

	case "moss_monkey":
		if s.world.Debug.DisabledEffects["agnosia"] {
			fmt.Printf("[DEBUG] Effet agnosia bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.AgnosiaTurns = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Une mousse envahissante vous aveugle !"))
		}

	case "specter":
		if s.world.Debug.DisabledEffects["amnesia"] {
			fmt.Printf("[DEBUG] Effet amnesia bloqué (désactivé via F12) — %s\n", c.Species)
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
		if s.world.Debug.DisabledEffects["bubble"] {
			fmt.Printf("[DEBUG] Effet bubble bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.VisualEffects["bubble"] = 2
		}

	case "fleeing_sprite":
		if s.world.Debug.DisabledEffects["vertige"] {
			fmt.Printf("[DEBUG] Effet vertige bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.VertigoTurns = 3
			s.world.Player.VisualEffects["vertige"] = 3
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Le monde tourne autour de vous..."))
		}

	case "flutterwing":
		if s.world.Debug.DisabledEffects["invert"] {
			fmt.Printf("[DEBUG] Effet invert bloqué (désactivé via F12) — %s\n", c.Species)
			return
		}
		if s.world.Player != nil {
			s.world.Player.VisualEffects["invert"] = 2
			s.world.EventBus.PublishImmediate(event.NewItemMessageEvent("Les couleurs s'inversent autour de vous..."))
		}
	}
}
