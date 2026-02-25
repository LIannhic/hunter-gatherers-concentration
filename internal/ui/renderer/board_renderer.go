// Package renderer gère l'affichage du jeu
package renderer

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// BoardRenderer dessine le plateau de jeu
type BoardRenderer struct {
	assets      *assets.Manager
	tileSize    int
	gridOffsetX int
	gridOffsetY int
	gridSpacing int // Espace entre les grids
	gridsPerRow int // Nombre de grids par ligne
}

// NewBoardRenderer crée un nouveau renderer
func NewBoardRenderer(am *assets.Manager) *BoardRenderer {
	return &BoardRenderer{
		assets:      am,
		tileSize:    64,
		gridOffsetX: 50,
		gridOffsetY: 100,
		gridSpacing: 30,
		gridsPerRow: 2,
	}
}

// SetGridOffset change la position du plateau à l'écran
func (r *BoardRenderer) SetGridOffset(x, y int) {
	r.gridOffsetX = x
	r.gridOffsetY = y
}

// GetTileSize retourne la taille des tuiles
func (r *BoardRenderer) GetTileSize() int {
	return r.tileSize
}

// GetGridOffset retourne le décalage du plateau
func (r *BoardRenderer) GetGridOffset() (int, int) {
	return r.gridOffsetX, r.gridOffsetY
}

// getGridLayout calcule la position d'affichage d'un grid
func (r *BoardRenderer) getGridLayout(gridID string, world *domain.World) (offsetX, offsetY int, grid *board.Grid) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return 0, 0, nil
	}

	// Trouve l'index du grid dans l'ordre stable
	idx := -1
	for i, id := range world.GridOrder {
		if id == gridID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return 0, 0, nil
	}

	// Calcule la position en grille
	row := idx / r.gridsPerRow
	col := idx % r.gridsPerRow

	// Calcule la taille d'un grid
	gridWidth := grid.Width * r.tileSize

	offsetX = r.gridOffsetX + col*(gridWidth+r.gridSpacing)
	offsetY = r.gridOffsetY + row*(grid.Height*r.tileSize+r.gridSpacing+30) // +30 pour le titre

	return offsetX, offsetY, grid
}

// Render dessine le plateau complet
func (r *BoardRenderer) Render(screen *ebiten.Image, world *domain.World) {
	// Dessine le titre
	title := fmt.Sprintf("Hunter-Gatherers Concentration - Tour %d", world.Turn)
	text.Draw(screen, title, basicfont.Face7x13, 10, 20, color.White)

	// Instructions
	text.Draw(screen, "Click to reveal, M to match selected", basicfont.Face7x13, 10, 40, color.White)
	text.Draw(screen, "S: Spawn test | C: Clear | SPACE: End turn | 1-9: Switch Grid", basicfont.Face7x13, 10, 55, color.White)

	// Affiche le grid actuel
	currentGridInfo := fmt.Sprintf("Current Grid: %s", world.CurrentGridID)
	text.Draw(screen, currentGridInfo, basicfont.Face7x13, 10, 70, color.RGBA{255, 255, 0, 255})

	// Dessine tous les grids dans l'ordre de création (évite le clignotement)
	for _, gridID := range world.GridOrder {
		r.renderGrid(screen, gridID, world)
	}

	// Dessine les infos sur les entités
	r.renderEntityInfo(screen, world)
}

// renderGrid dessine un grid spécifique
func (r *BoardRenderer) renderGrid(screen *ebiten.Image, gridID string, world *domain.World) {
	offsetX, offsetY, grid := r.getGridLayout(gridID, world)
	if grid == nil {
		return
	}

	// Dessine le titre du grid
	gridTitle := fmt.Sprintf("Grid: %s", gridID)
	if gridID == world.CurrentGridID {
		gridTitle += " [ACTIVE]"
	}
	titleColor := color.Color(color.White)
	if gridID == world.CurrentGridID {
		titleColor = color.RGBA{255, 255, 0, 255}
	}
	text.Draw(screen, gridTitle, basicfont.Face7x13, offsetX, offsetY-5, titleColor)

	// Dessine les tuiles du grid (ordre déterministe pour éviter le clignotement)
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			tile, ok := grid.Tiles[pos]
			if !ok {
				continue
			}
			sx := offsetX + x*r.tileSize
			sy := offsetY + y*r.tileSize
			r.renderTileAt(screen, sx, sy, tile, world)
		}
	}
}

