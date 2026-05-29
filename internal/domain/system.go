package domain

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/structure"
)

// =============================================================================
// SECTION 1: DÉFINITIONS DU CŒUR DU MONDE (WORLD & SYSTEM INTERFACE)
// =============================================================================

// System interface pour les systèmes ECS
type System interface {
	Update(world *World)
	Priority() int // Ordre d'exécution
}

// World contient tout l'état du jeu
type World struct {
	Grids      map[string]*board.Grid // Plusieurs grids indexés par ID
	GridOrder  []string               // Ordre stable des IDs de grid (pour affichage)
	Entities   *entity.Manager
	Components *component.Store
	EventBus   *event.Bus
	Turn       int
	MaxTurns   int
	PlayerID   string

	// Meta progression
	Hub *meta.Hub

	// Player logic
	Player *player.Player

	// Grids actifs pour le joueur (pour navigation entre grids)
	CurrentGridID string

	// Difficulty
	Difficulty meta.DifficultySettings

	// Dream Plane (Mega-board structure)
	DreamPlane *DreamPlane

	// Inventory is modeled as a separate logical grid and not part of the game zone list.
	InventoryGrid *board.Grid

	// Factories
	CreatureFactory *creature.Factory
	ResourceFactory *resource.Factory

	// Player
	playerPosition entity.Position

	// Turn state tracking
	tilesFlippedThisTurn []board.Position // Tracks tiles flipped in current turn (max 2)
	lastTurnNumber       int              // Used to detect turn changes

	// Progression
	WorldsCleared int

	// Real-time turn pressure timer
	TurnTimer *TurnTimer

	// Référence vers l'engine (pour la communication entre systèmes)
	Engine *Engine
}

// NewWorld initialise un nouveau monde avec les réglages par défaut
func NewWorld() *World {
	p := player.New("player_1")
	w := &World{
		Grids:                make(map[string]*board.Grid),
		GridOrder:            make([]string, 0),
		Entities:             entity.NewManager(),
		Components:           component.NewStore(),
		EventBus:             event.NewBus(),
		Turn:                 0,
		MaxTurns:             p.Stats.MaxSanity,
		CurrentGridID:        "",
		Difficulty:           meta.GetSettings(meta.LevelNormal),
		CreatureFactory:      creature.NewFactory(),
		ResourceFactory:      resource.NewFactory(),
		Hub:                  meta.NewHub(),
		Player:               p,
		playerPosition:       entity.Position{X: 0, Y: 0},
		tilesFlippedThisTurn: make([]board.Position, 0),
		lastTurnNumber:       0,
		TurnTimer:            NewTurnTimer(meta.GetSettings(meta.LevelNormal).TurnTimerDuration),
	}

	// Initialise la grille d'inventaire
	// 3 colonnes, 10 lignes pour 30 slots par défaut
	w.CreateGrid(board.InventoryGridID, 3, 10, board.BiomeDefault)

	return w
}

// =============================================================================
// SECTION 2: GESTION DES GRIDS ET NAVIGATION
// =============================================================================

// CreateGrid crée un nouveau grid et l'ajoute au monde
func (w *World) CreateGrid(id string, width, height int, biome board.BiomeType) *board.Grid {
	grid := board.NewGrid(id, width, height, biome)
	if id == board.InventoryGridID {
		w.InventoryGrid = grid
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

	if grid.NavigationForcedOpen {
		return true
	}
	if w.DreamPlane != nil && (gridID == w.DreamPlane.StartZoneID || gridID == w.DreamPlane.EndZoneID) {
		return true
	}

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
		return true
	}

	ratio := float64(grid.MatchedTargetsCount) / float64(total)
	return ratio >= w.Difficulty.NavThreshold
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
	if _, ok := w.Grids[gridID]; ok {
		w.CurrentGridID = gridID
		w.UpdateDiscovery()
		// Déclenche l'événement d'entrée pour les systèmes (comme Preview)
		w.EventBus.PublishImmediate(event.NewGridEnteredEvent(gridID))
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
		} else if w.DreamPlane.DiscoveryStates[zoneID] != board.StateVisited {
			w.DreamPlane.DiscoveryStates[zoneID] = board.StateHidden
		}
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

	gen := NewLayoutGenerator()
	w.DreamPlane = gen.GenerateDreamPlane(id, w.Difficulty.Level, w.WorldsCleared)

	// Nettoie les anciens grids et entités
	w.Grids = make(map[string]*board.Grid)
	w.GridOrder = make([]string, 0)
	w.Entities = entity.NewManager() // Reset des entités
	w.Components = component.NewStore()

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

	gen := NewLayoutGenerator()
	w.DreamPlane = gen.GeneratePlaytestPlane(id)

	w.Grids = make(map[string]*board.Grid)
	w.GridOrder = make([]string, 0)
	w.Entities = entity.NewManager()
	w.Components = component.NewStore()

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
	creatures := []string{"lumifly", "shadowstalker", "burrower", "specter", "moss_monkey", "stonewarden"}
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

	// 1. Rotation logique du plateau
	grid.RotateClockwise()

	// 2. Mise à jour de toutes les entités présentes sur cette grille
	for _, e := range w.Entities.GetAllActive() {
		if e.GetGridID() == gridID {
			// Recalcule la position physique
			oldPos := e.GetPosition()
			newPos := grid.TransformPosition(oldPos)

			// Met à jour la position dans l'interface et l'index du manager
			_ = w.Entities.UpdatePosition(e.GetID(), newPos)

			// Met à jour la transformation diédrique (Rotation du plateau : +90°)
			currentTrans := e.GetTransformation()
			newTrans := entity.Compose(currentTrans, entity.TransRot90)
			e.SetTransformation(newTrans)
		}
	}

	return nil
}

// =============================================================================
// SECTION 3: RECHERCHE ET ÉTAT DU JOUEUR
// =============================================================================

// GetGridForEntity retourne le grid sur lequel se trouve une entité
func (w *World) GetGridForEntity(entityID string) (*board.Grid, bool) {
	e, ok := w.Entities.Get(entity.ID(entityID))
	if !ok {
		return nil, false
	}
	return w.GetGrid(e.GetGridID())
}

// HasResourceAt vérifie s'il y a déjà une ressource à une position donnée
func (w *World) HasResourceAt(gridID string, pos board.Position) bool {
	grid, ok := w.Grids[gridID]
	if !ok {
		return false
	}
	plot, err := grid.Get(pos)
	if err != nil {
		return false
	}
	for _, id := range plot.EntitiesID {
		if e, ok := w.Entities.Get(entity.ID(id)); ok {
			if e.GetType() == entity.TypeResource {
				return true
			}
		}
	}
	return false
}

// SetPlayerPosition définit la position logique du joueur sur la grille
func (w *World) SetPlayerPosition(pos entity.Position) {
	w.playerPosition = pos
}

// GetPlayerPosition retourne la position logique du joueur
func (w *World) GetPlayerPosition() entity.Position {
	return w.playerPosition
}

// MoveSpeciesOneStepTowards est un wrapper pratique pour appeler le CreatureMovementSystem
// depuis d'autres couches (usecases, UI) sans exposer directement le movementSystem.
func (w *World) MoveSpeciesOneStepTowards(species string, target entity.Position) {
	if w.Engine != nil && w.Engine.movementSystem != nil {
		w.Engine.movementSystem.MoveSpeciesOneStepTowards(species, target, w)
	}
}

// =============================================================================
// SECTION 4: MÉCANIQUES DE TUILES (FLIP & MATCH)
// =============================================================================

// FlipTile bascule une entité entre caché et révélé et applique la transformation géométrique
func (w *World) FlipTile(gridID string, pos board.Position, flipDir entity.FlipDirection) (entity.Entity, error) {
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
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

	return ent, nil
}

