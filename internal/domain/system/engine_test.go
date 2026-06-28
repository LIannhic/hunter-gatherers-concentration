package system

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"testing"
)

func TestNewEngine(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	if engine.world != w {
		t.Error("Engine should reference world")
	}

	if len(engine.systems) != 11 {
		t.Errorf("Engine should have 11 systems, got %d", len(engine.systems))
	}
}

func TestEngineUpdate(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)
	engine := NewEngine(w)

	// Add a resource with lifecycle
	_, _ = w.SpawnResource("test", "dreamberry", entity.Position{X: 0, Y: 0})

	initialTurn := w.Turn

	engine.Update()

	if w.Turn != initialTurn+1 {
		t.Errorf("Turn should increase, expected %d, got %d", initialTurn+1, w.Turn)
	}
}

func TestEngineUpdateFrame(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	// Setup TurnTimer
	w.TurnTimer = NewTurnTimer(1.0)
	w.TurnTimer.Start()

	expired := false
	w.EventBus.SubscribeFunc(event.Type("turn_timer_expired"), func(e event.Event) {
		expired = true
	})

	// Update with small dt
	engine.UpdateFrame(0.5)
	if expired {
		t.Error("Timer should not be expired yet")
	}

	// Update to expiration
	engine.UpdateFrame(0.6)
	if !expired {
		t.Error("Timer should have expired and published event")
	}

	if w.TurnTimer.Remaining != w.TurnTimer.MaxTime {
		t.Error("Timer should have reset after expiration")
	}
}

func TestEngineReset(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	// Simulate state in systems
	engine.movementSystem.TrackReveal(board.Position{X: 1, Y: 1}, "test")
	engine.previewSystem.OnEnterGrid(w, "test")
	engine.comboSystem.Reset() // Ensure it's in a known state (though it's already empty)
	engine.comboSystem.CheatIncreaseCombo()

	if len(engine.movementSystem.recentReveals) == 0 {
		t.Error("movementSystem should have state")
	}

	engine.Reset()

	if len(engine.movementSystem.recentReveals) != 0 {
		t.Error("movementSystem state should be cleared after Reset")
	}

	// For PreviewSystem, we check if s.previewed is empty (indirectly via a second call)
	// We'd need to expose more or check behavior.
}
