package domain

import (
	"errors"
	"fmt"
	"math/rand"

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
	Grids      map[string]*board.Grid // Plusieurs grids indexés par ID
	GridOrder  []string               // Ordre stable des IDs de grid (pour affichage)
	Entities   *entity.Manager
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

	// Turn state tracking
	tilesFlippedThisTurn []board.Position // Tracks tiles flipped in current turn (max 2)
	lastTurnNumber       int              // Used to detect turn changes
}

func NewWorld() *World {
	return &World{
		Grids:                make(map[string]*board.Grid),
		GridOrder:            make([]string, 0),
		Entities:             entity.NewManager(),
		Components:           component.NewStore(),
		EventBus:             event.NewBus(),
		Turn:                 0,
		MaxTurns:             100,
		CurrentGridID:        "",
		CreatureFactory:      creature.NewFactory(),
		ResourceFactory:      resource.NewFactory(),
		playerPosition:       entity.Position{X: 0, Y: 0},
		tilesFlippedThisTurn: make([]board.Position, 0),
		lastTurnNumber:       0,
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

// AddFlippedTile adds a flipped tile to the current turn's tracking
func (w *World) AddFlippedTile(pos board.Position) {
	w.tilesFlippedThisTurn = append(w.tilesFlippedThisTurn, pos)
}

// GetFlippedTilesCount returns how many tiles have been flipped this turn
func (w *World) GetFlippedTilesCount() int {
	// Reset if turn has changed
	if w.lastTurnNumber != w.Turn {
		w.tilesFlippedThisTurn = make([]board.Position, 0)
		w.lastTurnNumber = w.Turn
	}
	return len(w.tilesFlippedThisTurn)
}

// CanFlipTile checks if another tile can be flipped this turn (max 2 per turn)
func (w *World) CanFlipTile() bool {
	w.GetFlippedTilesCount() // Sync turn tracking
	return len(w.tilesFlippedThisTurn) < 2
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

	// Vérifie si la case est valide et libre
	tile, err := grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return nil, err
	}
	if tile.EntityID != "" {
		return nil, fmt.Errorf("position (%d,%d) is already occupied by entity %s", pos.X, pos.Y, tile.EntityID)
	}
	if tile.Modifier.Obstructed {
		return nil, fmt.Errorf("position (%d,%d) is obstructed", pos.X, pos.Y)
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

		// Si la créature a un MovementProfile avancé, on utilise plutôt CreatureMovementSystem
		if c.MovementProfile != nil {
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

// CreatureMovementSystem gère les déplacements avancés des créatures
// Utilise les MovementProfile pour un contrôle fin du mouvement
type CreatureMovementSystem struct {
	recentReveals []board.Position // Tuiles récemment révélées pour TriggerOnEcho
}

func NewCreatureMovementSystem() *CreatureMovementSystem {
	return &CreatureMovementSystem{
		recentReveals: make([]board.Position, 0),
	}
}

func (s *CreatureMovementSystem) Priority() int { return 3 }

func (s *CreatureMovementSystem) TrackReveal(pos board.Position) {
	s.recentReveals = append(s.recentReveals, pos)
}

func (s *CreatureMovementSystem) ClearReveals() {
	s.recentReveals = s.recentReveals[:0]
}

func (s *CreatureMovementSystem) Update(world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)

	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok || c.MovementProfile == nil {
			continue
		}

		// Récupère le grid de la créature
		grid, ok := world.GetGrid(c.GetGridID())
		if !ok {
			continue
		}

		profile := c.MovementProfile

		// Vérifie si le déplacement doit se déclencher
		if !s.shouldTrigger(profile.Trigger, c, world, grid) {
			profile.Trigger.Reset()
			continue
		}

		// Vérifie la fréquence de déplacement
		if !profile.Frequency.CanMove() {
			continue
		}

		// Détermine combien de cases déplacer
		moveCount := profile.Frequency.GetMoveCount()

		// Exécute les mouvements
		for i := 0; i < moveCount; i++ {
			if !s.executeMove(c, profile, world, grid) {
				break
			}
		}

		profile.Trigger.Reset()
	}
}

func (s *CreatureMovementSystem) shouldTrigger(trigger creature.MovementTrigger, c *creature.Creature, world *World, grid *board.Grid) bool {
	switch trigger.Type {
	case creature.TriggerPassive:
		return false
	case creature.TriggerAuto:
		return true
	case creature.TriggerOnReveal:
		// Vérifie si la créature est sur une tuile révélée
		tile, _ := grid.Get(board.Position{X: c.GetPosition().X, Y: c.GetPosition().Y})
		if tile.State == board.Revealed && !trigger.WasRevealed {
			trigger.WasRevealed = true
			return true
		}
		trigger.WasRevealed = tile.State == board.Revealed
		return false
	case creature.TriggerOnEcho:
		// Se déclenche si une autre tuile a été révélée récemment
		return len(s.recentReveals) > 0
	case creature.TriggerProximity:
		// Vérifie si une tuile a été révélée dans le rayon
		for _, revealed := range s.recentReveals {
			dist := abs(revealed.X-c.GetPosition().X) + abs(revealed.Y-c.GetPosition().Y)
			if dist <= trigger.Radius {
				return true
			}
		}
		return false
	}
	return false
}

func (s *CreatureMovementSystem) executeMove(c *creature.Creature, profile *creature.MovementProfile, world *World, grid *board.Grid) bool {
	// Détermine la direction selon la navigation
	direction := s.getNavigationDirection(profile.Navigation, c, world, grid)

	if direction == (entity.Position{X: 0, Y: 0}) {
		return true // Pas de mouvement mais pas d'échec
	}

	// Calcule la nouvelle position
	currentPos := c.GetPosition()
	newPos := entity.Position{
		X: currentPos.X + direction.X,
		Y: currentPos.Y + direction.Y,
	}

	// Met à jour l'orientation
	profile.Orientation = directionToOrientation(direction)

	// Vérifie la validité et gère les collisions
	finalPos, success := s.handleCollision(profile.Collision, c, newPos, currentPos, world, grid)
	if !success {
		return false
	}

	// Exécute le mouvement selon le mode
	return s.applyMoveMode(profile.Mode, c, currentPos, finalPos, world, grid)
}

func (s *CreatureMovementSystem) getNavigationDirection(nav creature.NavigationLogic, c *creature.Creature, world *World, grid *board.Grid) entity.Position {
	switch nav.Type {
	case creature.NavWander:
		directions := []entity.Position{
			{X: 0, Y: -1}, {X: 0, Y: 1},
			{X: -1, Y: 0}, {X: 1, Y: 0},
		}
		// 30% de chance de suivre la direction privilégiée
		if nav.WanderBias != (entity.Position{}) && rand.Float32() < 0.3 {
			newPos := entity.Position{
				X: c.GetPosition().X + nav.WanderBias.X,
				Y: c.GetPosition().Y + nav.WanderBias.Y,
			}
			if tile, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y}); err == nil && tile.EntityID == "" && !tile.Modifier.Obstructed {
				return nav.WanderBias
			}
		}
		return directions[rand.Intn(len(directions))]

	case creature.NavPatrol:
		if len(nav.PatrolRoute) == 0 {
			return s.getNavigationDirection(creature.NavigationLogic{Type: creature.NavWander}, c, world, grid)
		}
		// Patrouille simplifiée
		target := nav.PatrolRoute[nav.PatrolIndex]
		current := c.GetPosition()
		dir := entity.Position{
			X: sign(target.X - current.X),
			Y: sign(target.Y - current.Y),
		}
		if dir.X == 0 && dir.Y == 0 {
			nextIndex := (nav.PatrolIndex + 1) % len(nav.PatrolRoute)
			target = nav.PatrolRoute[nextIndex]
			dir = entity.Position{
				X: sign(target.X - current.X),
				Y: sign(target.Y - current.Y),
			}
		}
		return dir

	case creature.NavOrientation:
		return c.MovementProfile.Orientation.ToVector()

	case creature.NavAttraction:
		// Vers le joueur (simplifié)
		playerPos := world.playerPosition
		current := c.GetPosition()
		return entity.Position{
			X: sign(playerPos.X - current.X),
			Y: sign(playerPos.Y - current.Y),
		}

	case creature.NavRepulsion:
		// S'éloigne du joueur
		playerPos := world.playerPosition
		current := c.GetPosition()
		return entity.Position{
			X: sign(current.X - playerPos.X),
			Y: sign(current.Y - playerPos.Y),
		}
	}
	return entity.Position{X: 0, Y: 0}
}

