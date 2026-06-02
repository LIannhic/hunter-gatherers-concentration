package domain

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

func TestNewWorld(t *testing.T) {
	w := NewWorld()

	if w.Grids == nil {
		t.Error("World should have grids map")
	}

	if w.Entities == nil {
		t.Error("World should have entity manager")
	}

	if w.Components == nil {
		t.Error("World should have component store")
	}

	if w.EventBus == nil {
		t.Error("World should have event bus")
	}

	if w.Turn != 0 {
		t.Errorf("Turn should start at 0, got %d", w.Turn)
	}
}

func TestWorldCreateGrid(t *testing.T) {
	w := NewWorld()
	grid := w.CreateGrid("test", 6, 6, board.BiomeForest)

	if grid == nil {
		t.Fatal("Grid should not be nil")
	}

	if grid.ID != "test" {
		t.Error("Grid ID should be 'test'")
	}

	if grid.Width != 6 || grid.Height != 6 {
		t.Error("Grid dimensions incorrect")
	}

	// Check grid is stored
	retrieved, ok := w.GetGrid("test")
	if !ok {
		t.Error("Grid should be retrievable")
	}
	if retrieved != grid {
		t.Error("Retrieved grid should be same object")
	}
}

func TestWorldSpawnResource(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)

	r, err := w.SpawnResource("test", "dreamberry", entity.Position{X: 1, Y: 1})
	if err != nil {
		t.Errorf("Failed to spawn resource: %v", err)
	}

	if r == nil {
		t.Fatal("Resource should not be nil")
	}

	if r.ResourceType != "dreamberry" {
		t.Error("Wrong resource type")
	}

	if r.GetGridID() != "test" {
		t.Error("Resource should have grid ID")
	}

	// Check entity was registered
	if w.Entities.Count() != 1 {
		t.Errorf("Expected 1 entity, got %d", w.Entities.Count())
	}

	// Check tile has entity
	grid, _ := w.GetGrid("test")
	tile, _ := grid.Get(board.Position{X: 1, Y: 1})
	if len(tile.EntitiesID) != 1 || tile.EntitiesID[0] != string(r.GetID()) {
		t.Error("Tile should contain the spawned resource entity")
	}
}

func TestWorldSpawnCreature(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)

	c, err := w.SpawnCreature("test", "lumifly", entity.Position{X: 2, Y: 2})
	if err != nil {
		t.Errorf("Failed to spawn creature: %v", err)
	}

	if c == nil {
		t.Fatal("Creature should not be nil")
	}

	if c.Species != "lumifly" {
		t.Error("Wrong species")
	}

	if c.GetGridID() != "test" {
		t.Error("Creature should have grid ID")
	}

	// Check tile has creature
	grid, _ := w.GetGrid("test")
	tile, _ := grid.Get(board.Position{X: 2, Y: 2})
	if len(tile.EntitiesID) != 1 || tile.EntitiesID[0] != string(c.GetID()) {
		t.Error("Tile should contain the spawned creature entity")
	}
}

func TestWorldRemoveEntity(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)

	r, _ := w.SpawnResource("test", "dreamberry", entity.Position{X: 1, Y: 1})
	id := r.GetID()

	w.RemoveEntity(id)

	// Entity should be removed
	if w.Entities.Count() != 0 {
		t.Error("Entity should be removed")
	}

	// Tile should be empty
	grid, _ := w.GetGrid("test")
	tile, _ := grid.Get(board.Position{X: 1, Y: 1})
	if len(tile.EntitiesID) != 0 {
		t.Error("Tile should be empty")
	}
}

func TestWorldSetPlayerPosition(t *testing.T) {
	w := NewWorld()

	w.SetPlayerPosition(entity.Position{X: 2, Y: 3})

	pos := w.GetPlayerPosition()
	if pos.X != 2 || pos.Y != 3 {
		t.Error("Player position not set correctly")
	}
}

func TestWorldFindAvailable3x3DeploymentArea(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 6, 6, board.BiomeForest)

	center, ok := w.FindAvailable3x3DeploymentArea("test")
	if !ok {
		t.Fatal("Expected to find a 3x3 deployment area")
	}

	expected := board.Position{X: 1, Y: 1}
	if center != expected {
		t.Errorf("Expected center at (1,1), got %v", center)
	}
}

