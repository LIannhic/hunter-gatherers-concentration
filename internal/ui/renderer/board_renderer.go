// Package renderer gère l'affichage du jeu
package renderer

import (
	"fmt"
	"image/color"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/actionbuttons"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// BoardRenderer dessine le plateau de jeu
type BoardRenderer struct {
	assets      *assets.Manager
	tileSize    float64
	gridOffsetX float64 // Position écran du coin haut-gauche du plateau
	gridOffsetY float64 //
	gridSpacing int     // Espace entre les cases (parcelles)
	gridsPerRow int     // Nombre de cases par ligne

	// Rotation visuelle globale du plateau (en degrés)
	boardRotation float64

	// Animations de flip en cours: clé = "gridID:x,y", valeur = état de l'animation
	flipAnimations map[string]*FlipAnimation

	// Gestionnaire réactif des boutons d'action
	ActionButtons *actionbuttons.Manager

	// Effet renderer pour les shaders
	effectRenderer *EffectRenderer

	// Effets de scanner en cours: clé = gridID
	activeScannerEffects map[string]*ScannerEffect

	// États de survol et rebond pour les animations avancées
	hoverStates  map[string]*HoverState  // Clé: EntityID
	bounceStates map[string]*BounceState // Clé: EntityID

	trackRenderer *TrackRenderer
	// Animation manager pour translations et calques
	AnimManager *AnimationManager
}

// HoverState suit le progrès du survol pour une tuile
type HoverState struct {
	Progress float32 // 0.0 à 1.0
	Dir      entity.FlipDirection
}

// BounceState suit le progrès du rebond élastique après un flip
type BounceState struct {
	ImpactT float32 // 0.0 à 1.0
	Dir     entity.FlipDirection
}

// FlipAnimation représente l'état d'une animation de flip
type FlipAnimation struct {
	GridID         string
	Position       board.Position
	FlipDirection  entity.FlipDirection
	StartTransform entity.Transformation
	EndTransform   entity.Transformation
	Progress       float64 // 0.0 à 1.0
	Speed          float64
	EntityID       string           // L'entité à afficher à la fin
	TileState      entity.TileState // État final de la tuile
}

// ScannerEffect représente l'état d'un effet de scanner
type ScannerEffect struct {
	GridID    string
	Positions []board.Position
	Progress  float64 // 0.0 à 1.0
	Duration  float64 // Durée totale en secondes
	Elapsed   float64 // Temps écoulé en secondes
}

// IsActive retourne true si l'animation est en cours
func (a *FlipAnimation) IsActive() bool {
	return a.Progress < 1.0
}

// NewBoardRenderer crée un nouveau renderer
func NewBoardRenderer(am *assets.Manager) *BoardRenderer {
	effectRenderer, err := NewEffectRenderer()
	if err != nil {
		fmt.Printf("[ERROR] Failed to initialize effect renderer: %v\n", err)
		effectRenderer = nil
	}

	r := &BoardRenderer{
		assets:               am,
		tileSize:             ui.TileSize,
		gridOffsetX:          ui.PlaymatX + ui.BoardRelativeX,
		gridOffsetY:          ui.PlaymatY + ui.BoardRelativeY,
		gridSpacing:          0,
		gridsPerRow:          1,
		flipAnimations:       make(map[string]*FlipAnimation),
		effectRenderer:       effectRenderer,
		activeScannerEffects: make(map[string]*ScannerEffect),
		hoverStates:          make(map[string]*HoverState),
		bounceStates:         make(map[string]*BounceState),
		trackRenderer:        NewTrackRenderer(ui.TileSize),
	}
	// Initialise le gestionnaire d'animations lié au renderer
	r.AnimManager = NewAnimationManager(r)
	return r
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

// ClearAnimations arrête toutes les animations de flip en cours
func (r *BoardRenderer) ClearAnimations() {
	r.flipAnimations = make(map[string]*FlipAnimation)
}

// StartFlipAnimation démarre une animation de flip pour une tuile
func (r *BoardRenderer) StartFlipAnimation(gridID string, pos board.Position, flipDir entity.FlipDirection, entityID string, finalState entity.TileState, startTrans, endTrans entity.Transformation) {
	key := fmt.Sprintf("%s:%d,%d", gridID, pos.X, pos.Y)
	r.flipAnimations[key] = &FlipAnimation{
		GridID:         gridID,
		Position:       pos,
		FlipDirection:  flipDir,
		StartTransform: startTrans,
		EndTransform:   endTrans,
		Progress:       0.0,
		Speed:          1.5 / (flipDuration * 60.0),
		EntityID:       entityID,
		TileState:      finalState,
	}
}

// UpdateAnimations met à jour toutes les animations de flip, survol et rebond
func (r *BoardRenderer) UpdateAnimations(world *domain.World) {
	for key, anim := range r.flipAnimations {
		anim.Progress += anim.Speed
		if anim.Progress >= 1.0 {
			anim.Progress = 1.0
			if ent, ok := world.Entities.Get(entity.ID(anim.EntityID)); ok {
				ent.SetTransformation(anim.EndTransform)
			}
			r.bounceStates[anim.EntityID] = &BounceState{ImpactT: 0, Dir: anim.FlipDirection}
			delete(r.flipAnimations, key)

			// Publie l'événement de fin pour débloquer l'UI
			world.EventBus.PublishImmediate(event.Event{
				Type:     event.AnimationEnded,
				SourceID: anim.EntityID,
				Payload: map[string]interface{}{
					"grid_id":  anim.GridID,
					"position": anim.Position,
				},
			})
		}
	}

	for id, bounce := range r.bounceStates {
		bounce.ImpactT += 0.04
		if bounce.ImpactT >= 1.0 {
			delete(r.bounceStates, id)
		}
	}

	// Update des translations gérées par le AnimationManager
	if r.AnimManager != nil {
		r.AnimManager.Update(world)
	}
}

// NotifyHover informe le renderer qu'une entité est survolée
func (r *BoardRenderer) NotifyHover(entityID string, dir entity.FlipDirection) {
	state, ok := r.hoverStates[entityID]
	if !ok {
		state = &HoverState{Progress: 0, Dir: dir}
		r.hoverStates[entityID] = state
	}
	state.Dir = dir
	state.Progress += 0.1
	if state.Progress > 1.0 {
		state.Progress = 1.0
	}
}

// DecayHoverStates réduit le hover des entités non notifiées
func (r *BoardRenderer) DecayHoverStates(activeThisFrame map[string]bool) {
	for id, state := range r.hoverStates {
		if !activeThisFrame[id] {
			state.Progress -= 0.1
			if state.Progress <= 0 {
				delete(r.hoverStates, id)
			}
		}
	}
}

// UpdateEffects met à jour tous les effets actifs
func (r *BoardRenderer) UpdateEffects(deltaTime float64) {
	for gridID, effect := range r.activeScannerEffects {
		effect.Elapsed += deltaTime
		effect.Progress = effect.Elapsed / effect.Duration
		if effect.Progress >= 1.0 {
			effect.Progress = 1.0
			delete(r.activeScannerEffects, gridID)
		}
	}
}

// SubscribeToEvents inscrit le renderer aux événements du monde
func (r *BoardRenderer) SubscribeToEvents(world *domain.World) {
	world.EventBus.SubscribeFunc(event.Type("scanner_triggered"), func(e event.Event) {
		if gridID, ok := e.Payload["grid_id"].(string); ok {
			positions, _ := e.Payload["positions"].([]board.Position)
			r.activeScannerEffects[gridID] = &ScannerEffect{
				GridID:    gridID,
				Positions: positions,
				Progress:  0.5,
				Duration:  3.0,
				Elapsed:   0.0,
			}
		}
	})

	// Démarre les animations de translation quand une créature se déplace
	world.EventBus.SubscribeFunc(event.CreatureMoved, func(e event.Event) {
		// On ne saute l'animation QUE si hidden est explicitement à true (furtivité)
		if hidden, ok := e.Payload["hidden"].(bool); ok && hidden {
			return
		}

		// Récupère les positions
		from, _ := e.Payload["from"].(entity.Position)
		to, _ := e.Payload["to"].(entity.Position)
		entityID, _ := e.Payload["entity_id"].(string)
		if entityID == "" {
			entityID = e.SourceID
		}

		// Mode détermine la strate de rendu (under, normal, over)
		var layer Layer = LayerNormal
		if modeStr, ok := e.Payload["mode"].(string); ok {
			switch modeStr {
			case "under":
				layer = LayerUnder
			case "over":
				layer = LayerOver
			default:
				layer = LayerNormal
			}
		}

		if r.AnimManager != nil {
			// Durée: 60 ticks = ~1s à 60fps
			r.AnimManager.StartTileMove(world, world.CurrentGridID, entityID, board.Position{X: from.X, Y: from.Y}, board.Position{X: to.X, Y: to.Y}, 60, layer)
		}
	})
}

// GetTileSize retourne la taille des tuiles
func (r *BoardRenderer) GetTileSize() int {
	return int(r.tileSize)
}

// GetGridOffset retourne le décalage du plateau
func (r *BoardRenderer) GetGridOffset() (int, int) {
	return int(r.gridOffsetX), int(r.gridOffsetY)
}

// ApplyTransformation applique une transformation D4 à une matrice GeoM d'Ebiten.
func (r *BoardRenderer) ApplyTransformation(geom *ebiten.GeoM, t entity.Transformation) {
	switch t {
	case entity.TransRot90:
		geom.Rotate(math.Pi / 2)
	case entity.TransRot180:
		geom.Rotate(math.Pi)
	case entity.TransRot270:
		geom.Rotate(3 * math.Pi / 2)
	case entity.TransMirrorH:
		geom.Scale(-1, 1)
	case entity.TransMirrorV:
		geom.Scale(1, -1)
	case entity.TransMirrorD1: // Diagonale \
		geom.Rotate(math.Pi / 2)
		geom.Scale(1, -1)
	case entity.TransMirrorD2: // Diagonale /
		geom.Rotate(-math.Pi / 2)
		geom.Scale(1, -1)
	}
}

// =========================================================================
// PIPELINE DE RENDU GÉNÉRAL (STRUCTURE EN COUCHES STRATIFIÉES)
// =========================================================================

func (r *BoardRenderer) Render(screen *ebiten.Image, world *domain.World) {
	// 1. On dessine l'arrière-plan global fixe (Playmat + ses Exits)
	r.renderPlaymat(screen, world)

	r.UpdateAnimations(world)
	r.UpdateEffects(1.0 / 60.0)

	if world.CurrentGridID != "" {
		grid, _ := world.GetGrid(world.CurrentGridID)
		if grid != nil {
			r.boardRotation = float64(int(grid.MainBearing) * 90)
		}

		// --- 2. COUCHE : FOND DE LA GRILLE (Les cases vides) ---
		// Toujours rendu sur le plateau (Board area)
		r.renderEmptyGrid(screen, world.CurrentGridID, world, false)

		isPortalZone := world.DreamPlane != nil && (world.CurrentGridID == world.DreamPlane.StartZoneID || world.CurrentGridID == world.DreamPlane.EndZoneID)
		getCenter := func(pos board.Position) (float64, float64) {
			return r.getTileCenter(pos, grid, isPortalZone)
		}

		// --- 3. STRATE : UNDER (Souterraine) ---
		r.renderTracksUnder(screen, world, getCenter)
		r.renderMovementsUnder(screen, world)
		r.renderEffectsUnder(screen, world)

		// --- 4. STRATE : TUILES & NORMAL ---
		// Les traces normales sont sous les tuiles statiques
		r.renderTracksNormal(screen, world, getCenter)
		r.renderGrid(screen, world.CurrentGridID, world, false, false)

		r.renderMovementsNormal(screen, world)
		r.renderEffectsNormal(screen, world, getCenter)

		// --- 5. STRATE : OVER (Au-dessus de la surface) ---
		r.renderTracksOver(screen, world, getCenter)
		r.renderMovementsOver(screen, world)

		// Le scanner glisse au-dessus de tout le monde sur le plateau
		r.renderEffectsOver(screen, world)
	}

	// 6. On dessine l'interface utilisateur tout en haut
	r.renderActionButtons(screen)
}

// =========================================================================
// DÉTAILS DES STRATES VISUELLES (TRACES, DÉPLACEMENTS, EFFETS)
// =========================================================================

func (r *BoardRenderer) renderEmptyGrid(screen *ebiten.Image, gridID string, world *domain.World, isLocalToPlaymat bool) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	// Si zone de portail, on affiche toujours une grille 6x6 d'emplacements vides
	renderWidth, renderHeight := grid.Width, grid.Height
	if isPortalZone {
		renderWidth, renderHeight = 6, 6
	}

	for y := 0; y < renderHeight; y++ {
		for x := 0; x < renderWidth; x++ {
			pos := board.Position{X: x, Y: y}

			if isPortalZone && !r.isPortalPosition(pos) {
				continue
			}

			var sx, sy float64
			if isLocalToPlaymat {
				absX, absY := r.calculateTileScreenPos(pos, grid, isPortalZone)
				sx = absX - ui.PlaymatX
				sy = absY - ui.PlaymatY
			} else {
				sx, sy = r.calculateTileScreenPos(pos, grid, isPortalZone)
			}

			r.renderEmptySquareAt(screen, sx, sy)
		}
	}
}

// Fonctions de branchement des calques Under/Normal/Over vers ton gestionnaire
func (r *BoardRenderer) renderTracksUnder(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	if r.trackRenderer != nil {
		r.trackRenderer.RenderUnder(screen, world, getCenter)
	}
}
func (r *BoardRenderer) renderMovementsUnder(screen *ebiten.Image, world *domain.World) {
	r.renderMovingEntities(screen, world, "under")
}
func (r *BoardRenderer) renderEffectsUnder(screen *ebiten.Image, world *domain.World) {
	// Évolutions futures : secousses sismiques souterraines
}

func (r *BoardRenderer) renderTracksNormal(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	if r.trackRenderer != nil {
		r.trackRenderer.RenderNormal(screen, world, getCenter)
	}
}
func (r *BoardRenderer) renderMovementsNormal(screen *ebiten.Image, world *domain.World) {
	r.renderMovingEntities(screen, world, "normal")
}
func (r *BoardRenderer) renderEffectsNormal(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	if r.trackRenderer != nil {
		r.trackRenderer.RenderEffectsNormal(screen, world, getCenter)
	}
}

func (r *BoardRenderer) renderTracksOver(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	if r.trackRenderer != nil {
		r.trackRenderer.RenderOver(screen, world, getCenter)
	}
}
func (r *BoardRenderer) renderMovementsOver(screen *ebiten.Image, world *domain.World) {
	r.renderMovingEntities(screen, world, "over")
}
func (r *BoardRenderer) renderEffectsOver(screen *ebiten.Image, world *domain.World) {
	if world.CurrentGridID != "" {
		r.renderScannerEffects(screen, world.CurrentGridID, world)
	}
}

// =========================================================================
// ARCHITECTURE DE COMPOSANTS STANDARD MODIFIÉE
// =========================================================================

func (r *BoardRenderer) renderPlaymat(screen *ebiten.Image, world *domain.World) {
	vector.StrokeRect(screen, ui.PlaymatX, ui.PlaymatY, ui.PlaymatW, ui.PlaymatH, 1, color.RGBA{100, 100, 100, 255}, true)

	r.renderExitTiles(screen, ui.ExitNorthX, ui.ExitNorthY, board.North, world, false, false)
	r.renderExitTiles(screen, ui.ExitEastX, ui.ExitEastY, board.East, world, false, false)
	r.renderExitTiles(screen, ui.ExitSouthX, ui.ExitSouthY, board.South, world, false, false)
	r.renderExitTiles(screen, ui.ExitWestX, ui.ExitWestY, board.West, world, false, false)
}

func (r *BoardRenderer) renderActionButtons(screen *ebiten.Image) {
	if r.ActionButtons == nil {
		return
	}
	states := r.ActionButtons.ComputeStates()
	for i := 0; i < 4; i++ {
		r.renderSingleButton(screen, states[i])
	}
}

func (r *BoardRenderer) renderSingleButton(screen *ebiten.Image, s actionbuttons.ButtonState) {
	var bgColor color.Color
	if s.Scrambled {
		bgColor = color.RGBA{120, 80, 80, 255}
	} else if s.Active {
		bgColor = color.RGBA{60, 60, 80, 255}
	} else {
		bgColor = color.RGBA{40, 40, 40, 255}
	}

	vector.DrawFilledRect(screen, float32(s.X), float32(s.Y), float32(s.Width), float32(s.Height), bgColor, true)

	if (s.ID == actionbuttons.BtnSkip || s.ID == actionbuttons.BtnEndTurn) && s.FillProgress > 0 {
		fillW := float32(s.Width * s.FillProgress)
		var fillColor color.Color
		if s.FillAlert {
			fillColor = color.RGBA{180, 50, 50, 200}
		} else {
			fillColor = color.RGBA{100, 80, 150, 160}
		}
		vector.DrawFilledRect(screen, float32(s.X), float32(s.Y), fillW, float32(s.Height), fillColor, true)
	}

	var borderColor color.Color
	if s.Active && !s.Scrambled {
		borderColor = color.RGBA{200, 200, 255, 255}
	} else if s.Active && s.Scrambled {
		borderColor = color.RGBA{255, 100, 100, 255}
	} else {
		borderColor = color.RGBA{80, 80, 80, 255}
	}
	vector.StrokeRect(screen, float32(s.X), float32(s.Y), float32(s.Width), float32(s.Height), 1, borderColor, true)

	var labelColor color.Color = color.White
	if !s.Active {
		labelColor = color.RGBA{120, 120, 120, 255}
	} else if s.Scrambled {
		labelColor = color.RGBA{255, 200, 200, 255}
	}

	text.Draw(screen, s.Label, basicfont.Face7x13, int(s.X+ui.ButtonTextRelativeX), int(s.Y+ui.ButtonTextRelativeY+15), labelColor)

	if s.Active {
		ix := s.X + ui.ButtonIconRelativeX
		iy := s.Y + ui.ButtonIconRelativeY
		indicatorColor := color.RGBA{100, 255, 100, 200}
		if s.Scrambled {
			indicatorColor = color.RGBA{255, 100, 100, 200}
		}
		vector.DrawFilledRect(screen, float32(ix), float32(iy), float32(ui.ButtonIconSize), float32(ui.ButtonIconSize), indicatorColor, true)
	}
}

func (r *BoardRenderer) renderExitTiles(screen *ebiten.Image, rx, ry float64, dir entity.Direction, world *domain.World, forceReveal bool, isLocalToPlaymat bool) {
	isOpen := world.IsNavigationOpen(world.CurrentGridID)
	hasExit := false
	if world.DreamPlane != nil {
		_, hasExit = world.DreamPlane.GetConnectedZone(world.CurrentGridID, dir)
	}
	if !hasExit {
		return
	}

	var ex, ey float64
	if isLocalToPlaymat {
		ex = rx
		ey = ry
	} else {
		ex = ui.PlaymatX + rx
		ey = ui.PlaymatY + ry
	}

	grid, _ := world.GetGrid(world.CurrentGridID)
	numTiles := 2
	isVertical := (dir == entity.DirEast || dir == entity.DirWest)

	for i := 0; i < numTiles; i++ {
		tx, ty := ex, ey
		if isVertical {
			ty += float64(i) * r.tileSize
		} else {
			tx += float64(i) * r.tileSize
		}

		var tileState entity.TileState = entity.Hidden | entity.Blocked
		if grid != nil {
			tileState = grid.ExitsState[dir][i]
		}

		if forceReveal {
			tileState |= entity.Revealed
			tileState &= ^entity.Hidden
		}

		if !hasExit || !isOpen {
			tileState |= entity.Blocked
		} else {
			tileState &= ^entity.Blocked
		}

		var tileImg *ebiten.Image
		if tileState&entity.Blocked != 0 {
			if tileState&entity.Revealed != 0 {
				tileImg = r.assets.GetImage("tile_blocked")
			} else {
				tileImg = r.assets.GetImage("tile_sealed")
			}
		} else if tileState&entity.Revealed != 0 {
			tileImg = r.assets.GetImage("tile_exit")
		} else {
			tileImg = r.assets.GetImage("tile_hidden")
		}

		// --- GESTION DES ANIMATIONS (FLIP & HOVER) ---
		entityID := fmt.Sprintf("exit_%s_%d", directionToName(dir), i)

		var animation *FlipAnimation
		for _, anim := range r.flipAnimations {
			if anim.EntityID == entityID && anim.GridID == world.CurrentGridID {
				animation = anim
				break
			}
		}

		if animation != nil && animation.IsActive() {
			grid, _ := world.GetGrid(world.CurrentGridID)
			themeName := "default"
			if grid != nil {
				themeName = string(grid.Biome)
			}
			theme := r.assets.GetTheme(themeName)
			r.renderFlippingTile(screen, tx, ty, animation, nil, themeName, theme.HiddenBorder)
			continue
		}

		op := &ebiten.DrawImageOptions{}

		// Gestion de la rotation et du miroir pour former la flèche
		if tileState&entity.Revealed != 0 && tileImg == r.assets.GetImage("tile_exit") {
			op.GeoM.Translate(-r.tileSize/2, -r.tileSize/2)

			// Rotation de base selon la direction de la sortie
			var angle float64
			switch dir {
			case entity.DirEast:
				angle = math.Pi / 2
			case entity.DirSouth:
				angle = math.Pi
			case entity.DirWest:
				angle = -math.Pi / 2
			}
			op.GeoM.Rotate(angle)

			// Effet miroir pour la deuxième tuile afin de compléter la flèche
			if i == 1 {
				op.GeoM.Scale(-1, 1)
			}

			op.GeoM.Translate(tx+r.tileSize/2, ty+r.tileSize/2)
		} else {
			op.GeoM.Translate(tx, ty)
		}

		// Application du Hover
		hover, hasHover := r.hoverStates[entityID]
		if hasHover && hover.Progress > 0 {
			// Création d'une géométrie temporaire pour le hover
			margin := (r.tileSize - ui.FaceSize) / 2
			htx, hty := float32(tx+margin), float32(ty+margin)

			// On simule une entité vide pour la géométrie
			geo := r.generateIdleGeometry(htx, hty, entityID, color.RGBA{100, 100, 200, 255})
			r.drawSlices(screen, geo, hover.Dir, r.assets.GetImage("white"))

			// Ajustement de l'opacité/élévation simple si on n'utilise pas drawSlices complet
			scale := 1.0 + 0.05*float64(hover.Progress)
			op.GeoM.Translate(-tx-r.tileSize/2, -ty-r.tileSize/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(tx+r.tileSize/2, ty+r.tileSize/2)
		}

		screen.DrawImage(tileImg, op)
	}
}

func directionToName(dir entity.Direction) string {
	switch dir {
	case entity.DirNorth:
		return "north"
	case entity.DirEast:
		return "east"
	case entity.DirSouth:
		return "south"
	case entity.DirWest:
		return "west"
	}
	return "unknown"
}

func (r *BoardRenderer) getGridSpacing(gridWidth, gridHeight int) (spacingX, spacingY, padX, padY float64) {
	if gridWidth <= 3 {
		spacingX = (ui.BoardW - float64(gridWidth)*r.tileSize) / float64(gridWidth+1)
		padX = spacingX
	} else if gridWidth > 1 {
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

func (r *BoardRenderer) calculateTileScreenPos(pos board.Position, grid *board.Grid, isPortalZone bool) (float64, float64) {
	spacingX, spacingY, padX, padY := r.getGridSpacing(grid.Width, grid.Height)
	if isPortalZone {
		spacingX, spacingY, padX, padY = r.getGridSpacing(6, 6)
	}
	sx := r.gridOffsetX + padX + float64(pos.X)*(r.tileSize+spacingX)
	sy := r.gridOffsetY + padY + float64(pos.Y)*(r.tileSize+spacingY)
	return sx, sy
}

// getTileCenter retourne le centre d'une case en coordonnées écran, avec rotation globale appliquée
func (r *BoardRenderer) getTileCenter(pos board.Position, grid *board.Grid, isPortal bool) (float64, float64) {
	x, y := r.calculateTileScreenPos(pos, grid, isPortal)
	cx, cy := x+r.tileSize/2, y+r.tileSize/2

	if r.boardRotation != 0 {
		// Le plateau pivote autour de son centre géométrique
		boardCenterX := r.gridOffsetX + ui.BoardW/2
		boardCenterY := r.gridOffsetY + ui.BoardH/2

		angle := r.boardRotation * math.Pi / 180
		cosA, sinA := math.Cos(angle), math.Sin(angle)

		relX := cx - boardCenterX
		relY := cy - boardCenterY

		cx = boardCenterX + relX*cosA - relY*sinA
		cy = boardCenterY + relX*sinA + relY*cosA
	}

	return cx, cy
}

func (r *BoardRenderer) isPortalPosition(pos board.Position) bool {
	portalPositions := []board.Position{
		{X: 1, Y: 1}, {X: 1, Y: 4}, {X: 4, Y: 1}, {X: 4, Y: 4},
		{X: 2, Y: 2}, {X: 2, Y: 3}, {X: 3, Y: 2}, {X: 3, Y: 3},
	}
	for _, p := range portalPositions {
		if p == pos {
			return true
		}
	}
	return false
}

// renderGrid s'occupe UNIQUEMENT d'empiler les tuiles immobiles
func (r *BoardRenderer) renderGrid(screen *ebiten.Image, gridID string, world *domain.World, forceReveal bool, isLocalToPlaymat bool) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}

			var sx, sy float64
			if isLocalToPlaymat {
				absX, absY := r.calculateTileScreenPos(pos, grid, isPortalZone)
				sx = absX - ui.PlaymatX
				sy = absY - ui.PlaymatY
			} else {
				sx, sy = r.calculateTileScreenPos(pos, grid, isPortalZone)
			}

			plot, ok := grid.Plots[pos]
			// Si pas de plot physique et pas d'entités, on ne fait rien.
			// Le fond de grille vide a déjà été traité au calque 2 par renderEmptyGrid.
			if !ok || len(plot.EntitiesID) == 0 {
				continue
			}

			// Rendu intelligent de la pile logique
			topID := plot.EntitiesID[len(plot.EntitiesID)-1]
			isAnimating := world.Components.Has(topID, "moving_animation")

			if isAnimating {
				// Si la tuile du dessus bouge, elle est dessinée par renderMovingEntities (strates 1, 2 ou 3).
				// Ici, on dessine ce qu'il y a EN DESSOUS (si présent) pour éviter un trou visuel sur la grille.
				if len(plot.EntitiesID) > 1 {
					underID := plot.EntitiesID[len(plot.EntitiesID)-2]
					r.renderSingleTileIDAt(screen, sx, sy, gridID, underID, world, forceReveal, 1.0)
				}
			} else {
				// Rendu normal de la tuile au sommet (état stable)
				r.renderSingleTileIDAt(screen, sx, sy, gridID, topID, world, forceReveal, 1.0)
			}
		}
	}
}

