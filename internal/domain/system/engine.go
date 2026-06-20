package system

import (
	"fmt"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// System interface pour les systèmes ECS
type System interface {
	Update(world *World)
	Priority() int // Ordre d'exécution
}

// Engine orchestre tous les systèmes
type Engine struct {
	systems        []System
	world          *World
	movementSystem *CreatureMovementSystem // Référence directe pour les mises à jour
	previewSystem  *PreviewSystem          // Référence pour les événements
	lootSystem     *LootSystem
}

// NewEngine initialise le moteur de jeu avec ses systèmes
func NewEngine(world *World) *Engine {
	moveSys := NewCreatureMovementSystem(world)
	prevSys := NewPreviewSystem()
	lootSys := NewLootSystem(world)

	e := &Engine{
		world: world,
		systems: []System{
			&LifecycleSystem{},
			&PropagationSystem{},
			NewAggressionSystem(world),
			&CreatureAISystem{},
			moveSys,
			&TriggerSystem{},
			&ToxicitySystem{},
			&TrackSystem{},
			lootSys,
		},
		movementSystem: moveSys,
		previewSystem:  prevSys,
		lootSystem:     lootSys,
	}

	// Lie l'engine au monde pour permettre aux systèmes de communiquer
	world.Engine = e

	// S'abonne aux entrées de grille
	world.EventBus.SubscribeFunc(event.GridEntered, func(ev event.Event) {
		gridID, _ := ev.Payload["grid_id"].(string)
		prevSys.OnEnterGrid(world, gridID)
	})

	return e
}

// ResetPreviews réinitialise le suivi des prévisualisations (pour une nouvelle partie)
func (e *Engine) ResetPreviews() {
	if e.previewSystem != nil {
		e.previewSystem.Reset()
	}
}

// Update fait progresser le tour de jeu
func (e *Engine) Update() {
	// Tri des systèmes par priorité
	for i := 0; i < len(e.systems)-1; i++ {
		for j := i + 1; j < len(e.systems); j++ {
			if e.systems[i].Priority() > e.systems[j].Priority() {
				e.systems[i], e.systems[j] = e.systems[j], e.systems[i]
			}
		}
	}

	for _, sys := range e.systems {
		sys.Update(e.world)
	}

	// Nettoie les révélations après le traitement des systèmes
	if e.movementSystem != nil {
		e.movementSystem.ClearReveals()
	}

	e.world.EventBus.ProcessQueue()
	e.world.Turn++

	// Rafraîchit l'état de navigation à la fin du tour pour détecter les changements de population
	e.world.IsNavigationOpen(e.world.CurrentGridID)

	// Diminue la santé mentale à chaque tour
	if e.world.Player != nil {
		e.world.Player.ConsumeSanity(1)

		// Décrémente la grâce (Flutterwing)
		if e.world.Player.GraceTurns > 0 {
			e.world.Player.GraceTurns--
		}

		// Met à jour les durées des effets visuels
		for name, duration := range e.world.Player.VisualEffects {
			if duration > 0 {
				e.world.Player.VisualEffects[name]--
			}
		}
	}

	e.world.EventBus.Publish(event.NewTurnEndedEvent(e.world.Turn))
}

// UpdateFrame effectue les mises à jour visuelles et les systèmes temps réel (pseudo-systèmes de frame)
func (e *Engine) UpdateFrame(dt float64) {
	// 1. Détecter grille vide (aucune entité matchable)
	isEmptyGrid := false
	if grid, ok := e.world.GetGrid(e.world.CurrentGridID); ok {
		hasMatchable := false
		for _, tile := range grid.Plots {
			if len(tile.EntitiesID) > 0 {
				if ent, ok := e.world.Entities.Get(entity.ID(tile.EntitiesID[len(tile.EntitiesID)-1])); ok {
					if ent.GetType() == entity.TypeResource || ent.GetType() == entity.TypeCreature {
						hasMatchable = true
						break
					}
				}
			}
		}
		isEmptyGrid = !hasMatchable
	}

	// 2. Détecter prévisualisation active
	isPreviewing := e.previewSystem != nil && e.previewSystem.IsPreviewActive(e.world.CurrentGridID)

	// 3. Détecter animations actives (via compteur dans World)
	isAnimating := e.world.ActiveAnimationCount > 0

	// 4. Calculer le facteur de temps
	timeScale := 1.0
	if isEmptyGrid {
		timeScale = 0.0 // Timer arrêté
	} else if isPreviewing || isAnimating {
		timeScale = 0.5 // 50% vitesse
	}

	// 5. Mise à jour du TurnTimer (Pression temporelle)
	if e.world.TurnTimer != nil {
		if timeScale > 0 {
			if e.world.TurnTimer.Update(dt * timeScale) {
				fmt.Println("[TIMER] Temps écoulé ! Auto-skip forcé.")
				e.world.TurnTimer.Reset()
				e.world.EventBus.PublishImmediate(event.Event{
					Type:     event.Type("turn_timer_expired"),
					SourceID: "engine",
				})
			}
		} else {
			e.world.TurnTimer.Stop()
		}

		targetDuration := e.world.Difficulty.TurnTimerDuration
		if e.world.Debug.OverrideDifficulty {
			targetDuration = e.world.Debug.Difficulty.TurnTimerDuration
		}
		if e.world.TurnTimer.MaxTime != targetDuration {
			e.world.TurnTimer.SetMaxTime(targetDuration)
		}
	}

	// 6. Mise à jour des systèmes temps réel (PreviewSystem, etc.)
	if e.previewSystem != nil {
		e.previewSystem.Update(e.world)
	}

	// 7. Traitement de la queue des événements à chaque frame
	// Essentiel pour que les événements Publish() (non-immédiats) soient consommés
	// par le Renderer ou les systèmes entre deux tours.
	e.world.EventBus.ProcessQueue()
}

// TrackTileReveal enregistre une interaction pour les systèmes de proximité
func (e *Engine) TrackTileReveal(pos board.Position, gridID string) {
	if e.movementSystem != nil {
		e.movementSystem.TrackReveal(pos, gridID)
	}
}

// AddSystem ajoute dynamiquement un système au moteur
func (e *Engine) AddSystem(s System) {
	e.systems = append(e.systems, s)
}

// GetWorld retourne la référence du monde
func (e *Engine) GetWorld() *World {
	return e.world
}

// GetTurn retourne le numéro du tour actuel
func (e *Engine) GetTurn() int {
	return e.world.Turn
}
