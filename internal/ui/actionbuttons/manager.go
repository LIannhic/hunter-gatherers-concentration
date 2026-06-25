package actionbuttons

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
)

// ButtonID identifie un des 4 boutons d'action
type ButtonID int

const (
	BtnMatch ButtonID = iota
	BtnSkip
	BtnEndTurn
	BtnMerge
)

const btnCount = 4

// ButtonState représente l'état calculé d'un bouton pour la frame courante
type ButtonState struct {
	ID           ButtonID
	Label        string
	Active       bool
	X, Y         float64 // Coordonnées cibles
	CurrentX     float64 // Coordonnées réelles pour l'animation
	CurrentY     float64 //
	Width        float64
	Height       float64
	Scrambled    bool    // Vrai si les coordonnées ont été altérées par un trouble
	FillProgress float64 // 0.0 → 1.0, remplissage temporel (Skip uniquement)
	FillAlert    bool    // Vrai si le timer est en phase critique ou expiré
	TextScale    float64 // Facteur d'échelle du texte (Aphasia)
}

// Manager gère l'état réactif des 4 boutons d'action du Playmat.
// Il est purement réactif : l'état des boutons est recalculé à chaque frame
// en fonction de l'état de la grille et des troubles cognitifs du joueur.
type Manager struct {
	// Références externes
	getRevealedTileCount func() int
	getPlayer            func() *player.Player
	getTimerProgress     func() float64 // 0.0 → 1.0 (temps écoulé)
	getTimerPanic        func() bool    // true si < 3s restantes
	getVictoryProgress   func() float64 // 0.0 → 1.0 (V0.2)
	isVictoryActive      func() bool    // true si portail déployé (V0.2)

	// Base coordinates (fixed by UI spec)
	baseCoords [btnCount]struct{ x, y float64 }
	baseLabels [btnCount]string

	// Animation state
	currentPos   [btnCount]struct{ x, y float64 }
	glitchLabels [btnCount]string
	lastGlitch   time.Time
	startTime    time.Time
	jumpTime     [btnCount]float64 // Temps restant pour le "saut" de whack-a-mole
	jumpTarget   [btnCount]int     // Index de la baseCoord cible

	// Cache du frame précédent pour stabilité visuelle
	lastScrambleSeed int64
	scrambleMapping  [btnCount]int // index -> permuted index
}

// NewManager crée un nouveau gestionnaire de boutons d'action.
// getRevealedTileCount doit retourner le nombre de tuiles révélées ce tour.
// getPlayer doit retourner le joueur courant (pour les StatusEffects).
// getTimerProgress / getTimerPanic fournissent l'état du compte à rebours temps réel.
func NewManager(getRevealedTileCount func() int, getPlayer func() *player.Player, getTimerProgress func() float64, getTimerPanic func() bool, getVictoryProgress func() float64, isVictoryActive func() bool) *Manager {
	m := &Manager{
		getRevealedTileCount: getRevealedTileCount,
		getPlayer:            getPlayer,
		getTimerProgress:     getTimerProgress,
		getTimerPanic:        getTimerPanic,
		getVictoryProgress:   getVictoryProgress,
		isVictoryActive:      isVictoryActive,
		baseCoords: [btnCount]struct{ x, y float64 }{
			{ui.ActionBtn1X, ui.ActionBtn1Y},
			{ui.ActionBtn2X, ui.ActionBtn2Y},
			{ui.ActionBtn3X, ui.ActionBtn3Y},
			{ui.ActionBtn4X, ui.ActionBtn4Y},
		},
		baseLabels: [btnCount]string{"MATCH", "SKIP", "TURN", "MERGE"},
		startTime:  time.Now(),
	}
	for i := 0; i < btnCount; i++ {
		m.currentPos[i].x = ui.PlaymatX + m.baseCoords[i].x
		m.currentPos[i].y = ui.PlaymatY + m.baseCoords[i].y
	}
	m.resetScramble()
	return m
}

