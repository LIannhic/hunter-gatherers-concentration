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
func (w *World) CreateGrid(id string, width, height int, biome board.BiomeType) *board.Grid {
	grid := board.NewGrid(id, width, height, biome)
	w.Grids[id] = grid
	w.GridOrder = append(w.GridOrder, id)
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
		return nil, fmt.Errorf("grid %s introuvable", gridID)
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}

	plot, err := grid.Get(boardPos)
	if err != nil {
		return nil, err
	}

	if len(plot.EntitiesID) > 0 {
		return nil, fmt.Errorf("position %v déjà occupée", pos)
	}
	if plot.Modifier.Obstructed {
		return nil, fmt.Errorf("position %v est obstruée", pos)
	}

	r := w.ResourceFactory.Create(rtype, entity.Position{X: pos.X, Y: pos.Y})
	r.SetGridID(gridID)

	idStr := string(r.GetID())
	w.Entities.Register(r)
	w.Components.Add(idStr, &r.Lifecycle)
	w.Components.Add(idStr, &r.Value)
	w.Components.Add(idStr, &r.Matchable)
	w.Components.Add(idStr, &r.Visual)

	plot.PushEntity(idStr)

	w.EventBus.Publish(event.NewEntityCreatedEvent(idStr, "resource"))
	return r, nil
}

// SpawnCreature crée une créature dans le monde sur un grid spécifique
func (w *World) SpawnCreature(gridID string, species string, pos entity.Position) (*creature.Creature, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}

	plot, err := grid.Get(boardPos)
	if err != nil {
		return nil, err
	}

	if len(plot.EntitiesID) > 0 {
		return nil, fmt.Errorf("position %v is already occupied by %d entities", pos, len(plot.EntitiesID))
	}
	if plot.Modifier.Obstructed {
		return nil, fmt.Errorf("position %v is obstructed", pos)
	}

	c, err := w.CreatureFactory.Create(species, pos)
	if err != nil {
		return nil, err
	}

	c.SetGridID(gridID)
	idStr := string(c.GetID())

	w.Entities.Register(c)
	w.Components.Add(idStr, &c.Behavior)
	w.Components.Add(idStr, &c.Mobility)
	w.Components.Add(idStr, &c.Visual)

	plot.PushEntity(idStr)

	w.EventBus.Publish(event.NewEntityCreatedEvent(idStr, "creature"))
	return c, nil
}

// RevealTile révèle une entité sur une position
// RevealTile révèle l'entité au sommet d'une pile sur une position donnée
func (w *World) RevealTile(gridID string, pos board.Position) (entity.Entity, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	// 1. Récupération du Plot (Parcelle)
	plot, err := grid.Get(pos)
	if err != nil {
		return nil, err
	}

	// 2. Vérification de la présence d'entités (Système de pile)
	n := len(plot.EntitiesID)
	if n == 0 {
		return nil, fmt.Errorf("aucune entité à la position %v", pos)
	}

	// 3. On récupère l'ID au SOMMET de la pile (le dernier ajouté)
	topEntityID := plot.EntitiesID[n-1]

	// 4. Récupération de l'entité via le Manager
	ent, ok := w.Entities.Get(entity.ID(topEntityID))
	if !ok {
		return nil, fmt.Errorf("l'entité %s est enregistrée sur le board mais absente du manager", topEntityID)
	}

	// 5. Mise à jour de l'état
	if ent.GetState() == entity.Hidden {
		ent.SetState(entity.Revealed)
	}

	return ent, nil
}

// MatchTile marque l'entité du SOMMET comme appairée
func (w *World) MatchTile(gridID string, pos board.Position) error {
	grid, ok := w.Grids[gridID]
	if !ok {
		return ErrGridNotFound
	}

	plot, err := grid.Get(pos)
	if err != nil {
		return err
	}

	n := len(plot.EntitiesID)
	if n == 0 {
		return errors.New("aucune entité à appairer à cette position")
	}

	topID := plot.EntitiesID[n-1]

	ent, ok := w.Entities.Get(entity.ID(topID))
	if !ok {
		return errors.New("entité au sommet non trouvée dans le manager")
	}

	ent.SetState(entity.Matched)

	w.EventBus.Publish(event.NewTileMatchedEvent(
		entity.Position{X: pos.X, Y: pos.Y},
		string(topID),
	))

	return nil
}

