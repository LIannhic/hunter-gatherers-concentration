package renderer

import (
	"math"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// Layer identifie la strate visuelle d'une animation
type Layer string

const (
	LayerUnder  Layer = "under"
	LayerNormal Layer = "normal"
	LayerOver   Layer = "over"
)

// TranslationAnim représente une translation d'une tuile (en pixels)
type TranslationAnim struct {
	EntityID     string
	GridID       string
	FromX, FromY float64
	ToX, ToY     float64
	Tick         int
	Duration     int
	TargetGridX  int
	TargetGridY  int
	Layer        Layer
}

// AnimationManager gère plusieurs translations en parallèle
type AnimationManager struct {
	renderer *BoardRenderer
	animes   map[string]*TranslationAnim // key = entityID
}

func NewAnimationManager(r *BoardRenderer) *AnimationManager {
	return &AnimationManager{
		renderer: r,
		animes:   make(map[string]*TranslationAnim),
	}
}

// StartTileMove démarre une animation de translation basée sur des positions de grille
func (m *AnimationManager) StartTileMove(world *domain.World, gridID string, entityID string, fromPos, toPos board.Position, durationTicks int, layer Layer) {
	grid, _ := world.GetGrid(gridID)
	isPortal := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)
	fx, fy := m.renderer.calculateTileScreenPos(fromPos, grid, isPortal)
	tx, ty := m.renderer.calculateTileScreenPos(toPos, grid, isPortal)

	// Crée/écrase le composant MovingAnimation attaché à l'entité
	ma := &component.MovingAnimation{
		StartX:        fx,
		StartY:        fy,
		CurrentX:      fx,
		CurrentY:      fy,
		TargetGridX:   toPos.X,
		TargetGridY:   toPos.Y,
		CurrentTick:   0,
		DurationTicks: durationTicks,
	}
	world.Components.Add(entityID, ma)

	m.animes[entityID] = &TranslationAnim{
		EntityID:    entityID,
		GridID:      gridID,
		FromX:       fx,
		FromY:       fy,
		ToX:         tx,
		ToY:         ty,
		Tick:        0,
		Duration:    durationTicks,
		TargetGridX: toPos.X,
		TargetGridY: toPos.Y,
		Layer:       layer,
	}
	// Publie un événement pour informer le reste du système (ex: bloquer l'input)
	world.EventBus.PublishImmediate(event.Event{
		Type:     event.AnimationStarted,
		SourceID: entityID,
		Payload: map[string]interface{}{
			"grid_id": gridID,
		},
		Timestamp: time.Now(),
	})
}

func smoothstep(t float64) float64 {
	// Smoothstep cubic (0..1)
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// Update avance toutes les animations et met à jour les composants dans le World
func (m *AnimationManager) Update(world *domain.World) {
	for id, a := range m.animes {
		a.Tick++
		t := float64(a.Tick) / math.Max(1, float64(a.Duration))
		if t > 1 {
			t = 1
		}
		et := smoothstep(t)

		curX := a.FromX + (a.ToX-a.FromX)*et
		curY := a.FromY + (a.ToY-a.FromY)*et

		if comp, ok := world.Components.Get(id, "moving_animation"); ok {
			ma := comp.(*component.MovingAnimation)
			ma.CurrentX = curX
			ma.CurrentY = curY
			ma.CurrentTick = a.Tick
		}

		if t >= 1 {
			// Terminé : on retire le composant d'animation pour rendre la tuile de façon stable
			world.Components.Remove(id, "moving_animation")
			delete(m.animes, id)
			// Publie l'événement de fin
			world.EventBus.PublishImmediate(event.Event{
				Type:     event.AnimationEnded,
				SourceID: id,
				Payload: map[string]interface{}{
					"grid_id": a.GridID,
				},
				Timestamp: time.Now(),
			})
		}
	}
}
