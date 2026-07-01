package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestLifecycleCircularDuration(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 5, 5, board.BiomeForest)
	w.CurrentGridID = "test"

	// Create a resource with a circular lifecycle
	// 4 stages (0, 1, 2, 3), 2 turns each.
	r, _ := w.SpawnResource("test", "dreamberry", entity.Position{X: 2, Y: 2})
	lc, ok := w.Components.Get(string(r.GetID()), "lifecycle")
	if !ok {
		t.Fatal("Lifecycle component not found")
	}
	lifecycle := lc.(*component.Lifecycle)
	lifecycle.MaxStages = 4
	lifecycle.StageNames = []string{"0", "1", "2", "3"}
	lifecycle.TurnsToNext = 2
	lifecycle.Cyclic = true
	lifecycle.CurrentStage = 0
	lifecycle.TurnsInStage = 0

	sys := &LifecycleSystem{}

	// Stage 0
	// Turn 1
	sys.Update(w)
	if lifecycle.CurrentStage != 0 || lifecycle.TurnsInStage != 1 {
		t.Errorf("Turn 1: Expected Stage 0, TurnsInStage 1, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Turn 2
	sys.Update(w)
	if lifecycle.CurrentStage != 1 || lifecycle.TurnsInStage != 0 {
		t.Errorf("Turn 2: Expected Stage 1, TurnsInStage 0 (just transitioned), got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Stage 1
	// Turn 3
	sys.Update(w)
	if lifecycle.CurrentStage != 1 || lifecycle.TurnsInStage != 1 {
		t.Errorf("Turn 3: Expected Stage 1, TurnsInStage 1, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Turn 4
	sys.Update(w)
	if lifecycle.CurrentStage != 2 || lifecycle.TurnsInStage != 0 {
		t.Errorf("Turn 4: Expected Stage 2, TurnsInStage 0, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Stage 2
	// Turn 5
	sys.Update(w)
	if lifecycle.CurrentStage != 2 || lifecycle.TurnsInStage != 1 {
		t.Errorf("Turn 5: Expected Stage 2, TurnsInStage 1, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Turn 6
	sys.Update(w)
	if lifecycle.CurrentStage != 3 || lifecycle.TurnsInStage != 0 {
		t.Errorf("Turn 6: Expected Stage 3, TurnsInStage 0, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Stage 3 (Last Stage)
	// Turn 7
	sys.Update(w)
	if lifecycle.CurrentStage != 3 || lifecycle.TurnsInStage != 1 {
		t.Errorf("Turn 7: Expected Stage 3, TurnsInStage 1, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Turn 8
	sys.Update(w)
	// Should cycle back to 0
	if lifecycle.CurrentStage != 0 || lifecycle.TurnsInStage != 0 {
		t.Errorf("Turn 8: Expected Stage 0, TurnsInStage 0 (reset), got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}
}

func TestPropagationAtTargetStage(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 5, 5, board.BiomeForest)
	w.CurrentGridID = "test"

	r, _ := w.SpawnResource("test", "dreamberry", entity.Position{X: 2, Y: 2})
	lc, _ := w.Components.Get(string(r.GetID()), "lifecycle")
	lifecycle := lc.(*component.Lifecycle)
	lifecycle.MaxStages = 4
	lifecycle.TurnsToNext = 2
	lifecycle.CanPropagate = true
	lifecycle.PropagationCount = 1
	lifecycle.MaxPropagations = 1
	lifecycle.PropagationsDone = 0

	lSys := &LifecycleSystem{}
	pSys := &PropagationSystem{}

	// Target: Stage 3 (index 2) - "fruit"
	lifecycle.PropagationStage = 2

	// We need to advance to Stage 2.
	// Turn 1: 0, 1
	// Turn 2: 1, 0
	// Turn 3: 1, 1
	// Turn 4: 2, 0 -> Should propagate here if target is 2.

	for i := 0; i < 3; i++ {
		lSys.Update(w)
		pSys.Update(w)
	}

	if lifecycle.CurrentStage != 1 || lifecycle.TurnsInStage != 1 {
		t.Fatalf("Expected Stage 1, TurnsInStage 1, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	// Turn 4
	lSys.Update(w)
	if lifecycle.CurrentStage != 2 || lifecycle.TurnsInStage != 0 {
		t.Fatalf("Turn 4: Expected Stage 2, TurnsInStage 0, got %d, %d", lifecycle.CurrentStage, lifecycle.TurnsInStage)
	}

	initialEntities := w.Entities.Count()
	pSys.Update(w)

	if w.Entities.Count() <= initialEntities {
		t.Errorf("Should have propagated at stage 2")
	}

	if lifecycle.PropagationsDone != 1 {
		t.Errorf("Expected PropagationsDone = 1, got %d", lifecycle.PropagationsDone)
	}
}


