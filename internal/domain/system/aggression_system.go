package system

import (
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// AggressionSystem gère le calcul modulaire de l'agressivité des créatures
type AggressionSystem struct {
	world *World
}

func NewAggressionSystem(world *World) *AggressionSystem {
	s := &AggressionSystem{world: world}
	world.EventBus.SubscribeFunc(event.TileRevealed, s.onTileRevealed)
	world.EventBus.SubscribeFunc(event.CreatureMoved, s.onCreatureMoved)
	world.EventBus.SubscribeFunc(event.AnimationEnded, s.onAnimationEnded)
	return s
}

func (s *AggressionSystem) Priority() int { return 1 } // Doit passer avant le mouvement

func (s *AggressionSystem) Update(world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok {
			continue
		}

		s.updateAggressionFactors(c)
		s.calculateTotalAggression(c)
	}
}

func (s *AggressionSystem) onCreatureMoved(e event.Event) {
	entID := entity.ID(e.SourceID)
	ent, ok := s.world.Entities.Get(entID)
	if !ok || ent.GetType() != entity.TypeCreature {
		return
	}

	c := ent.(*creature.Creature)

	// On ne fait pas monter la patience du Singe Mousse via le mouvement
	if c.Species == "moss_monkey" {
		return
	}

	// Récupère les infos du mouvement
	mode, _ := e.Payload["mode"].(string)

	if c.Behavior.AggressionFactors == nil {
		c.Behavior.AggressionFactors = make(map[string]int)
	}

	// 1. Incrément de patience de base (mouvement normal)
	patience := c.Behavior.AggressionFactors["patience"]
	patience += 2 // +2% par mouvement

	// 2. Bonus si rebond (collision avec un bord ou obstacle)
	if mode == "bounce" {
		patience += 10 // +10% supplémentaire si la créature s'impatiente contre un mur
		fmt.Printf("[AGGRESSION] %s s'impatiente contre un bord (+10%%)\n", c.Species)
	}

	c.Behavior.AggressionFactors["patience"] = patience
}

func (s *AggressionSystem) onTileRevealed(e event.Event) {
	reason, _ := e.Payload["reason"].(string)
	if reason != "player_action" {
		return
	}

	entID := entity.ID(e.SourceID)
	ent, ok := s.world.Entities.Get(entID)
	if !ok || ent.GetType() != entity.TypeCreature {
		return
	}

	c := ent.(*creature.Creature)

	// Le Singe Mousse ignore complètement les clics (révélations)
	if c.Species == "moss_monkey" {
		return
	}

	c.Behavior.RevealCount++

	settings := s.world.Difficulty
	if s.world.Debug.OverrideDifficulty {
		settings = s.world.Debug.Difficulty
	}

	// Calcul du facteur de révélation
	revealAgg := 0
	if settings.MaxSafeReveals >= 0 {
		// Formule : chaque clic vaut 100 / (MaxSafeReveals + 1)
		// On utilise une petite marge pour éviter les problèmes d'arrondi (ex: 99% au lieu de 100%)
		increment := 100.0 / float64(settings.MaxSafeReveals+1)
		revealAgg = int(float64(c.Behavior.RevealCount)*increment + 0.5)
	}

	if c.Behavior.AggressionFactors == nil {
		c.Behavior.AggressionFactors = make(map[string]int)
	}
	c.Behavior.AggressionFactors["reveals"] = revealAgg

	fmt.Printf("[AGGRESSION] %s révélée (%d fois). Facteur révélation: %d%%\n", c.Species, c.Behavior.RevealCount, revealAgg)

	// Recalcul immédiat pour déclencher l'attaque si nécessaire
	s.calculateTotalAggression(c)
}

func (s *AggressionSystem) updateAggressionFactors(c *creature.Creature) {
	if c.Behavior.AggressionFactors == nil {
		c.Behavior.AggressionFactors = make(map[string]int)
	}

	// 1. Facteur Inventaire
	inventoryAgg := 0
	if s.world.Player != nil {
		for _, item := range s.world.Player.Inventory.Items {
			// Exemple : porter une "dreamberry" énerve les créatures qui en mangent (comme le lumifly ?)
			// Pour l'instant, logique générique de "peur/haine" basée sur les tags
			if item.HasTag(c.Species + "_trophy") {
				inventoryAgg += 50
			}
		}
	}
	c.Behavior.AggressionFactors["inventory"] = inventoryAgg

	// 2. Facteur Grille (Species Anger)
	speciesAnger := 0
	revealedSameSpecies := 0
	for _, e := range s.world.Entities.GetAllActive() {
		if other, ok := e.(*creature.Creature); ok && other.Species == c.Species {
			if other.GetID() != c.GetID() && other.GetState()&entity.Revealed != 0 {
				revealedSameSpecies++
			}
		}
	}
	if revealedSameSpecies > 0 {
		speciesAnger = revealedSameSpecies * 20
	}
	c.Behavior.AggressionFactors["species_anger"] = speciesAnger

	// 3. Facteurs Spécifiques par Espèce
	switch c.Species {
	case "moss_monkey":
		s.updateMossMonkeyFactors(c)
	case "lumifly":
		s.updateLumiflyFactors(c)
	}
}

