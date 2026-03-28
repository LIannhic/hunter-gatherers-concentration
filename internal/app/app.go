// Package app orchestre les composants de haut niveau
// C'est le "wiring" de l'application
package app

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

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
	Renderer    *renderer.BoardRenderer
	TitleScreen *renderer.TitleScreen
	Input       *input.Handler
	HUD         *hud.HUD

	// Game State
	State domain.GameState

	// Debug
	debug *DebugStats
}

// NewApplication crée et configure l'application
func NewApplication() (*Application, error) {
	// Initialise le générateur aléatoire
	rand.Seed(time.Now().UnixNano())

	app := &Application{}

	// 1. Charge la configuration
	config := loader.DefaultConfig()
	app.Config = config

	// 2. Initialise le domaine
	app.World = domain.NewWorld()
	app.AssocEngine = domain.NewAssocEngine()
	app.Engine = domain.NewEngine(app.World)

	// 3. Crée plusieurs grids
	app.setupGrids()

	// 4. Infrastructure
	app.Assets = assets.NewManager()

	// 5. UI
	app.Renderer = renderer.NewBoardRenderer(app.Assets)
	app.TitleScreen = renderer.NewTitleScreen()
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

	// 10. État initial : Menu
	app.State = domain.StateMenu
	fmt.Println("[STATE] État initial: MENU")

	return app, nil
}

// setupGrids crée les grids initiaux
func (app *Application) setupGrids() {
	gridConfigs := []struct {
		id     string
		width  int
		height int
		biome  domain.BiomeType
	}{
		{"forest", 6, 6, domain.BiomeForest},
		{"cave", 6, 6, domain.BiomeCave},
		{"meadow", 6, 6, domain.BiomeForest},
		{"swamp", 6, 6, domain.BiomeCave},
	}

	for _, cfg := range gridConfigs {
		app.World.CreateGrid(cfg.id, cfg.width, cfg.height, cfg.biome)
		fmt.Printf("Created grid: %s (%dx%d)\n", cfg.id, cfg.width, cfg.height)
	}

	// Définit le premier grid comme actif
	app.World.SetCurrentGrid("forest")
}

func (app *Application) fillGridWithTraps(gridID string) {
	grid, ok := app.World.GetGrid(gridID)
	if !ok {
		return
	}

	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			plot, err := grid.Get(pos)
			if err != nil {
				continue
			}

			if len(plot.EntitiesID) > 0 || plot.Modifier.Obstructed {
				continue
			}

			trap := entity.NewBaseEntity(entity.TypeTrap)
			trap.SetGridID(gridID)
			trap.SetPosition(entity.Position{X: x, Y: y})

			app.World.Entities.Register(&trap)
			plot.PushEntity(string(trap.GetID()))
		}
	}
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
				app.World.SpawnResource(gridID, resourceTypes[i], entity.Position{X: pos.x, Y: pos.y})
			}
		}

		// Spawn une créature
		app.World.SpawnCreature(gridID, "lumifly", entity.Position{X: 3, Y: 3})
		app.World.SpawnCreature(gridID, "lumifly", entity.Position{X: 3, Y: 4})
	}

	// Callback spawn toutes les créatures de test (Shift+S)
	app.Input.OnSpawnAllCreatures = func(gridID string) {
		fmt.Printf("[ACTION] Spawn ALL creatures on grid %s\n", gridID)
		app.debug.Action()

		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		creatures := []struct {
			species string
			desc    string
		}{
			{"lumifly", "Volant (Over)"},
			{"shadowstalker", "Chasseur (Shadow)"},
			{"burrower", "Fouisseur (Under)"},
			{"specter", "Fantôme (Phase)"},
			{"echo_hound", "Rapide (Echo)"},
			{"fleeing_sprite", "Fuyard (Repulsion)"},
		}

		spawned := 0
		for _, c := range creatures {
			// Trouve une position vide pour chaque créature
			pos := app.findEmptyPosition(gridID)
			if pos == nil {
				fmt.Printf("[ERROR] No empty position for %s\n", c.species)
				continue
			}

			if _, err := app.World.SpawnCreature(gridID, c.species, *pos); err != nil {
				fmt.Printf("[ERROR] Failed to spawn %s: %v\n", c.species, err)
			} else {
				fmt.Printf("[SPAWN] %s at %v - %s\n", c.species, *pos, c.desc)
				spawned++
			}
		}

		fmt.Printf("[SPAWN] Total spawned: %d\n", spawned)
	}

	// Callback spawn créature aléatoire (F9)
	app.Input.OnSpawnRandomCreature = func(gridID string) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		creatures := []string{"lumifly", "shadowstalker", "burrower", "specter", "echo_hound", "fleeing_sprite"}
		species := creatures[rand.Intn(len(creatures))]

		// Trouve une position libre aléatoire
		pos := app.findEmptyPosition(gridID)
		if pos == nil {
			fmt.Println("[ERROR] No empty position available")
			return
		}

		if _, err := app.World.SpawnCreature(gridID, species, *pos); err != nil {
			fmt.Printf("[ERROR] Failed to spawn %s: %v\n", species, err)
		} else {
			fmt.Printf("[SPAWN RANDOM] %s at %v\n", species, *pos)
			app.debug.Spawn()
		}
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

	// Callback retour au menu
	app.Input.OnExitToMenu = func() {
		app.ReturnToMenu()
	}

	// Configure les callbacks de débogage
	app.setupDebugCallbacks()
}

