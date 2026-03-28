package usecase

import (
	"errors"
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// DefaultFlipDirection est la direction par défaut si non spécifiée
var DefaultFlipDirection = domain.FlipCenter

type Command interface {
	Execute() error
	CanExecute() bool
}

type RevealTileCommand struct {
	World         *domain.World
	GridID        string
	Position      board.Position
	FlipDirection domain.FlipDirection
}

func (c *RevealTileCommand) CanExecute() bool {
	// 1. Check grid exists
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return false
	}

	// 2. Check player is on this grid
	if c.World.CurrentGridID != c.GridID {
		return false
	}

	// 2. Check tile exists and has an entity that is hidden
	tile, err := grid.Get(c.Position)
	if err != nil {
		return false
	}

	if len(tile.EntitiesID) == 0 {
		return false
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	ent, ok := c.World.Entities.Get(entity.ID(topID))
	if !ok || ent.GetState() != entity.Hidden {
		return false
	}

	// 3. Check only 2 tiles can be flipped per turn (across all grids)
	if !c.World.CanFlipTile() {
		return false
	}

	return true
}

func (c *RevealTileCommand) Execute() error {
	if c.World.CurrentGridID != c.GridID {
		fmt.Println("Player is not on this grid")
		fmt.Printf("Player on %s but tried %s\n", c.World.CurrentGridID, c.GridID)
		return errors.New("player is not on this grid")
	}

	if !c.CanExecute() {
		return errors.New("cannot reveal this tile")
	}

	// Révèle l'entité via le world
	ent, err := c.World.RevealTile(c.GridID, c.Position)
	if err != nil {
		return err
	}

	// Track this flipped tile for the current turn
	c.World.AddFlippedTile(c.Position)

	// Publie l'événement avec la direction de flip
	c.World.EventBus.Publish(event.NewEntityRevealedEvent(
		entity.Position{X: c.Position.X, Y: c.Position.Y},
		string(ent.GetID()),
		c.GridID,
		c.FlipDirection,
	))

	return nil
}

type MatchResult struct {
	Success   bool
	IsMatch   bool
	Positions [2]board.Position
	Entities  [2]domain.Entity
}

type MatchTilesCommand struct {
	World      *domain.World
	AssocEng   *domain.AssocEngine
	GridID     string
	Pos1, Pos2 board.Position
	OnSuccess  func() // Callback appelé en cas de succès
	OnFailure  func() // Callback appelé en cas d'échec (pour cacher les cartes et passer le tour)
}

func (c *MatchTilesCommand) CanExecute() bool {
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return false
	}

	tile1, err1 := grid.Get(c.Pos1)
	tile2, err2 := grid.Get(c.Pos2)

	if err1 != nil || err2 != nil {
		return false
	}

	if len(tile1.EntitiesID) == 0 || len(tile2.EntitiesID) == 0 {
		return false
	}

	topID1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	topID2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
	e1, ok1 := c.World.Entities.Get(entity.ID(topID1))
	e2, ok2 := c.World.Entities.Get(entity.ID(topID2))

	if !ok1 || !ok2 {
		return false
	}

	// Vérifie que les entités sont bien révélées
	if e1.GetState() != entity.Revealed || e2.GetState() != entity.Revealed {
		return false
	}

	return true
}

