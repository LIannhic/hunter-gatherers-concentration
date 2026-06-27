package renderer

import (
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/actionbuttons"
)

// ButtonIconAnimType identifie le type d'animation des icônes de bouton
type ButtonIconAnimType int

const (
	ButtonAnimMatch ButtonIconAnimType = iota // MATCH/MERGE : approche puis merge ou éjection
	ButtonAnimEject                          // SKIP/TURN : rotation du bouton + éjection
)

// ButtonIconAnimState contient l'état calculé d'une animation d'icône pour une frame donnée
type ButtonIconAnimState struct {
	LeftX, LeftY     float64 // Position courante de l'icône gauche (écran)
	RightX, RightY   float64 // Position courante de l'icône droite (écran)
	LeftAlpha        float64 // Opacité de l'icône gauche (1=opaque, 0=invisible)
	RightAlpha       float64 // Opacité de l'icône droite
	LeftScale        float64 // Échelle de l'icône gauche
	RightScale       float64 // Échelle de l'icône droite
	ButtonRotation   float64 // Rotation du bouton en radians (SKIP/TURN uniquement)
	ButtonScaleX     float64 // Échelle horizontale du bouton (1.0 par défaut)
	ButtonScaleY     float64 // Échelle verticale du bouton (1.0 par défaut)
	ButtonOffsetX    float64 // Décalage X du bouton (pour rotation autour du centre)
	ButtonOffsetY    float64 // Décalage Y du bouton
	Finished         bool    // true si l'animation est terminée
}

// ButtonIconAnim représente une animation d'icônes sur un bouton d'action
type ButtonIconAnim struct {
	Type     ButtonIconAnimType
	ButtonID int
	IsValid  bool    // true=merge (valide), false=éjection (invalide)
	Progress float64 // 0.0 → 1.0
	Duration float64 // Durée totale en secondes

	// Entités à animer (pour récupérer les silhouettes)
	Entity1 domain.Entity
	Entity2 domain.Entity

	// Positions de départ des icônes (coordonnées écran absolues)
	LeftStartX, LeftStartY float64
	RightStartX, RightStartY float64
	CenterX, CenterY float64 // Centre du bouton (cible de rapprochement)

	//缓存 du dernier état calculé
	cachedState *ButtonIconAnimState
}

const (
	defaultButtonAnimDuration = 0.5 // secondes
	buttonApproachEnd         = 0.45 // fin de la phase d'approche (ratio)
	buttonEjectStart          = 0.35 // début de la phase d'éjection pour AnimEject
)

// StartButtonIconAnim démarre une animation d'icônes sur le bouton spécifié
func (r *BoardRenderer) StartButtonIconAnim(animType string, isValid bool, entity1ID, entity2ID string, targetButtonID actionbuttons.ButtonID, world *domain.World) {
	var animTypeConst ButtonIconAnimType
	switch animType {
	case "eject":
		animTypeConst = ButtonAnimEject
	default:
		animTypeConst = ButtonAnimMatch
	}

	buttonID := int(targetButtonID)
	if buttonID < 0 {
		switch animType {
		case "match":
			buttonID = 0
		case "merge":
			buttonID = 3
		case "eject":
			buttonID = 1
		}
	}

	// Résoudre les entités depuis les IDs
	var e1, e2 domain.Entity
	if world != nil {
		if entity1ID != "" {
			if ent, ok := world.Entities.Get(entity.ID(entity1ID)); ok {
				e1 = ent
			}
		}
		if entity2ID != "" {
			if ent, ok := world.Entities.Get(entity.ID(entity2ID)); ok {
				e2 = ent
			}
		}
	}

	// Calculer les positions de départ des icônes (coordonnées écran)
	// Ces positions correspondent au rendu statique dans renderSingleButton
	var leftPos, rightPos, centerPos [2]float64
	if r.ActionButtons != nil {
		states := r.ActionButtons.ComputeStates()
		if buttonID >= 0 && buttonID < len(states) {
			s := states[buttonID]
			leftPos[0] = s.CurrentX + float64(ui.ButtonIconLeftRelativeX) + float64(ui.ButtonIconSize)/2
			leftPos[1] = s.CurrentY + float64(ui.ButtonIconRelativeY) + float64(ui.ButtonIconSize)/2
			rightPos[0] = s.CurrentX + float64(ui.ButtonIconRelativeX) + float64(ui.ButtonIconSize)/2
			rightPos[1] = s.CurrentY + float64(ui.ButtonIconRelativeY) + float64(ui.ButtonIconSize)/2
			centerPos[0] = s.CurrentX + ui.ActionButtonW/2
			centerPos[1] = s.CurrentY + ui.ActionButtonH/2
		}
	}

	anim := &ButtonIconAnim{
		Type:          animTypeConst,
		ButtonID:      buttonID,
		IsValid:       isValid,
		Progress:      0.0,
		Duration:      defaultButtonAnimDuration,
		Entity1:       e1,
		Entity2:       e2,
		LeftStartX:    leftPos[0],
		LeftStartY:    leftPos[1],
		RightStartX:   rightPos[0],
		RightStartY:   rightPos[1],
		CenterX:       centerPos[0],
		CenterY:       centerPos[1],
		cachedState:   nil,
	}

	r.buttonIconAnims[buttonID] = anim
}

