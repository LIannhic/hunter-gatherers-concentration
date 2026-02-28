package usecase

import (
	"errors"
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/association"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

type MatchManager struct {
	grid       *board.Grid
	entities   *entity.Manager
	components *component.Store
	engine     *association.Engine
	eventBus   *event.Bus

	firstSelected entity.ID
}

func NewMatchManager(g *board.Grid, m *entity.Manager, s *component.Store, e *association.Engine, b *event.Bus) *MatchManager {
	fmt.Println("[MatchManager] Initialisation du gestionnaire d'associations")
	return &MatchManager{
		grid:          g,
		entities:      m,
		components:    s,
		engine:        e,
		eventBus:      b,
		firstSelected: "",
	}
}

func (mm *MatchManager) AttemptMatch(pos board.Position) (*association.Result, error) {
	fmt.Printf("\n[Input] Clic détecté sur la position : %s\n", pos.String())

	tile, err := mm.grid.Get(pos)
	if err != nil {
		fmt.Printf("[Erreur] Position invalide : %v\n", err)
		return nil, err
	}

	if tile.EntityID == "" {
		fmt.Println("[Action] Clic sur une tuile vide, rien ne se passe.")
		return nil, errors.New("tuile vide")
	}

	entityID := entity.ID(tile.EntityID)

	if mm.firstSelected == "" {
		mm.firstSelected = entityID
		mm.grid.Reveal(pos)
		fmt.Printf("[Sélection] Première tuile choisie : %s (ID: %s)\n", pos.String(), entityID)

		mm.eventBus.PublishImmediate(event.NewTileRevealedEvent(entity.Position(pos), string(entityID)))
		return nil, nil
	}

	if mm.firstSelected == entityID {
		fmt.Println("[Info] Le joueur a cliqué deux fois sur la même tuile.")
		return nil, errors.New("même tuile sélectionnée")
	}

	firstID := mm.firstSelected
	secondID := entityID
	fmt.Printf("[Analyse] Tentative d'association entre %s et %s\n", firstID, secondID)

	compA, okA := mm.components.Get(string(firstID), "matchable")
	compB, okB := mm.components.Get(string(secondID), "matchable")

	if !okA || !okB {
		fmt.Println("[Erreur] L'une des entités ne possède pas le composant 'matchable'")
		mm.ResetSelection(pos)
		return nil, errors.New("entités non matchables")
	}

	matchA := compA.(association.Matchable)
	matchB := compB.(association.Matchable)

	result, err := mm.engine.TryAssociate(matchA, matchB)

	if result.Success {
		fmt.Printf("[SUCCÈS] Type: %s | Message: %s\n", result.Type.String(), result.Message)

		mm.grid.Match(pos)
		if ent, ok := mm.entities.Get(firstID); ok {
			mm.grid.Match(board.Position(ent.GetPosition()))
		}

		for _, eff := range result.Effects {
			fmt.Printf("  -> Effet détecté : %s sur la cible %s\n", eff.Type, eff.Target)
		}

		mm.eventBus.PublishImmediate(event.NewTileMatchedEvent(entity.Position(pos), string(secondID)))
	} else {
		fmt.Printf("[ÉCHEC] Les tuiles ne correspondent pas. Raison : %v\n", err)

		mm.grid.Hide(pos)
		if ent, ok := mm.entities.Get(firstID); ok {
			mm.grid.Hide(board.Position(ent.GetPosition()))
		}
	}

	mm.firstSelected = ""
	mm.eventBus.PublishImmediate(event.NewAssociationMadeEvent("player", result.Type.String(), result.Success))

	return &result, nil
}

func (mm *MatchManager) ResetSelection(currentPos board.Position) {
	fmt.Println("[Action] Réinitialisation de la sélection forcée.")
	if mm.firstSelected != "" {
		if ent, ok := mm.entities.Get(mm.firstSelected); ok {
			mm.grid.Hide(board.Position(ent.GetPosition()))
		}
	}
	mm.grid.Hide(currentPos)
	mm.firstSelected = ""
}
