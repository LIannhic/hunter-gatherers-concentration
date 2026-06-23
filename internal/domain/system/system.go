package system

import (
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
)

// =============================================================================
// SYSTEMS ECS (LOGIQUE MÉTIER REGROUPÉE)
// =============================================================================

// --- SYSTEM: LIFECYCLE ---
type LifecycleSystem struct{}

func (s *LifecycleSystem) Priority() int { return 1 }

func (s *LifecycleSystem) Update(world *World) {
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

		if lifecycle.Progress() {
			stageName := lifecycle.GetCurrentStageName()
			ent, _ := world.Entities.Get(entity.ID(entityID))
			entType := "unknown"
			if ent != nil {
				entType = ent.GetType().String()
			}
			fmt.Printf("[LIFECYCLE] Entité %s (%s) a mûri au stade %d: %s\n", entityID, entType, lifecycle.CurrentStage, stageName)
			world.EventBus.Publish(event.NewResourceMaturedEvent(
				entityID,
				stageName,
			))
		}
	}
}

// --- SYSTEM: PROPAGATION ---
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

		if lifecycle.MaxPropagations != -1 && lifecycle.PropagationsDone >= lifecycle.MaxPropagations {
			continue
		}

		if !shouldPropagate(lifecycle) {
			continue
		}

		grid, ok := world.Grids[e.GetGridID()]
		if !ok {
			continue
		}

		pos := e.GetPosition()
		allNeighbors := grid.GetNeighbors(board.Position{X: pos.X, Y: pos.Y})

		var neighbors []*board.Plot
		for _, n := range allNeighbors {
			if abs(n.Position.X-pos.X)+abs(n.Position.Y-pos.Y) == 1 {
				neighbors = append(neighbors, n)
			}
		}

		rand.Shuffle(len(neighbors), func(i, j int) { neighbors[i], neighbors[j] = neighbors[j], neighbors[i] })

		maxToPropagate := lifecycle.PropagationCount
		if maxToPropagate <= 0 {
			maxToPropagate = 1
		}

		var validNeighbors []*board.Plot
		for _, neighbor := range neighbors {
			if world.HasResourceAt(e.GetGridID(), neighbor.Position) {
				continue
			}
			if neighbor.Modifier.Obstructed {
				continue
			}

			validNeighbors = append(validNeighbors, neighbor)
			if len(validNeighbors) == maxToPropagate {
				break
			}
		}

		if len(validNeighbors) < maxToPropagate {
			continue
		}

		propagatedCount := 0
		for _, targetNeighbor := range validNeighbors {
			spawnPos := entity.Position{
				X: targetNeighbor.Position.X,
				Y: targetNeighbor.Position.Y,
			}

			newRes, err := world.SpawnResource(e.GetGridID(), getResourceType(e), spawnPos)
			if err != nil {
				continue
			}

			if rand.Float32() < 0.99 {
				if comp, ok := world.Components.Get(string(newRes.GetID()), "lifecycle"); ok {
					if lc, ok := comp.(*component.Lifecycle); ok {
						lc.CanPropagate = false
						fmt.Printf("[PROPA] Une nouvelle %s est née stérile à %v\n", getResourceType(e), spawnPos)
					}
				}
			}

			if lifecycle.PropagationLevel != 0 {
				grid.RemoveEntity(targetNeighbor.Position, string(newRes.GetID()))
				grid.PlaceEntityAtBottom(targetNeighbor.Position, string(newRes.GetID()))
			}

			propagatedCount++

			world.EventBus.Publish(event.Event{
				Type:     event.ResourcePropagated,
				SourceID: string(newRes.GetID()),
				Payload: map[string]interface{}{
					"parent_id":     entityID,
					"from":          pos,
					"to":            spawnPos,
					"new_entity_id": string(newRes.GetID()),
				},
			})
		}

		if propagatedCount > 0 {
			lifecycle.TurnsInStage = 0
			lifecycle.PropagationsDone++
		}
	}
}

func shouldPropagate(l *component.Lifecycle) bool {
	isLastStage := l.CurrentStage == l.MaxStages-1
	return isLastStage && l.TurnsInStage >= l.TurnsToNext
}

func getResourceType(e entity.Entity) string {
	if r, ok := e.(*resource.Resource); ok {
		return r.ResourceType
	}
	return "unknown"
}

// --- SYSTEM: TRACK ---
type TrackSystem struct{}

func (s *TrackSystem) Priority() int { return 5 }

func (s *TrackSystem) Update(world *World) {
	tracks := world.Entities.GetByType(entity.TypeTrack)

	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok {
			continue
		}

		t.Duration--
		if t.Duration <= 0 {
			world.RemoveEntity(t.GetID())
		}
	}
}

// --- SYSTEM: TRIGGER ---
type TriggerSystem struct{}

func (s *TriggerSystem) Priority() int { return 4 }

func (s *TriggerSystem) Update(world *World) {
	for _, gridID := range world.GridOrder {
		grid, ok := world.GetGrid(gridID)
		if !ok {
			continue
		}

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
			for _, id := range n.EntitiesID {
				if e, ok := world.Entities.Get(entity.ID(id)); ok {
					if e.GetState()&entity.Hidden != 0 {
						e.SetState(entity.Revealed)
					}
				}
			}
		}
	}
}

