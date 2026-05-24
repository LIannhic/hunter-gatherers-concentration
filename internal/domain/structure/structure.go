package structure

import (
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Orientation représente l'orientation cardinale d'une structure
type Orientation = entity.Direction

const (
	North = entity.DirNorth
	East  = entity.DirEast
	South = entity.DirSouth
	West  = entity.DirWest
)

// NavType définit le type de structure de navigation
type NavType string

const (
	NavNorthRight NavType = "north-right"
	NavNorthLeft  NavType = "north-left"
	NavEastTop    NavType = "east-top"
	NavEastBottom NavType = "east-bottom"
	NavSouthRight NavType = "south-right"
	NavSouthLeft  NavType = "south-left"
	NavWestTop    NavType = "west-top"
	NavWestBottom NavType = "west-bottom"
)

type Structure struct {
	entity.BaseEntity
	SType string
}

func NewStructure(stype string, pos entity.Position) *Structure {
	s := &Structure{
		BaseEntity: entity.NewBaseEntity(entity.TypeStructure),
		SType:      stype,
	}
	s.SetPosition(pos)
	s.AddTag(stype)

	// Logique de visibilité initiale
	switch stype {
	case "commencement_portal":
		s.SetState(entity.Revealed)
	case "finish_portal":
		s.SetState(entity.Hidden)
	default:
		s.SetState(entity.Revealed)
	}
	return s
}

// NavigationStructure représente la logique métier d'une tuile de navigation
type NavigationStructure struct {
	entity.BaseEntity
	NavType    NavType
	BaseOrient Orientation
}

func NewNavigation(navType NavType, orient Orientation) *NavigationStructure {
	s := &NavigationStructure{
		BaseEntity: entity.NewBaseEntity(entity.TypeStructure),
		NavType:    navType,
		BaseOrient: orient,
	}
	s.AddTag("navigation")
	s.AddTag(string(navType))
	return s
}