// RevealTile est un wrapper autour de FlipTile pour la compatibilité (force l'état Revealed)
func (w *World) RevealTile(gridID string, pos board.Position, flipDir entity.FlipDirection) (entity.Entity, error) {
	ent, err := w.FlipTile(gridID, pos, flipDir)
	if err != nil {
		return nil, err
	}
	// Assure que l'état est bien Revealed
	ent.SetState(entity.Revealed)
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
                _, _ = w.FlipTile(gridID, pos, plot.Tilt.ToFlipDirection())

                // Notifie immédiatement le renderer graphique pour jouer l'animation de fermeture
                w.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
                    entity.Position(pos),
                    string(ent.GetID()),
                    gridID,
                    plot.Tilt.ToFlipDirection(),
                ))
            }
        }
    }
}

// =============================================================================
// SECTION 5: LOGIQUE DE SPAWN ET GESTION DES ENTITÉS
// =============================================================================

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

// =============================================================================
// SECTION 6: SYSTÈME DE PORTAIL PORTABLE
// =============================================================================

// FindAvailable3x3DeploymentArea cherche une zone 3x3 libre sur un grid
func (w *World) FindAvailable3x3DeploymentArea(gridID string) (board.Position, bool) {
	grid, ok := w.GetGrid(gridID)
	if !ok || grid.Width < 3 || grid.Height < 3 {
		return board.Position{}, false
	}

	for y := 0; y <= grid.Height-3; y++ {
		for x := 0; x <= grid.Width-3; x++ {
			okArea := true
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					plot, err := grid.Get(board.Position{X: x + dx, Y: y + dy})
					if err != nil {
						okArea = false
						break
					}
					if plot.Modifier.Obstructed {
						okArea = false
						break
					}
					for _, entityID := range plot.EntitiesID {
						if e, ok := w.Entities.Get(entity.ID(entityID)); ok {
							if e.GetType() == entity.TypeStructure {
								okArea = false
								break
							}
						}
					}
					if !okArea {
						break
					}
				}
				if !okArea {
					break
				}
			}
			if okArea {
				return board.Position{X: x + 1, Y: y + 1}, true
			}
		}
	}
	return board.Position{}, false
}

// findBest3x3DeploymentArea cherche une zone 3x3 avec un score optimal (moins d'obstructions)
func (w *World) findBest3x3DeploymentArea(gridID string) (board.Position, bool) {
	grid, ok := w.GetGrid(gridID)
	if !ok || grid.Width < 3 || grid.Height < 3 {
		return board.Position{}, false
	}

	bestScore := 1<<31 - 1
	bestPos := board.Position{}
	for y := 0; y <= grid.Height-3; y++ {
		for x := 0; x <= grid.Width-3; x++ {
			score := 0
			hasStructure := false
			for dy := 0; dy < 3; dy++ {
				for dx := 0; dx < 3; dx++ {
					plot, err := grid.Get(board.Position{X: x + dx, Y: y + dy})
					if err != nil {
						hasStructure = true
						break
					}
					if plot.Modifier.Obstructed {
						score += 10
					}
					for _, entityID := range plot.EntitiesID {
						if e, ok := w.Entities.Get(entity.ID(entityID)); ok {
							if e.GetType() == entity.TypeStructure {
								hasStructure = true
								break
							}
						}
					}
					if hasStructure {
						break
					}
					if len(plot.EntitiesID) > 0 {
						score += 1
					}
				}
				if hasStructure {
					break
				}
			}
			if hasStructure {
				continue
			}
			if score < bestScore {
				bestScore = score
				bestPos = board.Position{X: x + 1, Y: y + 1}
			}
		}
	}
	if bestScore == 1<<31-1 {
		return board.Position{}, false
	}
	return bestPos, true
}

func (w *World) isValid3x3DeploymentCenter(grid *board.Grid, center board.Position) bool {
	return center.X >= 1 && center.Y >= 1 && center.X <= grid.Width-2 && center.Y <= grid.Height-2
}

func (w *World) is3x3DeploymentAreaClear(grid *board.Grid, center board.Position) bool {
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			plot, err := grid.Get(board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy})
			if err != nil {
				return false
			}
			if plot.Modifier.Obstructed {
				return false
			}
			for _, entityID := range plot.EntitiesID {
				if e, ok := w.Entities.Get(entity.ID(entityID)); ok {
					if e.GetType() == entity.TypeStructure {
						return false
					}
				}
			}
		}
	}
	return true
}

func (w *World) clear3x3DeploymentArea(grid *board.Grid, center board.Position) {
	// 1. On crée une liste pour collecter TOUTES les entités de la zone 3x3
	idsToRemove := make([]string, 0)

	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			pos := board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy}

			// plot est un *Plot (pointeur)
			plot, err := grid.Get(pos)
			if err != nil {
				continue
			}

			// On accumule les IDs à supprimer du monde
			for _, id := range plot.EntitiesID {
				idsToRemove = append(idsToRemove, id)
			}

			// 2. On nettoie DIRECTEMENT la tuile dans la grille (via le pointeur)
			plot.Modifier.Obstructed = false
			plot.StructureID = ""
			plot.EntitiesID = nil // 'nil' réinitialise proprement la slice
		}
	}

	// 3. On détruit définitivement les entités auprès du gestionnaire du World
	for _, id := range idsToRemove {
		w.RemoveEntity(entity.ID(id))
	}
}

// HasPortablePortal vérifie si le joueur possède un portail portable dans son inventaire
func (w *World) HasPortablePortal() bool {
	for _, item := range w.Player.Inventory.Items {
		if item.SourceID == player.PortablePortalItemSourceID {
			return true
		}
	}
	return false
}

// RemovePortablePortal retire un portail portable de l'inventaire
func (w *World) RemovePortablePortal() bool {
	for idx, item := range w.Player.Inventory.Items {
		if item.SourceID == player.PortablePortalItemSourceID {
			_ = w.RemoveLootItem(idx)
			return true
		}
	}
	return false
}

func (w *World) applyPortablePortalLootTax() {
	taxAmount := int(float64(len(w.Player.Inventory.Items)) * float64(player.PortablePortalLootTaxPercent) / 100.0)
	if taxAmount <= 0 {
		return
	}

	indices := make([]int, 0, len(w.Player.Inventory.Items))
	for idx := range w.Player.Inventory.Items {
		indices = append(indices, idx)
	}

	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	indices = indices[:taxAmount]
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))
	for _, idx := range indices {
		_ = w.RemoveLootItem(idx)
	}
}

func (w *World) applyDreamBreachPenalty() {
	w.Player.TakeDamage(5, "dream_breach")
	w.Player.ConsumeSanity(0)
	w.EventBus.PublishImmediate(event.Event{
		Type:     event.Type("dream_breach"),
		SourceID: "portable_portal",
		Payload: map[string]interface{}{
			"damage": 5,
			"sanity": 0,
			"reason": "Forced portable portal deployment",
		},
	})
}

// DeployPortablePortal déploie le portail portable à une position automatique
func (w *World) DeployPortablePortal(gridID string) (entity.Entity, error) {
	return w.DeployPortablePortalAt(gridID, board.Position{X: -1, Y: -1})
}

// DeployPortablePortalAt déploie le portail portable à une position précise
func (w *World) DeployPortablePortalAt(gridID string, center board.Position) (entity.Entity, error) {
	if !w.HasPortablePortal() {
		return nil, errors.New("aucun portail portable disponible")
	}

	grid, ok := w.GetGrid(gridID)
	if !ok {
		return nil, ErrGridNotFound
	}

	forced := false
	if center.X < 0 || center.Y < 0 {
		center, ok = w.FindAvailable3x3DeploymentArea(gridID)
		if !ok {
			forced = true
			center, ok = w.findBest3x3DeploymentArea(gridID)
			if !ok {
				return nil, errors.New("impossible de trouver une zone 3x3 pour le portail portable")
			}
		}
	} else {
		if !w.isValid3x3DeploymentCenter(grid, center) {
			return nil, errors.New("zone de déploiement invalide")
		}
		if !w.is3x3DeploymentAreaClear(grid, center) {
			forced = true
		}
	}

	if forced {
		w.clear3x3DeploymentArea(grid, center)
	}

	portal, err := w.SpawnStructure(gridID, "portable_portal", entity.Position{X: center.X, Y: center.Y})
	if err != nil {
		return nil, err
	}

	w.RemovePortablePortal()
	w.applyPortablePortalLootTax()

	if forced {
		w.applyDreamBreachPenalty()
	}

	w.EventBus.PublishImmediate(event.Event{
		Type:     event.Type("portable_portal_deployed"),
		SourceID: string(portal.GetID()),
		Payload: map[string]interface{}{
			"grid_id":  gridID,
			"forced":   forced,
			"position": portal.GetPosition(),
		},
	})

	return portal, nil
}

