package system

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
)

// World contient tout l'état du jeu
type World struct {
	Grids      map[string]*board.Grid // Plusieurs grids indexés par ID
	GridOrder  []string               // Ordre stable des IDs de grid (pour affichage)
	Entities   *entity.Manager
	Components *component.Store
	EventBus   *event.Bus
	Turn       int
	MaxTurns   int
	PlayerID   string

	// Meta progression
	Hub *meta.Hub

	// Player logic
	Player *player.Player

	// Grids actifs pour le joueur (pour navigation entre grids)
	CurrentGridID string

	// Difficulty
	Difficulty meta.DifficultySettings

	// Dream Plane (Mega-board structure)
	DreamPlane *board.DreamPlane

	// Inventory is modeled as a separate logical grid and not part of the game zone list.
	InventoryGrid *board.Grid

	// Factories
	CreatureFactory *creature.Factory
	ResourceFactory *resource.Factory

	// Player
	playerPosition entity.Position

	// Turn state tracking
	tilesFlippedThisTurn []board.Position // Tracks tiles flipped in current turn (max 2)
	lastTurnNumber       int              // Used to detect turn changes

	// Progression
	WorldsCleared int

	// Real-time turn pressure timer
	TurnTimer *TurnTimer

	// Référence vers l'engine (pour la communication entre systèmes)
	Engine *Engine

	// Debug
	Debug DebugState
}

type DebugState struct {
	Visible            bool
	OverrideDifficulty bool
	Difficulty         meta.DifficultySettings
	AllowedCreatures   map[string]bool
	ActiveShaders      map[string]bool
}

// NewWorld initialise un nouveau monde avec les réglages par défaut
func NewWorld() *World {
	p := player.New("player_1")
	w := &World{
		Grids:                make(map[string]*board.Grid),
		GridOrder:            make([]string, 0),
		Entities:             entity.NewManager(),
		Components:           component.NewStore(),
		EventBus:             event.NewBus(),
		Turn:                 0,
		MaxTurns:             p.Stats.MaxSanity,
		CurrentGridID:        "",
		Difficulty:           meta.GetSettings(meta.LevelNormal),
		CreatureFactory:      creature.NewFactory(),
		ResourceFactory:      resource.NewFactory(),
		Hub:                  meta.NewHub(),
		Player:               p,
		playerPosition:       entity.Position{X: 0, Y: 0},
		tilesFlippedThisTurn: make([]board.Position, 0),
		lastTurnNumber:       0,
		TurnTimer:            NewTurnTimer(meta.GetSettings(meta.LevelNormal).TurnTimerDuration),
		Debug: DebugState{
			Difficulty: meta.GetSettings(meta.LevelNormal),
			AllowedCreatures: map[string]bool{
				"lumifly":         true,
				"shadowstalker":   true,
				"burrower":        true,
				"specter":         true,
				"echo_hound":      true,
				"fleeing_sprite":  true,
				"moss_monkey":     true,
				"stonewarden":     true,
				"flutterwing":     true,
				"dreamberry":      true,
				"moonstone":       true,
				"whispering_herb": true,
				"crystal_shard":   true,
				"moss_truffle":    true,
				"void_bloom":      true,
				"echo_crystal":    true,
				"sand_core":       true,
				"trap":            true,
				"start_portal":    true,
				"finish_portal":   true,
				"dolmen":          true,
				"obelisk":         true,
				"portable_portal": true,
			},
			ActiveShaders: make(map[string]bool),
		},
	}

	// Initialise la grille d'inventaire
	// 3 colonnes, 10 lignes pour 30 slots par défaut
	w.CreateGrid(board.InventoryGridID, 3, 10, board.BiomeDefault)

	w.initListeners()

	return w
}

func (w *World) initListeners() {
	w.EventBus.SubscribeFunc(event.AnimationEnded, func(e event.Event) {
		if animType, ok := e.Payload["animation_type"].(string); ok && animType == "attack" {
			if targetPos, ok := e.Payload["hit_target"].(entity.Position); ok {
				ent, exists := w.Entities.Get(entity.ID(e.SourceID))
				if !exists {
					return
				}
				// Création de la trace persistante (red beam)
				track := entity.NewTrack("intent_beam", 2, ent.GetPosition(), targetPos)
				track.SetGridID(ent.GetGridID())
				w.Entities.Register(track)
			}
		}
	})
}
