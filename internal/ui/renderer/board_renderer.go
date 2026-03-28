// Package renderer gère l'affichage du jeu
package renderer

import (
	"fmt"
	"image/color"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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

	// Rotation visuelle globale du plateau (en degrés)
	boardRotation float64

	// Animations de flip en cours: clé = "gridID:x,y", valeur = état de l'animation
	flipAnimations map[string]*FlipAnimation
}

// FlipAnimation représente l'état d'une animation de flip
type FlipAnimation struct {
	GridID        string
	Position      board.Position
	FlipDirection board.FlipDirection
	Progress      float64 // 0.0 à 1.0
	Speed         float64
	EntityID      string           // L'entité à afficher à la fin
	TileState     entity.TileState // État final de la tuile (changé board vers entity)
}

// IsActive retourne true si l'animation est en cours
func (a *FlipAnimation) IsActive() bool {
	return a.Progress < 1.0
}

// GetCurrentRotation retourne les angles de rotation actuels (X, Y) en fonction du progrès
func (a *FlipAnimation) GetCurrentRotation() (rotateX, rotateY float64) {
	targetX, targetY := a.FlipDirection.ToRotationAngles()
	// Interpole entre 0 et l'angle cible basé sur le progrès
	// Utilise une courbe ease-out pour un effet plus naturel
	eased := 1 - math.Pow(1-a.Progress, 3)
	return targetX * eased, targetY * eased
}

// NewBoardRenderer crée un nouveau renderer
func NewBoardRenderer(am *assets.Manager) *BoardRenderer {
	return &BoardRenderer{
		assets:         am,
		tileSize:       64,
		gridOffsetX:    50,
		gridOffsetY:    100,
		gridSpacing:    30,
		gridsPerRow:    2,
		flipAnimations: make(map[string]*FlipAnimation),
	}
}

// SetGridOffset change la position du plateau à l'écran
func (r *BoardRenderer) SetGridOffset(x, y int) {
	r.gridOffsetX = x
	r.gridOffsetY = y
}

// SetBoardRotation définit la rotation visuelle globale du plateau (en degrés)
func (r *BoardRenderer) SetBoardRotation(degrees float64) {
	r.boardRotation = math.Mod(degrees, 360)
}

// GetBoardRotation retourne la rotation actuelle du plateau
func (r *BoardRenderer) GetBoardRotation() float64 {
	return r.boardRotation
}

// RotateBoard ajoute une rotation au plateau (delta en degrés)
func (r *BoardRenderer) RotateBoard(delta float64) {
	r.SetBoardRotation(r.boardRotation + delta)
}

// StartFlipAnimation démarre une animation de flip pour une tuile
func (r *BoardRenderer) StartFlipAnimation(gridID string, pos board.Position, flipDir board.FlipDirection, entityID string, finalState entity.TileState) {
	key := fmt.Sprintf("%s:%d,%d", gridID, pos.X, pos.Y)
	r.flipAnimations[key] = &FlipAnimation{
		GridID:        gridID,
		Position:      pos,
		FlipDirection: flipDir,
		Progress:      0.0,
		Speed:         0.15, // Vitesse de l'animation
		EntityID:      entityID,
		TileState:     finalState,
	}
}

// UpdateAnimations met à jour toutes les animations de flip
func (r *BoardRenderer) UpdateAnimations() {
	for key, anim := range r.flipAnimations {
		anim.Progress += anim.Speed
		if anim.Progress >= 1.0 {
			anim.Progress = 1.0
			// Animation terminée, on peut la supprimer après un délai
			delete(r.flipAnimations, key)
		}
	}
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
	// Met à jour les animations
	r.UpdateAnimations()

	// Affiche le grid actuel et la rotation (en haut, pas de titre qui double le HUD)
	currentGridInfo := fmt.Sprintf("Grid: %s", world.CurrentGridID)
	if r.boardRotation != 0 {
		currentGridInfo += fmt.Sprintf(" (Rot: %.0f°)", r.boardRotation)
	}
	text.Draw(screen, currentGridInfo, basicfont.Face7x13, 250, 20, color.RGBA{255, 255, 0, 255})

	// Message d'aide rapide (une seule ligne)
	helpText := "S:Spawn | C:Clear | SPACE:Tour | 1-9:Grid | F9:Random"
	if r.boardRotation != 0 {
		helpText += " | R:Reset Rot"
	}
	text.Draw(screen, helpText, basicfont.Face7x13, 250, 35, color.Gray{180})

	// Dessine tous les grids dans l'ordre de création (évite le clignotement)
	for _, gridID := range world.GridOrder {
		r.renderGrid(screen, gridID, world)
	}

	// Dessine les infos sur les entités à droite
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
	// Dans renderGrid
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			sx := offsetX + x*r.tileSize
			sy := offsetY + y*r.tileSize
			plot, ok := grid.Plots[pos]
			if !ok {
				r.renderEmptySquareAt(screen, sx, sy)
				continue
			}
			r.renderTileAt(screen, sx, sy, plot, world)
		}
	}
}

