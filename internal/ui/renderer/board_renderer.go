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
		tileSize:       ui.TileSize,
		gridOffsetX:    ui.PlaymatX + ui.BoardRelativeX,
		gridOffsetY:    ui.PlaymatY + ui.BoardRelativeY,
		gridSpacing:    0,
		gridsPerRow:    1,
		flipAnimations: make(map[string]*FlipAnimation),
	}
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

// GetTileSize retourne la taille des tuiles (en int pour compatibilité)
func (r *BoardRenderer) GetTileSize() int {
	return int(r.tileSize)
}

// GetGridOffset retourne le décalage du plateau
func (r *BoardRenderer) GetGridOffset() (int, int) {
	return int(r.gridOffsetX), int(r.gridOffsetY)
}

// Render dessine le plateau complet
func (r *BoardRenderer) Render(screen *ebiten.Image, world *domain.World) {
	// Dessine le Playmat
	r.renderPlaymat(screen)

	// Met à jour les animations
	r.UpdateAnimations()

	// Dessine seulement le grid actif sur le playmat
	if world.CurrentGridID != "" {
		r.renderGrid(screen, world.CurrentGridID, world)
	}
}

func (r *BoardRenderer) renderPlaymat(screen *ebiten.Image) {
	// Playmat background
	vector.StrokeRect(screen, ui.PlaymatX, ui.PlaymatY, ui.PlaymatW, ui.PlaymatH, 1, color.RGBA{100, 100, 100, 255}, true)

	// Action buttons
	btnCoords := []struct{ x, y float64 }{
		{ui.ActionBtn1X, ui.ActionBtn1Y},
		{ui.ActionBtn2X, ui.ActionBtn2Y},
		{ui.ActionBtn3X, ui.ActionBtn3Y},
		{ui.ActionBtn4X, ui.ActionBtn4Y},
	}

	for _, c := range btnCoords {
		bx := ui.PlaymatX + c.x
		by := ui.PlaymatY + c.y

		// Button background
		vector.StrokeRect(screen, float32(bx), float32(by), float32(ui.ActionButtonW), float32(ui.ActionButtonH), 1, color.RGBA{150, 150, 150, 255}, true)

		// Button text space
		text.Draw(screen, "ACTION", basicfont.Face7x13, int(bx+ui.ButtonTextRelativeX), int(by+ui.ButtonTextRelativeY+15), color.White)

		// Button icon space
		ix := bx + ui.ButtonIconRelativeX
		iy := by + ui.ButtonIconRelativeY
		vector.StrokeRect(screen, float32(ix), float32(iy), float32(ui.ButtonIconSize), float32(ui.ButtonIconSize), 1, color.RGBA{200, 200, 200, 255}, true)
	}

	// Exits
	r.renderExit(screen, ui.ExitNorthX, ui.ExitNorthY, ui.ExitNorthW, ui.ExitNorthH, "NORTH")
	r.renderExit(screen, ui.ExitEastX, ui.ExitEastY, ui.ExitEastW, ui.ExitEastH, "EAST")
	r.renderExit(screen, ui.ExitSouthX, ui.ExitSouthY, ui.ExitSouthW, ui.ExitSouthH, "SOUTH")
	r.renderExit(screen, ui.ExitWestX, ui.ExitWestY, ui.ExitWestW, ui.ExitWestH, "WEST")
}

func (r *BoardRenderer) renderExit(screen *ebiten.Image, rx, ry, rw, rh float64, label string) {
	ex := ui.PlaymatX + rx
	ey := ui.PlaymatY + ry
	vector.StrokeRect(screen, float32(ex), float32(ey), float32(rw), float32(rh), 1, color.RGBA{100, 100, 255, 255}, true)
}

