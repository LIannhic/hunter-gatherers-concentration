package actionbuttons

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
)

func TestNewManager(t *testing.T) {
	m := NewManager(
		func() int { return 0 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()
	if len(states) != 4 {
		t.Fatalf("Expected 4 buttons, got %d", len(states))
	}
}

func TestButtonActivationAtRest(t *testing.T) {
	m := NewManager(
		func() int { return 0 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()

	// Match et Skip doivent être inactifs au repos
	if states[BtnMatch].Active {
		t.Error("Match should be inactive at rest")
	}
	if states[BtnSkip].Active {
		t.Error("Skip should be inactive at rest")
	}
	// EndTurn et Menu toujours actifs
	if !states[BtnEndTurn].Active {
		t.Error("EndTurn should always be active")
	}
	if states[BtnMerge].Active {
		t.Error("Merge should be inactive at rest")
	}
}

func TestButtonActivationWhenTwoTilesRevealed(t *testing.T) {
	m := NewManager(
		func() int { return 2 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0.5 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()

	if !states[BtnMatch].Active {
		t.Error("Match should be active when 2 tiles are revealed")
	}
	if !states[BtnSkip].Active {
		t.Error("Skip should be active when 2 tiles are revealed")
	}
}

func TestBaseCoordinates(t *testing.T) {
	m := NewManager(
		func() int { return 0 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()

	// Vérifie que les coordonnées de base sont correctes
	expected := []struct{ x, y float64 }{
		{ui.PlaymatX + ui.ActionBtn1X, ui.PlaymatY + ui.ActionBtn1Y},
		{ui.PlaymatX + ui.ActionBtn2X, ui.PlaymatY + ui.ActionBtn2Y},
		{ui.PlaymatX + ui.ActionBtn3X, ui.PlaymatY + ui.ActionBtn3Y},
		{ui.PlaymatX + ui.ActionBtn4X, ui.PlaymatY + ui.ActionBtn4Y},
	}

	for i, exp := range expected {
		if states[i].X != exp.x || states[i].Y != exp.y {
			t.Errorf("Button %d expected (%v,%v), got (%v,%v)", i, exp.x, exp.y, states[i].X, states[i].Y)
		}
	}
}

func TestImpairmentScrambling(t *testing.T) {
	p := player.New("test")
	p.AtaxiaTurns = 3

	m := NewManager(
		func() int { return 2 },
		func() []string { return nil },
		func() *player.Player { return p },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()

	scrambled := false
	for _, s := range states {
		if s.Scrambled {
			scrambled = true
			break
		}
	}
	if !scrambled {
		t.Error("At least one button should be scrambled when impairment is active")
	}
}

func TestHitTest(t *testing.T) {
	m := NewManager(
		func() int { return 2 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0.3 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()

	// Clic au centre du bouton Match (on utilise CurrentX/CurrentY car c'est ce que HitTest utilise)
	bx := int(states[BtnMatch].CurrentX + states[BtnMatch].Width/2)
	by := int(states[BtnMatch].CurrentY + states[BtnMatch].Height/2)
	id, ok := m.HitTest(bx, by, states)
	if !ok || id != BtnMatch {
		t.Errorf("HitTest should detect Match button at (%d,%d), got id=%d, ok=%v", bx, by, id, ok)
	}

	// Clic en dehors
	_, ok = m.HitTest(0, 0, states)
	if ok {
		t.Error("HitTest should not detect anything at (0,0)")
	}
}

func TestVictoryEndTurn(t *testing.T) {
	m := NewManager(
		func() int { return 0 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0.7 },
		func() bool { return true },
		func() bool { return false },
	)
	states := m.ComputeStates()
	if states[BtnEndTurn].Label != "END GAME" {
		t.Errorf("Expected EndTurn label to be END GAME, got %s", states[BtnEndTurn].Label)
	}
	if states[BtnEndTurn].FillProgress != 0.7 {
		t.Errorf("Expected EndTurn FillProgress 0.7, got %v", states[BtnEndTurn].FillProgress)
	}
	if !states[BtnEndTurn].FillAlert {
		t.Error("Expected EndTurn FillAlert to be true when victory is active")
	}
}

func TestTimerPanicSetsSkipAlert(t *testing.T) {
	m := NewManager(
		func() int { return 2 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0.2 },
		func() bool { return true },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()
	if !states[BtnSkip].FillAlert {
		t.Error("Expected Skip FillAlert to be true when timer panic is active")
	}
}

func TestAgnosiaSetsAllScrambled(t *testing.T) {
	p := player.New("test")
	p.AgnosiaTurns = 3

	m := NewManager(
		func() int { return 2 },
		func() []string { return nil },
		func() *player.Player { return p },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()
	for i := 0; i < 4; i++ {
		if !states[i].Scrambled {
			t.Errorf("Expected button %d to be scrambled under Agnosia", i)
		}
	}
}

func TestAphasiaAltersLabels(t *testing.T) {
	p := player.New("test")
	p.AphasiaTurns = 3

	m := NewManager(
		func() int { return 0 },
		func() []string { return nil },
		func() *player.Player { return p },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)

	// Force un glitch immédiat en avançant le temps ou en appelant ComputeStates plusieurs fois
	// Mais ici on va juste vérifier que les labels changent après le premier glitch
	// Comme m.lastGlitch est initialisé à zero time, le premier appel devrait glitche.
	states := m.ComputeStates()

	// Ensure at least one label differs from the base label at the same index
	diff := false
	base := [4]string{"MATCH", "SKIP", "TURN", "MERGE"}
	for i := 0; i < 4; i++ {
		if states[i].Label != base[i] {
			diff = true
			break
		}
	}
	if !diff {
		t.Error("Expected at least one label to be altered under Aphasia")
	}
}

func TestRevealedEntitiesPropagation(t *testing.T) {
	revealed := []string{"ent1", "ent2"}
	m := NewManager(
		func() int { return 2 },
		func() []string { return revealed },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return false },
	)
	states := m.ComputeStates()
	for i := 0; i < 4; i++ {
		if len(states[i].RevealedEntities) != 2 {
			t.Errorf("Button %d: expected 2 revealed entities, got %d", i, len(states[i].RevealedEntities))
		}
		if states[i].RevealedEntities[0] != "ent1" || states[i].RevealedEntities[1] != "ent2" {
			t.Errorf("Button %d: unexpected revealed entities content: %v", i, states[i].RevealedEntities)
		}
	}
}

func TestPortalMatchLabels(t *testing.T) {
	m := NewManager(
		func() int { return 2 },
		func() []string { return nil },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
		func() bool { return true }, // portal match
	)
	states := m.ComputeStates()

	if states[BtnMatch].Label != "EXTRACT" {
		t.Errorf("Expected BtnMatch label EXTRACT, got %s", states[BtnMatch].Label)
	}
	if states[BtnMerge].Label != "EXTRACT" {
		t.Errorf("Expected BtnMerge label EXTRACT, got %s", states[BtnMerge].Label)
	}
	if states[BtnSkip].Label != "STAY" {
		t.Errorf("Expected BtnSkip label STAY, got %s", states[BtnSkip].Label)
	}
	if states[BtnEndTurn].Label != "STAY" {
		t.Errorf("Expected BtnEndTurn label STAY, got %s", states[BtnEndTurn].Label)
	}
}