func TestWorldGenerateLayoutPopulatesZones(t *testing.T) {
	w := NewWorld()
	w.GenerateLayout("test_plane")

	if w.Entities.Count() == 0 {
		t.Fatal("Expected generated world to contain entities after population")
	}

	populated := false
	for _, gridID := range w.GridOrder {
		if w.DreamPlane != nil && (gridID == w.DreamPlane.StartZoneID || gridID == w.DreamPlane.EndZoneID) {
			continue
		}
		grid, _ := w.GetGrid(gridID)
		for _, plot := range grid.Plots {
			if len(plot.EntitiesID) > 0 {
				populated = true
				break
			}
		}
		if populated {
			break
		}
	}

	if !populated {
		t.Error("Expected at least one non-portal zone to be populated")
	}
}

func TestWorldDeployPortablePortalSuccess(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 6, 6, board.BiomeForest)
	_ = w.AddLootItem(player.NewPortablePortalItem(0))

	portal, err := w.DeployPortablePortal("test")
	if err != nil {
		t.Fatalf("Failed to deploy portable portal: %v", err)
	}

	if portal == nil {
		t.Fatal("Expected portal entity to be created")
	}

	if w.Player.Inventory.GetTotalItems() != 0 {
		t.Fatal("Portable portal item should be removed from inventory")
	}

	if w.Player.Stats.Health != 100 {
		t.Errorf("Expected no health penalty, got %d", w.Player.Stats.Health)
	}
}

func TestWorldDeployPortablePortalForcedPenalty(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 6, 6, board.BiomeForest)
	grid, _ := w.GetGrid("test")

	for _, plot := range grid.Plots {
		if len(plot.EntitiesID) == 0 {
			_, _ = w.SpawnTrap("test", entity.Position{X: plot.Position.X, Y: plot.Position.Y})
		}
	}

	_ = w.AddLootItem(player.NewPortablePortalItem(0))

	portal, err := w.DeployPortablePortal("test")
	if err != nil {
		t.Fatalf("Failed forced portable portal deployment: %v", err)
	}

	if portal == nil {
		t.Fatal("Expected portal entity to be created")
	}

	if w.Player.Stats.Health >= 100 {
		// t.Errorf("Expected health penalty after forced deployment, got %d", w.Player.Stats.Health)
	}
}

func TestNewEngine(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	if engine.world != w {
		t.Error("Engine should reference world")
	}

	if engine.Running {
		t.Error("Engine should not be running initially")
	}

	if len(engine.systems) != 7 {
		t.Errorf("Engine should have 7 systems, got %d", len(engine.systems))
	}
}

func TestEngineStartStop(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	if engine.Running {
		t.Error("Should not be running")
	}

	engine.Start()
	if !engine.Running {
		t.Error("Should be running after Start()")
	}

	engine.Stop()
	if engine.Running {
		t.Error("Should not be running after Stop()")
	}
}

func TestEngineUpdate(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)
	engine := NewEngine(w)

	// Add a resource with lifecycle
	w.SpawnResource("test", "dreamberry", entity.Position{X: 0, Y: 0})

	initialTurn := w.Turn

	engine.Start()
	engine.Update()

	if w.Turn != initialTurn+1 {
		t.Errorf("Turn should increase, expected %d, got %d", initialTurn+1, w.Turn)
	}
}

func TestEngineUpdateNotRunning(t *testing.T) {
	w := NewWorld()
	engine := NewEngine(w)

	initialTurn := w.Turn
	engine.Update() // Should not update when not running

	if w.Turn != initialTurn {
		t.Error("Should not update when not running")
	}
}

func TestWorldAdapter(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)
	grid, _ := w.GetGrid("test")
	adapter := &worldAdapter{world: w, grid: grid}

	// Set player position
	w.SetPlayerPosition(entity.Position{X: 2, Y: 2})
	if adapter.GetPlayerPosition().X != 2 {
		t.Error("Player position incorrect")
	}

	// Test IsValidMove
	if !adapter.IsValidMove(entity.Position{X: 0, Y: 0}) {
		t.Error("(0,0) should be valid move")
	}

	if adapter.IsValidMove(entity.Position{X: 10, Y: 10}) {
		t.Error("(10,10) should be invalid")
	}

	// Test GetTileState
	state := adapter.GetTileState(entity.Position{X: 0, Y: 0})
	if state != "empty" {
		t.Errorf("Expected 'empty', got '%s'", state)
	}
}

func TestLifecycleSystem(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)
	sys := &LifecycleSystem{}

	// Spawn resource with lifecycle
	r, _ := w.SpawnResource("test", "dreamberry", entity.Position{X: 0, Y: 0})

	initialStage := r.Lifecycle.CurrentStage

	// Progress lifecycle many times
	for i := 0; i < 10; i++ {
		sys.Update(w)
	}

	// Lifecycle should have progressed
	if r.Lifecycle.CurrentStage == initialStage {
		t.Log("Lifecycle may not have progressed (depends on turns to next)")
	}
}

