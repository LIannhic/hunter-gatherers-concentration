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
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// BoardRenderer dessine le plateau de jeu
type BoardRenderer struct {
	assets      *assets.Manager
	tileSize    float64
	gridOffsetX float64
	gridOffsetY float64
	gridSpacing float64 // Espace entre les grids
	gridsPerRow int     // Nombre de grids par ligne

	// Rotation visuelle globale du plateau (en degrés)
	boardRotation float64

	// Animations de flip en cours: clé = "gridID:x,y", valeur = état de l'animation
	flipAnimations map[string]*FlipAnimation

	// Config override for incursion mode
	singleGridID string
}

// FlipAnimation représente l'état d'une animation de flip
type FlipAnimation struct {
	GridID        string
	Position      board.Position
	FlipDirection board.FlipDirection
	Progress      float64 // 0.0 à 1.0
	Speed         float64
	EntityID      string           // L'entité à afficher à la fin
	TileState     entity.TileState // État final de la tuile
}

// IsActive retourne true si l'animation est en cours
func (a *FlipAnimation) IsActive() bool {
	return a.Progress < 1.0
}

// GetCurrentRotation retourne les angles de rotation actuels (X, Y) en fonction du progrès
func (a *FlipAnimation) GetCurrentRotation() (rotateX, rotateY float64) {
	targetX, targetY := a.FlipDirection.ToRotationAngles()
	eased := 1 - math.Pow(1-a.Progress, 3)
	return targetX * eased, targetY * eased
}

// NewBoardRenderer crée un nouveau renderer
func NewBoardRenderer(am *assets.Manager) *BoardRenderer {
	return &BoardRenderer{
		assets:         am,
		tileSize:       64,
		gridOffsetX:    50,
		gridOffsetY:    50,
		gridSpacing:    30,
		gridsPerRow:    2,
		flipAnimations: make(map[string]*FlipAnimation),
	}
}

// SetRenderConfig configure le renderer pour un affichage custom (mode incursion)
func (r *BoardRenderer) SetRenderConfig(offsetX, offsetY, tileSize float64, singleGridID string) {
	r.gridOffsetX = offsetX
	r.gridOffsetY = offsetY
	r.tileSize = tileSize
	r.singleGridID = singleGridID
}

// ResetRenderConfig restaure la configuration par défaut
func (r *BoardRenderer) ResetRenderConfig() {
	r.gridOffsetX = 50
	r.gridOffsetY = 50
	r.tileSize = 64
	r.singleGridID = ""
}

// SetGridOffset change la position du plateau à l'écran
func (r *BoardRenderer) SetGridOffset(x, y float64) {
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
		Speed:         0.15,
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
			delete(r.flipAnimations, key)
		}
	}
}

// GetTileSize retourne la taille des tuiles
func (r *BoardRenderer) GetTileSize() float64 {
	return r.tileSize
}

// GetGridOffset retourne le décalage du plateau
func (r *BoardRenderer) GetGridOffset() (float64, float64) {
	return r.gridOffsetX, r.gridOffsetY
}

// getGridLayout calcule la position d'affichage d'un grid
func (r *BoardRenderer) getGridLayout(gridID string, world *domain.World) (offsetX, offsetY float64, grid *board.Grid) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return 0, 0, nil
	}
}

	if r.singleGridID != "" && gridID == r.singleGridID {
		return r.gridOffsetX, r.gridOffsetY, grid
	}

	idx := -1
	for i, id := range world.GridOrder {
		if id == gridID {
			idx = i
			break
		}
	}
}

