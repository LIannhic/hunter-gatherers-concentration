// Package app orchestre les composants de haut niveau
// C'est le "wiring" de l'application
package app

import (
	"fmt"
	"image/color"
	"math/rand"
	"strings"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/loader"
	infraPersistence "github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/persistence"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/actionbuttons"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/hud"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/input"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/renderer"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/usecase"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// Application est le conteneur principal
type Application struct {
	// Domain
	World       *domain.World
	AssocEngine *domain.AssocEngine
	Engine      *domain.Engine

	// Infrastructure
	Assets      *assets.Manager
	Config      *loader.GameConfig
	Persistence *usecase.PersistenceManager

	// UI
	Renderer    *renderer.BoardRenderer
	TitleScreen *renderer.TitleScreen
	SaveMenu    *renderer.SaveMenu
	Input       *input.Handler
	HUD         *hud.HUD

	// Game State
	State domain.GameState

	// Session tracking
	sessionStartTime time.Time
	hasSaves         bool

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

	// Initialisation de la persistance (dossier ./saves)
	repo := infraPersistence.NewJsonRepository("./saves")
	app.Persistence = usecase.NewPersistenceManager(repo)

	// 5. UI
	app.Renderer = renderer.NewBoardRenderer(app.Assets)
	app.TitleScreen = renderer.NewTitleScreen()
	app.SaveMenu = renderer.NewSaveMenu()
	app.Input = input.NewHandler(app.World, app.AssocEngine)
	app.HUD = hud.NewHUD(app.World)
	app.HUD.SetAssetsManager(app.Assets)

	// Subscribe renderer to scanner events
	app.Renderer.SubscribeToEvents(app.World)

	// 5.1 Gestionnaire réactif des boutons d'action
	btnManager := actionbuttons.NewManager(
		func() int { return len(app.Input.GetRevealedTiles()) },
		func() *player.Player { return app.World.Player },
		func() float64 {
			if app.World.TurnTimer != nil {
				return app.World.TurnTimer.Progress()
			}
			return 0
		},
		func() bool {
			if app.World.TurnTimer != nil {
				return app.World.TurnTimer.IsPanic()
			}
			return false
		},
		func() float64 { return app.Input.GetVictoryTimerProgress() },
		func() bool { return app.Input.IsVictoryTimerActive() },
	)
	app.Renderer.ActionButtons = btnManager
	app.Input.SetActionButtonsManager(btnManager)

	// 5.2 Feedback temps réel sur le HUD (pulse Sanity Gauge)
	app.HUD.SetTimerCallbacks(
		func() float64 {
			if app.World.TurnTimer != nil {
				return app.World.TurnTimer.Remaining
			}
			return 0
		},
		func() bool {
			if app.World.TurnTimer != nil {
				return app.World.TurnTimer.IsPanic()
			}
			return false
		},
	)

	// 6. Connecte les composants UI
	app.Input.SetRenderer(app.Renderer)
	app.Input.OnToggleDetails = app.HUD.ToggleDetails
	app.Input.OnToggleInvDetails = app.HUD.ToggleInventoryDetails
	app.Input.OnToggleAssetsDetails = app.HUD.ToggleAssetsDetails
	app.Input.OnFillInventory = func() {
		fmt.Println("[DEBUG] Remplissage de l'inventaire avec des dreamberries et des limiers écho")
		for i := 0; i < 15; i++ {
			// 			loot := &player.LootItem{
			// 				ID:          string(entity.NewID()),
			// 				Name:        "bulle de savon",
			// 				Type:        entity.TypeResource,
			// 				SourceID:    "debug",
			// 				IsDeletable: true,
			// 			}
			// 			_ = app.World.Player.Inventory.AddItem(loot)
			_ = app.World.Player.Inventory.AddItem(player.NewDreamberryItem())
			_ = app.World.Player.Inventory.AddItem(player.NewEchoHoundItem())
		}
	}

	app.Input.OnUsePortablePortal = func(gridID string, center board.Position) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}
		fmt.Println("[ACTION] Utilisation du portail portable")
		cmd := &usecase.UsePortablePortalCommand{World: app.World, GridID: gridID, Center: center}
		if err := cmd.Execute(); err != nil {
			fmt.Printf("[ERROR] Impossible de déployer le portail portable : %v\n", err)
		} else {
			fmt.Println("[SUCCESS] Portail portable déployé. Démarrage du timer de victoire.")
			app.HUD.ClearActiveLootSelection()
			app.Input.SetPortablePortalMode(false)

			// Démarre le timer de victoire de 10 secondes
			app.Input.StartVictoryTimer(10.0)
		}
	}

	app.Input.OnVictory = func() {
		app.HUD.ShowVictory()
	}

	// 7. Configure les callbacks
	app.setupCallbacks()

	// 8. Subscribe aux événements pour les animations
	app.setupEventSubscriptions()

	// 9. Debug
	app.debug = NewDebugStats()

	// 10. État initial : Menu
	app.State = domain.StateMenu
	app.checkSaves()
	fmt.Println("[STATE] État initial: MENU")

	return app, nil
}