// --- SYSTEM: PREVIEW ---
type PreviewSystem struct {
	previewTimers map[string]int
	previewed     map[string]bool
}

func NewPreviewSystem() *PreviewSystem {
	return &PreviewSystem{
		previewTimers: make(map[string]int),
		previewed:     make(map[string]bool),
	}
}

func (s *PreviewSystem) Reset() {
	s.previewTimers = make(map[string]int)
	s.previewed = make(map[string]bool)
}

func (s *PreviewSystem) Priority() int { return 0 }

func (s *PreviewSystem) IsPreviewActive(gridID string) bool {
	return s.previewTimers[gridID] > 0
}

func (s *PreviewSystem) Update(world *World) {
	for gridID, timer := range s.previewTimers {
		if timer > 0 {
			s.previewTimers[gridID]--
			if s.previewTimers[gridID] == 0 {
				s.hideGrid(world, gridID)
			}
		}
	}
}

func (s *PreviewSystem) OnEnterGrid(world *World, gridID string) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	if s.previewed[gridID] {
		return
	}

	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)
	if isPortalZone {
		if gridID == world.DreamPlane.StartZoneID {
			s.previewTimers[gridID] = 2 * 60
			s.previewed[gridID] = true
		}
		return
	}

	settings := world.Difficulty
	if world.Debug.OverrideDifficulty {
		settings = world.Debug.Difficulty
	}

	if settings.PreviewDuration <= 0 {
		return
	}

	fmt.Printf("[PREVIEW] Première entrée sur %s (Difficulté: %s)\n", gridID, settings.Level)

	for _, tile := range grid.Plots {
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := world.Entities.Get(entity.ID(topID)); ok {
				if e.GetState()&entity.Hidden != 0 {
					e.SetState(entity.Revealed)
				}
			}
		}
	}

	s.previewed[gridID] = true
	s.previewTimers[gridID] = int(settings.PreviewDuration * 60)
}

func (s *PreviewSystem) hideGrid(world *World, gridID string) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	fmt.Printf("[PREVIEW] Fin du délai sur %s, masquage des tuiles.\n", gridID)
	for _, tile := range grid.Plots {
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := world.Entities.Get(entity.ID(topID)); ok {
				if e.GetType() == entity.TypeStructure {
					if e.HasTag("start_portal") {
						flipDir := tile.Tilt.ToFlipDirection()
						_, _ = world.FlipTile(gridID, tile.Position, flipDir, "system_hide")
						e.SetState(entity.Hidden | entity.Blocked)

						world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
							e.GetPosition(), string(e.GetID()), gridID, flipDir,
							map[string]interface{}{"reason": "system_hide"}))
						continue
					}
					continue
				}

				if e.GetState()&entity.Revealed != 0 {
					_, _ = world.FlipTile(gridID, tile.Position, tile.Tilt.ToFlipDirection(), "system_hide")

					world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
						e.GetPosition(), string(e.GetID()), gridID, tile.Tilt.ToFlipDirection(),
						map[string]interface{}{"reason": "system_hide"}))
				}
			}
		}
	}
}

// --- SYSTEM: LOOT ---
type LootSystem struct {
	world *World
}

func NewLootSystem(world *World) *LootSystem {
	ls := &LootSystem{world: world}
	world.EventBus.SubscribeFunc(event.TileMatched, ls.onTileMatched)
	return ls
}

func (s *LootSystem) Priority() int { return 10 }

func (s *LootSystem) Update(world *World) {}

func (s *LootSystem) onTileMatched(e event.Event) {
	entID := entity.ID(e.SourceID)
	name := "unknown"
	var eType entity.Type = entity.TypeResource

	if ent, ok := s.world.Entities.Get(entID); ok {
		name = s.getEntityName(ent)
		eType = ent.GetType()
	}

	if name == "unknown" || name == "" {
		if n, exists := e.Payload["name"].(string); exists {
			name = n
		}
		if t, exists := e.Payload["entity_type"].(entity.Type); exists {
			eType = t
		}
	}

	if name == "unknown" || name == "" || name == "trap" {
		return
	}

	sourceID := name
	loot := &player.LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         name,
		SourceID:     sourceID,
		OriginalType: eType,
		IsUsable:     true,
		IsDeletable:  true,
	}

	level, _ := e.Payload["level"].(int)
	loot.SetCumulationLevel(level)
	if level > 0 {
		loot.AddTag(fmt.Sprintf("level_%d", level))
	}

	loot.AddTag(fmt.Sprintf("original_type_%d", eType))
	loot.AddTag(name)

	err := s.world.AddLootItem(loot)
	if err != nil {
		fmt.Printf("[LOOT] Inventaire plein ! Le loot %s est perdu.\n", name)
		s.world.EventBus.PublishImmediate(event.NewInventoryFullEvent())
		return
	}

	fmt.Printf("[LOOT] Acquisition : %s (ID: %s)\n", name, entID)
	s.world.EventBus.PublishImmediate(event.NewLootAcquiredEvent(string(loot.GetID()), loot.Name, loot.OriginalType))
}

func (s *LootSystem) getEntityName(ent entity.Entity) string {
	if r, ok := ent.(*resource.Resource); ok {
		return r.ResourceType
	}
	if c, ok := ent.(*creature.Creature); ok {
		return c.Species
	}
	return "unknown_loot"
}

