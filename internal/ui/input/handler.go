package input

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/actionbuttons"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/usecase"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Renderer interface {
	GetTileSize() int
	GetGridOffset() (int, int)
	ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool)
	ScreenToLocalTile(screenX, screenY int, world *domain.World) (localX, localY int, gridID string, ok bool)
	RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, color color.Color, world *domain.World)
	RenderPortalPlacementPreview(screen *ebiten.Image, center board.Position, gridID string, world *domain.World)
}

type Handler struct {
	world       *domain.World
	assocEngine *domain.AssocEngine
	renderer    Renderer

	selectedTile   *board.Position
	selectedGridID string

	portablePortalMode bool

	OnTurnEnd             func()
	OnSpawnEntities       func(gridID string)
	OnSpawnAllCreatures   func(gridID string) // Shift+S: Spawn toutes les créatures
	OnSpawnRandomCreature func(gridID string) // F9: Spawn créature aléatoire
	OnClearBoard          func(gridID string)
	OnSwitchGrid          func(gridID string)
	OnRotateBoard         func(delta float64)                        // Callback pour la rotation du plateau
	OnResetRotation       func()                                     // Callback pour réinitialiser la rotation
	OnExitToMenu          func()                                     // Callback pour retourner au menu
	OnToggleDetails       func()                                     // Callback pour afficher les détails
	OnToggleInvDetails    func()                                     // Callback pour afficher l'inventaire détaillé
	OnFillInventory       func()                                     // Callback pour remplir l'inventaire (debug)
	OnRevealAll           func(gridID string)                        // F5: Cheat - révéler tout
	OnHideAll             func(gridID string)                        // F6: Cheat - cacher tout
	OnUnlockNavigation    func(gridID string)                        // F7: Cheat - désceller sorties
	OnUsePortablePortal   func(gridID string, center board.Position) // P / grid placement: Déployer le portail portable
	OnForceTurn           func()                                     // F3: Forcer le prochain tour
	OnToggleAutoMove      func()                                     // F10: Toggle mouvement auto

	// Gestionnaire réactif des boutons d'action
	actionButtons *actionbuttons.Manager

	// Gestion du tour de jeu memory
	revealedTiles []board.Position // Liste des tuiles révélées ce tour
	isProcessing  bool             // Évite les clics pendant l'animation / verrouille la grille quand 2 tuiles sont retournées

	isTransitioning bool // Bloque les entrées pendant le changement de zone
	transitionTimer int  // Frames restantes pour le blocage
}

func NewHandler(world *domain.World, assocEng *domain.AssocEngine) *Handler {
	return &Handler{
		world:       world,
		assocEngine: assocEng,
	}
}

func (h *Handler) SetRenderer(r Renderer) {
	h.renderer = r
}

func (h *Handler) SetActionButtonsManager(m *actionbuttons.Manager) {
	h.actionButtons = m
}

func (h *Handler) Update() error {
	if h.isTransitioning {
		h.transitionTimer--
		if h.transitionTimer <= 0 {
			h.isTransitioning = false
		}
		return nil
	}

	if err := h.handleMouse(); err != nil {
		return err
	}
	h.handleKeyboard()
	return nil
}

func (h *Handler) Draw(screen *ebiten.Image) {
	if h.renderer == nil {
		return
	}
	h.renderHighlights(screen)
}

// getEntityInfo retourne une description texte de l'entité pour la console
func (h *Handler) getEntityInfo(ent entity.Entity) string {
	if ent == nil {
		return "Vide"
	}
	switch e := ent.(type) {
	case *domain.Creature:
		return fmt.Sprintf("Créature:%s", e.Species)
	case *domain.Resource:
		return fmt.Sprintf("Ressource:%s", e.ResourceType)
	default:
		return ent.GetType().String()
	}
}

