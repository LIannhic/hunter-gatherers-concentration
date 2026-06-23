package renderer

import (
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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
	EntityID      string
	GridID        string
	FromX, FromY  float64
	ToX, ToY      float64
	Tick          int
	Duration      int
	FromGridX     int
	FromGridY     int
	TargetGridX   int
	TargetGridY   int
	Layer         Layer
	Mode          string // <--- Stocke "earthquake", "slide", "swap", etc.
	FlipDirection entity.FlipDirection
	ArcSign       float64 // +1.0 ou -1.0, direction de l'arc perpendiculaire au mouvement (pour swap)
}

// AttackAnim représente un lunge visuel (brusque aller, lent retour)
type AttackAnim struct {
	EntityID     string
	DirX, DirY   float64
	Tick         int
	Duration     int
	MaxAmplitude float64
}

// AnimationManager gère plusieurs translations et attaques en parallèle
type AnimationManager struct {
	renderer    *BoardRenderer
	animes      map[string]*TranslationAnim // key = entityID
	attackAnimes map[string]*AttackAnim      // key = entityID
}

func NewAnimationManager(r *BoardRenderer) *AnimationManager {
	return &AnimationManager{
		renderer:    r,
		animes:      make(map[string]*TranslationAnim),
		attackAnimes: make(map[string]*AttackAnim),
	}
}

// StartTileMove démarre une animation de translation basée sur des positions de grille
func (m *AnimationManager) StartTileMove(world *domain.World, gridID string, entityID string, fromPos, toPos board.Position, durationTicks int, layer Layer, mode string, flipDirection entity.FlipDirection) {
	if mode == "" {
		mode = "slide"
	}
	grid, _ := world.GetGrid(gridID)
	isPortal := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)
	fx, fy := m.renderer.calculateTileScreenPos(fromPos, grid, isPortal)
	tx, ty := m.renderer.calculateTileScreenPos(toPos, grid, isPortal)

	// Crée/écrase le composant MovingAnimation attaché à l'entité
	ma := &component.MovingAnimation{
		StartX:        fx,
		StartY:        fy,
		FromGridX:     fromPos.X,
		FromGridY:     fromPos.Y,
		CurrentX:      fx,
		CurrentY:      fy,
		TargetGridX:   toPos.X,
		TargetGridY:   toPos.Y,
		CurrentTick:   0,
		DurationTicks: durationTicks,
	}
	world.Components.Add(entityID, ma)

	arcSign := 1.0

	m.animes[entityID] = &TranslationAnim{
		EntityID:      entityID,
		GridID:        gridID,
		FromX:         fx,
		FromY:         fy,
		ToX:           tx,
		ToY:           ty,
		Tick:          0,
		Duration:      durationTicks,
		FromGridX:     fromPos.X,
		FromGridY:     fromPos.Y,
		TargetGridX:   toPos.X,
		TargetGridY:   toPos.Y,
		Layer:         layer,
		Mode:          mode,
		FlipDirection: flipDirection,
		ArcSign:       arcSign,
	}

	world.EventBus.PublishImmediate(event.NewAnimationStartedEvent(mode, entityID))
}

// StartAttack démarre une animation de lunge brusque
func (m *AnimationManager) StartAttack(world *domain.World, entityID string, dx, dy float64, hitTarget *entity.Position) {
	duration := 25
	m.attackAnimes[entityID] = &AttackAnim{
		EntityID:     entityID,
		DirX:         dx,
		DirY:         dy,
		Tick:         0,
		Duration:     duration,
		MaxAmplitude: 15.0, // Pixels de décalage max
	}

	aa := &component.AttackingAnimation{
		OffsetX:       0,
		OffsetY:       0,
		CurrentTick:   0,
		DurationTicks: duration,
		HitTarget:     hitTarget,
	}
	world.Components.Add(entityID, aa)

	world.EventBus.PublishImmediate(event.NewAnimationStartedEvent("attack", entityID))
}

func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// Update avance toutes les translations proprement
func (m *AnimationManager) Update(world *domain.World) {
	if len(m.animes) == 0 && len(m.attackAnimes) == 0 {
		return
	}

	anyAnimationFinished := false

	// --- Mise à jour des translations (Slide) ---
	for id, a := range m.animes {
		a.Tick++
		t := float64(a.Tick) / math.Max(1, float64(a.Duration))
		if t > 1 {
			t = 1
		}
		et := smoothstep(t)

		var curX, curY float64
		if a.Mode == "swap" || a.Mode == "swap_under" {
			dx := a.ToX - a.FromX
			dy := a.ToY - a.FromY
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1 {
				dist = 1
			}

			arcHeight := dist * 0.3 * a.ArcSign

			nx := -dy / dist
			ny := dx / dist

			midX := (a.FromX + a.ToX) / 2 + nx*arcHeight
			midY := (a.FromY + a.ToY) / 2 + ny*arcHeight

			omt := 1.0 - et
			curX = omt*omt*a.FromX + 2*omt*et*midX + et*et*a.ToX
			curY = omt*omt*a.FromY + 2*omt*et*midY + et*et*a.ToY
		} else {
			curX = a.FromX + (a.ToX-a.FromX)*et
			curY = a.FromY + (a.ToY-a.FromY)*et
		}

		if comp, ok := world.Components.Get(id, "moving_animation"); ok {
			ma := comp.(*component.MovingAnimation)
			ma.CurrentX = curX
			ma.CurrentY = curY
			ma.CurrentTick = a.Tick
		}

		if t >= 1 {
			world.Components.Remove(id, "moving_animation")
			delete(m.animes, id)
			anyAnimationFinished = true
		}
	}

	// --- Mise à jour des attaques (Lunge) ---
	for id, a := range m.attackAnimes {
		a.Tick++
		t := float64(a.Tick) / math.Max(1, float64(a.Duration))

		var offset float64
		lungeThreshold := 0.2 // 20% du temps pour l'aller brusque
		if t < lungeThreshold {
			// Aller brusque : 0 -> 1 en 20% du temps
			it := t / lungeThreshold
			offset = it * a.MaxAmplitude
		} else {
			// Retour lent : 1 -> 0 en 80% du temps
			it := (t - lungeThreshold) / (1.0 - lungeThreshold)
			if it > 1 {
				it = 1
			}
			// Utilisation de 1 - it pour le retour, avec un petit lissage
			offset = (1.0 - smoothstep(it)) * a.MaxAmplitude
		}

		if comp, ok := world.Components.Get(id, "attacking_animation"); ok {
			aa := comp.(*component.AttackingAnimation)
			aa.OffsetX = a.DirX * offset
			aa.OffsetY = a.DirY * offset
			aa.CurrentTick = a.Tick
		}

		if t >= 1 {
			var hitTarget interface{}
			if comp, ok := world.Components.Get(id, "attacking_animation"); ok {
				aa := comp.(*component.AttackingAnimation)
				if aa.HitTarget != nil {
					hitTarget = *aa.HitTarget
				}
			}

			world.Components.Remove(id, "attacking_animation")
			delete(m.attackAnimes, id)
			anyAnimationFinished = true

			payload := map[string]interface{}{
				"animation_type": "attack",
			}
			if hitTarget != nil {
				payload["hit_target"] = hitTarget
			}

			world.EventBus.PublishImmediate(event.Event{
				Type:     event.AnimationEnded,
				SourceID: id,
				Payload:  payload,
			})
		}
	}

	if anyAnimationFinished && len(m.animes) == 0 && len(m.attackAnimes) == 0 {
		world.EventBus.PublishImmediate(event.NewAnimationEndedEvent("manager_global", "manager_global"))
	}
}