// resetScramble réinitialise la permutation à l'identité
func (m *Manager) resetScramble() {
	for i := 0; i < btnCount; i++ {
		m.scrambleMapping[i] = i
	}
}

// ComputeStates recalcule l'état complet des 4 boutons pour la frame courante.
// Cette méthode est purement réactive : aucun état interne persistant n'est modifié
// hormis le cache de scrambling pour éviter le clignotement épileptique.
func (m *Manager) ComputeStates() [btnCount]ButtonState {
	revealedCount := m.getRevealedTileCount()
	p := m.getPlayer()
	elapsed := time.Since(m.startTime).Seconds()

	var states [btnCount]ButtonState
	for i := 0; i < btnCount; i++ {
		states[i] = ButtonState{
			ID:     ButtonID(i),
			Label:  m.baseLabels[i],
			Active: false,
			X:      ui.PlaymatX + m.baseCoords[i].x,
			Y:      ui.PlaymatY + m.baseCoords[i].y,
			Width:  ui.ActionButtonW,
			Height: ui.ActionButtonH,
		}
	}

	// --- RÈGLE MÉTIER : Activation selon le nombre de tuiles révélées ---
	if revealedCount >= 2 {
		states[BtnSkip].Active = true
		states[BtnMerge].Active = true
		states[BtnMatch].Active = true
	}
	states[BtnEndTurn].Active = true

	// --- V0.2 : TRANSITION END GAME ---
	if m.isVictoryActive != nil && m.isVictoryActive() {
		states[BtnEndTurn].Label = "END GAME"
		if m.getVictoryProgress != nil {
			states[BtnEndTurn].FillProgress = m.getVictoryProgress()
			states[BtnEndTurn].FillAlert = true
		}
	}

	// --- FEEDBACK TEMPS RÉEL : Remplissage du bouton Skip ---
	if m.getTimerProgress != nil {
		states[BtnSkip].FillProgress = m.getTimerProgress()
		states[BtnSkip].FillAlert = states[BtnSkip].FillProgress >= 1.0
	}
	if m.getTimerPanic != nil && m.getTimerPanic() {
		states[BtnSkip].FillAlert = true
	}

	// --- TROUBLES COGNITIFS ---
	if p != nil && (p.AphasiaTurns > 0 || p.AtaxiaTurns > 0 || p.AgnosiaTurns > 0 || p.AmnesiaTurns > 0) {
		m.applyImpairments(p, &states, elapsed)
	} else {
		m.resetScramble()
		// Reset jump states
		for i := 0; i < btnCount; i++ {
			m.jumpTime[i] = 0
		}
	}

	// --- ANIMATION ET INTERPOLATION ---
	dt := 0.15 // Vitesse de l'interpolation
	for i := 0; i < btnCount; i++ {
		targetX, targetY := states[i].X, states[i].Y

		speed := dt
		if p != nil && p.AtaxiaTurns > 0 {
			// Ataxia utilise les positions calculées par applyImpairments (Whack-a-mole)
			speed = 0.2 // Plus réactif pour les sauts
		}

		m.currentPos[i].x += (targetX - m.currentPos[i].x) * speed
		m.currentPos[i].y += (targetY - m.currentPos[i].y) * speed

		states[i].CurrentX = m.currentPos[i].x
		states[i].CurrentY = m.currentPos[i].y
		states[i].TextScale = 1.0

		// Pulsation pour Aphasia (Désynchronisée par bouton)
		if p != nil && p.AphasiaTurns > 0 {
			states[i].TextScale = 1.0 + 0.15*math.Sin(elapsed*7.0+float64(i)*1.5)
		}
	}

	return states
}