// =============================================================================
// SECTION 7: SYSTÈMES ECS (LOGIQUE MÉTIER)
// =============================================================================

// --- SYSTEM: LIFECYCLE ---
// Gère maturation/dégradation des ressources
type LifecycleSystem struct{}

func (s *LifecycleSystem) Priority() int { return 1 }

func (s *LifecycleSystem) Update(world *World) {
	// Récupère toutes les entités avec un composant Lifecycle
	entityIDs := world.Components.QueryByComponent("lifecycle")

	for _, entityID := range entityIDs {
		comp, ok := world.Components.Get(entityID, "lifecycle")
		if !ok {
			continue
		}

		lifecycle, ok := comp.(*component.Lifecycle)
		if !ok {
			continue
		}

		// Progresse le cycle
		if lifecycle.Progress() {
			// Émet un événement de maturation
			world.EventBus.Publish(event.NewResourceMaturedEvent(
				entityID,
				lifecycle.GetCurrentStageName(),
			))
		}
	}
}

// --- SYSTEM: PROPAGATION ---
// Gère l'expansion organique des ressources sur la grille
type PropagationSystem struct{}

func (s *PropagationSystem) Priority() int { return 2 }

func (s *PropagationSystem) Update(world *World) {
	resources := world.Entities.GetByType(entity.TypeResource)

	for _, e := range resources {
		entityID := string(e.GetID())

		comp, ok := world.Components.Get(entityID, "lifecycle")
		if !ok {
			continue
		}

		lifecycle := comp.(*component.Lifecycle)
		if !lifecycle.CanPropagate {
			continue
		}

		// Vérifie si la plante a atteint son quota de propagation
		if lifecycle.MaxPropagations != -1 && lifecycle.PropagationsDone >= lifecycle.MaxPropagations {
			continue
		}

		// Vérifie si la condition de propagation est remplie (au MaxStages)
		if !shouldPropagate(lifecycle) {
			continue
		}

		grid, ok := world.Grids[e.GetGridID()]
		if !ok {
			continue
		}

		pos := e.GetPosition()
		allNeighbors := grid.GetNeighbors(board.Position{X: pos.X, Y: pos.Y})

		// Filtre pour ne garder que les voisins cardinaux (Haut, Bas, Gauche, Droite)
		var neighbors []*board.Plot
		for _, n := range allNeighbors {
			// Distance de Manhattan == 1 signifie que c'est un voisin direct (pas une diagonale)
			if abs(n.Position.X-pos.X)+abs(n.Position.Y-pos.Y) == 1 {
				neighbors = append(neighbors, n)
			}
		}

		rand.Shuffle(len(neighbors), func(i, j int) { neighbors[i], neighbors[j] = neighbors[j], neighbors[i] })

		maxToPropagate := lifecycle.PropagationCount
		if maxToPropagate <= 0 {
			maxToPropagate = 1 // Fallback
		}

		// --- 1. PHASE DE VÉRIFICATION DES PLACES DISPONIBLES ---
		var validNeighbors []*board.Plot

		for _, neighbor := range neighbors {
			// Ne retient pas la case s'il y a déjà une ressource dessus
			if world.HasResourceAt(e.GetGridID(), neighbor.Position) {
				continue
			}

			// Ne retient pas la case si elle est obstruée
			if neighbor.Modifier.Obstructed {
				continue
			}

			validNeighbors = append(validNeighbors, neighbor)

			// Si on a trouvé assez de places, on peut s'arrêter de chercher
			if len(validNeighbors) == maxToPropagate {
				break
			}
		}

		// Condition CRITIQUE : Si on n'a pas trouvé EXACTEMENT le nombre requis de cases valides,
		// la propagation est considérée comme irréalisable. On passe à la plante suivante.
		if len(validNeighbors) < maxToPropagate {
			continue
		}

		// --- 2. PHASE D'EXÉCUTION (Garantie d'avoir le compte) ---
		propagatedCount := 0

		for _, targetNeighbor := range validNeighbors {
			spawnPos := entity.Position{
				X: targetNeighbor.Position.X,
				Y: targetNeighbor.Position.Y,
			}

			// Création de la ressource
			newRes, err := world.SpawnResource(e.GetGridID(), getResourceType(e), spawnPos)
			if err != nil {
				continue
			}

			// 99% de chance que la nouvelle ressource soit stérile
			if rand.Float32() < 0.99 {
				if comp, ok := world.Components.Get(string(newRes.GetID()), "lifecycle"); ok {
					if lc, ok := comp.(*component.Lifecycle); ok {
						lc.CanPropagate = false
						fmt.Printf("[PROPA] Une nouvelle %s est née stérile à %v\n", getResourceType(e), spawnPos)
					}
				}
			}

			// Ajustement de la hauteur dans le Plot si nécessaire
			if lifecycle.PropagationLevel != 0 {
				grid.RemoveEntity(targetNeighbor.Position, string(newRes.GetID()))
				grid.PlaceEntityAtBottom(targetNeighbor.Position, string(newRes.GetID()))
			}

			propagatedCount++

			world.EventBus.Publish(event.Event{
				Type:     event.ResourcePropagated,
				SourceID: string(newRes.GetID()),
				Payload: map[string]interface{}{
					"parent_id": entityID,
					"position":  targetNeighbor.Position,
				},
			})
		}

		// Si la propagation complète a eu lieu, on met à jour les compteurs
		if propagatedCount > 0 {
			lifecycle.TurnsInStage = 0
			lifecycle.PropagationsDone++
		}
	}
}

func shouldPropagate(l *component.Lifecycle) bool {
	// Propagation uniquement au dernier stade (ex: "gâté")
	isLastStage := l.CurrentStage == l.MaxStages-1
	// Condition : avoir passé suffisamment de tours dans ce stade (défini par TurnsToNext)
	return isLastStage && l.TurnsInStage >= l.TurnsToNext
}

func getResourceType(e entity.Entity) string {
	if r, ok := e.(*resource.Resource); ok {
		return r.ResourceType
	}
	return "unknown"
}

// --- SYSTEM: TRACk ---
// Gère la disparition progressive des traces (indices)
type TrackSystem struct{}

func (s *TrackSystem) Priority() int { return 5 }

func (s *TrackSystem) Update(world *World) {
	tracks := world.Entities.GetByType(entity.TypeTrack)

	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok {
			continue
		}

		t.Duration--
		if t.Duration <= 0 {
			world.RemoveEntity(t.GetID())
		}
	}
}

// --- SYSTEM: TRIGGER ---
// Gère les déclencheurs (terriers, leurres, etc.)
type TriggerSystem struct{}

func (s *TriggerSystem) Priority() int { return 4 }

func (s *TriggerSystem) Update(world *World) {
	for _, gridID := range world.GridOrder {
		grid, ok := world.GetGrid(gridID)
		if !ok {
			continue
		}

		// CORRECTION : Utilisation de .Plots au lieu de .Tiles
		for _, tile := range grid.Plots {
			if tile.StructureID == "" {
				continue
			}

			comp, ok := world.Components.Get(tile.StructureID, "trigger")
			if !ok {
				continue
			}

			trigger := comp.(*component.Trigger)
			if s.checkCondition(trigger.Condition, tile, world, grid) {
				s.executeAction(trigger.Action, tile, world, grid)
				if trigger.Consumed {
					world.Components.Remove(tile.StructureID, "trigger")
				}
			}
		}
	}
}

