// Package input gère les entrées utilisateur (souris, clavier)
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

// Handler gère les entrées utilisateur
type Handler struct {
	world       *domain.World
	assocEngine *domain.AssocEngine

	// État de la sélection (lié au grid actuel)
	selectedTile   *board.Position
	selectedGridID string
	renderer       Renderer

	// Callbacks pour les actions
	OnTurnEnd       func()
	OnSpawnEntities func(gridID string)
	OnClearBoard    func(gridID string)
	OnSwitchGrid    func(gridID string)
}

// Renderer interface minimale pour le rendu (évite dépendance circulaire)
type Renderer interface {
	GetTileSize() int
	GetGridOffset() (int, int)
	ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool)
	RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, color color.Color, world *domain.World)
}

// NewHandler crée un nouveau gestionnaire d'entrées
func NewHandler(world *domain.World, assocEng *domain.AssocEngine) *Handler {
	return &Handler{
		world:          world,
		assocEngine:    assocEng,
		selectedGridID: "",
	}
}

// SetRenderer définit le renderer pour les surbrillances
func (h *Handler) SetRenderer(r Renderer) {
	h.renderer = r
}

// Update traite les entrées utilisateur (logique uniquement)
func (h *Handler) Update() error {
	// Gestion de la souris
	if err := h.handleMouse(); err != nil {
		return err
	}

	// Gestion du clavier
	h.handleKeyboard()

	return nil
}

// Draw dessine les surbrillances et retours visuels
func (h *Handler) Draw(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	// Dessine les surbrillances
	h.renderHighlights(screen)
}

// handleMouse gère les clics souris
func (h *Handler) handleMouse() error {
	// Clic gauche : révéler ou sélectionner
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		fmt.Printf("[MOUSE] Click at screen (%d, %d)\n", x, y)

		pos, gridID, ok := h.getHoveredTile()
		if !ok {
			fmt.Printf("[MOUSE] No tile under cursor\n")
			return nil
		}
		fmt.Printf("[MOUSE] Tile at grid (%s, %d, %d)\n", gridID, pos.X, pos.Y)

		grid, ok := h.world.GetGrid(gridID)
		if !ok {
			return nil
		}

		// Vérifie si la tuile est valide
		tile, err := grid.Get(pos)
		if err != nil {
			return nil
		}

		// Si cachée, on la révèle
		if tile.State == board.Hidden {
			cmd := &usecase.RevealTileCommand{
				World:    h.world,
				GridID:   gridID,
				Position: pos,
			}
			if err := cmd.Execute(); err != nil {
				// Silencieux si échec (ex: déjà révélée)
				return nil
			}

			// Réinitialise la sélection après révélation
			h.selectedTile = nil
			h.selectedGridID = ""
			return nil
		}

		// Si révélée, on la sélectionne pour un futur match
		if tile.State == board.Revealed {
			// Si on clique sur la même tuile, on la désélectionne
			if h.selectedTile != nil && h.selectedGridID == gridID &&
				h.selectedTile.X == pos.X && h.selectedTile.Y == pos.Y {
				h.selectedTile = nil
				h.selectedGridID = ""
			} else {
				h.selectedTile = &pos
				h.selectedGridID = gridID
			}
		}
	}

	return nil
}

// handleKeyboard gère les touches du clavier
func (h *Handler) handleKeyboard() {
	// Debug: log toutes les touches qui viennent d'être pressées
	var keys []ebiten.Key
	keys = inpututil.AppendJustPressedKeys(keys)
	for _, key := range keys {
		fmt.Printf("[KEY] Just pressed: %v\n", key)
	}

	// M : tenter un match avec la tuile sélectionnée
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		fmt.Println("[KEY] M pressed - trying to match")
		h.tryMatchSelected()
	}

	// Espace : fin de tour
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if h.OnTurnEnd != nil {
			h.OnTurnEnd()
		}
	}

	// S : spawn des entités de test sur le grid actuel
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		fmt.Println("[KEY] S pressed")
		if h.OnSpawnEntities != nil {
			gridID := h.world.CurrentGridID
			if gridID == "" && len(h.world.GridOrder) > 0 {
				// Utilise le premier grid disponible
				gridID = h.world.GridOrder[0]
			}
			h.OnSpawnEntities(gridID)
		}
	}

	// C : clear board du grid actuel
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		fmt.Println("[KEY] C pressed")
		if h.OnClearBoard != nil {
			gridID := h.world.CurrentGridID
			if gridID == "" && len(h.world.GridOrder) > 0 {
				gridID = h.world.GridOrder[0]
			}
			h.OnClearBoard(gridID)
		}
	}

	// 1, 2, 3... : changer de grid
	for i := 0; i < 9; i++ {
		key := ebiten.Key(i + int(ebiten.Key1))
		if inpututil.IsKeyJustPressed(key) {
			gridIndex := i
			gridID := ""
			if gridIndex < len(h.world.GridOrder) {
				gridID = h.world.GridOrder[gridIndex]
			}
			if gridID != "" && h.OnSwitchGrid != nil {
				fmt.Printf("[KEY] Switching to grid %s\n", gridID)
				h.OnSwitchGrid(gridID)
				h.selectedTile = nil
				h.selectedGridID = ""
			}
		}
	}

	// Échap : désélectionne
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		h.selectedTile = nil
		h.selectedGridID = ""
	}
}