// --- LOGIQUE DE SCANNER (ECHO HOUND) ---
func (w *World) TriggerScannerEffect(gridID string, level int) error {
	_, ok := w.GetGrid(gridID)
	if !ok {
		return fmt.Errorf("grille introuvable")
	}

	fmt.Printf("[WORLD] L'Echo Hound hurle sur la zone %s (Niveau %d) !\n", gridID, level)

	scannedPositions := make([]board.Position, 0)
	for _, e := range w.Entities.GetAllActive() {
		if e.GetGridID() == gridID && e.GetState()&entity.Hidden != 0 {
			pos := e.GetPosition()
			scannedPositions = append(scannedPositions, board.Position{X: pos.X, Y: pos.Y})
		}
	}

	duration := 2.0 * math.Pow(2.2, float64(level))

	w.EventBus.PublishImmediate(event.Event{
		Type:     event.Type("scanner_triggered"),
		SourceID: "echo_hound",
		Payload: map[string]interface{}{
			"grid_id":   gridID,
			"positions": scannedPositions,
			"duration":  duration,
		},
	})

	return nil
}

// --- SYSTEM: CREATURE AI ---
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

		grid, ok := world.Grids[c.GetGridID()]
		if !ok {
			continue
		}

		action := ai.Decide(c, &worldAdapter{world: world, grid: grid})

		switch action.Type {
		case "move":
			if c.MovementProfile != nil {
				continue
			}

			oldPos := c.GetPosition()
			newPos := entity.Position{
				X: oldPos.X + action.Direction.X,
				Y: oldPos.Y + action.Direction.Y,
			}

			if !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
				continue
			}

			world.Engine.TrackTileReveal(board.Position{X: oldPos.X, Y: oldPos.Y}, grid.ID)
			world.Engine.TrackTileReveal(board.Position{X: newPos.X, Y: newPos.Y}, grid.ID)

			newPlot, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
			idStr := string(c.GetID())
			_, err := grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)
			if err != nil {
				continue
			}

			if len(newPlot.EntitiesID) > 0 {
				topID := newPlot.EntitiesID[len(newPlot.EntitiesID)-1]
				if _, ok := world.Entities.Get(entity.ID(topID)); ok {
					// On ne retire plus les pièges, on les ignore simplement (empilement possible)
				}
			}

			newPlot.PushEntity(idStr)
			world.Entities.UpdatePosition(c.GetID(), newPos)

			world.EventBus.Publish(event.NewCreatureMovedEvent(
				idStr, oldPos, newPos, "manifest", false, false,
			))

		case "spawn_trap":
			trap, err := world.SpawnTrap(c.GetGridID(), c.GetPosition())
			if err == nil {
				if grid, ok := world.GetGrid(c.GetGridID()); ok {
					pos := board.Position(c.GetPosition())
					id := string(trap.GetID())
					grid.RemoveEntity(pos, id)
					if plot, err := grid.Get(pos); err == nil {
						// On insère le piège à l'index 0 (tout en bas de la pile)
						plot.EntitiesID = append([]string{id}, plot.EntitiesID...)
					}
				}
				fmt.Printf("[ACTION] %s a posé un piège à %v\n", c.Species, c.GetPosition())
			}

		case "flee":
			fmt.Printf("[ACTION] %s fuit la zone car le plateau est plein !\n", c.Species)
			world.EventBus.Publish(event.NewCreatureFledEvent(
				string(c.GetID()), c.Species, c.GetGridID(), c.GetPosition(),
			))
			world.RemoveEntity(c.GetID())

		case "transform":
			targetID := action.TargetID
			if targetID != "" {
				if comp, ok := world.Components.Get(targetID, "lifecycle"); ok {
					if lifecycle, ok := comp.(*component.Lifecycle); ok {
						lifecycle.CurrentStage++
					}
				}
			}
		}
	}
}

// --- SYSTEM: CREATURE MOVEMENT (ADVANCED) ---
type RevealedTile struct {
	Position board.Position
	GridID   string
}

type CreatureMovementSystem struct {
	world         *World
	recentReveals []RevealedTile
}

func NewCreatureMovementSystem(world *World) *CreatureMovementSystem {
	s := &CreatureMovementSystem{
		world:         world,
		recentReveals: make([]RevealedTile, 0),
	}
	world.EventBus.SubscribeFunc(event.TileRevealed, s.onTileRevealed)
	return s
}

func (s *CreatureMovementSystem) onTileRevealed(e event.Event) {
	reason, _ := e.Payload["reason"].(string)
	if reason != "player_action" {
		return
	}
	pos := entity.Position{}
	if p, ok := e.Payload["position"].(entity.Position); ok {
		pos = p
	}
	gridID := ""
	if g, ok := e.Payload["grid_id"].(string); ok {
		gridID = g
	}
	s.TrackReveal(board.Position{X: pos.X, Y: pos.Y}, gridID)
}

func (s *CreatureMovementSystem) Priority() int { return 3 }

