package player

import (
	"errors"
)

// Stats représente les caractéristiques du joueur
type Stats struct {
	Health     int
	MaxHealth  int
	Mana       int
	MaxMana    int
	Sanity     int // Santé mentale pour les plans oniriques
	MaxSanity  int
	Experience int
	Level      int
}

// Inventory inventaire du joueur
type Inventory struct {
	Resources map[string]int // ID ressource -> quantité
	Artifacts []string       // IDs des artefacts
	Tools     []string       // IDs des outils équipés
	MaxSize   int
}

func NewInventory(maxSize int) *Inventory {
	return &Inventory{
		Resources: make(map[string]int),
		Artifacts: make([]string, 0),
		Tools:     make([]string, 0),
		MaxSize:   maxSize,
	}
}

func (inv *Inventory) AddResource(id string, qty int) error {
	current := inv.GetTotalItems()
	if current+qty > inv.MaxSize {
		return errors.New("inventaire plein")
	}
	inv.Resources[id] += qty
	return nil
}

func (inv *Inventory) RemoveResource(id string, qty int) error {
	if inv.Resources[id] < qty {
		return errors.New("quantité insuffisante")
	}
	inv.Resources[id] -= qty
	if inv.Resources[id] == 0 {
		delete(inv.Resources, id)
	}
	return nil
}

func (inv *Inventory) GetTotalItems() int {
	total := len(inv.Artifacts) + len(inv.Tools)
	for _, qty := range inv.Resources {
		total += qty
	}
	return total
}

func (inv *Inventory) HasResource(id string) bool {
	return inv.Resources[id] > 0
}

func (inv *Inventory) GetResourceCount(id string) int {
	return inv.Resources[id]
}

// Skills capacités débloquées
type Skills struct {
	UnlockedAssociations []string // Types d'association débloqués
	Resistances          map[string]int
	VisionRange          int
	RevealEfficiency     float64 // Réduction du coût de révélation
}

// Player entité joueur
type Player struct {
	ID        string
	Stats     Stats
	Inventory Inventory
	Skills    Skills
	Position  struct{ X, Y int }
}

func New(id string) *Player {
	return &Player{
		ID: id,
		Stats: Stats{
			Health:    100,
			MaxHealth: 100,
			Mana:      50,
			MaxMana:   50,
			Sanity:    100,
			MaxSanity: 100,
			Level:     1,
		},
		Inventory: *NewInventory(20),
		Skills: Skills{
			UnlockedAssociations: []string{"identical"},
			Resistances:          make(map[string]int),
			VisionRange:          1,
			RevealEfficiency:     1.0,
		},
	}
}

// ConsumeMana consomme du mana pour une action
func (p *Player) ConsumeMana(amount int) bool {
	if p.Stats.Mana >= amount {
		p.Stats.Mana -= amount
		return true
	}
	return false
}

// TakeDamage applique des dégâts
func (p *Player) TakeDamage(amount int, damageType string) {
	resistance := p.Skills.Resistances[damageType]
	actual := amount - (amount * resistance / 100)
	p.Stats.Health -= actual
	if p.Stats.Health < 0 {
		p.Stats.Health = 0
	}
}

// Heal soigne le joueur
func (p *Player) Heal(amount int) {
	p.Stats.Health += amount
	if p.Stats.Health > p.Stats.MaxHealth {
		p.Stats.Health = p.Stats.MaxHealth
	}
}

// RestoreMana restaure le mana
func (p *Player) RestoreMana(amount int) {
	p.Stats.Mana += amount
	if p.Stats.Mana > p.Stats.MaxMana {
		p.Stats.Mana = p.Stats.MaxMana
	}
}

// GainExperience ajoute de l'XP et gère les niveaux
func (p *Player) GainExperience(xp int) {
	p.Stats.Experience += xp
	threshold := p.Stats.Level * 100
	if p.Stats.Experience >= threshold {
		p.LevelUp()
		p.Stats.Experience -= threshold
	}
}

func (p *Player) LevelUp() {
	p.Stats.Level++
	p.Stats.MaxHealth += 10
	p.Stats.MaxMana += 5
	p.Stats.Health = p.Stats.MaxHealth
	p.Stats.Mana = p.Stats.MaxMana
}

// IsAlive vérifie si le joueur est en vie
func (p *Player) IsAlive() bool {
	return p.Stats.Health > 0
}

// UnlockAssociation débloque un nouveau type d'association
func (p *Player) UnlockAssociation(assocType string) {
	for _, a := range p.Skills.UnlockedAssociations {
		if a == assocType {
			return
		}
	}
	p.Skills.UnlockedAssociations = append(p.Skills.UnlockedAssociations, assocType)
}

// CanAssociate vérifie si le joueur peut faire ce type d'association
func (p *Player) CanAssociate(assocType string) bool {
	for _, a := range p.Skills.UnlockedAssociations {
		if a == assocType {
			return true
		}
	}
	return false
}