// tryMatchSelected tente d'appairer la tuile sélectionnée avec une autre tuile révélée
func (h *Handler) tryMatch() {
	if h.selectedTile == nil || h.selectedGridID == "" {
		return
	}

	grid, ok := h.world.GetGrid(h.selectedGridID)
	if !ok {
		return
	}

	// Trouve une autre tuile révélée avec une entité sur le même grid
	for pos, tile := range grid.Tiles {
		// Ignore la tuile sélectionnée
		if pos.X == h.selectedTile.X && pos.Y == h.selectedTile.Y {
			continue
		}

		// Cherche une tuile révélée avec une entité
		if tile.State == board.Revealed && tile.EntityID != "" {
			cmd := &usecase.MatchTilesCommand{
				World:    h.world,
				AssocEng: h.assocEngine,
				GridID:   h.selectedGridID,
				Pos1:     *h.selectedTile,
				Pos2:     pos,
			}

			if cmd.CanExecute() {
				if err := cmd.Execute(); err == nil {
					// Match réussi, désélectionne
					h.selectedTile = nil
					h.selectedGridID = ""
					return
				}
			}
		}
	}
}

// tryMatchSelected tente d'appairer avec une autre tuile révélée
func (h *Handler) tryMatchSelected() {
	if h.selectedTile == nil || h.selectedGridID == "" {
		return
	}

	grid, ok := h.world.GetGrid(h.selectedGridID)
	if !ok {
		return
	}

	// Cherche une autre tuile révélée pour le match sur le même grid
	for pos, tile := range grid.Tiles {
		if pos.X == h.selectedTile.X && pos.Y == h.selectedTile.Y {
			continue
		}

		if tile.State == board.Revealed && tile.EntityID != "" {
			cmd := &usecase.MatchTilesCommand{
				World:    h.world,
				AssocEng: h.assocEngine,
				GridID:   h.selectedGridID,
				Pos1:     *h.selectedTile,
				Pos2:     pos,
			}

			if err := cmd.Execute(); err == nil {
				h.selectedTile = nil
				h.selectedGridID = ""
				return
			}
		}
	}
}

// getHoveredTile retourne la tuile sous la souris
func (h *Handler) getHoveredTile() (board.Position, string, bool) {
	if h.renderer == nil {
		return board.Position{}, "", false
	}

	x, y := ebiten.CursorPosition()
	return h.renderer.ScreenToGrid(x, y, h.world)
}

// renderHighlights dessine les surbrillances
func (h *Handler) renderHighlights(screen *ebiten.Image) {
	if h.renderer == nil {
		return
	}

	// Surbrillance de la tuile sous la souris
	if hovered, gridID, ok := h.getHoveredTile(); ok {
		grid, err := h.world.GetGrid(gridID)
		if err {
			tile, tileErr := grid.Get(hovered)
			if tileErr == nil {
				var highlightColor color.Color
				switch tile.State {
				case board.Hidden:
					highlightColor = color.RGBA{255, 255, 0, 128}
				case board.Revealed:
					highlightColor = color.RGBA{0, 255, 255, 128}
				case board.Matched:
					highlightColor = color.RGBA{0, 255, 0, 128}
				}
				h.renderer.RenderSelectionHighlight(screen, hovered, gridID, highlightColor, h.world)
			}
		}
	}

	// Surbrillance de la tuile sélectionnée
	if h.selectedTile != nil && h.selectedGridID != "" {
		h.renderer.RenderSelectionHighlight(screen, *h.selectedTile, h.selectedGridID, color.RGBA{255, 100, 100, 200}, h.world)
	}
}

// GetSelectedTile retourne la tuile actuellement sélectionnée
func (h *Handler) GetSelectedTile() (*board.Position, string) {
	return h.selectedTile, h.selectedGridID
}

// ClearSelection désélectionne la tuile
func (h *Handler) ClearSelection() {
	h.selectedTile = nil
	h.selectedGridID = ""
}

// GetCurrentGridID retourne le grid actuellement sélectionné
func (h *Handler) GetCurrentGridID() string {
	if h.selectedGridID != "" {
		return h.selectedGridID
	}
	return h.world.CurrentGridID
}
