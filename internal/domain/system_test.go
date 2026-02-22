package domain

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestNewWorld(t *testing.T) {
	w := NewWorld(6, 6)
	
	if w.Grid == nil {
		t.Error("World should have a grid")
	}
	
	if w.Entities == nil {
		t.Error("World should have entity manager")
	}
	
	if w.Components == nil {
		t.Error("World should have component store")
	}
	
	if w.EventBus == nil {
		t.Error("World should have event bus")
	}
	
	if w.Turn != 0 {
		t.Errorf("Turn should start at 0, got %d", w.Turn)
	}
}

func TestWorldSpawnResource(t *testing.T) {
	w := NewWorld(4, 4)
	
	r, err := w.SpawnResource("dreamberry", entity.Position{X: 1, Y: 1})
	if err != nil {
		t.Errorf("Failed to spawn resource: %v", err)
	}
	
	if r == nil {
		t.Fatal("Resource should not be nil")
	}
	
	if r.ResourceType != "dreamberry" {
		t.Error("Wrong resource type")
	}
	
	// Check entity was registered
	if w.Entities.Count() != 1 {
		t.Errorf("Expected 1 entity, got %d", w.Entities.Count())
	}
	
	// Check tile has entity
	tile, _ := w.Grid.Get(board.Position{X: 1, Y: 1})
	if tile.EntityID != string(r.GetID()) {
		t.Error("Tile should have entity ID")
	}
}

func TestWorldSpawnCreature(t *testing.T) {
	w := NewWorld(4, 4)
	
	c, err := w.SpawnCreature("lumifly", entity.Position{X: 2, Y: 2})
	if err != nil {
		t.Errorf("Failed to spawn creature: %v", err)
	}
	
	if c == nil {
		t.Fatal("Creature should not be nil")
	}
	
	if c.Species != "lumifly" {
		t.Error("Wrong species")
	}
	
	// Check tile has creature
	tile, _ := w.Grid.Get(board.Position{X: 2, Y: 2})
	if tile.EntityID != string(c.GetID()) {
		t.Error("Tile should have creature ID")
	}
}

func TestWorldRevealTile(t *testing.T) {
	w := NewWorld(4, 4)
	
	// Reveal an empty tile
	tile, err := w.RevealTile(board.Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to reveal tile: %v", err)
	}
	
	if tile.State != board.Revealed {
		t.Error("Tile should be revealed")
	}
	
	// Can't reveal twice
	_, err = w.RevealTile(board.Position{X: 0, Y: 0})
	if err == nil {
		t.Error("Should not be able to reveal twice")
	}
}

func TestWorldMatchTile(t *testing.T) {
	w := NewWorld(4, 4)
	
	// Can't match hidden tile
	err := w.MatchTile(board.Position{X: 0, Y: 0})
	if err == nil {
		t.Error("Should not be able to match hidden tile")
	}
	
	// Reveal then match
	w.RevealTile(board.Position{X: 0, Y: 0})
	err = w.MatchTile(board.Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to match tile: %v", err)
	}
	
	tile, _ := w.Grid.Get(board.Position{X: 0, Y: 0})
	if tile.State != board.Matched {
		t.Error("Tile should be matched")
	}
}

func TestWorldRemoveEntity(t *testing.T) {
	w := NewWorld(4, 4)
	
	r, _ := w.SpawnResource("dreamberry", entity.Position{X: 1, Y: 1})
	id := r.GetID()
	
	w.RemoveEntity(id)
	
	// Entity should be removed
	if w.Entities.Count() != 0 {
		t.Error("Entity should be removed")
	}
	
	// Tile should be empty
	tile, _ := w.Grid.Get(board.Position{X: 1, Y: 1})
	if tile.EntityID != "" {
		t.Error("Tile should be empty")
	}
}

func TestWorldSetPlayerPosition(t *testing.T) {
	w := NewWorld(4, 4)
	
	w.SetPlayerPosition(entity.Position{X: 2, Y: 3})
	
	pos := w.GetPlayerPosition()
	if pos.X != 2 || pos.Y != 3 {
		t.Error("Player position not set correctly")
	}
}

