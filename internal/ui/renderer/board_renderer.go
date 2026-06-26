// Package renderer gère l'affichage du jeu
package renderer

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
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

	// Effets de séisme en cours: clé = gridID
	activeQuakeEffects map[string]*QuakeEffect

	// États de survol et rebond pour les animations avancées
	hoverStates  map[string]*HoverState  // Clé: EntityID
	bounceStates map[string]*BounceState // Clé: EntityID

	trackRenderer *TrackRenderer
	// Animation manager pour translations et calques
	AnimManager *AnimationManager

	// Hits en attente d'être affichés (creatureID -> position)
	pendingHits map[string]entity.Position

	// Snapshot du playmat de la frame précédente (pour le fantôme de rotation)
	playmatSnapshot *ebiten.Image

	// Frame buffer pour le shader quake (700×700, taille du playmat)
	quakeFrameBuffer *ebiten.Image

	// Debug: entités révélées visuellement sans modifier l'état réel
	debugRevealed map[entity.ID]bool

	// Effet Lumifly actif (nil si aucun)
	lumiflyEffect *LumiflyEffect

	// Buffers pré-alloués pour les effets (évite ebiten.NewImage à chaque frame)
	scannerBuffer *ebiten.Image
	lumiflyBuffer *ebiten.Image

	// Dernier tour rendu (pour détecter le changement de tour et clear les silhouettes)
	lastRenderedTurn int
}

// LumiflyEffect représente l'onde lumineuse circulaire du Lumifly
type LumiflyEffect struct {
	Centers      []entity.Position // Positions des lumifly émetteurs (en cases)
	Radius       float64           // Rayon maximal de l'onde (en cases)
	WaveDuration float64           // Durée de l'onde animée (fixe, ~0.5s)
	WaveElapsed  float64           // Temps écoulé pour l'onde
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

// Optimization: string cache to avoid fmt.Sprintf in render loop
var (
	gridPosCache = make(map[string]string) // "gridID:x,y"
)

func getGridPosKey(gridID string, x, y int) string {
	// Simple manual concatenation is faster than fmt.Sprintf for frequent calls
	return gridID + ":" + string(rune(x)) + "," + string(rune(y))
}

// ScannerEffect représente l'état d'un effet de scanner
type ScannerEffect struct {
	GridID    string
	Positions []board.Position
	Progress  float64 // 0.0 à 1.0
	Duration  float64 // Durée totale en secondes
	Elapsed   float64 // Temps écoulé en secondes
}

// QuakeEffect représente l'état d'un effet de séisme (Stonewarden)
type QuakeEffect struct {
	GridID        string
	Progress      float64 // 0.0 à 1.0
	Duration      float64 // Durée totale en secondes
	Elapsed       float64 // Temps écoulé en secondes
	RotationAngle float32 // Angle de rotation de l'ancienne orientation (radians)
	Clockwise     bool    // Sens de la rotation (true = horaire, false = antihoraire)
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
		activeQuakeEffects:   make(map[string]*QuakeEffect),
		hoverStates:          make(map[string]*HoverState),
		bounceStates:         make(map[string]*BounceState),
		trackRenderer:        NewTrackRenderer(ui.TileSize),
		pendingHits:          make(map[string]entity.Position),
	}
	// Initialise le gestionnaire d'animations lié au renderer
	r.AnimManager = NewAnimationManager(r)
	return r
}

// getScannerBuffer retourne le buffer pré-alloué pour l'effet scanner (lazy init).
func (r *BoardRenderer) getScannerBuffer() *ebiten.Image {
	if r.scannerBuffer == nil {
		r.scannerBuffer = ebiten.NewImage(int(ui.PlaymatW), int(ui.PlaymatH))
	}
	r.scannerBuffer.Clear()
	return r.scannerBuffer
}

