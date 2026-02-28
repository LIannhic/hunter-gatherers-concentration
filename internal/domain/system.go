package domain

import (
	"errors"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
)

// System interface pour les systèmes ECS
type System interface {
	Update(world *World)
	Priority() int // Ordre d'exécution
}

// World contient tout l'état du jeu
type World struct {
	Grids    map[string]*board.Grid // Plusieurs grids indexés par ID
	GridOrder []string              // Ordre stable des IDs de grid (pour affichage)
	Entities *entity.Manager
	Components *component.Store
	EventBus   *event.Bus
	Turn       int
	MaxTurns   int
	PlayerID   string

	// Grids actifs pour le joueur (pour navigation entre grids)
	CurrentGridID string

	// Factories
	CreatureFactory *creature.Factory
	ResourceFactory *resource.Factory

	// Player
	playerPosition entity.Position
}

func NewWorld() *World {
	return &World{
		Grids:           make(map[string]*board.Grid),
		GridOrder:       make([]string, 0),
		Entities:        entity.NewManager(),
		Components:      component.NewStore(),
		EventBus:        event.NewBus(),
		Turn:            0,
		MaxTurns:        100,
		CurrentGridID:   "",
		CreatureFactory: creature.NewFactory(),
		ResourceFactory: resource.NewFactory(),
		playerPosition:  entity.Position{X: 0, Y: 0},
	}
}

// CreateGrid crée un nouveau grid et l'ajoute au monde
func (w *World) CreateGrid(id string, width, height int) *board.Grid {
	grid := board.NewGrid(id, width, height)
	w.Grids[id] = grid
	w.GridOrder = append(w.GridOrder, id) // Garde l'ordre de création
	if w.CurrentGridID == "" {
		w.CurrentGridID = id
	}
	return grid
}

// GetGrid retourne un grid par son ID
func (w *World) GetGrid(id string) (*board.Grid, bool) {
	grid, ok := w.Grids[id]
	return grid, ok
}

// GetCurrentGrid retourne le grid actuel du joueur
func (w *World) GetCurrentGrid() (*board.Grid, bool) {
	if w.CurrentGridID == "" {
		return nil, false
	}
	return w.GetGrid(w.CurrentGridID)
}

// SetCurrentGrid change le grid actuel du joueur
func (w *World) SetCurrentGrid(gridID string) bool {
	if _, ok := w.Grids[gridID]; ok {
		w.CurrentGridID = gridID
		return true
	}
	return false
}

// GetGridForEntity retourne le grid sur lequel se trouve une entité
func (w *World) GetGridForEntity(entityID string) (*board.Grid, bool) {
	e, ok := w.Entities.Get(entity.ID(entityID))
	if !ok {
		return nil, false
	}
	return w.GetGrid(e.GetGridID())
}

func (w *World) SetPlayerPosition(pos entity.Position) {
	w.playerPosition = pos
}

func (w *World) GetPlayerPosition() entity.Position {
	return w.playerPosition
}

// SpawnResource crée une ressource dans le monde sur un grid spécifique
func (w *World) SpawnResource(gridID string, rtype string, pos entity.Position) (*resource.Resource, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	r := w.ResourceFactory.Create(rtype, pos)
	r.SetGridID(gridID)
	w.Entities.Register(r)
	w.Components.Add(string(r.GetID()), &r.Lifecycle)
	w.Components.Add(string(r.GetID()), &r.Value)
	w.Components.Add(string(r.GetID()), &r.Matchable)
	w.Components.Add(string(r.GetID()), &r.Visual)

	// Place sur la grille
	tile, _ := grid.Get(board.Position{X: pos.X, Y: pos.Y})
	tile.EntityID = string(r.GetID())

	w.EventBus.Publish(event.NewEntityCreatedEvent(string(r.GetID()), "resource"))
	return r, nil
}

// SpawnCreature crée une créature dans le monde sur un grid spécifique
func (w *World) SpawnCreature(gridID string, species string, pos entity.Position) (*creature.Creature, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	c, err := w.CreatureFactory.Create(species, pos)
	if err != nil {
		return nil, err
	}
	c.SetGridID(gridID)
	w.Entities.Register(c)
	w.Components.Add(string(c.GetID()), &c.Behavior)
	w.Components.Add(string(c.GetID()), &c.Mobility)
	w.Components.Add(string(c.GetID()), &c.Visual)

	// Place sur la grille
	tile, _ := grid.Get(board.Position{X: pos.X, Y: pos.Y})
	tile.EntityID = string(c.GetID())

	w.EventBus.Publish(event.NewEntityCreatedEvent(string(c.GetID()), "creature"))
	return c, nil
}