func (s *TriggerSystem) checkCondition(condition string, tile *board.Plot, world *World, grid *board.Grid) bool {
	if len(tile.EntitiesID) == 0 {
		return false
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	topEnt, ok := world.Entities.Get(entity.ID(topID))

	switch condition {
	case "reveal_with_creature":
		return ok && topEnt.GetType() == entity.TypeCreature && topEnt.GetState() == entity.Revealed

	case "creature_on_resource":
		if !ok || topEnt.GetType() != entity.TypeCreature {
			return false
		}

		neighbors := grid.GetNeighbors(tile.Position)
		for _, n := range neighbors {
			for _, id := range n.EntitiesID {
				if res, ok := world.Entities.Get(entity.ID(id)); ok {
					if res.GetType() == entity.TypeResource {
						return true
					}
				}
			}
		}
	}
	return false
}

func (s *TriggerSystem) executeAction(action string, tile *board.Plot, world *World, grid *board.Grid) {
	switch action {
	case "creature_flee":
		// On fait fuir la créature qui est au SOMMET de la pile (celle qui a marché sur le trigger)
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := world.Entities.Get(entity.ID(topID)); ok {
				if c, ok := e.(*creature.Creature); ok {
					c.Behavior.State = "fleeing"
				}
			}
		}

	case "reveal_adjacent":
		neighbors := grid.GetNeighbors(tile.Position)
		for _, n := range neighbors {
			// REVELATION : On révèle TOUTES les entités présentes sur les cases voisines
			// (Parce qu'un flash lumineux ou un bruit révèle tout ce qui est caché dans la pile)
			for _, id := range n.EntitiesID {
				if e, ok := world.Entities.Get(entity.ID(id)); ok {
					if e.GetState() == entity.Hidden {
						e.SetState(entity.Revealed)
					}
				}
			}
		}
	}
}

// --- SYSTEM: PREVIEW ---
// Gère la révélation des tuiles à l'entrée d'une grille
type PreviewSystem struct {
	previewTimers map[string]int  // Timer par gridID
	previewed     map[string]bool // Suit si une grille a déjà été prévisualisée
}

func NewPreviewSystem() *PreviewSystem {
	return &PreviewSystem{
		previewTimers: make(map[string]int),
		previewed:     make(map[string]bool),
	}
}

func (s *PreviewSystem) Reset() {
	s.previewTimers = make(map[string]int)
	s.previewed = make(map[string]bool)
}

func (s *PreviewSystem) Priority() int { return 0 }

func (s *PreviewSystem) Update(world *World) {
	// Gère le masquage des tuiles après le délai de preview
	for gridID, timer := range s.previewTimers {
		if timer > 0 {
			s.previewTimers[gridID]--
			if s.previewTimers[gridID] == 0 {
				s.hideGrid(world, gridID)
			}
		}
	}
}

func (s *PreviewSystem) OnEnterGrid(world *World, gridID string) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}

	// Vérifie si la grille a déjà été prévisualisée
	if s.previewed[gridID] {
		return
	}

	// Pas de prévisualisation dans les zones de commencement et de fin
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)
	if isPortalZone {
		if gridID == world.DreamPlane.StartZoneID {
			// Dans la zone de commencement, on laisse 2 secondes au joueur pour voir le portail
			s.previewTimers[gridID] = 2 * 60 // 2 secondes (Ajusté selon TurnTimer)
			s.previewed[gridID] = true
		}
		return
	}

	settings := world.Difficulty
	// Si la durée est <= 0, on ne fait pas de prévisualisation
	if settings.PreviewDuration <= 0 {
		return
	}

	fmt.Printf("[PREVIEW] Première entrée sur %s (Difficulté: %s)\n", gridID, settings.Level)

	// Détermine quelles entités révéler
	for _, tile := range grid.Plots {
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := world.Entities.Get(entity.ID(topID)); ok {
				if e.GetState()&entity.Hidden != 0 {
					// Révélation instantanée sans animation
					e.SetState(entity.Revealed)
				}
			}
		}
	}

	// Marque la grille comme prévisualisée
	s.previewed[gridID] = true

	// Lance le timer
	s.previewTimers[gridID] = int(settings.PreviewDuration * 60) // 60 fps
}

func (s *PreviewSystem) hideGrid(world *World, gridID string) {
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return
	}
	fmt.Printf("[PREVIEW] Fin du délai sur %s, masquage des tuiles.\n", gridID)
	for _, tile := range grid.Plots {
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if e, ok := world.Entities.Get(entity.ID(topID)); ok {
				// Gestion spécifique des structures
				if e.GetType() == entity.TypeStructure {
					if e.HasTag("start_portal") {
						// Portail de commencement : se cache et se bloque de manière permanente
						flipDir := tile.Tilt.ToFlipDirection()
						_, _ = world.FlipTile(gridID, tile.Position, flipDir)
						e.SetState(entity.Hidden | entity.Blocked) // Assure l'état final

						world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
							e.GetPosition(), string(e.GetID()), gridID, flipDir))
						continue
					}
					// Les autres structures (dolmens, portail de fin) restent révélées
					continue
				}

				if e.GetState()&entity.Revealed != 0 {
					// Masquage avec animation basée sur la pente (Slope)
					_, _ = world.FlipTile(gridID, tile.Position, tile.Tilt.ToFlipDirection())

					world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
						e.GetPosition(), string(e.GetID()), gridID, tile.Tilt.ToFlipDirection()))
				}
			}
		}
	}
}

// --- SYSTEM: LOOT ---
// Gère l'acquisition du loot lors d'un match
type LootSystem struct {
	world *World
}

func NewLootSystem(world *World) *LootSystem {
	ls := &LootSystem{world: world}
	// S'abonne aux événements de match
	world.EventBus.SubscribeFunc(event.TileMatched, ls.onTileMatched)
	return ls
}

func (s *LootSystem) Priority() int { return 10 }

func (s *LootSystem) Update(world *World) {
	// Pas de logique frame-by-frame pour le moment
}

func (s *LootSystem) onTileMatched(e event.Event) {
	// Récupère l'entité matchée principale
	entID := entity.ID(e.SourceID)

	name := "unknown"
	var eType entity.Type = entity.TypeResource

	// 1. Tente de récupérer depuis le manager si l'entité existe encore
	if ent, ok := s.world.Entities.Get(entID); ok {
		name = s.getEntityName(ent)
		eType = ent.GetType()
	}

	// 2. Sinon (ou si getEntityName a échoué), tente de récupérer depuis le payload
	if name == "unknown" || name == "" {
		if n, exists := e.Payload["name"].(string); exists {
			name = n
		}
		if t, exists := e.Payload["entity_type"].(entity.Type); exists {
			eType = t
		}
	}

	// 3. Cas particulier : pièges ou entités sans butin
	if name == "unknown" || name == "" || name == "trap" {
		return
	}

	// Pour les créatures et ressources, le SourceID correspond au nom de l'espèce ou du type
	// Cela permet de retrouver facilement les assets (ex: creature_echo_hound)
	sourceID := name

	// Un match = un loot
	loot := &player.LootItem{
		BaseEntity:   entity.NewBaseEntity(entity.TypeLoot),
		Name:         name,
		SourceID:     sourceID,
		OriginalType: eType,
		IsUsable:     name == player.EchoHoundItemName || name == player.DreamberryItemName || name == player.MoonstoneItemName || name == player.CrystalShardItemName || name == player.WhisperingHerbItemName || name == player.SpecterItemName || name == player.BurrowerItemName || name == player.ShadowstalkerItemName || name == player.MossMonkeyItemName || name == player.StonewardenItemName || name == player.LumiflyItemName,
		IsDeletable:  true,
	}
	// On garde le type d'origine en tag pour le rendu
	loot.AddTag(fmt.Sprintf("original_type_%d", eType))
	loot.AddTag(name)

	// Tente d'ajouter à l'inventaire via le World pour la synchronisation des grilles
	err := s.world.AddLootItem(loot)
	if err != nil {
		// Inventaire plein : on détruit le loot
		fmt.Printf("[LOOT] Inventaire plein ! Le loot %s est perdu.\n", name)
		s.world.EventBus.PublishImmediate(event.NewInventoryFullEvent())
		return
	}

	// Une fois le loot acquis, on peut retirer les entités du board
	fmt.Printf("[LOOT] Acquisition : %s (ID: %s)\n", name, entID)
	s.world.EventBus.PublishImmediate(event.NewLootAcquiredEvent(string(loot.GetID()), loot.Name, loot.OriginalType))
}