func (h *Handler) handleMouse() error {
	// Clic droit : Désélection
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if h.selectedTile != nil {
			fmt.Printf("[SÉLECTION] Tuile en %v désélectionnée (clic droit)\n", *h.selectedTile)
			h.ClearSelection()
		}
		return nil
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	// Priorité : gestion des clics sur les boutons d'action (même si isProcessing)
	if h.actionButtons != nil {
		states := h.actionButtons.ComputeStates()
		x, y := ebiten.CursorPosition()
		if btnID, ok := h.actionButtons.HitTest(x, y, states); ok {
			h.handleActionButtonClick(btnID)
			return nil
		}
	}

	if h.isProcessing {
		fmt.Println("[INPUT] Traitement en cours, veuillez patienter...")
		return nil
	}

	// Gestion des clics sur les sorties (navigation zone par zone)
	if dir, index, ok := h.getClickedExit(); ok {
		// Vérifie si la sortie existe
		if _, hasExit := h.world.DreamPlane.GetConnectedZone(h.world.CurrentGridID, dir); !hasExit {
			return nil
		}

		// Vérifie si la navigation est ouverte
		if !h.world.IsNavigationOpen(h.world.CurrentGridID) {
			fmt.Println("[NAVIGATION] Sortie scellée. Trouvez plus de paires !")
			return nil
		}

		grid, _ := h.world.GetGrid(h.world.CurrentGridID)

		// État actuel des tuiles de cette sortie
		states := grid.ExitsState[dir]
		currentState := states[index]

		// 1. Si la sortie est déjà appairée, on change de zone
		if currentState&entity.Matched != 0 {
			cmd := &usecase.SwitchZoneCommand{World: h.world, Direction: dir}
			if err := cmd.Execute(); err == nil {
				fmt.Printf("[NAVIGATION] Passage vers le %v -> %s\n", dir, h.world.CurrentGridID)
				h.ClearSelection()
				h.isTransitioning = true
				h.transitionTimer = 30
			}
			return nil
		}

		// 2. Sinon, on révèle la tuile si elle est cachée
		if currentState&entity.Hidden != 0 {
			// Révélation de la première moitié
			states[index] = (currentState & ^entity.Hidden) | entity.Revealed
			fmt.Printf("[NAVIGATION] Tuile de sortie %v #%d révélée\n", dir, index)
			grid.ExitsState[dir] = states

			// 3. Vérification de l'association via le DOMAINE
			otherIndex := 1 - index
			if states[otherIndex]&entity.Revealed != 0 {
				dirName := "north"
				switch dir {
				case board.South:
					dirName = "south"
				case board.East:
					dirName = "east"
				case board.West:
					dirName = "west"
				}

				// On crée des wrappers Matchable pour le moteur d'association
				m1 := &exitMatchable{id: fmt.Sprintf("exit_%s_%d", dirName, index)}
				m2 := &exitMatchable{id: fmt.Sprintf("exit_%s_%d", dirName, otherIndex)}

				result, err := h.assocEngine.TryAssociate(m1, m2)
				if err == nil && result.Success {
					states[0] |= entity.Matched
					states[1] |= entity.Matched
					grid.ExitsState[dir] = states
					fmt.Printf("[NAVIGATION] %s (Moteur Domaine)\n", result.Message)
				}
			}
		}
		return nil
	}

	pos, gridID, ok := h.getHoveredTile()
	if !ok {
		return nil
	}

	if h.portablePortalMode && h.OnUsePortablePortal != nil {
		grid, _ := h.world.GetGrid(gridID)
		if grid != nil && h.isValidPortalPreviewPosition(grid, pos) {
			h.OnUsePortablePortal(gridID, pos)
			return nil
		}
		fmt.Println("[ACTION] Zone de déploiement invalide pour le portail portable")
		return nil
	}

	grid, _ := h.world.GetGrid(gridID)
	plot, err := grid.Get(pos)
	if err != nil || len(plot.EntitiesID) == 0 {
		return nil
	}

	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, hasEntity := h.world.Entities.Get(entity.ID(topID))
	if !hasEntity {
		return nil
	}

	state := ent.GetState()
	if state&entity.Hidden != 0 {
		// Vérifie si on a déjà révélé 2 tuiles ce tour
		if len(h.revealedTiles) >= 2 {
			fmt.Println("[INPUT] Déjà 2 tuiles révélées ce tour. Veuillez attendre la fin du traitement.")
			return nil
		}

		// Ne permet pas de révéler une tuile bloquée
		if state&entity.Blocked != 0 {
			fmt.Println("[INPUT] Cette tuile est scellée ou bloquée.")
			return nil
		}

		// Calcule la direction de flip basée sur la position du clic dans la tuile
		flipDir := h.calculateFlipDirection(gridID)

		cmd := &usecase.RevealTileCommand{
			World:         h.world,
			GridID:        gridID,
			Position:      pos,
			FlipDirection: flipDir,
		}
		if err := cmd.Execute(); err == nil {
			info := h.getEntityInfo(ent)
			num := len(h.revealedTiles) + 1
			fmt.Printf("[SÉLECTION] Choix #%d : Tuile révélée en %v sur %s -> %s\n", num, pos, gridID, info)
			h.revealedTiles = append(h.revealedTiles, pos)
			// Action volontaire : reset du compte à rebours temps réel
			if h.world.TurnTimer != nil {
				h.world.TurnTimer.Reset()
			}
		}

		// On met à jour le gridID pour la résolution du match
		h.selectedGridID = gridID

		// On sélectionne la tuile pour le match
		h.selectedTile = &pos

		// Si on a révélé 2 tuiles, verrouille la grille et active les boutons Match/Skip
		if len(h.revealedTiles) == 2 {
			h.isProcessing = true
			fmt.Println("[MATCH] 2 tuiles révélées. Grille verrouillée. Choisissez Match ou Skip.")
		}

	} else if state&entity.Revealed != 0 {
		if ent.GetType() == entity.TypeTrap {
			fmt.Printf("[ACTION] Suppression du piège en %v\n", pos)
			h.world.RemoveEntity(ent.GetID())
			for i, p := range h.revealedTiles {
				if p == pos {
					h.revealedTiles = append(h.revealedTiles[:i], h.revealedTiles[i+1:]...)
					break
				}
			}
			return nil
		}

		if h.selectedTile != nil && h.selectedGridID == gridID && *h.selectedTile == pos {
			fmt.Printf("[SÉLECTION] Tuile en %v désélectionnée\n", pos)
			h.ClearSelection()
		} else {
			info := h.getEntityInfo(ent)
			fmt.Printf("[SÉLECTION] Tuile en %v sur %s sélectionnée : %s\n", pos, gridID, info)
			h.selectedTile = &pos
			h.selectedGridID = gridID
		}
	}
	return nil
}

