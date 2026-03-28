package input

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
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
}

type Handler struct {
	world       *domain.World
	assocEngine *domain.AssocEngine
	renderer    Renderer

	selectedTile   *board.Position
	selectedGridID string

	OnTurnEnd             func()
	OnSpawnEntities       func(gridID string)
	OnSpawnAllCreatures   func(gridID string) // Shift+S: Spawn toutes les créatures
	OnSpawnRandomCreature func(gridID string) // F9: Spawn créature aléatoire
	OnClearBoard          func(gridID string)
	OnSwitchGrid          func(gridID string)
	OnRotateBoard         func(delta float64) // Callback pour la rotation du plateau
	OnResetRotation       func()              // Callback pour réinitialiser la rotation
	OnExitToMenu          func()              // Callback pour retourner au menu
	OnRevealAll           func(gridID string) // F5: Cheat - révéler tout
	OnHideAll             func(gridID string) // F6: Cheat - cacher tout
	OnForceTurn           func()              // F3: Forcer le prochain tour
	OnToggleAutoMove      func()              // F10: Toggle mouvement auto

	// Gestion du tour de jeu memory
	revealedTiles []board.Position // Liste des tuiles révélées ce tour
	isProcessing  bool             // Évite les clics pendant l'animation
	matchTimer    int              // Compteur de frames pour le délai de matching
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

func (h *Handler) Update() error {
	if err := h.handleMouse(); err != nil {
		return err
	}
	h.handleKeyboard()
	h.updateMatchTimer()
	return nil
}

// updateMatchTimer gère le délai avant le matching automatique
func (h *Handler) updateMatchTimer() {
	if h.matchTimer > 0 {
		h.matchTimer--
		if h.matchTimer == 0 {
			h.processMatchAttempt()
		}
	}
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
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	if h.isProcessing {
		fmt.Println("[INPUT] Traitement en cours, veuillez patienter...")
		return nil
	}

	pos, gridID, ok := h.getHoveredTile()
	if !ok {
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

	switch ent.GetState() {
	case entity.Hidden:
		// Vérifie si on a déjà révélé 2 tuiles ce tour
		if len(h.revealedTiles) >= 2 {
			fmt.Println("[INPUT] Déjà 2 tuiles révélées ce tour. Veuillez attendre la fin du traitement.")
			return nil
		}

		// Calcule la direction de flip basée sur la position du clic dans la tuile
		flipDir := h.calculateFlipDirection(gridID, pos)

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
		}

		// On met à jour le gridID pour la résolution du match
		h.selectedGridID = gridID

		// On sélectionne la tuile pour le match
		h.selectedTile = &pos

		// Si on a révélé 2 tuiles, démarre le timer pour le match automatique
		if len(h.revealedTiles) == 2 {
			h.isProcessing = true
			h.matchTimer = 48 // 48 frames = 800ms à 60fps
			fmt.Println("[MATCH] Délai de 800ms avant résolution...")
		}

	case entity.Revealed:
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
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		h.tryMatchSelected()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if h.OnTurnEnd != nil {
			fmt.Println("[TOUR] Passage au tour suivant")
			h.OnTurnEnd()
		}
	}

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
		fmt.Println("[INPUT] Sélection annulée")
		h.ClearSelection()
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

func (h *Handler) setDifficulty(level meta.DifficultyLevel) {
	settings := meta.GetSettings(level)
	h.world.Difficulty = settings
	fmt.Printf("[DIFFICULTÉ] Niveau changé pour : %s\n", level)
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

			fmt.Printf("[MATCH] Comparaison manuelle : %s vs %s\n", h.getEntityInfo(e1), h.getEntityInfo(ent))

			cmd := &usecase.MatchTilesCommand{
				World:    h.world,
				AssocEng: h.assocEngine,
				GridID:   h.selectedGridID,
				Pos1:     *h.selectedTile,
				Pos2:     plot.Position,
				OnSuccess: func() {
					fmt.Println("[MATCH] ✅ Succès !")
					h.ClearSelection()
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

// calculateFlipDirection détermine la direction de flip basée sur la position du clic dans la tuile
func (h *Handler) calculateFlipDirection(gridID string, pos board.Position) domain.FlipDirection {
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
		switch ent.GetState() {
		case entity.Hidden:
			highlightColor = color.RGBA{255, 255, 0, 100}
		case entity.Revealed:
			highlightColor = color.RGBA{0, 255, 255, 100}
		default:
			highlightColor = color.RGBA{255, 255, 255, 50}
		}

		h.renderer.RenderSelectionHighlight(screen, hovered, gridID, highlightColor, h.world)
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

func (h *Handler) GetCurrentGridID() string {
	if h.selectedGridID != "" {
		return h.selectedGridID
	}
	return h.world.CurrentGridID
}

func (h *Handler) ClearSelection() {
	h.selectedTile = nil
	h.selectedGridID = ""
}

// ResetGameState réinitialise l'état du jeu (pour retour au menu)
func (h *Handler) ResetGameState() {
	h.selectedTile = nil
	h.selectedGridID = ""
	h.revealedTiles = nil
	h.isProcessing = false
	h.matchTimer = 0
}
