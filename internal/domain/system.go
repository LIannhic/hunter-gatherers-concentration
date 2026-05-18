package domain

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/resource"
)

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
	DreamPlane *board.DreamPlane

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
}

func NewWorld() *World {
	p := player.New("player_1")
	return &World{
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
}

// CreateGrid crée un nouveau grid et l'ajoute au monde
func (w *World) CreateGrid(id string, width, height int, biome board.BiomeType) *board.Grid {
	grid := board.NewGrid(id, width, height, biome)
	w.Grids[id] = grid
	w.GridOrder = append(w.GridOrder, id)
	if w.CurrentGridID == "" {
		w.CurrentGridID = id
	}
	return grid
}

func (w *World) GetCompletionRatio(gridID string) float64 {
	grid, ok := w.GetGrid(gridID)
	if !ok || grid.InitialMatchableCount == 0 {
		return 1.0 // Toujours ouvert si pas de contenu matchable
	}

	// Compte les entités matchables restantes
	currentMatchable := 0
	for _, e := range w.Entities.GetAllActive() {
		// On compte ressources, créatures et pièges comme éléments à appairer
		if e.GetGridID() == gridID && (e.GetType() == entity.TypeResource ||
			e.GetType() == entity.TypeCreature ||
			e.GetType() == entity.TypeTrap) {
			currentMatchable++
		}
	}

	matched := grid.InitialMatchableCount - currentMatchable
	if matched < 0 {
		matched = 0
	}
	return float64(matched) / float64(grid.InitialMatchableCount)
}

func (w *World) IsNavigationOpen(gridID string) bool {
	grid, ok := w.GetGrid(gridID)
	if !ok {
		return false
	}

	// Cheat flag
	if grid.NavigationForcedOpen {
		return true
	}

	// V0.2: Les zones de portail (Départ/Arrivée) sont toujours ouvertes à la navigation
	if w.DreamPlane != nil && (gridID == w.DreamPlane.StartZoneID || gridID == w.DreamPlane.EndZoneID) {
		return true
	}

	ratio := w.GetCompletionRatio(gridID)
	return ratio >= w.Difficulty.NavThreshold
}

// GetGrid retourne un grid par son ID
func (w *World) GetGrid(id string) (*board.Grid, bool) {
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
		// Déclenche l'événement d'entrée pour les systèmes (comme Preview)
		w.EventBus.PublishImmediate(event.NewGridEnteredEvent(gridID))
		return true
	}
	return false
}

// GenerateLayout génère la structure du monde (Dream Plane)
func (w *World) GenerateLayout(id string) {
	gen := board.NewLayoutGenerator()
	w.DreamPlane = gen.GenerateDreamPlane(id, w.Difficulty.Level, w.WorldsCleared)

	// Nettoie les anciens grids et entités
	w.Grids = make(map[string]*board.Grid)
	w.GridOrder = make([]string, 0)
	w.Entities = entity.NewManager() // Reset des entités
	w.Components = component.NewStore()

	// Enregistre les zones dans World
	for _, gridID := range []string{w.DreamPlane.StartZoneID, w.DreamPlane.EndZoneID} {
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
}

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

func (w *World) SetPlayerPosition(pos entity.Position) {
	w.playerPosition = pos
}

func (w *World) GetPlayerPosition() entity.Position {
	return w.playerPosition
}

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
	idsToRemove := make([]string, 0)
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			pos := board.Position{X: center.X - 1 + dx, Y: center.Y - 1 + dy}
			plot, err := grid.Get(pos)
			if err != nil {
				continue
			}
			for _, id := range plot.EntitiesID {
				idsToRemove = append(idsToRemove, id)
			}
			plot.Modifier.Obstructed = false
			plot.StructureID = ""
			plot.EntitiesID = []string{}
		}
	}

	for _, id := range idsToRemove {
		w.RemoveEntity(entity.ID(id))
	}
}

func (w *World) HasPortablePortal() bool {
	for _, item := range w.Player.Inventory.Items {
		if item.SourceID == player.PortablePortalItemSourceID {
			return true
		}
	}
	return false
}

