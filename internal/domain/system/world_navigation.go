package system

import (
	"errors"
	"fmt"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

// CreateGrid crée un nouveau grid et l'ajoute au monde
func (w *World) CreateGrid(id string, width, height int, biome board.BiomeType) *board.Grid {
	grid := board.NewGrid(id, width, height, biome)
	if id == board.InventoryGridID {
		w.InventoryGrid = grid
		// Ajoute une inclinaison par défaut (SlopeTop) à toutes les parcelles de l'inventaire pour les animations
		for x := 0; x < width; x++ {
			for y := 0; y < height; y++ {
				pos := board.Position{X: x, Y: y}
				if plot, err := grid.Get(pos); err == nil {
					plot.Tilt = board.SlopeTop
				}
			}
		}
		return grid
	}

	w.Grids[id] = grid
	w.GridOrder = append(w.GridOrder, id)
	if w.CurrentGridID == "" {
		w.CurrentGridID = id
	}
	return grid
}

func (w *World) IsNavigationOpen(gridID string) bool {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return false
	}

	isOpen := false
	if grid.NavigationForcedOpen {
		isOpen = true
	} else if w.DreamPlane != nil && (gridID == w.DreamPlane.StartZoneID || gridID == w.DreamPlane.EndZoneID) {
		isOpen = true
	} else {
		activeEntities := w.Entities.GetAllActive()

		remaining := 0
		for _, ent := range activeEntities {
			if ent.GetGridID() != gridID {
				continue
			}
			if ent.GetType() == entity.TypeCreature || ent.GetType() == entity.TypeResource {
				remaining++
			}
		}

		total := remaining + grid.MatchedTargetsCount
		if total == 0 {
			isOpen = true
		} else {
			ratio := float64(grid.MatchedTargetsCount) / float64(total)

			threshold := w.Difficulty.NavThreshold
			if w.Debug.OverrideDifficulty {
				threshold = w.Debug.Difficulty.NavThreshold
			}
			isOpen = ratio >= threshold
		}
	}

	// NOUVEAU : Détection de transition d'état pour les animations de scellage
	if grid.LastNavigationOpen != isOpen {
		grid.LastNavigationOpen = isOpen
		if isOpen {
			fmt.Printf("[NAVIGATION] La zone %s est désormais libre d'accès !\n", gridID)
			w.EventBus.PublishImmediate(event.NewNavigationOpenedEvent(gridID))
		} else {
			fmt.Printf("[NAVIGATION] La zone %s est de nouveau scellée !\n", gridID)
			w.EventBus.PublishImmediate(event.NewNavigationClosedEvent(gridID))
		}
	}

	return isOpen
}

// GetGrid retourne un grid par son ID
func (w *World) GetGrid(id string) (*board.Grid, bool) {
	if id == board.InventoryGridID {
		if w.InventoryGrid == nil {
			return nil, false
		}
		return w.InventoryGrid, true
	}
	grid, ok := w.Grids[id]
	return grid, ok
}

// GetCurrentGrid retourne le grid actuel du joueur
func (w *World) GetCurrentGrid() (*board.Grid, bool) {
	if w.CurrentGridID == "" {
		return nil, false
	}
	return w.GetGrid(w.CurrentGridID)
}

// SetCurrentGrid change le grid actuel du joueur
func (w *World) SetCurrentGrid(gridID string) bool {
	return w.SetCurrentGridFrom(gridID, -1) // -1 pour direction inconnue
}

// SetCurrentGridFrom change la grille active en spécifiant la direction d'arrivée
func (w *World) SetCurrentGridFrom(gridID string, arrivalDir entity.Direction) bool {
	if grid, ok := w.Grids[gridID]; ok {
		w.CurrentGridID = gridID
		w.UpdateDiscovery()

		// NOUVEAU : Persistance de l'entrée
		// Si on arrive d'une direction cardinale, l'entrée correspondante dans la nouvelle grille
		// est immédiatement considérée comme découverte et ouverte.
		if arrivalDir >= entity.DirNorth && arrivalDir <= entity.DirWest {
			// L'entrée est la direction OPPOSÉE à celle empruntée pour arriver ici.
			entranceDir := w.DreamPlane.OppositeDirection(arrivalDir)
			grid.ExitsState[entranceDir] = [2]entity.TileState{
				entity.Revealed | entity.Matched,
				entity.Revealed | entity.Matched,
			}
		}

		// Déclenche l'événement d'entrée pour les systèmes
		w.EventBus.PublishImmediate(event.Event{
			Type:     event.GridEntered,
			SourceID: gridID,
			Payload: map[string]interface{}{
				"grid_id":     gridID,
				"arrival_dir": arrivalDir,
			},
		})
		return true
	}
	return false
}