// setupDebugCallbacks configure les callbacks de débogage
func (app *Application) setupDebugCallbacks() {
	// F3: Forcer le prochain tour
	app.Input.OnForceTurn = func() {
		fmt.Println("[DEBUG] Forcing turn end")
		app.Engine.Update()
	}

	// F5: Révéler toutes les tuiles (cheat)
	app.Input.OnRevealAll = func(gridID string) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}
		grid, ok := app.World.GetGrid(gridID)
		if !ok {
			return
		}
		fmt.Printf("[CHEAT] Révélation de toutes les tuiles sur %s\n", gridID)
		for _, tile := range grid.Plots {
			if len(tile.EntitiesID) == 0 {
				continue
			}
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := app.World.Entities.Get(entity.ID(topID)); ok {
				if e.GetState() == entity.Hidden {
					e.SetState(entity.Revealed)
				}
			}
		}
	}

	// F6: Cacher toutes les tuiles (cheat)
	app.Input.OnHideAll = func(gridID string) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}
		grid, ok := app.World.GetGrid(gridID)
		if !ok {
			return
		}
		fmt.Printf("[CHEAT] Masquage de toutes les tuiles sur %s\n", gridID)
		for _, tile := range grid.Plots {
			if len(tile.EntitiesID) == 0 {
				continue
			}
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := app.World.Entities.Get(entity.ID(topID)); ok {
				e.SetState(entity.Hidden)
			}
		}
	}

	// F10: Toggle mouvement automatique
	app.Input.OnToggleAutoMove = func() {
		app.Engine.Running = !app.Engine.Running
		if app.Engine.Running {
			fmt.Println("[DEBUG] Mouvement automatique: ON")
		} else {
			fmt.Println("[DEBUG] Mouvement automatique: OFF")
		}
	}
}

// setupEventSubscriptions abonne le renderer aux événements pour les animations
func (app *Application) setupEventSubscriptions() {
	// Abonne le renderer aux événements TileRevealed pour démarrer les animations
	app.World.EventBus.SubscribeFunc(event.TileRevealed, func(e event.Event) {
		position, ok1 := e.Payload["position"].(entity.Position)
		entityID, ok3 := e.Payload["entity_id"].(string)
		gridID, ok4 := e.Payload["grid_id"].(string)
		flipDir, ok5 := e.Payload["flip_direction"].(board.FlipDirection)

		if ok1 && ok3 && ok4 && ok5 {
			if ent, ok := app.World.Entities.Get(entity.ID(entityID)); ok {
				// Démarre l'animation de flip
				app.Renderer.StartFlipAnimation(
					gridID,
					board.Position{X: position.X, Y: position.Y},
					flipDir,
					entityID,
					ent.GetState(),
				)
			}
		}
	})
}

// spawnInitialEntities crée quelques entités au démarrage sur différents grids
func (app *Application) spawnInitialEntities() {
	// 1. Spawn des entités réelles (Ressources et Créatures)
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
			app.World.SpawnResource(gridID, spawn.typ, entity.Position{X: spawn.pos.X, Y: spawn.pos.Y})
		}
	}

	// Spawn les créatures
	app.World.SpawnCreature("forest", "lumifly", entity.Position{X: 3, Y: 3})
	app.World.SpawnCreature("cave", "shadowstalker", entity.Position{X: 4, Y: 2})
	app.World.SpawnCreature("meadow", "burrower", entity.Position{X: 2, Y: 4})
	app.World.SpawnCreature("swamp", "specter", entity.Position{X: 3, Y: 3})
	app.World.SpawnCreature("forest", "echo_hound", entity.Position{X: 5, Y: 5})
	app.World.SpawnCreature("meadow", "fleeing_sprite", entity.Position{X: 1, Y: 1})

	// 2. COMPLÉTION : Remplit les cases encore vides avec des pièges (Traps)
	fmt.Println("[INIT] Filling remaining tiles with traps...")
	for _, gridID := range app.World.GridOrder {
		app.fillGridWithTraps(gridID)
	}
}