func (r *BoardRenderer) renderEmptySquareAt(screen *ebiten.Image, x, y float64) {
	tileImg := r.assets.GetImage("square_empty")
	if tileImg == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(tileImg, op)
}

// renderTileAt a été scindée pour plus de clarté
func (r *BoardRenderer) renderTileAt(screen *ebiten.Image, x, y float64, gridID string, plot *board.Plot, world *domain.World, forceReveal bool, alpha float32) {
	if len(plot.EntitiesID) == 0 {
		return
	}
	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	r.renderSingleTileIDAt(screen, x, y, gridID, topID, world, forceReveal, alpha)
}

func (r *BoardRenderer) renderSingleTileIDAt(screen *ebiten.Image, x, y float64, gridID string, entityID string, world *domain.World, forceReveal bool, alpha float32) {
	ent, ok := world.Entities.Get(entity.ID(entityID))
	if !ok {
		return
	}

	visualState := ent.GetState()
	if forceReveal {
		visualState |= entity.Revealed
		visualState &= ^entity.Hidden
	}

	grid, _ := world.GetGrid(gridID)
	themeName := "default"
	if grid != nil {
		themeName = string(grid.Biome)
	}

	var animation *FlipAnimation
	for _, anim := range r.flipAnimations {
		// Note: On vérifie l'EntityID car la position logique a déjà changé lors du mouvement
		if anim.EntityID == entityID && (gridID == "" || anim.GridID == gridID) {
			animation = anim
			break
		}
	}

	isFlipping := animation != nil && animation.IsActive() && !forceReveal
	if isFlipping {
		theme := r.assets.GetTheme(themeName)
		r.renderFlippingTile(screen, x, y, animation, ent, themeName, theme.HiddenBorder)
		return
	}

	var tileImg *ebiten.Image
	if visualState&entity.Matched != 0 {
		tileImg = r.assets.GetImage("tile_matched")
	} else if visualState&entity.Revealed != 0 {
		tileImg = r.getEntityRevealedImage(ent, themeName)
	} else {
		tileImg = r.assets.GetTileImage("hidden", themeName)
	}

	// Gestion des Overlays (Blocage / Scellé)
	var overlayImg *ebiten.Image
	if visualState&entity.Blocked != 0 {
		if visualState&entity.Revealed != 0 {
			overlayImg = r.assets.GetImage("tile_blocked")
		} else {
			overlayImg = r.assets.GetImage("tile_sealed")
		}
	}

	margin := (r.tileSize - ui.FaceSize) / 2
	tx, ty := float32(x+margin), float32(y+margin)
	cx, cy := float32(x+r.tileSize/2), float32(y+r.tileSize/2)

	theme := r.assets.GetTheme(themeName)

	geo := r.generateIdleGeometry(tx, ty, entityID, theme.HiddenBorder)
	r.ApplyBoardRotation(geo.V, cx, cy)

	// Application de l'alpha aux sommets pour la transparence (Ghost)
	if alpha < 1.0 {
		for i := range geo.V {
			geo.V[i].ColorA *= alpha
			geo.V[i].ColorR *= alpha
			geo.V[i].ColorG *= alpha
			geo.V[i].ColorB *= alpha
		}
	}

	if visualState&entity.Matched != 0 {
		const matchedScale = 1.2
		for i := range geo.V {
			geo.V[i].DstX = cx + (geo.V[i].DstX-cx)*matchedScale
			geo.V[i].DstY = cy + (geo.V[i].DstY-cy)*matchedScale
		}
	}

	faceImg := tileImg
	backImg := r.assets.GetImage("tile_hidden")

	// 1. Dessin du Dos
	r.drawGeometryPart(screen, geo.V, geo.I[6:12], backImg)
	// 2. Dessin de la Face
	r.drawGeometryPart(screen, geo.V, geo.I[:6], faceImg)
	// 3. Dessin de l'Overlay (si présent)
	if overlayImg != nil {
		r.drawGeometryPart(screen, geo.V, geo.I[:6], overlayImg)
	}

	id := string(ent.GetID())
	hover, hasHover := r.hoverStates[id]
	bounce, hasBounce := r.bounceStates[id]
	if (hasHover && hover.Progress > 0) || (hasBounce && bounce.ImpactT < 1.0) {
		hDir := entity.FlipTop
		if hasHover {
			hDir = hover.Dir
		} else if hasBounce {
			hDir = bounce.Dir
		}
		r.drawSlices(screen, geo, hDir, r.assets.GetImage("white"))
	}

	shouldShowContent := visualState&entity.Revealed != 0 || visualState&entity.Matched != 0
	if shouldShowContent && ent.GetType() != entity.TypeTrap {
		r.renderFlippingEntityTriangles(screen, geo.V[:4], ent, ent.GetTransformation())
	}
}

