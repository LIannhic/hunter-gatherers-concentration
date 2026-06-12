package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

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

func TestGetGrid_InvalidID(t *testing.T) {
	w := NewWorld()
	_, ok := w.GetGrid("non_existent_grid_id")
	if ok {
		t.Error("Expected ok=false for a non-existent grid ID")
	}
}

func TestSyncInventoryGrid_EmptyInventory(t *testing.T) {
	w := NewWorld()
	// Initialise la grille d'inventaire
	grid := w.CreateGrid(board.InventoryGridID, 3, 3, board.BiomeForest)

	// Simule une entité résiduelle fantôme dans un plot
	plot, _ := grid.Get(board.Position{X: 0, Y: 0})
	plot.EntitiesID = append(plot.EntitiesID, "ghost_entity")

	// L'inventaire du joueur est vide
	w.Player = player.New("TestPlayer")

	// Synchronise
	w.SyncInventoryGrid()

	// Vérifie que le nettoyage a bien eu lieu
	if len(plot.EntitiesID) != 0 {
		t.Error("Expected inventory grid plot to be cleaned up and empty after sync")
	}
}