// handleActionButtonClick traite les clics sur les boutons d'action du Playmat.
func (h *Handler) handleActionButtonClick(btnID actionbuttons.ButtonID) {
	switch btnID {
	case actionbuttons.BtnMatch:
		fmt.Println("[ACTION] Bouton Match activé")
		h.processMatchAttempt()
		if h.world.TurnTimer != nil {
			h.world.TurnTimer.Reset()
		}
	case actionbuttons.BtnSkip:
		fmt.Println("[ACTION] Bouton Skip activé")
		h.processSkip()
		if h.world.TurnTimer != nil {
			h.world.TurnTimer.Reset()
		}
	case actionbuttons.BtnEndTurn:
		fmt.Println("[ACTION] Bouton End Turn activé")
		// Si des tuiles sont révélées mais non matchées, on les recache d'abord
		if len(h.revealedTiles) > 0 {
			h.hideRevealedTiles()
		}
		if h.world.TurnTimer != nil {
			h.world.TurnTimer.Reset()
		}
		if h.OnTurnEnd != nil {
			h.OnTurnEnd()
		}
	case actionbuttons.BtnMenu:
		fmt.Println("[ACTION] Bouton Menu activé")
		if h.OnExitToMenu != nil {
			h.OnExitToMenu()
		}
	}
}

// processSkip recache les tuiles révélées et termine le tour.
func (h *Handler) processSkip() {
	if len(h.revealedTiles) == 0 {
		h.isProcessing = false
		return
	}

	h.hideRevealedTiles()
	h.isProcessing = false
	h.ClearSelection()

	if h.OnTurnEnd != nil {
		h.OnTurnEnd()
	}
}

// hideRevealedTiles remet l'état Hidden sur toutes les tuiles de revealedTiles.
func (h *Handler) hideRevealedTiles() {
	gridID := h.selectedGridID
	if gridID == "" {
		gridID = h.world.CurrentGridID
	}
	grid, ok := h.world.GetGrid(gridID)
	if !ok {
		h.revealedTiles = nil
		return
	}

	for _, pos := range h.revealedTiles {
		plot, err := grid.Get(pos)
		if err != nil || len(plot.EntitiesID) == 0 {
			continue
		}
		topID := plot.EntitiesID[len(plot.EntitiesID)-1]
		if ent, ok := h.world.Entities.Get(entity.ID(topID)); ok {
			ent.SetState(entity.Hidden)
		}
	}
	h.revealedTiles = nil
}

