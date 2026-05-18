package persistence

import (
	"testing"
)

func TestNewSaveData(t *testing.T) {
	slotID := 1
	save := NewSaveData(slotID)

	if save.Meta.SlotID != slotID {
		t.Errorf("attendu SlotID %d, obtenu %d", slotID, save.Meta.SlotID)
	}

	if save.Hub == nil {
		t.Error("le Hub ne devrait pas être nul")
	}

	if save.Player == nil {
		t.Error("le Player ne devrait pas être nul")
	}

	if save.Meta.CreatedAt.IsZero() {
		t.Error("la date de création ne devrait pas être vide")
	}
}
