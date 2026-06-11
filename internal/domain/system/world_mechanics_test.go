package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

func TestTileFlippingMechanics_LimitTwo(t *testing.T) {
	w := NewWorld()
	w.Turn = 1

	// Au début du tour, on peut flipper
	if !w.CanFlipTile() {
		t.Error("Should be able to flip at the start of a turn")
	}

	// Premier flip
	w.AddFlippedTile(board.Position{X: 0, Y: 0})
	if w.GetFlippedTilesCount() != 1 {
		t.Errorf("Expected 1 flipped tile, got %d", w.GetFlippedTilesCount())
	}
	if !w.CanFlipTile() {
		t.Error("Should still be able to flip a second tile")
	}

	// Deuxième flip
	w.AddFlippedTile(board.Position{X: 1, Y: 0})
	if w.GetFlippedTilesCount() != 2 {
		t.Errorf("Expected 2 flipped tiles, got %d", w.GetFlippedTilesCount())
	}
	if w.CanFlipTile() {
		t.Error("Should NOT be able to flip a third tile this turn")
	}

	// Changement de tour -> Reset automatique du compteur
	w.Turn = 2
	if w.GetFlippedTilesCount() != 0 {
		t.Errorf("Expected 0 flipped tiles on a new turn, got %d", w.GetFlippedTilesCount())
	}
	if !w.CanFlipTile() {
		t.Error("Should be able to flip again on a new turn")
	}
}

func TestMatchTile(t *testing.T) {
	w := NewWorld()
	w.CreateGrid("test_grid", 4, 4, board.BiomeForest)
	pos := board.Position{X: 1, Y: 1}

	// Pop d'une ressource cachée par défaut
	r, _ := w.SpawnResource("test_grid", "dreamberry", entity.Position(pos))

	// Matcher la tuile
	err := w.MatchTile("test_grid", pos)
	if err != nil {
		t.Fatalf("MatchTile failed: %v", err)
	}

	// Vérifie que l'entité possède bien le bit Matched
	if r.GetState()&entity.Matched == 0 {
		t.Error("Entity state should have the Matched bit flag active")
	}
}
