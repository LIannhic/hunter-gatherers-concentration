package entity

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// ID unique pour les entités
type ID string

func NewID() ID {
	return ID(uuid.New().String())
}

// TileState représente l'état visuel d'une tuile (entité)
type TileState int

const (
	Hidden    TileState = 1 << iota // 1
	Revealed                        // 2
	Matched                         // 4
	Blocked                         // 8
	Cumulated                       // 16
)

func (s TileState) String() string {
	res := ""
	if s&Hidden != 0 {
		res += "hidden "
	}
	if s&Revealed != 0 {
		res += "revealed "
	}
	if s&Matched != 0 {
		res += "matched "
	}
	if s&Blocked != 0 {
		res += "blocked "
	}
	if s&Cumulated != 0 {
		res += "cumulated "
	}
	if res == "" {
		return "unknown"
	}
	return res[:len(res)-1]
}

// Type d'entité
type Type int

const (
	TypeResource Type = iota
	TypeCreature
	TypeStructure
	TypeArtefact
	TypeTrap
	TypeLoot
	TypeTrack
)

// Direction représente les orientations cardinales
type Direction int

const (
	DirNorth Direction = iota
	DirEast
	DirSouth
	DirWest
	DirNorthEast
	DirSouthEast
	DirSouthWest
	DirNorthWest
)

func (d Direction) String() string {
	switch d {
	case DirNorth:
		return "North"
	case DirEast:
		return "East"
	case DirSouth:
		return "South"
	case DirWest:
		return "West"
	case DirNorthEast:
		return "North-East"
	case DirSouthEast:
		return "South-East"
	case DirSouthWest:
		return "South-West"
	case DirNorthWest:
		return "North-West"
	default:
		return "Unknown"
	}
}

// IsOpposite vérifie si deux directions sont strictement opposées
func (d Direction) IsOpposite(other Direction) bool {
	return (d == DirNorth && other == DirSouth) ||
		(d == DirSouth && other == DirNorth) ||
		(d == DirEast && other == DirWest) ||
		(d == DirWest && other == DirEast)
}

// ToVector convertit une direction en vecteur de position relative
func (d Direction) ToVector() Position {
	switch d {
	case DirNorth:
		return Position{X: 0, Y: -1}
	case DirEast:
		return Position{X: 1, Y: 0}
	case DirSouth:
		return Position{X: 0, Y: 1}
	case DirWest:
		return Position{X: -1, Y: 0}
	case DirNorthEast:
		return Position{X: 1, Y: -1}
	case DirSouthEast:
		return Position{X: 1, Y: 1}
	case DirSouthWest:
		return Position{X: -1, Y: 1}
	case DirNorthWest:
		return Position{X: -1, Y: -1}
	}
	return Position{X: 0, Y: 0}
}

// TransformDirection applique une transformation D4 à une direction cardinale
func TransformDirection(d Direction, t Transformation) Direction {
	// 1. Convertit Direction en vecteur local
	v := d.ToVector()

	// 2. Applique la transformation D4 au vecteur
	// (x, y) après transformation
	var nx, ny int
	switch t {
	case TransIdentity: // (x, y)
		nx, ny = v.X, v.Y
	case TransRot90: // (-y, x)
		nx, ny = -v.Y, v.X
	case TransRot180: // (-x, -y)
		nx, ny = -v.X, -v.Y
	case TransRot270: // (y, -x)
		nx, ny = v.Y, -v.X
	case TransMirrorH: // (-x, y) - Médiane verticale
		nx, ny = -v.X, v.Y
	case TransMirrorV: // (x, -y) - Médiane horizontale
		nx, ny = v.X, -v.Y
	case TransMirrorD1: // (y, x) - Diagonale \
		nx, ny = v.Y, v.X
	case TransMirrorD2: // (-y, -x) - Diagonale /
		nx, ny = -v.Y, -v.X
	default:
		nx, ny = v.X, v.Y
	}

	// 3. Reconvertit le vecteur transformé en Direction
	// Note : on gère les 8 directions cardinales et ordinales
	if nx == 0 && ny < 0 {
		return DirNorth
	}
	if nx > 0 && ny < 0 {
		return DirNorthEast
	}
	if nx > 0 && ny == 0 {
		return DirEast
	}
	if nx > 0 && ny > 0 {
		return DirSouthEast
	}
	if nx == 0 && ny > 0 {
		return DirSouth
	}
	if nx < 0 && ny > 0 {
		return DirSouthWest
	}
	if nx < 0 && ny == 0 {
		return DirWest
	}
	if nx < 0 && ny < 0 {
		return DirNorthWest
	}

	return d
}

