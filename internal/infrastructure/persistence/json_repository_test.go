package persistence

import (
	"os"
	"testing"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
)

func TestJsonRepository(t *testing.T) {
	// Création d'un répertoire temporaire pour les tests
	tmpDir, err := os.MkdirTemp("", "save_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo := NewJsonRepository(tmpDir)

	t.Run("Save and Load", func(t *testing.T) {
		data := persistence.NewSaveData(1)
		data.Meta.SessionCount = 5
		data.Hub.Family.Debt = 500

		err := repo.Save(1, data)
		if err != nil {
			t.Errorf("échec de la sauvegarde : %v", err)
		}

		loaded, err := repo.Load(1)
		if err != nil {
			t.Errorf("échec du chargement : %v", err)
		}

		if loaded.Meta.SessionCount != 5 {
			t.Errorf("attendu SessionCount 5, obtenu %d", loaded.Meta.SessionCount)
		}
	})

	t.Run("Latest Slot Logic", func(t *testing.T) {
		_ = repo.Save(1, persistence.NewSaveData(1))
		time.Sleep(10 * time.Millisecond)
		_ = repo.Save(2, persistence.NewSaveData(2))

		latest, _ := repo.GetLatestSlotID()
		if latest != 2 {
			t.Errorf("attendu slot 2 (plus récent), obtenu %d", latest)
		}
	})
}
