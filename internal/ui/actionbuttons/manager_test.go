package actionbuttons

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
)

func TestNewManager(t *testing.T) {
	m := NewManager(
		func() int { return 0 },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
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
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
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
	if !states[BtnMenu].Active {
		t.Error("Menu should always be active")
	}
}

func TestButtonActivationWhenTwoTilesRevealed(t *testing.T) {
	m := NewManager(
		func() int { return 2 },
		func() *player.Player { return player.New("test") },
		func() float64 { return 0.5 },
		func() bool { return false },
		func() float64 { return 0 },
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
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
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
	p.StatusEffects.AddImpairment(player.ImpairmentAtaxia)

	m := NewManager(
		func() int { return 2 },
		func() *player.Player { return p },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
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
		func() *player.Player { return player.New("test") },
		func() float64 { return 0.3 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
	)
	states := m.ComputeStates()

	// Clic au centre du bouton Match
	bx := int(states[BtnMatch].X + states[BtnMatch].Width/2)
	by := int(states[BtnMatch].Y + states[BtnMatch].Height/2)
	id, ok := m.HitTest(bx, by, states)
	if !ok || id != BtnMatch {
		t.Error("HitTest should detect Match button")
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
		func() *player.Player { return player.New("test") },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0.7 },
		func() bool { return true },
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
		func() *player.Player { return player.New("test") },
		func() float64 { return 0.2 },
		func() bool { return true },
		func() float64 { return 0 },
		func() bool { return false },
	)
	states := m.ComputeStates()
	if !states[BtnSkip].FillAlert {
		t.Error("Expected Skip FillAlert to be true when timer panic is active")
	}
}

func TestAgnosiaSetsAllScrambled(t *testing.T) {
	p := player.New("test")
	p.StatusEffects.AddImpairment(player.ImpairmentAgnosia)

	m := NewManager(
		func() int { return 2 },
		func() *player.Player { return p },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
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
	p.StatusEffects.AddImpairment(player.ImpairmentAphasia)

	m := NewManager(
		func() int { return 0 },
		func() *player.Player { return p },
		func() float64 { return 0 },
		func() bool { return false },
		func() float64 { return 0 },
		func() bool { return false },
	)
	states := m.ComputeStates()

	// Ensure at least one label differs from the base label at the same index
	diff := false
	base := [4]string{"MATCH", "SKIP", "TURN", "MENU"}
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
