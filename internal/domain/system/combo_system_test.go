package system

import (
	"testing"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

func TestComboSystem_onMatch(t *testing.T) {
	world := NewWorld()
	comboSys := NewComboSystem(world)

	// Simule un match
	ev := event.Event{
		Type: event.TileMatched,
		Payload: map[string]interface{}{
			"assoc_types": []string{"identical"},
		},
	}

	// On vérifie qu'un événement ComboTriggered est publié
	var triggered bool
	world.EventBus.SubscribeFunc(event.ComboTriggered, func(e event.Event) {
		triggered = true
		if e.Payload["count"].(int) != 1 {
			t.Errorf("Attendu combo count 1, eu %d", e.Payload["count"].(int))
		}
		if e.Payload["text"].(string) != "GOOD!" {
			t.Errorf("Attendu 'GOOD!', eu '%s'", e.Payload["text"].(string))
		}
	})

	comboSys.onMatch(ev)
	world.EventBus.ProcessQueue()

	if !triggered {
		t.Error("Événement ComboTriggered non publié")
	}
}

func TestComboSystem_Synergy(t *testing.T) {
	world := NewWorld()
	comboSys := NewComboSystem(world)

	// Simule un match synergy (multiple types)
	ev := event.Event{
		Type: event.TileMatched,
		Payload: map[string]interface{}{
			"assoc_types": []string{"identical", "elemental"},
		},
	}

	world.EventBus.SubscribeFunc(event.ComboTriggered, func(e event.Event) {
		if e.Payload["text"].(string) != "SYNERGY!" {
			t.Errorf("Attendu 'SYNERGY!', eu '%s'", e.Payload["text"].(string))
		}
		if e.Payload["juiciness"].(int) < 2 {
			t.Errorf("Synergie devrait avoir une juiciness élevée, eu %d", e.Payload["juiciness"].(int))
		}
	})

	comboSys.onMatch(ev)
	world.EventBus.ProcessQueue()
}

func TestComboSystem_Reset(t *testing.T) {
	world := NewWorld()
	comboSys := NewComboSystem(world)

	// Monte un combo
	comboSys.onMatch(event.Event{Type: event.TileMatched})
	if comboSys.comboCount != 1 {
		t.Fatal("Combo devrait être à 1")
	}

	// Simule des dégâts par match invalide
	world.EventBus.Publish(event.NewPlayerDamagedEvent("system", 10, "creature_fail", "invalid_match"))
	world.EventBus.ProcessQueue()

	if comboSys.comboCount != 0 {
		t.Errorf("Le combo devrait être reset à 0 après une erreur, eu %d", comboSys.comboCount)
	}
}

func TestComboSystem_TurnTimeout(t *testing.T) {
	world := NewWorld()
	comboSys := NewComboSystem(world)

	// Monte un combo
	comboSys.onMatch(event.Event{Type: event.TileMatched}) // Turn 0, matchThisTurn = true

	// Simule la fin du tour 0
	comboSys.Update(world) // matchThisTurn -> false
	world.Turn++           // Turn -> 1

	// Passe au tour suivant sans match (Update du tour 1)
	comboSys.Update(world) // Reset car world.Turn(1) > lastMatchTurn(0) ET !matchThisTurn

	if comboSys.comboCount != 0 {
		t.Errorf("Le combo devrait être reset si aucun match n'est fait pendant le tour, eu %d", comboSys.comboCount)
	}
}
