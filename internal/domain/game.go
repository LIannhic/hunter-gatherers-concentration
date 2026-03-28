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

// Ré-export des types principaux
type (
	// Entity & Position
	ID        = entity.ID
	Entity    = entity.Entity
	Position  = board.Position // On privilégie board.Position comme référence
	Type      = entity.Type
	TileState = entity.TileState

	// Board & Environnement
	Grid          = board.Grid
	Plot          = board.Plot
	Direction     = board.Direction
	FlipDirection = board.FlipDirection
	PlotModifier  = board.PlotModifier
	BiomeType     = board.BiomeType // Ajouté
	Climate       = board.Climate   // Ajouté
	Season        = board.Season    // Ajouté

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

	// Creature & Resource
	Creature = creature.Creature
	Action   = creature.Action
	AI       = creature.AI
	Resource = resource.Resource

	// Event & State
	Bus       = event.Bus
	Event     = event.Event
	EventType = event.Type
	GameState = event.GameState

	// Player, Meta & Assoc
	Player      = player.Player
	PlayerStats = player.Stats
	Family      = meta.Family
	AssocEngine = association.Engine
)

// Constantes
const (
	// Types d'entités
	TypeResource  = entity.TypeResource
	TypeCreature  = entity.TypeCreature
	TypeStructure = entity.TypeStructure
	TypeTrap      = entity.TypeTrap

	// États des tuiles
	Hidden   = entity.Hidden
	Revealed = entity.Revealed
	Matched  = entity.Matched
	Blocked  = entity.Blocked

	// Orientations
	North = board.North
	South = board.South
	East  = board.East
	West  = board.West

	// Biomes (Ajoutés pour faciliter CreateGrid)
	BiomeForest = board.BiomeForest
	BiomeCave   = board.BiomeCave
	BiomeDesert = board.BiomeDesert

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

	NewCreature        = creature.New
	NewCreatureFactory = creature.NewFactory

	NewResource        = resource.New
	NewResourceFactory = resource.NewFactory

	NewBus = event.NewBus

	NewPlayer = player.New

	NewFamily          = meta.NewFamily
	NewMetaProgression = meta.NewMetaProgression
	NewHub             = meta.NewHub

	NewAssocEngine = association.NewEngine

	NewPhaseChangedEvent = event.NewPhaseChangedEvent
)