// FlipDirection représente la direction de flip d'une tuile lors du reveal
// Cette information est purement visuelle et n'impacte pas la logique métier
type FlipDirection int

const (
	FlipTop FlipDirection = iota
	FlipTopRight
	FlipRight
	FlipBottomRight
	FlipBottom
	FlipBottomLeft
	FlipLeft
	FlipTopLeft
	FlipCenter // Flip direct (clic au centre)
)

func (f FlipDirection) String() string {
	switch f {
	case FlipTop:
		return "top"
	case FlipTopRight:
		return "top-right"
	case FlipRight:
		return "right"
	case FlipBottomRight:
		return "bottom-right"
	case FlipBottom:
		return "bottom"
	case FlipBottomLeft:
		return "bottom-left"
	case FlipLeft:
		return "left"
	case FlipTopLeft:
		return "top-left"
	case FlipCenter:
		return "center"
	}
	return "unknown"
}

// Rotate fait pivoter une direction de flip par pas de 45° (sens horaire).
func (f FlipDirection) Rotate(steps int) FlipDirection {
	if f == FlipCenter {
		return FlipCenter
	}
	newDir := (int(f) + steps) % 8
	if newDir < 0 {
		newDir += 8
	}
	return FlipDirection(newDir)
}

// ToRotationAngles retourne les angles de rotation (X, Y) pour l'animation de flip
// en degrés, selon la direction. Utilisé par le renderer pour l'animation.
func (f FlipDirection) ToRotationAngles() (rotateX, rotateY float64) {
	switch f {
	case FlipTop:
		return -90, 0
	case FlipTopRight:
		return -45, 45
	case FlipRight:
		return 0, 90
	case FlipBottomRight:
		return 45, 45
	case FlipBottom:
		return 90, 0
	case FlipBottomLeft:
		return 45, -45
	case FlipLeft:
		return 0, -90
	case FlipTopLeft:
		return -45, -45
	case FlipCenter:
		return 0, 0
	}
	return 0, 0
}

// CalculateFlipDirection détermine la direction de flip basée sur la position
// du clic dans une tuile. tileSize est la taille de la tuile, localX et localY
// sont les coordonnées du clic relatives à la tuile (0,0 = coin supérieur gauche).
// Retourne une des 8 directions périphériques (le côté opposé au curseur se déplace vers le curseur).
func CalculateFlipDirection(tileSize, localX, localY int) FlipDirection {
	// On divise en 3x3 zones
	centerStart := tileSize * 33 / 100
	centerEnd := tileSize * 66 / 100

	var vertical int // 0 = top, 1 = center, 2 = bottom
	if localY < centerStart {
		vertical = 0
	} else if localY > centerEnd {
		vertical = 2
	} else {
		vertical = 1
	}

	var horizontal int // 0 = left, 1 = center, 2 = right
	if localX < centerStart {
		horizontal = 0
	} else if localX > centerEnd {
		horizontal = 2
	} else {
		horizontal = 1
	}

	// Gestion du centre : on force vers le bord le plus proche pour garder 8 animations
	if vertical == 1 && horizontal == 1 {
		mid := tileSize / 2
		dx := localX - mid
		dy := localY - mid
		if Abs(dx) > Abs(dy) {
			if dx < 0 {
				horizontal = 0
			} else {
				horizontal = 2
			}
		} else {
			if dy < 0 {
				vertical = 0
			} else {
				vertical = 2
			}
		}
	}

	// Combine pour obtenir la direction
	switch vertical {
	case 0: // top
		switch horizontal {
		case 0:
			return FlipTopLeft
		case 1:
			return FlipTop
		case 2:
			return FlipTopRight
		}
	case 1: // center (normalement impossible ici suite au forçage)
		switch horizontal {
		case 0:
			return FlipLeft
		case 2:
			return FlipRight
		}
	case 2: // bottom
		switch horizontal {
		case 0:
			return FlipBottomLeft
		case 1:
			return FlipBottom
		case 2:
			return FlipBottomRight
		}
	}

	return FlipTop // Fallback
}