// UpdateButtonIconAnims avance toutes les animations d'icônes de bouton
func (r *BoardRenderer) UpdateButtonIconAnims(dt float64) {
	for id, anim := range r.buttonIconAnims {
		anim.Progress += dt / anim.Duration
		if anim.Progress >= 1.0 {
			anim.Progress = 1.0
		}
		anim.cachedState = nil // invalider le cache

		if anim.Progress >= 1.0 {
			delete(r.buttonIconAnims, id)
		}
	}
}

// IsButtonAnimating retourne true si une animation est active sur le bouton spécifié
func (r *BoardRenderer) IsButtonAnimating(buttonID int) bool {
	_, ok := r.buttonIconAnims[buttonID]
	return ok
}

// AnimatingButtonID retourne l'ID du bouton en animation, ou -1 si aucun
func (r *BoardRenderer) AnimatingButtonID() int {
	for id := range r.buttonIconAnims {
		return id
	}
	return -1
}

// GetButtonIconAnimState retourne l'état calculé de l'animation pour le bouton spécifié
func (r *BoardRenderer) GetButtonIconAnimState(buttonID int) *ButtonIconAnimState {
	anim, ok := r.buttonIconAnims[buttonID]
	if !ok {
		return nil
	}
	if anim.cachedState != nil {
		return anim.cachedState
	}

	state := computeButtonIconAnimState(anim)
	anim.cachedState = state
	return state
}

// computeButtonIconAnimState calcule l'état intermédiaire d'une animation
func computeButtonIconAnimState(anim *ButtonIconAnim) *ButtonIconAnimState {
	t := anim.Progress
	state := &ButtonIconAnimState{
		LeftAlpha:      1.0,
		RightAlpha:     1.0,
		LeftScale:      1.0,
		RightScale:     1.0,
		ButtonRotation: 0,
		ButtonScaleX:   1.0,
		ButtonScaleY:   1.0,
		ButtonOffsetX:  0,
		ButtonOffsetY:  0,
	}

	switch anim.Type {
	case ButtonAnimMatch:
		computeMatchState(anim, t, state)
	case ButtonAnimEject:
		computeEjectState(anim, t, state)
	}

	return state
}

