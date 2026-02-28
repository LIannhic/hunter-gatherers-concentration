package usecase

import (
	"errors"
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

type Command interface {
	Execute() error
	CanExecute() bool
}

type RevealTileCommand struct {
	World    *domain.World
	GridID   string
	Position board.Position
}

func (c *RevealTileCommand) CanExecute() bool {
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return false
	}
	tile, err := grid.Get(c.Position)
	if err != nil {
		return false
	}
	return tile.State == board.Hidden
}

func (c *RevealTileCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("cannot reveal this tile")
	}

	_, err := c.World.RevealTile(c.GridID, c.Position)
	if err != nil {
		return err
	}

	c.World.EventBus.Publish(domain.Event{
		Type:     domain.EventType("action_reveal"),
		SourceID: "player",
		Payload:  map[string]interface{}{"position": c.Position, "grid_id": c.GridID},
	})

	return nil
}

type MatchTilesCommand struct {
	World      *domain.World
	AssocEng   *domain.AssocEngine
	GridID     string
	Pos1, Pos2 board.Position
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

	if tile1.State != board.Revealed || tile2.State != board.Revealed {
		return false
	}

	if tile1.EntityID == "" || tile2.EntityID == "" {
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

	entity1, ok1 := c.World.Entities.Get(entity.ID(tile1.EntityID))
	entity2, ok2 := c.World.Entities.Get(entity.ID(tile2.EntityID))

	if !ok1 || !ok2 {
		return errors.New("entities not found")
	}

	res1, isRes1 := entity1.(*domain.Resource)
	res2, isRes2 := entity2.(*domain.Resource)

	if !isRes1 || !isRes2 {
		return errors.New("can only match resources")
	}

	result, err := c.AssocEng.TryAssociate(res1, res2)
	if err != nil || !result.Success {
		return fmt.Errorf("association failed: %v", err)
	}

	c.World.MatchTile(c.GridID, c.Pos1)
	c.World.MatchTile(c.GridID, c.Pos2)

	c.World.RemoveEntity(entity1.GetID())
	c.World.RemoveEntity(entity2.GetID())

	c.World.EventBus.Publish(domain.Event{
		Type:     domain.EventType("tiles_matched"),
		SourceID: "player",
		Payload: map[string]interface{}{
			"position1":  c.Pos1,
			"position2":  c.Pos2,
			"grid_id":    c.GridID,
			"assoc_type": result.Type.String(),
		},
	})

	return nil
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
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return errors.New("grid not found")
	}

	// Supprime toutes les entités de ce grid
	for _, e := range c.World.Entities.GetAllActive() {
		if e.GetGridID() == c.GridID {
			c.World.RemoveEntity(e.GetID())
		}
	}

	// Réinitialise les tuiles
	for _, tile := range grid.Tiles {
		tile.State = board.Hidden
		tile.EntityID = ""
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

	// Réinitialise les tuiles de tous les grids
	for _, gridID := range c.World.GridOrder {
		if grid, ok := c.World.GetGrid(gridID); ok {
			for _, tile := range grid.Tiles {
				tile.State = board.Hidden
				tile.EntityID = ""
			}
		}
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
	return nil
}
