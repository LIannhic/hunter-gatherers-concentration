package i18n

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets/locales"
)

// Manager gère le chargement et la récupération des traductions.
type Manager struct {
	mu           sync.RWMutex
	currentLang  string
	fallbackLang string
	locales      map[string]map[string]string
}

// Global default manager instance for ease of use across UI components
var Default *Manager

func init() {
	Default = NewManager("en")
	if enData, err := locales.FS.ReadFile("en.json"); err == nil {
		_ = Default.LoadLocale("en", enData)
	}
	if frData, err := locales.FS.ReadFile("fr.json"); err == nil {
		_ = Default.LoadLocale("fr", frData)
	}
}

// NewManager crée une nouvelle instance de Manager.
func NewManager(fallbackLang string) *Manager {
	return &Manager{
		currentLang:  fallbackLang,
		fallbackLang: fallbackLang,
		locales:      make(map[string]map[string]string),
	}
}

// LoadLocale charge un dictionnaire JSON de traduction pour une langue donnée.
func (m *Manager) LoadLocale(lang string, jsonData []byte) error {
	var kv map[string]string
	if err := json.Unmarshal(jsonData, &kv); err != nil {
		return fmt.Errorf("failed to parse locale %s: %w", lang, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.locales[lang] = kv
	return nil
}

// SetLanguage change la langue active.
func (m *Manager) SetLanguage(lang string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentLang = lang
}

// GetLanguage retourne la langue active.
func (m *Manager) GetLanguage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentLang
}

// T retourne la traduction correspondant à la clé donnée dans la langue active.
// Si la clé n'existe pas dans la langue active, utilise la langue de secours.
// Si des arguments sont fournis, applique fmt.Sprintf.
func (m *Manager) T(key string, args ...any) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var str string
	if dict, ok := m.locales[m.currentLang]; ok {
		str = dict[key]
	}
	if str == "" {
		if dict, ok := m.locales[m.fallbackLang]; ok {
			str = dict[key]
		}
	}
	if str == "" {
		str = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(str, args...)
	}
	return str
}

// T est un raccourci global vers Default.T.
func T(key string, args ...any) string {
	if Default == nil {
		if len(args) > 0 {
			return fmt.Sprintf(key, args...)
		}
		return key
	}
	return Default.T(key, args...)
}
