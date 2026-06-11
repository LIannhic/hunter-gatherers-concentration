package system

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/structure"
)

// getPlotForSpawn est une méthode utilitaire interne pour centraliser la récupération d'une parcelle valide pour le spawn
func (w *World) getPlotForSpawn(gridID string, pos entity.Position) (*board.Grid, *board.Plot, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, nil, ErrGridNotFound
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}
	plot, err := grid.Get(boardPos)
	if err != nil {
		return nil, nil, err
	}

	if plot.Modifier.Obstructed {
		return nil, nil, fmt.Errorf("position %v est obstruée", pos)
	}

	return grid, plot, nil
}

// SpawnResource crée une ressource dans le monde sur un grid spécifique
func (w *World) SpawnResource(gridID string, rtype string, pos entity.Position) (*resource.Resource, error) {
	_, plot, err := w.getPlotForSpawn(gridID, pos)
	if err != nil {
		return nil, err
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}

	// Uniquement une ressource par parcelle
	if w.HasResourceAt(gridID, boardPos) {
		return nil, fmt.Errorf("position %v contient déjà une ressource", pos)
	}

	r := w.ResourceFactory.Create(rtype, entity.Position{X: pos.X, Y: pos.Y})
	r.SetGridID(gridID)
	// Orientation aléatoire pour les ressources
	r.SetOrientation(entity.Direction(rand.Intn(4)))

	grid, _ := w.GetGrid(gridID)
	grid.InitialMatchableCount++

	idStr := string(r.GetID())
	w.Entities.Register(r)
	w.Components.Add(idStr, &r.Lifecycle)
	w.Components.Add(idStr, &r.Value)
	w.Components.Add(idStr, &r.Matchable)
	w.Components.Add(idStr, &r.Visual)

	plot.PushEntity(idStr)

	w.EventBus.Publish(event.NewEntityCreatedEvent(idStr, "resource"))
	return r, nil
}

// SpawnResourceLevel crée une resource à un niveau précis de la pile
func (w *World) SpawnResourceLevel(gridID string, rtype string, pos entity.Position) (*resource.Resource, error) {
	_, plot, err := w.getPlotForSpawn(gridID, pos)
	if err != nil {
		return nil, err
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}

	// Uniquement une ressource par parcelle
	if w.HasResourceAt(gridID, boardPos) {
		return nil, fmt.Errorf("position %v contient déjà une ressource", pos)
	}

	r := w.ResourceFactory.Create(rtype, entity.Position{X: pos.X, Y: pos.Y})
	r.SetGridID(gridID)
	// Orientation aléatoire pour les ressources
	r.SetOrientation(entity.Direction(rand.Intn(4)))

	grid, _ := w.GetGrid(gridID)
	grid.InitialMatchableCount++

	idStr := string(r.GetID())
	w.Entities.Register(r)
	w.Components.Add(idStr, &r.Lifecycle)
	w.Components.Add(idStr, &r.Value)
	w.Components.Add(idStr, &r.Matchable)
	w.Components.Add(idStr, &r.Visual)

	plot.PushEntityToBottom(idStr)

	w.EventBus.Publish(event.NewEntityCreatedEvent(idStr, "resource"))
	return r, nil
}

// SpawnCreature crée une créature dans le monde sur un grid spécifique
func (w *World) SpawnCreature(gridID string, species string, pos entity.Position) (*creature.Creature, error) {
	_, plot, err := w.getPlotForSpawn(gridID, pos)
	if err != nil {
		return nil, err
	}

	if len(plot.EntitiesID) > 0 {
		return nil, fmt.Errorf("position %v is already occupied by %d entities", pos, len(plot.EntitiesID))
	}

	c, err := w.CreatureFactory.Create(species, pos)
	if err != nil {
		return nil, err
	}

	c.SetGridID(gridID)
	// L'orientation est déjà définie par le factory/profil, mais on l'assure ici
	if c.MovementProfile != nil {
		c.SetOrientation(c.MovementProfile.Orientation.Direction)
	}

	grid, _ := w.GetGrid(gridID)
	grid.InitialMatchableCount++

	idStr := string(c.GetID())

	w.Entities.Register(c)
	w.Components.Add(idStr, &c.Behavior)
	w.Components.Add(idStr, &c.Mobility)
	w.Components.Add(idStr, &c.Visual)

	plot.PushEntity(idStr)

	w.EventBus.Publish(event.NewEntityCreatedEvent(idStr, "creature"))
	return c, nil
}