func (s *LootSystem) getEntityName(ent entity.Entity) string {
	if r, ok := ent.(*resource.Resource); ok {
		return r.ResourceType
	}
	if c, ok := ent.(*creature.Creature); ok {
		return c.Species
	}
	return "unknown_loot"
}

// --- LOGIQUE DE SCANNER (ECHO HOUND) ---

func (w *World) TriggerScannerEffect(gridID string) error {
	_, ok := w.GetGrid(gridID)
	if !ok {
		return errors.New("grille introuvable")
	}

	fmt.Printf("[WORLD] L'Echo Hound hurle sur la zone %s !\n", gridID)

	// 1. On crée une liste des positions des entités cachées
	scannedPositions := make([]board.Position, 0)

	for _, e := range w.Entities.GetAllActive() {
		if e.GetGridID() == gridID && e.GetState()&entity.Hidden != 0 {
			pos := e.GetPosition()
			scannedPositions = append(scannedPositions, board.Position{X: pos.X, Y: pos.Y})
		}
	}

	// 2. On publie un événement immédiat pour l'UI
	w.EventBus.PublishImmediate(event.Event{
		Type:     event.Type("scanner_triggered"),
		SourceID: "echo_hound",
		Payload: map[string]interface{}{
			"grid_id":   gridID,
			"positions": scannedPositions,
			"duration":  2.0,
		},
	})

	return nil
}

// --- SYSTEM: CREATURE AI ---
// Gère les prises de décision des créatures (pseudo-ECS)
type CreatureAISystem struct{}

func (s *CreatureAISystem) Priority() int { return 3 }

func (s *CreatureAISystem) Update(world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	ai := world.CreatureFactory.GetAI()

	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok {
			continue
		}

		grid, ok := world.Grids[c.GetGridID()]
		if !ok {
			continue
		}

		action := ai.Decide(c, &worldAdapter{world: world, grid: grid})

		switch action.Type {
		case "move":
			// Si la créature a un profil de mouvement, on délègue le déplacement
			// au CreatureMovementSystem pour profiter des triggers et animations.
			if c.MovementProfile != nil {
				continue
			}

			oldPos := c.GetPosition()
			newPos := entity.Position{
				X: oldPos.X + action.Direction.X,
				Y: oldPos.Y + action.Direction.Y,
			}

			if !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
				continue
			}

			// NOUVEAU : Enregistre le mouvement pour le trigger de proximité
			world.Engine.TrackTileReveal(board.Position{X: oldPos.X, Y: oldPos.Y})
			world.Engine.TrackTileReveal(board.Position{X: newPos.X, Y: newPos.Y})

			newPlot, _ := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})

			idStr := string(c.GetID())
			_, err := grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)
			if err != nil {
				continue
			}

			if len(newPlot.EntitiesID) > 0 {
				topID := newPlot.EntitiesID[len(newPlot.EntitiesID)-1]
				if oldEnt, ok := world.Entities.Get(entity.ID(topID)); ok {
					if oldEnt.GetType() == entity.TypeTrap {
						world.RemoveEntity(oldEnt.GetID())
					}
				}
			}

			newPlot.PushEntity(idStr)

			world.Entities.UpdatePosition(c.GetID(), newPos)

			world.EventBus.Publish(event.NewCreatureMovedEvent(
				idStr,
				oldPos,
				newPos,
				"manifest", // mode
				false,      // hidden
				false,      // audible
			))

		case "spawn_trap":
			// Le singe dépose un piège standard
			trap, err := world.SpawnTrap(c.GetGridID(), c.GetPosition())
			if err == nil {
				// On place le piège SOUS le singe dans la pile
				if grid, ok := world.GetGrid(c.GetGridID()); ok {
					pos := board.Position(c.GetPosition())
					grid.RemoveEntity(pos, string(trap.GetID()))
					grid.PlaceEntityAtBottom(pos, string(trap.GetID()))
				}
				fmt.Printf("[ACTION] %s a posé un piège à %v\n", c.Species, c.GetPosition())
			}

		case "flee":
			fmt.Printf("[ACTION] %s fuit la zone car le plateau est plein !\n", c.Species)
			// Publie l'événement de fuite avant de retirer l'entité
			world.EventBus.Publish(event.NewCreatureFledEvent(
				string(c.GetID()),
				c.Species,
				c.GetGridID(),
				c.GetPosition(),
			))
			world.RemoveEntity(c.GetID())

		case "transform":
			// Logique de transformation (pollinisation, etc.)
			targetID := action.TargetID
			if targetID != "" {
				if comp, ok := world.Components.Get(targetID, "lifecycle"); ok {
					if lifecycle, ok := comp.(*component.Lifecycle); ok {
						lifecycle.CurrentStage++ // Force la maturation
					}
				}
			}
		}
	}
}

// --- SYSTEM: CREATURE MOVEMENT (ADVANCED) ---
// Gère les déplacements avancés des créatures
type CreatureMovementSystem struct {
	recentReveals []board.Position // Tuiles récemment révélées pour TriggerOnEcho
}

func NewCreatureMovementSystem() *CreatureMovementSystem {
	return &CreatureMovementSystem{
		recentReveals: make([]board.Position, 0),
	}
}

func (s *CreatureMovementSystem) Priority() int { return 3 }

func (s *CreatureMovementSystem) TrackReveal(pos board.Position) {
	s.recentReveals = append(s.recentReveals, pos)
}

func (s *CreatureMovementSystem) ClearReveals() {
	s.recentReveals = s.recentReveals[:0]
}

func (s *CreatureMovementSystem) Update(world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)

	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok || c.MovementProfile == nil {
			continue
		}

		// Récupère le grid de la créature
		grid, ok := world.GetGrid(c.GetGridID())
		if !ok {
			continue
		}

		profile := c.MovementProfile

		// Vérifie si le déplacement doit se déclencher
		if !s.shouldTrigger(profile.Trigger, c) {
			continue
		}

		// Ignore si la créature a déjà bougé ce tour via un mouvement externe ou une commande spéciale
		if profile.Frequency.HasMovedThisTurn(world.Turn) {
			continue
		}

		// Vérifie la fréquence de déplacement
		if !profile.Frequency.CanMove() {
			continue
		}

		// Détermine combien de cases déplacer
		moveCount := profile.Frequency.GetMoveCount()

		// Exécute les mouvements
		moved := false
		for i := 0; i < moveCount; i++ {
			if !s.executeMove(c, profile, world, grid) {
				break
			}
			moved = true
		}

		if moved {
			profile.Frequency.MarkMoved(world.Turn)
		}

		// Comportements spécifiques après un déplacement déclenché par "OnReveal"
		if profile.Trigger.Type == creature.TriggerOnReveal {
			switch c.Species {
			case "stonewarden":
				// Après la première révélation, devient patrouilleur suivant son orientation
				profile.Trigger.Type = creature.TriggerAuto
				profile.Navigation.Type = creature.NavOrientation
			case "echo_hound":
				// Oriente ses révélations vers des ressources (dreamberries)
				profile.Navigation.Target = creature.TargetResource
			}
		}

		profile.Trigger.Reset()
	}
}

func (s *CreatureMovementSystem) shouldTrigger(trigger creature.MovementTrigger, c *creature.Creature) bool {
	switch trigger.Type {
	case creature.TriggerPassive:
		return false
	case creature.TriggerAuto:
		return true
	case creature.TriggerOnReveal:
		// Trigger when this creature's tile was revealed earlier this turn
		for _, revealed := range s.recentReveals {
			if revealed.X == c.GetPosition().X && revealed.Y == c.GetPosition().Y {
				if !trigger.Triggered {
					trigger.Triggered = true
					return true
				}
			}
		}
		return false
	case creature.TriggerOnEcho:
		return len(s.recentReveals) > 0
	case creature.TriggerProximity:
		for _, revealed := range s.recentReveals {
			dist := abs(revealed.X-c.GetPosition().X) + abs(revealed.Y-c.GetPosition().Y)
			if dist <= trigger.Radius {
				return true
			}
		}
		return false
	}
	return false
}

