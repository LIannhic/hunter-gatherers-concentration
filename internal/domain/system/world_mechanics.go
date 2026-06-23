package system

import (
	"errors"
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
)

// FlipTile bascule une entité entre caché et révélé et applique la transformation géométrique
func (w *World) FlipTile(gridID string, pos board.Position, flipDir entity.FlipDirection, reason string) (entity.Entity, error) {
	fmt.Printf("[WORLD-DEBUG] FlipTile appelé: Grid=%s, Pos=%v, Reason=%s\n", gridID, pos, reason)
	grid, ok := w.Grids[gridID]
	if !ok {
		// Tentative de récupération via InventoryGrid si c'est l'ID réservé
		if gridID == board.InventoryGridID {
			grid = w.InventoryGrid
		}

		if grid == nil {
			fmt.Printf("[WORLD-DEBUG] FlipTile ERREUR: Grid %s non trouvé\n", gridID)
			return nil, ErrGridNotFound
		}
	}

	// 1. Récupération du Plot (Parcelle)
	plot, err := grid.Get(pos)
	if err != nil {
		return nil, err
	}

	// 2. Vérification de la présence d'entités (Système de pile)
	n := len(plot.EntitiesID)
	if n == 0 {
		return nil, fmt.Errorf("aucune entité à la position %v", pos)
	}

	// 3. On récupère l'ID au SOMMET de la pile (le dernier ajouté)
	topEntityID := plot.EntitiesID[n-1]

	// 4. Récupération de l'entité via le Manager
	ent, ok := w.Entities.Get(entity.ID(topEntityID))
	if !ok {
		return nil, fmt.Errorf("l'entité %s est enregistrée sur le board mais absente du manager", topEntityID)
	}

	// 5. Basculement de l'état (Toggle Reveal/Hidden)
	currentState := ent.GetState()
	if currentState&entity.Revealed != 0 {
		ent.SetState(entity.Hidden)
	} else {
		ent.SetState(entity.Revealed)
	}

	// 6. Transformation diédrique persistante (Composition globale)
	currentTrans := ent.GetTransformation()
	applyTrans := flipDir.ToTransformation()

	// aux axes de l'écran (le curseur du joueur), peu importe l'état de la tuile.
	newTrans := entity.Compose(currentTrans, applyTrans)
	ent.SetTransformation(newTrans)

	fmt.Printf("[D4] Tuile %s : %s -> %s (via clic %s)\n",
		topEntityID, currentTrans.String(), newTrans.String(), flipDir.String())

	// 7. Notification IMMÉDIATE pour l'UI
	// On utilise PublishImmediate car le Renderer doit capter l'état exact au moment du clic
	// pour lancer le lerp avant que d'autres mutations ne surviennent.
	w.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
		entity.Position(pos),
		topEntityID,
		gridID,
		flipDir,
		map[string]interface{}{"reason": reason},
	))

	return ent, nil
}

// HideInventory retourne toutes les tuiles de l'inventaire face cachée.
func (w *World) HideInventory(flipDir entity.FlipDirection) {
	fmt.Printf("[WORLD] HideInventory déclenché avec direction %v\n", flipDir)
	for i, item := range w.Player.Inventory.Items {
		if item.GetState()&entity.Revealed != 0 {
			pos := board.Position{X: i % 3, Y: i / 3}
			// On utilise le tilt spécifique de la parcelle d'inventaire si disponible
			finalFlipDir := flipDir
			if w.InventoryGrid != nil {
				if plot, err := w.InventoryGrid.Get(pos); err == nil {
					finalFlipDir = plot.Tilt.ToFlipDirection()
				}
			}
			fmt.Printf("[WORLD] Retournement de l'item %d (%s) en %v via direction %v\n", i, item.Name, pos, finalFlipDir)
			_, _ = w.FlipTile(board.InventoryGridID, pos, finalFlipDir, "penalty")
		}
	}
}

// RevealInventory retourne toutes les tuiles de l'inventaire face visible avec animation.
func (w *World) RevealInventory(flipDir entity.FlipDirection) {
	fmt.Printf("[WORLD] RevealInventory déclenché avec direction %v\n", flipDir)
	for i, item := range w.Player.Inventory.Items {
		if item.GetState()&entity.Revealed == 0 {
			pos := board.Position{X: i % 3, Y: i / 3}
			finalFlipDir := flipDir
			if w.InventoryGrid != nil {
				if plot, err := w.InventoryGrid.Get(pos); err == nil {
					finalFlipDir = plot.Tilt.ToFlipDirection()
				}
			}
			fmt.Printf("[WORLD] Révélation de l'item %d (%s) en %v via direction %v\n", i, item.Name, pos, finalFlipDir)
			_, _ = w.FlipTile(board.InventoryGridID, pos, finalFlipDir, "amnesia_end")
		}
	}
}