// RevealTile révèle une tuile sur un grid spécifique
// Note: Cette méthode ne publie pas d'événement - c'est à l'appelant de le faire avec la direction de flip
func (w *World) RevealTile(gridID string, pos board.Position) (*board.Tile, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	tile, err := grid.Reveal(pos)
	if err != nil {
		return nil, err
	}
	// L'événement doit être publié par l'appelant avec la direction de flip appropriée
	return tile, nil
}

// MatchTiles appaire deux tuiles sur un grid spécifique
func (w *World) MatchTile(gridID string, pos board.Position) error {
	grid, ok := w.Grids[gridID]
	if !ok {
		return ErrGridNotFound
	}

	if err := grid.Match(pos); err != nil {
		return err
	}
	tile, _ := grid.Get(pos)
	w.EventBus.Publish(event.NewTileMatchedEvent(
		entity.Position{X: pos.X, Y: pos.Y},
		tile.EntityID,
	))
	return nil
}

// RemoveEntity supprime une entité du monde
func (w *World) RemoveEntity(id entity.ID) {
	e, ok := w.Entities.Get(id)
	if !ok {
		return
	}

	// Retire de sa grille
	grid, ok := w.GetGrid(e.GetGridID())
	if ok {
		pos := e.GetPosition()
		tile, _ := grid.Get(board.Position{X: pos.X, Y: pos.Y})
		if tile.EntityID == string(id) {
			tile.EntityID = ""
		}
	}

	// Retire des composants
	w.Components.RemoveEntity(string(id))

	// Retire du manager
	w.Entities.Remove(id)

	w.EventBus.Publish(event.NewEntityRemovedEvent(string(id), "harvested"))
}

// ErrGridNotFound est retourné quand un grid n'existe pas
var ErrGridNotFound = errors.New("grid not found")

// LifecycleSystem gère maturation/dégradation
type LifecycleSystem struct{}

func (s *LifecycleSystem) Priority() int { return 1 }

func (s *LifecycleSystem) Update(world *World) {
	// Récupère toutes les entités avec un composant Lifecycle
	entityIDs := world.Components.QueryByComponent("lifecycle")

	for _, entityID := range entityIDs {
		comp, ok := world.Components.Get(entityID, "lifecycle")
		if !ok {
			continue
		}

		lifecycle, ok := comp.(*component.Lifecycle)
		if !ok {
			continue
		}

		// Progresse le cycle
		if lifecycle.Progress() {
			// Émet un événement de maturation
			world.EventBus.Publish(event.NewResourceMaturedEvent(
				entityID,
				lifecycle.GetCurrentStageName(),
			))
		}
	}
}

// PropagationSystem gère la propagation des ressources
type PropagationSystem struct{}

func (s *PropagationSystem) Priority() int { return 2 }

func (s *PropagationSystem) Update(world *World) {
	resources := world.Entities.GetByType(entity.TypeResource)

	for _, e := range resources {
		entityID := string(e.GetID())
		comp, ok := world.Components.Get(entityID, "lifecycle")
		if !ok {
			continue
		}

		lifecycle := comp.(*component.Lifecycle)
		if !lifecycle.CanPropagate {
			continue
		}

		// Récupère le grid de l'entité
		grid, ok := world.GetGrid(e.GetGridID())
		if !ok {
			continue
		}

		// Vérifie les cases adjacentes vides
		pos := e.GetPosition()
		neighbors := grid.GetNeighbors(board.Position{X: pos.X, Y: pos.Y})

		for _, neighbor := range neighbors {
			if neighbor.EntityID != "" {
				continue // Case occupée
			}

			// Probabilité de propagation
			if shouldPropagate(lifecycle) {
				// Crée une nouvelle ressource sur le même grid
				newRes := world.ResourceFactory.Create(
					getResourceType(e),
					entity.Position{X: neighbor.Position.X, Y: neighbor.Position.Y},
				)
				newRes.SetGridID(e.GetGridID())
				world.Entities.Register(newRes)
				world.Components.Add(string(newRes.GetID()), &newRes.Lifecycle)
				world.Components.Add(string(newRes.GetID()), &newRes.Value)

				neighbor.EntityID = string(newRes.GetID())

				world.EventBus.Publish(event.Event{
					Type:     event.ResourcePropagated,
					SourceID: string(newRes.GetID()),
					Payload: map[string]interface{}{
						"parent_id": entityID,
						"position":  neighbor.Position,
					},
				})
			}
		}
	}
}