func (s *CreatureMovementSystem) executeMove(c *creature.Creature, profile *creature.MovementProfile, world *World, grid *board.Grid) bool {
	direction := s.getNavigationDirection(profile.Navigation, c, world, grid)

	if direction == (entity.Position{X: 0, Y: 0}) {
		return true
	}

	currentPos := c.GetPosition()
	newPos := entity.Position{
		X: currentPos.X + direction.X,
		Y: currentPos.Y + direction.Y,
	}

	profile.Orientation = directionToOrientation(direction)

	finalPos, success := s.handleCollision(profile.Collision, c, newPos, currentPos, world, grid)
	if !success {
		return false
	}

	return s.applyMoveMode(profile.Mode, c, currentPos, finalPos, world, grid)
}

func (s *CreatureMovementSystem) getNavigationDirection(nav creature.NavigationLogic, c *creature.Creature, world *World, grid *board.Grid) entity.Position {
	switch nav.Type {
	case creature.NavWander:
		directions := []entity.Position{
			{X: 0, Y: -1}, {X: 0, Y: 1},
			{X: -1, Y: 0}, {X: 1, Y: 0},
		}
		if nav.WanderBias != (entity.Position{}) && rand.Float32() < 0.3 {
			newPos := entity.Position{
				X: c.GetPosition().X + nav.WanderBias.X,
				Y: c.GetPosition().Y + nav.WanderBias.Y,
			}
			if s.isWalkable(c, newPos, grid, world) {
				return nav.WanderBias
			}
		}
		return directions[rand.Intn(len(directions))]

	case creature.NavPatrol:
		if len(nav.PatrolRoute) == 0 {
			return s.getNavigationDirection(creature.NavigationLogic{Type: creature.NavWander}, c, world, grid)
		}
		target := nav.PatrolRoute[nav.PatrolIndex]
		current := c.GetPosition()
		dir := entity.Position{
			X: sign(target.X - current.X),
			Y: sign(target.Y - current.Y),
		}
		if dir.X == 0 && dir.Y == 0 {
			nextIndex := (nav.PatrolIndex + 1) % len(nav.PatrolRoute)
			target = nav.PatrolRoute[nextIndex]
			dir = entity.Position{
				X: sign(target.X - current.X),
				Y: sign(target.Y - current.Y),
			}
		}
		return dir

	case creature.NavRelative:
		if len(nav.PatrolRoute) == 0 {
			return s.getNavigationDirection(creature.NavigationLogic{Type: creature.NavWander}, c, world, grid)
		}

		// 1. On récupère la direction locale du pattern (ex: {X:1, Y:0} pour aller à droite)
		baseDir := nav.PatrolRoute[nav.PatrolIndex]

		// 2. On ajuste le vecteur selon l'orientation (Direction) de la créature
		finalDir := baseDir
		if orient, ok := c.GetComponent("orientation").(*creature.Orientation); ok {
			switch orient.Direction {
			case entity.DirEast:
				// Rotation de 90° horaire : (X, Y) devient (-Y, X)
				finalDir = entity.Position{X: -baseDir.Y, Y: baseDir.X}
			case entity.DirSouth:
				// Rotation de 180° : (X, Y) devient (-X, -Y)
				finalDir = entity.Position{X: -baseDir.X, Y: -baseDir.Y}
			case entity.DirWest:
				// Rotation de 270° : (X, Y) devient (Y, -X)
				finalDir = entity.Position{X: baseDir.Y, Y: -baseDir.X}
				// case entity.DirNorth: reste identique (finalDir = baseDir)
			}
		}

		// 3. On calcule la position absolue ciblée sur la grille
		targetPos := entity.Position{
			X: c.GetPosition().X + finalDir.X,
			Y: c.GetPosition().Y + finalDir.Y,
		}

		// 4. Si le déplacement est impossible, on retourne un vecteur nul {0, 0} pour ce tour.
		if !s.isWalkable(c, targetPos, grid, world) {
			return entity.Position{X: 0, Y: 0}
		}

		// 5. Le déplacement est valide, on met à jour l'index persistant de l'IA pour le prochain tour
		profile := c.MovementProfile
		if profile != nil {
			profile.Navigation.PatrolIndex = (nav.PatrolIndex + 1) % len(nav.PatrolRoute)
		}

		return finalDir

	case creature.NavOrientation:
		return c.MovementProfile.Orientation.ToVector()

	case creature.NavAttraction:
		current := c.GetPosition()

		// Cas spécial : Singe Mousse qui cherche le vide
		if nav.Target == creature.TargetEmpty {
			emptyPlots := []board.Position{}
			for pos, plot := range grid.Plots {
				if len(plot.EntitiesID) == 0 {
					emptyPlots = append(emptyPlots, pos)
				}
			}

			if len(emptyPlots) == 0 {
				return entity.Position{X: 0, Y: 0}
			}

			// Recherche déterministe (Haut-Gauche) de la case la plus proche
			var nearest board.Position
			minDist := 9999
			for _, p := range emptyPlots {
				d := abs(p.X-current.X) + abs(p.Y-current.Y)
				if d < minDist {
					minDist = d
					nearest = p
				} else if d == minDist {
					if p.Y < nearest.Y || (p.Y == nearest.Y && p.X < nearest.X) {
						nearest = p
					}
				}
			}
			return entity.Position{
				X: sign(nearest.X - current.X),
				Y: sign(nearest.Y - current.Y),
			}
		}

		// Si on cible une ressource précise (ex: echo_hound -> dreamberry)
		if nav.Target == creature.TargetResource && nav.TargetName != "" {
			// Recherche de la ressource la plus proche
			var nearest entity.Position
			minDist := 9999
			for _, ent := range world.Entities.GetAllActive() {
				if res, ok := ent.(*resource.Resource); ok {
					if res.ResourceType == nav.TargetName {
						pos := res.GetPosition()
						d := abs(pos.X-current.X) + abs(pos.Y-current.Y)
						if d < minDist {
							minDist = d
							nearest = pos
						}
					}
				}
			}
			if minDist < 9999 {
				return entity.Position{X: sign(nearest.X - current.X), Y: sign(nearest.Y - current.Y)}
			}
		}

		// Défaut : Attraction vers le joueur
		playerPos := world.playerPosition
		return entity.Position{
			X: sign(playerPos.X - current.X),
			Y: sign(playerPos.Y - current.Y),
		}

	case creature.NavRepulsion:
		playerPos := world.playerPosition
		current := c.GetPosition()
		return entity.Position{
			X: sign(current.X - playerPos.X),
			Y: sign(current.Y - playerPos.Y),
		}
	}
	return entity.Position{X: 0, Y: 0}
}

// MoveSpeciesOneStepTowards déplace d'une case toutes les créatures d'une espèce donnée
// vers la position cible (utilisé par les comportements spéciaux comme les shadowstalkers).
func (s *CreatureMovementSystem) MoveSpeciesOneStepTowards(species string, target entity.Position, world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok {
			continue
		}
		if c.Species != species {
			continue
		}
		// Only consider creatures on the same grid as the player
		grid, ok := world.GetGrid(c.GetGridID())
		if !ok {
			continue
		}

		if c.MovementProfile != nil && c.MovementProfile.Frequency.HasMovedThisTurn(world.Turn) {
			continue
		}

		currentPos := c.GetPosition()
		// Compute a single step towards the target (Manhattan step)
		dir := entity.Position{
			X: sign(target.X - currentPos.X),
			Y: sign(target.Y - currentPos.Y),
		}
		if dir == (entity.Position{X: 0, Y: 0}) {
			continue
		}

		newPos := entity.Position{X: currentPos.X + dir.X, Y: currentPos.Y + dir.Y}

		finalPos, success := s.handleCollision(creature.CollisionHandler{Type: creature.CollidePhase}, c, newPos, currentPos, world, grid)
		if !success {
			// try next creature
			continue
		}

		// Apply move mode using creature's profile if available
		mode := creature.MovementMode{Type: creature.ModeNormal}
		if c.MovementProfile != nil {
			mode = c.MovementProfile.Mode
		}
		moved := s.applyMoveMode(mode, c, currentPos, finalPos, world, grid)
		if moved && c.MovementProfile != nil {
			c.MovementProfile.Frequency.MarkMoved(world.Turn)
		}
	}
}

