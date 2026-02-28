// Package domain fournit le cœur métier du jeu.
// Il ré-exporte les sous-packages pour faciliter l'utilisation.
package domain

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/association"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
)

// Ré-export des types principaux pour faciliter l'accès
type (
	// Entity
	ID       = entity.ID
	Entity   = entity.Entity
	Position = entity.Position
	Type     = entity.Type

	// Board
	Grid           = board.Grid
	Tile           = board.Tile
	TileState      = board.TileState
	Direction      = board.Direction
	FlipDirection  = board.FlipDirection
	TileModifier   = board.TileModifier

	// Component
	Component   = component.Component
	Store       = component.Store
	Lifecycle   = component.Lifecycle
	Visual      = component.Visual
	Matchable   = component.Matchable
	Mobility    = component.Mobility
	Behavior    = component.Behavior
	Inventory   = component.Inventory
	Value       = component.Value
	Trigger     = component.Trigger
	Concealment = component.Concealment

	// Creature
	Creature = creature.Creature
	Action   = creature.Action
	AI       = creature.AI

	// Resource
	Resource = resource.Resource

	// Event
	Bus       = event.Bus
	Event     = event.Event
	EventType = event.Type
	GameState = event.GameState

	// Player
	Player       = player.Player
	PlayerStats  = player.Stats
	PlayerSkills = player.Skills

	// Meta
	Family          = meta.Family
	MetaProgression = meta.MetaProgression
	Hub             = meta.Hub

	// Association
	AssocEngine = association.Engine
	AssocResult = association.Result
	AssocType   = association.Type
)

// Constants
const (
	TypeResource  = entity.TypeResource
	TypeCreature  = entity.TypeCreature
	TypeStructure = entity.TypeStructure
	TypeArtefact  = entity.TypeArtefact

	Hidden  = board.Hidden
	Revealed = board.Revealed
	Matched = board.Matched
	Blocked = board.Blocked

	North = board.North
	South = board.South
	East  = board.East
	West  = board.West

	// Flip directions
	FlipTop         = board.FlipTop
	FlipTopRight    = board.FlipTopRight
	FlipRight       = board.FlipRight
	FlipBottomRight = board.FlipBottomRight
	FlipBottom      = board.FlipBottom
	FlipBottomLeft  = board.FlipBottomLeft
	FlipLeft        = board.FlipLeft
	FlipTopLeft     = board.FlipTopLeft
	FlipCenter      = board.FlipCenter

	// Game states
	StateMenu     = event.StateMenu
	StatePlaying  = event.StatePlaying
	StateGameOver = event.StateGameOver
)

// Factory functions
var (
	NewID         = entity.NewID
	NewManager    = entity.NewManager
	NewBaseEntity = entity.NewBaseEntity

	NewGrid = board.NewGrid

	NewStore = component.NewStore

	NewCreature = creature.New
	NewCreatureFactory = creature.NewFactory

	NewResource = resource.New
	NewResourceFactory = resource.NewFactory

	NewBus = event.NewBus

	NewPlayer = player.New

	NewFamily = meta.NewFamily
	NewMetaProgression = meta.NewMetaProgression
	NewHub = meta.NewHub

	NewAssocEngine = association.NewEngine

	NewPhaseChangedEvent = event.NewPhaseChangedEvent
)