func (w *World) UpdateDiscovery() {
	if w.DreamPlane == nil || w.CurrentGridID == "" {
		return
	}

	if w.DreamPlane.DiscoveryStates == nil {
		w.DreamPlane.DiscoveryStates = make(map[string]board.DiscoveryState)
	}

	current := w.CurrentGridID
	if _, ok := w.DreamPlane.DiscoveryStates[current]; !ok {
		w.DreamPlane.DiscoveryStates[current] = board.StateHidden
	}

	// Marque la zone actuelle comme visitée.
	w.DreamPlane.DiscoveryStates[current] = board.StateVisited

	neighbors := make(map[string]bool)
	if conns, ok := w.DreamPlane.Connections[current]; ok {
		for _, neighborID := range conns {
			neighbors[neighborID] = true
			if _, ok := w.DreamPlane.DiscoveryStates[neighborID]; !ok {
				w.DreamPlane.DiscoveryStates[neighborID] = board.StateHidden
			}
		}
	}

	for zoneID := range w.DreamPlane.Zones {
		if zoneID == current {
			continue
		}
		if neighbors[zoneID] {
			if w.DreamPlane.DiscoveryStates[zoneID] != board.StateVisited {
				w.DreamPlane.DiscoveryStates[zoneID] = board.StateAdjacent
			}
		}
		// NOTE : On ne remet JAMAIS un état à Hidden si il était déjà Adjacent ou Visited.
		// Cela permet de garder les sorties découvertes affichées sur la mini-carte.
	}
}

// SyncInventoryGrid synchronise la grille logique d'inventaire avec la liste des objets du joueur.
func (w *World) SyncInventoryGrid() {
	grid, ok := w.GetGrid(board.InventoryGridID)
	if !ok {
		return
	}

	// 1. Vide tous les emplacements de la grille d'inventaire
	for _, plot := range grid.Plots {
		plot.EntitiesID = nil
	}

	// 2. Replace les entités selon leur nouvel index (compactage)
	for i, item := range w.Player.Inventory.Items {
		pos := board.Position{X: i % 3, Y: i / 3}
		_ = grid.PlaceEntity(pos, string(item.GetID()))
		item.SetPosition(pos)
		item.SetGridID(board.InventoryGridID)
	}
}

// RemoveLootItem retire un objet de l'inventaire par son index et synchronise la grille.
func (w *World) RemoveLootItem(index int) error {
	if index < 0 || index >= len(w.Player.Inventory.Items) {
		return errors.New("index invalide")
	}

	item := w.Player.Inventory.Items[index]
	// Retire de la liste slice
	err := w.Player.Inventory.RemoveItem(index)
	if err != nil {
		return err
	}

	// Désinscrit l'entité
	w.Entities.Remove(item.GetID())

	// Re-synchronise toute la grille pour le compactage
	w.SyncInventoryGrid()
	return nil
}

// AddLootItem ajoute un objet à l'inventaire et le synchronise sur la grille logicielle.
func (w *World) AddLootItem(item *player.LootItem) error {
	err := w.Player.Inventory.AddItem(item)
	if err != nil {
		return err
	}

	w.Entities.Register(item)
	item.SetState(entity.Revealed) // Toujours visible en inventaire
	w.SyncInventoryGrid()
	return nil
}

