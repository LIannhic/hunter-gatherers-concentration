package event

import (
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Type d'événement
type Type string

const (
	CreatureMoved      Type = "creature_moved"
	ResourceMatured    Type = "resource_matured"
	ResourcePropagated Type = "resource_propagated"
	AssociationMade    Type = "association_made"
	PlayerDamaged      Type = "player_damaged"
	CreatureCaptured   Type = "creature_captured"
	ExtractionStarted  Type = "extraction_started"
	PhaseChanged       Type = "phase_changed"
	TurnEnded          Type = "turn_ended"
	TileRevealed       Type = "tile_revealed"
	TileMatched        Type = "tile_matched"
	EntityCreated      Type = "entity_created"
	EntityRemoved      Type = "entity_removed"
	Victory            Type = "victory"
	GameOver           Type = "game_over"
)

// Event structure de base
type Event struct {
	Type      Type
	Timestamp time.Time
	Payload   map[string]interface{}
	SourceID  string // ID de l'entité source
}

// Handler interface pour les souscripteurs
type Handler interface {
	Handle(e Event)
}

// HandlerFunc permet d'utiliser une fonction comme handler
type HandlerFunc func(e Event)

func (f HandlerFunc) Handle(e Event) {
	f(e)
}

// Bus système de messagerie léger
type Bus struct {
	subscribers map[Type][]Handler
	queue       []Event
	history     []Event // Historique des événements
	maxHistory  int
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[Type][]Handler),
		queue:       make([]Event, 0),
		history:     make([]Event, 0),
		maxHistory:  100,
	}
}

func (b *Bus) Subscribe(eventType Type, handler Handler) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *Bus) SubscribeFunc(eventType Type, f func(Event)) {
	b.subscribers[eventType] = append(b.subscribers[eventType], HandlerFunc(f))
}

func (b *Bus) Unsubscribe(eventType Type, handler Handler) {
	handlers := b.subscribers[eventType]
	for i, h := range handlers {
		if h == handler {
			b.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (b *Bus) Publish(e Event) {
	b.queue = append(b.queue, e)
}

func (b *Bus) PublishImmediate(e Event) {
	b.dispatch(e)
	b.addToHistory(e)
}

func (b *Bus) ProcessQueue() {
	for _, e := range b.queue {
		b.dispatch(e)
		b.addToHistory(e)
	}
	b.queue = b.queue[:0] // Vide la queue
}

func (b *Bus) addToHistory(e Event) {
	b.history = append(b.history, e)
	if len(b.history) > b.maxHistory {
		b.history = b.history[1:]
	}
}

func (b *Bus) GetHistory() []Event {
	result := make([]Event, len(b.history))
	copy(result, b.history)
	return result
}

func (b *Bus) dispatch(e Event) {
	handlers := b.subscribers[e.Type]
	for _, h := range handlers {
		h.Handle(e)
	}
}

func (b *Bus) ClearQueue() {
	b.queue = b.queue[:0]
}

func (b *Bus) QueueSize() int {
	return len(b.queue)
}

// --- Événements spécifiques ---

func NewCreatureMovedEvent(creatureID string, from, to entity.Position) Event {
	return Event{
		Type:     CreatureMoved,
		SourceID: creatureID,
		Payload: map[string]interface{}{
			"from": from,
			"to":   to,
		},
		Timestamp: time.Now(),
	}
}

func NewResourceMaturedEvent(resourceID string, newStage string) Event {
	return Event{
		Type:     ResourceMatured,
		SourceID: resourceID,
		Payload: map[string]interface{}{
			"new_stage": newStage,
		},
		Timestamp: time.Now(),
	}
}

func NewAssociationMadeEvent(playerID string, assocType string, success bool) Event {
	return Event{
		Type:     AssociationMade,
		SourceID: playerID,
		Payload: map[string]interface{}{
			"type":    assocType,
			"success": success,
		},
		Timestamp: time.Now(),
	}
}

func NewTileRevealedEvent(tilePos entity.Position, entityID string) Event {
	return Event{
		Type:     TileRevealed,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"position":  tilePos,
			"entity_id": entityID,
		},
		Timestamp: time.Now(),
	}
}

func NewTileMatchedEvent(tilePos entity.Position, entityID string) Event {
	return Event{
		Type:     TileMatched,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"position":  tilePos,
			"entity_id": entityID,
		},
		Timestamp: time.Now(),
	}
}

func NewTurnEndedEvent(turnNumber int) Event {
	return Event{
		Type:     TurnEnded,
		SourceID: "system",
		Payload: map[string]interface{}{
			"turn": turnNumber,
		},
		Timestamp: time.Now(),
	}
}

func NewEntityCreatedEvent(entityID string, entityType string) Event {
	return Event{
		Type:     EntityCreated,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"entity_type": entityType,
		},
		Timestamp: time.Now(),
	}
}

func NewEntityRemovedEvent(entityID string, reason string) Event {
	return Event{
		Type:     EntityRemoved,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"reason": reason,
		},
		Timestamp: time.Now(),
	}
}

func NewVictoryEvent(turns int) Event {
	return Event{
		Type:     Victory,
		SourceID: "system",
		Payload: map[string]interface{}{
			"turns": turns,
		},
		Timestamp: time.Now(),
	}
}

func NewGameOverEvent(turns int, reason string) Event {
	return Event{
		Type:     GameOver,
		SourceID: "system",
		Payload: map[string]interface{}{
			"turns":  turns,
			"reason": reason,
		},
		Timestamp: time.Now(),
	}
}
