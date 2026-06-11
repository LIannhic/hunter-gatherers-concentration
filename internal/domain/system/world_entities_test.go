package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

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

	if w.Entities.Count() != 1 {
		t.Errorf("Expected 1 entity, got %d", w.Entities.Count())
	}

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

	if w.Entities.Count() != 0 {
		t.Error("Entity should be removed")
	}

	grid, _ := w.GetGrid("test")
	tile, _ := grid.Get(board.Position{X: 1, Y: 1})
	if len(tile.EntitiesID) != 0 {
		t.Error("Tile should be empty")
	}
}