func (r *BoardRenderer) renderHalfArrow(screen *ebiten.Image, x, y float64, dir board.Direction, index int) {
	centerX := float32(x + r.tileSize/2)
	centerY := float32(y + r.tileSize/2)
	size := float32(r.tileSize)
	color := color.RGBA{200, 200, 255, 200}
	thickness := float32(4)

	// Rapprochement vers la "couture" entre les deux tuiles (plus proche l'une de l'autre sans se toucher)
	seamOffset := size * 0.35
	switch dir {
	case board.North, board.South:
		if index == 0 { // Gauche
			centerX += seamOffset
		} else { // Droite
			centerX -= seamOffset
		}
	case board.East, board.West:
		if index == 0 { // Haut
			centerY += seamOffset
		} else { // Bas
			centerY -= seamOffset
		}
	}

	row := idx / r.gridsPerRow
	col := idx % r.gridsPerRow

	gridWidth := float64(grid.Width) * r.tileSize

	offsetX = r.gridOffsetX + float64(col)*(gridWidth+r.gridSpacing)
	offsetY = r.gridOffsetY + float64(row)*(float64(grid.Height)*r.tileSize+r.gridSpacing+30)

	return spacingX, spacingY, padX, padY
}

// Render dessine le plateau complet
func (r *BoardRenderer) Render(screen *ebiten.Image, world *domain.World) {
	r.UpdateAnimations()

	if r.singleGridID != "" {
		r.renderGrid(screen, r.singleGridID, world)
		return
	}

	for _, gridID := range world.GridOrder {
		r.renderGrid(screen, gridID, world)
	}
	return false
}

// renderGrid dessine un grid spécifique
func (r *BoardRenderer) renderGrid(screen *ebiten.Image, gridID string, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
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
	text.Draw(screen, gridTitle, basicfont.Face7x13, int(offsetX), int(offsetY)-5, titleColor)

	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
			sx := offsetX + float64(x)*r.tileSize
			sy := offsetY + float64(y)*r.tileSize
			plot, ok := grid.Plots[pos]
			if !ok {
				r.renderEmptySquareAt(screen, sx, sy)
				continue
			}
			r.renderTileAt(screen, sx, sy, gridID, plot, world)
		}
	}
}

// renderEmptySquareAt dessine un carré vide (sol nu)
func (r *BoardRenderer) renderEmptySquareAt(screen *ebiten.Image, x, y float64) {
	tileImg := r.assets.GetImage("square_empty")
	op := &ebiten.DrawImageOptions{}
	scale := r.tileSize / 64.0
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(tileImg, op)
}