// RevealTile force une entité à l'état Revealed sans faire de toggle inverse
func (w *World) RevealTile(gridID string, pos board.Position, flipDir entity.FlipDirection, reason string) (entity.Entity, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	plot, err := grid.Get(pos)
	if err != nil {
		return nil, err
	}

	if len(plot.EntitiesID) == 0 {
		return nil, fmt.Errorf("aucune entité à la position %v", pos)
	}

	topEntityID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, ok := w.Entities.Get(entity.ID(topEntityID))
	if !ok {
		return nil, fmt.Errorf("l'entité %s est absente du manager", topEntityID)
	}

	// 1. Force l'état Revealed (uniquement si pas déjà révélé)
	if ent.GetState()&entity.Revealed == 0 {
		ent.SetState(entity.Revealed)

		// 2. Applique la transformation géométrique du flip
		currentTrans := ent.GetTransformation()
		applyTrans := flipDir.ToTransformation()
		newTrans := entity.Compose(currentTrans, applyTrans)
		ent.SetTransformation(newTrans)

		fmt.Printf("[D4] Reveal %s : %s -> %s (via clic %s)\n",
			topEntityID, currentTrans.String(), newTrans.String(), flipDir.String())

		// 3. Publie l'événement pour déclencher l'animation visuelle dans l'UI
		w.EventBus.Publish(event.NewEntityRevealedEvent(
			entity.Position(pos),
			topEntityID,
			gridID,
			flipDir,
			map[string]interface{}{"reason": reason},
		))
	}

	return ent, nil
}

// MatchTile marque l'entité du SOMMET comme appairée
func (w *World) MatchTile(gridID string, pos board.Position) error {
	grid, ok := w.Grids[gridID]
	if !ok {
		return ErrGridNotFound
	}

	plot, err := grid.Get(pos)
	if err != nil {
		return err
	}

	n := len(plot.EntitiesID)
	if n == 0 {
		return errors.New("aucune entité à appairer à cette position")
	}

	topID := plot.EntitiesID[n-1]

	ent, ok := w.Entities.Get(entity.ID(topID))
	if !ok {
		return errors.New("entité au sommet non trouvée dans le manager")
	}

	ent.SetState(entity.Matched)

	return nil
}

// GetFlippedTilesCount retourne le nombre de tuiles retournées ce tour
func (w *World) GetFlippedTilesCount() int {
	// Reset if turn has changed
	if w.lastTurnNumber != w.Turn {
		w.tilesFlippedThisTurn = make([]board.Position, 0)
		w.lastTurnNumber = w.Turn
	}
	return len(w.tilesFlippedThisTurn)
}

// AddFlippedTile enregistre un tile révélé pendant le tour courant
func (w *World) AddFlippedTile(pos board.Position) {
	w.GetFlippedTilesCount() // Sync turn tracking
	w.tilesFlippedThisTurn = append(w.tilesFlippedThisTurn, pos)
}

// CanFlipTile vérifie si une autre tuile peut être retournée ce tour (max 2)
func (w *World) CanFlipTile() bool {
	w.GetFlippedTilesCount() // Sync turn tracking
	return len(w.tilesFlippedThisTurn) < 2
}

// HideAllUnmatchedTiles referme toutes les tuiles révélées qui n'ont pas été associées sur la grille courante.
func (w *World) HideAllUnmatchedTiles() {
	gridID := w.CurrentGridID
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return
	}

	for pos, plot := range grid.Plots {
		if len(plot.EntitiesID) == 0 {
			continue
		}

		topID := plot.EntitiesID[len(plot.EntitiesID)-1]
		if ent, exists := w.Entities.Get(entity.ID(topID)); exists {
			state := ent.GetState()

			// Si la tuile est visible mais pas encore validée (Matched)
			if state&entity.Revealed != 0 && state&entity.Matched == 0 {
				// Modifie l'état logique (retourne la tuile)
				_, _ = w.FlipTile(gridID, pos, plot.Tilt.ToFlipDirection(), "system_hide")

				// Notifie immédiatement le renderer graphique pour jouer l'animation de fermeture
				w.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
					entity.Position(pos),
					string(ent.GetID()),
					gridID,
					plot.Tilt.ToFlipDirection(),
					map[string]interface{}{"reason": "system_hide"},
				))
			}
		}
	}
}