// getLumiflyBuffer retourne le buffer pré-alloué pour l'effet lumifly (lazy init).
func (r *BoardRenderer) getLumiflyBuffer() *ebiten.Image {
	if r.lumiflyBuffer == nil {
		r.lumiflyBuffer = ebiten.NewImage(int(ui.PlaymatW), int(ui.PlaymatH))
	}
	r.lumiflyBuffer.Clear()
	return r.lumiflyBuffer
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

// TriggerQuakeEffect déclenche manuellement l'effet séisme (debug)
func (r *BoardRenderer) TriggerQuakeEffect(gridID string, clockwise bool, angle float32) {
	if gridID == "" {
		return
	}
	r.activeQuakeEffects[gridID] = &QuakeEffect{
		GridID:        gridID,
		Progress:      0.0,
		Duration:      0.75,
		Elapsed:       0.0,
		RotationAngle: angle,
		Clockwise:     clockwise,
	}
}

// ClearAnimations arrête toutes les animations de flip en cours
func (r *BoardRenderer) ClearAnimations() {
	r.flipAnimations = make(map[string]*FlipAnimation)
}

func (r *BoardRenderer) SetDebugReveal(entityID entity.ID, revealed bool) {
	if r.debugRevealed == nil {
		r.debugRevealed = make(map[entity.ID]bool)
	}
	r.debugRevealed[entityID] = revealed
}

func (r *BoardRenderer) SetDebugRevealAll(entities map[entity.ID]bool) {
	r.debugRevealed = entities
}

func (r *BoardRenderer) ClearDebugReveal() {
	r.debugRevealed = make(map[entity.ID]bool)
}

// StartFlipAnimation démarre une animation de flip pour une tuile
func (r *BoardRenderer) StartFlipAnimation(gridID string, pos board.Position, flipDir entity.FlipDirection, entityID string, finalState entity.TileState, startTrans, endTrans entity.Transformation) {
	key := gridID + ":" + fmt.Sprint(pos.X) + "," + fmt.Sprint(pos.Y) + ":" + entityID
	fmt.Printf("[ANIM-DEBUG] StartFlipAnimation: Key=%s, Dir=%v, State=%s\n", key, flipDir, finalState.String())
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
	// --- Mise à jour des rotations (ECS) ---
	rotationIDs := world.Components.QueryByComponent("rotation_animation")
	for _, id := range rotationIDs {
		if comp, ok := world.Components.Get(id, "rotation_animation"); ok {
			ra := comp.(*component.RotationAnimation)
			ra.CurrentTick++
			if ra.CurrentTick >= ra.DurationTicks {
				world.Components.Remove(id, "rotation_animation")
			}
		}
	}

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
					"grid_id":        anim.GridID,
					"position":       anim.Position,
					"animation_type": "flip", // Précise le type pour filtrage
					"tile_state":     anim.TileState,
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
func (r *BoardRenderer) UpdateEffects(deltaTime float64, world *domain.World) {
	for gridID, effect := range r.activeScannerEffects {
		effect.Elapsed += deltaTime
		effect.Progress = effect.Elapsed / effect.Duration
		if effect.Progress >= 1.0 {
			effect.Progress = 1.0
			delete(r.activeScannerEffects, gridID)
		}
	}

	for gridID, effect := range r.activeQuakeEffects {
		effect.Elapsed += deltaTime
		effect.Progress = effect.Elapsed / effect.Duration
		if effect.Progress >= 1.0 {
			effect.Progress = 1.0
			delete(r.activeQuakeEffects, gridID)
		}
	}

	if r.lumiflyEffect != nil {
		r.lumiflyEffect.WaveElapsed += deltaTime

		// On coupe l'effet immédiatement si le tour a changé
		if world.Turn > r.lastRenderedTurn {
			r.lumiflyEffect = nil
		}
	}
	r.lastRenderedTurn = world.Turn
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

	world.EventBus.SubscribeFunc(event.Type("lumifly_effect_triggered"), func(e event.Event) {
		centers, _ := e.Payload["centers"].([]entity.Position)
		radius, _ := e.Payload["radius"].(float64)
		waveDuration, _ := e.Payload["duration"].(float64)
		r.lumiflyEffect = &LumiflyEffect{
			Centers:      centers,
			Radius:       radius,
			WaveDuration: waveDuration,
			WaveElapsed:  0.0,
		}
		fmt.Printf("[LUMIFLY] Onde dorée déclenchée: %d centre(s), rayon=%.1f, durée=%.1fs\n", len(centers), radius, waveDuration)
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
		modeStr, _ := e.Payload["mode"].(string)

		switch modeStr {
		case "under":
			layer = LayerUnder
		case "over", "earthquake", "swap":
			layer = LayerOver
		case "swap_under":
			layer = LayerNormal
		default:
			layer = LayerNormal
		}

		if r.AnimManager != nil {
			flipDir := entity.FlipRight
			if modeStr == "earthquake" {
				flipDir = r.computeFlipDirection(board.Position{X: from.X, Y: from.Y}, board.Position{X: to.X, Y: to.Y})
			}
			r.AnimManager.StartTileMove(world, world.CurrentGridID, entityID, board.Position{X: from.X, Y: from.Y}, board.Position{X: to.X, Y: to.Y}, 45, layer, modeStr, flipDir)
		}
	})

	// Déclenche l'animation d'attaque (lunge) et l'effet séisme (Stonewarden)
	world.EventBus.SubscribeFunc(event.CreatureAttacked, func(e event.Event) {
		ent, ok := world.Entities.Get(entity.ID(e.SourceID))
		if !ok || ent.GetType() != entity.TypeCreature {
			return
		}

		creature := ent.(*domain.Creature)
		dx, dy := creature.GetLungeDirectionVector()

		var hitTarget *entity.Position
		if pos, ok := e.Payload["hit_target"].(*entity.Position); ok {
			hitTarget = pos
		}

		if r.AnimManager != nil {
			r.AnimManager.StartAttack(world, e.SourceID, dx, dy, hitTarget)
		}

		// Effet de séisme pour le Stonewarden (uniquement si l'attaque touche)
		if creature.Species == "stonewarden" && world.CurrentGridID != "" && hitTarget != nil {
			r.activeQuakeEffects[world.CurrentGridID] = &QuakeEffect{
				GridID:        world.CurrentGridID,
				Progress:      0.0,
				Duration:      0.75,
				Elapsed:       0.0,
				RotationAngle: math.Pi / 2, // 90 degrés
				Clockwise:     true,
			}
		}
	})

	// Enregistre les hits subis par le joueur pour les synchroniser avec l'attaque
	world.EventBus.SubscribeFunc(event.PlayerDamaged, func(e event.Event) {
		if pos, ok := e.Payload["position"].(entity.Position); ok {
			r.pendingHits[e.SourceID] = pos
		}
	})

	// Gère l'animation de propagation organique des ressources
	world.EventBus.SubscribeFunc(event.ResourcePropagated, func(e event.Event) {
		from, hasFrom := e.Payload["from"].(entity.Position)
		to, hasTo := e.Payload["to"].(entity.Position)
		newID, hasNewID := e.Payload["new_entity_id"].(string)

		if !hasFrom || !hasTo || !hasNewID {
			return
		}

		if r.AnimManager != nil {
			r.AnimManager.StartTileMove(
				world,
				world.CurrentGridID,
				newID,
				board.Position{X: from.X, Y: from.Y},
				board.Position{X: to.X, Y: to.Y},
				45,
				LayerNormal,
				"propagate",
				entity.FlipRight,
			)
		}
	})

	// Incrémenter le compteur d'animations actives
	world.EventBus.SubscribeFunc(event.AnimationStarted, func(e event.Event) {
		world.ActiveAnimationCount++
	})

	// Décrémenter le compteur d'animations actives
	world.EventBus.SubscribeFunc(event.AnimationEnded, func(e event.Event) {
		if world.ActiveAnimationCount > 0 {
			world.ActiveAnimationCount--
		}
	})
}

func (r *BoardRenderer) computeFlipDirection(from, to board.Position) entity.FlipDirection {
	dx := to.X - from.X
	dy := to.Y - from.Y

	switch {
	case dx == 0 && dy < 0:
		return entity.FlipTop
	case dx > 0 && dy < 0:
		return entity.FlipTopRight
	case dx > 0 && dy == 0:
		return entity.FlipRight
	case dx > 0 && dy > 0:
		return entity.FlipBottomRight
	case dx == 0 && dy > 0:
		return entity.FlipBottom
	case dx < 0 && dy > 0:
		return entity.FlipBottomLeft
	case dx < 0 && dy == 0:
		return entity.FlipLeft
	case dx < 0 && dy < 0:
		return entity.FlipTopLeft
	default:
		return entity.FlipRight
	}
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
		geom.Rotate(-math.Pi / 2)
		geom.Scale(1, -1)
	case entity.TransMirrorD2: // Diagonale /
		geom.Rotate(math.Pi / 2)
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
	r.UpdateEffects(1.0/60.0, world)

	if world.CurrentGridID != "" {
		grid, _ := world.GetGrid(world.CurrentGridID)

		// On prépare les traces pour cette frame
		if r.trackRenderer != nil {
			r.trackRenderer.PrepareFrame(world)
		}

		// NOTE : On ne synchronise plus r.boardRotation avec grid.MainBearing ici.
		// La rotation est gérée logiquement par world.RotateGrid qui déplace les tuiles
		// et met à jour leurs transformations. boardRotation reste à 0 sauf animation.

		// --- 2. COUCHE : FOND DE LA GRILLE (Les cases vides) ---
		// Toujours rendu sur le plateau (Board area)
		r.renderEmptyGrid(screen, world.CurrentGridID, world, false)

		getCenter := func(pos board.Position) (float64, float64) {
			return r.GetTileCenter(pos, grid)
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

		// Aperçu de l'empreinte de pas au curseur (semi-transparent)
		// if world.Player != nil && world.Player.IsAlive() {
		// 	cursorX, cursorY := ebiten.CursorPosition()
		// 	r.trackRenderer.DrawFootstepPreview(screen, float64(cursorX), float64(cursorY), world, getCenter)
		// }

		// Le scanner glisse au-dessus de tout le monde sur le plateau
		r.renderEffectsOver(screen, world)
	}

	// 6. On dessine l'interface utilisateur tout en haut
	r.renderActionButtons(screen)

	// 7. Capture du playmat À LA FIN (pour le fantôme de la prochaine frame)
	//    Le snapshot contient l'état de cette frame, qui sera le "ancien" lors de la prochaine rotation
	//    Le snapshot est plus grand que le playmat (QuakePadding sur chaque côté) pour éviter
	//    les espaces vides quand le shader tourne le ghost de 90°.
	if world.CurrentGridID != "" && r.effectRenderer != nil {
		snapW, snapH := int(ui.QuakeSnapW), int(ui.QuakeSnapH)
		if r.playmatSnapshot == nil || r.playmatSnapshot.Bounds().Dx() != snapW || r.playmatSnapshot.Bounds().Dy() != snapH {
			r.playmatSnapshot = ebiten.NewImage(snapW, snapH)
		}
		r.playmatSnapshot.Clear()
		subImg := screen.SubImage(image.Rect(int(ui.PlaymatX), int(ui.PlaymatY), int(ui.PlaymatX)+int(ui.PlaymatW), int(ui.PlaymatY)+int(ui.PlaymatH))).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(ui.QuakePadding), float64(ui.QuakePadding))
		r.playmatSnapshot.DrawImage(subImg, op)
	}
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
		r.renderLumiflyEffect(screen, world)

		// Rendu des menaces d'attaque (Intensions d'attaque) au-dessus de tout
		if r.trackRenderer != nil {
			getCenter := func(pos board.Position) (float64, float64) {
				grid, _ := world.GetGrid(world.CurrentGridID)
				return r.GetTileCenter(pos, grid)
			}
			r.trackRenderer.RenderAttackThreats(screen, world, getCenter)

			// Si le bonus du Fleeing Sprite est actif, on affiche toutes les zones de menace
			if world.Player != nil && world.Player.ThreatVisionTurns > 0 {
				r.trackRenderer.RenderPotentialThreats(screen, world, getCenter)
			}
		}
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

	if s.IsAgnosia {
		// Thème monochrome standardisé pour l'Agnosia
		bgColor = color.Black
	} else if s.IsAtaxia {
		// Thème Désert pour l'Ataxia (Whack-a-mole)
		bgColor = color.RGBA{195, 145, 85, 255}
	} else if s.Active {
		bgColor = color.RGBA{60, 60, 80, 255}
	} else {
		bgColor = color.RGBA{40, 40, 40, 255}
	}

	x, y := float32(s.CurrentX), float32(s.CurrentY)
	w, h := float32(s.Width), float32(s.Height)

	vector.FillRect(screen, x, y, w, h, bgColor, true)

	if s.FillProgress > 0 {
		fillW := w * float32(s.FillProgress)
		var fillColor color.Color
		if s.IsAgnosia {
			// Remplissage progressif de NOIR vers BLANC
			// À 100%, il devient BLANC et masque ainsi le texte/icône blanc (blanc sur blanc)
			gray := uint8(s.FillProgress * 255)
			fillColor = color.RGBA{gray, gray, gray, 255}
		} else if s.IsAtaxia {
			// Remplissage orangé/terracotta pour le désert
			fillColor = color.RGBA{215, 130, 60, 200}
		} else if s.FillAlert {
			fillColor = color.RGBA{180, 50, 50, 200}
		} else {
			fillColor = color.RGBA{100, 80, 150, 160}
		}
		vector.DrawFilledRect(screen, x, y, fillW, h, fillColor, true)
	}

	var borderColor color.Color
	if s.IsAgnosia {
		borderColor = color.White
	} else if s.IsAtaxia {
		borderColor = color.RGBA{225, 185, 125, 255}
	} else if s.Active {
		borderColor = color.RGBA{200, 200, 255, 255}
	} else {
		borderColor = color.RGBA{80, 80, 80, 255}
	}
	vector.StrokeRect(screen, x, y, w, h, 1, borderColor, true)

	var labelColor color.Color = color.White
	if s.IsAgnosia {
		labelColor = color.White
	} else if !s.Active && !s.IsAtaxia {
		labelColor = color.RGBA{120, 120, 120, 255}
	}

	// Application du TextScale pour l'effet pulsé (Aphasia)
	tx := x + float32(ui.ButtonTextRelativeX)
	ty := y + float32(ui.ButtonTextRelativeY) + 15

	if s.TextScale != 1.0 {
		// On dessine le texte sur une image temporaire pour pouvoir le scaler
		txtW, txtH := int(math.Ceil(float64(ui.ActionButtonW))), 30
		txtImg := ebiten.NewImage(txtW, txtH)
		text.Draw(txtImg, s.Label, basicfont.Face7x13, 0, 15, color.White)

		op := &ebiten.DrawImageOptions{}
		// Point de pivot au centre du bouton (approximatif pour le texte)
		op.GeoM.Translate(-float64(txtW)/4, -15)
		op.GeoM.Scale(s.TextScale, s.TextScale)
		op.GeoM.Translate(float64(tx)+float64(txtW)/4, float64(ty))
		op.ColorScale.ScaleWithColor(labelColor)
		screen.DrawImage(txtImg, op)
	} else {
		text.Draw(screen, s.Label, basicfont.Face7x13, int(tx), int(ty), labelColor)
	}

	if s.Active || s.IsAgnosia || s.IsAtaxia {
		ix := x + float32(ui.ButtonIconRelativeX)
		iy := y + float32(ui.ButtonIconRelativeY)
		var indicatorColor color.Color
		if s.IsAgnosia {
			indicatorColor = color.White
		} else if s.IsAtaxia {
			indicatorColor = color.RGBA{240, 170, 90, 200}
		} else {
			indicatorColor = color.RGBA{100, 255, 100, 200}
		}
		vector.FillRect(screen, ix, iy, float32(ui.ButtonIconSize), float32(ui.ButtonIconSize), indicatorColor, true)
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

	themeName := "default"
	if grid != nil {
		themeName = string(grid.Biome)
	}
	theme := r.assets.GetTheme(themeName)

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
				tileImg = r.assets.GetImage("tile_blocked_" + themeName)
			} else {
				tileImg = r.assets.GetImage("tile_sealed_" + themeName)
			}
		} else if tileState&entity.Revealed != 0 {
			tileImg = r.assets.GetImage("tile_exit_" + themeName)
		} else {
			tileImg = r.assets.GetImage("tile_hidden_" + themeName)
		}

		entityID := fmt.Sprintf("exit_%s_%d", board.DirectionToName(dir), i)

		// --- GESTION DES ANIMATIONS (FLIP) ---
		var animation *FlipAnimation
		for _, anim := range r.flipAnimations {
			if anim.EntityID == entityID && anim.GridID == world.CurrentGridID {
				animation = anim
				break
			}
		}

		if animation != nil && animation.IsActive() {
			r.renderFlippingTile(screen, tx, ty, animation, nil, themeName, theme.HiddenBorder)
			continue
		}

		// --- RENDU AVEC TILT (IDLE GEOMETRY) ---
		margin := (r.tileSize - ui.FaceSize) / 2
		gtx, gty := float32(tx+margin), float32(ty+margin)

		geo := r.generateIdleGeometry(gtx, gty, entityID, theme.HiddenBorder)
		// Pas de rotation globale du plateau pour les éléments du Playmat (sauf si souhaité)
		// r.ApplyBoardRotation(geo.V, cx, cy)

		// Réglage des UV pour l'icône de sortie (flèche)
		// La texture tile_exit est déjà une flèche pointant vers le Nord?
		// Non, les assets générés sont spécifiques.
		// Mais ici on utilise DrawTriangles, donc on doit gérer la rotation de la flèche via UV ou sommets.

		// Si c'est la flèche de sortie, on applique la rotation/miroir aux UV
		if tileState&entity.Revealed != 0 && tileImg == r.assets.GetImage("tile_exit_"+themeName) {
			var finalTrans entity.Transformation

			switch dir {
			case entity.DirNorth:
				// Inversé : i=0 est le miroir (gauche), i=1 est l'identité (droite)
				if i == 0 {
					finalTrans = entity.TransMirrorH
				} else {
					finalTrans = entity.TransIdentity
				}
			case entity.DirEast:
				// Inversé + Rotation 180° sur le bas (i=1)
				if i == 0 {
					finalTrans = entity.Compose(entity.TransRot270, entity.TransMirrorH)
				} else {
					finalTrans = entity.Compose(entity.TransRot270, entity.TransRot180)
				}
			case entity.DirSouth:
				// Déjà correct : i=0 est l'original, i=1 est le miroir
				if i == 0 {
					finalTrans = entity.TransRot180
				} else {
					finalTrans = entity.Compose(entity.TransRot180, entity.TransMirrorH)
				}
			case entity.DirWest:
				// Inversé + Rotation 180° sur le bas (i=1)
				if i == 0 {
					finalTrans = entity.Compose(entity.TransRot90, entity.TransRot180)
				} else {
					finalTrans = entity.Compose(entity.TransRot90, entity.TransMirrorH)
				}
			}

			uvCoords := GetTransformationGeometry(finalTrans)
			for j := 0; j < 4; j++ {
				geo.V[j].SrcX = uvCoords[j][0] * ui.FaceSize
				geo.V[j].SrcY = uvCoords[j][1] * ui.FaceSize
			}
		}

		// Dessin
		r.drawGeometryPart(screen, geo.V, geo.I[6:12], r.assets.GetImage("tile_hidden_"+themeName)) // Dos
		r.drawGeometryPart(screen, geo.V, geo.I[:6], tileImg)                                       // Face

		// Slices
		id := entityID
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
	}
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

// GetTileCenter retourne le centre d'une case en coordonnées écran.
// Note: On ne prend plus en compte boardRotation ici car la rotation est gérée logiquement par le swap d'index.
func (r *BoardRenderer) GetTileCenter(pos board.Position, grid *board.Grid) (float64, float64) {
	isPortalZone := grid.ID == "zone_start" || grid.ID == "zone_end"
	x, y := r.calculateTileScreenPos(pos, grid, isPortalZone)
	cx, cy := x+r.tileSize/2, y+r.tileSize/2

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
				// EFFET PEEK : Si la tuile du dessus est en train de se retourner (FlipAnimation active),
				// et qu'il y a une tuile en dessous, on dessine d'abord la tuile du dessous en forçant son reveal visuel.
				var topIsFlipping bool
				for _, anim := range r.flipAnimations {
					if anim.EntityID == topID && (gridID == "" || anim.GridID == gridID) && anim.IsActive() {
						topIsFlipping = true
						break
					}
				}

				if topIsFlipping && len(plot.EntitiesID) > 1 {
					underID := plot.EntitiesID[len(plot.EntitiesID)-2]
					// On passe forceReveal à true pour que le joueur puisse entrevoir la tuile inférieure pendant l'animation
					r.renderSingleTileIDAt(screen, sx, sy, gridID, underID, world, true, 1.0)
				}

				// Rendu normal de la tuile au sommet (qui dessinera le flip 3D par-dessus)
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

	// Gestion du décalage d'attaque
	if comp, ok := world.Components.Get(entityID, "attacking_animation"); ok {
		aa := comp.(*component.AttackingAnimation)
		x += aa.OffsetX
		y += aa.OffsetY
	}

	// Gestion de l'angle de rotation (Bounce/NavOrientation)
	var rotationAngle float64
	if comp, ok := world.Components.Get(entityID, "rotation_animation"); ok {
		ra := comp.(*component.RotationAnimation)
		t := float64(ra.CurrentTick) / float64(ra.DurationTicks)
		if t > 1 {
			t = 1
		}
		// Interpolation simple de l'angle de rotation relatif (0 -> TargetAngle)
		rotationAngle = (ra.TargetAngle - ra.CurrentAngle) * t
	}

	visualState := ent.GetState()
	if forceReveal || r.debugRevealed[ent.GetID()] {
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
		tileImg = r.assets.GetTileImage("matched", themeName)
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

	// --- ROTATION DYNAMIQUE (BOUNCE) ---
	if rotationAngle != 0 {
		angleRad := rotationAngle * math.Pi / 180
		cosA, sinA := float32(math.Cos(angleRad)), float32(math.Sin(angleRad))
		for i := range geo.V {
			relX := geo.V[i].DstX - cx
			relY := geo.V[i].DstY - cy
			geo.V[i].DstX = cx + relX*cosA - relY*sinA
			geo.V[i].DstY = cy + relX*sinA + relY*cosA
		}
	}

	// --- CUMUL : Mise à l'échelle et couleur (Uniquement si RÉVÉLÉ) ---
	if (visualState&entity.Revealed != 0 || visualState&entity.Matched != 0) && ent.GetCumulationLevel() > 0 {
		cumulScale := 1.0 + 0.15*float64(ent.GetCumulationLevel())
		for i := range geo.V {
			geo.V[i].DstX = cx + (geo.V[i].DstX-cx)*float32(cumulScale)
			geo.V[i].DstY = cy + (geo.V[i].DstY-cy)*float32(cumulScale)

			// Teinte légèrement dorée/brillante
			geo.V[i].ColorR *= 1.2
			geo.V[i].ColorG *= 1.1
		}
	}

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

	// 1. Transformation D4 du Dos (UVs miroirs horizontaux)
	if ent != nil {
		backW, backH := backImg.Size()
		bw, bh := float32(backW), float32(backH)
		backUv := GetTransformationGeometry(ent.GetTransformation())
		for i := 0; i < 4; i++ {
			geo.V[4+i].SrcX = (1.0 - backUv[i][0]) * bw
			geo.V[4+i].SrcY = backUv[i][1] * bh
		}
	}

	// 2. Dessin du Dos
	r.drawGeometryPart(screen, geo.V, geo.I[6:12], backImg)

	// 1b. Silhouette sur le Dos (alpha très faible, pour effets shader/révélation)
	r.renderSilhouetteOnBack(screen, geo, ent)

	// 3. Dessin de la Face (UVs D4 pour toutes les entités)
	w, h := faceImg.Size()
	fw, fh := float32(w), float32(h)
	uvCoords := GetTransformationGeometry(ent.GetTransformation())
	for i := 0; i < 4; i++ {
		geo.V[i].SrcX = uvCoords[i][0] * fw
		geo.V[i].SrcY = uvCoords[i][1] * fh
	}

	r.drawGeometryPart(screen, geo.V, geo.I[:6], faceImg)

	// 2b. Bordures fines pour tuiles cumulées
	if (visualState&entity.Revealed != 0 || visualState&entity.Matched != 0) && ent.GetCumulationLevel() > 0 {
		level := ent.GetCumulationLevel()
		var borderColor color.Color
		switch {
		case level >= 4:
			borderColor = color.RGBA{200, 50, 50, 255} // Rouge profond
		case level == 3:
			borderColor = color.RGBA{200, 120, 20, 255} // Orange ambré
		case level == 2:
			borderColor = color.RGBA{180, 160, 30, 255} // Jaune doré
		default:
			borderColor = color.RGBA{120, 200, 80, 255} // Vert doux
		}
		// Dessiner `level` bordures concentriques fines
		for b := 0; b < level; b++ {
			inset := float32(2 + b*3)
			strokeW := float32(1)
			vector.StrokeRect(screen,
				float32(x)+inset, float32(y)+inset,
				float32(r.tileSize)-inset*2, float32(r.tileSize)-inset*2,
				strokeW, borderColor, true)
		}
	}

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
	if shouldShowContent {
		// On applique la transformation de l'entité (qui inclut déjà la rotation de la grille via RotateGrid)
		r.renderFlippingEntityTriangles(screen, geo.V[:4], ent, ent.GetTransformation())
	}
}

func (r *BoardRenderer) getEntityRevealedImage(ent entity.Entity, themeName string) *ebiten.Image {
	if ent == nil {
		return r.assets.GetTileImage("revealed", themeName)
	}

	switch ent.GetType() {
	case entity.TypeTrap:
		return r.assets.GetImage("tile_trap_" + themeName)
	case entity.TypeStructure:
		if ent.HasTag("start_portal") || ent.HasTag("finish_portal") || ent.HasTag("portable_portal") || ent.HasTag("portal") {
			return r.assets.GetTileImage("portal", themeName)
		} else if ent.HasTag("dolmen") {
			return r.assets.GetTileImage("dolmen", themeName)
		} else if ent.HasTag("obelisk") {
			return r.assets.GetTileImage("obelisk", themeName)
		}
		return r.assets.GetTileImage("structure", themeName)
	default:
		return r.assets.GetTileImage("revealed", themeName)
	}
}

func (r *BoardRenderer) ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool) {
	// 1. Vérification de l'inventaire
	if float64(screenX) >= ui.InventoryX && float64(screenX) <= ui.InventoryX+ui.InventoryW &&
		float64(screenY) >= ui.InventoryY && float64(screenY) <= ui.InventoryY+ui.InventoryH {

		// Zone des slots (commence à InventoryY + 40, s'arrête avant les boutons à InventoryY + 321)
		slotZoneY := ui.InventoryY + 40
		if float64(screenY) >= float64(slotZoneY) && float64(screenY) <= float64(ui.InventoryY+321) {
			localY := float64(screenY) - float64(slotZoneY) + float64(world.Player.Inventory.ScrollOffset)
			localX := float64(screenX) - ui.InventoryX - 5

			rowH := ui.LootSlotSize + ui.LootSlotPadding
			row := int(localY / rowH)
			col := int(localX / rowH)

			if col >= 0 && col < ui.LootSlotsPerRow {
				return board.Position{X: col, Y: row}, board.InventoryGridID, true
			}
		}

		// Dans la zone inventaire mais pas dans les slots -> retourner grille inventaire avec position invalide
		return board.Position{X: -1, Y: -1}, board.InventoryGridID, true
	}

	// 2. Vérification des sorties (Navigation)
	px := float64(screenX) - ui.PlaymatX
	py := float64(screenY) - ui.PlaymatY

	if px >= ui.ExitNorthX && px < ui.ExitNorthX+ui.ExitNorthW && py >= ui.ExitNorthY && py < ui.ExitNorthY+ui.ExitNorthH {
		index := 0
		if px >= ui.ExitNorthX+ui.TileSize {
			index = 1
		}
		return board.Position{X: index, Y: 0}, "exit_north", true
	}
	if px >= ui.ExitEastX && px < ui.ExitEastX+ui.ExitEastW && py >= ui.ExitEastY && py < ui.ExitEastY+ui.ExitEastH {
		index := 0
		if py >= ui.ExitEastY+ui.TileSize {
			index = 1
		}
		return board.Position{X: index, Y: 0}, "exit_east", true
	}
	if px >= ui.ExitSouthX && px < ui.ExitSouthX+ui.ExitSouthW && py >= ui.ExitSouthY && py < ui.ExitSouthY+ui.ExitSouthH {
		index := 0
		if px >= ui.ExitSouthX+ui.TileSize {
			index = 1
		}
		return board.Position{X: index, Y: 0}, "exit_south", true
	}
	if px >= ui.ExitWestX && px < ui.ExitWestX+ui.ExitWestW && py >= ui.ExitWestY && py < ui.ExitWestY+ui.ExitWestH {
		index := 0
		if py >= ui.ExitWestY+ui.TileSize {
			index = 1
		}
		return board.Position{X: index, Y: 0}, "exit_west", true
	}

	// 3. Vérification de la grille de jeu principale
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

	// Cas de l'inventaire
	if gID == board.InventoryGridID {
		slotZoneY := float64(ui.InventoryY + 40)
		rowH := float64(ui.LootSlotSize + ui.LootSlotPadding)
		tileScreenX := ui.InventoryX + 5 + float64(pos.X)*rowH
		tileScreenY := slotZoneY + float64(pos.Y)*rowH - float64(world.Player.Inventory.ScrollOffset)
		return int(float64(screenX) - tileScreenX), int(float64(screenY) - tileScreenY), gID, true
	}

	// Cas des sorties
	if strings.HasPrefix(gID, "exit_") {
		px := float64(screenX) - ui.PlaymatX
		py := float64(screenY) - ui.PlaymatY
		var ex, ey float64
		switch gID {
		case "exit_north":
			ex, ey = ui.ExitNorthX, ui.ExitNorthY
		case "exit_east":
			ex, ey = ui.ExitEastX, ui.ExitEastY
		case "exit_south":
			ex, ey = ui.ExitSouthX, ui.ExitSouthY
		case "exit_west":
			ex, ey = ui.ExitWestX, ui.ExitWestY
		}
		if gID == "exit_north" || gID == "exit_south" {
			ex += float64(pos.X) * r.tileSize
		} else {
			ey += float64(pos.X) * r.tileSize
		}
		return int(px - ex), int(py - ey), gID, true
	}

	// Cas standard
	grid, _ := world.GetGrid(gID)
	isPortalZone := world.DreamPlane != nil && (gID == world.DreamPlane.StartZoneID || gID == world.DreamPlane.EndZoneID)

	tileScreenX, tileScreenY := r.calculateTileScreenPos(pos, grid, isPortalZone)
	localXVal, localYVal := float64(screenX)-tileScreenX, float64(screenY)-tileScreenY

	// NOTE : On ne pivote plus les coordonnées locales ici pour le survol (Hover).
	// On veut que le survol soit purement VISUEL (survoler le haut de la tuile sur l'écran lève le haut).

	return int(localXVal), int(localYVal), gID, true
}

func (r *BoardRenderer) RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, highlightColor color.Color, world *domain.World) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	var x, y float64
	if gridID == board.InventoryGridID {
		// Gestion spécifique pour la grille d'inventaire
		slotZoneY := float64(ui.InventoryY + 40)
		rowH := ui.LootSlotSize + ui.LootSlotPadding
		x = ui.InventoryX + 5 + float64(pos.X)*rowH
		y = slotZoneY + float64(pos.Y)*rowH - world.Player.Inventory.ScrollOffset

		// Ne dessine pas si en dehors de la zone visible de l'inventaire
		if y+ui.LootSlotSize <= slotZoneY || y >= slotZoneY+331 {
			return
		}
	} else {
		isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)
		x, y = r.calculateTileScreenPos(pos, grid, isPortalZone)
	}

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

	hasEntities := r.hasEntitiesIn3x3Area(grid, center, world)

	topLeft := board.Position{X: center.X - 1, Y: center.Y - 1}
	x, y := r.calculateTileScreenPos(topLeft, grid, isPortalZone)
	width := 3*r.tileSize + 2*spacingX
	height := 3*r.tileSize + 2*spacingY

	previewColor := color.RGBA{80, 180, 100, 140}
	fillColor := color.RGBA{80, 180, 100, 20}
	if hasEntities {
		previewColor = color.RGBA{180, 150, 30, 140}
		fillColor = color.RGBA{180, 150, 30, 20}
	}
	vector.FillRect(screen, float32(x), float32(y), float32(width), float32(height), fillColor, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 3, previewColor, true)
}

