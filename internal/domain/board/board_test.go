package board

import (
	"testing"
)

func TestNewGrid(t *testing.T) {
	g := NewGrid(4, 4)
	
	if g.Width != 4 || g.Height != 4 {
		t.Errorf("Expected 4x4 grid, got %dx%d", g.Width, g.Height)
	}
	
	expectedTiles := 16
	if len(g.Tiles) != expectedTiles {
		t.Errorf("Expected %d tiles, got %d", expectedTiles, len(g.Tiles))
	}
}

func TestGridGet(t *testing.T) {
	g := NewGrid(4, 4)
	
	tile, err := g.Get(Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to get valid tile: %v", err)
	}
	if tile.Position.X != 0 || tile.Position.Y != 0 {
		t.Error("Got wrong tile position")
	}
	
	// Test out of bounds
	_, err = g.Get(Position{X: 10, Y: 10})
	if err == nil {
		t.Error("Should return error for out of bounds")
	}
}

func TestGridIsValid(t *testing.T) {
	g := NewGrid(4, 4)
	
	tests := []struct {
		pos    Position
		valid  bool
	}{
		{Position{0, 0}, true},
		{Position{3, 3}, true},
		{Position{4, 0}, false},
		{Position{0, 4}, false},
		{Position{-1, 0}, false},
		{Position{0, -1}, false},
	}
	
	for _, tc := range tests {
		result := g.IsValid(tc.pos)
		if result != tc.valid {
			t.Errorf("IsValid(%v) = %v, expected %v", tc.pos, result, tc.valid)
		}
	}
}

func TestGridReveal(t *testing.T) {
	g := NewGrid(4, 4)
	
	tile, err := g.Reveal(Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to reveal tile: %v", err)
	}
	
	if tile.State != Revealed {
		t.Errorf("Expected state Revealed, got %v", tile.State)
	}
	
	// Can't reveal again
	_, err = g.Reveal(Position{X: 0, Y: 0})
	if err == nil {
		t.Error("Should not be able to reveal already revealed tile")
	}
}

func TestGridHide(t *testing.T) {
	g := NewGrid(4, 4)
	
	// Reveal then hide
	g.Reveal(Position{X: 0, Y: 0})
	err := g.Hide(Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to hide tile: %v", err)
	}
	
	tile, _ := g.Get(Position{X: 0, Y: 0})
	if tile.State != Hidden {
		t.Error("Tile should be hidden")
	}
}

func TestGridMatch(t *testing.T) {
	g := NewGrid(4, 4)
	
	// Can't match hidden tile
	err := g.Match(Position{X: 0, Y: 0})
	if err == nil {
		t.Error("Should not be able to match hidden tile")
	}
	
	// Reveal then match
	g.Reveal(Position{X: 0, Y: 0})
	err = g.Match(Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to match tile: %v", err)
	}
	
	tile, _ := g.Get(Position{X: 0, Y: 0})
	if tile.State != Matched {
		t.Error("Tile should be matched")
	}
	
	// Can't hide matched tile
	err = g.Hide(Position{X: 0, Y: 0})
	if err == nil {
		t.Error("Should not be able to hide matched tile")
	}
}

func TestGetNeighbors(t *testing.T) {
	g := NewGrid(4, 4)
	
	// Corner should have 2 neighbors
	neighbors := g.GetNeighbors(Position{X: 0, Y: 0})
	if len(neighbors) != 2 {
		t.Errorf("Corner should have 2 neighbors, got %d", len(neighbors))
	}
	
	// Edge should have 3 neighbors
	neighbors = g.GetNeighbors(Position{X: 1, Y: 0})
	if len(neighbors) != 3 {
		t.Errorf("Edge should have 3 neighbors, got %d", len(neighbors))
	}
	
	// Center should have 4 neighbors
	neighbors = g.GetNeighbors(Position{X: 1, Y: 1})
	if len(neighbors) != 4 {
		t.Errorf("Center should have 4 neighbors, got %d", len(neighbors))
	}
}

func TestDirectionVector(t *testing.T) {
	tests := []struct {
		dir      Direction
		expected Position
	}{
		{North, Position{0, -1}},
		{South, Position{0, 1}},
		{East, Position{1, 0}},
		{West, Position{-1, 0}},
	}
	
	for _, tc := range tests {
		result := tc.dir.Vector()
		if result != tc.expected {
			t.Errorf("%v.Vector() = %v, expected %v", tc.dir, result, tc.expected)
		}
	}
}

func TestPositionDistance(t *testing.T) {
	p1 := Position{X: 0, Y: 0}
	p2 := Position{X: 3, Y: 4}
	
	dist := p1.Distance(p2)
	if dist != 7 { // 3 + 4 = 7 (Manhattan distance)
		t.Errorf("Distance should be 7, got %d", dist)
	}
}

func TestGridPlaceEntity(t *testing.T) {
	g := NewGrid(4, 4)
	
	err := g.PlaceEntity(Position{X: 0, Y: 0}, "entity1")
	if err != nil {
		t.Errorf("Failed to place entity: %v", err)
	}
	
	tile, _ := g.Get(Position{X: 0, Y: 0})
	if tile.EntityID != "entity1" {
		t.Errorf("Expected entity1, got %s", tile.EntityID)
	}
	
	// Can't place on occupied tile
	err = g.PlaceEntity(Position{X: 0, Y: 0}, "entity2")
	if err == nil {
		t.Error("Should not be able to place on occupied tile")
	}
}

func TestCountByState(t *testing.T) {
	g := NewGrid(2, 2)
	
	// All tiles hidden initially
	if g.CountByState(Hidden) != 4 {
		t.Error("Expected 4 hidden tiles")
	}
	
	g.Reveal(Position{X: 0, Y: 0})
	g.Reveal(Position{X: 1, Y: 1})
	
	if g.CountByState(Revealed) != 2 {
		t.Error("Expected 2 revealed tiles")
	}
}