// getGridSpacing calcule l'espacement et les marges pour remplir l'espace de 525x525
func (r *BoardRenderer) getGridSpacing(gridWidth, gridHeight int) (spacingX, spacingY, padX, padY float64) {
	// Pour les grilles <= 3x3, on ajoute des marges (padding) aux extrémités
	if gridWidth <= 3 {
		spacingX = (ui.BoardW - float64(gridWidth)*r.tileSize) / float64(gridWidth+1)
		padX = spacingX
	} else if gridWidth > 1 {
		// Pour les grilles >= 4x4, on colle aux bords
		spacingX = (ui.BoardW - float64(gridWidth)*r.tileSize) / float64(gridWidth-1)
		padX = 0
	}

	if gridHeight <= 3 {
		spacingY = (ui.BoardH - float64(gridHeight)*r.tileSize) / float64(gridHeight+1)
		padY = spacingY
	} else if gridHeight > 1 {
		spacingY = (ui.BoardH - float64(gridHeight)*r.tileSize) / float64(gridHeight-1)
		padY = 0
	}

	return spacingX, spacingY, padX, padY
}

// calculateTileScreenPos calcule la position X,Y à l'écran pour une coordonnée de grille
func (r *BoardRenderer) calculateTileScreenPos(pos board.Position, grid *board.Grid, isPortalZone bool) (float64, float64) {
	spacingX, spacingY, padX, padY := r.getGridSpacing(grid.Width, grid.Height)
	if isPortalZone {
		spacingX, spacingY, padX, padY = r.getGridSpacing(6, 6)
	}

	sx := r.gridOffsetX + padX + float64(pos.X)*(r.tileSize+spacingX)
	sy := r.gridOffsetY + padY + float64(pos.Y)*(r.tileSize+spacingY)
	return sx, sy
}

// isPortalPosition vérifie si une coordonnée de grille correspond à une tuile interactive en zone de portail
func (r *BoardRenderer) isPortalPosition(pos board.Position) bool {
	portalPositions := []board.Position{
		{X: 1, Y: 1}, {X: 1, Y: 4}, {X: 4, Y: 1}, {X: 4, Y: 4}, // Structures
		{X: 2, Y: 2}, // Portail
	}
	for _, p := range portalPositions {
		if p == pos {
			return true
		}
	}
	return false
}

// renderGrid dessine un grid spécifique
func (r *BoardRenderer) renderGrid(screen *ebiten.Image, gridID string, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	// Rendu de la grille
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}

			sx, sy := r.calculateTileScreenPos(pos, grid, isPortalZone)

			plot, ok := grid.Plots[pos]
			if !ok {
				r.renderEmptySquareAt(screen, sx, sy)
				continue
			}
			r.renderTileAt(screen, sx, sy, gridID, plot, world)
		}
	}
}

