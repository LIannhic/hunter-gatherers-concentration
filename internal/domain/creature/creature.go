package creature

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

const (
	DirNorth = entity.DirNorth
	DirEast  = entity.DirEast
	DirSouth = entity.DirSouth
	DirWest  = entity.DirWest

	Forward       = entity.DirNorth
	Backward      = entity.DirSouth
	Right         = entity.DirEast
	Left          = entity.DirWest
	ForwardRight  = entity.DirNorthEast
	ForwardLeft   = entity.DirNorthWest
	BackwardRight = entity.DirSouthEast
	BackwardLeft  = entity.DirSouthWest
)

// Creature est une entité vivante avec comportement
type Creature struct {
	entity.BaseEntity
	Species         string
	Behavior        component.Behavior
	Mobility        component.Mobility
	Visual          component.Visual
	MovementProfile *MovementProfile // configuration complète du mouvement
	ThreatZone      []entity.Direction
}

func New(species string, pos entity.Position) *Creature {
	c := &Creature{
		BaseEntity: entity.NewBaseEntity(entity.TypeCreature),
		Species:    species,
		Behavior: component.Behavior{
			AggressionFactors: make(map[string]int),
		},
		ThreatZone: []entity.Direction{Forward}, // Par défaut face à elle
	}
	c.SetPosition(pos)
	c.AddTag("creature")
	c.AddTag(species)
	return c
}

func (c *Creature) SetBehavior(b component.Behavior) {
	c.Behavior = b
}

func (c *Creature) SetMobility(m component.Mobility) {
	c.Mobility = m
}

func (c *Creature) SetMovementProfile(m *MovementProfile) {
	c.MovementProfile = m
}

func (c *Creature) GetOrientation() entity.Direction {
	baseDirection := c.BaseEntity.GetOrientation()
	if c.MovementProfile != nil {
		baseDirection = entity.TransformDirection(c.MovementProfile.Orientation.Direction, c.BaseEntity.GetTransformation())
	}
	return baseDirection
}

func (c *Creature) SetOrientation(o entity.Direction) {
	if o == entity.DirNorth || o == entity.DirEast || o == entity.DirSouth || o == entity.DirWest {
		c.BaseEntity.SetOrientation(o)
		if c.MovementProfile != nil {
			c.MovementProfile.Orientation.Direction = entity.DirNorth
		}
	} else {
		if c.MovementProfile != nil {
			c.MovementProfile.Orientation.Direction = o
		}
		c.BaseEntity.SetTransformation(entity.TransIdentity)
	}
}

func (c *Creature) GetComponent(name string) interface{} {
	switch name {
	case "orientation":
		if c.MovementProfile != nil {
			return &c.MovementProfile.Orientation
		}
	case "behavior":
		return &c.Behavior
	case "mobility":
		return &c.Mobility
	}
	return nil
}

// Association compliance
func (c *Creature) GetMatchID() string      { return c.Species }
func (c *Creature) GetLogicKey() string     { return "" }
func (c *Creature) GetElement() string      { return "" }
func (c *Creature) GetNarrativeTag() string { return "" }
func (c *Creature) GetMatchTypes() []string { return []string{"identical"} }

// Action représente une intention de la créature
type Action struct {
	Type      string          // "move", "attack", "transform", "flee", "idle"
	Direction entity.Position // Pour move
	TargetID  string
	Metadata  map[string]interface{}
}

// AI définit le comportement
type AI interface {
	Decide(c *Creature, world WorldState) Action
}

// WorldState interface pour que l'IA puisse observer le monde
type WorldState interface {
	GetPlayerPosition() entity.Position
	GetNearbyCreatures(pos entity.Position, radius int) []*Creature
	GetResources(pos entity.Position, radius int) []string
	IsValidMove(pos entity.Position) bool
	GetTileState(pos entity.Position) string
	GetEmptyPlots() []entity.Position                       // Retourne les positions des cases strictement vides
	GetGridTotalPlots() int                                 // Nombre total de cases dans la grille
	IsGridSaturatedWithTraps() bool                         // Vérifie si plus aucune ressource n'est présente
	HasActivityNearby(pos entity.Position, radius int) bool // Détecte mouvement/révélation
}

// SimpleAI implémentation basique
type SimpleAI struct{}

