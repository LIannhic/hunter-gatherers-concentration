package domain

import (
	"testing"
)

// Test that re-exported types work correctly
func TestReexportedTypes(t *testing.T) {
	// Test entity types
	id := NewID()
	if len(id) == 0 {
		t.Error("NewID should return non-empty ID")
	}
	
	// Test grid creation
	grid := NewGrid(4, 4)
	if grid.Width != 4 {
		t.Error("Grid width should be 4")
	}
	
	// Test position
	p1 := Position{X: 1, Y: 2}
	p2 := Position{X: 3, Y: 4}
	result := p1.Add(p2)
	if result.X != 4 || result.Y != 6 {
		t.Error("Position Add failed")
	}
}

func TestIntegrationGameFlow(t *testing.T) {
	// Create a world
	world := NewWorld(6, 6)
	
	// Spawn some resources
	_, err := world.SpawnResource("dreamberry", Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to spawn resource: %v", err)
	}
	
	_, err = world.SpawnResource("moonstone", Position{X: 1, Y: 1})
	if err != nil {
		t.Errorf("Failed to spawn resource: %v", err)
	}
	
	// Spawn a creature
	_, err = world.SpawnCreature("lumifly", Position{X: 2, Y: 2})
	if err != nil {
		t.Errorf("Failed to spawn creature: %v", err)
	}
	
	// Check entity count
	if world.Entities.Count() != 3 {
		t.Errorf("Expected 3 entities, got %d", world.Entities.Count())
	}
	
	// Create and run engine
	engine := NewEngine(world)
	engine.Start()
	
	// Run a few turns
	for i := 0; i < 5; i++ {
		engine.Update()
	}
	
	if world.Turn != 5 {
		t.Errorf("Expected turn 5, got %d", world.Turn)
	}
}

func TestAssociationIntegration(t *testing.T) {
	engine := NewAssocEngine()
	
	// Create two resources with same match ID
	r1 := NewResource("dreamberry", Position{X: 0, Y: 0})
	r1.SetMatchable(Matchable{
		MatchID:    "dreamberry_pair",
		MatchTypes: []string{"identical"},
	})
	
	r2 := NewResource("dreamberry", Position{X: 1, Y: 1})
	r2.SetMatchable(Matchable{
		MatchID:    "dreamberry_pair",
		MatchTypes: []string{"identical"},
	})
	
	// Try to associate
	result, err := engine.TryAssociate(r1, r2)
	if err != nil {
		t.Errorf("Association failed: %v", err)
	}
	
	if !result.Success {
		t.Error("Association should succeed")
	}
	
	if result.Type != AssocType(0) { // Identical = 0
		t.Error("Should be identical association")
	}
}

func TestPlayerAndMetaIntegration(t *testing.T) {
	// Create player
	p := NewPlayer("hero")
	if p.ID != "hero" {
		t.Error("Player ID mismatch")
	}
	
	// Create family
	family := NewFamily()
	if len(family.Members) != 2 {
		t.Error("Family should have 2 members")
	}
	
	// Create hub
	hub := NewHub()
	if hub.Family == nil {
		t.Error("Hub should have family")
	}
	
	// Add to inventory
	hub.AddToInventory("gold", 100)
	if hub.Inventory["gold"] != 100 {
		t.Error("Inventory not updated")
	}
	
	// Prepare mission
	items := hub.PrepareMission([]string{})
	if items["ration"] != 2 {
		t.Error("Should have 2 rations")
	}
}

func TestEventBusIntegration(t *testing.T) {
	bus := NewBus()
	
	received := false
	bus.SubscribeFunc(EventType("creature_moved"), func(e Event) {
		received = true
	})
	
	// Publish event
	bus.Publish(Event{Type: EventType("creature_moved"), SourceID: "test"})
	bus.ProcessQueue()
	
	if !received {
		t.Error("Event handler should have been called")
	}
}

func TestConstants(t *testing.T) {
	// Test that constants are properly re-exported
	if TypeResource != 0 {
		t.Error("TypeResource constant mismatch")
	}
	
	if TypeCreature != 1 {
		t.Error("TypeCreature constant mismatch")
	}
	
	if Hidden != 0 {
		t.Error("Hidden constant mismatch")
	}
	
	if Revealed != 1 {
		t.Error("Revealed constant mismatch")
	}
	
	if North != 0 {
		t.Error("North constant mismatch")
	}
}
