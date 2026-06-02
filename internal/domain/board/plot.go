package board

import (
	"fmt"
)

// Slope représente l'inclinaison logique de la Parcelle (Vent, Courant, Piste...)
type Slope int

const (
	SlopeTop Slope = iota
	SlopeTopRight
	SlopeRight
	SlopeBottomRight
	SlopeBottom
	SlopeBottomLeft
	SlopeLeft
	SlopeTopLeft
	SlopeFlat // État neutre / Plat
)

// ToFlipDirection convertit une pente en direction de bascule graphique.
func (s Slope) ToFlipDirection() FlipDirection {
	switch s {
	case SlopeTop:
		return FlipTop
	case SlopeTopRight:
		return FlipTopRight
	case SlopeRight:
		return FlipRight
	case SlopeBottomRight:
		return FlipBottomRight
	case SlopeBottom:
		return FlipBottom
	case SlopeBottomLeft:
		return FlipBottomLeft
	case SlopeLeft:
		return FlipLeft
	case SlopeTopLeft:
		return FlipTopLeft
	default:
		return FlipCenter
	}
}

// Bearing représente l'orientation de la Grille (Cardinaux)
type Bearing int

const (
	BearingNorth Bearing = iota
	BearingEast
	BearingSouth
	BearingWest
	BearingMirror
)

// Plot représente une cellule individuelle sur le plateau de jeu.
type Plot struct {
	Position    Position
	EntitiesID  []string
	StructureID string
	Empty       bool
	LocalStage  SuccessionStage
	Tilt        Slope
	Modifier    PlotModifier
}

// PushEntity ajoute une entité au sommet de la pile de la parcelle.
func (p *Plot) PushEntity(id string) {
	p.EntitiesID = append(p.EntitiesID, id)
}

// PushEntityToBottom ajoute une entité au bas de la pile (ex: traces).
func (p *Plot) PushEntityToBottom(id string) {
	p.EntitiesID = append([]string{id}, p.EntitiesID...)
}

// PopEntity retire et retourne l'entité au sommet de la pile.
func (p *Plot) PopEntity() (string, bool) {
	if len(p.EntitiesID) == 0 {
		return "", false
	}
	lastIdx := len(p.EntitiesID) - 1
	id := p.EntitiesID[lastIdx]
	p.EntitiesID = p.EntitiesID[:lastIdx]
	return id, true
}

func (p *Plot) String() string {
	return fmt.Sprintf("Plot[%v entities=%v]", p.Position, p.EntitiesID)
}

// PlotModifier contient des attributs altérant le comportement ou l'affichage d'une parcelle.
type PlotModifier struct {
	Concealed    bool
	Obstructed   bool
	LuminousHint bool
}