func (r *BoardRenderer) hasEntitiesIn3x3Area(grid *board.Grid, center board.Position, world *domain.World) bool {
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			pos := board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy}
			plot, err := grid.Get(pos)
			if err != nil {
				continue
			}
			if len(plot.EntitiesID) > 0 {
				return true
			}
		}
	}
	return false
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

		if r.AnimManager != nil {
			if transAnim, ok := r.AnimManager.animes[id]; ok {
				if transAnim.Mode == "earthquake" {
					progress := float64(transAnim.Tick) / math.Max(1, float64(transAnim.Duration))
					grid, _ := world.GetGrid(world.CurrentGridID)
					themeName := "default"
					if grid != nil {
						themeName = string(grid.Biome)
					}
					theme := r.assets.GetTheme(themeName)
					r.renderEarthquakeTile360(screen, curX, curY, progress, ent, themeName, theme.HiddenBorder, transAnim.FlipDirection)
					continue
				}
				if transAnim.Mode == "propagate" {
					progress := float64(transAnim.Tick) / math.Max(1, float64(transAnim.Duration))
					grid, _ := world.GetGrid(world.CurrentGridID)
					themeName := "default"
					if grid != nil {
						themeName = string(grid.Biome)
					}

					// 1. Calcul des centres écrans d'origine (parent) et actuel (enfant)
					fromPos := board.Position{X: transAnim.FromGridX, Y: transAnim.FromGridY}
					pX, pY := r.GetTileCenter(fromPos, grid)
					cX := curX + r.tileSize/2
					cY := curY + r.tileSize/2

					// 2. Dessin du filament élastique entre les deux centres
					r.drawElasticFilament(screen, pX, pY, cX, cY, progress, themeName)

					// 3. Dessin de la tuile enfant déformée (effet de traîne visqueuse)
					r.renderPropagatingChildTile(screen, curX, curY, pX, pY, progress, world, id, themeName)
					continue
				}
			}
		}

		// Création d'un plot fictif pour renderTileAt
		fakePlot := &board.Plot{
			Position:   board.Position{X: anim.TargetGridX, Y: anim.TargetGridY},
			EntitiesID: []string{id},
		}

		// On force l'alpha à 1.0 par défaut pour les translations normales
		r.renderTileAt(screen, curX, curY, world.CurrentGridID, fakePlot, world, false, 1.0)
	}
}