func (r *BoardRenderer) renderEntityAt(screen *ebiten.Image, x, y float64, e entity.Entity) {
	centerX := float32(x + r.tileSize/2)

	switch ent := e.(type) {
	case *domain.Creature:
		icon := r.assets.GetCreatureIcon(ent.Species)
		op := &ebiten.DrawImageOptions{}

		// 1. Centrage de l'asset
		w, h := icon.Bounds().Dx(), icon.Bounds().Dy()
		op.GeoM.Translate(-float64(w)/2, -float64(h)/2)

		// 2. Application de la transformation D4 (Orientation)
		r.ApplyTransformation(&op.GeoM, ent.GetTransformation())

		// 3. Mise à l'échelle et placement final
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

	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			pos := board.Position{X: x, Y: y}
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

func (r *BoardRenderer) ScreenToLocalTile(screenX, screenY int, world *domain.World) (localX, localY int, gridID string, ok bool) {
	pos, gID, found := r.ScreenToGrid(screenX, screenY, world)
	if !found {
		return 0, 0, "", false
	}

	grid, _ := world.GetGrid(gID)
	isPortalZone := world.DreamPlane != nil && (gID == world.DreamPlane.StartZoneID || gID == world.DreamPlane.EndZoneID)

	tileScreenX, tileScreenY := r.calculateTileScreenPos(pos, grid, isPortalZone)
	return int(float64(screenX) - tileScreenX), int(float64(screenY) - tileScreenY), gID, true
}

func (r *BoardRenderer) RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, highlightColor color.Color, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	x, y := r.calculateTileScreenPos(pos, grid, isPortalZone)

	// Effet d'immunité (Shadowstalker) - On remplace la couleur par du gris si actif
	finalColor := highlightColor
	if world.Player != nil && world.Player.ImmunityTurns > 0 {
		finalColor = color.RGBA{150, 150, 160, 255} // Gris pierre/éthéré bien visible
	}

	vector.StrokeRect(screen, float32(x), float32(y), float32(r.tileSize), float32(r.tileSize), 3, finalColor, true)
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

// renderMovingEntities dessine les entités qui ont un composant MovingAnimation actif
func (r *BoardRenderer) renderMovingEntities(screen *ebiten.Image, world *domain.World, layer string) {
	if world.CurrentGridID == "" {
		return
	}

	movingIDs := world.Components.QueryByComponent("moving_animation")

	for _, id := range movingIDs {
		ent, ok := world.Entities.Get(entity.ID(id))
		if !ok || ent.GetGridID() != world.CurrentGridID {
			continue
		}

		// Vérification du layer (Strate)
		entLayer := "normal"
		// Priorise le layer attaché à l'animation (si présent) pour respecter l'intention
		if r.AnimManager != nil {
			if anim, ok := r.AnimManager.animes[id]; ok && anim != nil {
				entLayer = string(anim.Layer)
			}
		}
		if entLayer == "normal" {
			if c, ok := ent.(*domain.Creature); ok && c.MovementProfile != nil {
				entLayer = string(c.MovementProfile.Mode.Type)
			} else if ent.GetType() == entity.TypeResource {
				// Ressources en propagation (toujours normal pour l'instant)
				entLayer = "normal"
			}
		}

		if entLayer != layer {
			continue
		}

		comp, _ := world.Components.Get(id, "moving_animation")
		anim := comp.(*component.MovingAnimation)

		curX := anim.CurrentX
		curY := anim.CurrentY

		// Création d'un plot fictif pour renderTileAt
		fakePlot := &board.Plot{
			Position:   board.Position{X: anim.TargetGridX, Y: anim.TargetGridY},
			EntitiesID: []string{id},
		}

		r.renderTileAt(screen, curX, curY, world.CurrentGridID, fakePlot, world, false, 1.0)
	}
}

func (r *BoardRenderer) renderScannerEffects(screen *ebiten.Image, gridID string, world *domain.World) {
	effect, ok := r.activeScannerEffects[gridID]
	if !ok || r.effectRenderer == nil {
		return
	}

	// 1. Création de l'image source pour le shader (Revealed items)
	playmatW, playmatH := ui.PlaymatW, ui.PlaymatH
	srcImg := ebiten.NewImage(int(playmatW), int(playmatH))
	// On peut remplir avec un fond très sombre pour le style
	srcImg.Fill(color.RGBA{15, 15, 20, 255})

	// 2. Rendu uniquement des positions scannées
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

	for _, pos := range effect.Positions {
		absX, absY := r.calculateTileScreenPos(pos, grid, isPortalZone)
		sx := absX - ui.PlaymatX
		sy := absY - ui.PlaymatY

		plot, ok := grid.Plots[pos]
		if !ok || len(plot.EntitiesID) == 0 {
			continue
		}
		topID := plot.EntitiesID[len(plot.EntitiesID)-1]

		// On force le reveal pour voir ce qu'il y a dessous dans le buffer srcImg
		r.renderSingleTileIDAt(srcImg, sx, sy, gridID, topID, world, true, 1.0)
	}

	// 3. Paramètres de l'animation pour le shader
	// On balaie de gauche à droite sur toute la largeur du playmat
	fullWidth := float32(playmatW)
	margin := float32(200.0) // Pour que la vague commence/finisse hors champ

	// Progress va de -margin à fullWidth + margin
	currentX := -margin + float32(effect.Progress)*(fullWidth+2*margin)

	progress := float32(ui.PlaymatX) + currentX
	thickness := float32(120.0)
	erase := progress - thickness*2.5 // Traîne derrière la vague

	revealColor := color.RGBA{100, 200, 255, 255} // Bleu cyan spectral

	// 4. Appel du shader via l'EffectRenderer
	r.effectRenderer.DrawScannerEffect(screen, srcImg, int(ui.PlaymatX), int(ui.PlaymatY), progress, erase, thickness, revealColor)
}

// getEntityRevealedImage retourne l'image révélée appropriée pour une entité et un thème
func (r *BoardRenderer) getEntityRevealedImage(ent entity.Entity, themeName string) *ebiten.Image {
	if ent == nil {
		return r.assets.GetTileImage("revealed", themeName)
	}

	if ent.GetType() == entity.TypeTrap {
		return r.assets.GetTileImage("trap", themeName)
	}

	if ent.GetType() == entity.TypeStructure {
		if ent.HasTag("start_portal") || ent.HasTag("finish_portal") || ent.HasTag("portable_portal") {
			return r.assets.GetTileImage("portal", themeName)
		} else if ent.HasTag("dolmen") {
			return r.assets.GetTileImage("dolmen", themeName)
		} else if ent.HasTag("obelisk") {
			return r.assets.GetTileImage("obelisk", themeName)
		}
		return r.assets.GetTileImage("structure", themeName)
	}

	return r.assets.GetTileImage("revealed", themeName)
}