// renderTileAt dessine une tuile à une position écran spécifique
func (r *BoardRenderer) renderTileAt(screen *ebiten.Image, x, y float64, gridID string, plot *board.Plot, world *domain.World) {
	if len(plot.EntitiesID) == 0 {
		r.renderEmptySquareAt(screen, x, y)
		return
	}

	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, ok := world.Entities.Get(entity.ID(topID))
	if !ok {
		// Ce cas arrive si l'entité est absente du manager mais présente sur la grille (desync)
		// fmt.Printf("[RENDER ERR] Entity %s not found in manager at %v\n", topID, plot.Position)
		r.renderEmptySquareAt(screen, x, y)
		return
	}

	visualState := ent.GetState()
	var animation *FlipAnimation
	for _, anim := range r.flipAnimations {
		if anim.Position == plot.Position && (gridID == "" || anim.GridID == gridID) {
			animation = anim
			break
		}
	}

	var tileImg *ebiten.Image
	isFlipping := animation != nil && animation.IsActive()
	showRevealedSide := false
	if isFlipping {
		if animation.TileState&entity.Hidden != 0 {
			showRevealedSide = animation.Progress < 0.5
		} else {
			showRevealedSide = animation.Progress > 0.5
		}
	}

	if isFlipping {
		if showRevealedSide {
			if ent.GetType() == entity.TypeTrap {
				tileImg = r.assets.GetImage("tile_trap")
			} else {
				tileImg = r.assets.GetImage("tile_revealed")
			}
		} else {
			tileImg = r.assets.GetImage("tile_hidden")
		}
	} else {
		// Gestion des états prioritaires
		if visualState&entity.Matched != 0 {
			tileImg = r.assets.GetImage("tile_matched")
		} else if visualState&entity.Revealed != 0 {
			if ent.GetType() == entity.TypeTrap {
				tileImg = r.assets.GetImage("tile_trap")
			} else if ent.GetType() == entity.TypeStructure {
				if ent.HasTag("commencement_portal") || ent.HasTag("finish_portal") || ent.HasTag("portable_portal") {
					tileImg = r.assets.GetImage("tile_portal")
				} else if ent.HasTag("dolmen") {
					tileImg = r.assets.GetImage("tile_dolmen")
				} else if ent.HasTag("obelisk") {
					tileImg = r.assets.GetImage("tile_obelisk")
				} else {
					tileImg = r.assets.GetImage("tile_structure")
				}
			} else {
				tileImg = r.assets.GetImage("tile_revealed")
			}
		} else {
			// Par défaut ou Hidden
			tileImg = r.assets.GetImage("tile_hidden")
		}

		// Superposition du cadenas si Blocked
		if visualState&entity.Blocked != 0 {
			// Si c'est bloqué, on peut soit changer d'image soit superposer plus tard.
			// Pour l'instant, on utilise l'image 'tile_sealed' si c'est caché et bloqué,
			// ou 'tile_blocked' (croix rouge) si c'est révélé et bloqué.
			if visualState&entity.Revealed != 0 {
				tileImg = r.assets.GetImage("tile_blocked")
			} else {
				tileImg = r.assets.GetImage("tile_sealed")
			}
		}
	}

	scale := r.tileSize / 64.0
	center := 32.0 // centre de l'image source 64x64

	op := &ebiten.DrawImageOptions{}

	if r.boardRotation != 0 {
		op.GeoM.Translate(-center, -center)
		op.GeoM.Rotate(r.boardRotation * math.Pi / 180)
		op.GeoM.Translate(center, center)
	}

	if animation != nil && animation.IsActive() {
		rotateX, rotateY := animation.GetCurrentRotation()
		if rotateX != 0 {
			op.GeoM.Translate(0, -center)
			op.GeoM.Scale(1, math.Abs(math.Cos(rotateX*math.Pi/180)))
			op.GeoM.Translate(0, center)
		}
		if rotateY != 0 {
			op.GeoM.Translate(-center, 0)
			op.GeoM.Scale(math.Abs(math.Cos(rotateY*math.Pi/180)), 1)
			op.GeoM.Translate(center, 0)
		}
	} else if visualState&entity.Matched != 0 {
		// "Raised" effect: matched tiles are 20% larger
		scale := 1.2
		op.GeoM.Translate(-centerX, -centerY)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(centerX, centerY)
	}

	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(tileImg, op)

	shouldShowContent := visualState == entity.Revealed || visualState == entity.Matched
	if animation != nil && animation.IsActive() {
		if animation.TileState&entity.Hidden != 0 {
			shouldShowContent = animation.Progress < 0.5
		} else {
			shouldShowContent = animation.Progress > 0.5
		}
	}

	if ent.GetType() == entity.TypeStructure {
		// print pour débugger l'affichage des structures si nécessaire
		// fmt.Printf("[RENDER] Structure %s (tags: %v) at %v state %s\n", ent.GetID(), ent.HasTag("commencement_portal"), plot.Position, visualState)
	}

	if shouldShowContent && ent.GetType() != entity.TypeTrap {
		r.renderEntityAt(screen, x, y, ent)
	}
}

// renderPlot dessine une case individuelle (utilise l'ancien offset - pour compatibilité)
func (r *BoardRenderer) renderPlot(screen *ebiten.Image, pos board.Position, tile *board.Plot, world *domain.World) {
	x := r.gridOffsetX + float64(pos.X)*r.tileSize
	y := r.gridOffsetY + float64(pos.Y)*r.tileSize
	r.renderTileAt(screen, x, y, "", tile, world)
}

