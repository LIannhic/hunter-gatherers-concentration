package component

import (
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Error("NewStore should not return nil")
	}
}

func TestStoreAddAndGet(t *testing.T) {
	s := NewStore()
	
	lifecycle := &Lifecycle{
		CurrentStage: 1,
		MaxStages:    4,
	}
	
	s.Add("entity1", lifecycle)
	
	retrieved, ok := s.Get("entity1", "lifecycle")
	if !ok {
		t.Error("Failed to retrieve component")
	}
	
	retrievedLifecycle, ok := retrieved.(*Lifecycle)
	if !ok {
		t.Error("Retrieved wrong component type")
	}
	
	if retrievedLifecycle.CurrentStage != 1 {
		t.Error("Retrieved component has wrong data")
	}
}

func TestStoreHas(t *testing.T) {
	s := NewStore()
	
	lifecycle := &Lifecycle{}
	s.Add("entity1", lifecycle)
	
	if !s.Has("entity1", "lifecycle") {
		t.Error("Should have lifecycle component")
	}
	
	if s.Has("entity1", "visual") {
		t.Error("Should not have visual component")
	}
	
	if s.Has("entity2", "lifecycle") {
		t.Error("Entity2 should not have any components")
	}
}

func TestStoreRemove(t *testing.T) {
	s := NewStore()
	
	lifecycle := &Lifecycle{}
	s.Add("entity1", lifecycle)
	
	s.Remove("entity1", "lifecycle")
	
	if s.Has("entity1", "lifecycle") {
		t.Error("Component should be removed")
	}
}

func TestStoreGetAll(t *testing.T) {
	s := NewStore()
	
	lifecycle := &Lifecycle{}
	visual := &Visual{}
	
	s.Add("entity1", lifecycle)
	s.Add("entity1", visual)
	
	all := s.GetAll("entity1")
	if len(all) != 2 {
		t.Errorf("Expected 2 components, got %d", len(all))
	}
}

func TestStoreQueryByComponent(t *testing.T) {
	s := NewStore()
	
	s.Add("entity1", &Lifecycle{})
	s.Add("entity2", &Lifecycle{})
	s.Add("entity3", &Visual{})
	
	results := s.QueryByComponent("lifecycle")
	if len(results) != 2 {
		t.Errorf("Expected 2 entities with lifecycle, got %d", len(results))
	}
}

func TestStoreRemoveEntity(t *testing.T) {
	s := NewStore()
	
	s.Add("entity1", &Lifecycle{})
	s.Add("entity1", &Visual{})
	s.Add("entity2", &Lifecycle{})
	
	s.RemoveEntity("entity1")
	
	if s.Has("entity1", "lifecycle") || s.Has("entity1", "visual") {
		t.Error("Entity1 components should be removed")
	}
	
	if !s.Has("entity2", "lifecycle") {
		t.Error("Entity2 components should still exist")
	}
}

func TestLifecycleProgress(t *testing.T) {
	l := &Lifecycle{
		CurrentStage: 0,
		MaxStages:    3,
		StageNames:   []string{"young", "mature", "old"},
		TurnsToNext:  2,
	}
	
	// After 1 turn - no change
	changed := l.Progress()
	if changed {
		t.Error("Should not change after 1 turn")
	}
	if l.CurrentStage != 0 {
		t.Error("Stage should still be 0")
	}
	
	// After 2 turns - should change
	changed = l.Progress()
	if !changed {
		t.Error("Should change after 2 turns")
	}
	if l.CurrentStage != 1 {
		t.Errorf("Stage should be 1, got %d", l.CurrentStage)
	}
}

func TestLifecycleGetCurrentStageName(t *testing.T) {
	l := &Lifecycle{
		CurrentStage: 1,
		StageNames:   []string{"seed", "sprout", "plant"},
	}
	
	name := l.GetCurrentStageName()
	if name != "sprout" {
		t.Errorf("Expected 'sprout', got '%s'", name)
	}
	
	// Test out of bounds
	l.CurrentStage = 10
	name = l.GetCurrentStageName()
	if name != "unknown" {
		t.Errorf("Expected 'unknown' for out of bounds, got '%s'", name)
	}
}

func TestLifecycleIsMature(t *testing.T) {
	l := &Lifecycle{
		CurrentStage: 0,
		MaxStages:    4,
	}
	
	if l.IsMature() {
		t.Error("Stage 0 of 4 should not be mature")
	}
	
	l.CurrentStage = 2 // Half of 4
	if !l.IsMature() {
		t.Error("Stage 2 of 4 should be mature")
	}
}

func TestLifecycleIsDecayed(t *testing.T) {
	l := &Lifecycle{
		CurrentStage: 0,
		MaxStages:    4,
	}
	
	if l.IsDecayed() {
		t.Error("Stage 0 should not be decayed")
	}
	
	l.CurrentStage = 3 // Last stage
	if !l.IsDecayed() {
		t.Error("Last stage should be decayed")
	}
}

func TestMatchableInterface(t *testing.T) {
	m := &Matchable{
		MatchID:      "match1",
		LogicKey:     "key1",
		Element:      "fire",
		NarrativeTag: "story1",
		MatchTypes:   []string{"identical", "elemental"},
	}
	
	if m.Type() != "matchable" {
		t.Error("Wrong component type")
	}
	
	if m.GetMatchID() != "match1" {
		t.Error("GetMatchID failed")
	}
	
	if m.GetLogicKey() != "key1" {
		t.Error("GetLogicKey failed")
	}
	
	if m.GetElement() != "fire" {
		t.Error("GetElement failed")
	}
	
	if m.GetNarrativeTag() != "story1" {
		t.Error("GetNarrativeTag failed")
	}
	
	types := m.GetMatchTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 match types, got %d", len(types))
	}
}

func TestInventory(t *testing.T) {
	inv := &Inventory{
		Slots:    make([]string, 0),
		MaxSlots: 3,
	}
	
	// Add items
	if !inv.Add("item1") {
		t.Error("Should be able to add item1")
	}
	
	if !inv.Add("item2") {
		t.Error("Should be able to add item2")
	}
	
	if !inv.Add("item3") {
		t.Error("Should be able to add item3")
	}
	
	// Should be full now
	if inv.Add("item4") {
		t.Error("Should not be able to add item4 - inventory full")
	}
	
	if inv.Count() != 3 {
		t.Errorf("Expected 3 items, got %d", inv.Count())
	}
	
	// Remove item
	if !inv.Remove("item2") {
		t.Error("Should be able to remove item2")
	}
	
	if inv.Has("item2") {
		t.Error("item2 should be removed")
	}
	
	// Can't remove non-existent
	if inv.Remove("item99") {
		t.Error("Should not be able to remove non-existent item")
	}
	
	// Test Has
	if !inv.Has("item1") {
		t.Error("Should have item1")
	}
	if inv.Has("item99") {
		t.Error("Should not have item99")
	}
}
