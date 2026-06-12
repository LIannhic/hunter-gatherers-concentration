package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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

func TestWorldSetPlayerPosition(t *testing.T) {
	w := NewWorld()

	w.SetPlayerPosition(entity.Position{X: 2, Y: 3})

	pos := w.GetPlayerPosition()
	if pos.X != 2 || pos.Y != 3 {
		t.Error("Player position not set correctly")
	}
}

func TestMultipleGrids(t *testing.T) {
	w := NewWorld()

	// Create multiple grids
	w.CreateGrid("grid1", 4, 4, board.BiomeForest)
	w.CreateGrid("grid2", 6, 6, board.BiomeForest)

	if len(w.Grids) != 2 {
		t.Errorf("Expected 2 grids, got %d", len(w.Grids))
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

func TestWorldGenerateLayoutPopulatesZones(t *testing.T) {
	w := NewWorld()
	w.GenerateLayout("test_plane")

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
