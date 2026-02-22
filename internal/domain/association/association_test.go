package association

import (
	"testing"
)

// MockMatchable for testing
type MockMatchable struct {
	matchID      string
	logicKey     string
	element      string
	narrativeTag string
	matchTypes   []string
}

func (m *MockMatchable) GetMatchID() string       { return m.matchID }
func (m *MockMatchable) GetLogicKey() string      { return m.logicKey }
func (m *MockMatchable) GetElement() string       { return m.element }
func (m *MockMatchable) GetNarrativeTag() string  { return m.narrativeTag }
func (m *MockMatchable) GetMatchTypes() []string { return m.matchTypes }

func TestIdenticalStrategy(t *testing.T) {
	strategy := &IdenticalStrategy{}
	
	// Same match ID - should associate
	a := &MockMatchable{matchID: "card1"}
	b := &MockMatchable{matchID: "card1"}
	
	if !strategy.CanAssociate(a, b) {
		t.Error("Should associate identical items")
	}
	
	result, err := strategy.Resolve(a, b)
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
	if !result.Success {
		t.Error("Result should be successful")
	}
	
	// Different match IDs - should not associate
	c := &MockMatchable{matchID: "card2"}
	if strategy.CanAssociate(a, c) {
		t.Error("Should not associate different items")
	}
	
	// Empty match ID - should not associate
	d := &MockMatchable{matchID: ""}
	e := &MockMatchable{matchID: ""}
	if strategy.CanAssociate(d, e) {
		t.Error("Should not associate empty IDs")
	}
}

func TestLogicalStrategy(t *testing.T) {
	strategy := NewLogicalStrategy()
	
	// Key + Lock should associate
	key := &MockMatchable{logicKey: "key"}
	lock := &MockMatchable{logicKey: "lock"}
	
	if !strategy.CanAssociate(key, lock) {
		t.Error("Key and lock should associate")
	}
	
	result, err := strategy.Resolve(key, lock)
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
	if !result.Success {
		t.Error("Result should be successful")
	}
	
	// Reverse should also work
	if !strategy.CanAssociate(lock, key) {
		t.Error("Lock and key should associate (reverse)")
	}
	
	// Invalid combination
	hammer := &MockMatchable{logicKey: "hammer"}
	if strategy.CanAssociate(key, hammer) {
		t.Error("Key and hammer should not associate")
	}
}

func TestElementalStrategy(t *testing.T) {
	strategy := NewElementalStrategy()
	
	// Fire + Wood should associate (fire consumes wood)
	fire := &MockMatchable{element: "fire"}
	wood := &MockMatchable{element: "wood"}
	
	if !strategy.CanAssociate(fire, wood) {
		t.Error("Fire and wood should associate")
	}
	
	// Water + Fire should associate (water extinguishes fire)
	water := &MockMatchable{element: "water"}
	if !strategy.CanAssociate(water, fire) {
		t.Error("Water and fire should associate")
	}
	
	// Same element - should not associate
	fire2 := &MockMatchable{element: "fire"}
	if strategy.CanAssociate(fire, fire2) {
		t.Error("Same elements should not associate")
	}
	
	// Incompatible elements
	earth := &MockMatchable{element: "earth"}
	if strategy.CanAssociate(fire, earth) {
		t.Error("Fire and earth should not associate (no direct affinity)")
	}
	
	// Empty elements
	empty := &MockMatchable{element: ""}
	if strategy.CanAssociate(fire, empty) {
		t.Error("Empty element should not associate")
	}
}

func TestNarrativeStrategy(t *testing.T) {
	strategy := NewNarrativeStrategy()
	
	// Same story elements should associate
	spear := &MockMatchable{narrativeTag: "spear"}
	moon := &MockMatchable{narrativeTag: "moon"}
	
	if !strategy.CanAssociate(spear, moon) {
		t.Error("Spear and moon should associate (first_hunt story)")
	}
	
	result, err := strategy.Resolve(spear, moon)
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
	if !result.Success {
		t.Error("Result should be successful")
	}
	
	// Different stories should not associate
	herb := &MockMatchable{narrativeTag: "herb"} // healing story
	if strategy.CanAssociate(spear, herb) {
		t.Error("Different story elements should not associate")
	}
	
	// Same tag should not associate with itself
	if strategy.CanAssociate(spear, spear) {
		t.Error("Same tag should not associate")
	}
	
	// Empty tags
	empty := &MockMatchable{narrativeTag: ""}
	if strategy.CanAssociate(spear, empty) {
		t.Error("Empty tag should not associate")
	}
}

func TestAssocEngine(t *testing.T) {
	engine := NewEngine()
	
	// Test identical association
	card1 := &MockMatchable{matchID: "dragon", matchTypes: []string{"identical"}}
	card2 := &MockMatchable{matchID: "dragon", matchTypes: []string{"identical"}}
	
	result, err := engine.TryAssociate(card1, card2)
	if err != nil {
		t.Errorf("Association failed: %v", err)
	}
	if !result.Success {
		t.Error("Should successfully associate identical cards")
	}
	if result.Type != Identical {
		t.Error("Should be Identical association")
	}
}

func TestAssocEngineNoMatch(t *testing.T) {
	engine := NewEngine()
	
	// Completely different items
	a := &MockMatchable{matchID: "a", logicKey: "x", element: "unknown"}
	b := &MockMatchable{matchID: "b", logicKey: "y", element: "other"}
	
	result, err := engine.TryAssociate(a, b)
	if err == nil {
		t.Error("Should return error for incompatible items")
	}
	if result.Success {
		t.Error("Should not succeed for incompatible items")
	}
}

func TestAssocEnginePriority(t *testing.T) {
	engine := NewEngine()
	
	// Create custom strategy that always succeeds
	customStrategy := &alwaysMatchStrategy{}
	engine.RegisterStrategy(customStrategy)
	
	a := &MockMatchable{}
	b := &MockMatchable{}
	
	result, _ := engine.TryAssociate(a, b)
	
	// Should use the custom strategy (registered first = higher priority)
	if result.Type != 999 {
		t.Error("Should use custom strategy")
	}
}

type alwaysMatchStrategy struct{}

func (s *alwaysMatchStrategy) Type() Type { return Type(999) }
func (s *alwaysMatchStrategy) CanAssociate(a, b Matchable) bool { return true }
func (s *alwaysMatchStrategy) Resolve(a, b Matchable) (Result, error) {
	return Result{Success: true, Type: Type(999)}, nil
}

func TestGetStrategies(t *testing.T) {
	engine := NewEngine()
	strategies := engine.GetStrategies()
	
	// Should have 4 default strategies
	if len(strategies) != 4 {
		t.Errorf("Expected 4 strategies, got %d", len(strategies))
	}
}
