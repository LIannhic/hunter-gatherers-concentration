package creature

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestNewCreature(t *testing.T) {
	c := New("test_creature", entity.Position{X: 1, Y: 2})
	
	if c.Species != "test_creature" {
		t.Errorf("Expected species 'test_creature', got '%s'", c.Species)
	}
	
	if c.GetType() != entity.TypeCreature {
		t.Error("Expected TypeCreature")
	}
	
	pos := c.GetPosition()
	if pos.X != 1 || pos.Y != 2 {
		t.Errorf("Expected position (1,2), got (%d,%d)", pos.X, pos.Y)
	}
	
	if !c.HasTag("creature") {
		t.Error("Creature should have 'creature' tag")
	}
	
	if !c.HasTag("test_creature") {
		t.Error("Creature should have species tag")
	}
}

func TestCreatureSetBehavior(t *testing.T) {
	c := New("test", entity.Position{X: 0, Y: 0})
	
	behavior := component.Behavior{
		State:       "hunting",
		Aggression:  50,
		Territorial: true,
	}
	
	c.SetBehavior(behavior)
	
	if c.Behavior.State != "hunting" {
		t.Errorf("Expected state 'hunting', got '%s'", c.Behavior.State)
	}
}

func TestCreatureSetMobility(t *testing.T) {
	c := New("test", entity.Position{X: 0, Y: 0})
	
	mobility := component.Mobility{
		CanMove:     true,
		MovePattern: "random",
		Speed:       2,
	}
	
	c.SetMobility(mobility)
	
	if !c.Mobility.CanMove {
		t.Error("Creature should be able to move")
	}
	
	if c.Mobility.Speed != 2 {
		t.Errorf("Expected speed 2, got %d", c.Mobility.Speed)
	}
}

// Mock WorldState for testing
type mockWorldState struct {
	playerPos      entity.Position
	validMoves     map[entity.Position]bool
}

func (m *mockWorldState) GetPlayerPosition() entity.Position {
	return m.playerPos
}

func (m *mockWorldState) GetNearbyCreatures(pos entity.Position, radius int) []*Creature {
	return nil
}

func (m *mockWorldState) GetResources(pos entity.Position, radius int) []string {
	return nil
}

func (m *mockWorldState) IsValidMove(pos entity.Position) bool {
	if m.validMoves == nil {
		return true
	}
	return m.validMoves[pos]
}

func (m *mockWorldState) GetTileState(pos entity.Position) string {
	return "hidden"
}

func TestSimpleAIDecide(t *testing.T) {
	ai := &SimpleAI{}
	world := &mockWorldState{
		playerPos: entity.Position{X: 5, Y: 5},
		validMoves: map[entity.Position]bool{
			{X: 1, Y: 0}: true,
			{X: 0, Y: 1}: true,
			{X: 0, Y: 0}: true,
		},
	}
	
	// Test idle creature
	c := New("test", entity.Position{X: 0, Y: 0})
	c.SetMobility(component.Mobility{CanMove: true})
	c.SetBehavior(component.Behavior{State: "idle"})
	
	action := ai.Decide(c, world)
	
	// Idle creatures should move randomly
	if action.Type != "move" && action.Type != "idle" {
		t.Errorf("Expected 'move' or 'idle', got '%s'", action.Type)
	}
	
	// Test hunting creature
	c.SetBehavior(component.Behavior{State: "hunting"})
	action = ai.Decide(c, world)
	
	if action.Type != "move" && action.Type != "idle" {
		t.Errorf("Hunting creature should move or stay idle, got '%s'", action.Type)
	}
	
	// Test fleeing creature
	c.SetBehavior(component.Behavior{State: "fleeing"})
	c.SetPosition(entity.Position{X: 4, Y: 4}) // Close to player
	action = ai.Decide(c, world)
	
	if action.Type != "move" {
		t.Errorf("Fleeing creature should move, got '%s'", action.Type)
	}
	
	// Test immobile creature
	c.SetMobility(component.Mobility{CanMove: false})
	c.SetBehavior(component.Behavior{State: "idle"})
	action = ai.Decide(c, world)
	
	if action.Type != "idle" {
		t.Errorf("Immobile creature should stay idle, got '%s'", action.Type)
	}
}

func TestCreatureFactory(t *testing.T) {
	factory := NewFactory()
	
	tests := []struct {
		species string
		shouldWork bool
	}{
		{"lumifly", true},
		{"shadowstalker", true},
		{"burrower", true},
		{"unknown_creature", false},
	}
	
	for _, tc := range tests {
		c, err := factory.Create(tc.species, entity.Position{X: 0, Y: 0})
		
		if tc.shouldWork {
			if err != nil {
				t.Errorf("Failed to create %s: %v", tc.species, err)
				continue
			}
			if c == nil {
				t.Errorf("Created nil creature for %s", tc.species)
				continue
			}
			if c.Species != tc.species {
				t.Errorf("Expected species %s, got %s", tc.species, c.Species)
			}
		} else {
			if err == nil {
				t.Errorf("Expected error for unknown species %s", tc.species)
			}
		}
	}
}

func TestLumiflyProperties(t *testing.T) {
	factory := NewFactory()
	c, _ := factory.Create("lumifly", entity.Position{X: 0, Y: 0})
	
	if c.Behavior.State != "pollinating" {
		t.Errorf("Lumifly should be pollinating, got %s", c.Behavior.State)
	}
	
	if !c.HasTag("flying") {
		t.Error("Lumifly should have 'flying' tag")
	}
	
	if !c.HasTag("passive") {
		t.Error("Lumifly should have 'passive' tag")
	}
}

func TestShadowstalkerProperties(t *testing.T) {
	factory := NewFactory()
	c, _ := factory.Create("shadowstalker", entity.Position{X: 0, Y: 0})
	
	if c.Behavior.State != "hunting" {
		t.Errorf("Shadowstalker should be hunting, got %s", c.Behavior.State)
	}
	
	if c.Behavior.Aggression != 80 {
		t.Errorf("Shadowstalker should have aggression 80, got %d", c.Behavior.Aggression)
	}
	
	if !c.HasTag("dangerous") {
		t.Error("Shadowstalker should have 'dangerous' tag")
	}
}

func TestBurrowerProperties(t *testing.T) {
	factory := NewFactory()
	c, _ := factory.Create("burrower", entity.Position{X: 0, Y: 0})
	
	if c.Behavior.State != "hiding" {
		t.Errorf("Burrower should be hiding, got %s", c.Behavior.State)
	}
	
	if c.Mobility.MovePattern != "burrow" {
		t.Errorf("Burrower should have 'burrow' pattern, got %s", c.Mobility.MovePattern)
	}
}