// renderEmptySquareAt dessine un carré vide (sol nu)
func (r *BoardRenderer) renderEmptySquareAt(screen *ebiten.Image, x, y int) {
	tileImg := r.assets.GetImage("square_empty")
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(tileImg, op)
}

// renderTileAt dessine une tuile à une position écran spécifique
func (r *BoardRenderer) renderTileAt(screen *ebiten.Image, x, y int, plot *board.Plot, world *domain.World) {
	if len(plot.EntitiesID) == 0 {
		r.renderEmptySquareAt(screen, x, y)
		return
	}

	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, ok := world.Entities.Get(entity.ID(topID))
	if !ok {
		r.renderEmptySquareAt(screen, x, y)
		return
	}

	visualState := ent.GetState()

	var animation *FlipAnimation

	for _, anim := range r.flipAnimations {
		if anim.Position == tile.Position {
			animation = anim
			break
		}
	}

	// Fond de la tuile selon son état visuel
	var tileImg *ebiten.Image
	switch visualState {
	case entity.Hidden:
		tileImg = r.assets.GetImage("tile_hidden")
	case entity.Revealed:
		// Si c'est une tuile piège (trap), utilise l'asset spécifique
		if ent.GetType() == entity.TypeTrap {
			tileImg = r.assets.GetImage("tile_trap")
		} else {
			tileImg = r.assets.GetImage("tile_revealed")
		}
	case entity.Matched:
		tileImg = r.assets.GetImage("tile_matched")
	case entity.Blocked:
		tileImg = r.assets.GetImage("tile_blocked")
	default:
		tileImg = r.assets.GetImage("square_empty")
	}

	// Configure les options de dessin avec rotation et flip
	op := &ebiten.DrawImageOptions{}

	// Centre de la tuile pour les transformations
	centerX := float64(r.tileSize) / 2
	centerY := float64(r.tileSize) / 2

	// Applique la rotation du plateau si définie
	if r.boardRotation != 0 {
		op.GeoM.Translate(-centerX, -centerY)
		op.GeoM.Rotate(r.boardRotation * math.Pi / 180)
		op.GeoM.Translate(centerX, centerY)
	}

	// Applique l'animation de flip si active
	if animation != nil && animation.IsActive() {
		rotateX, rotateY := animation.GetCurrentRotation()
		if rotateX != 0 {
			op.GeoM.Translate(0, -centerY)
			op.GeoM.Scale(1, math.Abs(math.Cos(rotateX*math.Pi/180)))
			op.GeoM.Translate(0, centerY)
		}
		if rotateY != 0 {
			op.GeoM.Translate(-centerX, 0)
			op.GeoM.Scale(math.Abs(math.Cos(rotateY*math.Pi/180)), 1)
			op.GeoM.Translate(centerX, 0)
		}
	}

	// Translate à la position finale
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(tileImg, op)

	// Si la tuile est révélée ou appairée, montre le contenu (si ce n'est pas un piège)
	shouldShowContent := visualState == entity.Revealed || visualState == entity.Matched
	if animation != nil && animation.IsActive() && animation.Progress > 0.5 {
		shouldShowContent = true
	}

	if shouldShowContent && ent.GetType() != entity.TypeTrap {
		r.renderEntityAt(screen, x, y, ent)
	}
}

// renderPlot dessine une case individuelle (utilise l'ancien offset - pour compatibilité)
func (r *BoardRenderer) renderPlot(screen *ebiten.Image, pos board.Position, tile *board.Plot, world *domain.World) {
	x := r.gridOffsetX + pos.X*r.tileSize
	y := r.gridOffsetY + pos.Y*r.tileSize
	r.renderTileAt(screen, x, y, tile, world)
}

