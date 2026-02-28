// Package app orchestre les composants de haut niveau
// C'est le "wiring" de l'application
package app

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/loader"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/hud"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/input"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/renderer"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/usecase"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

// Application est le conteneur principal
type Application struct {
	// Domain
	World       *domain.World
	AssocEngine *domain.AssocEngine
	Engine      *domain.Engine

	// Infrastructure
	Assets *assets.Manager
	Config *loader.GameConfig

	// UI
	Renderer *renderer.BoardRenderer
	Input    *input.Handler
	HUD      *hud.HUD

	// Debug
	debug *DebugStats
}

// NewApplication crée et configure l'application
func NewApplication() (*Application, error) {
	app := &Application{}

	// 1. Charge la configuration
	config := loader.DefaultConfig()
	app.Config = config

	// 2. Initialise le domaine (sans grid - on va en créer plusieurs)
	app.World = domain.NewWorld()
	app.AssocEngine = domain.NewAssocEngine()
	app.Engine = domain.NewEngine(app.World)

	// 3. Crée plusieurs grids
	app.setupGrids()

	// 4. Infrastructure
	app.Assets = assets.NewManager()

	// 5. UI
	app.Renderer = renderer.NewBoardRenderer(app.Assets)
	app.Input = input.NewHandler(app.World, app.AssocEngine)
	app.HUD = hud.NewHUD(app.World)

	// 6. Connecte les composants UI
	app.Input.SetRenderer(app.Renderer)

	// 7. Configure les callbacks
	app.setupCallbacks()

	// 8. Subscribe aux événements pour les animations
	app.setupEventSubscriptions()

	// 9. Debug
	app.debug = NewDebugStats()

	// 9. Spawne quelques entités de test au démarrage
	fmt.Println("=== Spawning initial entities ===")
	app.spawnInitialEntities()
	fmt.Printf("=== Total entities: %d ===\n", app.World.Entities.Count())

	return app, nil
}

// setupGrids crée plusieurs grids pour le jeu
func (app *Application) setupGrids() {
	// Crée 4 grids de 6x6
	gridConfigs := []struct {
		id     string
		width  int
		height int
	}{
		{"forest", 6, 6},
		{"cave", 6, 6},
		{"meadow", 6, 6},
		{"swamp", 6, 6},
	}

	for _, cfg := range gridConfigs {
		app.World.CreateGrid(cfg.id, cfg.width, cfg.height)
		fmt.Printf("Created grid: %s (%dx%d)\n", cfg.id, cfg.width, cfg.height)
	}

	// Définit le premier grid comme actif
	app.World.SetCurrentGrid("forest")
}

// setupCallbacks connecte les actions aux use cases
func (app *Application) setupCallbacks() {
	// Callback fin de tour
	app.Input.OnTurnEnd = func() {
		fmt.Println("[ACTION] Turn ended")
		app.debug.Action()
		app.Engine.Update()
	}

	// Callback spawn entités de test
	app.Input.OnSpawnEntities = func(gridID string) {
		fmt.Printf("[ACTION] Spawn button pressed on grid %s\n", gridID)
		app.debug.Action()

		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		// Spawn quelques ressources
		positions := []struct{ x, y int }{
			{1, 1}, {2, 1}, // Paire dreamberry
			{3, 2}, {4, 2}, // Paire moonstone
			{1, 3}, {2, 3}, // Paire whispering_herb
		}

		resourceTypes := []string{
			"dreamberry", "dreamberry",
			"moonstone", "moonstone",
			"whispering_herb", "whispering_herb",
		}

		for i, pos := range positions {
			if i < len(resourceTypes) {
				app.World.SpawnResource(gridID, resourceTypes[i], domain.Position{X: pos.x, Y: pos.y})
			}
		}

		// Spawn une créature
		app.World.SpawnCreature(gridID, "lumifly", domain.Position{X: 3, Y: 3})
	}

	// Callback clear board
	app.Input.OnClearBoard = func(gridID string) {
		fmt.Printf("[ACTION] Clear button pressed on grid %s\n", gridID)
		app.debug.Action()

		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		cmd := &usecase.ClearBoardCommand{
			World:  app.World,
			GridID: gridID,
		}
		cmd.Execute()
	}

	// Callback switch grid
	app.Input.OnSwitchGrid = func(gridID string) {
		fmt.Printf("[ACTION] Switching to grid %s\n", gridID)
		cmd := &usecase.SwitchGridCommand{
			World:  app.World,
			GridID: gridID,
		}
		if err := cmd.Execute(); err != nil {
			fmt.Printf("[ERROR] Failed to switch grid: %v\n", err)
		}
	}

	// Callback rotation du plateau
	app.Input.OnRotateBoard = func(delta float64) {
		app.Renderer.RotateBoard(delta)
	}

	// Callback réinitialisation rotation
	app.Input.OnResetRotation = func() {
		app.Renderer.SetBoardRotation(0)
	}
}