func (r *BoardRenderer) renderScannerEffects(screen *ebiten.Image, gridID string, world *domain.World) {
	effect, ok := r.activeScannerEffects[gridID]
	if !ok || r.effectRenderer == nil {
		return
	}

	// 1. Image source pour le shader (buffer pré-alloué)
	srcImg := r.getScannerBuffer()
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
	fullWidth := float32(ui.PlaymatW)
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

func (r *BoardRenderer) renderLumiflyEffect(screen *ebiten.Image, world *domain.World) {
	if r.lumiflyEffect == nil || world.TurnTimer == nil {
		return
	}

	effect := r.lumiflyEffect
	gridID := world.CurrentGridID
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	// L'effet doré persiste tant que le tour n'est pas terminé (timer du monde)
	if !world.TurnTimer.IsExpired() && r.effectRenderer != nil {
		srcImg := r.getLumiflyBuffer()
		srcImg.Fill(color.RGBA{0, 0, 0, 0})

		isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)

		for _, tile := range grid.Plots {
			if len(tile.EntitiesID) == 0 {
				continue
			}
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			ent, ok := world.Entities.Get(entity.ID(topID))
			if !ok || ent.GetState()&entity.Revealed != 0 {
				continue
			}

			absX, absY := r.calculateTileScreenPos(tile.Position, grid, isPortalZone)
			sx := absX - ui.PlaymatX
			sy := absY - ui.PlaymatY

			r.renderEntityIconOnly(srcImg, sx, sy, ent)
		}

		step := r.tileSize + float64(r.gridSpacing)
		radius := float32(effect.Radius) * float32(step)

		// Progrès de l'onde initiale (lueur blanche)
		progress := float32(effect.WaveElapsed / effect.WaveDuration)

		glowColor := color.RGBA{255, 220, 100, 255}

		for _, center := range effect.Centers {
			absX, absY := r.calculateTileScreenPos(center, grid, isPortalZone)
			centerX := float32(absX + r.tileSize/2)
			centerY := float32(absY + r.tileSize/2)

			r.effectRenderer.DrawLumiflyEffect(screen, srcImg, int(ui.PlaymatX), int(ui.PlaymatY), centerX, centerY, radius, progress, float32(effect.WaveDuration), glowColor)
		}
	}
}

