// Package app orchestre les composants de haut niveau.
// C'est le "wiring" (câblage) principal de l'application.
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

// Application est le conteneur principal du cycle de vie du jeu.
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
	Renderer       *renderer.BoardRenderer
	EffectRenderer *renderer.EffectRenderer
	TitleScreen    *renderer.TitleScreen
	SaveMenu    *renderer.SaveMenu
	Input       *input.Handler
	HUD         *hud.HUD

	// Game State
	State domain.GameState

	// Session tracking
	sessionStartTime time.Time
	hasSaves         bool
	randSource       *rand.Rand

	// Debug
	debug *DebugStats
}

// NewApplication crée, injecte les dépendances et configure l'application.
func NewApplication() (*Application, error) {
	app := &Application{
		randSource: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// 1. Charge la configuration générale
	app.Config = loader.DefaultConfig()

	// 2. Initialise la logique du Domaine
	app.World = domain.NewWorld()
	app.AssocEngine = domain.NewAssocEngine()
	app.Engine = domain.NewEngine(app.World)

	// 3. Crée et interconnecte les grilles de zones
	app.setupGrids()

	// 4. Couche Infrastructure & Persistance (dossier ./saves)
	app.Assets = assets.NewManager()
	repo := infraPersistence.NewJsonRepository("./saves")
	app.Persistence = usecase.NewPersistenceManager(repo)

	// 5. Initialisation des composants UI et Rendering
	app.Renderer = renderer.NewBoardRenderer(app.Assets)
	app.EffectRenderer, _ = renderer.NewEffectRenderer()
	app.TitleScreen = renderer.NewTitleScreen()
	app.SaveMenu = renderer.NewSaveMenu()
	app.Input = input.NewHandler(app.World, app.AssocEngine)
	app.HUD = hud.NewHUD(app.World)
	app.HUD.SetAssetsManager(app.Assets)

	// Inscription du renderer aux événements de scan
	app.Renderer.SubscribeToEvents(app.World)

	// 5.1 Configuration du gestionnaire réactif des boutons d'action
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

	// 5.2 Feedback en temps réel sur le HUD (Pulsation de la jauge de Sanité)
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

	// 6. Connexion et liaisons des événements de l'UI
	app.Input.SetRenderer(app.Renderer)
	app.HUD.SetBoardRenderer(app.Renderer)
	app.Input.OnToggleDetails = app.HUD.ToggleDetails
	app.Input.OnToggleInvDetails = app.HUD.ToggleInventoryDetails
	app.Input.OnToggleAssetsDetails = app.HUD.ToggleAssetsDetails

	// Remplissage de débogage pour tester l'utilisation des objets
	app.Input.OnFillInventory = func() {
		fmt.Println("[DEBUG] Remplissage de l'inventaire avec divers objets.")
		items := []*player.LootItem{
			player.NewCrystalShardItem(),
			player.NewDreamberryItem(),
			player.NewMoonstoneItem(),
			player.NewWhisperingHerbItem(),
			player.NewBurrowerItem(),
			player.NewEchoHoundItem(),
			player.NewMossMonkeyItem(),
			player.NewShadowstalkerItem(),
			player.NewSpecterItem(),
			player.NewStonewardenItem(),

		}
		for _, item := range items {
			_ = app.World.AddLootItem(item)
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

			// Démarre le compte à rebours de victoire fixe de 5 secondes
			app.Input.StartVictoryTimer(5.0)
		}
	}

	app.Input.OnVictory = func() {
		app.HUD.ShowVictory()
	}

	// 7. Configuration des Callbacks de commandes de jeu
	app.setupCallbacks()

	// 8. Liaison des abonnements événementiels pour les animations visuelles
	app.setupEventSubscriptions()

	// 9. Initialisation des métriques de Débogage
	app.debug = NewDebugStats()

	// 10. Fixation de l'état initial : Affichage du Menu Principal
	app.State = domain.StateMenu
	app.checkSaves()
	fmt.Println("[STATE] État initial : MENU")

	return app, nil
}

func (app *Application) checkSaves() {
	metas, _ := app.Persistence.GetSaveSummaries()
	app.hasSaves = len(metas) > 0
	if app.hasSaves {
		app.TitleScreen.ButtonText = "CONTINUER"
	} else {
		app.TitleScreen.ButtonText = "DEMARRER"
	}
}

// setupGrids orchestre le layout initial via le générateur procédural.
func (app *Application) setupGrids() {
	app.World.GenerateLayout("dream_plane_1")
	fmt.Printf("Generated Dream Plane with %d zones\n", len(app.World.Grids))

	// Définit la zone de commencement comme grille active par défaut
	if app.World.DreamPlane != nil {
		app.World.CurrentGridID = app.World.DreamPlane.StartZoneID
	}
}

// setupCallbacks connecte les actions de l'Input Handler aux Use Cases sous-jacents.
func (app *Application) setupCallbacks() {
	// Fin de tour normale
	app.Input.OnTurnEnd = func() {
		fmt.Println("[ACTION] Turn ended")
		app.debug.Action()
		app.Engine.Update()
	}

	// Spawn et remplissage aléatoire complet d'une grille
	app.Input.OnSpawnEntities = func(gridID string) {
		fmt.Printf("[ACTION] Random fill button pressed on grid %s\n", gridID)
		app.debug.Action()

		if gridID == "" {
			gridID = app.World.CurrentGridID
		}
		app.World.FillGridRandomly(gridID)
	}

	// Spawn de toutes les variétés de créatures (Shift+S)
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

	// Spawn d'une créature aléatoire (F9)
	app.Input.OnSpawnRandomCreature = func(gridID string) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		creatures := []string{"lumifly", "shadowstalker", "burrower", "specter", "echo_hound", "fleeing_sprite", "moss_monkey"}
		species := creatures[app.randSource.Intn(len(creatures))]

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

	// Nettoyage complet du plateau de jeu
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

	// Navigation et changement de zone (Grille active)
	app.Input.OnSwitchGrid = func(gridID string) {
		fmt.Printf("[ACTION] Switching to grid %s\n", gridID)

		if app.World != nil {
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

	app.Input.OnRotateBoard = func(delta float64) {
		app.Renderer.RotateBoard(delta)
	}

	app.Input.OnResetRotation = func() {
		app.Renderer.SetBoardRotation(0)
	}

	app.Input.OnExitToMenu = func() {
		app.ReturnToMenu()
	}

	app.setupDebugCallbacks()
}

// setupDebugCallbacks configure les raccourcis et commandes de triche/débogage.
func (app *Application) setupDebugCallbacks() {
	// F3 : Forcer le passage au prochain tour
	app.Input.OnForceTurn = func() {
		fmt.Println("[DEBUG] Forcing turn end")
		app.Engine.Update()
	}

	// F5 : Révéler instantanément toutes les tuiles cachées du monde
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

	// F6 : Masquer instantanément toutes les tuiles du monde
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

	// F7 : Désceller les verrous de navigation de la zone actuelle
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

	// F8 : Basculer l'état bloqué de toutes les tuiles de la zone
	app.Input.OnClearBlocked = func(gridID string) {
		if gridID == "" {
			gridID = app.World.CurrentGridID
		}

		if grid, ok := app.World.GetGrid(gridID); ok {
			count := 0
			shouldBlock := true
			first := true

			for _, tile := range grid.Plots {
				for _, id := range tile.EntitiesID {
					if ent, ok := app.World.Entities.Get(entity.ID(id)); ok {
						if first {
							shouldBlock = (ent.GetState()&entity.Blocked == 0)
							first = false
						}
						state := ent.GetState()
						if shouldBlock {
							ent.SetState(state | entity.Blocked)
						} else {
							ent.SetState(state & ^entity.Blocked)
						}
						count++
					}
				}
			}
			action := "appliqué à"
			if !shouldBlock {
				action = "retiré de"
			}
			fmt.Printf("[CHEAT] État bloqué %s %d tuile(s) dans la zone %s\n", action, count, gridID)
		}
	}

	// F10 : Activer/Désactiver la boucle d'automatisation des mouvements de l'Engine
	app.Input.OnToggleAutoMove = func() {
		app.Engine.Running = !app.Engine.Running
		if app.Engine.Running {
			fmt.Println("[DEBUG] Mouvement automatique : ON")
		} else {
			fmt.Println("[DEBUG] Mouvement automatique : OFF")
		}
	}
}

// setupEventSubscriptions connecte l'EventBus aux animations graphiques du Renderer.
func (app *Application) setupEventSubscriptions() {
	app.World.EventBus.SubscribeFunc(event.TileRevealed, func(e event.Event) {
		position, ok1 := e.Payload["position"].(entity.Position)
		entityID, ok3 := e.Payload["entity_id"].(string)
		gridID, ok4 := e.Payload["grid_id"].(string)
		flipDir, ok5 := e.Payload["flip_direction"].(entity.FlipDirection)

		if ok1 && ok3 && ok4 && ok5 {
			app.Engine.TrackTileReveal(board.Position{X: position.X, Y: position.Y})

			var entState entity.TileState
			var startTrans, endTrans entity.Transformation

			if ent, ok := app.World.Entities.Get(entity.ID(entityID)); ok {
				entState = ent.GetState()
				endTrans = ent.GetTransformation()
				applyTrans := flipDir.ToTransformation()
				startTrans = entity.Compose(endTrans, applyTrans)
			} else if strings.HasPrefix(entityID, "exit_") {
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

// spawnInitialEntities injecte les dolmens, portails et génère le contenu des autres grilles.
func (app *Application) spawnInitialEntities() {
	fmt.Println("=== Spawning initial entities ===")
	app.World.PopulateInitialStructures()

	for _, gridID := range app.World.GridOrder {
		isPortalZone := app.World.DreamPlane != nil && (gridID == app.World.DreamPlane.StartZoneID || gridID == app.World.DreamPlane.EndZoneID)
		if !isPortalZone {
			app.World.FillGridRandomly(gridID)
		}
	}
}

// Update met à jour la logique de l'application selon son état global (Pattern State).
func (app *Application) Update() error {
	app.debug.Frame()

	if app.EffectRenderer != nil {
		app.EffectRenderer.Update()
	}

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

// updateMenu orchestre la navigation de l'écran titre et de la sélection de sauvegarde.
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
				app.StartGameWithSlot(0)
			} else {
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

// updatePlaying régit le cycle d'exécution du gameplay actif (Timers, inventaire, victoires).
func (app *Application) updatePlaying() error {
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

	app.Engine.UpdateFrame()

	if app.World.TurnTimer != nil {
		dt := 1.0 / 60.0

		isPortalZone := app.World.DreamPlane != nil && (app.World.CurrentGridID == app.World.DreamPlane.StartZoneID || app.World.CurrentGridID == app.World.DreamPlane.EndZoneID)
		if isPortalZone {
			dt = 0
		}

		if app.Input.IsPortablePortalMode() {
			dt /= 5.0
		}

		if len(app.World.Components.QueryByComponent("moving_animation")) > 0 {
			dt /= 4.0
		}

		if app.World.TurnTimer.Update(dt) {
			fmt.Println("[TIMER] Temps écoulé ! Auto-skip forcé.")
			app.Input.ResetTimerSkip()
			app.World.TurnTimer.Reset()
		}
	}

	app.HUD.Update()
	app.World.EventBus.ProcessQueue()

	if !app.World.Player.IsAlive() || app.World.Player.Stats.Sanity <= 0 || app.World.Player.Stats.Mana < 0 {
		fmt.Println("[STATE] GAME OVER - Statistiques épuisées")

		diff := string(app.World.Difficulty.Level)
		if err := app.Persistence.HandleDeath(app.World.Hub, app.World.Player, diff); err != nil {
			fmt.Printf("[SAVE] Erreur lors de la mise à jour du compteur de décès : %v\n", err)
		}
		app.State = domain.StateGameOver
	}

	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		prevSelectedIdx := app.HUD.GetSelectedLootIndex()

		if app.HUD.HandleClick(mx, my) {
			newSelectedIdx := app.HUD.GetSelectedLootIndex()
			app.Input.SetPortablePortalMode(app.HUD.IsPortablePortalSelected())

			if prevSelectedIdx == newSelectedIdx && newSelectedIdx != -1 {
				selectedItem := app.HUD.GetSelectedLootItem()

				if selectedItem != nil {
					inventoryIdx := -1
					for i, item := range app.World.Player.Inventory.Items {
						if item.GetID() == selectedItem.GetID() {
							inventoryIdx = i
							break
						}
					}

					if inventoryIdx >= 0 {
						var cmd interface{ Execute() error }

						switch selectedItem.Name {
						case "echo_hound":
							cmd = &usecase.UseScannerItemCommand{
								World:     app.World,
								GridID:    app.World.CurrentGridID,
								ItemIndex: inventoryIdx,
							}
						case "dreamberry", "moonstone", "crystal_shard", "whispering_herb", "specter", "burrower", "shadowstalker", "stonewarden", "moss_monkey", "lumifly":
							cmd = &usecase.UseLootItemCommand{
								World:     app.World,
								ItemIndex: inventoryIdx,
							}
						}

						if cmd != nil {
							fmt.Printf("[ACTION] Tentative d'utilisation de %s...\n", selectedItem.Name)
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

	app.HUD.HandleScroll(mx, my)
	return app.Input.Update()
}

// updateGameOver gère les clics et interactions sur l'écran d'échec.
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

// StartGameWithSlot démarre ou charge une expédition sur un emplacement donné.
func (app *Application) StartGameWithSlot(slotID int) {
	fmt.Printf("[SAVE] Starting game with slot %d\n", slotID)
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
		save.Meta.SessionCount++
		app.sessionStartTime = time.Now()

		app.World.Hub = meta.NewHub()
		app.World.Player = player.New(save.Player.ID)

		if save.Meta.Difficulty != "" {
			app.World.Difficulty = meta.GetSettings(meta.DifficultyLevel(save.Meta.Difficulty))
		}

		app.World.GenerateLayout("dream_plane_1")
		app.StartGame()
	}
}

// StartPlaytestGame instancie un environnement de test isolé et accéléré.
func (app *Application) StartPlaytestGame() {
	fmt.Println("[PLAYTEST] Starting playtest session")
	app.sessionStartTime = time.Now()

	app.World.Hub = meta.NewHub()
	app.World.Player = player.New("playtest_player")
	app.World.Difficulty = meta.GetSettings(meta.LevelPlaytest)

	app.World.GeneratePlaytestLayout("playtest_plane")
	app.StartGame()
}

// StartGame configure l'état d'initialisation commun à tout type de lancement de partie.
func (app *Application) StartGame() {
	oldState := app.State
	app.State = domain.StatePlaying
	app.SaveMenu.SetVisible(false)

	app.World.EventBus.Publish(domain.NewPhaseChangedEvent(oldState, app.State))
	fmt.Printf("[STATE] Transition: %s -> %s\n", oldState, app.State)

	app.World.Turn = 0
	app.World.MaxTurns = app.World.Player.Stats.MaxSanity

	// Initialisation propre de l'inventaire via le World
	app.World.Player.Inventory.Items = make([]*player.LootItem, 0, app.World.Player.Inventory.MaxSize)
	_ = app.World.AddLootItem(player.NewPortablePortalItem())
	app.World.Player.Inventory.ScrollOffset = 0

	app.Engine.ResetPreviews()
	app.Engine.Start()
	fmt.Println("[ENGINE] Started")

	if app.World.TurnTimer != nil {
		app.World.TurnTimer.SetMaxTime(app.World.Difficulty.TurnTimerDuration)
		app.World.TurnTimer.Reset()
		fmt.Printf("[TIMER] Compte à rebours démarré : %.1fs\n", app.World.Difficulty.TurnTimerDuration)
	}

	if app.World.Entities.Count() == 0 {
		fmt.Println("=== Spawning initial entities ===")
		app.spawnInitialEntities()
		fmt.Printf("=== Total entities: %d ===\n", app.World.Entities.Count())
	}

	if app.World.CurrentGridID != "" {
		app.World.SetCurrentGrid(app.World.CurrentGridID)
	}
}

// ReturnToMenu quitte la session active, sauvegarde l'état persistant et vide la mémoire temporaire.
func (app *Application) ReturnToMenu() {
	app.HUD.HideVictory()

	if app.World.TurnTimer != nil {
		app.World.TurnTimer.Stop()
	}

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

	app.World.EventBus.Publish(domain.NewPhaseChangedEvent(oldState, app.State))
	fmt.Printf("[STATE] Transition: %s -> %s\n", oldState, app.State)

	app.Input.ResetGameState()
	app.Renderer.SetBoardRotation(0)

	for _, gridID := range app.World.GridOrder {
		cmd := &usecase.ClearBoardCommand{World: app.World, GridID: gridID}
		cmd.Execute()
	}

	app.World.Player.Inventory.Items = make([]*player.LootItem, 0, app.World.Player.Inventory.MaxSize)
	fmt.Println("[MENU] Retour au menu principal")
}

// Draw distribue l'appel d'affichage graphique Ebitengine vers l'écran adéquat.
func (app *Application) Draw(screen *ebiten.Image) {
	switch app.State {
	case domain.StateMenu:
		app.drawMenu(screen)
	case domain.StatePlaying:
		app.drawPlaying(screen)
	case domain.StateGameOver:
		app.drawGameOver(screen)
	}
}

// drawMenu affiche l'écran d'accueil principal.
func (app *Application) drawMenu(screen *ebiten.Image) {
	app.TitleScreen.Render(screen, app.hasSaves)

	if app.hasSaves {
		text.Draw(screen, "[ CHANGER DE PROFIL ]", basicfont.Face7x13, 335, 430, color.RGBA{150, 150, 255, 255})
	}

	if app.SaveMenu.IsVisible() {
		app.SaveMenu.Render(screen)
	}
}

// drawPlaying gère le rendu complet de l'arène de jeu, des calques d'inputs et du HUD.
func (app *Application) drawPlaying(screen *ebiten.Image) {
	screen.Fill(color.Black)

	app.Renderer.Render(screen, app.World)
	app.Input.Draw(screen)
	app.HUD.Render(screen)

	// Application des shaders globaux (Biome + Attaques créatures + Sanité)
	if app.EffectRenderer != nil && app.World.Player != nil {
		ratio := float32(app.World.Player.Stats.Sanity) / float32(app.World.Player.Stats.MaxSanity)

		grid, _ := app.World.GetCurrentGrid()
		biome := ""
		if grid != nil {
			biome = string(grid.Biome)
		}

		mx, my := ebiten.CursorPosition()

		params := renderer.GlobalEffectParams{
			SanityRatio: ratio,
			Biome:       biome,
			UseBlur:     app.World.Player.VisualEffects["blur"] > 0,
			UseBubble:   app.World.Player.VisualEffects["bubble"] > 0,
			MousePos:    []float32{float32(mx) / float32(ui.ScreenWidth), float32(my) / float32(ui.ScreenHeight)},
		}
		app.EffectRenderer.ProcessGlobalEffects(screen, params)
	}

	if app.World.Entities.Count() == 0 {
		text.Draw(screen, "Appuyez sur S pour spawner des entites", basicfont.Face7x13, 200, 300, color.RGBA{255, 255, 0, 255})
	}
}

// drawGameOver dessine la fenêtre modale de défaite.
func (app *Application) drawGameOver(screen *ebiten.Image) {
	screen.Fill(color.Black)

	winW, winH := 600, 400
	x := (ui.ScreenWidth - winW) / 2
	y := (ui.ScreenHeight - winH) / 2

	vector.DrawFilledRect(screen, float32(x), float32(y), float32(winW), float32(winH), color.RGBA{40, 20, 20, 255}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(winW), float32(winH), 3, color.RGBA{255, 100, 100, 255}, true)

	text.Draw(screen, "GAME OVER", basicfont.Face7x13, x+250, y+50, color.RGBA{255, 100, 100, 255})
	text.Draw(screen, "Statistiques épuisées. Votre voyage s'arrête ici.", basicfont.Face7x13, x+120, y+100, color.White)

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

// findEmptyPosition localise aléatoirement une parcelle exempte d'obstacles et d'entités.
func (app *Application) findEmptyPosition(gridID string) *entity.Position {
	grid, ok := app.World.GetGrid(gridID)
	if !ok {
		return nil
	}

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

	pos := emptyPositions[app.randSource.Intn(len(emptyPositions))]
	return &pos
}

// Layout retourne la dimension d'affichage réclamée par Ebitengine en fonction du contexte applicatif.
func (app *Application) Layout(outsideWidth, outsideHeight int) (int, int) {
	if app.State == domain.StateMenu {
		return app.TitleScreen.Layout()
	}

	if app.State == domain.StatePlaying {
		return ui.ScreenWidth, ui.ScreenHeight
	}

	return 1100, 600
}