// setupEventSubscriptions abonne le renderer aux événements pour les animations
func (app *Application) setupEventSubscriptions() {
	// Abonne le renderer aux événements TileRevealed pour démarrer les animations
	app.World.EventBus.SubscribeFunc(event.TileRevealed, func(e event.Event) {
		position, ok1 := e.Payload["position"].(entity.Position)
		flipDir, ok2 := e.Payload["flip_direction"].(board.FlipDirection)
		entityID, ok3 := e.Payload["entity_id"].(string)
		gridID := e.SourceID

		if ok1 && ok2 && ok3 {
			// Démarre l'animation de flip
			app.Renderer.StartFlipAnimation(
				gridID,
				board.Position{X: position.X, Y: position.Y},
				flipDir,
				entityID,
				board.Revealed,
			)
		}
	})
}

// spawnInitialEntities crée quelques entités au démarrage sur différents grids
func (app *Application) spawnInitialEntities() {
	// Spawn des entités sur chaque grid
	gridSpawns := map[string][]struct {
		typ string
		pos domain.Position
	}{
		"forest": {
			{"dreamberry", domain.Position{X: 1, Y: 1}},
			{"dreamberry", domain.Position{X: 2, Y: 1}},
			{"whispering_herb", domain.Position{X: 4, Y: 3}},
		},
		"cave": {
			{"moonstone", domain.Position{X: 1, Y: 2}},
			{"moonstone", domain.Position{X: 2, Y: 3}},
		},
		"meadow": {
			{"dreamberry", domain.Position{X: 3, Y: 3}},
			{"whispering_herb", domain.Position{X: 4, Y: 4}},
		},
		"swamp": {
			{"whispering_herb", domain.Position{X: 2, Y: 2}},
			{"moonstone", domain.Position{X: 5, Y: 5}},
		},
	}

	for gridID, spawns := range gridSpawns {
		for _, spawn := range spawns {
			app.World.SpawnResource(gridID, spawn.typ, spawn.pos)
		}
	}

	// Spawn quelques créatures
	app.World.SpawnCreature("forest", "lumifly", domain.Position{X: 3, Y: 3})
	app.World.SpawnCreature("cave", "shadowstalker", domain.Position{X: 4, Y: 2})
	app.World.SpawnCreature("meadow", "burrower", domain.Position{X: 2, Y: 4})
	app.World.SpawnCreature("swamp", "lumifly", domain.Position{X: 1, Y: 1})
}

// Update met à jour l'application
func (app *Application) Update() error {
	// Stats debug
	app.debug.Frame()

	// Gère les entrées
	err := app.Input.Update()

	return err
}

// Draw dessine l'application
func (app *Application) Draw(screen *ebiten.Image) {
	// Fond noir
	screen.Fill(color.Black)

	// Dessine le plateau
	app.Renderer.Render(screen, app.World)

	// Dessine les surbrillances de sélection
	app.Input.Draw(screen)

	// Dessine le HUD
	app.HUD.Render(screen)

	// Message si aucune entité
	if app.World.Entities.Count() == 0 {
		text.Draw(screen, "Appuyez sur S pour spawner des entites", basicfont.Face7x13,
			200, 300,
			color.RGBA{255, 255, 0, 255})
	}
}

// Layout retourne la taille de la fenêtre
func (app *Application) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Calcule la taille nécessaire pour afficher tous les grids
	numGrids := len(app.World.Grids)
	if numGrids == 0 {
		return 800, 600
	}

	// Prend le premier grid comme référence de taille
	var gridWidth, gridHeight int
	if len(app.World.GridOrder) > 0 {
		if firstGrid, ok := app.World.GetGrid(app.World.GridOrder[0]); ok {
			gridWidth = firstGrid.Width * 64
			gridHeight = firstGrid.Height * 64
		}
	}

	gridsPerRow := 2
	numRows := (numGrids + gridsPerRow - 1) / gridsPerRow

	// Largeur suffisante pour la grille + le HUD
	width := gridWidth*gridsPerRow + 30*(gridsPerRow+1) + 300
	height := gridHeight*numRows + 50*(numRows+1) + 100

	if width < 800 {
		width = 800
	}
	if height < 600 {
		height = 600
	}

	return width, height
}