// renderEmptySquareAt dessine un carré vide
func (r *BoardRenderer) renderEmptySquareAt(screen *ebiten.Image, x, y float64) {
	tileImg := r.assets.GetImage("square_empty")
	op := &ebiten.DrawImageOptions{}
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
		if animation.TileState == entity.Hidden {
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
		switch visualState {
		case entity.Hidden:
			tileImg = r.assets.GetImage("tile_hidden")
		case entity.Revealed:
			if ent.GetType() == entity.TypeTrap {
				tileImg = r.assets.GetImage("tile_trap")
			} else if ent.GetType() == entity.TypeStructure {
				if ent.HasTag("commencement_portal") || ent.HasTag("finish_portal") {
					tileImg = r.assets.GetImage("tile_portal")
				} else if ent.HasTag("dolmen") {
					tileImg = r.assets.GetImage("tile_dolmen")
				} else if ent.HasTag("obelisk") {
					tileImg = r.assets.GetImage("tile_obelisk")
				} else {
					// Par défaut pour les dolmens et autres
					tileImg = r.assets.GetImage("tile_structure")
				}
			} else {
				tileImg = r.assets.GetImage("tile_revealed")
			}
		case entity.Matched:
			tileImg = r.assets.GetImage("tile_matched")
		case entity.Blocked:
			// Cas spécial : les portails bloqués affichent leur dos (cachés)
			if ent.GetType() == entity.TypeStructure && (ent.HasTag("commencement_portal") || ent.HasTag("finish_portal")) {
				tileImg = r.assets.GetImage("tile_hidden")
			} else {
				tileImg = r.assets.GetImage("tile_blocked")
			}
		default:
			tileImg = r.assets.GetImage("square_empty")
		}
	}

	op := &ebiten.DrawImageOptions{}
	centerX := r.tileSize / 2
	centerY := r.tileSize / 2

	if r.boardRotation != 0 {
		op.GeoM.Translate(-centerX, -centerY)
		op.GeoM.Rotate(r.boardRotation * math.Pi / 180)
		op.GeoM.Translate(centerX, centerY)
	}

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

	op.GeoM.Translate(x, y)
	screen.DrawImage(tileImg, op)

	shouldShowContent := visualState == entity.Revealed || visualState == entity.Matched
	if animation != nil && animation.IsActive() {
		if animation.TileState == entity.Hidden {
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

// renderPlot dessine une case individuelle (ancien offset pour compatibilité)
func (r *BoardRenderer) renderPlot(screen *ebiten.Image, pos board.Position, tile *board.Plot, world *domain.World) {
	x := r.gridOffsetX + float64(pos.X)*r.tileSize
	y := r.gridOffsetY + float64(pos.Y)*r.tileSize
	r.renderTileAt(screen, x, y, "", tile, world)
}

// renderEntityAt dessine une entité à une position écran spécifique
func (r *BoardRenderer) renderEntityAt(screen *ebiten.Image, x, y float64, e entity.Entity) {
	centerX := float32(x + r.tileSize/2)

	switch ent := e.(type) {
	case *domain.Creature:
		icon := r.assets.GetCreatureIcon(ent.Species)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-r.tileSize/2, -r.tileSize/2)
		op.GeoM.Scale(0.75, 0.75)
		op.GeoM.Translate(x+r.tileSize/2, y+r.tileSize/2)
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
		op.GeoM.Translate(-r.tileSize/2, -r.tileSize/2)
		op.GeoM.Scale(0.75, 0.75)
		op.GeoM.Translate(x+r.tileSize/2, y+r.tileSize/2)
		screen.DrawImage(icon, op)

		if len(stageName) > 0 {
			label := string(stageName[0])
			text.Draw(screen, label, basicfont.Face7x13, int(x+r.tileSize-12), int(y+r.tileSize-5), color.White)
		}
	}
}

// renderEntity dessine une entité sur une tuile (ancienne méthode pour compatibilité)
func (r *BoardRenderer) renderEntity(screen *ebiten.Image, x, y int, e entity.Entity) {
	r.renderEntityAt(screen, float64(x), float64(y), e)
}

// ScreenToGrid convertit les coordonnées écran en coordonnées grille et gridID
func (r *BoardRenderer) ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool) {
	gridID := world.CurrentGridID
	if gridID == "" {
		return board.Position{}, "", false
	}

	grid, ok := world.GetGrid(gridID)
	if !ok {
		return board.Position{}, "", false
	}

	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	// On cherche quelle tuile contient les coordonnées du curseur
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}

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

	grid, _ := world.GetGrid(gID)
	isPortalZone := world.DreamPlane != nil && (gID == world.DreamPlane.StartZoneID || gID == world.DreamPlane.EndZoneID)

	tileScreenX, tileScreenY := r.calculateTileScreenPos(pos, grid, isPortalZone)
	lx := float64(screenX) - tileScreenX
	ly := float64(screenY) - tileScreenY

	return int(lx), int(ly), gID, true
}

// RenderSelectionHighlight dessine une surbrillance sur une tuile sélectionnée
func (r *BoardRenderer) RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, highlightColor color.Color, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	x, y := r.calculateTileScreenPos(pos, grid, isPortalZone)
	vector.StrokeRect(screen, float32(x), float32(y), float32(r.tileSize), float32(r.tileSize), 3, highlightColor, true)
}
