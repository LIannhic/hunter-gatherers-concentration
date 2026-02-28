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

	// Récupère le grid et révèle la tuile directement
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return errors.New("grid not found")
	}

	tile, err := grid.Reveal(c.Position)
	if err != nil {
		return err
	}

	// Publie l'événement avec la direction de flip
	c.World.EventBus.Publish(event.NewTileRevealedEvent(
		entity.Position{X: c.Position.X, Y: c.Position.Y},
		tile.EntityID,
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

	// Détermine les types des entités
	res1, isRes1 := entity1.(*domain.Resource)
	res2, isRes2 := entity2.(*domain.Resource)
	cre1, isCre1 := entity1.(*domain.Creature)
	cre2, isCre2 := entity2.(*domain.Creature)

	fmt.Printf("[MATCH DEBUG] Entité 1: Resource=%v, Creature=%v\n", isRes1, isCre1)
	fmt.Printf("[MATCH DEBUG] Entité 2: Resource=%v, Creature=%v\n", isRes2, isCre2)

	// Vérifie si c'est une association valide
	isMatch := false
	matchType := ""

	// Cas 1 : Deux ressources - utilise le système d'association
	if isRes1 && isRes2 {
		fmt.Printf("[MATCH DEBUG] Comparaison de ressources: %s vs %s\n", res1.ResourceType, res2.ResourceType)
		result, err := c.AssocEng.TryAssociate(res1, res2)
		if err == nil && result.Success {
			isMatch = true
			matchType = result.Type.String()
			fmt.Printf("[MATCH DEBUG] Association ressource réussie: %s\n", matchType)
		} else {
			fmt.Printf("[MATCH DEBUG] Association ressource échouée: %v\n", err)
		}
	}

	// Cas 2 : Deux créatures - compare les espèces
	if !isMatch && isCre1 && isCre2 {
		fmt.Printf("[MATCH DEBUG] Comparaison de créatures: %s vs %s\n", cre1.Species, cre2.Species)
		if cre1.Species == cre2.Species {
			isMatch = true
			matchType = "creature_capture"
			fmt.Printf("[MATCH DEBUG] Créatures identiques !\n")
		} else {
			fmt.Printf("[MATCH DEBUG] Créatures différentes !\n")
		}
	}

	// Cas 3 : Mix ressource/créature - pas de match
	if !isMatch && ((isRes1 && isCre2) || (isCre1 && isRes2)) {
		fmt.Printf("[MATCH DEBUG] Mix ressource/créature - pas de match possible\n")
	}

	if isMatch {
		// Succès : les tuiles restent visibles et sont marquées comme appairées
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
				"assoc_type": matchType,
			},
		})

		if c.OnSuccess != nil {
			c.OnSuccess()
		}

		fmt.Println("[MATCH] Association réussie !")
		return nil
	} else {
		// Échec : cacher les tuiles et passer le tour
		fmt.Println("[MATCH] Échec de l'association - les cartes sont différentes")

		// Publie un événement d'échec
		c.World.EventBus.Publish(domain.Event{
			Type:     domain.EventType("match_failed"),
			SourceID: "player",
			Payload: map[string]interface{}{
				"position1": c.Pos1,
				"position2": c.Pos2,
				"grid_id":   c.GridID,
			},
		})

		// Cache les tuiles
		grid.Hide(c.Pos1)
		grid.Hide(c.Pos2)

		// Appelle le callback d'échec (pour passer le tour)
		if c.OnFailure != nil {
			c.OnFailure()
		}

		return errors.New("association échouée : les cartes ne correspondent pas")
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