// RotateDirection fait pivoter une direction d'un certain nombre de degrés.
// Supporte les multiples de 45°.
func RotateDirection(d Direction, degrees int) Direction {
	// Map des directions ordonnées circulairement (pas de 45°)
	// N, NE, E, SE, S, SW, W, NW
	order := []Direction{
		DirNorth, DirNorthEast, DirEast, DirSouthEast,
		DirSouth, DirSouthWest, DirWest, DirNorthWest,
	}

	// Trouve l'index actuel
	currentIdx := -1
	for i, dir := range order {
		if dir == d {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return d
	}

	// Calcule le nombre de pas de 45°
	steps := degrees / 45
	newIdx := (currentIdx + steps) % len(order)
	if newIdx < 0 {
		newIdx += len(order)
	}

	return order[newIdx]
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

func (t Type) String() string {
	switch t {
	case TypeResource:
		return "resource"
	case TypeCreature:
		return "creature"
	case TypeStructure:
		return "structure"
	case TypeArtefact:
		return "artefact"
	case TypeTrap:
		return "trap"
	case TypeLoot:
		return "loot"
	case TypeTrack:
		return "track"
	}
	return "unknown"
}

// Position utilitaire
type Position struct {
	X, Y int
}

func (p Position) Add(other Position) Position {
	return Position{X: p.X + other.X, Y: p.Y + other.Y}
}

func (p Position) Distance(other Position) int {
	dx := p.X - other.X
	if dx < 0 {
		dx = -dx
	}
	dy := p.Y - other.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy // Distance de Manhattan
}

func (p Position) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

// Entity est l'interface de base pour tous les éléments du jeu
type Entity interface {
	GetID() ID
	GetType() Type
	GetPosition() Position
	SetPosition(Position)
	GetGridID() string
	SetGridID(string)
	IsActive() bool
	Deactivate()
	GetState() TileState
	SetState(TileState)
	GetTransformation() Transformation
	SetTransformation(Transformation)
	GetOrientation() Direction // Orientation cardinale
	SetOrientation(Direction)
	GetCumulationLevel() int
	SetCumulationLevel(int)
	AddTag(string)
	HasTag(string) bool
	RemoveTag(string)

	IsCumulated() bool

	// Association compliance
	GetMatchID() string
	GetLogicKey() string
	GetElement() string
	GetNarrativeTag() string
	GetMatchTypes() []string
	GetCategory() string

	// Hoverable compliance (Pseudo-ECS logic)
	GetHoverID() string
	IsHoverAllowed() bool
}

// Transformation représente une des 8 symétries du carré (groupe diédrique D4)
type Transformation uint8

const (
	TransIdentity Transformation = iota // 0: e
	TransRot90                          // 1: r
	TransRot180                         // 2: r^2
	TransRot270                         // 3: r^3
	TransMirrorH                        // 4: s (Miroir Horizontal - Médiane Verticale)
	TransMirrorD1                       // 5: sr (Miroir Diagonale \)
	TransMirrorV                        // 6: sr^2 (Miroir Vertical - Médiane Horizontale)
	TransMirrorD2                       // 7: sr^3 (Miroir Diagonale /)
)

var d4Table = [8][8]Transformation{
	{0, 1, 2, 3, 4, 5, 6, 7}, // e
	{1, 2, 3, 0, 7, 4, 5, 6}, // r
	{2, 3, 0, 1, 6, 7, 4, 5}, // r^2
	{3, 0, 1, 2, 5, 6, 7, 4}, // r^3
	{4, 5, 6, 7, 0, 1, 2, 3}, // s
	{5, 6, 7, 4, 3, 0, 1, 2}, // sr
	{6, 7, 4, 5, 2, 3, 0, 1}, // sr^2
	{7, 4, 5, 6, 1, 2, 3, 0}, // sr^3
}

// Compose combine deux transformations (base * apply)
func Compose(base, apply Transformation) Transformation {
	if base > 7 || apply > 7 {
		return base
	}
	// CORRECTION : La table D4 est structurée tel que d4Table[Deuxième_Op][Première_Op]
	// Pour que le joueur agisse SUR la tuile, son clic est la deuxième opération.
	return d4Table[apply][base]
}

// ToTransformation convertit une direction de flip en transformation D4
func (f FlipDirection) ToTransformation() Transformation {
	switch f {
	case FlipLeft, FlipRight:
		return TransMirrorH
	case FlipTop, FlipBottom:
		return TransMirrorV
	case FlipTopRight, FlipBottomLeft:
		return TransMirrorD1
	case FlipTopLeft, FlipBottomRight:
		return TransMirrorD2
	default:
		return TransIdentity
	}
}

// BaseEntity implémentation commune
type BaseEntity struct {
	ID        ID
	EType     Type
	Pos       Position
	GridID    string // ID du grid sur lequel se trouve l'entité
	Active    bool
	State     TileState // L'état appartient maintenant à l'entité
	Transform Transformation
	CumulationLevel int
	Tags      []string
	Metadata  map[string]interface{}
}

func NewTrap(pos Position) *BaseEntity {
	return &BaseEntity{
		ID:        NewID(),
		EType:     TypeTrap,
		Pos:       pos,
		Active:    true,
		State:     Hidden,
		Transform: TransIdentity,
		Tags:      []string{"trap"},
		Metadata:  make(map[string]interface{}),
	}
}

func NewBaseEntity(etype Type) BaseEntity {
	return BaseEntity{
		ID:        NewID(),
		EType:     etype,
		GridID:    "", // Doit être défini après création
		Active:    true,
		State:     Hidden, // Par défaut caché
		Transform: TransIdentity,
		Tags:      make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}
}

// GetGridID retourne l'ID du grid de l'entité
func (e *BaseEntity) GetGridID() string {
	return e.GridID
}

// SetGridID définit l'ID du grid de l'entité
func (e *BaseEntity) SetGridID(gridID string) {
	e.GridID = gridID
}

func (e *BaseEntity) GetID() ID              { return e.ID }
func (e *BaseEntity) GetType() Type          { return e.EType }
func (e *BaseEntity) GetPosition() Position  { return e.Pos }
func (e *BaseEntity) SetPosition(p Position) { e.Pos = p }
func (e *BaseEntity) IsActive() bool         { return e.Active }
func (e *BaseEntity) Deactivate()            { e.Active = false }
func (e *BaseEntity) GetState() TileState    { return e.State }
func (e *BaseEntity) SetState(s TileState)   { e.State = s }

func (e *BaseEntity) GetTransformation() Transformation  { return e.Transform }
func (e *BaseEntity) SetTransformation(t Transformation) { e.Transform = t }

func (e *BaseEntity) GetCumulationLevel() int { return e.CumulationLevel }
func (e *BaseEntity) SetCumulationLevel(l int) { e.CumulationLevel = l }

func (e *BaseEntity) GetOrientation() Direction {
	// Déduit une direction cardinale vers laquelle pointe le "Haut" de l'objet
	// après application de la transformation D4.
	// On transforme DirNorth (0, -1) pour voir où il finit.
	return TransformDirection(DirNorth, e.Transform)
}

func (e *BaseEntity) SetOrientation(o Direction) {
	switch o {
	case DirNorth:
		e.Transform = TransIdentity
	case DirEast:
		e.Transform = TransRot90
	case DirSouth:
		e.Transform = TransRot180
	case DirWest:
		e.Transform = TransRot270
	}
}

func (e *BaseEntity) AddTag(tag string) {
	for _, t := range e.Tags {
		if t == tag {
			return
		}
	}
	e.Tags = append(e.Tags, tag)
}

func (e *BaseEntity) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (e *BaseEntity) RemoveTag(tag string) {
	for i, t := range e.Tags {
		if t == tag {
			e.Tags = append(e.Tags[:i], e.Tags[i+1:]...)
			return
		}
	}
}

func (e *BaseEntity) IsCumulated() bool {
	return e.CumulationLevel > 0
}

func (e *BaseEntity) GetHoverID() string {
	return string(e.ID)
}

func (e *BaseEntity) IsHoverAllowed() bool {
	return e.State&Blocked == 0
}

// Matchable compliance (Default implementations)
func (e *BaseEntity) GetMatchID() string {
	if e.EType == TypeTrap {
		return "trap"
	}
	return ""
}
func (e *BaseEntity) GetLogicKey() string      { return "" }
func (e *BaseEntity) GetElement() string       { return "" }
func (e *BaseEntity) GetNarrativeTag() string  { return "" }
func (e *BaseEntity) GetMatchTypes() []string { return []string{"identical"} }

func (e *BaseEntity) GetCategory() string {
	return e.EType.String()
}

// Manager gère toutes les entités
type Manager struct {
	entities    map[ID]Entity
	byType      map[Type]map[ID]Entity
	byPos       map[Position]ID
	cacheByType map[Type][]Entity
	dirtyTypes  map[Type]bool
	// Cache pour GetAllActive()
	activeCache []Entity
	dirtyActive bool
}

func NewManager() *Manager {
	return &Manager{
		entities:    make(map[ID]Entity),
		byType:      make(map[Type]map[ID]Entity),
		byPos:       make(map[Position]ID),
		cacheByType: make(map[Type][]Entity),
		dirtyTypes:  make(map[Type]bool),
	}
}

func (m *Manager) Register(e Entity) {
	if e.GetType() == TypeLoot {
		fmt.Printf("[ENTITY] Enregistrement d'un LOOT: %s (ID: %s, State: %s)\n", e.GetCategory(), e.GetID(), e.GetState())
	}
	m.entities[e.GetID()] = e
	if m.byType[e.GetType()] == nil {
		m.byType[e.GetType()] = make(map[ID]Entity)
	}
	m.byType[e.GetType()][e.GetID()] = e
	m.byPos[e.GetPosition()] = e.GetID()

	// Invalide le cache pour ce type
	m.dirtyTypes[e.GetType()] = true
	m.dirtyActive = true
}

func (m *Manager) Remove(id ID) {
	e, ok := m.entities[id]
	if !ok {
		return
	}
	delete(m.entities, id)
	delete(m.byType[e.GetType()], id)
	delete(m.byPos, e.GetPosition())

	// Invalide le cache pour ce type
	m.dirtyTypes[e.GetType()] = true
	m.dirtyActive = true
}

func (m *Manager) Get(id ID) (Entity, bool) {
	e, ok := m.entities[id]
	return e, ok
}

func (m *Manager) GetByPosition(pos Position) (Entity, bool) {
	id, ok := m.byPos[pos]
	if !ok {
		return nil, false
	}
	return m.Get(id)
}

func (m *Manager) UpdatePosition(id ID, newPos Position) error {
	e, ok := m.entities[id]
	if !ok {
		return fmt.Errorf("entité %s non trouvée", id)
	}
	delete(m.byPos, e.GetPosition())
	e.SetPosition(newPos)
	m.byPos[newPos] = id
	return nil
}

func (m *Manager) GetByType(t Type) []Entity {
	// Retourne le cache s'il est à jour
	if !m.dirtyTypes[t] && m.cacheByType[t] != nil {
		return m.cacheByType[t]
	}

	result := make([]Entity, 0, len(m.byType[t]))
	for _, e := range m.byType[t] {
		result = append(result, e)
	}
	// Trie par ID pour avoir un ordre stable entre les frames
	sort.Slice(result, func(i, j int) bool {
		return result[i].GetID() < result[j].GetID()
	})

	// Met à jour le cache
	m.cacheByType[t] = result
	m.dirtyTypes[t] = false

	return result
}

func (m *Manager) GetAllActive() []Entity {
	if !m.dirtyActive && m.activeCache != nil {
		return m.activeCache
	}
	result := make([]Entity, 0)
	for _, e := range m.entities {
		if e.IsActive() {
			result = append(result, e)
		}
	}
	m.activeCache = result
	m.dirtyActive = false
	return result
}

func (m *Manager) QueryByTag(tag string) []Entity {
	result := make([]Entity, 0)
	for _, e := range m.entities {
		if e.HasTag(tag) && e.IsActive() {
			result = append(result, e)
		}
	}
	return result
}

func (m *Manager) Count() int {
	return len(m.entities)
}

func (m *Manager) CountByType(t Type) int {
	return len(m.byType[t])
}

func (t Transformation) String() string {
	switch t {
	case TransIdentity:
		return "Identity (0°)"
	case TransRot90:
		return "Rot90 (90°)"
	case TransRot180:
		return "Rot180 (180°)"
	case TransRot270:
		return "Rot270 (270°)"
	case TransMirrorH:
		return "MirrorH (Flip G/D)"
	case TransMirrorV:
		return "MirrorV (Flip H/B)"
	case TransMirrorD1:
		return "MirrorD1 (Diago \\)"
	case TransMirrorD2:
		return "MirrorD2 (Diago /)"
	default:
		return "Unknown"
	}
}

type Track struct {
	BaseEntity
	Kind     string   // "mud", "claws", "broken_grass", etc.
	Duration int      // Nombre de tours restants avant disparition
	FromPos  Position // Case de départ (équivaut à e.Pos)
	ToPos    Position // Case d'arrivée du monstre
	OffsetX  float64  // Décalage X depuis le centre de la tuile (pour positionnement sur le bord)
	OffsetY  float64  // Décalage Y depuis le centre de la tuile
	Angle    float64  // Angle de rotation en radians (pour orienter vers le centre)
}

func NewTrack(kind string, duration int, from, to Position) *Track {
	t := &Track{
		BaseEntity: NewBaseEntity(TypeTrack),
		Kind:       kind,
		Duration:   duration,
		FromPos:    from,
		ToPos:      to,
	}
	t.SetPosition(from) // Par défaut on le lie à la case de départ
	t.AddTag("track")
	t.AddTag(kind)
	return t
}