func (s *CreatureMovementSystem) TrackReveal(pos board.Position, gridID string) {
	s.recentReveals = append(s.recentReveals, RevealedTile{Position: pos, GridID: gridID})
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

		grid, ok := world.GetGrid(c.GetGridID())
		if !ok {
			continue
		}

		profile := c.MovementProfile
		if !s.shouldTrigger(profile.Trigger, c) {
			if s.shouldWander(profile, c, world) && !profile.Frequency.HasMovedThisTurn(world.Turn) && profile.Frequency.CanMove() {
				s.executeWander(c, profile, world, grid)
				profile.Frequency.MarkMoved(world.Turn)
			}
			continue
		}

		if profile.Frequency.HasMovedThisTurn(world.Turn) {
			continue
		}

		if !profile.Frequency.CanMove() {
			continue
		}

		moveCount := profile.Frequency.GetMoveCount()
		moved := false
		for i := 0; i < moveCount; i++ {
			if !s.executeMove(c, profile, world, grid) {
				break
			}
			moved = true
		}

		if moved {
			profile.Frequency.MarkMoved(world.Turn)
		}

		if profile.Trigger.Type == creature.TriggerOnReveal {
			switch c.Species {
			case "stonewarden":
				profile.Trigger.Type = creature.TriggerAuto
				profile.Navigation.Type = creature.NavOrientation
			}
		}

		profile.Trigger.Reset()
	}
}

func (s *CreatureMovementSystem) shouldTrigger(trigger creature.MovementTrigger, c *creature.Creature) bool {
	creatureGridID := c.GetGridID()
	switch trigger.Type {
	case creature.TriggerPassive:
		return false
	case creature.TriggerAuto:
		return true
	case creature.TriggerOnReveal:
		for _, revealed := range s.recentReveals {
			if revealed.GridID == creatureGridID &&
				revealed.Position.X == c.GetPosition().X &&
				revealed.Position.Y == c.GetPosition().Y {
				if !trigger.Triggered {
					trigger.Triggered = true
					return true
				}
			}
		}
		return false
	case creature.TriggerOnEcho:
		for _, revealed := range s.recentReveals {
			if revealed.GridID == creatureGridID {
				return true
			}
		}
		return false
	case creature.TriggerProximity:
		for _, revealed := range s.recentReveals {
			if revealed.GridID != creatureGridID {
				continue
			}
			dist := abs(revealed.Position.X-c.GetPosition().X) + abs(revealed.Position.Y-c.GetPosition().Y)
			if dist <= trigger.Radius {
				return true
			}
		}
		return false
	}
	return false
}

// shouldWander vérifie si une créature devrait errer en fallback
// quand le trigger ne se déclenche pas et que la cible n'existe pas.
func (s *CreatureMovementSystem) shouldWander(profile *creature.MovementProfile, c *creature.Creature, world *World) bool {
	switch profile.Navigation.Type {
	case creature.NavAttraction, creature.NavRepulsion:
		return !s.hasTarget(profile, c, world)
	default:
		return false
	}
}

// hasTarget vérifie si la cible de navigation de la créature existe sur la grille.
func (s *CreatureMovementSystem) hasTarget(profile *creature.MovementProfile, c *creature.Creature, world *World) bool {
	grid, ok := world.Grids[c.GetGridID()]
	if !ok {
		return false
	}
	wa := &worldAdapter{world: world, grid: grid}
	target := wa.FindNearestTarget(c.GetPosition(), profile.Navigation.Target, profile.Navigation.TargetName, profile.Navigation.ExcludedStages)
	return target != nil
}

// executeWander déplace une créature dans une direction aléatoire valide.
func (s *CreatureMovementSystem) executeWander(c *creature.Creature, profile *creature.MovementProfile, world *World, grid *board.Grid) {
	directions := []entity.Position{
		{X: 0, Y: -1}, {X: 0, Y: 1},
		{X: -1, Y: 0}, {X: 1, Y: 0},
	}
	rand.Shuffle(len(directions), func(i, j int) {
		directions[i], directions[j] = directions[j], directions[i]
	})

	currentPos := c.GetPosition()
	for _, dir := range directions {
		newPos := entity.Position{X: currentPos.X + dir.X, Y: currentPos.Y + dir.Y}
		wa := &worldAdapter{world: world, grid: grid}
		if wa.IsWalkable(c, newPos) {
			s.applyMoveMode(profile.Mode, c, currentPos, newPos, world, grid)
			return
		}
	}
}

func sign(val int) int {
	if val < 0 {
		return -1
	}
	if val > 0 {
		return 1
	}
	return 1
}