// RemoveEntity supprime une entité du monde, de sa pile sur la grille et de l'ECS
func (w *World) RemoveEntity(id entity.ID) {
	idStr := string(id)

	e, ok := w.Entities.Get(id)
	if !ok {
		return
	}

	grid, ok := w.Grids[e.GetGridID()]
	if ok {
		pos := e.GetPosition()
		_, err := grid.RemoveEntity(board.Position{X: pos.X, Y: pos.Y}, idStr)
		if err != nil {
			fmt.Printf("Erreur lors du retrait du board: %v\n", err)
		}
	}

	w.Components.RemoveEntity(idStr)
	w.Entities.Remove(id)

	w.EventBus.Publish(event.NewEntityRemovedEvent(idStr, "harvested"))
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

// PropagationSystem gère l'expansion organique des ressources sur la grille
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

		grid, ok := world.Grids[e.GetGridID()]
		if !ok {
			continue
		}

		pos := e.GetPosition()
		neighbors := grid.GetNeighbors(board.Position{X: pos.X, Y: pos.Y})

		for _, neighbor := range neighbors {
			if len(neighbor.EntitiesID) > 0 {
				continue
			}

			if neighbor.Modifier.Obstructed {
				continue
			}

			if shouldPropagate(lifecycle) {
				spawnPos := entity.Position{
					X: neighbor.Position.X,
					Y: neighbor.Position.Y,
				}

				newRes, err := world.SpawnResource(
					e.GetGridID(),
					getResourceType(e),
					spawnPos,
				)

				if err != nil {
					continue
				}

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
	return l.CurrentStage >= 2
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
		if !ok || c.MovementProfile != nil {
			continue
		}

		grid, ok := world.Grids[c.GetGridID()]
		if !ok {
			continue
		}

		action := ai.Decide(c, &worldAdapter{world: world, grid: grid})

		switch action.Type {
		case "move":
			oldPos := c.GetPosition()
			newPos := entity.Position{
				X: oldPos.X + action.Direction.X,
				Y: oldPos.Y + action.Direction.Y,
			}

			if !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
				continue
			}

			newPlot, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})

			idStr := string(c.GetID())
			_, err := grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)
			if err != nil {
				continue
			}

			if len(newPlot.EntitiesID) > 0 {
				topID := newPlot.EntitiesID[len(newPlot.EntitiesID)-1]
				if oldEnt, ok := world.Entities.Get(entity.ID(topID)); ok {
					if oldEnt.GetType() == entity.TypeTrap {
						world.RemoveEntity(oldEnt.GetID())
					}
				}
			}

			newPlot.PushEntity(idStr)

			world.Entities.UpdatePosition(c.GetID(), newPos)

			world.EventBus.Publish(event.NewCreatureMovedEvent(
				idStr,
				oldPos,
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
		if c.GetState() == entity.Revealed && !trigger.WasRevealed {
			trigger.WasRevealed = true
			return true
		}
		trigger.WasRevealed = c.GetState() == entity.Revealed
		return false
	case creature.TriggerOnEcho:
		return len(s.recentReveals) > 0
	case creature.TriggerProximity:
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
	direction := s.getNavigationDirection(profile.Navigation, c, world, grid)

	if direction == (entity.Position{X: 0, Y: 0}) {
		return true
	}

	currentPos := c.GetPosition()
	newPos := entity.Position{
		X: currentPos.X + direction.X,
		Y: currentPos.Y + direction.Y,
	}

	profile.Orientation = directionToOrientation(direction)

	finalPos, success := s.handleCollision(profile.Collision, c, newPos, currentPos, world, grid)
	if !success {
		return false
	}

	return s.applyMoveMode(profile.Mode, c, currentPos, finalPos, world, grid)
}

func (s *CreatureMovementSystem) getNavigationDirection(nav creature.NavigationLogic, c *creature.Creature, world *World, grid *board.Grid) entity.Position {
	switch nav.Type {
	case creature.NavWander:
		directions := []entity.Position{
			{X: 0, Y: -1}, {X: 0, Y: 1},
			{X: -1, Y: 0}, {X: 1, Y: 0},
		}
		if nav.WanderBias != (entity.Position{}) && rand.Float32() < 0.3 {
			newPos := entity.Position{
				X: c.GetPosition().X + nav.WanderBias.X,
				Y: c.GetPosition().Y + nav.WanderBias.Y,
			}
			if tile, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y}); err == nil && len(tile.EntitiesID) == 0 && !tile.Modifier.Obstructed {
				return nav.WanderBias
			}
		}
		return directions[rand.Intn(len(directions))]

	case creature.NavPatrol:
		if len(nav.PatrolRoute) == 0 {
			return s.getNavigationDirection(creature.NavigationLogic{Type: creature.NavWander}, c, world, grid)
		}
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
		playerPos := world.playerPosition
		current := c.GetPosition()
		return entity.Position{
			X: sign(playerPos.X - current.X),
			Y: sign(playerPos.Y - current.Y),
		}

	case creature.NavRepulsion:
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
	// Helper pour vérifier si une case est "traversable" (vide ou piège)
	isWalkable := func(pos entity.Position) bool {
		tile, err := grid.Get(board.Position{X: pos.X, Y: pos.Y})
		if err != nil || tile.Modifier.Obstructed {
			return false
		}
		if len(tile.EntitiesID) == 0 {
			return true
		}
		// On peut marcher si le sommet est un piège
		topID := tile.EntitiesID[len(tile.EntitiesID)-1]
		if ent, ok := world.Entities.Get(entity.ID(topID)); ok {
			return ent.GetType() == entity.TypeTrap
		}
		return false
	}

	canMove := isWalkable(newPos)

	switch coll.Type {
	case creature.CollideStop:
		if !canMove {
			return currentPos, false
		}
		return newPos, true

	case creature.CollideBounce:
		if !canMove {
			c.MovementProfile.Orientation.Rotate(180)
			return currentPos, false
		}
		return newPos, true

	case creature.CollideSlide:
		if canMove {
			return newPos, true
		}

		dx := newPos.X - currentPos.X
		dy := newPos.Y - currentPos.Y

		// Tentative de glissade latérale
		if dy != 0 {
			slidePos := entity.Position{X: currentPos.X, Y: newPos.Y}
			if isWalkable(slidePos) {
				return slidePos, true
			}
		}
		if dx != 0 {
			slidePos := entity.Position{X: newPos.X, Y: currentPos.Y}
			if isWalkable(slidePos) {
				return slidePos, true
			}
		}
		return currentPos, false

	case creature.CollidePhase:
		return newPos, true
	}

	return newPos, true
}