func shouldPropagate(l *component.Lifecycle) bool {
	// Logique de probabilité basée sur le stade
	return l.CurrentStage >= 2 // Fruit mûr
}

func getResourceType(e entity.Entity) string {
	if r, ok := e.(*resource.Resource); ok {
		return r.ResourceType
	}
	return "unknown"
}

// CreatureAISystem gère les comportements
type CreatureAISystem struct{}

func (s *CreatureAISystem) Priority() int { return 3 }

func (s *CreatureAISystem) Update(world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	ai := world.CreatureFactory.GetAI()

	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok {
			continue
		}

		// Récupère le grid de la créature
		grid, ok := world.GetGrid(c.GetGridID())
		if !ok {
			continue
		}

		// Prend une décision
		action := ai.Decide(c, &worldAdapter{world: world, grid: grid})

		// Exécute l'action
		switch action.Type {
		case "move":
			newPos := entity.Position{
				X: c.GetPosition().X + action.Direction.X,
				Y: c.GetPosition().Y + action.Direction.Y,
			}

			// Met à jour la grille
			oldTile, _ := grid.Get(board.Position{X: c.GetPosition().X, Y: c.GetPosition().Y})
			newTile, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})

			oldTile.EntityID = ""
			newTile.EntityID = string(c.GetID())

			world.Entities.UpdatePosition(c.GetID(), newPos)

			world.EventBus.Publish(event.NewCreatureMovedEvent(
				string(c.GetID()),
				c.GetPosition(),
				newPos,
			))

		case "transform":
			// Logique de transformation (pollinisation, etc.)
			targetID := action.TargetID
			if targetID != "" {
				if comp, ok := world.Components.Get(targetID, "lifecycle"); ok {
					if lifecycle, ok := comp.(*component.Lifecycle); ok {
						lifecycle.CurrentStage++ // Force la maturation
					}
				}
			}
		}
	}
}

// worldAdapter adapte World pour l'interface creature.WorldState
type worldAdapter struct {
	world *World
	grid  *board.Grid
}

func (wa *worldAdapter) GetPlayerPosition() entity.Position {
	return wa.world.playerPosition
}

func (wa *worldAdapter) GetNearbyCreatures(pos entity.Position, radius int) []*creature.Creature {
	var result []*creature.Creature
	creatures := wa.world.Entities.GetByType(entity.TypeCreature)

	for _, e := range creatures {
		// Vérifie que la créature est sur le même grid
		if e.GetGridID() != wa.grid.ID {
			continue
		}
		if c, ok := e.(*creature.Creature); ok {
			dist := abs(c.GetPosition().X-pos.X) + abs(c.GetPosition().Y-pos.Y)
			if dist <= radius {
				result = append(result, c)
			}
		}
	}
	return result
}

func (wa *worldAdapter) GetResources(pos entity.Position, radius int) []string {
	var result []string
	resources := wa.world.Entities.GetByType(entity.TypeResource)

	for _, e := range resources {
		// Vérifie que la ressource est sur le même grid
		if e.GetGridID() != wa.grid.ID {
			continue
		}
		dist := abs(e.GetPosition().X-pos.X) + abs(e.GetPosition().Y-pos.Y)
		if dist <= radius {
			result = append(result, string(e.GetID()))
		}
	}
	return result
}

func (wa *worldAdapter) IsValidMove(pos entity.Position) bool {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return false
	}
	return tile.EntityID == "" && !tile.Modifier.Obstructed
}