func (w *World) RemovePortablePortal() bool {
	for idx, item := range w.Player.Inventory.Items {
		if item.SourceID == player.PortablePortalItemSourceID {
			_ = w.Player.Inventory.RemoveItem(idx)
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
		_ = w.Player.Inventory.RemoveItem(idx)
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

func (w *World) DeployPortablePortal(gridID string) (entity.Entity, error) {
	return w.DeployPortablePortalAt(gridID, board.Position{X: -1, Y: -1})
}

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

// GetFlippedTilesCount returns how many tiles have been flipped this turn
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

// CanFlipTile checks if another tile can be flipped this turn (max 2 per turn)
func (w *World) CanFlipTile() bool {
	w.GetFlippedTilesCount() // Sync turn tracking
	return len(w.tilesFlippedThisTurn) < 2
}

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
	grid, ok := w.Grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	boardPos := board.Position{X: pos.X, Y: pos.Y}
	plot, err := grid.Get(boardPos)
	if err != nil {
		return nil, err
	}

	if len(plot.EntitiesID) > 0 {
		return nil, fmt.Errorf("position %v déjà occupée", pos)
	}

	// Création explicite d'un pointeur vers une BaseEntity
	trapPtr := &entity.BaseEntity{
		ID:       entity.NewID(),
		EType:    entity.TypeTrap,
		Pos:      pos,
		GridID:   gridID,
		Active:   true,
		State:    entity.Hidden,
		Tags:     []string{"trap"},
		Metadata: make(map[string]interface{}),
	}

	w.Entities.Register(trapPtr)
	grid.InitialMatchableCount++
	plot.PushEntity(string(trapPtr.GetID()))

	w.EventBus.Publish(event.NewEntityCreatedEvent(string(trapPtr.GetID()), "trap"))
	return trapPtr, nil
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

	// Création explicite d'un pointeur vers une BaseEntity
	structPtr := &entity.BaseEntity{
		ID:       entity.NewID(),
		EType:    entity.TypeStructure,
		Pos:      pos,
		GridID:   gridID,
		Active:   true,
		State:    entity.Hidden,
		Tags:     []string{stype},
		Metadata: make(map[string]interface{}),
	}

	// Logique de visibilité initiale
	switch stype {
	case "commencement_portal":
		structPtr.SetState(entity.Revealed)
	case "finish_portal":
		structPtr.SetState(entity.Hidden)
	default:
		structPtr.SetState(entity.Revealed)
	}

	w.Entities.Register(structPtr)
	plot.PushEntity(string(structPtr.GetID()))

	w.EventBus.Publish(event.NewEntityCreatedEvent(string(structPtr.GetID()), "structure"))
	return structPtr, nil
}

// RevealTile révèle une entité sur une position
// RevealTile révèle l'entité au sommet d'une pile sur une position donnée
func (w *World) RevealTile(gridID string, pos board.Position) (entity.Entity, error) {
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

	// 5. Mise à jour de l'état
	state := ent.GetState()
	if state&entity.Hidden != 0 {
		ent.SetState(entity.Revealed)
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

// RemoveEntity supprime une entité du monde, de sa pile sur la grille et de l'ECS
func (w *World) RemoveEntity(id entity.ID) {
	idStr := string(id)

	e, ok := w.Entities.Get(id)
	if !ok {
		return
	}

	grid, ok := w.Grids[e.GetGridID()]
	if ok {
		pos := e.GetPosition()
		_, err := grid.RemoveEntity(board.Position{X: pos.X, Y: pos.Y}, idStr)
		if err != nil {
			fmt.Printf("Erreur lors du retrait du board: %v\n", err)
		}
	}

	w.Components.RemoveEntity(idStr)
	w.Entities.Remove(id)

	w.EventBus.Publish(event.NewEntityRemovedEvent(idStr, "harvested"))
}

// ErrGridNotFound est retourné quand un grid n'existe pas
var ErrGridNotFound = errors.New("grid not found")

// LifecycleSystem gère maturation/dégradation
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

// PropagationSystem gère l'expansion organique des ressources sur la grille
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
		neighbors := grid.GetNeighbors(board.Position{X: pos.X, Y: pos.Y})
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
				// Sécurité si le spawn échoue techniquement (ne devrait pas arriver)
				continue
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

// CreatureAISystem gère les comportements
type CreatureAISystem struct{}

func (s *CreatureAISystem) Priority() int { return 3 }

func (s *CreatureAISystem) Update(world *World) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	ai := world.CreatureFactory.GetAI()

	for _, e := range creatures {
		c, ok := e.(*creature.Creature)
		if !ok || c.MovementProfile != nil {
			continue
		}

		grid, ok := world.Grids[c.GetGridID()]
		if !ok {
			continue
		}

		action := ai.Decide(c, &worldAdapter{world: world, grid: grid})

		switch action.Type {
		case "move":
			oldPos := c.GetPosition()
			newPos := entity.Position{
				X: oldPos.X + action.Direction.X,
				Y: oldPos.Y + action.Direction.Y,
			}

			if !grid.IsValid(board.Position{X: newPos.X, Y: newPos.Y}) {
				continue
			}

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
			))

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

// CreatureMovementSystem gère les déplacements avancés des créatures
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

		// Vérifie la fréquence de déplacement
		if !profile.Frequency.CanMove() {
			continue
		}

		// Détermine combien de cases déplacer
		moveCount := profile.Frequency.GetMoveCount()

		// Exécute les mouvements
		for i := 0; i < moveCount; i++ {
			if !s.executeMove(c, profile, world, grid) {
				break
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
		if c.GetState() == entity.Revealed && !trigger.WasRevealed {
			trigger.WasRevealed = true
			return true
		}
		trigger.WasRevealed = c.GetState() == entity.Revealed
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

	case creature.NavOrientation:
		return c.MovementProfile.Orientation.ToVector()

	case creature.NavAttraction:
		playerPos := world.playerPosition
		current := c.GetPosition()
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

	// Détermine la position dans la pile selon le mode de mouvement
	mode := creature.ModeBento
	if c.MovementProfile != nil {
		mode = c.MovementProfile.Mode.Type
	}

	switch mode {
	case creature.ModeUnder:
		// Se place à la base de la pile (caché par les autres entités)
		grid.PlaceEntityAtBottom(board.Position{X: newPos.X, Y: newPos.Y}, idStr)
	default:
		// Se place au sommet de la pile (priorité visuelle)
		grid.PlaceEntity(board.Position{X: newPos.X, Y: newPos.Y}, idStr)
	}

	world.Entities.UpdatePosition(c.GetID(), newPos)

	// Émission de l'événement
	silent := mode == creature.ModeShadow || mode == creature.ModeUnder
	if silent {
		world.EventBus.Publish(event.Event{
			Type:     event.CreatureMoved,
			SourceID: string(c.GetID()),
			Payload: map[string]interface{}{
				"from":   oldPos,
				"to":     newPos,
				"mode":   "silent",
				"hidden": true,
			},
		})
	} else {
		world.EventBus.Publish(event.NewCreatureMovedEvent(
			string(c.GetID()),
			oldPos,
			newPos,
		))
	}

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

// worldAdapter adapte World pour l'interface creature.WorldState
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

	if len(tile.EntitiesID) == 0 {
		return "empty"
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	if ent, ok := wa.world.Entities.Get(entity.ID(topID)); ok {
		return ent.GetState().String()
	}
	return "unknown"
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TriggerSystem gère les déclencheurs (terriers, etc.)
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

// PreviewSystem gère la révélation des tuiles à l'entrée d'une grille
type PreviewSystem struct {
	previewTimers map[string]int // Timer par gridID
}

func NewPreviewSystem() *PreviewSystem {
	return &PreviewSystem{
		previewTimers: make(map[string]int),
	}
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

	// Pas de prévisualisation dans les zones de commencement et de fin
	isPortalZone := world.DreamPlane != nil && (gridID == world.DreamPlane.StartZoneID || gridID == world.DreamPlane.EndZoneID)
	if isPortalZone {
		if gridID == world.DreamPlane.StartZoneID {
			// Dans la zone de commencement, on laisse 5 secondes au joueur pour voir le portail
			s.previewTimers[gridID] = 5 * 60 // 5 secondes
		}
		return
	}

	settings := world.Difficulty
	if settings.PreviewRatio <= 0 {
		return
	}

	fmt.Printf("[PREVIEW] Entrée sur %s (Difficulté: %s)\n", gridID, settings.Level)

	// Détermine quelles entités révéler
	var allEntities []entity.Entity
	for _, tile := range grid.Plots {
		if len(tile.EntitiesID) > 0 {
			topID := tile.EntitiesID[len(tile.EntitiesID)-1]
			if ent, ok := world.Entities.Get(entity.ID(topID)); ok {
				allEntities = append(allEntities, ent)
			}
		}
	}

	if settings.Level == meta.LevelNormal {
		s.revealHalfPairs(world, allEntities, gridID)
	} else {
		// Easy ou Insane
		for _, e := range allEntities {
			if e.GetState()&entity.Hidden != 0 {
				e.SetState(entity.Revealed)
				world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
					e.GetPosition(), string(e.GetID()), gridID, board.FlipCenter))
			}
		}
	}

	// Lance le timer si une durée est définie
	if settings.PreviewDuration > 0 {
		s.previewTimers[gridID] = int(settings.PreviewDuration * 60) // 60 fps
	}
}

func (s *PreviewSystem) revealHalfPairs(world *World, entities []entity.Entity, gridID string) {
	typeGroups := make(map[string][]entity.Entity)
	for _, e := range entities {
		resType := ""
		if res, ok := e.(*resource.Resource); ok {
			resType = "res_" + res.ResourceType
		} else if cre, ok := e.(*creature.Creature); ok {
			resType = "cre_" + cre.Species
		}

		if resType != "" {
			typeGroups[resType] = append(typeGroups[resType], e)
		}
	}

	ratio := world.Difficulty.PreviewRatio
	for _, group := range typeGroups {
		// Mélange pour ne pas toujours révéler les mêmes positions
		rand.Shuffle(len(group), func(i, j int) {
			group[i], group[j] = group[j], group[i]
		})

		// On révèle un nombre de tuiles proportionnel au ratio (0.5 pour Normal)
		countToReveal := int(float64(len(group)) * ratio)
		for i := 0; i < countToReveal; i++ {
			e := group[i]
			if e.GetState()&entity.Hidden != 0 {
				e.SetState(entity.Revealed)
				world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
					e.GetPosition(), string(e.GetID()), gridID, board.FlipCenter))
			}
		}
	}
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
					if e.HasTag("commencement_portal") {
						// Portail de commencement : se cache et se bloque de manière permanente
						e.SetState(entity.Hidden | entity.Blocked)
						world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
							e.GetPosition(), string(e.GetID()), gridID, board.FlipCenter))
						continue
					}
					// Les autres structures (dolmens, portail de fin) restent révélées
					continue
				}

				if e.GetState()&entity.Revealed != 0 {
					e.SetState(entity.Hidden)
					world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
						e.GetPosition(), string(e.GetID()), gridID, board.FlipCenter))
				}
			}
		}
	}
}