func (ai *SimpleAI) Decide(c *Creature, world WorldState) Action {
	if !c.Mobility.CanMove {
		return Action{Type: "idle"}
	}

	switch c.Behavior.State {
	case "spreading_moss":
		currentPos := c.GetPosition()
		emptyPlots := world.GetEmptyPlots()

		if world.GetTileState(currentPos) == "alone" {
			return Action{Type: "spawn_trap", Metadata: map[string]interface{}{"trap_type": "moss_lure"}}
		}

		if !world.HasActivityNearby(currentPos, 4) {
			return Action{Type: "idle"}
		}

		if world.IsGridSaturatedWithTraps() {
			return Action{Type: "flee"}
		}

		if len(emptyPlots) > 0 {
			var nearest entity.Position
			minDist := 9999
			for _, p := range emptyPlots {
				d := entity.Abs(p.X-currentPos.X) + entity.Abs(p.Y-currentPos.Y)
				if d < minDist {
					minDist = d
					nearest = p
				} else if d == minDist {
					if p.Y < nearest.Y || (p.Y == nearest.Y && p.X < nearest.X) {
						nearest = p
					}
				}
			}

			dx, dy := nearest.X-currentPos.X, nearest.Y-currentPos.Y
			var move entity.Position
			if entity.Abs(dx) > entity.Abs(dy) {
				move.X = entity.Sign(dx)
			} else {
				move.Y = entity.Sign(dy)
			}

			if world.IsValidMove(entity.Position{X: currentPos.X + move.X, Y: currentPos.Y + move.Y}) {
				return Action{Type: "move", Direction: move}
			}
		}
		return Action{Type: "idle"}

	case "fleeing":
		playerPos := world.GetPlayerPosition()
		creaturePos := c.GetPosition()

		var bestMove entity.Position
		maxDist := -1

		directions := []entity.Position{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, dir := range directions {
			newPos := entity.Position{X: creaturePos.X + dir.X, Y: creaturePos.Y + dir.Y}
			if world.IsValidMove(newPos) {
				dist := entity.Abs(newPos.X-playerPos.X) + entity.Abs(newPos.Y-playerPos.Y)
				if dist > maxDist {
					maxDist = dist
					bestMove = dir
				}
			}
		}
		return Action{Type: "move", Direction: bestMove}

	case "hunting":
		playerPos := world.GetPlayerPosition()
		creaturePos := c.GetPosition()

		dx := playerPos.X - creaturePos.X
		dy := playerPos.Y - creaturePos.Y

		var move entity.Position
		if entity.Abs(dx) > entity.Abs(dy) {
			move.X = entity.Sign(dx)
		} else {
			move.Y = entity.Sign(dy)
		}

		newPos := entity.Position{X: creaturePos.X + move.X, Y: creaturePos.Y + move.Y}
		if world.IsValidMove(newPos) {
			return Action{Type: "move", Direction: move}
		}
		return Action{Type: "idle"}

	case "pollinating":
		resources := world.GetResources(c.GetPosition(), 2)
		if len(resources) > 0 {
			return Action{
				Type:     "transform",
				TargetID: resources[0],
				Metadata: map[string]interface{}{"effect": "pollinate"},
			}
		}
		return randomMove(world, c.GetPosition())

	default:
		return randomMove(world, c.GetPosition())
	}
}

func randomMove(world WorldState, pos entity.Position) Action {
	directions := []entity.Position{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	dir := directions[rand.Intn(len(directions))]
	newPos := entity.Position{X: pos.X + dir.X, Y: pos.Y + dir.Y}

	if world.IsValidMove(newPos) {
		return Action{Type: "move", Direction: dir}
	}
	return Action{Type: "idle"}
}

// ============================================================================
// FACTORY REFACTORISÉE (Intégration fine des règles de perception)
// ============================================================================

type Factory struct {
	ai AI
}

func NewFactory() *Factory {
	return &Factory{ai: &SimpleAI{}}
}

func (f *Factory) Create(species string, pos entity.Position) (*Creature, error) {
	c := New(species, pos)

	switch species {
	case "lumifly":
		c.SetBehavior(component.Behavior{
			State:          "pollinating",
			Transformation: "pollinize",
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeSmall,
			Weight:  component.WeightLight,
		})
		c.SetMovementProfile(&MovementProfile{
			Trigger: MovementTrigger{Type: TriggerAuto},
			Navigation: NavigationLogic{
				Type:           NavAttraction,
				Target:         TargetResource,
				TargetName:     "dreamberry",
				ExcludedStages: []int{0, 3}, // Ignore le stade 1 (index 0) et le stade 4 (index 3)
				WanderBias:     entity.Position{X: 0, Y: -1},
			},
			Mode: MovementMode{Type: ModeOver}, // Couche supérieure (Z-sorting haut)
			Perception: PerceptionProfile{
				Stealth:  StealthManifest, // Glissement visible à l'écran
				Acoustic: AcousticSilent,  // Vol silencieux
			},
			Frequency:   *NewMovementFrequency(FreqDelay, 0, 1),
			Orientation: Orientation{Direction: Forward},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
		c.SetThreatZone(ThreatCone)
		c.AddTag("flying")
		c.AddTag("passive")

	case "flutterwing":
		c.SetBehavior(component.Behavior{
			State:          "dancing",
			AggressionBase: 0,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeSmall,
			Weight:  component.WeightLight,
		})
		c.SetMovementProfile(&MovementProfile{
			Trigger: MovementTrigger{Type: TriggerProximity, Radius: 2},
			Navigation: NavigationLogic{
				Type:   NavRepulsion,
				Target: TargetPlayer,
			},
			Mode: MovementMode{Type: ModeOver},
			Perception: PerceptionProfile{
				Stealth:  StealthManifest,
				Acoustic: AcousticSilent,
			},
			Frequency:   *NewMovementFrequency(FreqDelay, 0, 1),
			Orientation: Orientation{Direction: Forward},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
		c.SetThreatZone(ThreatFrontal) // Très peu menaçant
		c.AddTag("flying")
		c.AddTag("passive")
		c.AddTag("elusive")

	case "shadowstalker":
		c.SetBehavior(component.Behavior{
			State:          "hunting",
			AggressionBase: 80,
			Territorial:    true,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeMedium,
			Weight:  component.WeightMedium,
		})
		c.SetMovementProfile(&MovementProfile{
			Trigger:    MovementTrigger{Type: TriggerProximity, Radius: 4},
			Navigation: NavigationLogic{Type: NavAttraction, Target: TargetPlayer},
			Mode:       MovementMode{Type: ModeSwap}, // échange de place avec une tuile adjacente pour surprendre le joueur
			Perception: PerceptionProfile{
				Stealth:          StealthCloaked, // Totalement invisible à l'œil nu lors du mouvement
				Acoustic:         AcousticSilent, // Pas de bruit
				TelegraphsIntent: true,           // ajoute des traces de griffures sur les tuiles autour de lui
			},
			Frequency:   *NewMovementFrequency(FreqVelocity, 1, 0),
			Orientation: Orientation{Direction: Forward},
			Collision:   CollisionHandler{Type: CollideBounce},
		})
		c.SetThreatZone(ThreatCone)
		c.AddTag("dangerous")
		c.AddTag("aggressive")

	case "burrower":
		c.SetBehavior(component.Behavior{
			State:          "hiding",
			AggressionBase: 20,
			Territorial:    false,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeSmall,
			Weight:  component.WeightLight,
		})

		// Le burrower se déplace par rapport à lui-même : Droite -> Avant -> Gauche -> Arrière
		relativePattern := []entity.Position{
			{X: 1, Y: 0},  // Droite relative
			{X: 0, Y: -1}, // Avant relatif
			{X: -1, Y: 0}, // Gauche relative
			{X: 0, Y: 1},  // Arrière relatif
		}

		c.SetMovementProfile(&MovementProfile{
			Trigger: MovementTrigger{Type: TriggerAuto},
			Navigation: NavigationLogic{
				Type:        NavRelative,
				PatrolRoute: relativePattern,
				PatrolIndex: 0,
			},
			Mode: MovementMode{Type: ModeUnder},
			Perception: PerceptionProfile{
				Stealth:       StealthManifest,
				Acoustic:      AcousticSilent,
				LeavesTracks:  true,
				TrackType:     "mud",
				TrackDuration: 2,
			},
			Frequency:   *NewMovementFrequency(FreqDelay, 0, 1),
			Orientation: Orientation{Direction: Forward},
			Collision:   CollisionHandler{Type: CollidePhase, CanPhaseThrough: []string{"dirt", "soil"}},
		})
		c.SetThreatZone(ThreatCone)
		c.AddTag("elusive")

	case "specter":
		c.SetBehavior(component.Behavior{
			State:          "haunting",
			AggressionBase: 60,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeMedium,
			Weight:  component.WeightLight,
		})
		// Utilise le profil global réécrit dans movement.go
		c.SetMovementProfile(SpecterProfile())
		c.SetThreatZone(ThreatCone)
		c.AddTag("ethereal")
		c.AddTag("dangerous")

	case "stonewarden":
		c.SetBehavior(component.Behavior{
			State:          "guarding",
			AggressionBase: 40,
			Territorial:    true,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeLarge,
			Weight:  component.WeightHeavy,
		})
		// Se déplace une fois quand révélée, puis commence une patrouille basée sur son orientation
		c.SetMovementProfile(&MovementProfile{
			Trigger:    MovementTrigger{Type: TriggerOnReveal},
			Navigation: NavigationLogic{Type: NavOrientation},
			Mode:       MovementMode{Type: ModeNormal},
			Perception: PerceptionProfile{Stealth: StealthManifest, Acoustic: AcousticSilent},
Frequency:   *NewMovementFrequency(FreqDelay, 0, 1),
			Orientation: Orientation{Direction: entity.DirNorthEast},
			Collision:  CollisionHandler{Type: CollideStop},
		})
		c.SetThreatZone(ThreatCone)
		c.AddTag("static")

	case "echo_hound":
		c.SetBehavior(component.Behavior{
			State:          "echoing",
			AggressionBase: 50,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeMedium,
			Weight:  component.WeightMedium,
		})
		c.SetMovementProfile(&MovementProfile{
			Trigger: MovementTrigger{Type: TriggerOnEcho},
			// Quand une tile est révélée, se dirige vers la ressource la plus proche (dreamberries)
			Navigation: NavigationLogic{Type: NavAttraction, Target: TargetResource, TargetName: "dreamberry"},
			Mode:       MovementMode{Type: ModeNormal},
			Perception: PerceptionProfile{
				Stealth:       StealthManifest, // On le voit courir à toute vitesse
				Acoustic:      AcousticEcho,    // Il fait énormément de bruit (lourd)
				LeavesTracks:  true,            // Laisse de grosses traces de griffes
				TrackType:     "claws",
				TrackDuration: 2,
			},
			Frequency:   *NewMovementFrequency(FreqVelocity, 1, 0),
			Orientation: Orientation{Direction: Forward},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
		c.SetThreatZone(ThreatCone)
		c.AddTag("fast")

	case "fleeing_sprite":
		c.SetBehavior(component.Behavior{
			State:          "fleeing",
			AggressionBase: 0,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   2,
		})
		c.SetMovementProfile(FleeingProfile())
		c.SetThreatZone(ThreatCone)
		c.AddTag("passive")
		c.AddTag("elusive")

	case "moss_monkey":
		c.SetBehavior(component.Behavior{
			State:       "spreading_moss",
			Territorial: true,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
			Size:    component.SizeMedium,
			Weight:  component.WeightMedium,
		})
		c.SetMovementProfile(&MovementProfile{
			Trigger:    MovementTrigger{Type: TriggerProximity, Radius: 4},
			Navigation: NavigationLogic{Type: NavAttraction, Target: TargetEmpty},
			Mode:       MovementMode{Type: ModeNormal},
			Perception: PerceptionProfile{
				Stealth:      StealthManifest, // Glissement normal
				Acoustic:     AcousticSilent,
				LeavesTracks: false, // Le singe mousse ne laisse pas de traces, mais des pièges
			},
			Frequency: *NewMovementFrequency(FreqDelay, 0, 1),
			// NOTE: Le comportement du singe-mousse est validé. Son orientation Backward
			// est intentionnelle pour son mode de déplacement "à reculons" ou sa zone de menace.
			Orientation: Orientation{Direction: Backward},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
		c.SetThreatZone(ThreatCone)
		c.AddTag("territorial")
		c.AddTag("dangerous_on_reveal")
		c.AddTag("climb")

	default:
		return nil, fmt.Errorf("espèce inconnue: %s", species)
	}

	return c, nil
}

// CreatePatroller crée une créature avec un itinéraire de patrouille
func (f *Factory) CreatePatroller(species string, pos entity.Position, route []entity.Position) (*Creature, error) {
	c, err := f.Create(species, pos)
	if err != nil {
		return nil, err
	}

	c.SetMovementProfile(PatrollerProfile(route))
	c.SetBehavior(component.Behavior{
		State:          "patrolling",
		AggressionBase: 30,
		Territorial:    true,
	})
	c.SetMobility(component.Mobility{
		CanMove: true,
		Speed:   1,
	})
	c.AddTag("patroller")

	return c, nil
}

func (f *Factory) GetAI() AI   { return f.ai }
func (f *Factory) SetAI(ai AI) { f.ai = ai }

// GetActiveThreatDirections retourne les directions menacées réelles après transformation D4
func (c *Creature) GetActiveThreatDirections() []entity.Direction {
	transform := c.GetTransformation()
	activeThreats := make([]entity.Direction, len(c.ThreatZone))

	for i, localDir := range c.ThreatZone {
		activeThreats[i] = entity.TransformDirection(localDir, transform)
	}

	return activeThreats
}

// GetLungeDirectionVector retourne le vecteur unitaire du décalage visuel d'attaque.
// Dans le cas d'un cône, on prend la direction centrale (la première du slice par convention interne).
func (c *Creature) GetLungeDirectionVector() (dx, dy float64) {
	if len(c.ThreatZone) == 0 {
		return 0, 0
	}

	// Pour ThreatCone, ThreatZone[0] est DirNorth (le centre).
	// On transforme cette direction locale par la transformation actuelle de l'entité.
	targetDir := entity.TransformDirection(c.ThreatZone[0], c.GetTransformation())
	v := targetDir.ToVector()
	return float64(v.X), float64(v.Y)
}

func (c *Creature) SetThreatZone(zone []entity.Direction) {
	c.ThreatZone = zone
}

// IsPositionThreatened vérifie si une position cible est menacée par la créature.
func (c *Creature) IsPositionThreatened(target entity.Position) bool {
	currentPos := c.GetPosition()

	// 1. Calcule la direction relative (vecteur unitaire)
	dx := target.X - currentPos.X
	dy := target.Y - currentPos.Y

	var relDir entity.Direction
	found := false

	// Recherche de la direction correspondante parmi les 8 possibles
	if dx == 0 && dy < 0 {
		relDir = entity.DirNorth
		found = true
	} else if dx > 0 && dy < 0 {
		relDir = entity.DirNorthEast
		found = true
	} else if dx > 0 && dy == 0 {
		relDir = entity.DirEast
		found = true
	} else if dx > 0 && dy > 0 {
		relDir = entity.DirSouthEast
		found = true
	} else if dx == 0 && dy > 0 {
		relDir = entity.DirSouth
		found = true
	} else if dx < 0 && dy > 0 {
		relDir = entity.DirSouthWest
		found = true
	} else if dx < 0 && dy == 0 {
		relDir = entity.DirWest
		found = true
	} else if dx < 0 && dy < 0 {
		relDir = entity.DirNorthWest
		found = true
	}

	if !found {
		return false
	}

	// 2. Vérifie si cette direction est dans les zones de menace actives
	activeThreats := c.GetActiveThreatDirections()
	for _, threat := range activeThreats {
		if threat == relDir {
			return true
		}
	}

	return false
}

// Gabarits de zones de menace (Relatifs à la créature)
var (
	// Menace uniquement la case devant elle
	ThreatFrontal = []entity.Direction{Forward}

	// Cône à 3x3 devant (Avant, Avant-Droite, Avant-Gauche)
	ThreatCone = []entity.Direction{Forward, ForwardRight, ForwardLeft}

	// Menace en croix (Cardinaux relatifs)
	ThreatCross = []entity.Direction{Forward, Right, Backward, Left}

	// Menace les diagonales relatives
	ThreatDiagonals = []entity.Direction{ForwardRight, BackwardRight, BackwardLeft, ForwardLeft}

	// Autour d'elle à 360° (Les 8 directions)
	ThreatFull360 = []entity.Direction{
		Forward, ForwardRight, Right, BackwardRight,
		Backward, BackwardLeft, Left, ForwardLeft,
	}
)