func (s *CreatureMovementSystem) applyMoveMode(mode creature.MovementMode, c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid) bool {
	if mode.Type == creature.ModeSwap {
		tile, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			swappedEntity, ok := world.Entities.Get(entity.ID(topID))
			if ok {
				idStr := string(c.GetID())
				grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)
				grid.RemoveEntity(board.Position{X: newPos.X, Y: newPos.Y}, topID)
				grid.PlaceEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, topID)
				grid.PlaceEntity(board.Position{X: newPos.X, Y: newPos.Y}, idStr)
				swappedEntity.SetPosition(oldPos)
				c.SetPosition(newPos)
				world.Entities.UpdatePosition(swappedEntity.GetID(), oldPos)
				world.Entities.UpdatePosition(c.GetID(), newPos)

				return true
			}
		}
	}

	// Dans tous les autres cas (ou si c'était du sol nu), on fait un déplacement normal (doMove)
	silent := mode.Type == creature.ModeShadow || mode.Type == creature.ModeUnder
	return s.doMove(c, oldPos, newPos, world, grid, silent)
}

func (s *CreatureMovementSystem) doMove(c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid, silent bool) bool {
	idStr := string(c.GetID())

	grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)

	newTile, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
	if len(newTile.EntitiesID) > 0 {
		topID := newTile.EntitiesID[len(newTile.EntitiesID)-1]
		if ent, ok := world.Entities.Get(entity.ID(topID)); ok && ent.GetType() == entity.TypeTrap {
			world.RemoveEntity(ent.GetID())
		}
	}

	grid.PlaceEntity(board.Position{X: newPos.X, Y: newPos.Y}, idStr)

	world.Entities.UpdatePosition(c.GetID(), newPos)

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

	return len(tile.EntitiesID) == 0 && !tile.Modifier.Obstructed
}