func (app *Application) checkSaves() {
	metas, _ := app.Persistence.GetSaveSummaries()
	app.hasSaves = len(metas) > 0
	// Mise à jour du texte du bouton de l'écran titre
	if app.hasSaves {
		app.TitleScreen.ButtonText = "CONTINUER"
	} else {
		app.TitleScreen.ButtonText = "DEMARRER"
	}
}

// setupGrids crée les grids initiaux via le générateur procédural
func (app *Application) setupGrids() {
	app.World.GenerateLayout("dream_plane_1")
	fmt.Printf("Generated Dream Plane with %d zones\n", len(app.World.Grids))

	// Définit la grille de commencement comme grille active par défaut
	if app.World.DreamPlane != nil {
		app.World.CurrentGridID = app.World.DreamPlane.StartZoneID
	}
}

// FillGridRandomly remplit un grid avec des paires d'entités et des pièges
func (app *Application) FillGridRandomly(gridID string) {
	grid, ok := app.World.GetGrid(gridID)
	if !ok {
		return
	}

	fmt.Printf("[INIT] Filling grid %s randomly...\n", gridID)

	// 1. Liste toutes les positions libres
	var positions []entity.Position
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			plot, _ := grid.Get(pos)
			if len(plot.EntitiesID) == 0 && !plot.Modifier.Obstructed {
				positions = append(positions, entity.Position{X: x, Y: y})
			}
		}
	}

	// 2. Mélange les positions
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	// 3. Types disponibles
	resourceTypes := []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"}
	creatureTypes := []string{"lumifly", "shadowstalker", "burrower", "specter", "echo_hound", "fleeing_sprite", "moss_monkey"}

	posIdx := 0
	totalTiles := len(positions)

	// On remplit par paires tant qu'on a de la place
	for posIdx < totalTiles-1 {
		// Choisit aléatoirement entre Ressource, Créature ou Piège
		choice := rand.Float32()

		if choice < 0.4 {
			// Paire de Ressources (40% de chance)
			resType := resourceTypes[rand.Intn(len(resourceTypes))]
			fmt.Printf("  - [%s] Spawning resource pair: %s at %v and %v\n", gridID, resType, positions[posIdx], positions[posIdx+1])
			app.World.SpawnResource(gridID, resType, positions[posIdx])
			app.World.SpawnResource(gridID, resType, positions[posIdx+1])
			posIdx += 2
		} else if choice < 0.8 {
			// Paire de Créatures (40% de chance)
			creType := creatureTypes[rand.Intn(len(creatureTypes))]
			fmt.Printf("  - [%s] Spawning creature pair: %s at %v and %v\n", gridID, creType, positions[posIdx], positions[posIdx+1])
			app.World.SpawnCreature(gridID, creType, positions[posIdx])
			app.World.SpawnCreature(gridID, creType, positions[posIdx+1])
			posIdx += 2
		} else {
			// Paire de Pièges (20% de chance)
			fmt.Printf("  - [%s] Spawning trap pair at %v and %v\n", gridID, positions[posIdx], positions[posIdx+1])
			app.World.SpawnTrap(gridID, positions[posIdx])
			app.World.SpawnTrap(gridID, positions[posIdx+1])
			posIdx += 2
		}
	}

	// Si le nombre de cases était impair, on met un dernier piège
	if posIdx < totalTiles {
		fmt.Printf("  - [%s] Spawning lone trap at %v\n", gridID, positions[posIdx])
		app.World.SpawnTrap(gridID, positions[posIdx])
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

	// Callback spawn entités de test (remplacé par remplissage aléatoire)
	app.Input.OnSpawnEntities = func(gridID string) {
		fmt.Printf("[ACTION] Random fill button pressed on grid %s\n", gridID)
		app.debug.Action()

		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		app.FillGridRandomly(gridID)
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
			{"lumifly", "Flying (Over)"},
			{"shadowstalker", "Hunter (Shadow)"},
			{"burrower", "Burrower (Under)"},
			{"specter", "Ghost (Phase)"},
			{"echo_hound", "Fast (Echo)"},
			{"fleeing_sprite", "Fleeing (Repulsion)"},
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

		creatures := []string{"lumifly", "shadowstalker", "burrower", "specter", "echo_hound", "fleeing_sprite", "moss_monkey"}
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

		// Extraction et affichage du biome pour le diagnostic
		if app.World != nil {
			// Syntaxe Go avec vérification directe de la présence (booléen)
			if g, ok := app.World.GetGrid(gridID); ok && g != nil {
				fmt.Printf("[DEBUG-BIOME] Grid: %s | Biome Type: %s\n", gridID, g.Biome)
			} else {
				fmt.Printf("[DEBUG-BIOME] Impossible de lire le biome pour la grille %s\n", gridID)
			}
		}

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
		fmt.Println("[CHEAT] Révélation instantanée de TOUTES les tuiles")
		app.Renderer.ClearAnimations()

		for _, gID := range app.World.GridOrder {
			if grid, ok := app.World.GetGrid(gID); ok {
				for _, tile := range grid.Plots {
					if len(tile.EntitiesID) == 0 {
						continue
					}
					topID := tile.EntitiesID[len(tile.EntitiesID)-1]
					if e, ok := app.World.Entities.Get(entity.ID(topID)); ok {
						if e.GetState()&entity.Hidden != 0 {
							e.SetState(entity.Revealed)
							app.Engine.TrackTileReveal(tile.Position)
						}
					}
				}
			}
		}
	}

	// F6: Cacher toutes les tuiles (cheat)
	app.Input.OnHideAll = func(gridID string) {
		fmt.Println("[CHEAT] Masquage instantané de TOUTES les tuiles")
		app.Renderer.ClearAnimations()

		for _, gID := range app.World.GridOrder {
			if grid, ok := app.World.GetGrid(gID); ok {
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
		}
	}

	// F7: Désceller les sorties (cheat)
	app.Input.OnUnlockNavigation = func(gridID string) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}
		cmd := &usecase.UnlockNavigationCommand{
			World:  app.World,
			GridID: gridID,
		}
		if err := cmd.Execute(); err == nil {
			fmt.Println("[CHEAT] Sorties déscellées pour la zone actuelle")
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
		flipDir, ok5 := e.Payload["flip_direction"].(entity.FlipDirection)

		if ok1 && ok3 && ok4 && ok5 {
			// Enregistre la révélation pour les triggers de créatures
			app.Engine.TrackTileReveal(board.Position{X: position.X, Y: position.Y})

			var entState entity.TileState
			var startTrans, endTrans entity.Transformation

			// CAS 1 : Entité réelle (Ressource, Créature, Piège, Structure)
			if ent, ok := app.World.Entities.Get(entity.ID(entityID)); ok {
				entState = ent.GetState()
				endTrans = ent.GetTransformation()
				applyTrans := flipDir.ToTransformation()
				startTrans = entity.Compose(endTrans, applyTrans)
			} else if strings.HasPrefix(entityID, "exit_") {
				// CAS 2 : Tuile de sortie (Navigation)
				// Format attendu: exit_north_0
				parts := strings.Split(entityID, "_")
				if len(parts) == 3 {
					dirName := parts[1]
					idx := 0
					if parts[2] == "1" {
						idx = 1
					}

					var dir entity.Direction
					switch dirName {
					case "north":
						dir = entity.DirNorth
					case "east":
						dir = entity.DirEast
					case "south":
						dir = entity.DirSouth
					case "west":
						dir = entity.DirWest
					}

					if grid, ok := app.World.GetGrid(gridID); ok {
						entState = grid.ExitsState[dir][idx]
						endTrans = grid.ExitsTransform[dir][idx]
						applyTrans := flipDir.ToTransformation()
						startTrans = entity.Compose(endTrans, applyTrans)
					}
				}
			}

			// Démarre l'animation de flip si on a récupéré les infos
			if entState != 0 {
				app.Renderer.StartFlipAnimation(
					gridID,
					board.Position{X: position.X, Y: position.Y},
					flipDir,
					entityID,
					entState,
					startTrans,
					endTrans,
				)
			}
		}
	})
}

