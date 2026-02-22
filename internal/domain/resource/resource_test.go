package resource

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestNewResource(t *testing.T) {
	r := New("test_resource", entity.Position{X: 1, Y: 2})
	
	if r.ResourceType != "test_resource" {
		t.Errorf("Expected type 'test_resource', got '%s'", r.ResourceType)
	}
	
	if r.GetType() != entity.TypeResource {
		t.Error("Expected TypeResource")
	}
	
	pos := r.GetPosition()
	if pos.X != 1 || pos.Y != 2 {
		t.Errorf("Expected position (1,2), got (%d,%d)", pos.X, pos.Y)
	}
	
	if !r.HasTag("resource") {
		t.Error("Resource should have 'resource' tag")
	}
	
	if !r.HasTag("test_resource") {
		t.Error("Resource should have type tag")
	}
}

func TestResourceSetLifecycle(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	
	lifecycle := component.Lifecycle{
		CurrentStage: 0,
		MaxStages:    4,
		StageNames:   []string{"seed", "sprout", "mature", "withered"},
		TurnsToNext:  2,
		CanPropagate: true,
	}
	
	r.SetLifecycle(lifecycle)
	
	if r.Lifecycle.MaxStages != 4 {
		t.Errorf("Expected 4 stages, got %d", r.Lifecycle.MaxStages)
	}
}

func TestResourceSetValue(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	
	value := component.Value{
		BaseValue:    100,
		CurrentValue: 50,
		DegradeRate:  5,
	}
	
	r.SetValue(value)
	
	if r.Value.BaseValue != 100 {
		t.Errorf("Expected base value 100, got %d", r.Value.BaseValue)
	}
}

func TestResourceSetMatchable(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	
	matchable := component.Matchable{
		MatchID:    "test_match",
		MatchTypes: []string{"identical", "elemental"},
		Element:    "fire",
	}
	
	r.SetMatchable(matchable)
	
	if r.Matchable.MatchID != "test_match" {
		t.Errorf("Expected match ID 'test_match', got '%s'", r.Matchable.MatchID)
	}
}

func TestResourceUpdate(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	
	r.SetLifecycle(component.Lifecycle{
		CurrentStage: 0,
		MaxStages:    3,
		StageNames:   []string{"young", "mature", "old"},
		TurnsToNext:  1,
		CanPropagate: false,
	})
	
	r.SetValue(component.Value{
		BaseValue:    100,
		CurrentValue: 100,
		DegradeRate:  10,
	})
	
	// First update - should progress stage and degrade
	r.Update()
	
	if r.Value.CurrentValue != 90 { // 100 - 10 degrade
		t.Errorf("Expected value 90 after degrade, got %d", r.Value.CurrentValue)
	}
}

func TestResourceCanPropagate(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	
	// Can't propagate at stage 0
	r.SetLifecycle(component.Lifecycle{
		CurrentStage: 0,
		MaxStages:    4,
		CanPropagate: true,
	})
	
	if r.CanPropagate() {
		t.Error("Should not be able to propagate at stage 0")
	}
	
	// Can propagate at stage 1+
	r.Lifecycle.CurrentStage = 1
	if !r.CanPropagate() {
		t.Error("Should be able to propagate at stage 1")
	}
	
	// Can't propagate if disabled
	r.Lifecycle.CanPropagate = false
	if r.CanPropagate() {
		t.Error("Should not propagate when disabled")
	}
}

func TestResourceIsHarvestable(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	r.SetValue(component.Value{CurrentValue: 10})
	
	if !r.IsHarvestable() {
		t.Error("Resource with value > 0 should be harvestable")
	}
	
	r.Value.CurrentValue = 0
	if r.IsHarvestable() {
		t.Error("Resource with value 0 should not be harvestable")
	}
}

func TestResourceMatchableInterface(t *testing.T) {
	r := New("test", entity.Position{X: 0, Y: 0})
	r.SetMatchable(component.Matchable{
		MatchID:      "match1",
		LogicKey:     "key1",
		Element:      "fire",
		NarrativeTag: "story1",
		MatchTypes:   []string{"identical"},
	})
	
	// Test interface methods
	if r.GetMatchID() != "match1" {
		t.Error("GetMatchID failed")
	}
	
	if r.GetLogicKey() != "key1" {
		t.Error("GetLogicKey failed")
	}
	
	if r.GetElement() != "fire" {
		t.Error("GetElement failed")
	}
	
	if r.GetNarrativeTag() != "story1" {
		t.Error("GetNarrativeTag failed")
	}
	
	if len(r.GetMatchTypes()) != 1 {
		t.Error("GetMatchTypes failed")
	}
}

func TestResourceFactory(t *testing.T) {
	factory := NewFactory()
	
	// Test dreamberry
	dreamberry := factory.Create("dreamberry", entity.Position{X: 0, Y: 0})
	if dreamberry.ResourceType != "dreamberry" {
		t.Error("Failed to create dreamberry")
	}
	if dreamberry.Lifecycle.MaxStages != 4 {
		t.Error("Dreamberry should have 4 stages")
	}
	// Dreamberry starts at stage 0, so it can't propagate yet
	if dreamberry.CanPropagate() {
		t.Error("Dreamberry at stage 0 should not be able to propagate")
	}
	
	// Progress to stage 1
	dreamberry.Lifecycle.CurrentStage = 1
	if !dreamberry.CanPropagate() {
		t.Error("Dreamberry at stage 1 should be able to propagate")
	}
	
	// Test moonstone
	moonstone := factory.Create("moonstone", entity.Position{X: 1, Y: 1})
	if moonstone.ResourceType != "moonstone" {
		t.Error("Failed to create moonstone")
	}
	if moonstone.Lifecycle.TurnsToNext != -1 {
		t.Error("Moonstone should not progress automatically")
	}
	
	// Test whispering_herb
	herb := factory.Create("whispering_herb", entity.Position{X: 2, Y: 2})
	if herb.ResourceType != "whispering_herb" {
		t.Error("Failed to create whispering_herb")
	}
	
	// Test unknown type (should still work with defaults)
	unknown := factory.Create("unknown", entity.Position{X: 3, Y: 3})
	if unknown.ResourceType != "unknown" {
		t.Error("Should create resource even for unknown type")
	}
}

func TestResourceUpdateValueByStage(t *testing.T) {
	factory := NewFactory()
	r := factory.Create("dreamberry", entity.Position{X: 0, Y: 0})
	
	baseValue := r.Value.BaseValue
	
	// bourgeon stage (0)
	r.Lifecycle.CurrentStage = 0
	r.updateValueByStage()
	if r.Value.CurrentValue != baseValue/4 {
		t.Errorf("Expected value %d at bourgeon stage, got %d", baseValue/4, r.Value.CurrentValue)
	}
	
	// fruit stage (2)
	r.Lifecycle.CurrentStage = 2
	r.Value.CurrentValue = 0
	r.updateValueByStage()
	if r.Value.CurrentValue != baseValue {
		t.Errorf("Expected value %d at fruit stage, got %d", baseValue, r.Value.CurrentValue)
	}
}
