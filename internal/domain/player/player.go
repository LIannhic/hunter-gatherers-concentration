package player

import (
	"errors"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// --- Constantes ---

// IDs de source et noms pour les objets et ressources
const (
	PortablePortalItemSourceID   = "portable_portal"
	PortablePortalItemName       = "portail portable"
	PortablePortalLootTaxPercent = 25

	EchoHoundItemSourceID        = "echo_hound_source"
	EchoHoundItemName            = "echo_hound"

	DreamberryItemSourceID       = "dreamberry_source"
	DreamberryItemName           = "dreamberry"
	MoonstoneItemSourceID        = "moonstone_source"
	MoonstoneItemName            = "moonstone"
	CrystalShardItemSourceID     = "crystal_shard_source"
	CrystalShardItemName         = "crystal_shard"
	WhisperingHerbItemSourceID   = "whispering_herb_source"
	WhisperingHerbItemName       = "whispering_herb"
	SpecterItemSourceID          = "specter_source"
	SpecterItemName              = "specter"
	BurrowerItemSourceID         = "burrower_source"
	BurrowerItemName             = "burrower"
)

// --- Types de base ---

// Stats représente les caractéristiques de progression et d'état du joueur.
type Stats struct {
	Health     int // Points de vie actuels
	MaxHealth  int // Maximum de points de vie
	Mana       int // Mana actuel pour les compétences
	MaxMana    int // Maximum de mana
	Sanity     int // Santé mentale (utilisée dans les plans oniriques)
	MaxSanity  int // Maximum de santé mentale
	Experience int // Points d'expérience accumulés au niveau actuel
	Level      int // Niveau actuel du joueur
}

// LootItem représente un objet physique ou une entité capturée dans l'inventaire.
type LootItem struct {
	ID          string      // Identifiant unique de l'instance
	Name        string      // Nom affiché de l'objet
	Type        entity.Type // Catégorie d'objet (Artefact, Créature, Ressource...)
	SourceID    string      // ID de référence pour les données sources (ex: atlas)
	IsUsable    bool        // Si l'objet possède une action d'utilisation
	IsDeletable bool        // Si l'objet peut être supprimé manuellement par le joueur
}

// BorderPosition définit l'ancrage du joueur sur la périphérie du plateau.
type BorderPosition int

const (
	BorderTop BorderPosition = iota
	BorderTopRight
	BorderRight
	BorderBottomRight
	BorderBottom
	BorderBottomLeft
	BorderLeft
	BorderTopLeft
)

// Skills regroupe les capacités passives et les bonus débloqués.
type Skills struct {
	UnlockedAssociations []string       // Types d'associations autorisées
	Resistances          map[string]int // Réductions de dégâts par type
	VisionRange          int            // Portée de dévoilement autour du joueur
	RevealEfficiency     float64        // Multiplicateur d'efficacité
}

// Inventory gère le stockage des objets et le comptage des ressources.
type Inventory struct {
	Items          []*LootItem    // Liste des objets occupant des slots individuels
	ResourceCounts map[string]int // Compteurs pour les ressources empilables
	MaxSize        int            // Capacité maximale totale (objets + ressources)
	ScrollOffset   float64        // Décalage pour le rendu de l'interface d'inventaire
}

// Player est la structure principale représentant le personnage joueur.
type Player struct {
	ID            string
	Stats         Stats
	Inventory     Inventory
	Skills        Skills
	StatusEffects *StatusEffects
	Position      entity.Position
	anchor        BorderPosition // Position sur la bordure du plateau
}

// --- Constructeurs ---

// New crée un nouveau joueur avec les statistiques et compétences de départ.
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
		Inventory:     *NewInventory(30),
		StatusEffects: NewStatusEffects(),
		Skills: Skills{
			UnlockedAssociations: []string{"identical"},
			Resistances:          make(map[string]int),
			VisionRange:          1,
			RevealEfficiency:     1.0,
		},
	}
}

// NewInventory initialise un inventaire avec une capacité spécifiée.
func NewInventory(maxSize int) *Inventory {
	return &Inventory{
		Items:          make([]*LootItem, 0, maxSize),
		ResourceCounts: make(map[string]int),
		MaxSize:        maxSize,
		ScrollOffset:   0,
	}
}

// Constructeurs d'objets spécifiques

func NewPortablePortalItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        PortablePortalItemName,
		Type:        entity.TypeArtefact,
		SourceID:    PortablePortalItemSourceID,
		IsUsable:    true,
		IsDeletable: false,
	}
}

func NewEchoHoundItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        EchoHoundItemName,
		Type:        entity.TypeCreature,
		SourceID:    EchoHoundItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

func NewDreamberryItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        DreamberryItemName,
		Type:        entity.TypeResource,
		SourceID:    DreamberryItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

func NewMoonstoneItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        MoonstoneItemName,
		Type:        entity.TypeResource,
		SourceID:    MoonstoneItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

func NewCrystalShardItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        CrystalShardItemName,
		Type:        entity.TypeResource,
		SourceID:    CrystalShardItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

func NewWhisperingHerbItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        WhisperingHerbItemName,
		Type:        entity.TypeResource,
		SourceID:    WhisperingHerbItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

func NewSpecterItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        SpecterItemName,
		Type:        entity.TypeCreature,
		SourceID:    SpecterItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

func NewBurrowerItem() *LootItem {
	return &LootItem{
		ID:          string(entity.NewID()),
		Name:        BurrowerItemName,
		Type:        entity.TypeCreature,
		SourceID:    BurrowerItemSourceID,
		IsUsable:    true,
		IsDeletable: true,
	}
}