// processMatchAttempt tente d'associer les 2 tuiles révélées
func (h *Handler) processMatchAttempt() {
	if len(h.revealedTiles) != 2 {
		h.isProcessing = false
		return
	}

	pos1 := h.revealedTiles[0]
	pos2 := h.revealedTiles[1]

	// SÉCURITÉ : Vérifie si le gridID est valide
	gridID := h.selectedGridID
	if gridID == "" {
		gridID = h.world.CurrentGridID
	}

	grid, ok := h.world.GetGrid(gridID)
	if !ok {
		fmt.Printf("[MATCH] Erreur : Grid %s non trouvé\n", gridID)
		h.revealedTiles = nil
		h.isProcessing = false
		return
	}

	tile1, _ := grid.Get(pos1)
	tile2, _ := grid.Get(pos2)

	if len(tile1.EntitiesID) == 0 || len(tile2.EntitiesID) == 0 {
		h.revealedTiles = nil
		h.isProcessing = false
		return
	}

	id1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	id2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
	e1, _ := h.world.Entities.Get(entity.ID(id1))
	e2, _ := h.world.Entities.Get(entity.ID(id2))

	if e1 == nil || e2 == nil {
		h.revealedTiles = nil
		h.isProcessing = false
		return
	}

	if e1.GetType() == entity.TypeTrap && e2.GetType() == entity.TypeTrap {
		fmt.Println("[MATCH] ✅ Deux pièges appairés ! Ils sont supprimés.")
		h.world.RemoveEntity(e1.GetID())
		h.world.RemoveEntity(e2.GetID())
		h.revealedTiles = nil
		h.isProcessing = false
		h.ClearSelection()
		if h.OnTurnEnd != nil {
			h.OnTurnEnd()
		}
		return
	}

	// CAS ÉCHEC : Un piège et autre chose (Ressource ou Créature)
	if e1.GetType() == entity.TypeTrap || e2.GetType() == entity.TypeTrap {
		fmt.Printf("[MATCH] ❌ Échec : %s ne peut pas être appairé avec un Piège.\n", h.getEntityInfo(e1))
		h.revealedTiles = nil
		h.isProcessing = false
		h.ClearSelection()

		// On recache les entités
		e1.SetState(entity.Hidden)
		e2.SetState(entity.Hidden)
		if h.OnTurnEnd != nil {
			h.OnTurnEnd()
		}
		return
	}

	fmt.Printf("[MATCH] Comparaison de la paire : %s vs %s\n", h.getEntityInfo(e1), h.getEntityInfo(e2))

	cmd := &usecase.MatchTilesCommand{
		World:    h.world,
		AssocEng: h.assocEngine,
		GridID:   gridID,
		Pos1:     pos1,
		Pos2:     pos2,
		OnSuccess: func() {
			fmt.Printf("[MATCH] ✅ Succès ! Paire de %s trouvée.\n", h.getEntityInfo(e1))
			h.revealedTiles = nil
			h.isProcessing = false
			h.ClearSelection()
			if h.OnTurnEnd != nil {
				h.OnTurnEnd()
			}
		},
		OnFailure: func() {
			fmt.Printf("[MATCH] ❌ Échec ! %s et %s ne correspondent pas.\n", h.getEntityInfo(e1), h.getEntityInfo(e2))
			h.revealedTiles = nil
			h.isProcessing = false
			h.ClearSelection()
			if h.OnTurnEnd != nil {
				h.OnTurnEnd()
			}
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Printf("[MATCH] %v\n", err)
		h.revealedTiles = nil
		h.isProcessing = false
	}
}

func (h *Handler) handleKeyboard() {
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		if h.OnToggleDetails != nil {
			h.OnToggleDetails()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		if h.OnToggleInvDetails != nil {
			h.OnToggleInvDetails()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if h.OnFillInventory != nil {
			h.OnFillInventory()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if h.OnUsePortablePortal != nil {
			center := board.Position{X: -1, Y: -1}
			if hovered, _, ok := h.getHoveredTile(); ok {
				center = hovered
			}
			h.OnUsePortablePortal(h.GetCurrentGridID(), center)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		if len(h.revealedTiles) == 2 {
			fmt.Println("[ACTION] Raccourci clavier : Match")
			h.processMatchAttempt()
			if h.world.TurnTimer != nil {
				h.world.TurnTimer.Reset()
			}
		} else {
			h.tryMatchSelected()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if len(h.revealedTiles) == 2 {
			fmt.Println("[ACTION] Raccourci clavier : Skip")
			h.processSkip()
			if h.world.TurnTimer != nil {
				h.world.TurnTimer.Reset()
			}
		} else if h.OnTurnEnd != nil {
			fmt.Println("[TOUR] Passage au tour suivant")
			if h.world.TurnTimer != nil {
				h.world.TurnTimer.Reset()
			}
			h.OnTurnEnd()
		}
	}

	// Navigation entre zones (Dream Plane)
	h.handleNavigationKeys()

	// Changement de difficulté (Touches F1, F2, F3, F4)
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		h.setDifficulty(meta.LevelEasy)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		h.setDifficulty(meta.LevelNormal)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		h.setDifficulty(meta.LevelHard)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		h.setDifficulty(meta.LevelInsane)
	}

	// S: Spawn entités de base, Shift+S: Spawn toutes les créatures
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if h.OnSpawnAllCreatures != nil {
				h.OnSpawnAllCreatures(h.GetCurrentGridID())
			}
		} else {
			if h.OnSpawnEntities != nil {
				h.OnSpawnEntities(h.GetCurrentGridID())
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		if h.OnSpawnRandomCreature != nil {
			h.OnSpawnRandomCreature(h.GetCurrentGridID())
		}
	}

	// F5: Cheat - révéler toutes les tuiles
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if h.OnRevealAll != nil {
			h.OnRevealAll(h.GetCurrentGridID())
		}
	}

	// F6: Cheat - cacher toutes les tuiles
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		if h.OnHideAll != nil {
			h.OnHideAll(h.GetCurrentGridID())
		}
	}

	// F7: Cheat - désceller les sorties
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		if h.OnUnlockNavigation != nil {
			h.OnUnlockNavigation(h.GetCurrentGridID())
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		fmt.Println("[KEY] C : Nettoyage du plateau")
		if h.OnClearBoard != nil {
			h.OnClearBoard(h.GetCurrentGridID())
		}
	}

	for i := 0; i < 9; i++ {
		key := ebiten.Key(i + int(ebiten.Key1))
		if inpututil.IsKeyJustPressed(key) {
			if i < len(h.world.GridOrder) {
				gridID := h.world.GridOrder[i]
				fmt.Printf("[INPUT] Changement de grille : -> %s\n", gridID)
				if h.OnSwitchGrid != nil {
					h.OnSwitchGrid(gridID)
					h.ClearSelection()
				}
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		fmt.Println("[INPUT] Abandon de la partie")
		if h.OnExitToMenu != nil {
			h.OnExitToMenu()
		} else {
			h.ClearSelection()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		fmt.Println("[ACTION] Réinitialisation de la rotation")
		if h.OnResetRotation != nil {
			h.OnResetRotation()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPEqual) {
		if h.OnRotateBoard != nil {
			h.OnRotateBoard(15)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		if h.OnRotateBoard != nil {
			h.OnRotateBoard(-15)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackslash) {
		fmt.Println("[KEY] \\ : Retour au menu")
		if h.OnExitToMenu != nil {
			h.OnExitToMenu()
		}
	}
}

func (h *Handler) handleNavigationKeys() {
	var dir board.Direction
	var pressed bool

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		dir = board.North
		pressed = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) && !ebiten.IsKeyPressed(ebiten.KeyShift) {
		dir = board.South
		pressed = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		dir = board.West
		pressed = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		dir = board.East
		pressed = true
	}

	if pressed {
		cmd := &usecase.SwitchZoneCommand{World: h.world, Direction: dir}
		if err := cmd.Execute(); err == nil {
			fmt.Printf("[NAVIGATION] Passage à la zone: %s\n", h.world.CurrentGridID)
			h.ClearSelection()
			h.isTransitioning = true
			h.transitionTimer = 30
		}
	}
}

func (h *Handler) setDifficulty(level meta.DifficultyLevel) {
	settings := meta.GetSettings(level)
	h.world.Difficulty = settings
	fmt.Printf("[DIFFICULTÉ] Niveau changé pour : %s\n", level)
	// Synchronise la durée du timer temps réel
	if h.world.TurnTimer != nil {
		h.world.TurnTimer.SetMaxTime(settings.TurnTimerDuration)
	}
	h.world.EventBus.PublishImmediate(domain.Event{
		Type:     domain.EventType("difficulty_changed"),
		SourceID: "player",
		Payload: map[string]interface{}{
			"level": string(level),
		},
	})
}

func (h *Handler) tryMatchSelected() {
	if h.selectedTile == nil {
		fmt.Println("[MATCH] Erreur : Aucune tuile sélectionnée")
		return
	}

	grid, ok := h.world.GetGrid(h.selectedGridID)
	if !ok {
		return
	}

	for _, plot := range grid.Plots {
		if plot.Position.X == h.selectedTile.X && plot.Position.Y == h.selectedTile.Y {
			continue
		}

		if len(plot.EntitiesID) == 0 {
			continue
		}

		topID := plot.EntitiesID[len(plot.EntitiesID)-1]
		ent, ok := h.world.Entities.Get(entity.ID(topID))
		if !ok {
			continue
		}

		if ent.GetState() == entity.Revealed {
			id1 := ""
			if tile1, err := grid.Get(*h.selectedTile); err == nil && len(tile1.EntitiesID) > 0 {
				id1 = tile1.EntitiesID[len(tile1.EntitiesID)-1]
			}
			e1, _ := h.world.Entities.Get(entity.ID(id1))

			fmt.Printf("[MATCH] Comparaison manuelle : %s vs %s\\n", h.getEntityInfo(e1), h.getEntityInfo(ent))

			cmd := &usecase.MatchTilesCommand{
				World:    h.world,
				AssocEng: h.assocEngine,
				GridID:   h.selectedGridID,
				Pos1:     *h.selectedTile,
				Pos2:     plot.Position,
				OnSuccess: func() {
					fmt.Println("[MATCH] ✅ Succès !")
					h.ClearSelection()
					if h.OnTurnEnd != nil {
						h.OnTurnEnd()
					}
				},
				OnFailure: func() {
					fmt.Println("[MATCH] ❌ Échec !")
					h.ClearSelection()
					if h.OnTurnEnd != nil {
						h.OnTurnEnd()
					}
				},
			}

			if err := cmd.Execute(); err != nil {
				fmt.Printf("[MATCH] Erreur : %v\n", err)
			}
			return
		}
	}
}

func (h *Handler) getHoveredTile() (board.Position, string, bool) {
	if h.renderer == nil {
		return board.Position{}, "", false
	}
	x, y := ebiten.CursorPosition()
	return h.renderer.ScreenToGrid(x, y, h.world)
}

func (h *Handler) getClickedExit() (board.Direction, int, bool) {
	x, y := ebiten.CursorPosition()
	// Coordonnées relatives au Playmat
	px := float64(x) - ui.PlaymatX
	py := float64(y) - ui.PlaymatY

	return h.checkExitClick(px, py)
}

func (h *Handler) checkExitClick(px, py float64) (board.Direction, int, bool) {
	if px >= ui.ExitNorthX && px < ui.ExitNorthX+ui.ExitNorthW && py >= ui.ExitNorthY && py < ui.ExitNorthY+ui.ExitNorthH {
		index := 0
		if px >= ui.ExitNorthX+ui.TileSize {
			index = 1
		}
		return board.North, index, true
	}
	if px >= ui.ExitEastX && px < ui.ExitEastX+ui.ExitEastW && py >= ui.ExitEastY && py < ui.ExitEastY+ui.ExitEastH {
		index := 0
		if py >= ui.ExitEastY+ui.TileSize {
			index = 1
		}
		return board.East, index, true
	}
	if px >= ui.ExitSouthX && px < ui.ExitSouthX+ui.ExitSouthW && py >= ui.ExitSouthY && py < ui.ExitSouthY+ui.ExitSouthH {
		index := 0
		if px >= ui.ExitSouthX+ui.TileSize {
			index = 1
		}
		return board.South, index, true
	}
	if px >= ui.ExitWestX && px < ui.ExitWestX+ui.ExitWestW && py >= ui.ExitWestY && py < ui.ExitWestY+ui.ExitWestH {
		index := 0
		if py >= ui.ExitWestY+ui.TileSize {
			index = 1
		}
		return board.West, index, true
	}
	return 0, 0, false
}

// calculateFlipDirection détermine la direction de flip basée sur la position du clic dans la tuile
func (h *Handler) calculateFlipDirection(gridID string) domain.FlipDirection {
	if h.renderer == nil {
		return usecase.DefaultFlipDirection
	}

	// Récupère la position locale du clic dans la tuile
	cursorX, cursorY := ebiten.CursorPosition()
	localX, localY, gID, ok := h.renderer.ScreenToLocalTile(cursorX, cursorY, h.world)
	if !ok || gID != gridID {
		return usecase.DefaultFlipDirection
	}

	tileSize := h.renderer.GetTileSize()
	return board.CalculateFlipDirection(tileSize, localX, localY)
}

func (h *Handler) renderHighlights(screen *ebiten.Image) {
	if hovered, gridID, ok := h.getHoveredTile(); ok {
		grid, ok := h.world.GetGrid(gridID)
		if !ok {
			return
		}

		tile, err := grid.Get(hovered)
		if err != nil {
			return
		}

		if len(tile.EntitiesID) == 0 {
			return
		}

		topID := tile.EntitiesID[len(tile.EntitiesID)-1]
		ent, ok := h.world.Entities.Get(entity.ID(topID))
		if !ok {
			return
		}

		var highlightColor color.Color
		state := ent.GetState()
		if state&entity.Hidden != 0 {
			highlightColor = color.RGBA{255, 255, 0, 100}
		} else if state&entity.Revealed != 0 {
			highlightColor = color.RGBA{0, 255, 255, 100}
		} else {
			highlightColor = color.RGBA{255, 255, 255, 50}
		}

		h.renderer.RenderSelectionHighlight(screen, hovered, gridID, highlightColor, h.world)
	}

	if h.portablePortalMode {
		if hovered, gridID, ok := h.getHoveredTile(); ok {
			grid, ok := h.world.GetGrid(gridID)
			if ok && h.isValidPortalPreviewPosition(grid, hovered) {
				h.renderer.RenderPortalPlacementPreview(screen, hovered, gridID, h.world)
			}
		}
	}

	if h.selectedTile != nil {
		h.renderer.RenderSelectionHighlight(
			screen,
			*h.selectedTile,
			h.selectedGridID,
			color.RGBA{255, 0, 0, 150},
			h.world,
		)
	}
}

func (h *Handler) isValidPortalPreviewPosition(grid *board.Grid, center board.Position) bool {
	return center.X >= 1 && center.Y >= 1 && center.X <= grid.Width-2 && center.Y <= grid.Height-2
}

func (h *Handler) GetCurrentGridID() string {
	if h.selectedGridID != "" {
		return h.selectedGridID
	}
	return h.world.CurrentGridID
}

// GetRevealedTiles retourne les tuiles révélées pendant le tour courant.
// Utilisé par le gestionnaire de boutons d'action pour le calcul réactif.
func (h *Handler) GetRevealedTiles() []board.Position {
	return h.revealedTiles
}

func (h *Handler) ClearSelection() {
	h.selectedTile = nil
	h.selectedGridID = ""
}

func (h *Handler) SetPortablePortalMode(active bool) {
	h.portablePortalMode = active
}

func (h *Handler) IsPortablePortalMode() bool {
	return h.portablePortalMode
}

// ResetTimerSkip est appelé par l'app quand le timer temps réel expire.
// Il simule un Skip sans reset du timer (le reset est fait côté app).
func (h *Handler) ResetTimerSkip() {
	if len(h.revealedTiles) > 0 {
		h.hideRevealedTiles()
	}
	h.isProcessing = false
	h.ClearSelection()
	if h.OnTurnEnd != nil {
		h.OnTurnEnd()
	}
}

// ResetGameState réinitialise l'état du jeu (pour retour au menu)
func (h *Handler) ResetGameState() {
	h.selectedTile = nil
	h.selectedGridID = ""
	h.revealedTiles = nil
	h.isProcessing = false
}

// exitMatchable est un wrapper pour soumettre les sorties au moteur d'association
type exitMatchable struct {
	id string
}

func (m *exitMatchable) GetMatchID() string      { return m.id }
func (m *exitMatchable) GetLogicKey() string     { return "" }
func (m *exitMatchable) GetElement() string      { return "" }
func (m *exitMatchable) GetNarrativeTag() string { return "" }
func (m *exitMatchable) GetMatchTypes() []string { return []string{"orientation"} }
