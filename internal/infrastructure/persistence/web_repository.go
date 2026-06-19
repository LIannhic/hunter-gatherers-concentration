//go:build js

package persistence

import (
	"encoding/json"
	"fmt"
	"sort"
	"syscall/js"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
)

// WebRepository implémente persistence.Repository en utilisant le LocalStorage du navigateur.
type WebRepository struct {
	storage js.Value
}

// NewWebRepository crée une nouvelle instance du dépôt Web.
func NewWebRepository() *WebRepository {
	window := js.Global().Get("window")
	if window.IsUndefined() {
		return nil
	}
	storage := window.Get("localStorage")
	return &WebRepository{storage: storage}
}

// Save enregistre les données dans le slot spécifié via LocalStorage.
func (r *WebRepository) Save(slotID int, data *persistence.SaveData) error {
	if slotID < 1 || slotID > 3 {
		return fmt.Errorf("ID de slot invalide : %d", slotID)
	}

	data.Meta.UpdatedAt = time.Now()
	data.Meta.SlotID = slotID

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("erreur de sérialisation : %w", err)
	}

	key := r.getSlotKey(slotID)
	r.storage.Call("setItem", key, string(bytes))
	return nil
}

// Load récupère les données d'un slot depuis LocalStorage.
func (r *WebRepository) Load(slotID int) (*persistence.SaveData, error) {
	key := r.getSlotKey(slotID)
	item := r.storage.Call("getItem", key)
	if item.IsNull() || item.IsUndefined() {
		return nil, fmt.Errorf("aucune sauvegarde pour le slot %d", slotID)
	}

	var data persistence.SaveData
	if err := json.Unmarshal([]byte(item.String()), &data); err != nil {
		return nil, fmt.Errorf("erreur de désérialisation du slot %d : %w", slotID, err)
	}

	return &data, nil
}

// GetAllMetadata récupère les métadonnées de tous les slots.
func (r *WebRepository) GetAllMetadata() ([]persistence.Metadata, error) {
	var metas []persistence.Metadata
	for i := 1; i <= 3; i++ {
		if r.Exists(i) {
			data, err := r.Load(i)
			if err == nil {
				metas = append(metas, data.Meta)
			}
		}
	}
	return metas, nil
}

// GetLatestSlotID retourne l'ID du slot le plus récent.
func (r *WebRepository) GetLatestSlotID() (int, error) {
	metas, err := r.GetAllMetadata()
	if err != nil || len(metas) == 0 {
		return 0, fmt.Errorf("aucune sauvegarde trouvée")
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.After(metas[j].UpdatedAt)
	})

	return metas[0].SlotID, nil
}

// Exists vérifie si une sauvegarde existe dans LocalStorage.
func (r *WebRepository) Exists(slotID int) bool {
	key := r.getSlotKey(slotID)
	item := r.storage.Call("getItem", key)
	return !item.IsNull() && !item.IsUndefined()
}

// Delete supprime une sauvegarde de LocalStorage.
func (r *WebRepository) Delete(slotID int) error {
	key := r.getSlotKey(slotID)
	r.storage.Call("removeItem", key)
	return nil
}

func (r *WebRepository) getSlotKey(slotID int) string {
	return fmt.Sprintf("hgc_slot_%d", slotID)
}