// computeMatchState calcule l'état pour une animation MATCH/MERGE
func computeMatchState(anim *ButtonIconAnim, t float64, state *ButtonIconAnimState) {
	// Phase 1 : Approche (0 → buttonApproachEnd)
	// Phase 2 : Résolution (buttonApproachEnd → 1.0)

	if t <= buttonApproachEnd {
		// Phase d'approche : interpolation smoothstep vers le centre
		et := smoothstep(t / buttonApproachEnd)
		state.LeftX = anim.LeftStartX + (anim.CenterX-anim.LeftStartX)*et
		state.LeftY = anim.LeftStartY + (anim.CenterY-anim.LeftStartY)*et
		state.RightX = anim.RightStartX + (anim.CenterX-anim.RightStartX)*et
		state.RightY = anim.RightStartY + (anim.CenterY-anim.RightStartY)*et
		return
	}

	// Phase de résolution
	resolveT := (t - buttonApproachEnd) / (1.0 - buttonApproachEnd)
	resolveT = math.Min(resolveT, 1.0)

	if anim.IsValid {
		// Valide : superposition + fondu sortant
		// Les deux icônes restent au centre et disparaissent
		state.LeftX = anim.CenterX
		state.LeftY = anim.CenterY
		state.RightX = anim.CenterX
		state.RightY = anim.CenterY
		fadeT := smoothstep(resolveT)
		state.LeftAlpha = 1.0 - fadeT
		state.RightAlpha = 1.0 - fadeT
		state.LeftScale = 1.0 - fadeT*0.7  // réduit à 30%
		state.RightScale = 1.0 - fadeT*0.7
	} else {
		// Invalide : éjection vers l'extérieur + fondu sortante
		// Calcul de la direction d'éjection (éloignement du centre)
		leftDirX := anim.LeftStartX - anim.CenterX
		leftDirY := anim.LeftStartY - anim.CenterY
		rightDirX := anim.RightStartX - anim.CenterX
		rightDirY := anim.RightStartY - anim.CenterY

		// Normaliser
		leftLen := math.Sqrt(leftDirX*leftDirX + leftDirY*leftDirY)
		rightLen := math.Sqrt(rightDirX*rightDirX + rightDirY*rightDirY)
		if leftLen < 1 {
			leftLen = 1
		}
		if rightLen < 1 {
			rightLen = 1
		}
		leftDirX /= leftLen
		leftDirY /= leftLen
		rightDirX /= rightLen
		rightDirY /= rightLen

		// Distance d'éjection : dépasse les bords du bouton (~40px)
		ejectDist := 40.0
		easeT := smoothstep(resolveT)

		state.LeftX = anim.CenterX + leftDirX*ejectDist*easeT
		state.LeftY = anim.CenterY + leftDirY*ejectDist*easeT
		state.RightX = anim.CenterX + rightDirX*ejectDist*easeT
		state.RightY = anim.CenterY + rightDirY*ejectDist*easeT

		// Fondu sortant
		state.LeftAlpha = 1.0 - easeT
		state.RightAlpha = 1.0 - easeT
	}
}

// computeEjectState calcule l'état pour une animation SKIP/TURN (rotation + éjection)
func computeEjectState(anim *ButtonIconAnim, t float64, state *ButtonIconAnimState) {
	// Phase 1 : Rotation du bouton (0 → buttonEjectStart)
	// Phase 2 : Éjection des silhouettes + rotation continue

	if t <= buttonEjectStart {
		// Phase de rotation initiale : les silhouettes restent en place
		// Le bouton pivote de 0 à 90°
		rotT := smoothstep(t / buttonEjectStart)
		state.ButtonRotation = rotT * math.Pi / 2 // 0 → 90°

		// Les silhouettes restent à leurs positions initiales
		state.LeftX = anim.LeftStartX
		state.LeftY = anim.LeftStartY
		state.RightX = anim.RightStartX
		state.RightY = anim.RightStartY
		return
	}

	// Phase d'éjection
	resolveT := (t - buttonEjectStart) / (1.0 - buttonEjectStart)
	resolveT = math.Min(resolveT, 1.0)
	easeT := smoothstep(resolveT)

	// Rotation continue : 90° → 360°
	state.ButtonRotation = math.Pi/2 + easeT*3*math.Pi/2

	// Éjection des silhouettes vers l'extérieur
	leftDirX := anim.LeftStartX - anim.CenterX
	leftDirY := anim.LeftStartY - anim.CenterY
	rightDirX := anim.RightStartX - anim.CenterX
	rightDirY := anim.RightStartY - anim.CenterY

	leftLen := math.Sqrt(leftDirX*leftDirX + leftDirY*leftDirY)
	rightLen := math.Sqrt(rightDirX*rightDirX + rightDirY*rightDirY)
	if leftLen < 1 {
		leftLen = 1
	}
	if rightLen < 1 {
		rightLen = 1
	}
	leftDirX /= leftLen
	leftDirY /= leftLen
	rightDirX /= rightLen
	rightDirY /= rightLen

	ejectDist := 50.0
	state.LeftX = anim.LeftStartX + leftDirX*ejectDist*easeT
	state.LeftY = anim.LeftStartY + leftDirY*ejectDist*easeT
	state.RightX = anim.RightStartX + rightDirX*ejectDist*easeT
	state.RightY = anim.RightStartY + rightDirY*ejectDist*easeT

	// Fondu sortant
	state.LeftAlpha = 1.0 - easeT
	state.RightAlpha = 1.0 - easeT
}
