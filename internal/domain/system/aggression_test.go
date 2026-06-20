package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
)

func TestAggressionSystem_Reveal(t *testing.T) {
	w := NewWorld()
	w.Difficulty = meta.GetSettings(meta.LevelNormal) // MaxSafeReveals: 2
	w.CreateGrid("test", 5, 5, board.BiomeForest)
	w.CurrentGridID = "test"

	// On s'assure que l'AggressionSystem est bien présent
	NewAggressionSystem(w)

	c, _ := w.SpawnCreature("test", "lumifly", entity.Position{X: 2, Y: 2})
	c.SetOrientation(entity.DirNorth) // Menace le Nord (2, 1)

	// On place le joueur au Nord de la créature
	w.SetPlayerPosition(entity.Position{X: 2, Y: 1})
	initialHP := w.Player.Stats.Health

	// 1ère révélation (player_action)
	ev1 := event.Event{
		Type:     event.TileRevealed,
		SourceID: string(c.GetID()),
		Payload: map[string]interface{}{
			"position": c.GetPosition(),
			"reason":   "player_action",
		},
	}
	w.EventBus.PublishImmediate(ev1)

	if c.Behavior.RevealCount != 1 {
		t.Errorf("RevealCount attendu 1, eu %d", c.Behavior.RevealCount)
	}

	// Aggression attendue : 1 * (100 / (2+1)) = 33
	if c.Behavior.Aggression < 30 || c.Behavior.Aggression > 35 {
		t.Errorf("Aggression attendue autour de 33, eu %d", c.Behavior.Aggression)
	}

	// 2ème révélation
	w.EventBus.PublishImmediate(ev1)
	if c.Behavior.Aggression < 60 || c.Behavior.Aggression > 70 {
		t.Errorf("Aggression attendue autour de 66, eu %d", c.Behavior.Aggression)
	}

	// 3ème révélation -> Dépassement !
	w.EventBus.PublishImmediate(ev1)
	if c.Behavior.Aggression != 100 {
		t.Errorf("Aggression attendue 100, eu %d", c.Behavior.Aggression)
	}

	// Simulation de la fin de l'animation de flip
	w.EventBus.PublishImmediate(event.Event{
		Type:     event.AnimationEnded,
		SourceID: string(c.GetID()),
		Payload: map[string]interface{}{
			"animation_type": "flip",
			"tile_state":     entity.Revealed,
		},
	})

	if w.Player.Stats.Health >= initialHP {
		t.Errorf("Le joueur aurait dû prendre des dégâts après 3 révélations en Normal")
	}
}

func TestAggressionSystem_Factors(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 5, 5, board.BiomeForest)
	sys := NewAggressionSystem(w)

	c, _ := w.SpawnCreature("test", "lumifly", entity.Position{X: 2, Y: 2})

	// Test du facteur Species Anger
	// On ajoute une autre créature de la même espèce et on la révèle
	c2, _ := w.SpawnCreature("test", "lumifly", entity.Position{X: 0, Y: 0})
	c2.SetState(entity.Revealed)

	sys.Update(w)

	anger, ok := c.Behavior.AggressionFactors["species_anger"]
	if !ok || anger == 0 {
		t.Errorf("Facteur species_anger manquant ou nul alors qu'un congénère est révélé")
	}

	if c.Behavior.Aggression != anger {
		t.Errorf("Aggression totale devrait être égale à species_anger (%d), eu %d", anger, c.Behavior.Aggression)
	}
}

func TestAggressionSystem_InsaneDifficulty(t *testing.T) {
	w := NewWorld()
	w.Difficulty = meta.GetSettings(meta.LevelInsane) // MaxSafeReveals: 0
	w.CreateGrid("test", 5, 5, board.BiomeForest)
	w.CurrentGridID = "test"
	NewAggressionSystem(w)

	c, _ := w.SpawnCreature("test", "lumifly", entity.Position{X: 2, Y: 2})
	c.SetOrientation(entity.DirNorth)
	w.SetPlayerPosition(entity.Position{X: 2, Y: 1})
	initialHP := w.Player.Stats.Health

	// 1ère révélation en Folie -> Attaque immédiate
	ev := event.Event{
		Type:     event.TileRevealed,
		SourceID: string(c.GetID()),
		Payload: map[string]interface{}{
			"position": c.GetPosition(),
			"reason":   "player_action",
		},
	}
	w.EventBus.PublishImmediate(ev)

	if c.Behavior.Aggression != 100 {
		t.Errorf("En Folie, l'agression devrait être de 100 dès le 1er clic, eu %d", c.Behavior.Aggression)
	}

	// Simulation fin animation
	w.EventBus.PublishImmediate(event.Event{
		Type:     event.AnimationEnded,
		SourceID: string(c.GetID()),
		Payload: map[string]interface{}{
			"animation_type": "flip",
			"tile_state":     entity.Revealed,
		},
	})

	if w.Player.Stats.Health >= initialHP {
		t.Errorf("En Folie, le joueur aurait dû prendre des dégâts dès la 1ère révélation")
	}
}
