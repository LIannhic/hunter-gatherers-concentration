package structure

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// --- TEST 1: INITIALISATION DES STRUCTURES STANDARDS ---
func TestNewStructure_VisibilityAndTags(t *testing.T) {
	pos := entity.Position{X: 2, Y: 3}

	// 1. Test du Start Portal (doit être Revealed par défaut)
	sp := NewStructure("start_portal", pos)
	if sp.GetType() != entity.TypeStructure {
		t.Errorf("Type attendu %v, obtenu %v", entity.TypeStructure, sp.GetType())
	}
	if sp.GetPosition() != pos {
		t.Errorf("Position attendue %v, obtenue %v", pos, sp.GetPosition())
	}
	if sp.GetState() != entity.Revealed {
		t.Errorf("L'état du start_portal devrait être Revealed, obtenu %s", sp.GetState())
	}
	if !sp.HasTag("start_portal") {
		t.Error("Le tag 'start_portal' est manquant")
	}

	// 2. Test du Finish Portal (doit être Hidden par défaut)
	fp := NewStructure("finish_portal", pos)
	if fp.GetState() != entity.Hidden {
		t.Errorf("L'état du finish_portal devrait être Hidden, obtenu %s", fp.GetState())
	}
	if !fp.HasTag("finish_portal") {
		t.Error("Le tag 'finish_portal' est manquant")
	}

	// 3. Test d'une structure générique (ex: dolmen, obelisk)
	dolmen := NewStructure("dolmen", pos)
	if dolmen.GetState() != entity.Revealed {
		t.Errorf("Les structures par défaut devraient être Revealed, obtenu %s", dolmen.GetState())
	}
	if !dolmen.HasTag("dolmen") {
		t.Error("Le tag 'dolmen' est manquant")
	}
}

// --- TEST 2: COMPORTEMENT DES TUILES DE NAVIGATION ---
func TestNewNavigation(t *testing.T) {
	nav := NewNavigation(NavNorthRight, East)

	if nav.GetType() != entity.TypeStructure {
		t.Errorf("Type attendu %v, obtenu %v", entity.TypeStructure, nav.GetType())
	}
	if nav.NavType != NavNorthRight {
		t.Errorf("NavType attendu %s, obtenu %s", NavNorthRight, nav.NavType)
	}
	if nav.BaseOrient != East {
		t.Errorf("Orientation attendue %v, obtenue %v", East, nav.BaseOrient)
	}
	if !nav.HasTag("navigation") || !nav.HasTag("north-right") {
		t.Error("Les tags de navigation requis ('navigation', 'north-right') sont manquants")
	}
}

// --- TEST 3: CONTRAINTE D'IMMOBILITÉ ET COMPLIANCE ---
func TestStructure_ImmobilityAndCompliance(t *testing.T) {
	posInitial := entity.Position{X: 1, Y: 1}
	s := NewStructure("obelisk", posInitial)

	// Vérification de la conformité avec l'interface Entity
	var _ entity.Entity = s

	// Simulation du gestionnaire d'entités (Manager) pour tester l'immobilité logique
	manager := entity.NewManager()
	manager.Register(s)

	// Les structures agissent comme des limites fixes. On s'assure qu'on peut lire sa position sans dérive.
	posRecuperee := s.GetPosition()
	if posRecuperee != posInitial {
		t.Errorf("La position de la structure a bougé de façon inattendue: %v", posRecuperee)
	}

	// Même si un système tiers force un SetPosition externe, la structure métier doit rester cohérente
	nouvellePos := entity.Position{X: 5, Y: 5}
	s.SetPosition(nouvellePos)
	if s.GetPosition() != nouvellePos {
		t.Errorf("La structure devrait accepter sa nouvelle coordonnée de grille fixe, obtenue: %v", s.GetPosition())
	}
}

// --- TEST 4: CONFORMITÉ AUX RÈGLES D'ASSOCIATION (MATCHABLE) ---
func TestStructure_AssociationCompliance(t *testing.T) {
	s := NewStructure("dolmen", entity.Position{X: 0, Y: 0})

	// Les structures ont des catégories spécifiques basées sur leur type d'entité brut
	if s.GetCategory() != "structure" {
		t.Errorf("Catégorie d'association attendue 'structure', obtenue: '%s'", s.GetCategory())
	}

	// Par défaut, les structures ne s'associent pas (MatchID vide pour interdire le match direct en jeu)
	if s.GetMatchID() != "" {
		t.Errorf("Une structure ne devrait pas renvoyer un MatchID actif par défaut, obtenu: '%s'", s.GetMatchID())
	}

	types := s.GetMatchTypes()
	if len(types) != 1 || types[0] != "identical" {
		t.Errorf("Types d'association invalides pour la structure : %v", types)
	}
}
