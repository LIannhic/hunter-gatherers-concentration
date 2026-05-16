package persistence

// Repository définit les opérations de persistance autorisées par le domaine.
// L'implémentation technique (JSON, etc.) se trouve dans la couche infrastructure.
type Repository interface {
	// Save enregistre les données dans le slot spécifié (1, 2 ou 3).
	Save(slotID int, data *SaveData) error

	// Load récupère les données d'un slot.
	Load(slotID int) (*SaveData, error)

	// Delete supprime définitivement un slot.
	Delete(slotID int) error

	// GetAllMetadata récupère uniquement les en-têtes pour l'affichage du menu de sélection.
	GetAllMetadata() ([]Metadata, error)

	// GetLatestSlotID retourne l'ID du slot le plus récemment mis à jour (pour le bouton "Continuer").
	GetLatestSlotID() (int, error)

	// Exists vérifie si un slot contient déjà une sauvegarde.
	Exists(slotID int) bool
}
