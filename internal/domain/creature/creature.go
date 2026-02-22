package creature

import (
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Creature est une entité vivante avec comportement
type Creature struct {
	entity.BaseEntity
	Species  string
	Behavior component.Behavior
	Mobility component.Mobility
	Visual   component.Visual
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

// Action représente une intention de la créature
type Action struct {
	Type      string          // "move", "attack", "transform", "flee", "hide"
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
}

// SimpleAI implémentation basique
type SimpleAI struct{}

func (ai *SimpleAI) Decide(c *Creature, world WorldState) Action {
	if !c.Mobility.CanMove {
		return Action{Type: "idle"}
	}

	// Logique simple basée sur l'état
	switch c.Behavior.State {
	case "fleeing":
		// S'éloigne du joueur
		playerPos := world.GetPlayerPosition()
		creaturePos := c.GetPosition()

		var bestMove entity.Position
		maxDist := -1

		// Teste les 4 directions
		directions := []entity.Position{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
		for _, dir := range directions {
			newPos := entity.Position{
				X: creaturePos.X + dir.X,
				Y: creaturePos.Y + dir.Y,
			}
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
		// Approche le joueur
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

		newPos := entity.Position{
			X: creaturePos.X + move.X,
			Y: creaturePos.Y + move.Y,
		}

		if world.IsValidMove(newPos) {
			return Action{Type: "move", Direction: move}
		}
		return Action{Type: "idle"}

	case "pollinating":
		// Cherche les ressources à transformer
		resources := world.GetResources(c.GetPosition(), 2)
		if len(resources) > 0 {
			return Action{
				Type:     "transform",
				TargetID: resources[0],
				Metadata: map[string]interface{}{"effect": "pollinate"},
			}
		}
		// Mouvement aléatoire
		return randomMove(world, c.GetPosition())

	default: // idle
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

// Factory pour créer des créatures préconfigurées
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
			Aggression:     0,
			Territorial:    false,
			Transformation: "pollinize",
			LeavesTraces:   true,
		})
		c.SetMobility(component.Mobility{
			CanMove:     true,
			MovePattern: "random",
			Speed:       1,
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
			CanMove:     true,
			MovePattern: "hunter",
			Speed:       2,
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
			CanMove:     true,
			MovePattern: "burrow",
			Speed:       1,
		})
		c.AddTag("elusive")

	default:
		return nil, fmt.Errorf("espèce inconnue: %s", species)
	}

	return c, nil
}

func (f *Factory) GetAI() AI {
	return f.ai
}

func (f *Factory) SetAI(ai AI) {
	f.ai = ai
}