func (s *CreatureMovementSystem) handleCollision(coll creature.CollisionHandler, c *creature.Creature, newPos, currentPos entity.Position, world *World, grid *board.Grid) (entity.Position, bool) {
	tile, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
	if err != nil {
		return currentPos, false
	}

	canMove := tile.EntityID == "" && !tile.Modifier.Obstructed

	switch coll.Type {
	case creature.CollideStop:
		if !canMove {
			return currentPos, false
		}
		return newPos, true

	case creature.CollideBounce:
		if !canMove {
			// Inverse l'orientation
			c.MovementProfile.Orientation.Rotate(180)
			return currentPos, false
		}
		return newPos, true

	case creature.CollideSlide:
		if canMove {
			return newPos, true
		}
		// Essaie de glisser
		dx := newPos.X - currentPos.X
		dy := newPos.Y - currentPos.Y
		// Glisse horizontalement
		if dy != 0 {
			slidePos := entity.Position{X: currentPos.X, Y: newPos.Y}
			if t, err := grid.Get(board.Position{X: slidePos.X, Y: slidePos.Y}); err == nil && t.EntityID == "" && !t.Modifier.Obstructed {
				return slidePos, true
			}
		}
		// Glisse verticalement
		if dx != 0 {
			slidePos := entity.Position{X: newPos.X, Y: currentPos.Y}
			if t, err := grid.Get(board.Position{X: slidePos.X, Y: slidePos.Y}); err == nil && t.EntityID == "" && !t.Modifier.Obstructed {
				return slidePos, true
			}
		}
		return currentPos, false

	case creature.CollidePhase:
		// Vérifie si on peut traverser
		tileType := "empty"
		if tile.Modifier.Obstructed {
			tileType = "wall"
		}
		for _, phaseType := range coll.CanPhaseThrough {
			if tileType == phaseType {
				return newPos, true
			}
		}
		if !canMove {
			return currentPos, false
		}
		return newPos, true
	}

	return newPos, true
}

