package board

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Position et Direction mappées depuis le domaine pour faciliter l'accès.
type Position = entity.Position
type Direction = entity.Direction

const (
	North = entity.DirNorth
	East  = entity.DirEast
	South = entity.DirSouth
	West  = entity.DirWest
)

// FlipDirection représente le sens de bascule d'une tuile.
type FlipDirection = entity.FlipDirection

const (
	FlipTop         = entity.FlipTop
	FlipTopRight    = entity.FlipTopRight
	FlipRight       = entity.FlipRight
	FlipBottomRight = entity.FlipBottomRight
	FlipBottom      = entity.FlipBottom
	FlipBottomLeft  = entity.FlipBottomLeft
	FlipLeft        = entity.FlipLeft
	FlipTopLeft     = entity.FlipTopLeft
	FlipCenter      = entity.FlipCenter
)

// Hoverable interface pour les éléments interactifs (tuiles, sorties, butin).
type Hoverable interface {
	GetHoverID() string
	IsHoverAllowed() bool
}

// InventoryGridID est l'identifiant réservé pour la grille d'inventaire logicielle.
const InventoryGridID = "inventory"