func (s *CreatureMovementSystem) isWalkable(c *creature.Creature, pos entity.Position, grid *board.Grid, world *World) bool {
	tile, err := grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil || tile.Modifier.Obstructed {
		return false
	}

	// Case vide : toujours ok
	if len(tile.EntitiesID) == 0 {
		return true
	}

	// Mode Phase (spectres) : traverse tout dans les limites
	if c.MovementProfile != nil && c.MovementProfile.Collision.Type == creature.CollidePhase {
		return true
	}

	// Capacité "Grimpe" : permet de passer par-dessus les autres entités
	if c.HasTag("climb") {
		return true
	}

	// Vérifie le sommet pour les pièges
	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	if ent, ok := world.Entities.Get(entity.ID(topID)); ok {
		if ent.GetType() == entity.TypeTrap {
			return true
		}
	}

	// Cohabitation pour Over/Under
	if c.MovementProfile != nil {
		mode := c.MovementProfile.Mode.Type
		if mode == creature.ModeOver || mode == creature.ModeUnder {
			return true
		}
	}

	return false
}

func (s *CreatureMovementSystem) handleCollision(coll creature.CollisionHandler, c *creature.Creature, newPos, currentPos entity.Position, world *World, grid *board.Grid) (entity.Position, bool) {
	canMove := s.isWalkable(c, newPos, grid, world)

	switch coll.Type {
	case creature.CollideStop:
		if !canMove {
			return currentPos, false
		}
		return newPos, true

	case creature.CollideBounce:
		if !canMove {
			c.MovementProfile.Orientation.Rotate(180)
			return currentPos, false
		}
		return newPos, true

	case creature.CollideSlide:
		if canMove {
			return newPos, true
		}

		dx := newPos.X - currentPos.X
		dy := newPos.Y - currentPos.Y

		// Tentative de glissade latérale
		if dy != 0 {
			slidePos := entity.Position{X: currentPos.X, Y: newPos.Y}
			if s.isWalkable(c, slidePos, grid, world) {
				return slidePos, true
			}
		}
		if dx != 0 {
			slidePos := entity.Position{X: newPos.X, Y: currentPos.Y}
			if s.isWalkable(c, slidePos, grid, world) {
				return slidePos, true
			}
		}
		return currentPos, false

	case creature.CollidePhase:
		if !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
			return currentPos, false
		}
		return newPos, true
	}

	if !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
		return currentPos, false
	}
	return newPos, true
}

func (s *CreatureMovementSystem) applyMoveMode(mode creature.MovementMode, c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid) bool {
	if mode.Type == creature.ModeSwap {
		tile, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
		if err == nil && len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			swappedEntity, ok := world.Entities.Get(entity.ID(topID))
			if ok {
				idStr := string(c.GetID())
				grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)
				grid.RemoveEntity(board.Position{X: newPos.X, Y: newPos.Y}, topID)
				grid.PlaceEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, topID)
				grid.PlaceEntity(board.Position{X: newPos.X, Y: newPos.Y}, idStr)
				swappedEntity.SetPosition(oldPos)
				c.SetPosition(newPos)
				world.Entities.UpdatePosition(swappedEntity.GetID(), oldPos)
				world.Entities.UpdatePosition(c.GetID(), newPos)

				return true
			}
		}
	}

	// Dans tous les autres cas (ou si c'était du sol nu), on fait un déplacement normal (doMove)
	return s.doMove(c, oldPos, newPos, world, grid)
}

func (s *CreatureMovementSystem) doMove(c *creature.Creature, oldPos, newPos entity.Position, world *World, grid *board.Grid) bool {
	if grid == nil || !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
		return false
	}

	idStr := string(c.GetID())
	grid.RemoveEntity(board.Position{X: oldPos.X, Y: oldPos.Y}, idStr)

	newTile, err := grid.Get(board.Position{X: newPos.X, Y: newPos.Y})
	if err == nil && newTile != nil && len(newTile.EntitiesID) > 0 {
		topID := newTile.EntitiesID[len(newTile.EntitiesID)-1]
		if ent, ok := world.Entities.Get(entity.ID(topID)); ok && ent.GetType() == entity.TypeTrap {
			world.RemoveEntity(ent.GetID())
		}
	}

	// 1. Détermine la position dans la pile selon le mode PHYSIQUE de déplacement
	mode := creature.ModeNormal
	if c.MovementProfile != nil {
		mode = c.MovementProfile.Mode.Type
	}

	switch mode {
	case creature.ModeUnder:
		// Se place à la base de la pile (caché visuellement sous les autres entités)
		grid.PlaceEntityAtBottom(board.Position{X: newPos.X, Y: newPos.Y}, idStr)
	default:
		// ModeNormal, ModeOver, ModeSwap se placent au sommet
		grid.PlaceEntity(board.Position{X: newPos.X, Y: newPos.Y}, idStr)
	}

	world.Entities.UpdatePosition(c.GetID(), newPos)

	// 2. Détermination de la discrétion basée sur le nouveau profil de PERCEPTION
	isCloaked := false
	isAudible := false

	if c.MovementProfile != nil {
		isCloaked = c.MovementProfile.Perception.Stealth == creature.StealthCloaked
		isAudible = c.MovementProfile.Perception.Acoustic == creature.AcousticEcho
	}

	if c.MovementProfile != nil && c.MovementProfile.Perception.LeavesTracks {
		pProfile := c.MovementProfile.Perception

		// On passe les deux positions pour matérialiser l'interstice
		trackEnt := entity.NewTrack(pProfile.TrackType, pProfile.TrackDuration, oldPos, newPos)
		trackEnt.SetGridID(c.GetGridID())
		world.Entities.Register(trackEnt)

		// Les traces ne sont plus enregistrées sur la pile de la grille (Plot.EntitiesID)
		// car ce ne sont pas des tuiles interactives (Memory).
		// Elles sont gérées directement par le TrackRenderer via le manager d'entités.
	}

	// Émission de l'événement mis à jour
	// hidden est vrai UNIQUEMENT si la créature est camouflée (cloaked)
	// mode transmet la strate (under, normal, over)
	world.EventBus.Publish(event.Event{
		Type:     event.CreatureMoved,
		SourceID: string(c.GetID()),
		Payload: map[string]interface{}{
			"from":    oldPos,
			"to":      newPos,
			"mode":    string(mode),
			"hidden":  isCloaked, // Indique si on saute l'animation (Shadowstalker)
			"audible": isAudible,
		},
	})
	return true
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

func directionToOrientation(dir entity.Position) creature.Orientation {
	if dir.X > 0 {
		return creature.Orientation{Direction: creature.DirEast}
	}
	if dir.X < 0 {
		return creature.Orientation{Direction: creature.DirWest}
	}
	if dir.Y > 0 {
		return creature.Orientation{Direction: creature.DirSouth}
	}
	return creature.Orientation{Direction: creature.DirNorth}
}

// --- WORLD ADAPTER ---
// Adapte World pour l'interface creature.WorldState
type worldAdapter struct {
	world *World
	grid  *board.Grid
}

func (wa *worldAdapter) GetPlayerPosition() entity.Position {
	return wa.world.playerPosition
}

