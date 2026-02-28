package input

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
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

	OnTurnEnd       func()
	OnSpawnEntities func(gridID string)
	OnClearBoard    func(gridID string)
	OnSwitchGrid    func(gridID string)
	OnRotateBoard   func(delta float64) // Callback pour la rotation du plateau
	OnResetRotation func()              // Callback pour réinitialiser la rotation

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
	tile, err := grid.Get(pos)
	if err != nil {
		return nil
	}

	switch tile.State {
	case board.Hidden:
		if tile.EntityID == "" {
			fmt.Printf("[EASY MODE] Tuile vide en %v : Retrait automatique.\n", pos)
			tile.State = board.Blocked
			return nil
		}

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
			fmt.Printf("[INPUT] Tuile révélée en %v sur %s (flip: %s)\n", pos, gridID, flipDir.String())
			h.revealedTiles = append(h.revealedTiles, pos)
		}

		h.selectedTile = &pos
		h.selectedGridID = gridID

		// Si on a révélé 2 tuiles, démarre le timer pour le match automatique
		// (environ 800ms à 60fps = 48 frames)
		if len(h.revealedTiles) == 2 {
			h.isProcessing = true
			h.matchTimer = 48 // 48 frames = 800ms à 60fps
			fmt.Println("[MATCH] Délai de 800ms avant résolution...")
		}

	case board.Revealed:
		if h.selectedTile != nil && h.selectedGridID == gridID && *h.selectedTile == pos {
			fmt.Println("[INPUT] Sélection annulée")
			h.ClearSelection()
		} else {
			fmt.Printf("[INPUT] Tuile sélectionnée : %v\n", pos)
			h.selectedTile = &pos
			h.selectedGridID = gridID
		}
	}
	return nil
}

// processMatchAttempt tente d'associer les 2 tuiles révélées (appelé après le délai)
func (h *Handler) processMatchAttempt() {
	if len(h.revealedTiles) != 2 {
		h.isProcessing = false
		return
	}

	pos1 := h.revealedTiles[0]
	pos2 := h.revealedTiles[1]

	fmt.Printf("[MATCH] Tentative d'association entre %v et %v...\n", pos1, pos2)

	cmd := &usecase.MatchTilesCommand{
		World:    h.world,
		AssocEng: h.assocEngine,
		GridID:   h.selectedGridID,
		Pos1:     pos1,
		Pos2:     pos2,
		OnSuccess: func() {
			fmt.Println("[MATCH] ✅ Association réussie ! Les tuiles restent visibles.")
			h.revealedTiles = nil
			h.isProcessing = false
			h.ClearSelection()
		},
		OnFailure: func() {
			fmt.Println("[MATCH] ❌ Association échouée ! Les tuiles sont retournées.")
			h.revealedTiles = nil
			h.isProcessing = false
			h.ClearSelection()
			// Passe le tour
			if h.OnTurnEnd != nil {
				fmt.Println("[TURN] Fin du tour après échec")
				h.OnTurnEnd()
			}
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Printf("[MATCH] %v\n", err)
		// En cas d'erreur, réinitialise quand même
		h.revealedTiles = nil
		h.isProcessing = false
	}
}

func (h *Handler) countRevealedTiles(g *board.Grid) int {
	count := 0
	for _, tile := range g.Tiles {
		if tile.State == board.Revealed {
			count++
		}
	}
	return count
}

func (h *Handler) handleKeyboard() {
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		fmt.Println("[KEY] Touche M pressée : tentative de Match...")
		h.tryMatchSelected()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if h.OnTurnEnd != nil {
			fmt.Println("[KEY] Espace : Fin du tour")
			h.OnTurnEnd()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		fmt.Println("[KEY] S : Spawn entités")
		if h.OnSpawnEntities != nil {
			h.OnSpawnEntities(h.GetCurrentGridID())
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
				fmt.Printf("[KEY] Touche %d : Switch vers %s\n", i+1, gridID)
				if h.OnSwitchGrid != nil {
					h.OnSwitchGrid(gridID)
					h.ClearSelection()
				}
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		fmt.Println("[KEY] Echap : Sélection nettoyée")
		h.ClearSelection()
	}

	// Rotation du plateau
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		fmt.Println("[KEY] R : Réinitialisation de la rotation")
		if h.OnResetRotation != nil {
			h.OnResetRotation()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPEqual) {
		fmt.Println("[KEY] + : Rotation horaire")
		if h.OnRotateBoard != nil {
			h.OnRotateBoard(15)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		fmt.Println("[KEY] - : Rotation anti-horaire")
		if h.OnRotateBoard != nil {
			h.OnRotateBoard(-15)
		}
	}
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

	for _, tile := range grid.Tiles {
		if tile.Position.X == h.selectedTile.X && tile.Position.Y == h.selectedTile.Y {
			continue
		}

		if tile.State == board.Revealed {
			fmt.Printf("[MATCH] Comparaison entre %v et %v...\n", *h.selectedTile, tile.Position)

			if tile.EntityID == "" {
				fmt.Println("[MATCH] Bug : Une tuile vide a été révélée par erreur.")
				continue
			}

			cmd := &usecase.MatchTilesCommand{
				World:    h.world,
				AssocEng: h.assocEngine,
				GridID:   h.selectedGridID,
				Pos1:     *h.selectedTile,
				Pos2:     tile.Position,
				OnSuccess: func() {
					fmt.Println("[MATCH] Paire trouvée ! Le joueur peut continuer.")
					h.ClearSelection()
				},
				OnFailure: func() {
					fmt.Println("[MATCH] Échec ! Les cartes sont retournées et le tour passe.")
					h.ClearSelection()
					// Déclenche la fin de tour
					if h.OnTurnEnd != nil {
						fmt.Println("[TURN] Fin du tour après échec d'association")
						h.OnTurnEnd()
					}
				},
			}

			if err := cmd.Execute(); err == nil {
				// Succès géré par OnSuccess
				return
			} else {
				// Échec géré par OnFailure
				fmt.Printf("[MATCH] %v\n", err)
				return
			}
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
		if grid, ok := h.world.GetGrid(gridID); ok {
			if tile, err := grid.Get(hovered); err == nil {
				var highlightColor color.Color
				switch tile.State {
				case board.Hidden:
					highlightColor = color.RGBA{255, 255, 0, 100} // Jaune : Survolé
				case board.Revealed:
					highlightColor = color.RGBA{0, 255, 255, 100} // Cyan : Déjà ouvert
				case board.Blocked:
					return // Pas de highlight pour les tuiles retirées
				default:
					highlightColor = color.RGBA{255, 255, 255, 50}
				}
				h.renderer.RenderSelectionHighlight(screen, hovered, gridID, highlightColor, h.world)
			}
		}
	}

	if h.selectedTile != nil {
		h.renderer.RenderSelectionHighlight(screen, *h.selectedTile, h.selectedGridID, color.RGBA{255, 0, 0, 150}, h.world)
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
	// Note: on ne réinitialise pas revealedTiles ici car c'est géré par processMatchAttempt
}
