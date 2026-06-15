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
	TileMerged         Type = "tile_merged"
	LootAcquired       Type = "loot_acquired"
	InventoryFull      Type = "inventory_full"
	EntityCreated      Type = "entity_created"
	EntityRemoved      Type = "entity_removed"
	DifficultyChanged  Type = "difficulty_changed"
	GridEntered        Type = "grid_entered"
	WorldGenerated     Type = "world_generated"
	CreatureFled       Type = "creature_fled"
	ItemMessage        Type = "item_message"
	NavigationOpened   Type = "navigation_opened"
	NavigationClosed   Type = "navigation_closed"
	LevelUp            Type = "level_up"
	// Animation events pour l'UI
	AnimationStarted Type = "animation_started"
	AnimationEnded   Type = "animation_ended"
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

func NewCreatureMovedEvent(creatureID string, from, to entity.Position, mode string, hidden bool, audible bool) Event {
	return Event{
		Type:     CreatureMoved,
		SourceID: creatureID,
		Payload: map[string]interface{}{
			"from":    from,
			"to":      to,
			"mode":    mode,
			"hidden":  hidden,
			"audible": audible,
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

func NewEntityRevealedEvent(tilePos entity.Position, entityID string, gridID string, flipDir entity.FlipDirection, payload ...map[string]interface{}) Event {
	p := map[string]interface{}{
		"position":       tilePos,
		"entity_id":      entityID,
		"grid_id":        gridID,
		"flip_direction": flipDir,
	}
	if len(payload) > 0 && payload[0] != nil {
		for k, v := range payload[0] {
			p[k] = v
		}
	}
	return Event{
		Type:     TileRevealed,
		SourceID: entityID,
		Payload:  p,
		Timestamp: time.Now(),
	}
}

func NewTileMatchedEvent(tilePos entity.Position, entityID string, name string, entityType entity.Type, level int) Event {
	return Event{
		Type:     TileMatched,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"position":    tilePos,
			"entity_id":   entityID,
			"name":        name,
			"entity_type": entityType,
			"level":       level,
		},
		Timestamp: time.Now(),
	}
}

func NewTileMergedEvent(tilePos entity.Position, entityID string, name string, entityType entity.Type, level int) Event {
	return Event{
		Type:     TileMerged,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"position":    tilePos,
			"entity_id":   entityID,
			"name":        name,
			"entity_type": entityType,
			"level":       level,
		},
		Timestamp: time.Now(),
	}
}

func NewLootAcquiredEvent(itemID string, name string, entityType entity.Type) Event {
	return Event{
		Type:     LootAcquired,
		SourceID: itemID,
		Payload: map[string]interface{}{
			"item_id":     itemID,
			"name":        name,
			"entity_type": entityType,
		},
		Timestamp: time.Now(),
	}
}

func NewItemMessageEvent(message string) Event {
	return Event{
		Type:     ItemMessage,
		SourceID: "item_message",
		Payload: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now(),
	}
}

func NewInventoryFullEvent() Event {
	return Event{
		Type:      InventoryFull,
		SourceID:  "inventory",
		Payload:   map[string]interface{}{},
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

func NewDifficultyChangedEvent(level string) Event {
	return Event{
		Type:     DifficultyChanged,
		SourceID: "system",
		Payload: map[string]interface{}{
			"level": level,
		},
		Timestamp: time.Now(),
	}
}

func NewGridEnteredEvent(gridID string) Event {
	return Event{
		Type:     GridEntered,
		SourceID: "player",
		Payload: map[string]interface{}{
			"grid_id": gridID,
		},
		Timestamp: time.Now(),
	}
}

func NewCreatureFledEvent(creatureID string, species string, gridID string, position entity.Position) Event {
	return Event{
		Type:     CreatureFled,
		SourceID: creatureID,
		Payload: map[string]interface{}{
			"species":  species,
			"grid_id":  gridID,
			"position": position,
		},
		Timestamp: time.Now(),
	}
}

func NewWorldGeneratedEvent(planeID string, zoneCount int) Event {
	return Event{
		Type:     WorldGenerated,
		SourceID: planeID,
		Payload: map[string]interface{}{
			"zone_count": zoneCount,
		},
		Timestamp: time.Now(),
	}
}

func NewNavigationOpenedEvent(gridID string) Event {
	return Event{
		Type:     NavigationOpened,
		SourceID: "system",
		Payload: map[string]interface{}{
			"grid_id": gridID,
		},
		Timestamp: time.Now(),
	}
}

func NewNavigationClosedEvent(gridID string) Event {
	return Event{
		Type:     NavigationClosed,
		SourceID: "system",
		Payload: map[string]interface{}{
			"grid_id": gridID,
		},
		Timestamp: time.Now(),
	}
}

func NewLevelUpEvent(newLevel int) Event {
	return Event{
		Type:     LevelUp,
		SourceID: "player",
		Payload: map[string]interface{}{
			"level": newLevel,
		},
		Timestamp: time.Now(),
	}
}

// GameState représente l'état actuel du jeu
type GameState string

const (
	StateMenu     GameState = "menu"
	StatePlaying  GameState = "playing"
	StateGameOver GameState = "game_over"
)

func NewPhaseChangedEvent(from, to GameState) Event {
	return Event{
		Type:     PhaseChanged,
		SourceID: "game",
		Payload: map[string]interface{}{
			"from": string(from),
			"to":   string(to),
		},
		Timestamp: time.Now(),
	}
}

// NewAnimationStartedEvent crée un événement signalant le début d'une animation graphique.
func NewAnimationStartedEvent(animationType string, targetID string, payload ...map[string]interface{}) Event {
	p := map[string]interface{}{
		"animation_type": animationType,
	}
	if len(payload) > 0 && payload[0] != nil {
		for k, v := range payload[0] {
			p[k] = v
		}
	}
	return Event{
		Type:      AnimationStarted,
		SourceID:  targetID,
		Payload:   p,
		Timestamp: time.Now(),
	}
}

// NewAnimationEndedEvent crée un événement signalant la fin d'une animation graphique.
func NewAnimationEndedEvent(animationType string, targetID string, payload ...map[string]interface{}) Event {
	p := map[string]interface{}{
		"animation_type": animationType,
	}
	if len(payload) > 0 && payload[0] != nil {
		for k, v := range payload[0] {
			p[k] = v
		}
	}
	return Event{
		Type:      AnimationEnded,
		SourceID:  targetID,
		Payload:   p,
		Timestamp: time.Now(),
	}
}