// SpawnTrap crée un piège sur un grid spécifique
func (w *World) SpawnTrap(gridID string, pos entity.Position) (entity.Entity, error) {
	_, plot, err := w.getPlotForSpawn(gridID, pos)
	if err != nil {
		return nil, err
	}

	trap := entity.NewTrap(pos)
	trap.SetGridID(gridID)

	w.Entities.Register(trap)

	grid, _ := w.GetGrid(gridID)
	grid.InitialMatchableCount++
	plot.PushEntity(string(trap.GetID()))

	w.EventBus.Publish(event.NewEntityCreatedEvent(string(trap.GetID()), "trap"))
	return trap, nil
}

// SpawnStructure crée une structure sur un grid spécifique
func (w *World) SpawnStructure(gridID string, stype string, pos entity.Position) (entity.Entity, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}
	plot, err := grid.Get(boardPos)
	if err != nil {
		return nil, err
	}

	s := structure.NewStructure(stype, pos)
	s.SetGridID(gridID)

	w.Entities.Register(s)
	plot.PushEntity(string(s.GetID()))

	// Les dolmens et obélisques sont physiquement bloquants
	if stype == "dolmen" || stype == "obelisk" {
		plot.Modifier.Obstructed = true
	}

	w.EventBus.Publish(event.NewEntityCreatedEvent(string(s.GetID()), "structure"))
	return s, nil
}

// PopulateInitialStructures parcourt toutes les zones pour créer les entités structures
func (w *World) PopulateInitialStructures() {
	fmt.Println("[WORLD] Population des structures initiales...")
	for _, gridID := range w.GridOrder {
		grid, _ := w.GetGrid(gridID)

		isStartZone := w.DreamPlane != nil && gridID == w.DreamPlane.StartZoneID
		isEndZone := w.DreamPlane != nil && gridID == w.DreamPlane.EndZoneID
		isPortalZone := isStartZone || isEndZone

		if isStartZone {
			fmt.Printf("[INIT] Zone de DÉPART (%s) détectée.\n", gridID)
		} else if isEndZone {
			fmt.Printf("[INIT] Zone de FIN (%s) détectée.\n", gridID)
		}

		for pos, plot := range grid.Plots {
			if plot.StructureID != "" {
				stype := "unknown"
				// Décodage de l'ID de structure (ex: "start_portal" ou "struct_dolmen_1_1")
				if plot.StructureID == "start_portal" || plot.StructureID == "finish_portal" {
					stype = plot.StructureID
				} else if strings.HasPrefix(plot.StructureID, "struct_") {
					parts := strings.Split(plot.StructureID, "_")
					if len(parts) >= 2 {
						stype = parts[1]
					}
				}

				if stype != "unknown" {
					_, err := w.SpawnStructure(gridID, stype, entity.Position{X: pos.X, Y: pos.Y})
					if err == nil && isPortalZone {
						fmt.Printf("  - [%s] Structure créée : %s en (%d, %d)\n", gridID, stype, pos.X, pos.Y)
					}
				}
			}
		}
	}
}

// RemoveEntity supprime une entité du monde, de sa pile sur la grille et de l'ECS
func (w *World) RemoveEntity(id entity.ID) {
	idStr := string(id)

	e, ok := w.Entities.Get(id)
	if !ok {
		return
	}

	pos := e.GetPosition()
	gridID := e.GetGridID()

	// Enregistre l'activité pour les déclencheurs de créatures
	if w.Engine != nil {
		w.Engine.TrackTileReveal(board.Position{X: pos.X, Y: pos.Y})
	}

	// Ne tente de retirer de la grille que si ce n'est pas une Trace (qui n'y est jamais enregistrée)
	if e.GetType() != entity.TypeTrack {
		grid, ok := w.Grids[gridID]
		if ok {
			_, err := grid.RemoveEntity(board.Position{X: pos.X, Y: pos.Y}, idStr)
			if err != nil {
				//fmt.Printf("[WORLD] Erreur lors du retrait de %s du board: %v\n", idStr, err)
			} else {
				fmt.Printf("[WORLD] Entité %s supprimée de la grille %s à la position %v\n", idStr, gridID, pos)
			}
		}
	}

	w.Components.RemoveEntity(idStr)
	w.Entities.Remove(id)

	w.EventBus.Publish(event.NewEntityRemovedEvent(idStr, "harvested"))
}