func (wa *worldAdapter) GetNearbyCreatures(pos entity.Position, radius int) []*creature.Creature {
	var result []*creature.Creature
	creatures := wa.world.Entities.GetByType(entity.TypeCreature)

	for _, e := range creatures {
		if e.GetGridID() != wa.grid.ID {
			continue
		}
		if c, ok := e.(*creature.Creature); ok {
			dist := abs(c.GetPosition().X-pos.X) + abs(c.GetPosition().Y-pos.Y)
			if dist <= radius {
				result = append(result, c)
			}
		}
	}
	return result
}

func (wa *worldAdapter) GetResources(pos entity.Position, radius int) []string {
	var result []string
	resources := wa.world.Entities.GetByType(entity.TypeResource)

	for _, e := range resources {
		if e.GetGridID() != wa.grid.ID {
			continue
		}
		dist := abs(e.GetPosition().X-pos.X) + abs(e.GetPosition().Y-pos.Y)
		if dist <= radius {
			result = append(result, string(e.GetID()))
		}
	}
	return result
}

func (wa *worldAdapter) GetEmptyPlots() []entity.Position {
	var empty []entity.Position
	for pos, plot := range wa.grid.Plots {
		if len(plot.EntitiesID) == 0 {
			empty = append(empty, entity.Position{X: pos.X, Y: pos.Y})
		}
	}
	return empty
}

func (wa *worldAdapter) GetGridTotalPlots() int {
	return wa.grid.Width * wa.grid.Height
}

func (wa *worldAdapter) HasActivityNearby(pos entity.Position, radius int) bool {
	// Vérifie si une révélation ou un mouvement a eu lieu dans le rayon
	if wa.world.Engine.movementSystem == nil {
		return false
	}

	for _, activityPos := range wa.world.Engine.movementSystem.recentReveals {
		dist := abs(activityPos.X-pos.X) + abs(activityPos.Y-pos.Y)
		if dist <= radius {
			return true
		}
	}
	return false
}

func (wa *worldAdapter) IsValidMove(pos entity.Position) bool {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return false
	}

	// Retourne vrai si la case est vide ou contient un piège (accessible par défaut à l'IA simple)
	if len(tile.EntitiesID) == 0 {
		return true
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	if ent, ok := wa.world.Entities.Get(entity.ID(topID)); ok {
		if ent.GetType() == entity.TypeTrap {
			return true
		}
	}

	return false
}

func (wa *worldAdapter) GetTileState(pos entity.Position) string {
	tile, err := wa.grid.Get(board.Position{X: pos.X, Y: pos.Y})
	if err != nil {
		return "invalid"
	}

	count := len(tile.EntitiesID)
	if count == 0 {
		return "empty"
	}

	// CAS SPÉCIAL : Si la case ne contient qu'une seule entité, l'IA vérifie si c'est elle-même
	// (Puisque GetTileState est appelé par une créature, si count == 1, c'est forcément elle)
	if count == 1 {
		return "alone"
	}

	return "occupied"
}

func (wa *worldAdapter) IsGridSaturatedWithTraps() bool {
	// On considère la grille saturée s'il ne reste plus aucune ressource à matcher
	// et aucune créature à saboter (autre que les singes mousses eux-mêmes)
	for _, plot := range wa.grid.Plots {
		for _, id := range plot.EntitiesID {
			if ent, ok := wa.world.Entities.Get(entity.ID(id)); ok {
				eType := ent.GetType()
				// S'il reste une ressource ou une créature (autre que moss_monkey), la grille n'est pas saturée
				if eType == entity.TypeResource {
					return false
				}
				if eType == entity.TypeCreature {
					if c, ok := ent.(*creature.Creature); ok {
						if c.Species != "moss_monkey" {
							return false
						}
					}
				}
			}
		}
	}
	return true
}

// --- UTILS ---

// ErrGridNotFound est retourné quand un grid n'existe pas
var ErrGridNotFound = errors.New("grid not found")

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// --- SYSTEM: ATTACK INTENT ---

// AttackIntent représente une intention d'attaque canalisée vers le curseur
type AttackIntent struct {
	SourcePos entity.Position
	TargetX   float64
	TargetY   float64
	Intensity float64
	IsActive  bool
}

type IntentSystem struct {
	world *World
}

func (s *IntentSystem) UpdateIntent(intent *AttackIntent, cursorPixelX, cursorPixelY float64, tileSize, spacing float64) {
	// 1. Met à jour la position de la cible dans l'intention
	intent.TargetX = cursorPixelX
	intent.TargetY = cursorPixelY

	// Note : L'angle directionnel sera calculé directement au moment du Draw
	// pour éviter de stocker trop de variables d'état volatiles dans le domaine.
}

// =============================================================================
// SECTION 8: ENGINE (ORCHESTRATION DES SYSTÈMES)
// =============================================================================

// Engine orchestre tous les systèmes
type Engine struct {
	systems        []System
	world          *World
	Running        bool
	movementSystem *CreatureMovementSystem // Référence directe pour les mises à jour
	previewSystem  *PreviewSystem          // Référence pour les événements
	lootSystem     *LootSystem
}

// NewEngine initialise le moteur de jeu avec ses systèmes
func NewEngine(world *World) *Engine {
	moveSys := NewCreatureMovementSystem()
	prevSys := NewPreviewSystem()
	lootSys := NewLootSystem(world)

	e := &Engine{
		world: world,
		systems: []System{
			&LifecycleSystem{},
			&PropagationSystem{},
			&CreatureAISystem{},
			moveSys,
			&TriggerSystem{},
			&TrackSystem{},
			lootSys,
		},
		Running:        false,
		movementSystem: moveSys,
		previewSystem:  prevSys,
		lootSystem:     lootSys,
	}

	// Lie l'engine au monde pour permettre aux systèmes de communiquer
	world.Engine = e

	// S'abonne aux entrées de grille
	world.EventBus.SubscribeFunc(event.GridEntered, func(ev event.Event) {
		gridID := ev.Payload["grid_id"].(string)
		prevSys.OnEnterGrid(world, gridID)
	})

	return e
}

// Start active la simulation automatique
func (e *Engine) Start() {
	e.Running = true
}

// Stop met en pause la simulation automatique
func (e *Engine) Stop() {
	e.Running = false
}

// ResetPreviews réinitialise le suivi des prévisualisations (pour une nouvelle partie)
func (e *Engine) ResetPreviews() {
	if e.previewSystem != nil {
		e.previewSystem.Reset()
	}
}

// Update fait progresser le tour de jeu
func (e *Engine) Update() {
	if !e.Running {
		return
	}

	for i := 0; i < len(e.systems)-1; i++ {
		for j := i + 1; j < len(e.systems); j++ {
			if e.systems[i].Priority() > e.systems[j].Priority() {
				e.systems[i], e.systems[j] = e.systems[j], e.systems[i]
			}
		}
	}

	for _, sys := range e.systems {
		sys.Update(e.world)
	}

	// Nettoie les révélations après le traitement des systèmes
	if e.movementSystem != nil {
		e.movementSystem.ClearReveals()
	}

	e.world.EventBus.ProcessQueue()
	e.world.Turn++

	// Diminue la santé mentale à chaque tour
	if e.world.Player != nil {
		e.world.Player.ConsumeSanity(1)
	}

	e.world.EventBus.Publish(event.NewTurnEndedEvent(e.world.Turn))
}

// UpdateFrame effectue les mises à jour visuelles (temps réel)
func (e *Engine) UpdateFrame() {
	if e.previewSystem != nil {
		e.previewSystem.Update(e.world)
	}
}

// TrackTileReveal enregistre une interaction pour les systèmes de proximité
func (e *Engine) TrackTileReveal(pos board.Position) {
	if e.movementSystem != nil {
		e.movementSystem.TrackReveal(pos)
	}
}

// AddSystem ajoute dynamiquement un système au moteur
func (e *Engine) AddSystem(s System) {
	e.systems = append(e.systems, s)
}

// GetWorld retourne la référence du monde
func (e *Engine) GetWorld() *World {
	return e.world
}

// GetTurn retourne le numéro du tour actuel
func (e *Engine) GetTurn() int {
	return e.world.Turn
}