// spawnInitialEntities crée quelques entités au démarrage sur différents grids
func (app *Application) spawnInitialEntities() {
	fmt.Println("[INIT] Populating dream plane...")
	for _, gridID := range app.World.GridOrder {
		grid, ok := app.World.GetGrid(gridID)
		if !ok {
			continue
		}

		isStartZone := gridID == app.World.DreamPlane.StartZoneID
		isEndZone := gridID == app.World.DreamPlane.EndZoneID
		isPortalZone := isStartZone || isEndZone

		if isStartZone {
			fmt.Printf("[INIT] Zone de DÉPART (%s) détectée.\n", gridID)
		} else if isEndZone {
			fmt.Printf("[INIT] Zone de FIN (%s) détectée.\n", gridID)
		}

		// 1. Spawner les structures marquées par le générateur
		for pos, plot := range grid.Plots {
			if plot.StructureID != "" {
				stype := "unknown"
				if plot.StructureID == "commencement_portal" || plot.StructureID == "finish_portal" {
					stype = plot.StructureID
				} else if strings.HasPrefix(plot.StructureID, "struct_") {
					parts := strings.Split(plot.StructureID, "_")
					if len(parts) >= 2 {
						stype = parts[1]
					}
				}

				if stype != "unknown" {
					_, err := app.World.SpawnStructure(gridID, stype, entity.Position{X: pos.X, Y: pos.Y})
					if err == nil && isPortalZone {
						fmt.Printf("  - [%s] Structure créée : %s en (%d, %d)\n", gridID, stype, pos.X, pos.Y)
					}
				}
			}
		}

		// 2. Remplissage aléatoire du reste (seulement si ce n'est pas une zone de portail)
		if !isPortalZone {
			app.FillGridRandomly(gridID)
		}
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

// updateMenu gère l'écran titre et le menu de sauvegarde
func (app *Application) updateMenu() error {
	if app.SaveMenu.IsVisible() {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			app.SaveMenu.SetVisible(false)
			return nil
		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			action := app.SaveMenu.HandleClick(x, y)

			switch action.Type {
			case renderer.ActionBack:
				app.SaveMenu.SetVisible(false)
			case renderer.ActionLoad, renderer.ActionNew:
				app.StartGameWithSlot(action.Slot)
			case renderer.ActionDelete:
				_ = app.Persistence.DeleteSave(action.Slot)
				metas, _ := app.Persistence.GetSaveSummaries()
				app.SaveMenu.UpdateMetas(metas)
				app.checkSaves()
			}
		}
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if app.TitleScreen.IsStartButtonClicked(x, y) {
			if app.hasSaves {
				// Charger la dernière partie directement (Bouton "Continuer")
				app.StartGameWithSlot(0)
			} else {
				// Première fois : ouvrir la sélection
				metas, _ := app.Persistence.GetSaveSummaries()
				app.SaveMenu.UpdateMetas(metas)
				app.SaveMenu.SetVisible(true)
			}
		} else if app.TitleScreen.IsPlaytestButtonClicked(x, y) {
			app.StartPlaytestGame()
		} else if app.hasSaves && app.TitleScreen.IsProfileButtonClicked(x, y) {
			metas, _ := app.Persistence.GetSaveSummaries()
			app.SaveMenu.UpdateMetas(metas)
			app.SaveMenu.SetVisible(true)
		}
	}
	return nil
}

// updatePlaying gère le jeu en cours
func (app *Application) updatePlaying() error {
	// Vérification Victoire
	if app.HUD.IsVictoryVisible() {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			app.HUD.HideVictory()
			app.ReturnToMenu()
			return nil
		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			action := app.HUD.HandleVictoryClick(mx, my)
			switch action {
			case "replay":
				app.HUD.HideVictory()
				app.StartGameWithSlot(0)
			case "menu":
				app.HUD.HideVictory()
				app.ReturnToMenu()
			}
		}
		return nil
	}

	// Met à jour les processus temps réel (comme les timers de preview)
	app.Engine.UpdateFrame()

	// Mise à jour du compte à rebours temps réel (60 fps fixe)
	if app.World.TurnTimer != nil {
		dt := 1.0 / 60.0

		// 1. ARRÊT DU TIMER : zones de départ et de fin
		isPortalZone := app.World.DreamPlane != nil && (app.World.CurrentGridID == app.World.DreamPlane.StartZoneID || app.World.CurrentGridID == app.World.DreamPlane.EndZoneID)
		if isPortalZone {
			dt = 0 // Stoppe l'écoulement
		}

		// 2. RALENTISSEMENT : pendant la prévisualisation (mode portail portable)
		if app.Input.IsPortablePortalMode() {
			dt /= 5.0 // 5x plus lent
		}

		// 3. RALENTISSEMENT : pendant les animations de déplacement (UI)
		if len(app.World.Components.QueryByComponent("moving_animation")) > 0 {
			// Ralentit le timer pour que les animations soient visibles (x4 slower)
			dt /= 4.0
		}

		expired := app.World.TurnTimer.Update(dt)
		if expired {
			fmt.Println("[TIMER] Temps écoulé ! Auto-skip forcé.")
			// Simule un Skip : recache les tuiles et consomme le tour
			app.Input.ResetTimerSkip()
			app.World.TurnTimer.Reset()
		}
	}

	// Mise à jour de l'HUD (animations, timers)
	app.HUD.Update()

	// Traite les événements en attente
	app.World.EventBus.ProcessQueue()

	// Vérification de la mort
	if !app.World.Player.IsAlive() || app.World.Player.Stats.Sanity <= 0 || app.World.Player.Stats.Mana < 0 {
		fmt.Println("[STATE] GAME OVER - Statistiques épuisées")

		// Logique de mort persistante
		diff := string(app.World.Difficulty.Level)
		if err := app.Persistence.HandleDeath(app.World.Hub, app.World.Player, diff); err != nil {
			fmt.Printf("[SAVE] Erreur lors de la mise à jour du compteur de décès : %v\n", err)
		}

		app.State = domain.StateGameOver
	}

	// Gère les entrées HUD (ex: fermer fenêtre détails)
	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// On mémorise l'index sélectionné AVANT le clic pour détecter le double-clic (usage)
		prevSelectedIdx := app.HUD.GetSelectedLootIndex()

		if app.HUD.HandleClick(mx, my) {
			newSelectedIdx := app.HUD.GetSelectedLootIndex()

			app.Input.SetPortablePortalMode(app.HUD.IsPortablePortalSelected())

			// Si on a un double-clic valide sur un objet de l'inventaire
			if prevSelectedIdx == newSelectedIdx && newSelectedIdx != -1 {
				selectedItem := app.HUD.GetSelectedLootItem()

				if selectedItem != nil {
					// 1. FACTORISATION : On cherche l'index réel dans l'inventaire une seule fois
					inventoryIdx := -1
					for i, item := range app.World.Player.Inventory.Items {
						if item.ID == selectedItem.ID {
							inventoryIdx = i
							break
						}
					}

					// 2. DISPATCHER DE COMMANDES
					if inventoryIdx >= 0 {
						var cmd interface{ Execute() error } // Interface temporaire pour lier nos commandes

						switch selectedItem.Name {
						case "echo_hound":
							cmd = &usecase.UseScannerItemCommand{
								World:     app.World,
								GridID:    app.World.CurrentGridID,
								ItemIndex: inventoryIdx,
							}

						case "dreamberry":
							cmd = &usecase.UseDreamberryItemCommand{
								World:     app.World,
								ItemIndex: inventoryIdx,
							}
						}

						// 3. EXÉCUTION DE LA COMMANDE
						if cmd != nil {
							if err := cmd.Execute(); err != nil {
								fmt.Printf("[ERROR] Échec de l'utilisation de %s : %v\n", selectedItem.Name, err)
							} else {
								fmt.Printf("[SUCCESS] %s utilisé avec succès.\n", selectedItem.Name)
								app.HUD.ClearActiveLootSelection()
							}
						}
					}
				}
			}
			return nil
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if app.HUD.HandleRightClick(mx, my) {
			return nil
		}
	}

	// Gère le scroll HUD
	app.HUD.HandleScroll(mx, my)

	// Gère les entrées
	return app.Input.Update()
}