func (wa *worldAdapter) GetTileState(pos entity.Position) string {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return "invalid"
	}
	switch tile.State {
	case board.Hidden:
		return "hidden"
	case board.Revealed:
		return "revealed"
	case board.Matched:
		return "matched"
	default:
		return "unknown"
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TriggerSystem gère les déclencheurs (terriers, etc.)
type TriggerSystem struct{}

func (s *TriggerSystem) Priority() int { return 4 }

func (s *TriggerSystem) Update(world *World) {
	// Vérifie les conditions de trigger sur chaque tuile de chaque grid
	for _, gridID := range world.GridOrder {
		grid, ok := world.GetGrid(gridID)
		if !ok {
			continue
		}
		for _, tile := range grid.Tiles {
			if tile.StructureID == "" {
				continue
			}

			// Récupère le composant trigger de la structure
			comp, ok := world.Components.Get(tile.StructureID, "trigger")
			if !ok {
				continue
			}

			trigger := comp.(*component.Trigger)

			// Évalue la condition
			if s.checkCondition(trigger.Condition, tile, world, grid) {
				s.executeAction(trigger.Action, tile, world, grid)
				if trigger.Consumed {
					// Supprime le trigger
					world.Components.Remove(tile.StructureID, "trigger")
				}
			}
		}
	}
}

func (s *TriggerSystem) checkCondition(condition string, tile *board.Tile, world *World, grid *board.Grid) bool {
	switch condition {
	case "reveal_with_creature":
		// Si la tuile est révélée ET contient une créature
		if tile.State != board.Revealed {
			return false
		}
		if tile.EntityID == "" {
			return false
		}
		e, ok := world.Entities.Get(entity.ID(tile.EntityID))
		return ok && e.GetType() == entity.TypeCreature

	case "creature_on_resource":
		if tile.EntityID == "" {
			return false
		}
		e, ok := world.Entities.Get(entity.ID(tile.EntityID))
		if !ok || e.GetType() != entity.TypeCreature {
			return false
		}
		// Vérifie si une ressource est à proximité
		neighbors := grid.GetNeighbors(tile.Position)
		for _, n := range neighbors {
			if n.EntityID != "" {
				if res, ok := world.Entities.Get(entity.ID(n.EntityID)); ok {
					if res.GetType() == entity.TypeResource {
						return true
					}
				}
			}
		}
	}
	return false
}

func (s *TriggerSystem) executeAction(action string, tile *board.Tile, world *World, grid *board.Grid) {
	switch action {
	case "creature_flee":
		// La créature fuit (change de comportement)
		if tile.EntityID != "" {
			if e, ok := world.Entities.Get(entity.ID(tile.EntityID)); ok {
				if c, ok := e.(*creature.Creature); ok {
					c.Behavior.State = "fleeing"
				}
			}
		}

	case "reveal_adjacent":
		// Révèle les tuiles adjacentes
		neighbors := grid.GetNeighbors(tile.Position)
		for _, n := range neighbors {
			if n.State == board.Hidden {
				n.State = board.Revealed
			}
		}
	}
}

// Engine orchestre tous les systèmes
type Engine struct {
	systems []System
	world   *World
	Running bool
}

func NewEngine(world *World) *Engine {
	return &Engine{
		world: world,
		systems: []System{
			&LifecycleSystem{},
			&PropagationSystem{},
			&CreatureAISystem{},
			&TriggerSystem{},
		},
		Running: false,
	}
}

func (e *Engine) Start() {
	e.Running = true
}

func (e *Engine) Stop() {
	e.Running = false
}

func (e *Engine) Update() {
	if !e.Running {
		return
	}

	// Trie par priorité (bubble sort simple)
	for i := 0; i < len(e.systems)-1; i++ {
		for j := i + 1; j < len(e.systems); j++ {
			if e.systems[i].Priority() > e.systems[j].Priority() {
				e.systems[i], e.systems[j] = e.systems[j], e.systems[i]
			}
		}
	}

	// Exécute chaque système
	for _, sys := range e.systems {
		sys.Update(e.world)
	}

	// Traite les événements
	e.world.EventBus.ProcessQueue()

	e.world.Turn++

	e.world.EventBus.Publish(event.NewTurnEndedEvent(e.world.Turn))
}

func (e *Engine) AddSystem(s System) {
	e.systems = append(e.systems, s)
}

func (e *Engine) GetWorld() *World {
	return e.world
}

func (e *Engine) GetTurn() int {
	return e.world.Turn
}