// GetHoverableAt retourne l'élément interactif à une position donnée d'une grille.
// Gère les entités standard, le butin d'inventaire et les sorties (via GridID spéciaux).
func (w *World) GetHoverableAt(gridID string, pos board.Position) board.Hoverable {
	// 1. Cas de la grille d'inventaire
	if gridID == board.InventoryGridID {
		grid, ok := w.GetGrid(gridID)
		if !ok {
			return nil
		}
		plot, err := grid.Get(pos)
		if err != nil || len(plot.EntitiesID) == 0 {
			return nil
		}
		topID := plot.EntitiesID[len(plot.EntitiesID)-1]
		ent, ok := w.Entities.Get(entity.ID(topID))
		if !ok {
			return nil
		}
		return ent
	}

	// 2. Cas des sorties (Navigation) - On utilise des GridID virtuels "exit_north", etc.
	if strings.HasPrefix(gridID, "exit_") {
		dirStr := strings.TrimPrefix(gridID, "exit_")
		var dir entity.Direction
		switch dirStr {
		case "north":
			dir = entity.DirNorth
		case "east":
			dir = entity.DirEast
		case "south":
			dir = entity.DirSouth
		case "west":
			dir = entity.DirWest
		default:
			return nil
		}

		grid, ok := w.GetGrid(w.CurrentGridID)
		if !ok {
			return nil
		}

		index := pos.X // Pour les sorties, on utilise X comme index (0 ou 1)
		if index < 0 || index > 1 {
			return nil
		}

		return &board.ExitTile{
			Direction: dir,
			Index:     index,
			State:     grid.ExitsState[dir][index],
		}
	}

	// 3. Cas standard (Grilles de jeu)
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return nil
	}

	plot, err := grid.Get(pos)
	if err != nil || len(plot.EntitiesID) == 0 {
		return nil
	}

	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, ok := w.Entities.Get(entity.ID(topID))
	if !ok {
		return nil
	}

	return ent
}

// GenerateLayout génère la structure du monde (Dream Plane)
func (w *World) GenerateLayout(id string) {
	fmt.Printf("[WORLD] Génération du Dream Plane: %s (Difficulté: %s)\n", id, w.Difficulty.Level)

	gen := board.NewLayoutGenerator()
	w.DreamPlane = gen.GenerateDreamPlane(id, w.Difficulty.Level, w.WorldsCleared)

	// Nettoie les anciens grids et entités
	w.Grids = make(map[string]*board.Grid)
	w.GridOrder = make([]string, 0)
	w.Entities = entity.NewManager() // Reset des entités
	w.Components = component.NewStore()
	w.RevealedBySpecies = make(map[string]int)

	// Enregistre les zones dans World
	// Priorité au début et à la fin
	priorityIDs := []string{w.DreamPlane.StartZoneID, w.DreamPlane.EndZoneID}
	for _, gridID := range priorityIDs {
		if grid, ok := w.DreamPlane.Zones[gridID]; ok {
			w.Grids[grid.ID] = grid
			w.GridOrder = append(w.GridOrder, grid.ID)
		}
	}

	// Ajoute les autres zones (intermédiaires et impasses)
	for id, grid := range w.DreamPlane.Zones {
		found := false
		for _, addedID := range w.GridOrder {
			if addedID == id {
				found = true
				break
			}
		}
		if !found {
			w.Grids[id] = grid
			w.GridOrder = append(w.GridOrder, id)
		}
	}

	w.CurrentGridID = w.DreamPlane.StartZoneID
	w.UpdateDiscovery()

	// Place le joueur au centre de la zone de départ
	if grid, ok := w.Grids[w.CurrentGridID]; ok {
		w.SetPlayerPosition(entity.Position{X: grid.Width / 2, Y: grid.Height / 2})
	}

	w.PopulateInitialStructures()
	w.PopulateZones()

	fmt.Printf("[WORLD] Layout généré avec %d zones. Départ: %s, Fin: %s\n",
		len(w.Grids), w.DreamPlane.StartZoneID, w.DreamPlane.EndZoneID)

	w.EventBus.PublishImmediate(event.NewWorldGeneratedEvent(id, len(w.Grids)))
}

// PopulateZones remplit toutes les zones non-portal générées avec des entités aléatoires.
func (w *World) PopulateZones() {
	if w.DreamPlane == nil {
		return
	}

	for _, gridID := range w.GridOrder {
		if gridID == w.DreamPlane.StartZoneID || gridID == w.DreamPlane.EndZoneID {
			continue
		}
		w.FillGridRandomly(gridID)
	}
}

