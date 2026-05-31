package board

import (
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// ExitTile représente une tuile de sortie pour la navigation entre les zones.
type ExitTile struct {
	Direction entity.Direction
	Index     int
	State     entity.TileState
}

// GetHoverID retourne l'identifiant unique pour le survol.
func (e *ExitTile) GetHoverID() string {
	return fmt.Sprintf("exit_%s_%d", DirectionToName(e.Direction), e.Index)
}

// IsHoverAllowed vérifie si la tuile de sortie peut être survolée (non bloquée).
func (e *ExitTile) IsHoverAllowed() bool {
	return e.State&entity.Blocked == 0
}

// DirectionToName convertit une direction en sa représentation textuelle en anglais.
func DirectionToName(dir entity.Direction) string {
	switch dir {
	case entity.DirNorth:
		return "north"
	case entity.DirEast:
		return "east"
	case entity.DirSouth:
		return "south"
	case entity.DirWest:
		return "west"
	}
	return "unknown"
}

// CalculateFlipDirection détermine la direction de bascule basée sur la position du clic.
func CalculateFlipDirection(tileSize, localX, localY int) FlipDirection {
	return entity.CalculateFlipDirection(tileSize, localX, localY)
}