// renderTileAt dessine une tuile à une position écran spécifique
func (r *BoardRenderer) renderTileAt(screen *ebiten.Image, x, y int, tile *board.Tile, world *domain.World) {
	// Fond de la tuile selon son état
	var tileImg *ebiten.Image
	switch tile.State {
	case board.Hidden:
		tileImg = r.assets.GetImage("tile_hidden")
	case board.Revealed:
		tileImg = r.assets.GetImage("tile_revealed")
	case board.Matched:
		tileImg = r.assets.GetImage("tile_matched")
	default:
		tileImg = r.assets.GetImage("tile_hidden")
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(tileImg, op)

	// Si la tuile est révélée ou appairée, montre le contenu
	if tile.State == board.Revealed || tile.State == board.Matched {
		if tile.EntityID != "" {
			// Cherche l'entité
			if e, ok := world.Entities.Get(domain.ID(tile.EntityID)); ok {
				r.renderEntityAt(screen, x, y, e)
			}
		}
	}
}

// renderTile dessine une tuile individuelle (utilise l'ancien offset - pour compatibilité)
func (r *BoardRenderer) renderTile(screen *ebiten.Image, pos board.Position, tile *board.Tile, world *domain.World) {
	x := r.gridOffsetX + pos.X*r.tileSize
	y := r.gridOffsetY + pos.Y*r.tileSize
	r.renderTileAt(screen, x, y, tile, world)
}

// renderEntityAt dessine une entité à une position écran spécifique
func (r *BoardRenderer) renderEntityAt(screen *ebiten.Image, x, y int, e domain.Entity) {
	centerX := float32(x + r.tileSize/2)

	switch ent := e.(type) {
	case *domain.Creature:
		// Icône de créature
		icon := r.assets.GetCreatureIcon(ent.Species)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+8), float64(y+8))
		op.GeoM.Scale(0.75, 0.75)
		screen.DrawImage(icon, op)

		// Petit indicateur de comportement
		behaviorColor := color.RGBA{200, 200, 200, 255}
		switch ent.Behavior.State {
		case "hunting":
			behaviorColor = color.RGBA{255, 100, 100, 255}
		case "fleeing":
			behaviorColor = color.RGBA{100, 100, 255, 255}
		case "pollinating":
			behaviorColor = color.RGBA{100, 255, 100, 255}
		}
		vector.DrawFilledCircle(screen, centerX, float32(y+10), 4, behaviorColor, true)

	case *domain.Resource:
		// Icône de ressource
		icon := r.assets.GetResourceIcon(ent.ResourceType)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+8), float64(y+8))
		op.GeoM.Scale(0.75, 0.75)
		screen.DrawImage(icon, op)

		// Indicateur de stade
		stageName := ent.Lifecycle.GetCurrentStageName()
		if len(stageName) > 0 {
			// Première lettre du stade
			label := string(stageName[0])
			text.Draw(screen, label, basicfont.Face7x13, x+r.tileSize-12, y+r.tileSize-5, color.White)
		}
	}
}

// renderEntity dessine une entité sur une tuile (ancienne méthode pour compatibilité)
func (r *BoardRenderer) renderEntity(screen *ebiten.Image, x, y int, e domain.Entity) {
	r.renderEntityAt(screen, x, y, e)
}

// renderEntityInfo affiche les informations sur les entités visibles
func (r *BoardRenderer) renderEntityInfo(screen *ebiten.Image, world *domain.World) {
	// Calcule la position d'info à droite de tous les grids
	maxWidth := 0
	for _, gridID := range world.GridOrder {
		if grid, ok := world.GetGrid(gridID); ok {
			if grid.Width > maxWidth {
				maxWidth = grid.Width
			}
		}
	}

	infoX := r.gridOffsetX + maxWidth*r.tileSize*r.gridsPerRow + r.gridSpacing*(r.gridsPerRow+1)
	if infoX < 500 {
		infoX = 500
	}
	infoY := r.gridOffsetY

	text.Draw(screen, "=== ENTITIES ===", basicfont.Face7x13, infoX, infoY, color.White)
	infoY += 20

	// Compte les entités par grid dans l'ordre de création
	for _, gridID := range world.GridOrder {
		gridInfo := fmt.Sprintf("Grid %s:", gridID)
		text.Draw(screen, gridInfo, basicfont.Face7x13, infoX, infoY, color.RGBA{200, 200, 100, 255})
		infoY += 15

		resources := 0
		creatures := 0

		for _, e := range world.Entities.GetByType(domain.TypeResource) {
			if e.GetGridID() == gridID {
				resources++
			}
		}
		for _, e := range world.Entities.GetByType(domain.TypeCreature) {
			if e.GetGridID() == gridID {
				creatures++
			}
		}

		info := fmt.Sprintf("  Resources: %d", resources)
		text.Draw(screen, info, basicfont.Face7x13, infoX+10, infoY, color.White)
		infoY += 15

		info = fmt.Sprintf("  Creatures: %d", creatures)
		text.Draw(screen, info, basicfont.Face7x13, infoX+10, infoY, color.White)
		infoY += 20
	}

	// Liste des créatures par grid
	infoY += 10
	text.Draw(screen, "=== CREATURES ===", basicfont.Face7x13, infoX, infoY, color.White)
	infoY += 20

	for _, gridID := range world.GridOrder {
		for _, c := range world.Entities.GetByType(domain.TypeCreature) {
			if c.GetGridID() != gridID {
				continue
			}
			if creature, ok := c.(*domain.Creature); ok {
				info := fmt.Sprintf("[%s] %s (%s)", gridID, creature.Species, creature.Behavior.State)
				text.Draw(screen, info, basicfont.Face7x13, infoX, infoY, color.White)
				infoY += 12
			}
		}
	}
}

// ScreenToGrid convertit les coordonnées écran en coordonnées grille et gridID
func (r *BoardRenderer) ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool) {
	// Essaie chaque grid dans l'ordre de création
	for _, gridID := range world.GridOrder {
		offsetX, offsetY, grid := r.getGridLayout(gridID, world)
		if grid == nil {
			continue
		}

		x := screenX - offsetX
		y := screenY - offsetY

		if x < 0 || y < 0 {
			continue
		}

		gridX := x / r.tileSize
		gridY := y / r.tileSize

		if gridX < grid.Width && gridY < grid.Height {
			return board.Position{X: gridX, Y: gridY}, gridID, true
		}
	}

	return board.Position{}, "", false
}

// RenderSelectionHighlight dessine une surbrillance sur une tuile sélectionnée d'un grid spécifique
func (r *BoardRenderer) RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, highlightColor color.Color, world *domain.World) {
	offsetX, offsetY, grid := r.getGridLayout(gridID, world)
	if grid == nil {
		return
	}

	x := offsetX + pos.X*r.tileSize
	y := offsetY + pos.Y*r.tileSize

	// Dessine un rectangle de surbrillance
	vector.StrokeRect(screen, float32(x), float32(y), float32(r.tileSize), float32(r.tileSize), 3, highlightColor, true)
}
