package player

import "testing"

func TestNewStatusEffects(t *testing.T) {
	s := NewStatusEffects()
	if s.HasAnyImpairment() {
		t.Error("New status effects should have no impairments")
	}
}

func TestAddRemoveImpairment(t *testing.T) {
	s := NewStatusEffects()
	s.AddImpairment(ImpairmentAtaxia)
	if !s.HasImpairment(ImpairmentAtaxia) {
		t.Error("Should have ataxia")
	}
	if !s.HasAnyImpairment() {
		t.Error("Should have any impairment")
	}
	s.RemoveImpairment(ImpairmentAtaxia)
	if s.HasImpairment(ImpairmentAtaxia) {
		t.Error("Should not have ataxia after removal")
	}
}

func TestGetActiveImpairments(t *testing.T) {
	s := NewStatusEffects()
	s.AddImpairment(ImpairmentAphasia)
	s.AddImpairment(ImpairmentAmnesia)
	imps := s.GetActiveImpairments()
	if len(imps) != 2 {
		t.Fatalf("Expected 2 impairments, got %d", len(imps))
	}
}

func TestClearImpairments(t *testing.T) {
	s := NewStatusEffects()
	s.AddImpairment(ImpairmentAgnosia)
	s.Clear()
	if s.HasAnyImpairment() {
		t.Error("Should have no impairments after clear")
	}
}
