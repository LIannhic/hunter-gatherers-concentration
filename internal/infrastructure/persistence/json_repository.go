package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
)

// JsonRepository implémente persistence.Repository en stockant des fichiers JSON sur le disque.
type JsonRepository struct {
	basePath string
}

// NewJsonRepository crée une nouvelle instance du dépôt JSON.
func NewJsonRepository(path string) *JsonRepository {
	// Crée le dossier s'il n'existe pas
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.MkdirAll(path, 0755)
	}
	return &JsonRepository{basePath: path}
}

// Save enregistre les données dans le slot spécifié.
func (r *JsonRepository) Save(slotID int, data *persistence.SaveData) error {
	if slotID < 1 || slotID > 3 {
		return fmt.Errorf("ID de slot invalide : %d (doit être entre 1 et 3)", slotID)
	}

	data.Meta.UpdatedAt = time.Now()
	data.Meta.SlotID = slotID

	path := r.getSlotPath(slotID)
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("erreur de sérialisation : %w", err)
	}

	return os.WriteFile(path, bytes, 0644)
}

// Load récupère les données d'un slot.
func (r *JsonRepository) Load(slotID int) (*persistence.SaveData, error) {
	path := r.getSlotPath(slotID)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("impossible de lire le slot %d : %w", slotID, err)
	}

	var data persistence.SaveData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, fmt.Errorf("erreur de désérialisation du slot %d : %w", slotID, err)
	}

	return &data, nil
}

// GetAllMetadata récupère uniquement les métadonnées de tous les slots existants.
func (r *JsonRepository) GetAllMetadata() ([]persistence.Metadata, error) {
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
func (r *JsonRepository) GetLatestSlotID() (int, error) {
	metas, err := r.GetAllMetadata()
	if err != nil || len(metas) == 0 {
		return 0, fmt.Errorf("aucune sauvegarde trouvée")
	}

	// Trie par date de mise à jour descendante
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.After(metas[j].UpdatedAt)
	})

	return metas[0].SlotID, nil
}

// Exists vérifie si un fichier de sauvegarde existe pour ce slot.
func (r *JsonRepository) Exists(slotID int) bool {
	_, err := os.Stat(r.getSlotPath(slotID))
	return err == nil
}

// Delete supprime le fichier de sauvegarde du slot.
func (r *JsonRepository) Delete(slotID int) error {
	if !r.Exists(slotID) {
		return nil
	}
	return os.Remove(r.getSlotPath(slotID))
}

func (r *JsonRepository) getSlotPath(slotID int) string {
	return filepath.Join(r.basePath, fmt.Sprintf("slot_%d.json", slotID))
}