func (s *CreatureMovementSystem) executeMove(c *creature.Creature, profile *creature.MovementProfile, world *World, grid *board.Grid) bool {
	wa := &worldAdapter{world: world, grid: grid}
	direction := profile.Navigation.DecideDirection(wa, c)
	if direction == (entity.Position{X: 0, Y: 0}) {
		return true
	}

	currentPos := c.GetPosition()
	newPos := entity.Position{
		X: currentPos.X + direction.X,
		Y: currentPos.Y + direction.Y,
	}

	if !wa.IsWalkable(c, newPos) {
		if profile.Navigation.Type == creature.NavOrientation {
			dir := direction
			hitH := !wa.IsWalkable(c, entity.Position{X: currentPos.X + dir.X, Y: currentPos.Y})
			hitV := !wa.IsWalkable(c, entity.Position{X: currentPos.X, Y: currentPos.Y + dir.Y})

			rotateDegrees := 90
			if hitH && hitV {
				rotateDegrees = 180
			} else if hitV {
				rotateDegrees = -90 * sign(dir.X) * sign(dir.Y)
			} else if hitH {
				rotateDegrees = 90 * sign(dir.X) * sign(dir.Y)
			}

			newOrient := entity.RotateDirection(c.GetOrientation(), rotateDegrees)
			c.SetOrientation(newOrient)

			world.Components.Add(string(c.GetID()), &component.RotationAnimation{
				CurrentAngle:  0,
				TargetAngle:   float64(rotateDegrees),
				DurationTicks: 15,
				CurrentTick:   0,
			})

			world.EventBus.PublishImmediate(event.NewAnimationStartedEvent("rotate", string(c.GetID()), map[string]interface{}{
				"angle_degrees": rotateDegrees,
			}))
			return false
		}

		finalPos, success := profile.Collision.HandleCollision(wa, c, newPos)
		if !success {
			return false
		}
		newPos = finalPos
	}

	if profile.Navigation.Type != creature.NavRelative && profile.Navigation.Type != creature.NavOrientation {
		profile.Orientation.Direction = directionToOrientation(direction).Direction
		c.BaseEntity.SetOrientation(profile.Orientation.Direction)
	}

	return s.applyMoveMode(profile.Mode, c, currentPos, newPos, world, grid)
}


func (s *CreatureMovementSystem) MoveSpeciesOneStepTowards(species string, target entity.Position, world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok || c.Species != species {
			continue
		}
		grid, ok := world.GetGrid(c.GetGridID())
		if !ok {
			continue
		}

		if c.MovementProfile != nil && c.MovementProfile.Frequency.HasMovedThisTurn(world.Turn) {
			continue
		}

		currentPos := c.GetPosition()
		dir := entity.Position{
			X: entity.Sign(target.X - currentPos.X),
			Y: entity.Sign(target.Y - currentPos.Y),
		}
		if dir == (entity.Position{X: 0, Y: 0}) {
			continue
		}

		newPos := entity.Position{X: currentPos.X + dir.X, Y: currentPos.Y + dir.Y}

		coll := creature.CollisionHandler{Type: creature.CollideStop}
		if c.MovementProfile != nil {
			coll = c.MovementProfile.Collision
			if c.MovementProfile.Mode.Type == creature.ModeSwap {
				coll.Type = creature.CollidePhase
				coll.CanPhaseThrough = []string{"ground", "wall", "structure"}
			}
		}

		finalPos, success := s.handleCollision(coll, c, newPos, currentPos, world, grid)
		if !success {
			continue
		}

		mode := creature.MovementMode{Type: creature.ModeNormal}
		if c.MovementProfile != nil {
			mode = c.MovementProfile.Mode
		}
		moved := s.applyMoveMode(mode, c, currentPos, finalPos, world, grid)
		if moved && c.MovementProfile != nil {
			c.MovementProfile.Frequency.MarkMoved(world.Turn)
		}
	}
}

func (s *CreatureMovementSystem) isWalkable(c *creature.Creature, pos entity.Position, grid *board.Grid, world *World) bool {
	wa := &worldAdapter{world: world, grid: grid}
	return wa.IsWalkable(c, pos)
}

func (s *CreatureMovementSystem) handleCollision(coll creature.CollisionHandler, c *creature.Creature, newPos, currentPos entity.Position, world *World, grid *board.Grid) (entity.Position, bool) {
	if c.MovementProfile != nil && c.MovementProfile.Mode.Type == creature.ModeSwap {
		if grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
			return newPos, true
		}
	}

	wa := &worldAdapter{world: world, grid: grid}
	return coll.HandleCollision(wa, c, newPos)
}

func (s *CreatureMovementSystem) applyMoveMode(mode creature.MovementMode, c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid) bool {
	if mode.Type == creature.ModeSwap {
		tile, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
		if err == nil && len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			swappedEntity, ok := world.Entities.Get(entity.ID(topID))
			if ok {
				idStr := string(c.GetID())
				grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)
				grid.RemoveEntity(board.Position{X: newPos.X, Y: newPos.Y}, topID)

				// Placement direct via slice manipulation (Swap: on garde l'ordre existant mais on change les positions logiques)
				if plotOld, err := grid.Get(board.Position{X: oldPos.X, Y: oldPos.Y}); err == nil {
					plotOld.EntitiesID = append(plotOld.EntitiesID, topID)
				}
				if plotNew, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y}); err == nil {
					plotNew.EntitiesID = append(plotNew.EntitiesID, idStr)
				}

			swappedEntity.SetPosition(oldPos)
			c.SetPosition(newPos)
			world.Entities.UpdatePosition(swappedEntity.GetID(), oldPos)
			world.Entities.UpdatePosition(c.GetID(), newPos)

			isCloaked := false
			if c.MovementProfile != nil {
				isCloaked = c.MovementProfile.Perception.Stealth == creature.StealthCloaked
			}
			world.EventBus.Publish(event.NewCreatureMovedEvent(
				idStr, oldPos, newPos, "swap", isCloaked, false,
			))

			otherHidden := false
			if otherCreature, ok := swappedEntity.(*creature.Creature); ok && otherCreature.MovementProfile != nil {
				otherHidden = otherCreature.MovementProfile.Perception.Stealth == creature.StealthCloaked
			}
			world.EventBus.Publish(event.NewCreatureMovedEvent(
				topID, newPos, oldPos, "swap_under", otherHidden, false,
			))

			return true
			}
		}
	}

	return s.doMove(c, oldPos, newPos, world, grid)
}

