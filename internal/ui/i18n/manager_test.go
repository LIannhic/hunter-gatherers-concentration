package i18n

import (
	"testing"
)

func TestManager(t *testing.T) {
	m := NewManager("en")

	enJSON := []byte(`{
		"menu.start": "START",
		"hud.score": "Score: %d"
	}`)

	frJSON := []byte(`{
		"menu.start": "DÉMARRER",
		"hud.score": "Score : %d"
	}`)

	if err := m.LoadLocale("en", enJSON); err != nil {
		t.Fatalf("Failed to load en locale: %v", err)
	}
	if err := m.LoadLocale("fr", frJSON); err != nil {
		t.Fatalf("Failed to load fr locale: %v", err)
	}

	// Default language is "en"
	if got := m.T("menu.start"); got != "START" {
		t.Errorf("Expected 'START', got %q", got)
	}
	if got := m.T("hud.score", 100); got != "Score: 100" {
		t.Errorf("Expected 'Score: 100', got %q", got)
	}

	// Switch to French
	m.SetLanguage("fr")
	if got := m.T("menu.start"); got != "DÉMARRER" {
		t.Errorf("Expected 'DÉMARRER', got %q", got)
	}
	if got := m.T("hud.score", 100); got != "Score : 100" {
		t.Errorf("Expected 'Score : 100', got %q", got)
	}

	// Key missing in "fr", fallback to "en"
	missingInFrJSON := []byte(`{}`)
	m.LoadLocale("fr", missingInFrJSON)
	if got := m.T("menu.start"); got != "START" {
		t.Errorf("Expected fallback 'START', got %q", got)
	}

	// Key missing in all languages, returns key
	if got := m.T("unknown.key"); got != "unknown.key" {
		t.Errorf("Expected 'unknown.key', got %q", got)
	}
}