// renderEntityIconOnly dessine uniquement l'icône de l'entité (sans fond/bordure de tuile) avec rotation
func (r *BoardRenderer) renderEntityIconOnly(screen *ebiten.Image, x, y float64, ent entity.Entity) {
	iconSize := ui.FaceSize * 0.75

	var icon *ebiten.Image
	switch e := ent.(type) {
	case *domain.Creature:
		icon = r.assets.GetCreatureSilhouette(e.Species)
	case *domain.Resource:
		stageName := e.Lifecycle.GetCurrentStageName()
		icon = r.assets.GetResourceSilhouette(e.ResourceType, stageName)
	case *player.LootItem:
		if e.OriginalType == entity.TypeCreature {
			icon = r.assets.GetCreatureSilhouette(e.SourceID)
		} else if e.OriginalType == entity.TypeResource {
			icon = r.assets.GetResourceSilhouette(e.SourceID, "")
		}
	}

	if icon == nil {
		return
	}

	// Centrer l'icône dans la tuile avec prise en compte de la rotation
	op := &ebiten.DrawImageOptions{}
	w, h := icon.Size()

	// 1. Centrer l'origine
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)

	// 2. Appliquer la transformation de l'entité
	r.ApplyTransformation(&op.GeoM, ent.GetTransformation())

	// 3. Scale
	op.GeoM.Scale(iconSize/float64(w), iconSize/float64(h))

	// 4. Positionner (x,y sont déjà relatifs au coin haut-gauche de la tuile dans srcImg)
	op.GeoM.Translate(x+r.tileSize/2, y+r.tileSize/2)

	screen.DrawImage(icon, op)
}

