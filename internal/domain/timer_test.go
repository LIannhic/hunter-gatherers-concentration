package domain

import "testing"

func TestNewTurnTimer(t *testing.T) {
	timer := NewTurnTimer(10.0)
	if timer.MaxTime != 10.0 {
		t.Errorf("Expected MaxTime 10.0, got %v", timer.MaxTime)
	}
	if timer.Remaining != 10.0 {
		t.Errorf("Expected Remaining 10.0, got %v", timer.Remaining)
	}
	if timer.Running {
		t.Error("New timer should not be running")
	}
}

func TestTurnTimerUpdate(t *testing.T) {
	timer := NewTurnTimer(10.0)
	timer.Start()

	expired := timer.Update(5.0)
	if expired {
		t.Error("Should not be expired after 5s")
	}
	if timer.Remaining != 5.0 {
		t.Errorf("Expected Remaining 5.0, got %v", timer.Remaining)
	}

	expired = timer.Update(5.0)
	if !expired {
		t.Error("Should be expired after 10s total")
	}
	if timer.Remaining != 0 {
		t.Errorf("Expected Remaining 0, got %v", timer.Remaining)
	}
}

func TestTurnTimerReset(t *testing.T) {
	timer := NewTurnTimer(8.0)
	timer.Start()
	timer.Update(3.0)
	timer.Reset()

	if timer.Remaining != 8.0 {
		t.Errorf("Expected Remaining 8.0 after reset, got %v", timer.Remaining)
	}
	if !timer.Running {
		t.Error("Timer should be running after reset")
	}
}

func TestTurnTimerProgress(t *testing.T) {
	timer := NewTurnTimer(10.0)
	timer.Start()
	timer.Update(2.5)

	p := timer.Progress()
	expected := 0.25
	if p != expected {
		t.Errorf("Expected progress %v, got %v", expected, p)
	}
}

func TestTurnTimerIsPanic(t *testing.T) {
	timer := NewTurnTimer(10.0)
	timer.Start()
	timer.Update(7.5)

	if !timer.IsPanic() {
		t.Error("Should be in panic with 2.5s remaining")
	}

	timer.Reset()
	timer.Update(5.0)
	if timer.IsPanic() {
		t.Error("Should not be in panic with 5s remaining")
	}
}

func TestTurnTimerSetMaxTime(t *testing.T) {
	timer := NewTurnTimer(10.0)
	timer.Start()
	timer.Update(3.0)

	timer.SetMaxTime(5.0)
	if timer.MaxTime != 5.0 {
		t.Errorf("Expected MaxTime 5.0, got %v", timer.MaxTime)
	}
	// Remaining était 7.0, supérieur au nouveau max → clamp à 5.0
	if timer.Remaining != 5.0 {
		t.Errorf("Expected Remaining clamped to 5.0, got %v", timer.Remaining)
	}

	// Baisse le max en dessous du remaining actuel
	timer.SetMaxTime(4.0)
	if timer.Remaining != 4.0 {
		t.Errorf("Expected Remaining clamped to 4.0, got %v", timer.Remaining)
	}
}