func (wa *worldAdapter) GetTileState(pos entity.Position) string {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return "invalid"
	}

	if len(tile.EntitiesID) == 0 {
		return "empty"
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	if ent, ok := wa.world.Entities.Get(entity.ID(topID)); ok {
		return ent.GetState().String()
	}
	return "unknown"
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
	for _, gridID := range world.GridOrder {
		grid, ok := world.GetGrid(gridID)
		if !ok {
			continue
		}

		// CORRECTION : Utilisation de .Plots au lieu de .Tiles
		for _, tile := range grid.Plots {
			if tile.StructureID == "" {
				continue
			}

			comp, ok := world.Components.Get(tile.StructureID, "trigger")
			if !ok {
				continue
			}

			trigger := comp.(*component.Trigger)
			if s.checkCondition(trigger.Condition, tile, world, grid) {
				s.executeAction(trigger.Action, tile, world, grid)
				if trigger.Consumed {
					world.Components.Remove(tile.StructureID, "trigger")
				}
			}
		}
	}
}

func (s *TriggerSystem) checkCondition(condition string, tile *board.Plot, world *World, grid *board.Grid) bool {
	if len(tile.EntitiesID) == 0 {
		return false
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	topEnt, ok := world.Entities.Get(entity.ID(topID))

	switch condition {
	case "reveal_with_creature":
		return ok && topEnt.GetType() == entity.TypeCreature && topEnt.GetState() == entity.Revealed

	case "creature_on_resource":
		if !ok || topEnt.GetType() != entity.TypeCreature {
			return false
		}

		neighbors := grid.GetNeighbors(tile.Position)
		for _, n := range neighbors {
			for _, id := range n.EntitiesID {
				if res, ok := world.Entities.Get(entity.ID(id)); ok {
					if res.GetType() == entity.TypeResource {
						return true
					}
				}
			}
		}
	}
	return false
}

func (s *TriggerSystem) executeAction(action string, tile *board.Plot, world *World, grid *board.Grid) {
	switch action {
	case "creature_flee":
		// On fait fuir la créature qui est au SOMMET de la pile (celle qui a marché sur le trigger)
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := world.Entities.Get(entity.ID(topID)); ok {
				if c, ok := e.(*creature.Creature); ok {
					c.Behavior.State = "fleeing"
				}
			}
		}

	case "reveal_adjacent":
		neighbors := grid.GetNeighbors(tile.Position)
		for _, n := range neighbors {
			// REVELATION : On révèle TOUTES les entités présentes sur les cases voisines
			// (Parce qu'un flash lumineux ou un bruit révèle tout ce qui est caché dans la pile)
			for _, id := range n.EntitiesID {
				if e, ok := world.Entities.Get(entity.ID(id)); ok {
					if e.GetState() == entity.Hidden {
						e.SetState(entity.Revealed)
					}
				}
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

	for i := 0; i < len(e.systems)-1; i++ {
		for j := i + 1; j < len(e.systems); j++ {
			if e.systems[i].Priority() > e.systems[j].Priority() {
				e.systems[i], e.systems[j] = e.systems[j], e.systems[i]
			}
		}
	}

	if e.movementSystem != nil {
		e.movementSystem.TrackReveal(board.Position{}) // Utilisation factice pour correspondre à l'ancienne signature si nécessaire
		e.movementSystem.ClearReveals()
	}

	for _, sys := range e.systems {
		sys.Update(e.world)
	}

	e.world.EventBus.ProcessQueue()
	e.world.Turn++
	e.world.EventBus.Publish(event.NewTurnEndedEvent(e.world.Turn))
}

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