func (s *CreatureMovementSystem) doMove(c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid) bool {
	if grid == nil || !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
		return false
	}

	idStr := string(c.GetID())
	grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)

	mode := creature.ModeNormal
	if c.MovementProfile != nil {
		mode = c.MovementProfile.Mode.Type
	}

	plot, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
	if err != nil {
		return false
	}

	// On ne supprime plus les pièges, on les ignore (Z-sorting via slice manipulation)
	if mode == creature.ModeUnder {
		// Prepend (Au tout début du tableau pour être en dessous)
		plot.EntitiesID = append([]string{idStr}, plot.EntitiesID...)
	} else {
		// Append (A la fin du tableau pour être au sommet)
		plot.EntitiesID = append(plot.EntitiesID, idStr)
	}

	world.Entities.UpdatePosition(c.GetID(), newPos)

	isCloaked := false
	isAudible := false

	if c.MovementProfile != nil {
		isCloaked = c.MovementProfile.Perception.Stealth == creature.StealthCloaked
		isAudible = c.MovementProfile.Perception.Acoustic == creature.AcousticEcho
	}

	if c.MovementProfile != nil && c.MovementProfile.Perception.LeavesTracks {
		pProfile := c.MovementProfile.Perception
		trackEnt := entity.NewTrack(pProfile.TrackType, pProfile.TrackDuration, oldPos, newPos)
		trackEnt.SetGridID(c.GetGridID())
		world.Entities.Register(trackEnt)
	}

	world.EventBus.Publish(event.NewCreatureMovedEvent(
		string(c.GetID()), oldPos, newPos, string(mode), isCloaked, isAudible,
	))
	return true
}

func directionToOrientation(dir entity.Position) creature.Orientation {
	if dir.X > 0 {
		return creature.Orientation{Direction: creature.Right}
	}
	if dir.X < 0 {
		return creature.Orientation{Direction: creature.Left}
	}
	if dir.Y > 0 {
		return creature.Orientation{Direction: creature.Backward}
	}
	return creature.Orientation{Direction: creature.Forward}
}

// --- WORLD ADAPTER ---
type worldAdapter struct {
	world *World
	grid  *board.Grid
}

func (wa *worldAdapter) GetPlayerPosition() entity.Position {
	return wa.world.playerPosition
}

func (wa *worldAdapter) IsPlayerOnBoard() bool {
	return wa.world.IsPlayerOnBoard()
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

func (wa *worldAdapter) GetEmptyPlots() []entity.Position {
	var empty []entity.Position
	for pos, plot := range wa.grid.Plots {
		if len(plot.EntitiesID) == 0 {
			empty = append(empty, entity.Position{X: pos.X, Y: pos.Y})
		}
	}
	return empty
}

func (wa *worldAdapter) GetGridTotalPlots() int {
	return wa.grid.Width * wa.grid.Height
}

func (wa *worldAdapter) IsTileRevealed(pos entity.Position) bool {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil || len(tile.EntitiesID) == 0 {
		return false
	}
	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	if ent, ok := wa.world.Entities.Get(entity.ID(topID)); ok {
		return ent.GetState()&entity.Revealed != 0
	}
	return false
}

func (wa *worldAdapter) WasTileRecentlyRevealed(pos entity.Position) bool {
	if wa.world.Engine == nil || wa.world.Engine.movementSystem == nil {
		return false
	}
	gridID := wa.grid.ID
	for _, revealed := range wa.world.Engine.movementSystem.recentReveals {
		if revealed.GridID == gridID &&
			revealed.Position.X == pos.X &&
			revealed.Position.Y == pos.Y {
			return true
		}
	}
	return false
}

func (wa *worldAdapter) FindNearestTarget(from entity.Position, targetType creature.TargetType, targetName string, excludedStages []int) *entity.Position {
	var nearest *entity.Position
	minDist := 9999

	switch targetType {
	case creature.TargetPlayer:
		// Seule l'attraction/répulsion du joueur sur sa propre grille est possible
		if wa.world.CurrentGridID != wa.grid.ID {
			return nil
		}
		if !wa.world.IsPlayerOnBoard() {
			return nil
		}
		pos := wa.world.playerPosition
		nearest = &pos
	case creature.TargetResource:
		resources := wa.world.Entities.GetByType(entity.TypeResource)
		for _, e := range resources {
			if e.GetGridID() != wa.grid.ID {
				continue
			}

			// Filtrage par nom si spécifié
			idStr := string(e.GetID())
			if targetName != "" {
				if r, ok := e.(*resource.Resource); ok {
					if r.ResourceType != targetName {
						continue
					}
				}
			}

			// Filtrage par stade de cycle de vie
			if len(excludedStages) > 0 {
				if comp, ok := wa.world.Components.Get(idStr, "lifecycle"); ok {
					if lc, ok := comp.(*component.Lifecycle); ok {
						isExcluded := false
						for _, stage := range excludedStages {
							if lc.CurrentStage == stage {
								isExcluded = true
								break
							}
						}
						if isExcluded {
							continue
						}
					}
				}
			}

			pos := e.GetPosition()
			d := abs(pos.X-from.X) + abs(pos.Y-from.Y)
			if d < minDist {
				minDist = d
				nearest = &pos
			}
		}
	}
	return nearest
}

func (wa *worldAdapter) GetTileType(pos entity.Position) string {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return "invalid"
	}
	if tile.Modifier.Obstructed {
		return "wall"
	}
	return "ground"
}