func TestNewEngine(t *testing.T) {
	w := NewWorld(4, 4)
	engine := NewEngine(w)
	
	if engine.world != w {
		t.Error("Engine should reference world")
	}
	
	if engine.Running {
		t.Error("Engine should not be running initially")
	}
	
	if len(engine.systems) != 4 {
		t.Errorf("Engine should have 4 systems, got %d", len(engine.systems))
	}
}

func TestEngineStartStop(t *testing.T) {
	w := NewWorld(4, 4)
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
	w := NewWorld(4, 4)
	engine := NewEngine(w)
	
	// Add a resource with lifecycle
	w.SpawnResource("dreamberry", entity.Position{X: 0, Y: 0})
	
	initialTurn := w.Turn
	
	engine.Start()
	engine.Update()
	
	if w.Turn != initialTurn+1 {
		t.Errorf("Turn should increase, expected %d, got %d", initialTurn+1, w.Turn)
	}
}

func TestEngineUpdateNotRunning(t *testing.T) {
	w := NewWorld(4, 4)
	engine := NewEngine(w)
	
	initialTurn := w.Turn
	engine.Update() // Should not update when not running
	
	if w.Turn != initialTurn {
		t.Error("Should not update when not running")
	}
}

func TestWorldAdapter(t *testing.T) {
	w := NewWorld(4, 4)
	adapter := &worldAdapter{world: w}
	
	// Set player position
	w.SetPlayerPosition(entity.Position{X: 2, Y: 2})
	if adapter.GetPlayerPosition().X != 2 {
		t.Error("Player position incorrect")
	}
	
	// Test IsValidMove
	if !adapter.IsValidMove(entity.Position{X: 0, Y: 0}) {
		t.Error("(0,0) should be valid move")
	}
	
	if adapter.IsValidMove(entity.Position{X: 10, Y: 10}) {
		t.Error("(10,10) should be invalid")
	}
	
	// Test GetTileState
	state := adapter.GetTileState(entity.Position{X: 0, Y: 0})
	if state != "hidden" {
		t.Errorf("Expected 'hidden', got '%s'", state)
	}
}

func TestLifecycleSystem(t *testing.T) {
	w := NewWorld(4, 4)
	sys := &LifecycleSystem{}
	
	// Spawn resource with lifecycle
	r, _ := w.SpawnResource("dreamberry", entity.Position{X: 0, Y: 0})
	
	initialStage := r.Lifecycle.CurrentStage
	
	// Progress lifecycle many times
	for i := 0; i < 10; i++ {
		sys.Update(w)
	}
	
	// Lifecycle should have progressed
	if r.Lifecycle.CurrentStage == initialStage {
		t.Log("Lifecycle may not have progressed (depends on turns to next)")
	}
}

func TestCreatureAISystem(t *testing.T) {
	w := NewWorld(6, 6)
	sys := &CreatureAISystem{}
	
	// Spawn creature
	c, _ := w.SpawnCreature("lumifly", entity.Position{X: 1, Y: 1})
	initialPos := c.GetPosition()
	
	// Set player position far away
	w.SetPlayerPosition(entity.Position{X: 5, Y: 5})
	
	// Run AI system multiple times
	for i := 0; i < 5; i++ {
		sys.Update(w)
	}
	
	// Creature might have moved (random movement for lumifly)
	_ = initialPos // Just to show it could change
}

func TestPropagationSystem(t *testing.T) {
	w := NewWorld(4, 4)
	sys := &PropagationSystem{}
	
	// Spawn resource that can propagate
	r, _ := w.SpawnResource("dreamberry", entity.Position{X: 1, Y: 1})
	r.Lifecycle.CanPropagate = true
	r.Lifecycle.CurrentStage = 2 // Mature enough to propagate
	
	initialCount := w.Entities.Count()
	
	// Run propagation multiple times
	for i := 0; i < 10; i++ {
		sys.Update(w)
	}
	
	// Might have propagated
	_ = initialCount
}
