package meta

import (
	"errors"
)

// Need représente un besoin familial
type Need struct {
	Type     string // "food", "debt", "health", "comfort"
	Current  int
	Required int
	Urgency  int // 1-5, 5 = critique
}

func (n Need) String() string {
	return n.Type
}

// Family gère l'état du foyer
type Family struct {
	Members []FamilyMember
	Needs   []Need
	Debt    int
	Day     int
}

type FamilyMember struct {
	Name   string
	Status string // "healthy", "sick", "weak"
	Age    int
}

func NewFamily() *Family {
	return &Family{
		Members: []FamilyMember{
			{Name: "Sœur aînée", Status: "healthy", Age: 16},
			{Name: "Frère cadet", Status: "healthy", Age: 8},
		},
		Needs: []Need{
			{Type: "food", Current: 50, Required: 100, Urgency: 3},
			{Type: "debt", Current: 200, Required: 0, Urgency: 4},
			{Type: "health", Current: 80, Required: 100, Urgency: 2},
		},
		Debt: 1000,
		Day:  1,
	}
}

// Allocate distribue des ressources aux besoins
func (f *Family) Allocate(resourceType string, amount int) error {
	for i := range f.Needs {
		if f.Needs[i].Type == resourceType {
			f.Needs[i].Current += amount
			if f.Needs[i].Current > f.Needs[i].Required {
				f.Needs[i].Current = f.Needs[i].Required
			}
			return nil
		}
	}
	return errors.New("besoin non trouvé")
}

// PayDebt réduit la dette
func (f *Family) PayDebt(amount int) {
	f.Debt -= amount
	if f.Debt < 0 {
		f.Debt = 0
	}
}

// NextDay passe au jour suivant, applique dégradation
func (f *Family) NextDay() {
	f.Day++

	// Dégradation des besoins
	for i := range f.Needs {
		switch f.Needs[i].Type {
		case "food":
			f.Needs[i].Current -= 20
		case "health":
			f.Needs[i].Current -= 5
		}

		// Met à jour l'urgence
		ratio := float64(f.Needs[i].Current) / float64(f.Needs[i].Required)
		switch {
		case ratio < 0.2:
			f.Needs[i].Urgency = 5
		case ratio < 0.4:
			f.Needs[i].Urgency = 4
		case ratio < 0.6:
			f.Needs[i].Urgency = 3
		case ratio < 0.8:
			f.Needs[i].Urgency = 2
		default:
			f.Needs[i].Urgency = 1
		}
	}
}

// IsStable vérifie si la famille est en sécurité
func (f *Family) IsStable() bool {
	for _, need := range f.Needs {
		if need.Urgency >= 4 {
			return false
		}
	}
	return true
}

// GetNeed retourne un besoin par type
func (f *Family) GetNeed(needType string) (Need, bool) {
	for _, need := range f.Needs {
		if need.Type == needType {
			return need, true
		}
	}
	return Need{}, false
}

// MetaProgression gère la progression entre missions
type MetaProgression struct {
	UnlockedZones   []string
	UnlockedRecipes []string
	Reputation      int            // Influence les prix et quêtes
	Knowledge       map[string]int // Connaissance des créatures/ressources
}

func NewMetaProgression() *MetaProgression {
	return &MetaProgression{
		UnlockedZones:   []string{"twilight_woods"},
		UnlockedRecipes: []string{"basic_potion"},
		Reputation:      0,
		Knowledge:       make(map[string]int),
	}
}

func (m *MetaProgression) UnlockZone(zone string) {
	for _, z := range m.UnlockedZones {
		if z == zone {
			return
		}
	}
	m.UnlockedZones = append(m.UnlockedZones, zone)
}

func (m *MetaProgression) IsZoneUnlocked(zone string) bool {
	for _, z := range m.UnlockedZones {
		if z == zone {
			return true
		}
	}
	return false
}

func (m *MetaProgression) AddKnowledge(subject string, amount int) {
	m.Knowledge[subject] += amount
}

func (m *MetaProgression) GetKnowledge(subject string) int {
	return m.Knowledge[subject]
}

// Hub centralise la méta-progression
type Hub struct {
	Family      *Family
	Progression *MetaProgression
	Inventory   map[string]int // Stockage entre missions
}

func NewHub() *Hub {
	return &Hub{
		Family:      NewFamily(),
		Progression: NewMetaProgression(),
		Inventory:   make(map[string]int),
	}
}

// PrepareMission transfère l'inventaire vers le joueur
func (h *Hub) PrepareMission(selectedTools []string) map[string]int {
	startingItems := make(map[string]int)

	// Ajoute les outils sélectionnés
	for _, tool := range selectedTools {
		if qty, ok := h.Inventory[tool]; ok && qty > 0 {
			startingItems[tool] = 1
			h.Inventory[tool]--
		}
	}

	// Consommables de base
	startingItems["ration"] = 2

	return startingItems
}

// ReturnFromMission traite le retour de mission
func (h *Hub) ReturnFromMission(loot map[string]int, success bool) {
	if !success {
		// Pénalités
		h.Family.Needs[0].Current -= 30 // Moins de nourriture
		return
	}

	// Transfère le butin
	for item, qty := range loot {
		h.Inventory[item] += qty
	}

	h.Family.NextDay()
}

func (h *Hub) AddToInventory(item string, qty int) {
	h.Inventory[item] += qty
}