func (wa *worldAdapter) GetEntitiesAt(pos entity.Position) []entity.Entity {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return nil
	}
	var result []entity.Entity
	for _, id := range tile.EntitiesID {
		if e, ok := wa.world.Entities.Get(entity.ID(id)); ok {
			result = append(result, e)
		}
	}
	return result
}

func (wa *worldAdapter) IsWalkable(c *creature.Creature, pos entity.Position) bool {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil || tile.Modifier.Obstructed {
		return false
	}

	// 1. Stats de la créature arrivante
	cMob, _ := c.GetComponent("mobility").(*component.Mobility)
	cSize := component.SizeMedium
	cWeight := component.WeightMedium
	if cMob != nil {
		cSize = cMob.Size
		cWeight = cMob.Weight
	}

	creatureCount := 0
	// 2. Vérification de la cohabitation dans la pile EntitiesID de la parcelle
	for _, id := range tile.EntitiesID {
		ent, ok := wa.world.Entities.Get(entity.ID(id))
		if !ok {
			continue
		}

		if other, ok := ent.(*creature.Creature); ok {
			creatureCount++

			// Règle 1 : "Premier arrivé, premier servi" - Pas de cohabitation même espèce
			if other.Species == c.Species {
				return false
			}

			// Règle 2 : "Condition stricte" - Unicité de la taille (Small, Medium, Large)
			otherMob, _ := other.GetComponent("mobility").(*component.Mobility)
			otherSize := component.SizeMedium
			otherWeight := component.WeightMedium
			if otherMob != nil {
				otherSize = otherMob.Size
				otherWeight = otherMob.Weight
			}

			if otherSize == cSize {
				// Règle 3 : "Priorité au poids" - Cohabitation même taille QUE si strictement plus lourd
				if cWeight <= otherWeight {
					return false
				}
			}
		}
	}

	// Règle 4 : 3 créatures MAXIMUM par parcelle
	if creatureCount >= 3 {
		return false
	}

	// Les créatures ignorent les ressources et les pièges (peuvent s'empiler dans EntitiesID)
	return true
}

func (wa *worldAdapter) HasActivityNearby(pos entity.Position, radius int) bool {
	if wa.world.Engine.movementSystem == nil {
		return false
	}

	gridID := wa.grid.ID
	for _, activityPos := range wa.world.Engine.movementSystem.recentReveals {
		if activityPos.GridID != gridID {
			continue
		}
		dist := abs(activityPos.Position.X-pos.X) + abs(activityPos.Position.Y-pos.Y)
		if dist <= radius {
			return true
		}
	}
	return false
}

func (wa *worldAdapter) IsValidMove(pos entity.Position) bool {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return false
	}
	if tile.Modifier.Obstructed {
		return false
	}
	return true
}

func (wa *worldAdapter) GetTileState(pos entity.Position) string {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return "invalid"
	}

	count := len(tile.EntitiesID)
	if count == 0 {
		return "empty"
	}

	if count == 1 {
		return "alone"
	}

	return "occupied"
}

func (wa *worldAdapter) IsGridSaturatedWithTraps() bool {
	for _, plot := range wa.grid.Plots {
		hasTrap := false
		for _, id := range plot.EntitiesID {
			if ent, ok := wa.world.Entities.Get(entity.ID(id)); ok {
				if ent.GetType() == entity.TypeTrap {
					hasTrap = true
					break
				}
			}
		}
		if !hasTrap {
			return false
		}
	}
	return true
}

var ErrGridNotFound = errors.New("grid not found")

func abs(x int) int {
	return entity.Abs(x)
}

// --- SYSTEM: ATTACK INTENT ---
type AttackIntent struct {
	SourcePos entity.Position
	TargetX   float64
	TargetY   float64
	Intensity float64
	IsActive  bool
}

type IntentSystem struct {
	world *World
}

func (s *IntentSystem) UpdateIntent(intent *AttackIntent, cursorPixelX, cursorPixelY float64, tileSize, spacing float64) {
	intent.TargetX = cursorPixelX
	intent.TargetY = cursorPixelY
}

// --- SYSTEM: TOXICITY ---
type ToxicitySystem struct{}

func (s *ToxicitySystem) Priority() int { return 6 }