func (s *AggressionSystem) updateMossMonkeyFactors(c *creature.Creature) {
	grid, ok := s.world.GetGrid(c.GetGridID())
	if !ok {
		return
	}

	// Calcul des cases vides (logique originale du singe mousse)
	emptyCount := 0
	for _, plot := range grid.Plots {
		if len(plot.EntitiesID) == 0 {
			emptyCount++
		}
	}

	totalPlots := grid.Width * grid.Height
	if totalPlots > 0 {
		agg := (emptyCount * 200) / totalPlots
		if agg > 100 {
			agg = 100
		}
		c.Behavior.AggressionFactors["empty_plots"] = agg
	}
}

func (s *AggressionSystem) updateLumiflyFactors(c *creature.Creature) {
	grid, ok := s.world.GetGrid(c.GetGridID())
	if !ok {
		return
	}

	pos := c.GetPosition()
	dreamberryThreat := 0

	// Scan des 8 cases environnantes
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}

			checkPos := board.Position{X: pos.X + dx, Y: pos.Y + dy}
			if !grid.IsValid(checkPos) {
				continue
			}

			plot, _ := grid.Get(checkPos)
			for _, id := range plot.EntitiesID {
				ent, ok := s.world.Entities.Get(entity.ID(id))
				if !ok {
					continue
				}

				// Si on trouve une Dreamberry au Stade 4 (index 3)
				if ent.GetType() == entity.TypeResource && ent.GetMatchID() == "dreamberry" {
					if comp, ok := s.world.Components.Get(id, "lifecycle"); ok {
						if lc, ok := comp.(*component.Lifecycle); ok {
							if lc.CurrentStage >= 3 { // Stade 4 = index 3
								dreamberryThreat = 100
								break
							}
						}
					}
				}
			}
			if dreamberryThreat > 0 {
				break
			}
		}
		if dreamberryThreat > 0 {
			break
		}
	}

	c.Behavior.AggressionFactors["toxic_dreamberry"] = dreamberryThreat
}

func (s *AggressionSystem) calculateTotalAggression(c *creature.Creature) {
	total := c.Behavior.AggressionBase
	for _, val := range c.Behavior.AggressionFactors {
		total += val
	}

	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}

	c.Behavior.Aggression = total
	// NOTE: L'attaque n'est plus déclenchée ici mais dans onAnimationEnded
}

func (s *AggressionSystem) onAnimationEnded(e event.Event) {
	animType, _ := e.Payload["animation_type"].(string)
	if animType != "flip" {
		return
	}

	// On ne réagit qu'aux fins d'animations de RÉVÉLATION (pas fermeture)
	finalState, ok := e.Payload["tile_state"].(entity.TileState)
	if ok && finalState&entity.Revealed == 0 {
		return
	}

	entID := entity.ID(e.SourceID)
	ent, ok := s.world.Entities.Get(entID)
	if !ok || ent.GetType() != entity.TypeCreature {
		return
	}

	c := ent.(*creature.Creature)
	if c.Behavior.Aggression >= 100 {
		s.triggerExceededAttack(c)
	}
}

func (s *AggressionSystem) triggerExceededAttack(c *creature.Creature) {
	if !s.world.IsPlayerOnBoard() {
		return
	}

	fmt.Printf("[AGGRESSION] %s est excédée ! ATTAQUE IMMÉDIATE.\n", c.Species)

	// --- Logique de Confrontation Périphérique ---
	// Le joueur est sur un bord d'une case.
	playerPos := s.world.GetPlayerPosition()
	creaturePos := c.GetPosition()
	anchor := s.world.Player.GetAnchor()

	isThreateningPlayer := false

	// Cas 1 : Le joueur est sur la MÊME case que la créature (interaction directe)
	if playerPos.X == creaturePos.X && playerPos.Y == creaturePos.Y {
		outwardDir := anchor.GetOutwardDirection()
		for _, threat := range c.GetActiveThreatDirections() {
			if threat == outwardDir {
				isThreateningPlayer = true
				break
			}
		}
	} else {
		// Cas 2 : Le joueur est sur une case ADJACENTE
		// On utilise la logique standard de menace par position
		isThreateningPlayer = c.IsPositionThreatened(playerPos)
	}

	// --- 1. Notification Visuelle (Lunge) ---
	// Toujours publiée pour que la créature s'anime, même si le joueur n'est pas touché
	var hitTarget *entity.Position
	if isThreateningPlayer {
		target := playerPos
		hitTarget = &target
	}

	s.world.EventBus.Publish(event.Event{
		Type:     event.CreatureAttacked,
		SourceID: string(c.GetID()),
		Payload: map[string]interface{}{
			"hit_target": hitTarget,
		},
	})

	// --- 2. Application des Dégâts (Logique) ---
	if s.world.Player != nil && isThreateningPlayer {
		damage := 10
		if s.world.Player.GraceTurns > 0 {
			fmt.Printf("[AGGRESSION] Grâce active : l'attaque de %s est évitée !\n", c.Species)
			return
		}

		s.world.Player.TakeDamage(damage, "physical")

		// Déclenche les effets visuels (Shaders) selon l'espèce
		if c.Species == "shadowstalker" {
			s.world.Player.VisualEffects["blur"] = 3
		} else if c.Species == "lumifly" {
			s.world.Player.VisualEffects["bubble"] = 3
		}

		// Publie l'événement de dégâts pour les retours HUD/Console
		s.world.EventBus.Publish(event.NewPlayerDamagedEvent(
			string(c.GetID()),
			damage,
			"physical",
			"confrontation",
			map[string]interface{}{"position": playerPos},
		))
	}
}
