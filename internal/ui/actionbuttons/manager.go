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
	IsAgnosia    bool    // Vrai si l'agnosie est active
	IsAtaxia     bool    // Vrai si l'ataxie est active
	FillProgress float64 // 0.0 → 1.0, remplissage temporel (Skip uniquement)
	FillAlert    bool    // Vrai si le timer est en phase critique ou expiré
	TextScale    float64 // Facteur d'échelle du texte (Aphasia)
	RevealedEntities []string // IDs des entités révélées ce tour
}

// Manager gère l'état réactif des 4 boutons d'action du Playmat.
// Il est purement réactif : l'état des boutons est recalculé à chaque frame
// en fonction de l'état de la grille et des troubles cognitifs du joueur.
type Manager struct {
	// Références externes
	getRevealedTileCount func() int
	getRevealedEntities  func() []string
	getPlayer            func() *player.Player
	getTimerProgress     func() float64 // 0.0 → 1.0 (temps écoulé)
	getTimerPanic        func() bool    // true si < 3s restantes
	getVictoryProgress   func() float64 // 0.0 → 1.0 (V0.2)
	isVictoryActive      func() bool    // true si portail déployé (V0.2)
	isPortalMatch        func() bool    // true si duo de portails sélectionnés

	// Base coordinates (fixed by UI spec)
	baseCoords [btnCount]struct{ x, y float64 }
	baseLabels [btnCount]string

	// Animation state
	currentPos   [btnCount]struct{ x, y float64 }
	glitchLabels [btnCount]string
	lastGlitch   time.Time
	startTime    time.Time
	// Ataxia state (remplace jumpTime/jumpTarget/jumpPhase)
	ataxiaPhase     [btnCount]int        // 0=stable, 1=sortie, 2=entrée
	ataxiaPhaseTime [btnCount]float64    // elapsed au début de la phase
	ataxiaSlot      [btnCount]int        // slot stable ou cible

	// Cache du frame précédent pour stabilité visuelle
	lastScrambleSeed int64
	scrambleMapping  [btnCount]int // index -> permuted index
}

