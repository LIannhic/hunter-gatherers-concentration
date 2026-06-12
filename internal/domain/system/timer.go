package system

// TurnTimer gère le compte à rebours temps réel par tour.
// Il simule la pression temporelle : l'inaction force une décision.
type TurnTimer struct {
	MaxTime   float64 // Durée maximale en secondes (depuis la difficulté)
	Remaining float64 // Temps restant en secondes
	Running   bool    // Le timer est-il actif ?
}

// NewTurnTimer crée un timer inactif avec la durée maximale fournie.
func NewTurnTimer(maxTime float64) *TurnTimer {
	return &TurnTimer{
		MaxTime:   maxTime,
		Remaining: maxTime,
		Running:   false,
	}
}

// Update décrémente le timer selon le delta-temps écoulé (en secondes).
// Retourne true si le timer vient d'atteindre 0 (expiration).
func (t *TurnTimer) Update(dt float64) bool {
	if !t.Running || t.Remaining <= 0 {
		return false
	}
	t.Remaining -= dt
	if t.Remaining <= 0 {
		t.Remaining = 0
		return true
	}
	return false
}

// Reset remet le timer à sa valeur maximale et le démarre.
func (t *TurnTimer) Reset() {
	t.Remaining = t.MaxTime
	t.Running = true
}

// Start démarre le timer sans le remettre à zéro.
func (t *TurnTimer) Start() {
	t.Running = true
}

// Stop arrête le décompte.
func (t *TurnTimer) Stop() {
	t.Running = false
}

// IsExpired retourne true si le temps est écoulé.
func (t *TurnTimer) IsExpired() bool {
	return t.Remaining <= 0
}

// Progress retourne le ratio de remplissage (0.0 = vide, 1.0 = plein/temps écoulé).
func (t *TurnTimer) Progress() float64 {
	if t.MaxTime <= 0 {
		return 1.0
	}
	p := 1.0 - (t.Remaining / t.MaxTime)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// IsPanic retourne true s'il reste moins de 3 secondes (phase d'urgence).
func (t *TurnTimer) IsPanic() bool {
	return t.Remaining > 0 && t.Remaining < 3.0
}

// SetMaxTime change la durée maximale (utile lors d'un changement de difficulté).
func (t *TurnTimer) SetMaxTime(maxTime float64) {
	t.MaxTime = maxTime
	if t.Remaining > t.MaxTime {
		t.Remaining = t.MaxTime
	}
}
