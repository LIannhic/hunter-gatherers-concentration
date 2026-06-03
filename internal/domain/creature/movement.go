package creature

import (
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// ============================================================================
// DÉCLENCHEURS (Triggers) - Quand la créature se déplace
// ============================================================================

type TriggerType string

const (
	TriggerPassive   TriggerType = "passive"   // Aucun mouvement (Ressource fixe)
	TriggerAuto      TriggerType = "auto"      // Se déplace à la fin de chaque tour
	TriggerOnReveal  TriggerType = "on_reveal" // Se déplace dès qu'elle est révélée (Memory)
	TriggerOnEcho    TriggerType = "on_echo"   // Se déplace quand une tuile spécifique est révélée
	TriggerProximity TriggerType = "proximity" // Se déplace si action dans rayon N cases
)

type MovementTrigger struct {
	Type        TriggerType
	Radius      int  // Pour Proximity: rayon de détection
	Triggered   bool // État: a été déclenché ce tour
	WasRevealed bool // Pour OnReveal: était révélée au tour précédent
}

func (mt *MovementTrigger) ShouldTrigger(world WorldQuery, creature *Creature) bool {
	switch mt.Type {
	case TriggerPassive:
		return false
	case TriggerAuto:
		return true
	case TriggerOnReveal:
		isRevealed := world.IsTileRevealed(creature.GetPosition())
		if isRevealed && !mt.WasRevealed {
			mt.WasRevealed = true
			return true
		}
		mt.WasRevealed = isRevealed
		return false
	case TriggerOnEcho:
		return mt.Triggered
	case TriggerProximity:
		return mt.checkProximity(world, creature.GetPosition())
	}
	return false
}

func (mt *MovementTrigger) checkProximity(world WorldQuery, pos entity.Position) bool {
	for x := -mt.Radius; x <= mt.Radius; x++ {
		for y := -mt.Radius; y <= mt.Radius; y++ {
			if entity.Abs(x)+entity.Abs(y) <= mt.Radius {
				checkPos := entity.Position{X: pos.X + x, Y: pos.Y + y}
				if world.WasTileRecentlyRevealed(checkPos) {
					return true
				}
			}
		}
	}
	return false
}

func (mt *MovementTrigger) Reset()   { mt.Triggered = false }
func (mt *MovementTrigger) Trigger() { mt.Triggered = true }

// ============================================================================
// LOGIQUE DE CIBLE (Navigation) - Où la créature va
// ============================================================================

type NavigationType string

const (
	NavWander      NavigationType = "wander"      // Errance directionnelle
	NavPatrol      NavigationType = "patrol"      // Suit un itinéraire
	NavRelative    NavigationType = "relative"    // Par rapport à sa position actuelle
	NavOrientation NavigationType = "orientation" // D'après l'orientation diédrique
	NavAttraction  NavigationType = "attraction"  // Vise une cible spécifique
	NavRepulsion   NavigationType = "repulsion"   // S'éloigne de la cible
)

type TargetType string

const (
	TargetResource  TargetType = "resource"
	TargetCursor    TargetType = "cursor"
	TargetCreature  TargetType = "creature"
	TargetStructure TargetType = "structure"
	TargetEmpty     TargetType = "empty"
	TargetPlayer    TargetType = "player"
)

type NavigationLogic struct {
	Type        NavigationType
	Target      TargetType
	PatrolRoute []entity.Position
	PatrolIndex int
	WanderBias  entity.Position
	TargetName  string // Optionnel: nom/ID spécifique de la cible (ex: "dreamberry")
}

func (nl *NavigationLogic) DecideDirection(world WorldQuery, creature *Creature) entity.Position {
	switch nl.Type {
	case NavWander:
		return nl.wander(world, creature)
	case NavPatrol:
		return nl.patrol(world, creature)
	case NavRelative:
		return nl.relative(world, creature)
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
	if creature != nil && world != nil && nl.WanderBias != (entity.Position{}) && rand.Float32() < 0.3 {
		targetPos := entity.Position{
			X: creature.GetPosition().X + nl.WanderBias.X,
			Y: creature.GetPosition().Y + nl.WanderBias.Y,
		}
		if world.IsValidMove(targetPos) {
			return nl.WanderBias
		}
	}
	return directions[rand.Intn(len(directions))]
}

func (nl *NavigationLogic) patrol(world WorldQuery, creature *Creature) entity.Position {
	if len(nl.PatrolRoute) == 0 {
		return nl.wander(world, creature)
	}
	target := nl.PatrolRoute[nl.PatrolIndex]
	current := creature.GetPosition()
	dir := entity.Position{X: entity.Sign(target.X - current.X), Y: entity.Sign(target.Y - current.Y)}

	if dir.X == 0 && dir.Y == 0 {
		nl.PatrolIndex = (nl.PatrolIndex + 1) % len(nl.PatrolRoute)
		target = nl.PatrolRoute[nl.PatrolIndex]
		dir = entity.Position{X: entity.Sign(target.X - current.X), Y: entity.Sign(target.Y - current.Y)}
	}
	return dir
}

func (nl *NavigationLogic) relative(world WorldQuery, creature *Creature) entity.Position {
	if len(nl.PatrolRoute) == 0 {
		return nl.wander(world, creature)
	}

	// 1. On récupère la direction de base définie dans notre pattern (ex: {X: 0, Y: -1} voulant dire "En avant")
	baseDir := nl.PatrolRoute[nl.PatrolIndex]

	// 2. On prend en compte l'orientation actuelle de la créature.
	// Si la créature regarde à l'Est, son "En avant" ({X:0, Y:-1} local) doit devenir un mouvement vers l'Est spatial.
	orient := creature.GetOrientation()

	// On applique la rotation de l'orientation au vecteur baseDir
	finalDir := applyOrientationToVector(orient, baseDir)

	// 3. On calcule la case du monde visée pour tester sa validité
	targetPos := entity.Position{
		X: creature.GetPosition().X + finalDir.X,
		Y: creature.GetPosition().Y + finalDir.Y,
	}

	// 4. La créature attend sagement sur place. Si l'obstacle bouge ou s'ouvre, elle reprendra son pattern exact.
	if !world.IsValidMove(targetPos) {
		return entity.Position{X: 0, Y: 0}
	}

	// 5. Le mouvement est valide ! On passe à l'étape suivante du pattern pour le prochain tour
	nl.PatrolIndex = (nl.PatrolIndex + 1) % len(nl.PatrolRoute)

	return finalDir
}

// Fonction utilitaire pour adapter un vecteur (X, Y) local à l'orientation absolue de la grille
func applyOrientationToVector(dir entity.Direction, localDir entity.Position) entity.Position {
	switch dir {
	case entity.DirNorth:
		return localDir // Pas de changement
	case entity.DirEast:
		// Une rotation de 90° horaire : (X, Y) devient (-Y, X)
		return entity.Position{X: -localDir.Y, Y: localDir.X}
	case entity.DirSouth:
		// Une rotation de 180° : (X, Y) devient (-X, -Y)
		return entity.Position{X: -localDir.X, Y: -localDir.Y}
	case entity.DirWest:
		// Une rotation de 270° : (X, Y) devient (Y, -X)
		return entity.Position{X: localDir.Y, Y: -localDir.X}
	}
	return localDir
}

func (nl *NavigationLogic) followOrientation(creature *Creature) entity.Position {
	if orient, ok := creature.GetComponent("orientation").(*Orientation); ok {
		return orient.ToVector()
	}
	return entity.Position{X: 0, Y: -1}
}

func (nl *NavigationLogic) moveToward(world WorldQuery, creature *Creature) entity.Position {
	target := world.FindNearestTarget(creature.GetPosition(), nl.Target)
	if target == nil {
		return nl.wander(world, creature)
	}
	return entity.Position{X: entity.Sign(target.X - creature.GetPosition().X), Y: entity.Sign(target.Y - creature.GetPosition().Y)}
}

func (nl *NavigationLogic) moveAway(world WorldQuery, creature *Creature) entity.Position {
	target := world.FindNearestTarget(creature.GetPosition(), nl.Target)
	if target == nil {
		return nl.wander(world, creature)
	}
	return entity.Position{X: entity.Sign(creature.GetPosition().X - target.X), Y: entity.Sign(creature.GetPosition().Y - target.Y)}
}

// ============================================================================
// REFACTORISATION : PHYSIQUE DU MOUVEMENT (Couches / Spatialité)
// ============================================================================

type MoveMode string

const (
	ModeNormal MoveMode = "normal" // Déplacement standard au sol
	ModeOver   MoveMode = "over"   // Passe au-dessus des autres tuiles
	ModeUnder  MoveMode = "under"  // Passe en dessous des autres tuiles
	ModeSwap   MoveMode = "swap"   // Interversion physique de deux tuiles
)

type MovementMode struct {
	Type     MoveMode
	SwapMode bool
}

func (mm *MovementMode) ApplyMovement(world ExtendedWorldState, creature *Creature, newPos entity.Position) bool {
	switch mm.Type {
	case ModeNormal:
		return world.MoveEntity(creature, newPos)
	case ModeSwap:
		return world.SwapEntities(creature.GetPosition(), newPos)
	case ModeOver:
		creature.AddTag("flying")
		return world.MoveEntity(creature, newPos)
	case ModeUnder:
		creature.AddTag("burrowed")
		// Reste géré par MoveEntity car la visibilité graphique dépend désormais de StealthLevel
		return world.MoveEntity(creature, newPos)
	}
	return false
}

// ============================================================================
// NOUVEAU : RÈGLES DE PERCEPTION (Furtivité, Acoustique, Indices)
// ============================================================================

type StealthLevel string

const (
	StealthManifest StealthLevel = "manifest" // La tuile glisse visiblement (ex: Bento)
	StealthCloaked  StealthLevel = "cloaked"  // Déplacement invisible à l'œil nu (ex: Shadow)
)

type AcousticLevel string

const (
	AcousticSilent AcousticLevel = "silent" // Aucun bruit généré
	AcousticEcho   AcousticLevel = "echo"   // Émet un stimulus sonore
)

// PerceptionProfile régit comment le monde/joueur perçoit les actions de cette entité
type PerceptionProfile struct {
	Stealth  StealthLevel  // Visibilité de la translation de la tuile
	Acoustic AcousticLevel // Niveau sonore de la tuile en mouvement

	// Indices du Passé (A posteriori)
	LeavesTracks  bool
	TrackType     string // Ex: "mud", "broken_grass", "scent"
	TrackDuration int    // Nombre de tours avant disparition

	// Indices du Futur (A priori)
	TelegraphsIntent bool // Indique des intentions d'attaque, de mouvement, etc...
}

// ============================================================================
// FRÉQUENCE (Rythme) - À quelle fréquence la créature se déplace
// ============================================================================

type FrequencyType string

const (
	FreqVelocity FrequencyType = "velocity"
	FreqDelay    FrequencyType = "delay"
	FreqInstant  FrequencyType = "instant"
)

type MovementFrequency struct {
	Type          FrequencyType
	Velocity      int
	Delay         int
	TurnCounter   int
	TurnLastMoved int
}

func (mf *MovementFrequency) CanMove() bool {
	switch mf.Type {
	case FreqVelocity, FreqInstant:
		return true
	case FreqDelay:
		mf.TurnCounter++
		if mf.TurnCounter >= mf.Delay {
			mf.TurnCounter = 0
			return true
		}
		return false
	}
	return false
}

func (mf *MovementFrequency) GetMoveCount() int {
	if mf.Type == FreqVelocity {
		return mf.Velocity
	}
	return 1
}

func (mf *MovementFrequency) HasMovedThisTurn(turn int) bool {
	return mf.TurnLastMoved == turn
}

func (mf *MovementFrequency) MarkMoved(turn int) {
	mf.TurnLastMoved = turn
}

// ============================================================================
// ORIENTATION, COLLISION & CONFIGURATION GLOBALE
// ============================================================================

type RelativeDirection string

const (
	RelForward  RelativeDirection = "forward"
	RelBackward RelativeDirection = "backward"
	RelLeft     RelativeDirection = "left"
	RelRight    RelativeDirection = "right"
	RelDiagFL   RelativeDirection = "diag_fl"
	RelDiagFR   RelativeDirection = "diag_fr"
	RelDiagBL   RelativeDirection = "diag_bl"
	RelDiagBR   RelativeDirection = "diag_br"
)

type Orientation = component.Orientation
type Direction = entity.Direction

func GetRelativeDirection(o *Orientation, rel RelativeDirection) entity.Position {
	vec := o.ToVector()
	switch rel {
	case RelForward:
		return entity.Position{X: vec.X, Y: vec.Y}
	case RelBackward:
		return entity.Position{X: -vec.X, Y: -vec.Y}
	case RelLeft:
		switch o.Direction {
		case entity.DirNorth:
			return entity.Position{X: -1, Y: 0}
		case entity.DirEast:
			return entity.Position{X: 0, Y: -1}
		case entity.DirSouth:
			return entity.Position{X: 1, Y: 0}
		case entity.DirWest:
			return entity.Position{X: 0, Y: 1}
		}
	case RelRight:
		switch o.Direction {
		case entity.DirNorth:
			return entity.Position{X: 1, Y: 0}
		case entity.DirEast:
			return entity.Position{X: 0, Y: 1}
		case entity.DirSouth:
			return entity.Position{X: -1, Y: 0}
		case entity.DirWest:
			return entity.Position{X: 0, Y: -1}
		}
	case RelDiagFL:
		l := GetRelativeDirection(o, RelLeft)
		return entity.Position{X: vec.X + l.X, Y: vec.Y + l.Y}
	case RelDiagFR:
		r := GetRelativeDirection(o, RelRight)
		return entity.Position{X: vec.X + r.X, Y: vec.Y + r.Y}
	case RelDiagBL:
		b := GetRelativeDirection(o, RelBackward)
		l := GetRelativeDirection(o, RelLeft)
		return entity.Position{X: b.X + l.X, Y: b.Y + l.Y}
	case RelDiagBR:
		b := GetRelativeDirection(o, RelBackward)
		r := GetRelativeDirection(o, RelRight)
		return entity.Position{X: b.X + r.X, Y: b.Y + r.Y}
	}
	return entity.Position{X: 0, Y: 0}
}

type CollisionType string

const (
	CollideStop   CollisionType = "stop"
	CollideBounce CollisionType = "bounce"
	CollideSlide  CollisionType = "slide"
	CollidePhase  CollisionType = "phase"
)

type CollisionHandler struct {
	Type            CollisionType
	CanPhaseThrough []string
	Exceptions      func(world WorldQuery, creature *Creature, targetPos entity.Position) bool // Règles dynamiques d'exception
}

func (ch *CollisionHandler) HandleCollision(world WorldQuery, creature *Creature, attemptedPos entity.Position) (entity.Position, bool) {
	// 1. Vérifie les exceptions dynamiques en premier
	if ch.Exceptions != nil && ch.Exceptions(world, creature, attemptedPos) {
		return attemptedPos, true
	}

	// 2. Si la case est marchable, on y va directement
	if world.IsWalkable(creature, attemptedPos) {
		return attemptedPos, true
	}

	// 3. Sinon, on applique le comportement de collision
	switch ch.Type {
	case CollideStop:
		return creature.GetPosition(), false
	case CollideBounce:
		if orient, ok := creature.GetComponent("orientation").(*Orientation); ok {
			orient.Rotate(180)
		}
		return creature.GetPosition(), false
	case CollideSlide:
		return ch.trySlide(world, creature, attemptedPos)
	case CollidePhase:
		if ch.canPhase(world, attemptedPos) {
			return attemptedPos, true
		}
		return creature.GetPosition(), false
	}
	return creature.GetPosition(), false
}

func (ch *CollisionHandler) trySlide(world WorldQuery, creature *Creature, attemptedPos entity.Position) (entity.Position, bool) {
	current := creature.GetPosition()
	dx, dy := attemptedPos.X-current.X, attemptedPos.Y-current.Y
	if dy != 0 {
		if slidePos := (entity.Position{X: current.X, Y: attemptedPos.Y}); world.IsWalkable(creature, slidePos) {
			return slidePos, true
		}
	}
	if dx != 0 {
		if slidePos := (entity.Position{X: attemptedPos.X, Y: current.Y}); world.IsWalkable(creature, slidePos) {
			return slidePos, true
		}
	}
	lateral := []entity.Position{
		{X: current.X + 1, Y: current.Y}, {X: current.X - 1, Y: current.Y},
		{X: current.X, Y: current.Y + 1}, {X: current.X, Y: current.Y - 1},
	}
	for _, pos := range lateral {
		if world.IsWalkable(creature, pos) {
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

// Profil de mouvement global mis à jour
type MovementProfile struct {
	Trigger     MovementTrigger
	Navigation  NavigationLogic
	Mode        MovementMode
	Perception  PerceptionProfile // Intégration des règles de perception
	Frequency   MovementFrequency
	Orientation Orientation
	Collision   CollisionHandler
}

type MovementRequest struct {
	Creature   *Creature
	From, To   entity.Position
	Direction  RelativeDirection
	IsRotation bool
}

type MovementResult struct {
	Success       bool
	FinalPosition entity.Position
	Rotated       bool
	NewDirection  Direction
	SwappedWith   string
}

type WorldQuery interface {
	WorldState
	IsTileRevealed(pos entity.Position) bool
	WasTileRecentlyRevealed(pos entity.Position) bool
	FindNearestTarget(from entity.Position, targetType TargetType) *entity.Position
	GetTileType(pos entity.Position) string
	GetEntitiesAt(pos entity.Position) []entity.Entity
	IsWalkable(c *Creature, pos entity.Position) bool
}

type ExtendedWorldState interface {
	WorldState
	MoveEntity(creature *Creature, newPos entity.Position) bool
	MoveEntitySilent(creature *Creature, newPos entity.Position) bool
	CanMoveTo(pos entity.Position) bool
	SwapEntities(pos1, pos2 entity.Position) bool
}

// Profiles préconfigurés mis à jour avec la sémantique de perception
func DefaultMovementProfile() *MovementProfile {
	return &MovementProfile{
		Trigger:    MovementTrigger{Type: TriggerAuto},
		Navigation: NavigationLogic{Type: NavWander},
		Mode:       MovementMode{Type: ModeNormal},
		Perception: PerceptionProfile{Stealth: StealthManifest, Acoustic: AcousticSilent},
		Frequency:  MovementFrequency{Type: FreqDelay, Delay: 1},
		Collision:  CollisionHandler{Type: CollideStop},
	}
}

func FleeingProfile() *MovementProfile {
	return &MovementProfile{
		Trigger:    MovementTrigger{Type: TriggerProximity, Radius: 3},
		Navigation: NavigationLogic{Type: NavRepulsion, Target: TargetPlayer},
		Mode:       MovementMode{Type: ModeNormal},
		Perception: PerceptionProfile{
			Stealth:       StealthManifest,
			Acoustic:      AcousticEcho, // Mais fait du bruit en fuyant !
			LeavesTracks:  true,         // Laisse des indices
			TrackType:     "broken_grass",
			TrackDuration: 3,
		},
		Frequency: MovementFrequency{Type: FreqVelocity, Velocity: 1},
		Collision: CollisionHandler{Type: CollideSlide},
	}
}

func SpecterProfile() *MovementProfile {
	return &MovementProfile{
		Trigger:    MovementTrigger{Type: TriggerOnEcho}, // Se déplace quand une tuile spécifique est révélée
		Navigation: NavigationLogic{Type: NavWander},
		Mode:       MovementMode{Type: ModeUnder}, // premier rendu en dessous des tuiles, mais la logique de déplacement reste standard (peut être bloqué par des murs, etc.)
		Perception: PerceptionProfile{
			Stealth:          StealthCloaked,
			Acoustic:         AcousticSilent,
			TelegraphsIntent: true, // Indique sa prochaine destination
		},
		Frequency: MovementFrequency{Type: FreqDelay, Delay: 1},
		Collision: CollisionHandler{Type: CollidePhase, CanPhaseThrough: []string{"wall", "structure"}},
	}
}

// PassiveProfile configure une créature immobile
func PassiveProfile() *MovementProfile {
	return &MovementProfile{
		Trigger:    MovementTrigger{Type: TriggerPassive},
		Navigation: NavigationLogic{Type: NavWander},
		Mode:       MovementMode{Type: ModeNormal},
		Perception: PerceptionProfile{Stealth: StealthManifest, Acoustic: AcousticSilent},
		Frequency:  MovementFrequency{Type: FreqInstant},
		Collision:  CollisionHandler{Type: CollideStop},
	}
}

// PatrollerProfile configure une créature qui suit une route spécifique
func PatrollerProfile(route []entity.Position) *MovementProfile {
	return &MovementProfile{
		Trigger:    MovementTrigger{Type: TriggerAuto},
		Navigation: NavigationLogic{Type: NavPatrol, PatrolRoute: route},
		Mode:       MovementMode{Type: ModeNormal},
		Perception: PerceptionProfile{Stealth: StealthManifest, Acoustic: AcousticSilent},
		Frequency:  MovementFrequency{Type: FreqDelay, Delay: 1},
		Collision:  CollisionHandler{Type: CollideStop},
	}
}

// RelativePatrollerProfile configure une créature qui suit un pattern de déplacements relatifs constants.
func RelativePatrollerProfile(pattern []entity.Position) *MovementProfile {
	return &MovementProfile{
		Trigger: MovementTrigger{Type: TriggerAuto},
		Navigation: NavigationLogic{
			Type:        NavRelative, // Notre nouveau type de navigation
			PatrolRoute: pattern,
			PatrolIndex: 0,
		},
		Mode:       MovementMode{Type: ModeNormal}, // Mode normal par défaut (le burrower le surchargera en ModeUnder)
		Perception: PerceptionProfile{Stealth: StealthManifest, Acoustic: AcousticSilent},
		Frequency:  MovementFrequency{Type: FreqDelay, Delay: 1},
		Collision:  CollisionHandler{Type: CollideStop},
	}
}