func (c *MatchTilesCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("cannot match these tiles")
	}

	grid, _ := c.World.GetGrid(c.GridID)

	tile1, _ := grid.Get(c.Pos1)
	tile2, _ := grid.Get(c.Pos2)

	topID1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	topID2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]

	entity1, _ := c.World.Entities.Get(entity.ID(topID1))
	entity2, _ := c.World.Entities.Get(entity.ID(topID2))

	// Détermine les types des entités
	res1, isRes1 := entity1.(*domain.Resource)
	res2, isRes2 := entity2.(*domain.Resource)
	cre1, isCre1 := entity1.(*domain.Creature)
	cre2, isCre2 := entity2.(*domain.Creature)

	// Vérifie si c'est une association valide
	isMatch := false
	matchType := ""

	// Cas 1 : Deux ressources - utilise le système d'association
	if isRes1 && isRes2 {
		result, err := c.AssocEng.TryAssociate(res1, res2)
		if err == nil && result.Success {
			isMatch = true
			matchType = result.Type.String()
		}
	}

	// Cas 2 : Deux créatures - compare les espèces
	if !isMatch && isCre1 && isCre2 {
		if cre1.Species == cre2.Species {
			isMatch = true
			matchType = "creature_capture"
		}
	}

	if isMatch {
		// Succès : les entités sont marquées comme appairées
		c.World.MatchTile(c.GridID, c.Pos1)
		c.World.MatchTile(c.GridID, c.Pos2)

		// Note: on les retire du monde (elles seront nettoyées de la grille par RemoveEntity)
		c.World.RemoveEntity(entity1.GetID())
		c.World.RemoveEntity(entity2.GetID())

		c.World.EventBus.Publish(domain.Event{
			Type:     domain.EventType("tiles_matched"),
			SourceID: "player",
			Payload: map[string]interface{}{
				"position1":  c.Pos1,
				"position2":  c.Pos2,
				"grid_id":    c.GridID,
				"assoc_type": matchType,
			},
		})

		if c.OnSuccess != nil {
			c.OnSuccess()
		}

		return nil
	} else {
		// Échec : recacher les entités
		entity1.SetState(entity.Hidden)
		entity2.SetState(entity.Hidden)

		if c.OnFailure != nil {
			c.OnFailure()
		}

		return errors.New("association échouée")
	}
}

type EndTurnCommand struct {
	World *domain.World
}

func (c *EndTurnCommand) CanExecute() bool {
	return true
}

func (c *EndTurnCommand) Execute() error {
	c.World.EventBus.Publish(event.NewTurnEndedEvent(c.World.Turn))
	c.World.EventBus.ProcessQueue()

	return nil
}

type SpawnTestEntitiesCommand struct {
	World  *domain.World
	GridID string
}

func (c *SpawnTestEntitiesCommand) CanExecute() bool {
	return true
}

func (c *SpawnTestEntitiesCommand) Execute() error {
	positions := []board.Position{
		{X: 1, Y: 1}, {X: 2, Y: 1},
		{X: 3, Y: 2}, {X: 4, Y: 2},
		{X: 1, Y: 3}, {X: 2, Y: 3},
	}

	resourceTypes := []string{"dreamberry", "dreamberry", "moonstone", "moonstone", "whispering_herb", "whispering_herb"}

	for i, pos := range positions {
		if i < len(resourceTypes) {
			_, err := c.World.SpawnResource(c.GridID, resourceTypes[i], entity.Position{X: pos.X, Y: pos.Y})
			if err != nil {
				return err
			}
		}
	}

	c.World.SpawnCreature(c.GridID, "lumifly", entity.Position{X: 3, Y: 3})

	return nil
}

type ClearBoardCommand struct {
	World  *domain.World
	GridID string
}

func (c *ClearBoardCommand) CanExecute() bool {
	return true
}

func (c *ClearBoardCommand) Execute() error {
	if _, ok := c.World.GetGrid(c.GridID); !ok {
		return errors.New("grid not found")
	}

	// Supprime toutes les entités de ce grid
	for _, e := range c.World.Entities.GetAllActive() {
		if e.GetGridID() == c.GridID {
			c.World.RemoveEntity(e.GetID())
		}
	}

	return nil
}

type ClearAllBoardsCommand struct {
	World *domain.World
}

func (c *ClearAllBoardsCommand) CanExecute() bool {
	return true
}

func (c *ClearAllBoardsCommand) Execute() error {
	// Supprime toutes les entités
	for _, e := range c.World.Entities.GetAllActive() {
		c.World.RemoveEntity(e.GetID())
	}

	return nil
}

type SwitchGridCommand struct {
	World  *domain.World
	GridID string
}

func (c *SwitchGridCommand) CanExecute() bool {
	_, ok := c.World.GetGrid(c.GridID)
	return ok
}

func (c *SwitchGridCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("grid not found")
	}
	c.World.SetCurrentGrid(c.GridID)

	// Update player position to center of new grid
	grid, _ := c.World.GetGrid(c.GridID)
	playerPos := entity.Position{X: grid.Width / 2, Y: grid.Height / 2}
	c.World.SetPlayerPosition(playerPos)

	return nil
}
