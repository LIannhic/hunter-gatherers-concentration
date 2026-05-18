package player

import (
	"errors"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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

// PortablePortal constants
const (
	PortablePortalItemSourceID   = "portable_portal"
	PortablePortalItemName       = "portail portable"
	PortablePortalLootTaxPercent = 25
)

// LootItem représente un objet récolté
type LootItem struct {
	ID          string
	Name        string
	Type        entity.Type
	SourceID    string
	IsDeletable bool // Si faux, l'objet ne peut pas être supprimé par le joueur
}

func NewPortablePortalItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        PortablePortalItemName,
		Type:        entity.TypeArtefact,
		SourceID:    PortablePortalItemSourceID,
		IsDeletable: false,
	}
}

// Inventory inventaire du joueur
type Inventory struct {
	Items          []*LootItem    // Liste ordonnée des objets (slots)
	ResourceCounts map[string]int // Quantités de ressources stockées
	MaxSize        int
	ScrollOffset   float64 // Changé en float64 pour défilement fluide
}

func NewInventory(maxSize int) *Inventory {
	return &Inventory{
		Items:          make([]*LootItem, 0, maxSize),
		ResourceCounts: make(map[string]int),
		MaxSize:        maxSize,
		ScrollOffset:   0,
	}
}

// AddItem ajoute un objet à l'inventaire dans le premier slot disponible
func (inv *Inventory) AddItem(item *LootItem) error {
	if inv.GetTotalItemCount()+itemCountForLoot(item) >= inv.MaxSize {
		return errors.New("inventaire plein")
	}
	inv.Items = append(inv.Items, item)
	return nil
}

func itemCountForLoot(_ *LootItem) int {
	// Les artefacts comptent comme un slot unique
	return 1
}

// RemoveItem retire un objet par son index
func (inv *Inventory) RemoveItem(index int) error {
	if index < 0 || index >= len(inv.Items) {
		return errors.New("index invalide")
	}
	inv.Items = append(inv.Items[:index], inv.Items[index+1:]...)
	return nil
}

func (inv *Inventory) AddResource(resourceType string, amount int) error {
	if amount <= 0 {
		return errors.New("quantité invalide")
	}

	if inv.GetTotalItemCount()+amount > inv.MaxSize {
		return errors.New("inventaire plein")
	}

	inv.ResourceCounts[resourceType] += amount
	return nil
}

func (inv *Inventory) RemoveResource(resourceType string, amount int) error {
	if amount <= 0 {
		return errors.New("quantité invalide")
	}

	current := inv.GetResourceCount(resourceType)
	if amount > current {
		return errors.New("quantité insuffisante")
	}

	newCount := current - amount
	if newCount == 0 {
		delete(inv.ResourceCounts, resourceType)
	} else {
		inv.ResourceCounts[resourceType] = newCount
	}

	return nil
}

func (inv *Inventory) GetResourceCount(resourceType string) int {
	return inv.ResourceCounts[resourceType]
}

func (inv *Inventory) HasResource(resourceType string) bool {
	return inv.GetResourceCount(resourceType) > 0
}

func (inv *Inventory) GetTotalItems() int {
	return len(inv.Items)
}

func (inv *Inventory) GetTotalResourceCount() int {
	total := 0
	for _, count := range inv.ResourceCounts {
		total += count
	}
	return total
}

func (inv *Inventory) GetTotalItemCount() int {
	return inv.GetTotalItems() + inv.GetTotalResourceCount()
}

func (inv *Inventory) IsFull() bool {
	return inv.GetTotalItemCount() >= inv.MaxSize
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
		Inventory: *NewInventory(30),
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

// ConsumeSanity diminue la santé mentale
func (p *Player) ConsumeSanity(amount int) {
	p.Stats.Sanity -= amount
	if p.Stats.Sanity < 0 {
		p.Stats.Sanity = 0
	}
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