// LootSystem gère l'acquisition du loot lors d'un match
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

	// Un match = un loot
	loot := &player.LootItem{
		ID:          string(entity.NewID()),
		Name:        name,
		Type:        eType,
		SourceID:    string(entID),
		IsDeletable: true, // Par défaut, les items de match sont supprimables
	}

	// Tente d'ajouter à l'inventaire
	err := s.world.Player.Inventory.AddItem(loot)
	if err != nil {
		// Inventaire plein : on détruit le loot
		fmt.Printf("[LOOT] Inventaire plein ! Le loot %s est perdu.\n", name)
		s.world.EventBus.PublishImmediate(event.NewInventoryFullEvent())
		return
	}

	// Une fois le loot acquis, on peut retirer les entités du board (Optionnel selon gameplay)
	// Pour v0.2, les tuiles "Matched" restent visibles en taille 1.2x mais sont "récoltées" techniquement

	fmt.Printf("[LOOT] Acquisition : %s (ID: %s)\n", name, entID)
	s.world.EventBus.PublishImmediate(event.NewLootAcquiredEvent(loot.ID, loot.Name, loot.Type))
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

// Engine orchestre tous les systèmes
type Engine struct {
	systems        []System
	world          *World
	Running        bool
	movementSystem *CreatureMovementSystem // Référence directe pour les mises à jour
	previewSystem  *PreviewSystem          // Référence pour les événements
	lootSystem     *LootSystem
}

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
			lootSys,
		},
		Running:        false,
		movementSystem: moveSys,
		previewSystem:  prevSys,
		lootSystem:     lootSys,
	}

	// S'abonne aux entrées de grille
	world.EventBus.SubscribeFunc(event.GridEntered, func(ev event.Event) {
		gridID := ev.Payload["grid_id"].(string)
		prevSys.OnEnterGrid(world, gridID)
	})

	return e
}

func (e *Engine) Start() {
	e.Running = true
}

func (e *Engine) Stop() {
	e.Running = false
}

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

	if e.movementSystem != nil {
		e.movementSystem.TrackReveal(board.Position{}) // Utilisation factice pour correspondre à l'ancienne signature si nécessaire
		e.movementSystem.ClearReveals()
	}

	for _, sys := range e.systems {
		sys.Update(e.world)
	}

	e.world.EventBus.ProcessQueue()
	e.world.Turn++

	// Diminue la santé mentale à chaque tour
	if e.world.Player != nil {
		e.world.Player.ConsumeSanity(1)
	}

	e.world.EventBus.Publish(event.NewTurnEndedEvent(e.world.Turn))
}

func (e *Engine) UpdateFrame() {
	if e.previewSystem != nil {
		e.previewSystem.Update(e.world)
	}
}

func (e *Engine) TrackTileReveal(pos board.Position) {
	if e.movementSystem != nil {
		e.movementSystem.TrackReveal(pos)
	}
}

func (e *Engine) AddSystem(s System) {
	e.systems = append(e.systems, s)
}

func (e *Engine) GetWorld() *World {
	return e.world
}

func (e *Engine) GetTurn() int {
	return e.world.Turn
}
