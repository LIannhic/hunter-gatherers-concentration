package hud

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
)

func TestMessageQueue(t *testing.T) {
	world := domain.NewWorld()
	h := NewHUD(world)

	// 1. Ajouter un message
	h.AddMessage("Test Message", "left")
	if len(h.queueLeft) != 1 {
		t.Errorf("Expected queue size 1, got %d", len(h.queueLeft))
	}

	// 2. Première mise à jour : le message devient actif
	h.Update()
	if h.activeLeft == nil {
		t.Fatal("Message should be active after Update")
	}
	if h.activeLeft.Text != "Test Message" {
		t.Errorf("Expected 'Test Message', got %s", h.activeLeft.Text)
	}

	// 3. Simuler le défilement
	// On force X à être très négatif pour simuler la fin du premier passage
	h.activeLeft.X = -1000
	h.Update() // Devrait réinitialiser X et incrémenter RepeatCount

	if h.activeLeft.RepeatCount != 1 {
		t.Errorf("Expected RepeatCount 1, got %d", h.activeLeft.RepeatCount)
	}
	if h.activeLeft.X < 0 {
		t.Error("X should be reset to BoxWidth after repeat")
	}

	// 4. Deuxième passage
	h.activeLeft.X = -1000
	h.Update() // Devrait passer active à nil

	if h.activeLeft != nil {
		t.Error("Message should be finished after 2 passes")
	}
}

func TestMessageQueueSeparation(t *testing.T) {
	world := domain.NewWorld()
	h := NewHUD(world)

	h.AddMessage("Left", "left")
	h.AddMessage("Right", "right")

	h.Update()

	if h.activeLeft == nil || h.activeLeft.Text != "Left" {
		t.Error("Left area should have active message 'Left'")
	}
	if h.activeRight == nil || h.activeRight.Text != "Right" {
		t.Error("Right area should have active message 'Right'")
	}
}
