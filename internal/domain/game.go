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
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/structure"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/system"
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
	Direction     = entity.Direction
	FlipDirection = entity.FlipDirection
	PlotModifier  = board.PlotModifier
	BiomeType     = board.BiomeType
	Climate       = board.Climate
	Season        = board.Season

	// World & Engine
	World      = system.World
	Engine     = system.Engine
	TurnTimer  = system.TurnTimer
	DebugState = system.DebugState

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

	// Structure
	Structure           = structure.Structure
	NavigationStructure = structure.NavigationStructure
	NavType             = structure.NavType

	// Dream Plane (Mega-board structure)
	DreamPlane = board.DreamPlane

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

	// Persistence
	SaveData = persistence.SaveData
	SaveMeta = persistence.Metadata
	SaveRepo = persistence.Repository
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
	North = entity.DirNorth
	South = entity.DirSouth
	East  = entity.DirEast
	West  = entity.DirWest

	// Biomes (Ajoutés pour faciliter CreateGrid)
	BiomeForest = board.BiomeForest
	BiomeCave   = board.BiomeCave
	BiomeDesert = board.BiomeDesert

	// Flip directions
	FlipTop         = entity.FlipTop
	FlipTopRight    = entity.FlipTopRight
	FlipRight       = entity.FlipRight
	FlipBottomRight = entity.FlipBottomRight
	FlipBottom      = entity.FlipBottom
	FlipBottomLeft  = entity.FlipBottomLeft
	FlipLeft        = entity.FlipLeft
	FlipTopLeft     = entity.FlipTopLeft
	FlipCenter      = entity.FlipCenter

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

	NewLayoutGenerator = board.NewLayoutGenerator

	NewStore = component.NewStore

	NewCreature        = creature.New
	NewCreatureFactory = creature.NewFactory

	NewResource        = resource.New
	NewResourceFactory = resource.NewFactory

	NewStructure  = structure.NewStructure
	NewNavigation = structure.NewNavigation

	NewBus = event.NewBus

	NewPlayer = player.New

	NewFamily          = meta.NewFamily
	NewMetaProgression = meta.NewMetaProgression
	NewHub             = meta.NewHub

	NewAssocEngine = association.NewEngine

	NewPhaseChangedEvent = event.NewPhaseChangedEvent

	NewSaveData = persistence.NewSaveData

	// Core World & Engine
	NewWorld     = system.NewWorld
	NewEngine    = system.NewEngine
	NewTurnTimer = system.NewTurnTimer
)