// renderSilhouetteOnBack dessine la silhouette de l'entité sur le dos de la tuile
// avec un alpha très faible. Le shader peut ensuite amplifier cette silhouette.
func (r *BoardRenderer) renderSilhouetteOnBack(screen *ebiten.Image, geo thickGeometry, ent entity.Entity) {
	if ent == nil {
		return
	}

	var silhouette *ebiten.Image
	switch e := ent.(type) {
	case *domain.Creature:
		silhouette = r.assets.GetCreatureSilhouette(e.Species)
	case *domain.Resource:
		stageName := e.Lifecycle.GetCurrentStageName()
		silhouette = r.assets.GetResourceSilhouette(e.ResourceType, stageName)
	case *player.LootItem:
		if e.OriginalType == entity.TypeCreature {
			silhouette = r.assets.GetCreatureSilhouette(e.SourceID)
		} else if e.OriginalType == entity.TypeResource {
			silhouette = r.assets.GetResourceSilhouette(e.SourceID, "")
		}
	}

	if silhouette == nil {
		return
	}

	// Copier les sommets du dos (V[4:8]) pour la silhouette
	vSil := make([]ebiten.Vertex, 4)
	for i := 0; i < 4; i++ {
		vSil[i] = geo.V[4+i]
	}

	// Centrer et réduire à 75%
	cx := (vSil[0].DstX + vSil[1].DstX + vSil[2].DstX + vSil[3].DstX) / 4
	cy := (vSil[0].DstY + vSil[1].DstY + vSil[2].DstY + vSil[3].DstY) / 4
	const silScale = 0.75
	for i := 0; i < 4; i++ {
		vSil[i].DstX = cx + (vSil[i].DstX-cx)*silScale
		vSil[i].DstY = cy + (vSil[i].DstY-cy)*silScale
	}

	// UV coords : transformation de l'entité + miroir horizontal
	sw, sh := silhouette.Size()
	sfw, sfh := float32(sw), float32(sh)
	silUvCoords := GetTransformationGeometry(ent.GetTransformation())
	for i := 0; i < 4; i++ {
		// Miroir horizontal : inverse l'axe U (1.0 - u)
		vSil[i].SrcX = (1.0 - silUvCoords[i][0]) * sfw
		vSil[i].SrcY = silUvCoords[i][1] * sfh
	}

	// Alpha très faible (10%) — silhouette basique
	for i := range vSil {
		vSil[i].ColorA *= 0.1
	}

	indices := []uint16{0, 1, 2, 0, 2, 3}
	r.drawGeometryPart(screen, vSil, indices, silhouette)
}