// applyImpairments modifie les coordonnées et/ou labels des boutons
func (m *Manager) applyImpairments(p *player.Player, states *[btnCount]ButtonState, elapsed float64) {
	// Ataxia : Whack-a-mole dynamique
	if p.AtaxiaTurns > 0 {
		for i := 0; i < btnCount; i++ {
			// Si le timer de saut est expiré, on choisit une nouvelle cible
			if elapsed > m.jumpTime[i] {
				m.jumpTarget[i] = rand.Intn(btnCount)
				// Dure entre 1.5 et 3 secondes
				m.jumpTime[i] = elapsed + 1.5 + rand.Float64()*1.5
			}

			// Destination cible basée sur le whack-a-mole
			targetBase := m.baseCoords[m.jumpTarget[i]]
			states[i].X = ui.PlaymatX + targetBase.x
			states[i].Y = ui.PlaymatY + targetBase.y

			// Animation de "saut" : on s'éloigne du centre du playmat avant de revenir
			remaining := m.jumpTime[i] - elapsed
			if remaining > 1.2 { // Phase de départ (0.3s)
				// On simule une sortie d'écran vers le bord le plus proche
				if states[i].Y < ui.PlaymatY+ui.PlaymatH/2 {
					states[i].Y -= 100 // Saut vers le haut
				} else {
					states[i].Y += 100 // Saut vers le bas
				}
			}

			states[i].Scrambled = true
		}
	}

	// Aphasia : Brouille les labels périodiquement (toutes les 0.5s)
	if p.AphasiaTurns > 0 {
		if time.Since(m.lastGlitch) > 500*time.Millisecond {
			m.lastGlitch = time.Now()
			symbols := []rune{'@', '#', '$', '!', '%', '&', '*', '?'}
			for i := 0; i < btnCount; i++ {
				label := []rune(m.baseLabels[i])
				// Glitch 2 caractères au hasard
				for n := 0; n < 2; n++ {
					idx := rand.Intn(len(label))
					label[idx] = symbols[rand.Intn(len(symbols))]
				}
				m.glitchLabels[i] = string(label)
			}
		}
		for i := 0; i < btnCount; i++ {
			states[i].Label = m.glitchLabels[i]
		}
	}

	// Agnosia : géré côté renderer via le flag Scrambled
	if p.AgnosiaTurns > 0 {
		for i := 0; i < btnCount; i++ {
			states[i].Scrambled = true
		}
	}

	// Amnesia : Désactivation partielle
	if p.AmnesiaTurns > 0 {
		seed := time.Now().UnixMilli() / 2000
		r := rand.New(rand.NewSource(seed))
		for i := 0; i < btnCount; i++ {
			if r.Float32() < 0.3 {
				states[i].Active = false
			}
		}
	}
}

// regenerateScramble crée une nouvelle permutation des positions de boutons.
// Le seed est dérivé du temps et des troubles actifs pour garantir
// la reproductibilité sur une frame donnée tout en changeant régulièrement.
func (m *Manager) regenerateScramble(effects *player.StatusEffects, seed int64) {
	// Construit un seed composite basé sur les troubles actifs
	composite := seed
	imps := effects.GetActiveImpairments()
	sort.Slice(imps, func(i, j int) bool { return imps[i] < imps[j] })
	for _, imp := range imps {
		composite += int64(imp) * 997
	}

	r := rand.New(rand.NewSource(composite))
	perm := r.Perm(btnCount)
	for i := 0; i < btnCount; i++ {
		m.scrambleMapping[i] = perm[i]
	}
}

// HitTest détecte si les coordonnées écran (x,y) sont sur un bouton actif.
// Retourne le ButtonID et true si un bouton actif est touché.
func (m *Manager) HitTest(x, y int, states [btnCount]ButtonState) (ButtonID, bool) {
	fx, fy := float64(x), float64(y)
	for i := 0; i < btnCount; i++ {
		s := states[i]
		if !s.Active {
			continue
		}
		if fx >= s.CurrentX && fx <= s.CurrentX+s.Width &&
			fy >= s.CurrentY && fy <= s.CurrentY+s.Height {
			return s.ID, true
		}
	}
	return -1, false
}