// Update met à jour l'application
func (app *Application) Update() error {
	// Stats debug
	app.debug.Frame()

	// Gestion selon l'état du jeu
	switch app.State {
	case domain.StateMenu:
		return app.updateMenu()
	case domain.StatePlaying:
		return app.updatePlaying()
	case domain.StateGameOver:
		return app.updateGameOver()
	}

	return nil
}

// updateMenu gère l'écran titre
func (app *Application) updateMenu() error {
	// Vérifie le clic sur le bouton démarrer
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if app.TitleScreen.IsStartButtonClicked(x, y) {
			app.StartGame()
		}
	}
	return nil
}

// updatePlaying gère le jeu en cours
func (app *Application) updatePlaying() error {
	// Gère les entrées
	return app.Input.Update()
}

// updateGameOver gère l'écran de fin
func (app *Application) updateGameOver() error {
	return nil
}

// StartGame démarre le jeu depuis le menu
func (app *Application) StartGame() {
	oldState := app.State
	app.State = domain.StatePlaying

	// Publie l'événement de changement de phase
	app.World.EventBus.Publish(domain.NewPhaseChangedEvent(oldState, app.State))

	fmt.Printf("[STATE] Transition: %s -> %s\n", oldState, app.State)

	// Démarre l'engine (nécessaire pour les mouvements des créatures)
	app.Engine.Start()
	fmt.Println("[ENGINE] Started")

	// Spawn les entités initiales si nécessaire
	if app.World.Entities.Count() == 0 {
		fmt.Println("=== Spawning initial entities ===")
		app.spawnInitialEntities()
		fmt.Printf("=== Total entities: %d ===\n", app.World.Entities.Count())
	}
}

// ReturnToMenu retourne au menu principal
func (app *Application) ReturnToMenu() {
	oldState := app.State
	app.State = domain.StateMenu

	// Publie l'événement de changement de phase
	app.World.EventBus.Publish(domain.NewPhaseChangedEvent(oldState, app.State))

	fmt.Printf("[STATE] Transition: %s -> %s\n", oldState, app.State)

	// Réinitialise l'état du jeu
	app.Input.ResetGameState()

	// Réinitialise la rotation du plateau
	app.Renderer.SetBoardRotation(0)

	fmt.Println("[MENU] Retour au menu principal")
}

// Draw dessine l'application
func (app *Application) Draw(screen *ebiten.Image) {
	// Gestion selon l'état du jeu
	switch app.State {
	case domain.StateMenu:
		app.drawMenu(screen)
	case domain.StatePlaying:
		app.drawPlaying(screen)
	case domain.StateGameOver:
		app.drawGameOver(screen)
	}
}

// drawMenu dessine l'écran titre
func (app *Application) drawMenu(screen *ebiten.Image) {
	app.TitleScreen.Render(screen)
}

// drawPlaying dessine le jeu en cours
func (app *Application) drawPlaying(screen *ebiten.Image) {
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

// drawGameOver dessine l'écran de fin
func (app *Application) drawGameOver(screen *ebiten.Image) {
	screen.Fill(color.Black)
	text.Draw(screen, "GAME OVER", basicfont.Face7x13, 350, 300, color.White)
}

// findEmptyPosition trouve une position vide aléatoire sur le grid
func (app *Application) findEmptyPosition(gridID string) *entity.Position {
	grid, ok := app.World.GetGrid(gridID)
	if !ok {
		return nil
	}

	// Collecte toutes les positions vides
	var emptyPositions []entity.Position
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			tile, _ := grid.Get(pos)
			if len(tile.EntitiesID) == 0 && !tile.Modifier.Obstructed {
				emptyPositions = append(emptyPositions, entity.Position{X: x, Y: y})
			}
		}
	}

	if len(emptyPositions) == 0 {
		return nil
	}

	pos := emptyPositions[rand.Intn(len(emptyPositions))]
	return &pos
}

// Layout retourne la taille de la fenêtre
func (app *Application) Layout(outsideWidth, outsideHeight int) (int, int) {
	// En mode menu, utilise la taille de l'écran titre
	if app.State == domain.StateMenu {
		return app.TitleScreen.Layout()
	}

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
