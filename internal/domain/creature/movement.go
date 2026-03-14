package creature

import (
	"math"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// ============================================================================
// DÉCLENCHEURS (Triggers) - Quand la créature se déplace
// ============================================================================

type TriggerType string

const (
	TriggerPassive    TriggerType = "passive"    // Aucun mouvement (Ressource fixe)
	TriggerAuto       TriggerType = "auto"       // Se déplace à la fin de chaque tour
	TriggerOnReveal   TriggerType = "on_reveal"  // Se déplace dès qu'elle est révélée
	TriggerOnEcho     TriggerType = "on_echo"    // Se déplace si une autre tuile est révélée ailleurs
	TriggerProximity  TriggerType = "proximity"  // Se déplace si action dans rayon N cases
)

// MovementTrigger définit quand la créature se déplace
type MovementTrigger struct {
	Type          TriggerType
	Radius        int  // Pour Proximity: rayon de détection
	Triggered     bool // État: a été déclenché ce tour
	WasRevealed   bool // Pour OnReveal: était révélée au tour précédent
}

// ShouldTrigger vérifie si le déplacement doit se déclencher
func (mt *MovementTrigger) ShouldTrigger(world WorldQuery, creature *Creature) bool {
	switch mt.Type {
	case TriggerPassive:
		return false
	case TriggerAuto:
		return true
	case TriggerOnReveal:
		// Se déclenche quand la créature devient visible
		isRevealed := world.IsTileRevealed(creature.GetPosition())
		if isRevealed && !mt.WasRevealed {
			mt.WasRevealed = true
			return true
		}
		mt.WasRevealed = isRevealed
		return false
	case TriggerOnEcho:
		// Se déclenche quand UNE AUTRE tuile est révélée
		return mt.Triggered
	case TriggerProximity:
		// Vérifie si une action a eu lieu dans le rayon
		return mt.checkProximity(world, creature.GetPosition())
	}
	return false
}

func (mt *MovementTrigger) checkProximity(world WorldQuery, pos entity.Position) bool {
	// Vérifie les tuiles dans le rayon pour des actions récentes
	for x := -mt.Radius; x <= mt.Radius; x++ {
		for y := -mt.Radius; y <= mt.Radius; y++ {
			if math.Abs(float64(x))+math.Abs(float64(y)) <= float64(mt.Radius) {
				checkPos := entity.Position{X: pos.X + x, Y: pos.Y + y}
				if world.WasTileRecentlyRevealed(checkPos) {
					return true
				}
			}
		}
	}
	return false
}

// Reset remet l'état du déclencheur
func (mt *MovementTrigger) Reset() {
	mt.Triggered = false
}

// Trigger force le déclenchement (pour Echo)
func (mt *MovementTrigger) Trigger() {
	mt.Triggered = true
}

// ============================================================================
// LOGIQUE DE CIBLE (Navigation) - Où la créature va
// ============================================================================

type NavigationType string

const (
	NavWander       NavigationType = "wander"       // Errance: direction aléatoire
	NavPatrol       NavigationType = "patrol"       // Patrouille: suit un itinéraire
	NavOrientation  NavigationType = "orientation"  // D'après l'orientation de la créature
	NavAttraction   NavigationType = "attraction"   // Vise une cible spécifique
	NavRepulsion    NavigationType = "repulsion"    // S'éloigne de la cible
)

type TargetType string

const (
	TargetResource    TargetType = "resource"
	TargetCursor      TargetType = "cursor"
	TargetCreature    TargetType = "creature"
	TargetStructure   TargetType = "structure"
	TargetEmpty       TargetType = "empty"
	TargetPlayer      TargetType = "player"
)

// NavigationLogic définit comment la créature choisit sa destination
type NavigationLogic struct {
	Type           NavigationType
	Target         TargetType      // Pour Attraction/Repulsion
	PatrolRoute    []entity.Position // Pour Patrouille
	PatrolIndex    int             // Index actuel dans la route
	WanderBias     entity.Position // Direction privilégiée pour errance
}

// DecideDirection retourne la direction choisie
func (nl *NavigationLogic) DecideDirection(world WorldQuery, creature *Creature) entity.Position {
	switch nl.Type {
	case NavWander:
		return nl.wander(world, creature)
	case NavPatrol:
		return nl.patrol(world, creature)
	case NavOrientation:
		return nl.followOrientation(creature)
	case NavAttraction:
		return nl.moveToward(world, creature)
	case NavRepulsion:
		return nl.moveAway(world, creature)
	}
	return entity.Position{X: 0, Y: 0}
}

func (nl *NavigationLogic) wander(world WorldQuery, creature *Creature) entity.Position {
	directions := []entity.Position{
		{X: 0, Y: -1}, {X: 0, Y: 1},
		{X: -1, Y: 0}, {X: 1, Y: 0},
	}
	
	// 30% de chance de suivre la direction privilégiée
	if nl.WanderBias != (entity.Position{}) && rand.Float32() < 0.3 {
		if world.IsValidMove(entity.Position{
			X: creature.GetPosition().X + nl.WanderBias.X,
			Y: creature.GetPosition().Y + nl.WanderBias.Y,
		}) {
			return nl.WanderBias
		}
	}
	
	// Direction aléatoire
	return directions[rand.Intn(len(directions))]
}

func (nl *NavigationLogic) patrol(world WorldQuery, creature *Creature) entity.Position {
	if len(nl.PatrolRoute) == 0 {
		return nl.wander(world, creature)
	}
	
	// Va vers le prochain point de patrouille
	target := nl.PatrolRoute[nl.PatrolIndex]
	current := creature.GetPosition()
	
	dir := entity.Position{
		X: Sign(target.X - current.X),
		Y: Sign(target.Y - current.Y),
	}
	
	// Si on est arrivé, passe au point suivant
	if dir.X == 0 && dir.Y == 0 {
		nl.PatrolIndex = (nl.PatrolIndex + 1) % len(nl.PatrolRoute)
		target = nl.PatrolRoute[nl.PatrolIndex]
		dir = entity.Position{
			X: Sign(target.X - current.X),
			Y: Sign(target.Y - current.Y),
		}
	}
	
	return dir
}

func (nl *NavigationLogic) followOrientation(creature *Creature) entity.Position {
	// Retourne la direction de l'orientation actuelle
	if orient, ok := creature.GetComponent("orientation").(*Orientation); ok {
		return orient.ToVector()
	}
	// Par défaut: vers le nord
	return entity.Position{X: 0, Y: -1}
}

func (nl *NavigationLogic) moveToward(world WorldQuery, creature *Creature) entity.Position {
	target := world.FindNearestTarget(creature.GetPosition(), nl.Target)
	if target == nil {
		return nl.wander(world, creature)
	}
	
	current := creature.GetPosition()
	return entity.Position{
		X: Sign(target.X - current.X),
		Y: Sign(target.Y - current.Y),
	}
}

func (nl *NavigationLogic) moveAway(world WorldQuery, creature *Creature) entity.Position {
	target := world.FindNearestTarget(creature.GetPosition(), nl.Target)
	if target == nil {
		return nl.wander(world, creature)
	}
	
	current := creature.GetPosition()
	return entity.Position{
		X: Sign(current.X - target.X),
		Y: Sign(current.Y - target.Y),
	}
}

// ============================================================================
// MODE DE DÉPLACEMENT - Comment la créature se déplace
// ============================================================================

type MoveMode string

const (
	ModeBento      MoveMode = "bento"      // Visible: la tuile glisse
	ModeShadow     MoveMode = "shadow"     // Invisible: permute face cachée
	ModeSwap       MoveMode = "swap"       // Inversion: échange avec la cible
	ModeOver       MoveMode = "over"       // Sur les tuiles: masque une tuile
	ModeUnder      MoveMode = "under"      // Sous les tuiles: caché par une tuile
)

// MovementMode définit le mode visuel/mécanique du déplacement
type MovementMode struct {
	Type      MoveMode
	SwapMode  bool   // Si true, échange avec l'occupant de la case cible
}

// ApplyMovement applique le mouvement selon le mode
// Cette méthode est utilisée par les systèmes externes
func (mm *MovementMode) ApplyMovement(world ExtendedWorldState, creature *Creature, newPos entity.Position) bool {
	switch mm.Type {
	case ModeBento:
		return world.MoveEntity(creature, newPos)
	case ModeShadow:
		if world.CanMoveTo(newPos) {
			world.MoveEntitySilent(creature, newPos)
			return true
		}
		return false
	case ModeSwap:
		return world.SwapEntities(creature.GetPosition(), newPos)
	case ModeOver:
		creature.AddTag("flying")
		return world.MoveEntity(creature, newPos)
	case ModeUnder:
		creature.AddTag("burrowed")
		return world.MoveEntitySilent(creature, newPos)
	}
	return false
}

// ============================================================================
// FRÉQUENCE (Rythme) - À quelle fréquence la créature se déplace
// ============================================================================

type FrequencyType string

const (
	FreqVelocity   FrequencyType = "velocity"   // V cases par tour
	FreqDelay      FrequencyType = "delay"      // Tous les T tours
	FreqInstant    FrequencyType = "instant"    // Immédiat
)

// MovementFrequency définit le rythme du déplacement
type MovementFrequency struct {
	Type        FrequencyType
	Velocity    int   // Nombre de cases par tour
	Delay       int   // Tours entre deux mouvements
	TurnCounter int   // Compteur interne
}

// CanMove vérifie si la créature peut se déplacer ce tour
func (mf *MovementFrequency) CanMove() bool {
	switch mf.Type {
	case FreqVelocity:
		return true // La vélocité est gérée dans le mouvement
	case FreqDelay:
		mf.TurnCounter++
		if mf.TurnCounter >= mf.Delay {
			mf.TurnCounter = 0
			return true
		}
		return false
	case FreqInstant:
		return true
	}
	return false
}

// GetMoveCount retourne le nombre de cases à déplacer ce tour
func (mf *MovementFrequency) GetMoveCount() int {
	switch mf.Type {
	case FreqVelocity:
		return mf.Velocity
	case FreqDelay, FreqInstant:
		return 1
	}
	return 1
}

// ============================================================================
// ORIENTATION & DIRECTION RELATIVE
// ============================================================================

type Direction int

const (
	DirNorth Direction = iota
	DirEast
	DirSouth
	DirWest
)

type RelativeDirection string

const (
	RelForward   RelativeDirection = "forward"   // Marche avant
	RelBackward  RelativeDirection = "backward"  // Recul
	RelLeft      RelativeDirection = "left"      // Latéral gauche
	RelRight     RelativeDirection = "right"     // Latéral droit
	RelDiagFL    RelativeDirection = "diag_fl"   // Diagonal avant-gauche
	RelDiagFR    RelativeDirection = "diag_fr"   // Diagonal avant-droit
	RelDiagBL    RelativeDirection = "diag_bl"   // Diagonal arrière-gauche
	RelDiagBR    RelativeDirection = "diag_br"   // Diagonal arrière-droit
	RelRotate90  RelativeDirection = "rotate_90" // Rotation 90°
	RelRotate180 RelativeDirection = "rotate_180"// Rotation 180°
)

// Orientation définit la direction vers laquelle regarde la créature
type Orientation struct {
	Direction Direction
}

func (o *Orientation) Type() string { return "orientation" }

func (o *Orientation) ToVector() entity.Position {
	switch o.Direction {
	case DirNorth:
		return entity.Position{X: 0, Y: -1}
	case DirEast:
		return entity.Position{X: 1, Y: 0}
	case DirSouth:
		return entity.Position{X: 0, Y: 1}
	case DirWest:
		return entity.Position{X: -1, Y: 0}
	}
	return entity.Position{X: 0, Y: -1}
}

func (o *Orientation) Rotate(degrees int) {
	switch degrees {
	case 90:
		o.Direction = Direction((int(o.Direction) + 1) % 4)
	case 180:
		o.Direction = Direction((int(o.Direction) + 2) % 4)
	case 270, -90:
		o.Direction = Direction((int(o.Direction) + 3) % 4)
	}
}

// GetRelativeDirection convertit une direction relative en vecteur absolu
func (o *Orientation) GetRelativeDirection(rel RelativeDirection) entity.Position {
	switch rel {
	case RelForward:
		return o.ToVector()
	case RelBackward:
		vec := o.ToVector()
		return entity.Position{X: -vec.X, Y: -vec.Y}
	case RelLeft:
		switch o.Direction {
		case DirNorth:
			return entity.Position{X: -1, Y: 0}
		case DirEast:
			return entity.Position{X: 0, Y: -1}
		case DirSouth:
			return entity.Position{X: 1, Y: 0}
		case DirWest:
			return entity.Position{X: 0, Y: 1}
		}
	case RelRight:
		switch o.Direction {
		case DirNorth:
			return entity.Position{X: 1, Y: 0}
		case DirEast:
			return entity.Position{X: 0, Y: 1}
		case DirSouth:
			return entity.Position{X: -1, Y: 0}
		case DirWest:
			return entity.Position{X: 0, Y: -1}
		}
	case RelDiagFL:
		forward := o.ToVector()
		left := o.GetRelativeDirection(RelLeft)
		return entity.Position{X: forward.X + left.X, Y: forward.Y + left.Y}
	case RelDiagFR:
		forward := o.ToVector()
		right := o.GetRelativeDirection(RelRight)
		return entity.Position{X: forward.X + right.X, Y: forward.Y + right.Y}
	case RelDiagBL:
		backward := o.GetRelativeDirection(RelBackward)
		left := o.GetRelativeDirection(RelLeft)
		return entity.Position{X: backward.X + left.X, Y: backward.Y + left.Y}
	case RelDiagBR:
		backward := o.GetRelativeDirection(RelBackward)
		right := o.GetRelativeDirection(RelRight)
		return entity.Position{X: backward.X + right.X, Y: backward.Y + right.Y}
	}
	return entity.Position{X: 0, Y: 0}
}

// ============================================================================
// CONTRAINTES DE TERRAIN (Collision)
// ============================================================================

type CollisionType string

const (
	CollideStop      CollisionType = "stop"       // S'arrête devant obstacle
	CollideBounce    CollisionType = "bounce"     // Rebondit (180°)
	CollideSlide     CollisionType = "slide"      // Contourne (glisse)
	CollidePhase     CollisionType = "phase"      // Traversée (spectres)
)

// CollisionHandler gère les collisions
type CollisionHandler struct {
	Type          CollisionType
	CanPhaseThrough []string // Types de tuiles traversables (pour Phase)
}

// HandleCollision gère une collision et retourne la nouvelle direction/position
func (ch *CollisionHandler) HandleCollision(world WorldQuery, creature *Creature, attemptedPos entity.Position) (entity.Position, bool) {
	switch ch.Type {
	case CollideStop:
		return creature.GetPosition(), false // Reste sur place
	case CollideBounce:
		// Inverse l'orientation
		if orient, ok := creature.GetComponent("orientation").(*Orientation); ok {
			orient.Rotate(180)
		}
		return creature.GetPosition(), false
	case CollideSlide:
		// Essaie de glisser le long de l'obstacle
		return ch.trySlide(world, creature, attemptedPos)
	case CollidePhase:
		// Vérifie si on peut traverser
		if ch.canPhase(world, attemptedPos) {
			return attemptedPos, true
		}
		return creature.GetPosition(), false
	}
	return creature.GetPosition(), false
}

func (ch *CollisionHandler) trySlide(world WorldQuery, creature *Creature, attemptedPos entity.Position) (entity.Position, bool) {
	current := creature.GetPosition()
	dx := attemptedPos.X - current.X
	dy := attemptedPos.Y - current.Y
	
	// Essaie de glisser horizontalement si bloqué verticalement
	if dy != 0 {
		slidePos := entity.Position{X: current.X, Y: attemptedPos.Y}
		if world.IsValidMove(slidePos) {
			return slidePos, true
		}
	}
	
	// Essaie de glisser verticalement si bloqué horizontalement
	if dx != 0 {
		slidePos := entity.Position{X: attemptedPos.X, Y: current.Y}
		if world.IsValidMove(slidePos) {
			return slidePos, true
		}
	}
	
	// Essaie les directions latérales
	lateral := []entity.Position{
		{X: current.X + 1, Y: current.Y},
		{X: current.X - 1, Y: current.Y},
		{X: current.X, Y: current.Y + 1},
		{X: current.X, Y: current.Y - 1},
	}
	
	for _, pos := range lateral {
		if world.IsValidMove(pos) {
			return pos, true
		}
	}
	
	return current, false
}

func (ch *CollisionHandler) canPhase(world WorldQuery, pos entity.Position) bool {
	tileType := world.GetTileType(pos)
	for _, t := range ch.CanPhaseThrough {
		if t == tileType {
			return true
		}
	}
	return false
}

// ============================================================================
// CONFIGURATION COMPLÈTE DU MOUVEMENT
// ============================================================================

// MovementProfile regroupe toute la configuration de déplacement
type MovementProfile struct {
	Trigger    MovementTrigger
	Navigation NavigationLogic
	Mode       MovementMode
	Frequency  MovementFrequency
	Orientation Orientation
	Collision  CollisionHandler
}

// MovementRequest représente une intention de mouvement
type MovementRequest struct {
	Creature      *Creature
	From          entity.Position
	To            entity.Position
	Direction     RelativeDirection
	IsRotation    bool
}

// MovementResult représente le résultat d'un mouvement
type MovementResult struct {
	Success       bool
	FinalPosition entity.Position
	Rotated       bool
	NewDirection  Direction
	SwappedWith   string // ID de l'entité échangée (si ModeSwap)
}

// ============================================================================
// INTERFACES ÉTENDUES POUR LE MOUVEMENT
// ============================================================================

// WorldQuery interface pour les requêtes sur le monde
type WorldQuery interface {
	WorldState
	IsTileRevealed(pos entity.Position) bool
	WasTileRecentlyRevealed(pos entity.Position) bool
	FindNearestTarget(from entity.Position, targetType TargetType) *entity.Position
	GetTileType(pos entity.Position) string
	GetEntitiesAt(pos entity.Position) []entity.Entity
}

// ExtendedWorldState interface étendue pour les actions de mouvement
type ExtendedWorldState interface {
	WorldState
	MoveEntity(creature *Creature, newPos entity.Position) bool
	MoveEntitySilent(creature *Creature, newPos entity.Position) bool
	CanMoveTo(pos entity.Position) bool
	SwapEntities(pos1, pos2 entity.Position) bool
}

// ============================================================================
// UTILITAIRES
// ============================================================================

// Sign retourne le signe d'un entier (-1, 0, ou 1)
func Sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// DefaultMovementProfile retourne un profil de mouvement par défaut
func DefaultMovementProfile() *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{
			Type: TriggerAuto,
		},
		Navigation: NavigationLogic{
			Type: NavWander,
		},
		Mode: MovementMode{
			Type: ModeBento,
		},
		Frequency: MovementFrequency{
			Type:     FreqDelay,
			Delay:    1,
		},
		Orientation: Orientation{
			Direction: DirNorth,
		},
		Collision: CollisionHandler{
			Type: CollideStop,
		},
	}
}