// NewManager crée un nouveau gestionnaire de boutons d'action.
// getRevealedTileCount doit retourner le nombre de tuiles révélées ce tour.
// getPlayer doit retourner le joueur courant (pour les StatusEffects).
// getTimerProgress / getTimerPanic fournissent l'état du compte à rebours temps réel.
func NewManager(getRevealedTileCount func() int, getRevealedEntities func() []string, getPlayer func() *player.Player, getTimerProgress func() float64, getTimerPanic func() bool, getVictoryProgress func() float64, isVictoryActive func() bool, isPortalMatch func() bool) *Manager {
	m := &Manager{
		getRevealedTileCount: getRevealedTileCount,
		getRevealedEntities:  getRevealedEntities,
		getPlayer:            getPlayer,
		getTimerProgress:     getTimerProgress,
		getTimerPanic:        getTimerPanic,
		getVictoryProgress:   getVictoryProgress,
		isVictoryActive:      isVictoryActive,
		isPortalMatch:        isPortalMatch,
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
	var revealedEntities []string
	if m.getRevealedEntities != nil {
		revealedEntities = m.getRevealedEntities()
	}
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
			RevealedEntities: revealedEntities,
		}
	}

	// --- RÈGLE MÉTIER : Activation selon le nombre de tuiles révélées ---
	if revealedCount >= 2 {
		states[BtnSkip].Active = true
		states[BtnMerge].Active = true
		states[BtnMatch].Active = true

		// V0.3 : Changement de labels pour le portail
		if m.isPortalMatch != nil && m.isPortalMatch() {
			states[BtnMatch].Label = "EXTRACT"
			states[BtnMerge].Label = "EXTRACT"
			states[BtnSkip].Label = "STAY"
		}
	}
	// TURN est actif sauf quand 2+ tuiles sont révélées (le joueur doit choisir)
	// Exception : victoire active (END GAME)
	if revealedCount < 2 || (m.isVictoryActive != nil && m.isVictoryActive()) {
		states[BtnEndTurn].Active = true
		if m.isPortalMatch != nil && m.isPortalMatch() {
			states[BtnEndTurn].Label = "STAY"
		}
	}

	// --- V0.2 : TRANSITION END GAME ---
	if m.isVictoryActive != nil && m.isVictoryActive() {
		states[BtnEndTurn].Label = "END GAME"
		if m.getVictoryProgress != nil {
			states[BtnEndTurn].FillProgress = m.getVictoryProgress()
			states[BtnEndTurn].FillAlert = true
		}
	} else if revealedCount >= 2 && m.isPortalMatch != nil && m.isPortalMatch() {
		// V0.3 : STAY prend le pas sur TURN/END TURN si portail sélectionné
		states[BtnEndTurn].Label = "STAY"
	}

	// --- FEEDBACK TEMPS RÉEL : Remplissage des boutons Skip et Turn ---
	if m.getTimerProgress != nil {
		timerProgress := m.getTimerProgress()
		states[BtnSkip].FillProgress = timerProgress
		states[BtnSkip].FillAlert = states[BtnSkip].FillProgress >= 1.0
		// Turn utilise le timer seulement si la victoire n'est pas active
		if !(m.isVictoryActive != nil && m.isVictoryActive()) {
			states[BtnEndTurn].FillProgress = timerProgress
			states[BtnEndTurn].FillAlert = states[BtnEndTurn].FillProgress >= 1.0
		}
	}
	if m.getTimerPanic != nil && m.getTimerPanic() {
		states[BtnSkip].FillAlert = true
		if !(m.isVictoryActive != nil && m.isVictoryActive()) {
			states[BtnEndTurn].FillAlert = true
		}
	}

	// --- TROUBLES COGNITIFS ---
	if p != nil && (p.AphasiaTurns > 0 || p.AtaxiaTurns > 0 || p.AgnosiaTurns > 0) {
		m.applyImpairments(p, &states, elapsed)
	} else {
		m.resetScramble()
		for i := 0; i < btnCount; i++ {
			m.ataxiaPhaseTime[i] = 0
			m.ataxiaSlot[i] = i
			m.ataxiaPhase[i] = 0
		}
	}

	// --- ANIMATION ET INTERPOLATION ---
	dt := 0.15 // Vitesse de l'interpolation
	ataxiaActive := p != nil && p.AtaxiaTurns > 0
	for i := 0; i < btnCount; i++ {
		targetX, targetY := states[i].X, states[i].Y

		if ataxiaActive && m.ataxiaPhase[i] != 0 {
			// Ataxia en phase 1 ou 2 : utilisation directe des positions calculées
			m.currentPos[i].x = targetX
			m.currentPos[i].y = targetY
		} else {
			speed := dt
			m.currentPos[i].x += (targetX - m.currentPos[i].x) * speed
			m.currentPos[i].y += (targetY - m.currentPos[i].y) * speed
		}

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
	isAgnosia := p.AgnosiaTurns > 0
	isAtaxia := p.AtaxiaTurns > 0

	for i := 0; i < btnCount; i++ {
		states[i].IsAgnosia = isAgnosia
		states[i].IsAtaxia = isAtaxia
	}

	// Ataxia : boutons sortent de l'écran et reviennent à une nouvelle position
	if isAtaxia {
		const btnHalfH = 21.88
		exitTop := -btnHalfH
		exitBottom := float64(ui.ScreenHeight) + btnHalfH
		centerY := float64(ui.PlaymatY) + float64(ui.PlaymatH)/2

		// Initialisation
		if m.ataxiaPhaseTime[0] == 0 {
			for i := 0; i < btnCount; i++ {
				m.ataxiaPhase[i] = 0
				m.ataxiaPhaseTime[i] = elapsed
				m.ataxiaSlot[i] = i
			}
		}

		// Bitmap des slots occupés (stable + incoming)
		occupied := 0
		for i := 0; i < btnCount; i++ {
			if m.ataxiaPhase[i] == 0 || m.ataxiaPhase[i] == 2 {
				occupied |= 1 << uint(m.ataxiaSlot[i])
			}
		}

		for i := 0; i < btnCount; i++ {
			states[i].Scrambled = true
			slot := m.ataxiaSlot[i]
			base := m.baseCoords[slot]
			slotY := float64(ui.PlaymatY) + base.y

			switch m.ataxiaPhase[i] {
			case 0: // Stable — assis au slot
				states[i].X = float64(ui.PlaymatX) + base.x
				states[i].Y = slotY

				duration := 0.8 + rand.Float64()*1.2 // 0.8–2.0s
				if elapsed-m.ataxiaPhaseTime[i] >= duration {
					occupied &^= 1 << uint(slot)    // libère le slot
					m.ataxiaPhase[i] = 1
					m.ataxiaPhaseTime[i] = elapsed
				}

			case 1: // Sortie ease-out (0.2s)
				duration := 0.2
				progress := (elapsed - m.ataxiaPhaseTime[i]) / duration
				if progress >= 1.0 {
					progress = 1.0
					// Choisit un slot libre
					available := (0x0F) & ^occupied
					if available == 0 {
						available = 0x0F & ^(1 << uint(slot))
						if available == 0 {
							available = 0x0F
						}
					}
					candidates := make([]int, 0, btnCount)
					for b := 0; b < btnCount; b++ {
						if available&(1<<uint(b)) != 0 {
							candidates = append(candidates, b)
						}
					}
					slot = candidates[rand.Intn(len(candidates))]
					m.ataxiaSlot[i] = slot
					occupied |= 1 << uint(slot) // réserve le nouveau slot
					m.ataxiaPhase[i] = 2
					m.ataxiaPhaseTime[i] = elapsed
				}

				states[i].X = float64(ui.PlaymatX) + base.x
				exitY := exitBottom
				if slotY < centerY {
					exitY = exitTop
				}
				t := 1.0 - (1.0-progress)*(1.0-progress) // ease-out quad
				states[i].Y = slotY + (exitY-slotY)*t

			case 2: // Entrée ease-in (0.2s)
				duration := 0.2
				progress := (elapsed - m.ataxiaPhaseTime[i]) / duration
				if progress >= 1.0 {
					progress = 1.0
					m.ataxiaPhase[i] = 0
					m.ataxiaPhaseTime[i] = elapsed
				}

				targetBase := m.baseCoords[slot]
				targetY := float64(ui.PlaymatY) + targetBase.y
				states[i].X = float64(ui.PlaymatX) + targetBase.x

				enterY := exitBottom
				if targetY < centerY {
					enterY = exitTop
				}
				t := progress * progress // ease-in quad
				states[i].Y = enterY + (targetY-enterY)*t
			}
		}
	}

	// Aphasia : Brouille les labels périodiquement (toutes les 0.5s)
	if p.AphasiaTurns > 0 {
		if time.Since(m.lastGlitch) > 500*time.Millisecond {
			m.lastGlitch = time.Now()
			symbols := []rune{'@', '#', '$', '!', '%', '&', '*', '?', '0', '3', '4', '7', '1', '8'}
			for i := 0; i < btnCount; i++ {
				sourceLabel := m.baseLabels[i]
				if p.AgnosiaTurns > 0 {
					sourceLabel = "BOUTON"
				}
				label := []rune(sourceLabel)

				// Glitch proportionnel à la longueur (environ 30-40% des caractères)
				numGlitches := (len(label) / 3) + 1
				for n := 0; n < numGlitches; n++ {
					idx := rand.Intn(len(label))
					// Remplacement par un symbole ou un chiffre "glitché"
					label[idx] = symbols[rand.Intn(len(symbols))]
				}
				m.glitchLabels[i] = string(label)
			}
		}
		for i := 0; i < btnCount; i++ {
			states[i].Label = m.glitchLabels[i]
		}
	}

	// Agnosia : rend les boutons indifférenciés (identiques)
	if p.AgnosiaTurns > 0 {
		for i := 0; i < btnCount; i++ {
			states[i].Label = "BOUTON"
			states[i].Scrambled = true
			// Applique le timer à tous les boutons pour l'uniformité
			if m.getTimerProgress != nil {
				states[i].FillProgress = m.getTimerProgress()
				states[i].FillAlert = states[i].FillProgress >= 1.0
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