func (s *ToxicitySystem) Update(world *World) {
	if world.Player == nil {
		return
	}

	if !world.IsPlayerOnBoard() {
		return
	}

	totalDamage := 0.0
	stackCount := 0
	maxDegression := 0.0

	// 1. Collecte des dangers actifs (seulement sur la grille actuelle)
	gridID := world.CurrentGridID
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	for _, tile := range grid.Plots {
			if len(tile.EntitiesID) == 0 {
				continue
			}

			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if _, ok := world.Entities.Get(entity.ID(topID)); !ok {
				continue
			}

			comp, ok := world.Components.Get(topID, "hazard")
			if !ok {
				continue
			}

			hazard := comp.(*component.Hazard)
			if !hazard.IsConstant {
				continue
			}

			// Vérification du stade via Lifecycle
			currentStage := 0
			stageName := "unknown"
			if lcComp, ok := world.Components.Get(topID, "lifecycle"); ok {
				lc := lcComp.(*component.Lifecycle)
				currentStage = lc.CurrentStage
				stageName = lc.GetCurrentStageName()
			}

			if !hazard.IsActive(currentStage) {
				continue
			}

			// 2. Calcul de l'effet local (Proximité ou Grid-wide)
			dist := abs(tile.Position.X-world.playerPosition.X) + abs(tile.Position.Y-world.playerPosition.Y)
			inRange := false
			if hazard.Radius == 0 {
				// Rayon 0 : s'applique si sur la même grille (car le joueur est sur le bord)
				inRange = (gridID == world.CurrentGridID)
			} else {
				inRange = (dist <= hazard.Radius)
			}

			if inRange {
				// Dégâts de base
				localDmg := hazard.BaseDamage

				// Dégressivité par distance (uniquement si radius > 0)
				if hazard.Radius > 0 && dist > 0 {
					localDmg *= 1.0 - (float64(dist) / float64(hazard.Radius+1))
				}

				if hazard.DamageType == "amnesia" {
					dist := abs(tile.Position.X-world.playerPosition.X) + abs(tile.Position.Y-world.playerPosition.Y)
					fmt.Printf("[TOXICITY] Amnésie détectée: %s (Stade: %s) à %v, Dist: %d, Radius: %d\n", topID, stageName, tile.Position, dist, hazard.Radius)
				} else {
					fmt.Printf("[TOXICITY] Hazard détecté: %s (Type: %s, Stade: %s) à %v\n", topID, hazard.DamageType, stageName, tile.Position)
				}

				totalDamage += localDmg
				stackCount++
				if hazard.DegressionFactor > maxDegression {
					maxDegression = hazard.DegressionFactor
				}
		}
	}

	if stackCount == 0 {
		return
	}

	// 3. Application de la dégressivité par cumul (Diminishing returns)
	// Formule: Total = Somme / (1 + (N-1) * Facteur)
	finalDamage := totalDamage
	if stackCount > 1 {
		finalDamage = totalDamage / (1.0 + float64(stackCount-1)*maxDegression)
	}

	if finalDamage > 0 {
		dmgInt := int(finalDamage)
		if dmgInt == 0 && finalDamage > 0 {
			dmgInt = 1 // Minimum 1 dégât si actif
		}

		// CAS PARTICULIER : Amnésie (Void Bloom)
		isAmnesia := false
		amnesiaDuration := 0
		// On vérifie si un des hazards actifs est de type amnesia
		for _, tile := range grid.Plots {
			if len(tile.EntitiesID) == 0 {
				continue
			}
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if comp, ok := world.Components.Get(topID, "hazard"); ok {
				hazard := comp.(*component.Hazard)
				if hazard.DamageType == "amnesia" {
					dist := abs(tile.Position.X-world.playerPosition.X) + abs(tile.Position.Y-world.playerPosition.Y)
					fmt.Printf("[TOXICITY-DEBUG] Bloom at %v, Player at %v, Dist: %d, Radius: %d\n", tile.Position, world.playerPosition, dist, hazard.Radius)
					if dist <= hazard.Radius || (hazard.Radius == 0 && gridID == world.CurrentGridID) {
						if lcComp, ok := world.Components.Get(topID, "lifecycle"); ok {
							lc := lcComp.(*component.Lifecycle)
							if hazard.IsActive(lc.CurrentStage) {
								isAmnesia = true
								amnesiaDuration = dmgInt + lc.CurrentStage
								break
							}
						}
					}
				}
			}
		}

		if isAmnesia {
			fmt.Printf("[TOXICITY] Le joueur subit une amnésie de proximité (Void Bloom)\n")
			if world.Player.AmnesiaTurns <= 0 {
				// Récupère l'inclinaison de l'inventaire pour l'animation
				flipDir := entity.FlipCenter
				if world.InventoryGrid != nil {
					if plot, err := world.InventoryGrid.Get(board.Position{X: 0, Y: 0}); err == nil {
						flipDir = plot.Tilt.ToFlipDirection()
					}
				}
				world.HideInventory(flipDir)
				world.EventBus.Publish(event.NewAmnesiaStartedEvent("void_bloom", amnesiaDuration))
				world.EventBus.Publish(event.NewItemMessageEvent("Vous avez du mal à vous souvenir."))
			}
			world.Player.AmnesiaTurns = amnesiaDuration
		} else {
			fmt.Printf("[TOXICITY] Le joueur subit %d dégâts de poison (%d stacks actifs, dégressivité appliquée)\n", dmgInt, stackCount)
			world.Player.TakeDamage(dmgInt, "poison")
			world.EventBus.Publish(event.NewItemMessageEvent("Vous toussez du sang."))
			world.EventBus.PublishImmediate(event.NewPlayerDamagedEvent(
				"toxicity_system",
				dmgInt,
				"poison",
				"toxicity",
				map[string]interface{}{"stack_count": stackCount},
			))
		}
	}
}