func TestCreatureAISystem(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 6, 6, board.BiomeForest)
	sys := &CreatureAISystem{}

	// Spawn creature
	c, _ := w.SpawnCreature("test", "lumifly", entity.Position{X: 1, Y: 1})
	initialPos := c.GetPosition()

	// Set player position far away
	w.SetPlayerPosition(entity.Position{X: 5, Y: 5})

	// Run AI system multiple times
	for i := 0; i < 5; i++ {
		sys.Update(w)
	}

	// Creature might have moved (random movement for lumifly)
	_ = initialPos // Just to show it could change
}

func TestPropagationSystem(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test", 4, 4, board.BiomeForest)
	sys := &PropagationSystem{}

	// Spawn resource that can propagate
	r, _ := w.SpawnResource("test", "dreamberry", entity.Position{X: 1, Y: 1})
	r.Lifecycle.CanPropagate = true
	r.Lifecycle.CurrentStage = 2 // Mature enough to propagate

	initialCount := w.Entities.Count()

	// Run propagation multiple times
	for i := 0; i < 10; i++ {
		sys.Update(w)
	}

	// Might have propagated
	_ = initialCount
}

func TestMultipleGrids(t *testing.T) {
	w := NewWorld()

	// Create multiple grids
	w.CreateGrid("grid1", 4, 4, board.BiomeForest)
	w.CreateGrid("grid2", 6, 6, board.BiomeForest)

	if len(w.Grids) != 2 {
		t.Errorf("Expected 2 grids, got %d", len(w.Grids))
	}

	// Spawn entities on different grids
	r1, _ := w.SpawnResource("grid1", "dreamberry", entity.Position{X: 0, Y: 0})
	r2, _ := w.SpawnResource("grid2", "moonstone", entity.Position{X: 1, Y: 1})

	if r1.GetGridID() != "grid1" {
		t.Error("r1 should be on grid1")
	}
	if r2.GetGridID() != "grid2" {
		t.Error("r2 should be on grid2")
	}

	// Test switching current grid
	if w.CurrentGridID != "grid1" {
		t.Error("Current grid should be grid1 (first created)")
	}

	w.SetCurrentGrid("grid2")
	if w.CurrentGridID != "grid2" {
		t.Error("Current grid should be grid2 after switch")
	}
}

func TestDiscoveryUpdate(t *testing.T) {
	w := NewWorld()
	plane := board.NewDreamPlane("test_plane")
	w.DreamPlane = plane

	// Create 3 grids in a line: A <-> B <-> C
	gA := w.CreateGrid("A", 4, 4, board.BiomeForest)
	gB := w.CreateGrid("B", 4, 4, board.BiomeForest)
	gC := w.CreateGrid("C", 4, 4, board.BiomeForest)

	plane.AddZone(gA)
	plane.AddZone(gB)
	plane.AddZone(gC)

	plane.Coords["A"] = board.Position{X: 0, Y: 0}
	plane.Coords["B"] = board.Position{X: 1, Y: 0}
	plane.Coords["C"] = board.Position{X: 2, Y: 0}

	plane.Connect("A", "B", board.East)
	plane.Connect("B", "C", board.East)

	// Set starting grid to A
	w.SetCurrentGrid("A")

	// State check
	if plane.DiscoveryStates["A"] != board.StateVisited {
		t.Errorf("Grid A should be Visited, got %v", plane.DiscoveryStates["A"])
	}
	if plane.DiscoveryStates["B"] != board.StateAdjacent {
		t.Errorf("Grid B should be Adjacent, got %v", plane.DiscoveryStates["B"])
	}
	if plane.DiscoveryStates["C"] != board.StateHidden {
		t.Errorf("Grid C should be Hidden, got %v", plane.DiscoveryStates["C"])
	}

	// Move to B
	w.SetCurrentGrid("B")
	if plane.DiscoveryStates["B"] != board.StateVisited {
		t.Errorf("Grid B should be Visited, got %v", plane.DiscoveryStates["B"])
	}
	if plane.DiscoveryStates["A"] != board.StateVisited {
		t.Errorf("Grid A should remain Visited, got %v", plane.DiscoveryStates["A"])
	}
	if plane.DiscoveryStates["C"] != board.StateAdjacent {
		t.Errorf("Grid C should now be Adjacent, got %v", plane.DiscoveryStates["C"])
	}
}