func (r *BoardRenderer) renderQuakeEffects(screen *ebiten.Image, gridID string, world *domain.World) {
	effect, ok := r.activeQuakeEffects[gridID]
	if !ok || r.effectRenderer == nil {
		return
	}

	snapW, snapH := int(ui.QuakeSnapW), int(ui.QuakeSnapH)

	// Image source : snapshot de la frame précédente (ancienne orientation) — 990×990
	src := r.playmatSnapshot
	if src == nil {
		src = ebiten.NewImage(snapW, snapH)
	}

	// Frame buffer 990×990 pour le shader
	if r.quakeFrameBuffer == nil || r.quakeFrameBuffer.Bounds().Dx() != snapW || r.quakeFrameBuffer.Bounds().Dy() != snapH {
		r.quakeFrameBuffer = ebiten.NewImage(snapW, snapH)
	}
	r.quakeFrameBuffer.Clear()

	// Sens de rotation : horaire = angle positif, antihoraire = angle négatif
	angle := effect.RotationAngle
	if !effect.Clockwise {
		angle = -angle
	}

	centerX := float32(0.5)
	centerY := float32(0.5)
	resolution := []float32{float32(snapW), float32(snapH)}
	ghostSize := []float32{float32(snapW), float32(snapH)}

	// Le shader tourne le ghost 990×990 dans le frame buffer 990×990
	r.effectRenderer.DrawQuakeEffect(
		r.quakeFrameBuffer, src, src,
		0, 0,
		float32(effect.Progress),
		angle,
		1.0,
		[]float32{centerX, centerY},
		resolution,
		ghostSize,
	)

	// Cropper le centre 700×700 du frame buffer (spritesheet style)
	pad := ui.QuakePadding
	centerCrop := r.quakeFrameBuffer.SubImage(image.Rect(pad, pad, pad+int(ui.PlaymatW), pad+int(ui.PlaymatH))).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(ui.PlaymatX), float64(ui.PlaymatY))
	screen.DrawImage(centerCrop, op)
}

