package actionbuttons

import (
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
	X, Y         float64 // Coordonnées finales à l'écran (déjà transformées)
	Width        float64
	Height       float64
	Scrambled    bool    // Vrai si les coordonnées ont été altérées par un trouble
	FillProgress float64 // 0.0 → 1.0, remplissage temporel (Skip uniquement)
	FillAlert    bool    // Vrai si le timer est en phase critique ou expiré
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
		// Le système verrouille la grille et active Skip
		states[BtnSkip].Active = true

		// Match et Merge sont soumis à des conditions sur l'état des tuiles
		// Pour l'instant, on active Merge si 2 tuiles sont retournées (la commande vérifiera si elles sont déjà cumulées)
		states[BtnMerge].Active = true

		// Match n'est activé QUE si les conditions de la commande sont remplies (tuiles cumulées)
		// Comme le manager est réactif, on laisse l'input handler ou la commande fournir cette info ?
		// Pour rester simple et réactif ici, on active les boutons si 2 tuiles sont là,
		// mais on pourrait affiner si on passait l'état "Cumulated" au manager.
		states[BtnMatch].Active = true
	}
	// Le bouton EndTurn reste toujours actif
	states[BtnEndTurn].Active = true

	// --- V0.2 : TRANSITION END GAME ---
	if m.isVictoryActive != nil && m.isVictoryActive() {
		states[BtnEndTurn].Label = "END GAME"
		if m.getVictoryProgress != nil {
			states[BtnEndTurn].FillProgress = m.getVictoryProgress()
			states[BtnEndTurn].FillAlert = true // Toujours coloré pour indiquer l'importance
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

	// --- TROUBLES COGNITIFS : Transformation des coordonnées ---
	if p != nil && p.StatusEffects != nil && p.StatusEffects.HasAnyImpairment() {
		m.applyImpairments(p.StatusEffects, &states)
	} else {
		m.resetScramble()
	}

	return states
}

// applyImpairments modifie les coordonnées et/ou labels des boutons
// en fonction des troubles cognitifs actifs du joueur.
func (m *Manager) applyImpairments(effects *player.StatusEffects, states *[btnCount]ButtonState) {
	// Détermine un seed stable par frame pour éviter le clignotement
	// tout en permettant un changement régulier (toutes les 2 secondes ~ 120 frames)
	now := time.Now().UnixMilli()
	seed := now / 2000 // changement toutes les 2 secondes

	// Regénère la permutation si le seed a changé
	if seed != m.lastScrambleSeed {
		m.lastScrambleSeed = seed
		m.regenerateScramble(effects, seed)
	}

	// Applique la permutation de positions (Ataxia / Agnosia / etc.)
	// Tout trouble cognitif peut déclencher le scrambling des positions
	if effects.HasImpairment(player.ImpairmentAtaxia) ||
		effects.HasImpairment(player.ImpairmentAgnosia) ||
		effects.HasImpairment(player.ImpairmentAphasia) ||
		effects.HasImpairment(player.ImpairmentAmnesia) {
		newCoords := [btnCount]struct{ x, y float64 }{}
		for i := 0; i < btnCount; i++ {
			mapped := m.scrambleMapping[i]
			newCoords[i] = m.baseCoords[mapped]
		}
		for i := 0; i < btnCount; i++ {
			states[i].X = ui.PlaymatX + newCoords[i].x
			states[i].Y = ui.PlaymatY + newCoords[i].y
			states[i].Scrambled = true
		}
	}

	// Aphasia : brouille les labels pour rendre l'identification difficile
	if effects.HasImpairment(player.ImpairmentAphasia) {
		scrambledLabels := [btnCount]string{"???", "???", "???", "???"}
		// Mélange partiel : on permute les labels de manière déterministe
		r := rand.New(rand.NewSource(seed))
		perm := r.Perm(btnCount)
		for i := 0; i < btnCount; i++ {
			scrambledLabels[i] = m.baseLabels[perm[i]]
		}
		for i := 0; i < btnCount; i++ {
			states[i].Label = scrambledLabels[i]
		}
	}

	// Agnosia : rend les boutons indifférenciés (même couleur visuelle suggérée)
	// -> géré côté renderer via le flag Scrambled
	if effects.HasImpairment(player.ImpairmentAgnosia) {
		for i := 0; i < btnCount; i++ {
			states[i].Scrambled = true
		}
	}

	// Amnesia : désactive aléatoirement des boutons (perte de mémoire des actions)
	if effects.HasImpairment(player.ImpairmentAmnesia) {
		r := rand.New(rand.NewSource(seed + 1))
		for i := 0; i < btnCount; i++ {
			if r.Float32() < 0.3 { // 30% de chance d'oublier un bouton
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
		if fx >= s.X && fx <= s.X+s.Width &&
			fy >= s.Y && fy <= s.Y+s.Height {
			return s.ID, true
		}
	}
	return -1, false
}
