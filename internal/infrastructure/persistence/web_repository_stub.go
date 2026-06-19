//go:build !js

package persistence

import (
	"fmt"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
)

// WebRepository stub for non-WASM builds
type WebRepository struct{}

func NewWebRepository() *WebRepository {
	return nil
}

func (r *WebRepository) Save(slotID int, data *persistence.SaveData) error {
	return fmt.Errorf("WebRepository is only available in WASM builds")
}

func (r *WebRepository) Load(slotID int) (*persistence.SaveData, error) {
	return nil, fmt.Errorf("WebRepository is only available in WASM builds")
}

func (r *WebRepository) GetAllMetadata() ([]persistence.Metadata, error) {
	return nil, fmt.Errorf("WebRepository is only available in WASM builds")
}

func (r *WebRepository) GetLatestSlotID() (int, error) {
	return 0, fmt.Errorf("WebRepository is only available in WASM builds")
}

func (r *WebRepository) Exists(slotID int) bool {
	return false
}

func (r *WebRepository) Delete(slotID int) error {
	return fmt.Errorf("WebRepository is only available in WASM builds")
}