// updateGameOver gère l'écran de fin
func (app *Application) updateGameOver() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		app.ReturnToMenu()
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		action := app.HUD.HandleGameOverClick(mx, my)
		switch action {
		case "replay":
			app.StartGameWithSlot(0)
		case "menu":
			app.ReturnToMenu()
		}
	}
	return nil
}

// StartGameWithSlot démarre le jeu avec un slot spécifique
func (app *Application) StartGameWithSlot(slotID int) {
	fmt.Printf("[SAVE] Starting game with slot %d\n", slotID)

	// Tentative de chargement ou création
	var save *domain.SaveData
	var err error

	if slotID == 0 {
		save, err = app.Persistence.LoadLatestGame()
	} else {
		save, err = app.Persistence.LoadGame(slotID)
		if err != nil {
			fmt.Printf("[SAVE] No save found for slot %d, creating new game\n", slotID)
			save, _ = app.Persistence.CreateNewGame(slotID)
		}
	}

	if save != nil {
		// Mise à jour des stats de session
		save.Meta.SessionCount++
		app.sessionStartTime = time.Now()

		// Chaque expédition démarre avec un nouvel état de joueur et de hub.
		app.World.Hub = meta.NewHub()
		app.World.Player = player.New(save.Player.ID)

		// Applique la difficulté sauvegardée
		if save.Meta.Difficulty != "" {
			app.World.Difficulty = meta.GetSettings(meta.DifficultyLevel(save.Meta.Difficulty))
		}

		// Régénère le monde à chaque nouveau départ (même après Game Over)
		app.World.GenerateLayout("dream_plane_1")

		app.StartGame()
	}
}