// renderEntityAt dessine une entité à une position écran spécifique
func (r *BoardRenderer) renderEntityAt(screen *ebiten.Image, x, y int, e entity.Entity) {
	centerX := float32(x + r.tileSize/2)

	switch ent := e.(type) {
	case *domain.Creature:
		// Icône de créature - centrée dans la tuile
		icon := r.assets.GetCreatureIcon(ent.Species)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(r.tileSize)/2, -float64(r.tileSize)/2)
		op.GeoM.Scale(0.75, 0.75)
		op.GeoM.Translate(float64(x+r.tileSize/2), float64(y+r.tileSize/2))
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
		// Icône de ressource - centrée dans la tuile
		icon := r.assets.GetResourceIcon(ent.ResourceType)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(r.tileSize)/2, -float64(r.tileSize)/2)
		op.GeoM.Scale(0.75, 0.75)
		op.GeoM.Translate(float64(x+r.tileSize/2), float64(y+r.tileSize/2))
		screen.DrawImage(icon, op)

		// Indicateur de stade
		stageName := ent.Lifecycle.GetCurrentStageName()
		if len(stageName) > 0 {
			label := string(stageName[0])
			text.Draw(screen, label, basicfont.Face7x13, x+r.tileSize-12, y+r.tileSize-5, color.White)
		}
	}
}

// renderEntity dessine une entité sur une tuile (ancienne méthode pour compatibilité)
func (r *BoardRenderer) renderEntity(screen *ebiten.Image, x, y int, e entity.Entity) {
	r.renderEntityAt(screen, x, y, e)
}

// renderEntityInfo affiche les informations sur les entités visibles (panneau de droite)
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
	infoY := r.gridOffsetY + 30

	text.Draw(screen, "-- ENTITIES --", basicfont.Face7x13, infoX, infoY, color.RGBA{100, 255, 100, 255})
	infoY += 18

	for _, gridID := range world.GridOrder {
		resources := 0
		creatures := 0

		for _, e := range world.Entities.GetByType(entity.TypeResource) {
			if e.GetGridID() == gridID {
				resources++
			}
		}
		for _, e := range world.Entities.GetByType(entity.TypeCreature) {
			if e.GetGridID() == gridID {
				creatures++
			}
		}

		info := fmt.Sprintf("[%s] R:%d C:%d", gridID, resources, creatures)
		text.Draw(screen, info, basicfont.Face7x13, infoX, infoY, color.White)
		infoY += 14
	}

	infoY += 8
	text.Draw(screen, "-- CREATURES --", basicfont.Face7x13, infoX, infoY, color.RGBA{100, 255, 100, 255})
	infoY += 16

	maxToShow := 12
	shown := 0

	for _, gridID := range world.GridOrder {
		for _, c := range world.Entities.GetByType(entity.TypeCreature) {
			if shown >= maxToShow {
				text.Draw(screen, "...", basicfont.Face7x13, infoX, infoY, color.Gray{128})
				return
			}
			if c.GetGridID() != gridID {
				continue
			}
			if creature, ok := c.(*domain.Creature); ok {
				info := fmt.Sprintf("[%s] %s", gridID, creature.Species)
				text.Draw(screen, info, basicfont.Face7x13, infoX, infoY, color.Gray{200})
				infoY += 13
				shown++
			}
		}
	}
}

// ScreenToGrid convertit les coordonnées écran en coordonnées grille et gridID
func (r *BoardRenderer) ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool) {
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

// ScreenToLocalTile convertit les coordonnées écran en coordonnées locales dans une tuile
func (r *BoardRenderer) ScreenToLocalTile(screenX, screenY int, world *domain.World) (localX, localY int, gridID string, ok bool) {
	pos, gID, found := r.ScreenToGrid(screenX, screenY, world)
	if !found {
		return 0, 0, "", false
	}

	offsetX, offsetY, _ := r.getGridLayout(gID, world)
	tileScreenX := offsetX + pos.X*r.tileSize
	tileScreenY := offsetY + pos.Y*r.tileSize
	lx := screenX - tileScreenX
	ly := screenY - tileScreenY

	return lx, ly, gID, true
}

// RenderSelectionHighlight dessine une surbrillance sur une tuile sélectionnée
func (r *BoardRenderer) RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, highlightColor color.Color, world *domain.World) {
	offsetX, offsetY, grid := r.getGridLayout(gridID, world)
	if grid == nil {
		return
	}

	x := offsetX + pos.X*r.tileSize
	y := offsetY + pos.Y*r.tileSize
	vector.StrokeRect(screen, float32(x), float32(y), float32(r.tileSize), float32(r.tileSize), 3, highlightColor, true)
}
