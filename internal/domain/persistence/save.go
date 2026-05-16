package persistence

import (
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

// Metadata contient les informations de suivi pour l'affichage du menu et les statistiques globales
type Metadata struct {
	SlotID         int       `json:"slot_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SessionCount   int       `json:"session_count"` // Nombre d'expéditions lancées
	DeathCount     int       `json:"death_count"`   // Nombre d'échecs
	TotalPlaytime  float64   `json:"total_playtime"` // Temps total en secondes
	MaxScore       int       `json:"max_score"`
	LastScore      int       `json:"last_score"`
}

// SaveData regroupe l'état complet du joueur et de sa progression méta
type SaveData struct {
	Meta   Metadata       `json:"meta"`
	Hub    *meta.Hub      `json:"hub"`    // État du foyer et progression
	Player *player.Player `json:"player"` // État du personnage (Stats, Skills, etc.)
}

// NewSaveData crée une nouvelle structure de sauvegarde initialisée pour un slot
func NewSaveData(slotID int) *SaveData {
	return &SaveData{
		Meta: Metadata{
			SlotID:    slotID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Hub:    meta.NewHub(),
		Player: player.New("player_default"),
	}
}
