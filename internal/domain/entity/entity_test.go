package entity

import (
	"testing"
)

func TestNewID(t *testing.T) {
	id1 := NewID()
	id2 := NewID()
	
	if id1 == id2 {
		t.Error("NewID should generate unique IDs")
	}
	
	if len(id1) == 0 {
		t.Error("NewID should not return empty string")
	}
}

func TestBaseEntity(t *testing.T) {
	e := NewBaseEntity(TypeResource)
	
	if e.GetType() != TypeResource {
		t.Errorf("Expected type %v, got %v", TypeResource, e.GetType())
	}
	
	if !e.IsActive() {
		t.Error("New entity should be active")
	}
	
	e.Deactivate()
	if e.IsActive() {
		t.Error("Entity should be inactive after Deactivate()")
	}
}

func TestPosition(t *testing.T) {
	p1 := Position{X: 1, Y: 2}
	p2 := Position{X: 3, Y: 4}
	
	result := p1.Add(p2)
	if result.X != 4 || result.Y != 6 {
		t.Errorf("Add failed: expected (4,6), got (%d,%d)", result.X, result.Y)
	}
	
	dist := p1.Distance(p2)
	if dist != 4 { // |1-3| + |2-4| = 2 + 2 = 4
		t.Errorf("Distance failed: expected 4, got %d", dist)
	}
}

func TestEntityManager(t *testing.T) {
	mgr := NewManager()
	
	// Test registration
	e1 := &BaseEntity{
		ID:     NewID(),
		EType:  TypeResource,
		Active: true,
	}
	e1.SetPosition(Position{X: 0, Y: 0})
	
	mgr.Register(e1)
	
	if mgr.Count() != 1 {
		t.Errorf("Expected 1 entity, got %d", mgr.Count())
	}
	
	// Test Get
	retrieved, ok := mgr.Get(e1.GetID())
	if !ok {
		t.Error("Failed to retrieve entity")
	}
	if retrieved.GetID() != e1.GetID() {
		t.Error("Retrieved wrong entity")
	}
	
	// Test GetByPosition
	byPos, ok := mgr.GetByPosition(Position{X: 0, Y: 0})
	if !ok {
		t.Error("Failed to get entity by position")
	}
	if byPos.GetID() != e1.GetID() {
		t.Error("GetByPosition returned wrong entity")
	}
	
	// Test UpdatePosition
	err := mgr.UpdatePosition(e1.GetID(), Position{X: 5, Y: 5})
	if err != nil {
		t.Errorf("UpdatePosition failed: %v", err)
	}
	
	// Old position should be empty
	_, ok = mgr.GetByPosition(Position{X: 0, Y: 0})
	if ok {
		t.Error("Old position should be empty after update")
	}
	
	// Test Remove
	mgr.Remove(e1.GetID())
	if mgr.Count() != 0 {
		t.Errorf("Expected 0 entities after removal, got %d", mgr.Count())
	}
}

func TestEntityTags(t *testing.T) {
	e := NewBaseEntity(TypeCreature)
	
	e.AddTag("hostile")
	if !e.HasTag("hostile") {
		t.Error("HasTag should return true for added tag")
	}
	
	// Duplicate tags should be ignored
	e.AddTag("hostile")
	if len(e.Tags) != 1 {
		t.Errorf("Duplicate tags should be ignored, got %d tags", len(e.Tags))
	}
	
	e.RemoveTag("hostile")
	if e.HasTag("hostile") {
		t.Error("Tag should be removed after RemoveTag")
	}
}

func TestQueryByTag(t *testing.T) {
	mgr := NewManager()
	
	e1 := NewBaseEntity(TypeResource)
	e1.SetPosition(Position{X: 0, Y: 0})
	e1.AddTag("valuable")
	
	e2 := NewBaseEntity(TypeCreature)
	e2.SetPosition(Position{X: 1, Y: 1})
	e2.AddTag("hostile")
	
	mgr.Register(&e1)
	mgr.Register(&e2)
	
	valuables := mgr.QueryByTag("valuable")
	if len(valuables) != 1 {
		t.Errorf("Expected 1 valuable entity, got %d", len(valuables))
	}
}

func TestGetByType(t *testing.T) {
	mgr := NewManager()
	
	e1 := NewBaseEntity(TypeResource)
	e1.SetPosition(Position{X: 0, Y: 0})
	
	e2 := NewBaseEntity(TypeCreature)
	e2.SetPosition(Position{X: 1, Y: 1})
	
	mgr.Register(&e1)
	mgr.Register(&e2)
	
	resources := mgr.GetByType(TypeResource)
	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	
	creatures := mgr.GetByType(TypeCreature)
	if len(creatures) != 1 {
		t.Errorf("Expected 1 creature, got %d", len(creatures))
	}
}
