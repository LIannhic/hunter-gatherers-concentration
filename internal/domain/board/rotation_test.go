package board

import (
	"testing"
)

func TestGridRotateClockwise(t *testing.T) {
	g := NewGrid("test", 4, 4, BiomeForest)

	// Placement d'une entité en (1, 0)
	// Après rotation 90° horaire :
	// newX = height - 1 - oldY = 4 - 1 - 0 = 3
	// newY = oldX = 1
	// Résultat attendu : (3, 1)

	pos := Position{X: 1, Y: 0}
	g.PlaceEntity(pos, "test_entity")

	g.RotateClockwise()

	// Vérifie le bearing
	if g.MainBearing != BearingEast {
		t.Errorf("Expected MainBearing to be East (1), got %d", g.MainBearing)
	}

	// Vérifie la nouvelle position de la parcelle
	newPos := Position{X: 3, Y: 1}
	plot, err := g.Get(newPos)
	if err != nil {
		t.Fatalf("Plot at %v not found after rotation: %v", newPos, err)
	}

	if len(plot.EntitiesID) != 1 || plot.EntitiesID[0] != "test_entity" {
		t.Errorf("Expected entity at %v, but plot entities are %v", newPos, plot.EntitiesID)
	}

	// Vérifie que l'ancienne position est vide
	oldPlot, _ := g.Get(pos)
	if len(oldPlot.EntitiesID) != 0 {
		t.Errorf("Expected old plot at %v to be empty, but got %v", pos, oldPlot.EntitiesID)
	}
}
