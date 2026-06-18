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

	if engine.Running {
		t.Error("Engine should not be running initially")
	}

	if len(engine.systems) != 8 {
		t.Errorf("Engine should have 8 systems, got %d", len(engine.systems))
	}
}

func TestEngineStartStop(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	if engine.Running {
		t.Error("Should not be running")
	}

	engine.Start()
	if !engine.Running {
		t.Error("Should be running after Start()")
	}

	engine.Stop()
	if engine.Running {
		t.Error("Should not be running after Stop()")
	}
}

func TestEngineUpdate(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)
	engine := NewEngine(w)

	// Add a resource with lifecycle
	_, _ = w.SpawnResource("test", "dreamberry", entity.Position{X: 0, Y: 0})

	initialTurn := w.Turn

	engine.Start()
	engine.Update()

	if w.Turn != initialTurn+1 {
		t.Errorf("Turn should increase, expected %d, got %d", initialTurn+1, w.Turn)
	}
}

func TestEngineUpdateNotRunning(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	initialTurn := w.Turn
	engine.Update() // Should not update when not running

	if w.Turn != initialTurn {
		t.Error("Should not update when not running")
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