func (s *CreatureMovementSystem) applyMoveMode(mode creature.MovementMode, c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid) bool {
	switch mode.Type {
	case creature.ModeBento:
		// Déplacement visible standard
		return s.doMove(c, oldPos, newPos, world, grid, false)

	case creature.ModeShadow:
		// Déplacement invisible
		return s.doMove(c, oldPos, newPos, world, grid, true)

	case creature.ModeSwap:
		// Échange avec l'entité à la position cible
		tile, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
		if tile.EntityID != "" {
			// Trouve l'entité à échanger
			swappedEntity, ok := world.Entities.Get(entity.ID(tile.EntityID))
			if ok {
				// Échange les positions
				swappedEntity.SetPosition(oldPos)
				c.SetPosition(newPos)
				// Met à jour la grille
				oldTile, _ := grid.Get(board.Position{X: oldPos.X, Y: oldPos.Y})
				newTile, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
				oldTile.EntityID = tile.EntityID
				newTile.EntityID = string(c.GetID())
				// Met à jour le manager
				world.Entities.UpdatePosition(swappedEntity.GetID(), oldPos)
				world.Entities.UpdatePosition(c.GetID(), newPos)
				// Événement spécial pour swap
				world.EventBus.Publish(event.Event{
					Type:     event.CreatureMoved,
					SourceID: string(c.GetID()),
					Payload: map[string]interface{}{
						"from":         oldPos,
						"to":           newPos,
						"mode":         "swap",
						"swapped_with": string(swappedEntity.GetID()),
					},
				})
				return true
			}
		}
		return s.doMove(c, oldPos, newPos, world, grid, false)

	case creature.ModeOver:
		c.AddTag("flying")
		return s.doMove(c, oldPos, newPos, world, grid, false)

	case creature.ModeUnder:
		c.AddTag("burrowed")
		return s.doMove(c, oldPos, newPos, world, grid, true)
	}

	return s.doMove(c, oldPos, newPos, world, grid, false)
}

func (s *CreatureMovementSystem) doMove(c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid, silent bool) bool {
	// Met à jour la grille
	oldTile, _ := grid.Get(board.Position{X: oldPos.X, Y: oldPos.Y})
	newTile, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})

	oldTile.EntityID = ""
	newTile.EntityID = string(c.GetID())

	// Met à jour la position de l'entité
	c.SetPosition(newPos)
	world.Entities.UpdatePosition(c.GetID(), newPos)

	// Publie l'événement
	if silent {
		world.EventBus.Publish(event.Event{
			Type:     event.CreatureMoved,
			SourceID: string(c.GetID()),
			Payload: map[string]interface{}{
				"from":   oldPos,
				"to":     newPos,
				"mode":   "silent",
				"hidden": true,
			},
		})
	} else {
		world.EventBus.Publish(event.NewCreatureMovedEvent(
			string(c.GetID()),
			oldPos,
			newPos,
		))
	}

	return true
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

func directionToOrientation(dir entity.Position) creature.Orientation {
	if dir.X > 0 {
		return creature.Orientation{Direction: creature.DirEast}
	}
	if dir.X < 0 {
		return creature.Orientation{Direction: creature.DirWest}
	}
	if dir.Y > 0 {
		return creature.Orientation{Direction: creature.DirSouth}
	}
	return creature.Orientation{Direction: creature.DirNorth}
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
	systems        []System
	world          *World
	Running        bool
	movementSystem *CreatureMovementSystem // Référence directe pour les mises à jour
}

func NewEngine(world *World) *Engine {
	moveSys := NewCreatureMovementSystem()
	return &Engine{
		world: world,
		systems: []System{
			&LifecycleSystem{},
			&PropagationSystem{},
			&CreatureAISystem{},
			moveSys,
			&TriggerSystem{},
		},
		Running:        false,
		movementSystem: moveSys,
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

	// Réinitialise les révélations du tour précédent
	if e.movementSystem != nil {
		e.movementSystem.ClearReveals()
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

// TrackTileReveal notifie le système de mouvement qu'une tuile a été révélée
// Utilisé pour les triggers OnEcho et Proximity
func (e *Engine) TrackTileReveal(pos board.Position) {
	if e.movementSystem != nil {
		e.movementSystem.TrackReveal(pos)
	}
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