// renderEntityAt dessine une entité à une position écran spécifique
func (r *BoardRenderer) renderEntityAt(screen *ebiten.Image, x, y float64, e entity.Entity) {
	centerX := float32(x + r.tileSize/2)
	scale := r.tileSize / 64.0
	iconScale := 0.75 * scale

	switch ent := e.(type) {
	case *domain.Creature:
		icon := r.assets.GetCreatureIcon(ent.Species)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-32, -32) // centre de l'image 64x64
		op.GeoM.Scale(iconScale, iconScale)
		op.GeoM.Translate(float64(x)+float64(r.tileSize)/2, float64(y)+float64(r.tileSize)/2)
		screen.DrawImage(icon, op)

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
		stageName := ent.Lifecycle.GetCurrentStageName()
		icon := r.assets.GetResourceIcon(ent.ResourceType, stageName)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-32, -32)
		op.GeoM.Scale(iconScale, iconScale)
		op.GeoM.Translate(float64(x)+float64(r.tileSize)/2, float64(y)+float64(r.tileSize)/2)
		screen.DrawImage(icon, op)

		if len(stageName) > 0 {
			label := string(stageName[0])
			text.Draw(screen, label, basicfont.Face7x13, int(x)+int(r.tileSize)-12, int(y)+int(r.tileSize)-5, color.White)
		}
	}
}

// renderEntity dessine une entité sur une tuile (ancienne méthode pour compatibilité)
func (r *BoardRenderer) renderEntity(screen *ebiten.Image, x, y int, e entity.Entity) {
	r.renderEntityAt(screen, float64(x), float64(y), e)
}

// ScreenToGrid convertit les coordonnées écran en coordonnées grille et gridID
func (r *BoardRenderer) ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool) {
	if r.singleGridID != "" {
		grid, ok := world.GetGrid(r.singleGridID)
		if !ok {
			return board.Position{}, "", false
		}
		x := float64(screenX) - r.gridOffsetX
		y := float64(screenY) - r.gridOffsetY
		if x < 0 || y < 0 {
			return board.Position{}, "", false
		}
		gridX := int(x / r.tileSize)
		gridY := int(y / r.tileSize)
		if gridX < grid.Width && gridY < grid.Height {
			return board.Position{X: gridX, Y: gridY}, r.singleGridID, true
		}
		return board.Position{}, "", false
	}

	for _, gridID := range world.GridOrder {
		offsetX, offsetY, grid := r.getGridLayout(gridID, world)
		if grid == nil {
			continue
		}

		x := float64(screenX) - offsetX
		y := float64(screenY) - offsetY

	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

		gridX := int(x / r.tileSize)
		gridY := int(y / r.tileSize)

			// Si on est en zone de portail, on ignore les cases non-interactives
			if isPortalZone && !r.isPortalPosition(pos) {
				continue
			}

			sx, sy := r.calculateTileScreenPos(pos, grid, isPortalZone)
			if float64(screenX) >= sx && float64(screenX) < sx+r.tileSize &&
				float64(screenY) >= sy && float64(screenY) < sy+r.tileSize {
				return pos, gridID, true
			}
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
	tileScreenX := offsetX + float64(pos.X)*r.tileSize
	tileScreenY := offsetY + float64(pos.Y)*r.tileSize
	lx := screenX - int(tileScreenX)
	ly := screenY - int(tileScreenY)

	return int(lx), int(ly), gID, true
}

// RenderSelectionHighlight dessine une surbrillance sur une tuile sélectionnée
func (r *BoardRenderer) RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, highlightColor color.Color, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	x := offsetX + float64(pos.X)*r.tileSize
	y := offsetY + float64(pos.Y)*r.tileSize
	vector.StrokeRect(screen, float32(x), float32(y), float32(r.tileSize), float32(r.tileSize), 3, highlightColor, true)
}

func (r *BoardRenderer) RenderPortalPlacementPreview(screen *ebiten.Image, center board.Position, gridID string, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	if center.X < 1 || center.Y < 1 || center.X > grid.Width-2 || center.Y > grid.Height-2 {
		return
	}

	spacingX, spacingY, _, _ := r.getGridSpacing(grid.Width, grid.Height)
	if isPortalZone {
		spacingX, spacingY, _, _ = r.getGridSpacing(6, 6)
	}

	topLeft := board.Position{X: center.X - 1, Y: center.Y - 1}
	x, y := r.calculateTileScreenPos(topLeft, grid, isPortalZone)
	width := 3*r.tileSize + 2*spacingX
	height := 3*r.tileSize + 2*spacingY
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 4, color.RGBA{0, 200, 100, 120}, true)
}
