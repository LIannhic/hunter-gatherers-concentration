package board

import (
	"testing"
)

func TestNewGrid(t *testing.T) {
	// Ajout du biome forest pour correspondre à la signature
	g := NewGrid("test", 4, 4, BiomeForest)

	if g.Width != 4 || g.Height != 4 {
		t.Errorf("Expected 4x4 grid, got %dx%d", g.Width, g.Height)
	}

	expectedPlots := 16
	if len(g.Plots) != expectedPlots {
		t.Errorf("Expected %d plots, got %d", expectedPlots, len(g.Plots))
	}
}

func TestGridGet(t *testing.T) {
	g := NewGrid("test", 4, 4, BiomeForest)

	plot, err := g.Get(Position{X: 0, Y: 0})
	if err != nil {
		t.Errorf("Failed to get valid plot: %v", err)
	}
	if plot.Position.X != 0 || plot.Position.Y != 0 {
		t.Error("Got wrong plot position")
	}

	// Test out of bounds
	_, err = g.Get(Position{X: 10, Y: 10})
	if err == nil {
		t.Error("Should return error for out of bounds")
	}
}

func TestGridIsValid(t *testing.T) {
	g := NewGrid("test", 4, 4, BiomeForest)

	tests := []struct {
		pos   Position
		valid bool
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

func TestGetNeighbors(t *testing.T) {
	g := NewGrid("test", 4, 4, BiomeForest)

	// Corner should have 2 neighbors
	neighbors := g.GetNeighbors(Position{X: 0, Y: 0})
	if len(neighbors) != 2 {
		t.Errorf("Corner should have 2 neighbors, got %d", len(neighbors))
	}

	// Center should have 4 neighbors
	neighbors = g.GetNeighbors(Position{X: 1, Y: 1})
	if len(neighbors) != 4 {
		t.Errorf("Center should have 4 neighbors, got %d", len(neighbors))
	}
}

func TestGridStacking(t *testing.T) {
	g := NewGrid("test", 4, 4, BiomeForest)
	pos := Position{X: 1, Y: 1}

	g.PlaceEntity(pos, "entity_bottom")
	g.PlaceEntity(pos, "entity_top")

	plot, _ := g.Get(pos)

	if len(plot.EntitiesID) != 2 {
		t.Errorf("Expected 2 entities in stack, got %d", len(plot.EntitiesID))
	}

	top, _ := g.RemoveEntity(pos, "entity_top")
	if top != "entity_top" {
		t.Errorf("Expected entity_top to be removed, got %s", top)
	}

	bottom, _ := g.RemoveEntity(pos, "entity_bottom")
	if bottom != "entity_bottom" {
		t.Errorf("Expected entity_bottom to be removed, got %s", bottom)
	}
}

func TestCalculateFlipDirection(t *testing.T) {
	tileSize := 100

	tests := []struct {
		localX   int
		localY   int
		expected FlipDirection
	}{
		{50, 50, FlipCenter},      // Centre
		{50, 10, FlipTop},         // Haut
		{10, 50, FlipLeft},        // Gauche
		{90, 90, FlipBottomRight}, // Bas-Droite
	}

	for _, tc := range tests {
		result := CalculateFlipDirection(tileSize, tc.localX, tc.localY)
		if result != tc.expected {
			t.Errorf("At (%d, %d) expected %s, got %s", tc.localX, tc.localY, tc.expected.String(), result.String())
		}
	}
}
