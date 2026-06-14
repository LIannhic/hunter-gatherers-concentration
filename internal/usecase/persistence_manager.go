package usecase

import (
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

// PersistenceManager gère la logique de haut niveau pour les sauvegardes
type PersistenceManager struct {
	repo        persistence.Repository
	currentSlot int
}

func NewPersistenceManager(repo persistence.Repository) *PersistenceManager {
	return &PersistenceManager{
		repo: repo,
	}
}

// GetCurrentSlot retourne le slot actif
func (m *PersistenceManager) GetCurrentSlot() int {
	return m.currentSlot
}

// SetCurrentSlot définit le slot actif
func (m *PersistenceManager) SetCurrentSlot(slotID int) {
	m.currentSlot = slotID
}

// CreateNewGame initialise un nouveau slot (Action "New Game")
func (m *PersistenceManager) CreateNewGame(slotID int) (*persistence.SaveData, error) {
	newData := persistence.NewSaveData(slotID)
	err := m.repo.Save(slotID, newData)
	if err == nil {
		m.currentSlot = slotID
	}
	return newData, err
}

// LoadGame charge une partie (Action "Load")
func (m *PersistenceManager) LoadGame(slotID int) (*persistence.SaveData, error) {
	data, err := m.repo.Load(slotID)
	if err == nil {
		m.currentSlot = slotID
	}
	return data, err
}

// LoadLatestGame charge la dernière sauvegarde (Action "Continue")
func (m *PersistenceManager) LoadLatestGame() (*persistence.SaveData, error) {
	slotID, err := m.repo.GetLatestSlotID()
	if err != nil {
		return nil, err
	}
	return m.LoadGame(slotID)
}

// IncrementSessionCount incrémente le compteur d'expéditions dès le lancement.
func (m *PersistenceManager) IncrementSessionCount(slotID int) error {
	save, err := m.repo.Load(slotID)
	if err != nil {
		return err
	}
	save.Meta.SessionCount++
	return m.repo.Save(slotID, save)
}

// SaveCurrentGame met à jour les métadonnées de session sans persister l'état de l'expédition en cours.
func (m *PersistenceManager) SaveCurrentGame(hub *meta.Hub, p *player.Player, difficulty string, sessionDuration float64) error {
	if m.currentSlot == 0 {
		return fmt.Errorf("aucun slot actif")
	}

	save, err := m.repo.Load(m.currentSlot)
	if err != nil {
		return err
	}

	// On ne conserve pas l'état du monde ni du joueur : chaque expédition est nouvelle.
	save.Meta.Difficulty = difficulty
	save.Meta.TotalPlaytime += sessionDuration

	// Mise à jour du score (basé sur l'XP totale)
	save.Meta.LastScore = p.Stats.TotalExperience
	if save.Meta.LastScore > save.Meta.MaxScore {
		save.Meta.MaxScore = save.Meta.LastScore
	}

	return m.repo.Save(m.currentSlot, save)
}

// HandleDeath implémente la logique de "Fail State" persistante
func (m *PersistenceManager) HandleDeath(hub *meta.Hub, p *player.Player, difficulty string, sessionDuration float64) error {
	if m.currentSlot == 0 {
		return fmt.Errorf("aucun slot de sauvegarde actif")
	}

	save, err := m.repo.Load(m.currentSlot)
	if err != nil {
		return err
	}

	// Met à jour les statistiques de mort et les méta-données
	save.Meta.DeathCount++
	save.Meta.Difficulty = difficulty
	save.Meta.TotalPlaytime += sessionDuration

	// Mise à jour du score même en cas de mort
	save.Meta.LastScore = p.Stats.TotalExperience
	if save.Meta.LastScore > save.Meta.MaxScore {
		save.Meta.MaxScore = save.Meta.LastScore
	}

	// On ne sauvegarde PAS l'état actuel du monde/joueur (qui est mort)
	// mais on sauvegarde la progression méta acquise.

	return m.repo.Save(m.currentSlot, save)
}

// GetSaveSummaries retourne les métadonnées pour le menu
func (m *PersistenceManager) GetSaveSummaries() ([]persistence.Metadata, error) {
	return m.repo.GetAllMetadata()
}

// DeleteSave supprime un slot
func (m *PersistenceManager) DeleteSave(slotID int) error {
	return m.repo.Delete(slotID)
}