// --- Méthodes de l'Inventaire ---

// AddItem ajoute un objet unique dans le premier slot disponible.
func (inv *Inventory) AddItem(item *LootItem) error {
	if inv.GetTotalItemCount()+itemCountForLoot(item) > inv.MaxSize {
		return errors.New("inventaire plein")
	}
	inv.Items = append(inv.Items, item)
	return nil
}

// RemoveItem retire l'objet à l'index spécifié.
func (inv *Inventory) RemoveItem(index int) error {
	if index < 0 || index >= len(inv.Items) {
		return errors.New("index invalide")
	}
	inv.Items = append(inv.Items[:index], inv.Items[index+1:]...)
	return nil
}

// AddResource incrémente la quantité d'une ressource.
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

// RemoveResource décrémente la quantité d'une ressource et nettoie l'entrée si nulle.
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

// GetResourceCount retourne le nombre d'unités possédées pour une ressource.
func (inv *Inventory) GetResourceCount(resourceType string) int {
	return inv.ResourceCounts[resourceType]
}

// HasResource vérifie la présence d'au moins une unité d'une ressource.
func (inv *Inventory) HasResource(resourceType string) bool {
	return inv.GetResourceCount(resourceType) > 0
}

// GetItem retourne l'objet à l'index donné sans le supprimer.
func (inv *Inventory) GetItem(index int) (*LootItem, error) {
	if index < 0 || index >= len(inv.Items) {
		return nil, errors.New("index invalide")
	}
	return inv.Items[index], nil
}

// GetTotalItems retourne le nombre de slots d'objets occupés.
func (inv *Inventory) GetTotalItems() int {
	return len(inv.Items)
}

// GetTotalResourceCount retourne le total cumulé de toutes les ressources.
func (inv *Inventory) GetTotalResourceCount() int {
	total := 0
	for _, count := range inv.ResourceCounts {
		total += count
	}
	return total
}

// GetTotalItemCount retourne l'occupation totale de l'inventaire.
func (inv *Inventory) GetTotalItemCount() int {
	return inv.GetTotalItems() + inv.GetTotalResourceCount()
}

// IsFull vérifie si l'inventaire est à capacité maximale.
func (inv *Inventory) IsFull() bool {
	return inv.GetTotalItemCount() >= inv.MaxSize
}

// itemCountForLoot retourne le poids en slots d'un objet.
func itemCountForLoot(_ *LootItem) int {
	// Les objets occupent actuellement tous 1 slot.
	return 1
}

// --- Méthodes du Joueur ---

// IsAlive vérifie si le joueur est toujours en vie.
func (p *Player) IsAlive() bool {
	return p.Stats.Health > 0
}

// TakeDamage applique des dégâts après réduction par les résistances.
func (p *Player) TakeDamage(amount int, damageType string) {
	resistance := p.Skills.Resistances[damageType]
	actual := amount - (amount * resistance / 100)
	p.Stats.Health -= actual
	if p.Stats.Health < 0 {
		p.Stats.Health = 0
	}
}

// Heal restaure des points de vie (borné au max).
func (p *Player) Heal(amount int) {
	p.Stats.Health += amount
	if p.Stats.Health > p.Stats.MaxHealth {
		p.Stats.Health = p.Stats.MaxHealth
	}
}

// ConsumeMana consomme du mana pour une action (retourne faux si insuffisant).
func (p *Player) ConsumeMana(amount int) bool {
	if p.Stats.Mana >= amount {
		p.Stats.Mana -= amount
		return true
	}
	return false
}

// RestoreMana restaure du mana (borné au max).
func (p *Player) RestoreMana(amount int) {
	p.Stats.Mana += amount
	if p.Stats.Mana > p.Stats.MaxMana {
		p.Stats.Mana = p.Stats.MaxMana
	}
}

// ConsumeSanity réduit la santé mentale (borné à 0).
func (p *Player) ConsumeSanity(amount int) {
	p.Stats.Sanity -= amount
	if p.Stats.Sanity < 0 {
		p.Stats.Sanity = 0
	}
}

// RestoreSanity restaure la santé mentale (borné au max).
func (p *Player) RestoreSanity(amount int) {
	p.Stats.Sanity += amount
	if p.Stats.Sanity > p.Stats.MaxSanity {
		p.Stats.Sanity = p.Stats.MaxSanity
	}
}

// GainExperience gère l'ajout d'XP et le passage de niveau.
func (p *Player) GainExperience(xp int) {
	p.Stats.Experience += xp
	threshold := p.Stats.Level * 100
	if p.Stats.Experience >= threshold {
		p.LevelUp()
		p.Stats.Experience -= threshold
	}
}

// LevelUp augmente le niveau et les statistiques de base.
func (p *Player) LevelUp() {
	p.Stats.Level++
	p.Stats.MaxHealth += 10
	p.Stats.MaxMana += 5
	p.Stats.Health = p.Stats.MaxHealth
	p.Stats.Mana = p.Stats.MaxMana
}

// UnlockAssociation débloque une nouvelle règle d'association si pas déjà connue.
func (p *Player) UnlockAssociation(assocType string) {
	if p.CanAssociate(assocType) {
		return
	}
	p.Skills.UnlockedAssociations = append(p.Skills.UnlockedAssociations, assocType)
}

// CanAssociate vérifie si un type d'association est débloqué.
func (p *Player) CanAssociate(assocType string) bool {
	for _, a := range p.Skills.UnlockedAssociations {
		if a == assocType {
			return true
		}
	}
	return false
}
