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
)

// Creature est une entité vivante avec comportement
type Creature struct {
	entity.BaseEntity
	Species         string
	Behavior        component.Behavior
	Mobility        component.Mobility
	Visual          component.Visual
	MovementProfile *MovementProfile // configuration complète du mouvement
}

func New(species string, pos entity.Position) *Creature {
	c := &Creature{
		BaseEntity: entity.NewBaseEntity(entity.TypeCreature),
		Species:    species,
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
	if c.MovementProfile != nil {
		return c.MovementProfile.Orientation.Direction
	}
	return c.BaseEntity.GetOrientation()
}

func (c *Creature) SetOrientation(o entity.Direction) {
	if c.MovementProfile != nil {
		c.MovementProfile.Orientation.Direction = o
	}
	c.BaseEntity.SetOrientation(o)
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
		totalPlots := world.GetGridTotalPlots()

		if world.GetTileState(currentPos) == "alone" {
			return Action{Type: "spawn_trap", Metadata: map[string]interface{}{"trap_type": "moss_lure"}}
		}

		if !world.HasActivityNearby(currentPos, 4) {
			return Action{Type: "idle"}
		}

		if len(emptyPlots) == 0 && world.IsGridSaturatedWithTraps() {
			return Action{Type: "flee"}
		}

		if totalPlots > 0 {
			c.Behavior.Aggression = (len(emptyPlots) * 200) / totalPlots
			if c.Behavior.Aggression > 100 {
				c.Behavior.Aggression = 100
			}
		}

		if len(emptyPlots) > 0 {
			var nearest entity.Position
			minDist := 9999
			for _, p := range emptyPlots {
				d := abs(p.X-currentPos.X) + abs(p.Y-currentPos.Y)
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
			if abs(dx) > abs(dy) {
				move.X = sign(dx)
			} else {
				move.Y = sign(dy)
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
				dist := abs(newPos.X-playerPos.X) + abs(newPos.Y-playerPos.Y)
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
		if abs(dx) > abs(dy) {
			move.X = sign(dx)
		} else {
			move.Y = sign(dy)
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
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
		})
		c.SetMovementProfile(&MovementProfile{
			Trigger:    MovementTrigger{Type: TriggerAuto},
			Navigation: NavigationLogic{Type: NavWander, WanderBias: entity.Position{X: 0, Y: -1}},
			Mode:       MovementMode{Type: ModeOver}, // Couche supérieure (Z-sorting haut)
			Perception: PerceptionProfile{
				Stealth:  StealthManifest, // Glissement visible à l'écran
				Acoustic: AcousticSilent,  // Vol silencieux
			},
			Frequency:   MovementFrequency{Type: FreqDelay, Delay: 1},
			Orientation: Orientation{Direction: DirNorth},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
		c.AddTag("flying")
		c.AddTag("passive")

	case "shadowstalker":
		c.SetBehavior(component.Behavior{
			State:       "hunting",
			Aggression:  80,
			Territorial: true,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
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
			Frequency:   MovementFrequency{Type: FreqVelocity, Velocity: 2},
			Orientation: Orientation{Direction: DirNorth},
			Collision:   CollisionHandler{Type: CollideBounce},
		})
		c.AddTag("dangerous")
		c.AddTag("aggressive")

	case "burrower":
		c.SetBehavior(component.Behavior{
			State:       "hiding",
			Aggression:  20,
			Territorial: false,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
		})

		// Pour dessiner et contourner un espace de 3x3 cases,
		// il faut faire 2 pas (translations) par côté.
		squarePattern3x3 := []entity.Position{
			{X: 1, Y: 0},  // 1 pas à Droite
			{X: 0, Y: 1},  // 1 pas en Bas
			{X: -1, Y: 0}, // 1 pas à Gauche
			{X: 0, Y: -1}, // 1 pas en Haut
		}

		c.SetMovementProfile(&MovementProfile{
			Trigger: MovementTrigger{Type: TriggerAuto},
			Navigation: NavigationLogic{
				Type:        NavRelative,
				PatrolRoute: squarePattern3x3,
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
			Frequency:   MovementFrequency{Type: FreqDelay, Delay: 1},
			Orientation: Orientation{Direction: DirNorth},
			Collision:   CollisionHandler{Type: CollidePhase, CanPhaseThrough: []string{"dirt", "soil"}},
		})
		c.AddTag("elusive")

	case "specter":
		c.SetBehavior(component.Behavior{
			State:      "haunting",
			Aggression: 60,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
		})
		// Utilise le profil global réécrit dans movement.go
		c.SetMovementProfile(SpecterProfile())
		c.AddTag("ethereal")
		c.AddTag("dangerous")

	case "stonewarden":
		c.SetBehavior(component.Behavior{
			State:       "guarding",
			Aggression:  40,
			Territorial: true,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
		})
		// Se déplace une fois quand révélée, puis commence une patrouille basée sur son orientation
		c.SetMovementProfile(&MovementProfile{
			Trigger:    MovementTrigger{Type: TriggerOnReveal},
			Navigation: NavigationLogic{Type: NavOrientation},
			Mode:       MovementMode{Type: ModeNormal},
			Perception: PerceptionProfile{Stealth: StealthManifest, Acoustic: AcousticSilent},
			Frequency:  MovementFrequency{Type: FreqDelay, Delay: 1},
			Collision:  CollisionHandler{Type: CollideStop},
		})
		c.AddTag("static")

	case "echo_hound":
		c.SetBehavior(component.Behavior{
			State:      "echoing",
			Aggression: 50,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   1,
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
			Frequency:   MovementFrequency{Type: FreqVelocity, Velocity: 1},
			Orientation: Orientation{Direction: DirNorth},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
		c.AddTag("fast")

	case "fleeing_sprite":
		c.SetBehavior(component.Behavior{
			State:      "fleeing",
			Aggression: 0,
		})
		c.SetMobility(component.Mobility{
			CanMove: true,
			Speed:   2,
		})
		c.SetMovementProfile(FleeingProfile())
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
			Frequency:   MovementFrequency{Type: FreqDelay, Delay: 1},
			Orientation: Orientation{Direction: DirSouth},
			Collision:   CollisionHandler{Type: CollideSlide},
		})
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
		State:       "patrolling",
		Aggression:  30,
		Territorial: true,
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
