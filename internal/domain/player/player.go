package player

import (
	"errors"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// --- Constantes ---

// IDs de source et noms pour les objets et ressources
const (
    // Butin initial
	PortablePortalItemSourceID   = "portable_portal"
	PortablePortalItemName       = "portail portable"
	PortablePortalLootTaxPercent = 50

    // Butin ressources
	CrystalShardItemSourceID     = "crystal_shard"
	CrystalShardItemName         = "crystal_shard"
	DreamberryItemSourceID       = "dreamberry"
	DreamberryItemName           = "dreamberry"
	MoonstoneItemSourceID        = "moonstone"
	MoonstoneItemName            = "moonstone"
	WhisperingHerbItemSourceID   = "whispering_herb"
	WhisperingHerbItemName       = "whispering_herb"

	// Butin ressources exclusives
	MossTruffleItemSourceID     = "moss_truffle"
	MossTruffleItemName         = "moss_truffle"
	EchoCrystalItemSourceID      = "echo_crystal"
	EchoCrystalItemName          = "echo_crystal"
	VoidBloomItemSourceID        = "void_bloom"
	VoidBloomItemName            = "void_bloom"
	SandCoreItemSourceID         = "sand_core"
	SandCoreItemName             = "sand_core"

	// Butin créatures
	BurrowerItemSourceID         = "burrower"
   	BurrowerItemName             = "burrower"
	EchoHoundItemSourceID        = "echo_hound"
   	EchoHoundItemName            = "echo_hound"
   	FleeingSpriteSourceID        = "fleeing_sprite"
   	FleeingSpriteName            = "fleeing_sprite"
	FlutterwingItemSourceID      = "flutterwing"
	FlutterwingItemName          = "flutterwing"
	LumiflyItemSourceID          = "lumifly"
	LumiflyItemName              = "lumifly"
	MossMonkeyItemSourceID       = "moss_monkey"
	MossMonkeyItemName           = "moss_monkey"
	ShadowstalkerItemSourceID    = "shadowstalker"
	ShadowstalkerItemName        = "shadowstalker"
	SpecterItemSourceID          = "specter"
	SpecterItemName              = "specter"
	StonewardenItemSourceID      = "stonewarden"
	StonewardenItemName          = "stonewarden"

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
	Experience      int // Points d'expérience accumulés au niveau actuel
	TotalExperience int // Points d'expérience totaux accumulés (Score)
	Level           int // Niveau actuel du joueur
}

// LootItem représente un objet physique ou une entité capturée dans l'inventaire.
type LootItem struct {
	entity.BaseEntity
	Name         string      // Nom affiché de l'objet
	SourceID     string      // ID de référence pour les données sources (ex: atlas)
	OriginalType entity.Type // Type d'entité d'origine (Créature, Ressource...)
	IsUsable     bool        // Si l'objet possède une action d'utilisation
	IsDeletable  bool        // Si l'objet peut être supprimé manuellement par le joueur
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

func (b BorderPosition) String() string {
	switch b {
	case BorderTop:
		return "Top"
	case BorderTopRight:
		return "TopRight"
	case BorderRight:
		return "Right"
	case BorderBottomRight:
		return "BottomRight"
	case BorderBottom:
		return "Bottom"
	case BorderBottomLeft:
		return "BottomLeft"
	case BorderLeft:
		return "Left"
	case BorderTopLeft:
		return "TopLeft"
	default:
		return "Unknown"
	}
}

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

	// Buffs temporaires
	ImmunityTurns int // Nombre de tours d'immunité restants
	ThreatVisionTurns int // Nombre de tours où les zones de menace sont visibles
	GraceTurns int // Nombre de tours où les créatures n'attaquent pas lors de la révélation (Flutterwing)

	// Effets visuels (Shaders)
	VisualEffects map[string]int // "blur", "bubble", etc. -> Durée en tours
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
		VisualEffects: make(map[string]int),
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

func NewPortablePortalItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         PortablePortalItemName,
		SourceID:     PortablePortalItemSourceID,
		OriginalType: entity.TypeArtefact,
		IsUsable:     true,
		IsDeletable:  false,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewEchoHoundItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         EchoHoundItemName,
		SourceID:     EchoHoundItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewDreamberryItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         DreamberryItemName,
		SourceID:     DreamberryItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewMoonstoneItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         MoonstoneItemName,
		SourceID:     MoonstoneItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewCrystalShardItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         CrystalShardItemName,
		SourceID:     CrystalShardItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewWhisperingHerbItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         WhisperingHerbItemName,
		SourceID:     WhisperingHerbItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewSpecterItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         SpecterItemName,
		SourceID:     SpecterItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewShadowstalkerItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         ShadowstalkerItemName,
		SourceID:     ShadowstalkerItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewMossMonkeyItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         MossMonkeyItemName,
		SourceID:     MossMonkeyItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewStonewardenItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         StonewardenItemName,
		SourceID:     StonewardenItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewLumiflyItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         LumiflyItemName,
		SourceID:     LumiflyItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewBurrowerItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         BurrowerItemName,
		SourceID:     BurrowerItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewFleeingSpriteItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         FleeingSpriteName,
		SourceID:     FleeingSpriteSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewFlutterwingItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         FlutterwingItemName,
		SourceID:     FlutterwingItemSourceID,
		OriginalType: entity.TypeCreature,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewMossTruffleItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         MossTruffleItemName,
		SourceID:     MossTruffleItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewEchoCrystalItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         EchoCrystalItemName,
		SourceID:     EchoCrystalItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewVoidBloomItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         VoidBloomItemName,
		SourceID:     VoidBloomItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
}

func NewSandCoreItem(level int) *LootItem {
	item := &LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         SandCoreItemName,
		SourceID:     SandCoreItemSourceID,
		OriginalType: entity.TypeResource,
		IsUsable:     true,
		IsDeletable:  true,
	}
	item.SetCumulationLevel(level)
	return item
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
	if p.ImmunityTurns > 0 {
		return // Immunisé
	}
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

	// Décrémentation des buffs de tour
	if amount > 0 && p.ImmunityTurns > 0 {
		p.ImmunityTurns--
	}
	if amount > 0 && p.ThreatVisionTurns > 0 {
		p.ThreatVisionTurns--
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
	p.Stats.TotalExperience += xp
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
	p.Heal(10)       // Soigne du montant de l'augmentation
	p.RestoreMana(5) // Restaure du montant de l'augmentation
}

// GetFlipDirection retourne la direction de bascule d'une tuile basée sur l'ancrage du joueur.
func (b BorderPosition) GetFlipDirection() entity.FlipDirection {
	switch b {
	case BorderTop:
		return entity.FlipTop
	case BorderBottom:
		return entity.FlipBottom
	case BorderLeft:
		return entity.FlipLeft
	case BorderRight:
		return entity.FlipRight
	case BorderTopLeft:
		return entity.FlipTopLeft
	case BorderTopRight:
		return entity.FlipTopRight
	case BorderBottomLeft:
		return entity.FlipBottomLeft
	case BorderBottomRight:
		return entity.FlipBottomRight
	default:
		return entity.FlipCenter
	}
}

// GetInwardDirection convertit l'ancre du joueur en direction cardinale vers l'intérieur du plateau.
func (b BorderPosition) GetInwardDirection() entity.Direction {
	switch b {
	case BorderTop:
		return entity.DirSouth
	case BorderBottom:
		return entity.DirNorth
	case BorderLeft:
		return entity.DirEast
	case BorderRight:
		return entity.DirWest
	default:
		return entity.DirNorth
	}
}

// GetOutwardDirection retourne la direction de la créature vers le joueur quand
// le joueur est sur un bord de la même tuile que la créature.
func (b BorderPosition) GetOutwardDirection() entity.Direction {
	switch b {
	case BorderTop:
		return entity.DirNorth
	case BorderTopRight:
		return entity.DirNorthEast
	case BorderRight:
		return entity.DirEast
	case BorderBottomRight:
		return entity.DirSouthEast
	case BorderBottom:
		return entity.DirSouth
	case BorderBottomLeft:
		return entity.DirSouthWest
	case BorderLeft:
		return entity.DirWest
	case BorderTopLeft:
		return entity.DirNorthWest
	default:
		return entity.DirNorth
	}
}

// IsLookingAt vérifie si le joueur regarde vers une position donnée (Champ de vision 180° depuis l'arête).
func (p *Player) IsLookingAt(target entity.Position) bool {
	dir := p.anchor.GetInwardDirection()
	playerPos := p.Position

	dx := target.X - playerPos.X
	dy := target.Y - playerPos.Y

	// La validation dépend de l'ancrage et de la direction intérieure.
	// Le joueur regarde vers l'intérieur du plateau depuis sa bordure.
	switch dir {
	case entity.DirNorth:
		// Joueur en bas, regarde vers le haut (Y diminue)
		return dy < 0
	case entity.DirSouth:
		// Joueur en haut, regarde vers le bas (Y augmente)
		return dy > 0
	case entity.DirEast:
		// Joueur à gauche, regarde vers la droite (X augmente)
		return dx > 0
	case entity.DirWest:
		// Joueur à droite, regarde vers la gauche (X diminue)
		return dx < 0
	}
	return false
}

// UnlockAssociation débloque une nouvelle règle d'association si pas déjà connue.
func (p *Player) UnlockAssociation(assocType string) {
	if p.CanAssociate(assocType) {
		return
	}
	p.Skills.UnlockedAssociations = append(p.Skills.UnlockedAssociations, assocType)
}

// GetAnchor retourne l'ancrage actuel du joueur.
func (p *Player) GetAnchor() BorderPosition {
	return p.anchor
}

// SetAnchor définit l'ancrage du joueur.
func (p *Player) SetAnchor(anchor BorderPosition) {
	p.anchor = anchor
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