// RenderQuakeOverlay ré-affiche l'effet quake au-dessus de tout (après les shaders globaux).
func (r *BoardRenderer) RenderQuakeOverlay(screen *ebiten.Image, world *domain.World) {
	if world.CurrentGridID == "" {
		return
	}
	r.renderQuakeEffects(screen, world.CurrentGridID, world)
}

func (r *BoardRenderer) RenderInventoryLoot(target *ebiten.Image, world *domain.World, selectedIdx int, selection map[int]bool, confirmAll bool) {
	inv := &world.Player.Inventory
	grid, ok := world.GetGrid(board.InventoryGridID)
	if !ok {
		return
	}

	rowH := ui.LootSlotSize + ui.LootSlotPadding
	theme := r.assets.GetTheme("default")

	for i := 0; i < inv.MaxSize; i++ {
		row := i / ui.LootSlotsPerRow
		col := i % ui.LootSlotsPerRow

		sx := float64(col)*rowH + 5
		sy := float64(row)*rowH - inv.ScrollOffset

		// On saute ce qui est hors du viewport du buffer (331px de haut)
		if sy+ui.LootSlotSize < 0 || sy > 331 {
			continue
		}

		pos := board.Position{X: col, Y: row}
		plot, err := grid.Get(pos)
		if err != nil || len(plot.EntitiesID) == 0 {
			// Slot vide : on dessine juste le fond
			r.renderEmptySquareAt(target, sx, sy)
			continue
		}

		entityID := plot.EntitiesID[len(plot.EntitiesID)-1]
		ent, ok := world.Entities.Get(entity.ID(entityID))
		if !ok {
			r.renderEmptySquareAt(target, sx, sy)
			continue
		}

		// Rendu du slot avec TILT
		margin := (r.tileSize - ui.FaceSize) / 2
		gtx, gty := float32(sx+margin), float32(sy+margin)

		// Note: On utilise entityID pour récupérer le hover state
		geo := r.generateIdleGeometry(gtx, gty, entityID, theme.HiddenBorder)

		// Détermine l'image de la face selon le type d'objet
		faceImg := r.getEntityRevealedImage(ent, "default")

		// --- GESTION DES ANIMATIONS (FLIP) ---
		var animation *FlipAnimation
		for _, anim := range r.flipAnimations {
			if anim.EntityID == entityID && anim.GridID == board.InventoryGridID {
				animation = anim
				break
			}
		}

		if animation != nil && animation.IsActive() {
			r.renderFlippingTile(target, sx, sy, animation, ent, "default", theme.HiddenBorder)
			continue
		}

		// --- LOGIQUE D'INVENTAIRE CACHÉ (Amnésie / Difficulté Insane) ---
		isHidden := world.Difficulty.ForceHiddenInventory || world.Player.AmnesiaTurns > 0

		// On dessine le dos (toujours pour l'épaisseur/ombre si besoin)
		r.drawGeometryPart(target, geo.V, geo.I[6:12], r.assets.GetImage("tile_hidden_default"))

		if !isHidden {
			// Révélé : Face + Icône
			r.drawGeometryPart(target, geo.V, geo.I[:6], faceImg)

			// Bordures fines pour tuiles cumulées dans l'inventaire
			if ent.GetCumulationLevel() > 0 {
				level := ent.GetCumulationLevel()
				var borderColor color.Color
				switch {
				case level >= 4:
					borderColor = color.RGBA{200, 50, 50, 255}
				case level == 3:
					borderColor = color.RGBA{200, 120, 20, 255}
				case level == 2:
					borderColor = color.RGBA{180, 160, 30, 255}
				default:
					borderColor = color.RGBA{120, 200, 80, 255}
				}
				for b := 0; b < level; b++ {
					inset := float32(2 + b*3)
					vector.StrokeRect(target,
						float32(sx)+inset, float32(sy)+inset,
						float32(r.tileSize)-inset*2, float32(r.tileSize)-inset*2,
						1, borderColor, true)
				}
			}

			r.renderFlippingEntityTriangles(target, geo.V[:4], ent, entity.TransIdentity)
		}

		// Highlights persistants (Usage / Suppression)
		// On les dessine inclinés par dessus la face
		highlight := selection[i]
		if confirmAll && i < len(inv.Items) && inv.Items[i].IsDeletable {
			highlight = true
		}

		if selectedIdx == i {
			// Bordure bleue de sélection active
			indices := []uint16{0, 1, 2, 0, 2, 3}
			r.drawTiltedFrame(target, geo.V[:4], indices, color.RGBA{0, 180, 255, 200})
		} else if highlight {
			// Bordure rouge de suppression
			indices := []uint16{0, 1, 2, 0, 2, 3}
			r.drawTiltedFrame(target, geo.V[:4], indices, color.RGBA{255, 100, 100, 200})
		}

		// Slices pour le tilt
		hover, hasHover := r.hoverStates[entityID]
		if hasHover && hover.Progress > 0 {
			// Bordure cyan de survol (Tilted)
			indices := []uint16{0, 1, 2, 0, 2, 3}
			r.drawTiltedFrame(target, geo.V[:4], indices, color.RGBA{0, 255, 255, 100})
		}

		bounce, hasBounce := r.bounceStates[entityID]
		if (hasHover && hover.Progress > 0) || (hasBounce && bounce.ImpactT < 1.0) {
			hDir := entity.FlipTop
			if hasHover {
				hDir = hover.Dir
			} else if hasBounce {
				hDir = bounce.Dir
			}
			r.drawSlices(target, geo, hDir, r.assets.GetImage("white"))
		}
	}
}

// drawTiltedFrame dessine un cadre rectangulaire incliné en utilisant les sommets fournis
func (r *BoardRenderer) drawTiltedFrame(target *ebiten.Image, v []ebiten.Vertex, indices []uint16, clr color.Color) {
	cr, cg, cb, ca := clr.RGBA()
	fR, fG, fB, fA := float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff

	// Pour l'instant, on dessine une face semi-transparente pour marquer la sélection
	vFrame := make([]ebiten.Vertex, 4)
	copy(vFrame, v)
	for i := range vFrame {
		vFrame[i].ColorR, vFrame[i].ColorG, vFrame[i].ColorB, vFrame[i].ColorA = fR, fG, fB, fA*0.3
		vFrame[i].SrcX, vFrame[i].SrcY = 0, 0
	}
	r.drawGeometryPart(target, vFrame, indices, r.assets.GetImage("white"))

	// Et on ajoute les 4 lignes de bordure
	for i := 0; i < 4; i++ {
		p1 := v[i]
		p2 := v[(i+1)%4]
		vector.StrokeLine(target, p1.DstX, p1.DstY, p2.DstX, p2.DstY, 2, clr, true)
	}
}
