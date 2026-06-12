package system

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

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

func TestDeployPortablePortal_Success(t *testing.T) {
	w := NewWorld()
	gridID := "dream_zone"
	w.CreateGrid(gridID, 6, 6, board.BiomeForest)

	// Don d'un portail au joueur + 2 autres items pour la taxe
	_ = w.AddLootItem(player.NewPortablePortalItem(0))
	_ = w.AddLootItem(player.NewPortablePortalItem(1))
	_ = w.AddLootItem(player.NewPortablePortalItem(2))

	initialHealth := w.Player.Stats.Health

	portal, err := w.DeployPortablePortal(gridID)
	if err != nil {
		t.Fatalf("Le déploiement automatique a échoué: %v", err)
	}

	if portal == nil {
		t.Fatal("Le portail créé ne devrait pas être nil")
	}

	// Au lieu de w.HasPortablePortal(), on vérifie que le nombre total d'items a diminué (consommation + taxe éventuelle)
	if w.Player.Inventory.GetTotalItems() >= 3 {
		t.Error("Le portail portable utilisé aurait dû être consommé")
	}

	if w.Player.Stats.Health != initialHealth {
		t.Errorf("Le joueur n'aurait pas dû subir de dégâts. Vie: %d", w.Player.Stats.Health)
	}

	pos := portal.GetPosition()
	if pos.X != 1 || pos.Y != 1 {
		t.Errorf("Le portail a été déployé en %v au lieu de (1,1)", pos)
	}
}

func TestDeployPortablePortal_ForcedPenaltyAndEarthquake(t *testing.T) {
	w := NewWorld()
	gridID := "obstructed_zone"
	w.CreateGrid(gridID, 4, 4, board.BiomeForest)

	// Obstruer REÉLLEMENT toutes les cases pour forcer la pénalité
	grid, _ := w.GetGrid(gridID)
	for _, plot := range grid.Plots {
		plot.Modifier.Obstructed = true
	}

	_ = w.AddLootItem(player.NewPortablePortalItem(0))
	initialHealth := w.Player.Stats.Health

	portal, err := w.DeployPortablePortal(gridID)
	if err != nil {
		t.Fatalf("Le déploiement forcé aurait dû réussir mais a renvoyé une erreur: %v", err)
	}

	expectedHealth := initialHealth - 5
	if w.Player.Stats.Health != expectedHealth {
		t.Errorf("Le joueur aurait dû subir 5 points de dégâts. Attendu: %d, Obtenu: %d", expectedHealth, w.Player.Stats.Health)
	}

	// Vérifier l'effet séisme : la zone 3x3 autour du portail doit être libérée d'obstructions
	center := portal.GetPosition()
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			checkPos := board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy}
			plot, err := grid.Get(checkPos)
			if err == nil && plot.Modifier.Obstructed {
				t.Errorf("L'effet séisme a échoué: la case %v est toujours obstruée", checkPos)
			}
		}
	}
}

func TestDeployPortablePortalAt_InvalidCoordinates(t *testing.T) {
	w := NewWorld()
	gridID := "small_zone"
	w.CreateGrid(gridID, 4, 4, board.BiomeForest)
	_ = w.AddLootItem(player.NewPortablePortalItem(0))

	_, err := w.DeployPortablePortalAt(gridID, board.Position{X: 0, Y: 0})
	if err == nil {
		t.Error("Le déploiement en (0,0) aurait dû être rejeté comme invalide")
	}
}

func TestDeployPortablePortal_NoItem(t *testing.T) {
	w := NewWorld()
	gridID := "zone"
	w.CreateGrid(gridID, 5, 5, board.BiomeForest)

	_, err := w.DeployPortablePortal(gridID)
	if err == nil {
		t.Error("Le déploiement aurait dû échouer car le joueur n'a pas de portail")
	}
}