// GeneratePlaytestLayout génère un monde de test dense avec toutes les entités
func (w *World) GeneratePlaytestLayout(id string) {
	fmt.Printf("[WORLD] Génération du Layout de PLAYTEST: %s\n", id)

	gen := board.NewLayoutGenerator()
	w.DreamPlane = gen.GeneratePlaytestPlane(id)

	w.Grids = make(map[string]*board.Grid)
	w.GridOrder = make([]string, 0)
	w.Entities = entity.NewManager()
	w.Components = component.NewStore()
	w.RevealedBySpecies = make(map[string]int)

	grid := w.DreamPlane.Zones[w.DreamPlane.StartZoneID]
	w.Grids[grid.ID] = grid
	w.GridOrder = append(w.GridOrder, grid.ID)
	w.CurrentGridID = grid.ID
	w.UpdateDiscovery()

	// Population de test précise pour le debug
	fmt.Println("[WORLD] Population de la zone de playtest (Echo Hound en 1,1)...")

	// 1. Un Echo Hound isolé pour tester l'animation et l'orientation
	_, _ = w.SpawnCreature(grid.ID, "echo_hound", entity.Position{X: 1, Y: 1})
	_, _ = w.SpawnCreature(grid.ID, "echo_hound", entity.Position{X: 2, Y: 1}) // Sa paire

	// 2. Population automatique pour le reste
	creatures := []string{"lumifly", "shadowstalker", "burrower", "specter", "moss_monkey", "stonewarden", "flutterwing"}
	resources := []string{"dreamberry", "moonstone", "whispering_herb", "crystal_shard"}

	fmt.Println("[WORLD] Population de la zone de playtest...")

	placePair := func(name string, isCreature bool) {
		count := 0
		for y := 0; y < grid.Height && count < 2; y++ {
			for x := 0; x < grid.Width && count < 2; x++ {
				pos := board.Position{X: x, Y: y}
				plot, _ := grid.Get(pos)
				if plot.StructureID == "" && len(plot.EntitiesID) == 0 {
					if isCreature {
						_, _ = w.SpawnCreature(grid.ID, name, entity.Position(pos))
					} else {
						_, _ = w.SpawnResource(grid.ID, name, entity.Position(pos))
					}
					count++
				}
			}
		}
	}

	for _, c := range creatures {
		placePair(c, true)
	}
	for _, r := range resources {
		placePair(r, false)
	}

	fmt.Printf("[WORLD] Playtest layout prêt. Zones: %d, Entités: %d\n",
		len(w.Grids), w.Entities.Count())

	w.EventBus.PublishImmediate(event.NewWorldGeneratedEvent(id, len(w.Grids)))
}

// RotateGrid fait pivoter une grille et met à jour ses entités
func (w *World) RotateGrid(gridID string) error {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return ErrGridNotFound
	}

	// 1. Sauvegarde la hauteur pour la transformation
	oldHeight := grid.Height

	// 2. Rotation logique du plateau (permute Width/Height, tourne les pentes)
	grid.RotateClockwise()

	// 3. Mise à jour de toutes les entités présentes sur cette grille
	for _, e := range w.Entities.GetAllActive() {
		if e.GetGridID() == gridID {
			// Recalcule la position physique avec l'ancienne hauteur
			oldPos := e.GetPosition()
			newPos := board.Position{
				X: oldHeight - 1 - oldPos.Y,
				Y: oldPos.X,
			}

			// Met à jour la position dans l'interface et l'index du manager
			_ = w.Entities.UpdatePosition(e.GetID(), newPos)

			// Met à jour la transformation diédrique (Rotation du plateau : +90°)
			// On compose la transformation actuelle avec une rotation de 90°.
			currentTrans := e.GetTransformation()
			newTrans := entity.Compose(currentTrans, entity.TransRot90)
			e.SetTransformation(newTrans)
		}
	}

	// 4. Mettre à jour la position du joueur si il est sur cette grille
	if w.CurrentGridID == gridID && w.playerOnBoard {
		oldPlayerPos := w.playerPosition
		w.playerPosition = board.Position{
			X: oldHeight - 1 - oldPlayerPos.Y,
			Y: oldPlayerPos.X,
		}
	}

	// 5. Mettre à jour les tuiles retournées suivies (Memory)
	for i, pos := range w.tilesFlippedThisTurn {
		// On suppose ici que w.tilesFlippedThisTurn sont sur la grille actuelle
		w.tilesFlippedThisTurn[i] = board.Position{
			X: oldHeight - 1 - pos.Y,
			Y: pos.X,
		}
	}

	// 6. Mise à jour des connexions dans le DreamPlane
	if w.DreamPlane != nil {
		w.DreamPlane.RotateConnectionsClockwise(gridID)
	}

	return nil
}