// StartPlaytestGame démarre une session de test dense
func (app *Application) StartPlaytestGame() {
	fmt.Println("[PLAYTEST] Starting playtest session")
	app.sessionStartTime = time.Now()

	app.World.Hub = meta.NewHub()
	app.World.Player = player.New("playtest_player")
	app.World.Difficulty = meta.GetSettings(meta.LevelPlaytest)

	// Utilise le layout de test
	app.World.GeneratePlaytestLayout("playtest_plane")

	app.StartGame()
}

// StartGame démarre le jeu depuis le menu (logique commune)
func (app *Application) StartGame() {
	oldState := app.State
	app.State = domain.StatePlaying
	app.SaveMenu.SetVisible(false)

	// Publie l'événement de changement de phase
	app.World.EventBus.Publish(domain.NewPhaseChangedEvent(oldState, app.State))

	fmt.Printf("[STATE] Transition: %s -> %s\n", oldState, app.State)

	// Réinitialise le monde (tour, etc.)
	app.World.Turn = 0
	app.World.MaxTurns = app.World.Player.Stats.MaxSanity

	// V0.2: L'inventaire est vidé à chaque nouvelle partie
	app.World.Player.Inventory.Items = make([]*player.LootItem, 0, app.World.Player.Inventory.MaxSize)
	_ = app.World.Player.Inventory.AddItem(player.NewPortablePortalItem())
	app.World.Player.Inventory.ScrollOffset = 0

	// Démarre l'engine (nécessaire pour les mouvements des créatures)
	app.Engine.Start()
	fmt.Println("[ENGINE] Started")

	// Démarre le compte à rebours temps réel avec la durée de la difficulté courante
	if app.World.TurnTimer != nil {
		app.World.TurnTimer.SetMaxTime(app.World.Difficulty.TurnTimerDuration)
		app.World.TurnTimer.Reset()
		fmt.Printf("[TIMER] Compte à rebours démarré : %.1fs\n", app.World.Difficulty.TurnTimerDuration)
	}

	// Spawn les entités initiales si nécessaire
	if app.World.Entities.Count() == 0 {
		fmt.Println("=== Spawning initial entities ===")
		app.spawnInitialEntities()
		fmt.Printf("=== Total entities: %d ===\n", app.World.Entities.Count())
	}

	// DÉCLENCHEMENT DU PREVIEW : On active la grille actuelle au démarrage réel du jeu.
	// Cela force le PreviewSystem à s'activer sur une grille maintenant peuplée.
	if app.World.CurrentGridID != "" {
		app.World.SetCurrentGrid(app.World.CurrentGridID)
	}
}

