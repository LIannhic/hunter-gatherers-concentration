package event

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestNewBus(t *testing.T) {
	b := NewBus()
	if b == nil {
		t.Error("NewBus should not return nil")
	}
}

func TestBusSubscribeAndPublish(t *testing.T) {
	b := NewBus()
	
	received := false
	handler := HandlerFunc(func(e Event) {
		received = true
	})
	
	b.Subscribe(CreatureMoved, handler)
	b.Publish(Event{Type: CreatureMoved})
	b.ProcessQueue()
	
	if !received {
		t.Error("Handler should have been called")
	}
}

func TestBusSubscribeFunc(t *testing.T) {
	b := NewBus()
	
	received := false
	b.SubscribeFunc(CreatureMoved, func(e Event) {
		received = true
	})
	
	b.Publish(Event{Type: CreatureMoved})
	b.ProcessQueue()
	
	if !received {
		t.Error("Handler should have been called")
	}
}

func TestBusMultipleSubscribers(t *testing.T) {
	b := NewBus()
	
	count := 0
	b.SubscribeFunc(CreatureMoved, func(e Event) {
		count++
	})
	b.SubscribeFunc(CreatureMoved, func(e Event) {
		count++
	})
	
	b.Publish(Event{Type: CreatureMoved})
	b.ProcessQueue()
	
	if count != 2 {
		t.Errorf("Expected 2 handler calls, got %d", count)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	b := NewBus()
	
	// Note: Unsubscribe is hard to test with HandlerFunc because 
	// function values can't be compared. In practice, unsubscribe
	// would be used with a named handler struct.
	// This test just verifies the method doesn't panic.
	b.Unsubscribe(CreatureMoved, HandlerFunc(func(e Event) {}))
}

func TestBusDifferentEventTypes(t *testing.T) {
	b := NewBus()
	
	creatureMovedCalled := false
	resourceMaturedCalled := false
	
	b.SubscribeFunc(CreatureMoved, func(e Event) {
		creatureMovedCalled = true
	})
	b.SubscribeFunc(ResourceMatured, func(e Event) {
		resourceMaturedCalled = true
	})
	
	b.Publish(Event{Type: CreatureMoved})
	b.ProcessQueue()
	
	if !creatureMovedCalled {
		t.Error("CreatureMoved handler should have been called")
	}
	if resourceMaturedCalled {
		t.Error("ResourceMatured handler should not have been called")
	}
}

func TestBusPublishImmediate(t *testing.T) {
	b := NewBus()
	
	received := false
	b.SubscribeFunc(CreatureMoved, func(e Event) {
		received = true
	})
	
	b.PublishImmediate(Event{Type: CreatureMoved})
	
	// Should be received immediately without ProcessQueue
	if !received {
		t.Error("Handler should have been called immediately")
	}
}

func TestBusQueueSize(t *testing.T) {
	b := NewBus()
	
	if b.QueueSize() != 0 {
		t.Error("Queue should be empty initially")
	}
	
	b.Publish(Event{Type: CreatureMoved})
	b.Publish(Event{Type: ResourceMatured})
	
	if b.QueueSize() != 2 {
		t.Errorf("Queue should have 2 events, got %d", b.QueueSize())
	}
	
	b.ProcessQueue()
	
	if b.QueueSize() != 0 {
		t.Error("Queue should be empty after processing")
	}
}

func TestBusClearQueue(t *testing.T) {
	b := NewBus()
	
	b.Publish(Event{Type: CreatureMoved})
	b.Publish(Event{Type: ResourceMatured})
	
	b.ClearQueue()
	
	if b.QueueSize() != 0 {
		t.Error("Queue should be empty after clear")
	}
}

func TestBusHistory(t *testing.T) {
	b := NewBus()
	
	b.PublishImmediate(Event{Type: CreatureMoved, SourceID: "1"})
	b.PublishImmediate(Event{Type: ResourceMatured, SourceID: "2"})
	
	history := b.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 events in history, got %d", len(history))
	}
}

func TestNewCreatureMovedEvent(t *testing.T) {
	from := entity.Position{X: 0, Y: 0}
	to := entity.Position{X: 1, Y: 1}
	
	e := NewCreatureMovedEvent("creature1", from, to)
	
	if e.Type != CreatureMoved {
		t.Error("Wrong event type")
	}
	
	if e.SourceID != "creature1" {
		t.Error("Wrong source ID")
	}
	
	if e.Payload["from"] != from {
		t.Error("Wrong from position")
	}
	
	if e.Payload["to"] != to {
		t.Error("Wrong to position")
	}
}

func TestNewResourceMaturedEvent(t *testing.T) {
	e := NewResourceMaturedEvent("resource1", "fruit")
	
	if e.Type != ResourceMatured {
		t.Error("Wrong event type")
	}
	
	if e.SourceID != "resource1" {
		t.Error("Wrong source ID")
	}
	
	if e.Payload["new_stage"] != "fruit" {
		t.Error("Wrong stage in payload")
	}
}

func TestNewAssociationMadeEvent(t *testing.T) {
	e := NewAssociationMadeEvent("player1", "identical", true)
	
	if e.Type != AssociationMade {
		t.Error("Wrong event type")
	}
	
	if e.Payload["type"] != "identical" {
		t.Error("Wrong association type")
	}
	
	if e.Payload["success"] != true {
		t.Error("Wrong success value")
	}
}

func TestNewTileRevealedEvent(t *testing.T) {
	pos := entity.Position{X: 2, Y: 3}
	flipDir := board.FlipCenter
	e := NewTileRevealedEvent(pos, "entity1", flipDir)
	
	if e.Type != TileRevealed {
		t.Error("Wrong event type")
	}
	
	if e.Payload["position"] != pos {
		t.Error("Wrong position")
	}
	
	if e.Payload["entity_id"] != "entity1" {
		t.Error("Wrong entity_id")
	}
	
	if e.Payload["flip_direction"] != flipDir {
		t.Error("Wrong flip_direction")
	}
}

func TestNewTurnEndedEvent(t *testing.T) {
	e := NewTurnEndedEvent(5)
	
	if e.Type != TurnEnded {
		t.Error("Wrong event type")
	}
	
	if e.Payload["turn"] != 5 {
		t.Error("Wrong turn number")
	}
}
