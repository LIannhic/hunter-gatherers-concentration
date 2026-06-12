package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestGetGridForEntity(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("zone_A", 4, 4, board.BiomeForest)

	// Cas 1 : L'entité n'existe pas
	_, ok := w.GetGridForEntity("unknown_id")
	if ok {
		t.Error("Expected ok=false for an unknown entity ID")
	}

	// Cas 2 : L'entité existe et est associée à sa grille
	r, err := w.SpawnResource("zone_A", "dreamberry", entity.Position{X: 1, Y: 1})
	if err != nil {
		t.Fatalf("Failed to spawn resource for test: %v", err)
	}

	grid, ok := w.GetGridForEntity(string(r.GetID()))
	if !ok {
		t.Fatal("Expected to find grid for the spawned resource")
	}
	if grid.ID != "zone_A" {
		t.Errorf("Expected grid ID 'zone_A', got '%s'", grid.ID)
	}
}

func TestHasResourceAt(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("zone_B", 4, 4, board.BiomeForest)

	posWithResource := board.Position{X: 2, Y: 2}
	posEmpty := board.Position{X: 0, Y: 0}
	posInvalid := board.Position{X: 99, Y: 99}

	// Ajout d'une ressource
	_, _ = w.SpawnResource("zone_B", "dreamberry", entity.Position(posWithResource))

	// Cas 1 : Il y a bien une ressource
	if !w.HasResourceAt("zone_B", posWithResource) {
		t.Errorf("Expected HasResourceAt to be true at %v", posWithResource)
	}

	// Cas 2 : La case est vide
	if w.HasResourceAt("zone_B", posEmpty) {
		t.Errorf("Expected HasResourceAt to be false at %v (empty plot)", posEmpty)
	}

	// Cas 3 : ID de grille invalide
	if w.HasResourceAt("unknown_grid", posWithResource) {
		t.Error("Expected HasResourceAt to be false for an unknown grid ID")
	}

	// Cas 4 : Position hors-limites de la grille
	if w.HasResourceAt("zone_B", posInvalid) {
		t.Error("Expected HasResourceAt to be false for out-of-bounds coordinates")
	}
}

func TestMoveSpeciesOneStepTowards_SafeWithNilEngine(t *testing.T) {
	w := NewWorld()
	w.Engine = nil // On force l'Engine à nil pour vérifier que le wrapper ne panic pas

	// Ce test passe si la fonction gère le cas de l'Engine manquant sans planter (Crash/Panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MoveSpeciesOneStepTowards panicked with nil engine: %v", r)
		}
	}()

	w.MoveSpeciesOneStepTowards("lumifly", entity.Position{X: 1, Y: 1})
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