// ReturnToMenu retourne au menu principal
func (app *Application) ReturnToMenu() {
	// Cache les fenêtres de fin
	app.HUD.HideVictory()

	// Arrête le timer temps réel
	if app.World.TurnTimer != nil {
		app.World.TurnTimer.Stop()
	}

	// Sauvegarde de la progression avant de quitter
	if app.State == domain.StatePlaying {
		duration := time.Since(app.sessionStartTime).Seconds()
		diff := string(app.World.Difficulty.Level)
		if err := app.Persistence.SaveCurrentGame(app.World.Hub, app.World.Player, diff, duration); err != nil {
			fmt.Printf("[SAVE] Erreur lors de la sauvegarde auto : %v\n", err)
		}
	}

	oldState := app.State
	app.State = domain.StateMenu
	app.checkSaves()

	// Publie l'événement de changement de phase
	app.World.EventBus.Publish(domain.NewPhaseChangedEvent(oldState, app.State))

	fmt.Printf("[STATE] Transition: %s -> %s\n", oldState, app.State)

	// Réinitialise l'état du jeu
	app.Input.ResetGameState()

	// Réinitialise la rotation du plateau
	app.Renderer.SetBoardRotation(0)

	// Vide les boards pour la prochaine partie
	for _, gridID := range app.World.GridOrder {
		cmd := &usecase.ClearBoardCommand{World: app.World, GridID: gridID}
		cmd.Execute()
	}

	// Vide l'inventaire
	app.World.Player.Inventory.Items = make([]*player.LootItem, 0, app.World.Player.Inventory.MaxSize)

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
	app.TitleScreen.Render(screen, app.hasSaves)

	if app.hasSaves {
		text.Draw(screen, "[ CHANGER DE PROFIL ]", basicfont.Face7x13, 335, 430, color.RGBA{150, 150, 255, 255})
	}

	if app.SaveMenu.IsVisible() {
		app.SaveMenu.Render(screen)
	}
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

	winW, winH := 600, 400
	x := (ui.ScreenWidth - winW) / 2
	y := (ui.ScreenHeight - winH) / 2

	vector.DrawFilledRect(screen, float32(x), float32(y), float32(winW), float32(winH), color.RGBA{40, 20, 20, 255}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(winW), float32(winH), 3, color.RGBA{255, 100, 100, 255}, true)

	text.Draw(screen, "GAME OVER", basicfont.Face7x13, x+250, y+50, color.RGBA{255, 100, 100, 255})
	text.Draw(screen, "Statistiques épuisées. Votre voyage s'arrête ici.", basicfont.Face7x13, x+120, y+100, color.White)

	// Boutons
	btnW, btnH := 160, 40

	// Bouton REJOUER
	bx1 := x + 100
	by := y + 300
	vector.DrawFilledRect(screen, float32(bx1), float32(by), float32(btnW), float32(btnH), color.RGBA{80, 40, 40, 255}, true)
	vector.StrokeRect(screen, float32(bx1), float32(by), float32(btnW), float32(btnH), 1, color.White, true)
	text.Draw(screen, "REJOUER", basicfont.Face7x13, bx1+50, by+25, color.White)

	// Bouton MENU
	bx2 := x + 340
	vector.DrawFilledRect(screen, float32(bx2), float32(by), float32(btnW), float32(btnH), color.RGBA{40, 40, 40, 255}, true)
	vector.StrokeRect(screen, float32(bx2), float32(by), float32(btnW), float32(btnH), 1, color.White, true)
	text.Draw(screen, "MENU", basicfont.Face7x13, bx2+60, by+25, color.White)
}

// drawVictory dessine l'écran de victoire de la version 0.2
func (app *Application) drawVictory(screen *ebiten.Image) {
	screen.Fill(color.Black)
	text.Draw(screen, "Victory", basicfont.Face7x13, 350, 300, color.White)
	text.Draw(screen, "Chemin vers l'enveloppe corporel introuvable. Appuyez sur Echap pour recommencer.", basicfont.Face7x13, 200, 350, color.Gray{180})
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

	// En jeu, utilise la taille fixe définie dans l'issue
	if app.State == domain.StatePlaying {
		return ui.ScreenWidth, ui.ScreenHeight
	}

	return 1100, 600
}