// PassiveProfile créature immobile (ressource fixe)
func PassiveProfile() *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{
			Type: TriggerPassive,
		},
		Navigation: NavigationLogic{
			Type: NavWander,
		},
		Mode: MovementMode{
			Type: ModeBento,
		},
		Frequency: MovementFrequency{
			Type: FreqInstant,
		},
		Orientation: Orientation{
			Direction: DirNorth,
		},
		Collision: CollisionHandler{
			Type: CollideStop,
		},
	}
}

// HunterProfile chasseur agressif
func HunterProfile() *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{
			Type: TriggerAuto,
		},
		Navigation: NavigationLogic{
			Type:   NavAttraction,
			Target: TargetPlayer,
		},
		Mode: MovementMode{
			Type: ModeBento,
		},
		Frequency: MovementFrequency{
			Type:     FreqVelocity,
			Velocity: 2,
		},
		Orientation: Orientation{
			Direction: DirNorth,
		},
		Collision: CollisionHandler{
			Type: CollideBounce,
		},
	}
}

// FleeingProfile créature qui fuit
func FleeingProfile() *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{
			Type: TriggerProximity,
			Radius: 3,
		},
		Navigation: NavigationLogic{
			Type:   NavRepulsion,
			Target: TargetPlayer,
		},
		Mode: MovementMode{
			Type: ModeShadow,
		},
		Frequency: MovementFrequency{
			Type:     FreqVelocity,
			Velocity: 1,
		},
		Orientation: Orientation{
			Direction: DirNorth,
		},
		Collision: CollisionHandler{
			Type: CollideSlide,
		},
	}
}

// SpecterProfile spectre qui traverse les murs
func SpecterProfile() *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{
			Type: TriggerOnEcho,
		},
		Navigation: NavigationLogic{
			Type: NavWander,
		},
		Mode: MovementMode{
			Type: ModeShadow,
		},
		Frequency: MovementFrequency{
			Type:     FreqDelay,
			Delay:    2,
		},
		Orientation: Orientation{
			Direction: DirNorth,
		},
		Collision: CollisionHandler{
			Type:            CollidePhase,
			CanPhaseThrough: []string{"wall", "structure", "architecture"},
		},
	}
}

// PatrollerProfile garde qui patrouille
func PatrollerProfile(route []entity.Position) *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{
			Type: TriggerAuto,
		},
		Navigation: NavigationLogic{
			Type:        NavPatrol,
			PatrolRoute: route,
		},
		Mode: MovementMode{
			Type: ModeBento,
		},
		Frequency: MovementFrequency{
			Type:  FreqDelay,
			Delay: 1,
		},
		Orientation: Orientation{
			Direction: DirNorth,
		},
		Collision: CollisionHandler{
			Type: CollideStop,
		},
	}
}
