package usecase

import (
	"errors"
	"fmt"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

// DefaultFlipDirection est la direction par défaut si non spécifiée
var DefaultFlipDirection = domain.FlipCenter

type Command interface {
	Execute() error
	CanExecute() bool
}

// --- REVEAL TILE COMMAND ---

type RevealTileCommand struct {
	World         *domain.World
	GridID        string
	Position      board.Position
	FlipDirection domain.FlipDirection
}

func (c *RevealTileCommand) CanExecute() bool {
	// 1. Vérifie si la grille existe
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return false
	}

	// 2. Vérifie si le joueur est sur cette grille
	if c.World.CurrentGridID != c.GridID {
		return false
	}

	// 3. Vérifie si la parcelle existe et contient une entité cachée
	tile, err := grid.Get(c.Position)
	if err != nil {
		return false
	}

	if len(tile.EntitiesID) == 0 {
		return false
	}

	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	ent, ok := c.World.Entities.Get(entity.ID(topID))
	if !ok {
		return false
	}

	// Vérifie le coût en mana pour révéler une tuile cumulée (Coût = Niveau)
	if ent.GetCumulationLevel() > 0 {
		if c.World.Player.Stats.Mana < ent.GetCumulationLevel() {
			return false
		}
	}

	state := ent.GetState()
	// Si c'est un piège révélé, on peut toujours le recacher (flip inverse)
	if ent.GetType() == entity.TypeTrap && state&entity.Revealed != 0 {
		return state&entity.Matched == 0
	}

	// Sinon, impossible de révéler une tuile déjà révélée ou appairée
	if state&entity.Revealed != 0 || state&entity.Matched != 0 {
		return false
	}

	// 4. Vérifie la limite de tuiles retournées par tour
	if !c.World.CanFlipTile() {
		return false
	}

	return true
}

func (c *RevealTileCommand) Execute() error {
	if c.World.CurrentGridID != c.GridID {
		fmt.Printf("[ERREUR] Joueur sur la grille %s mais a tenté %s\n", c.World.CurrentGridID, c.GridID)
		return errors.New("le joueur n'est pas sur cette grille")
	}

	if !c.CanExecute() {
		return errors.New("impossible de révéler cette tuile")
	}

	// Consommation de mana pour les tuiles cumulées
	grid, _ := c.World.GetGrid(c.GridID)
	tile, _ := grid.Get(c.Position)
	topID := tile.EntitiesID[len(tile.EntitiesID)-1]
	ent, _ := c.World.Entities.Get(entity.ID(topID))

	if ent.GetCumulationLevel() > 0 {
		c.World.Player.ConsumeMana(ent.GetCumulationLevel())
		fmt.Printf("[ALCHIMIE] Révélation d'une tuile Niv.%d : -%d Mana\n", ent.GetCumulationLevel(), ent.GetCumulationLevel())
	}

	// Calcule la position réelle du joueur en périphérie de la tuile et son ancrage
	offset, border := flipToPlayerState(c.FlipDirection)
	playerPos := entity.Position{
		X: c.Position.X + offset.X,
		Y: c.Position.Y + offset.Y,
	}

	// Met à jour l'état du joueur
	c.World.Player.SetAnchor(border)
	c.World.SetPlayerPosition(playerPos)

	// Déplace les shadowstalkers d'une case vers le joueur (comportement pré-révélation)
	if c.World != nil {
		c.World.MoveSpeciesOneStepTowards("shadowstalker", c.World.GetPlayerPosition())
	}

	// Révèle l'entité via le world
	ent, err := c.World.RevealTile(c.GridID, c.Position, c.FlipDirection)
	if err != nil {
		return err
	}

	// Logique de Confrontation (Zone de Menace)
	if cre, ok := ent.(*creature.Creature); ok {
		isThreatened := cre.IsPositionThreatened(playerPos)

		fmt.Printf("[DEBUG] Reveal Créature: %s en %v | Menacé ? %v\n", cre.Species, cre.GetPosition(), isThreatened)

		// Si le joueur est sous l'effet de Grace (Flutterwing), il évite l'attaque
		if isThreatened && c.World.Player.GraceTurns > 0 {
			fmt.Printf("[ABILITY] Grâce active : l'attaque de %s est évitée !\n", cre.Species)
			isThreatened = false
		}

		if isThreatened {
			fmt.Printf("[COMBAT] Confrontation ! La créature %s attaque le joueur en %v\n", cre.Species, playerPos)
			c.World.Player.TakeDamage(10, "physical")

			// Déclenche les effets visuels (Shaders) selon l'espèce
			if cre.Species == "shadowstalker" {
				c.World.Player.VisualEffects["blur"] = 3 // Dure 3 tours
			} else if cre.Species == "lumifly" {
				c.World.Player.VisualEffects["bubble"] = 3
			}

			// Publie un événement de dégâts pour l'UI (le renderer gérera le feedback visuel)
			c.World.EventBus.Publish(event.Event{
				Type:     event.PlayerDamaged,
				SourceID: string(cre.GetID()),
				Payload: map[string]interface{}{
					"damage":   10,
					"type":     "physical",
					"reason":   "confrontation",
					"position": playerPos, // Ajoute la position pour le renderer
				},
			})
		}
	}

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

// --- MATCH TILES COMMAND ---

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
	OnSuccess  func()
	OnFailure  func()
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
	if e1.GetState()&entity.Revealed == 0 || e2.GetState()&entity.Revealed == 0 {
		return false
	}

	// Vérifie le coût en mana
	// Match = 1 mana.
	if c.World.Player.Stats.Mana < 1 {
		return false
	}

	return true
}

func (c *MatchTilesCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible d'appairer ces tuiles (mana insuffisant ou invalide)")
	}

	grid, _ := c.World.GetGrid(c.GridID)
	tile1, _ := grid.Get(c.Pos1)
	tile2, _ := grid.Get(c.Pos2)

	topID1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	topID2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]

	entity1, _ := c.World.Entities.Get(entity.ID(topID1))
	entity2, _ := c.World.Entities.Get(entity.ID(topID2))

	level := entity1.GetCumulationLevel()

	c.World.Player.ConsumeMana(1)

	// Refactorisation DDD : La logique d'association est déléguée au Domaine (AssocEngine)
	result, err := c.AssocEng.TryAssociate(entity1, entity2)

	if err == nil && result.Success {
		// --- LOGIQUE DE MATCH FINAL (LOOT) ---
		c.World.MatchTile(c.GridID, c.Pos1)
		c.World.MatchTile(c.GridID, c.Pos2)

		if entity1.GetType() == entity.TypeCreature || entity1.GetType() == entity.TypeResource {
			grid.MatchedTargetsCount += 2
			c.World.IsNavigationOpen(c.GridID)
		}

		name := "unknown"
		if r := entity1.GetMatchID(); r != "" {
			name = r
		}

		c.World.EventBus.Publish(event.Event{
			Type:     event.TileMatched,
			SourceID: string(entity1.GetID()),
			Payload: map[string]interface{}{
				"position":    c.Pos1,
				"entity_id":   string(entity1.GetID()),
				"other_id":    string(entity2.GetID()),
				"grid_id":     c.GridID,
				"name":        name,
				"entity_type": entity1.GetType(),
				"assoc_type":  result.Type.String(),
				"level":       level,
			},
		})

		c.World.RemoveEntity(entity1.GetID())
		c.World.RemoveEntity(entity2.GetID())

		if c.OnSuccess != nil {
			c.OnSuccess()
		}
		return nil
	}

	// Échec : Association invalide
	creatureCount := 0
	if entity1.GetType() == entity.TypeCreature {
		creatureCount++
	}
	if entity2.GetType() == entity.TypeCreature {
		creatureCount++
	}

	resourceCount := 0
	if entity1.GetType() == entity.TypeResource {
		resourceCount++
	}
	if entity2.GetType() == entity.TypeResource {
		resourceCount++
	}

	if creatureCount > 0 {
		damage := creatureCount * 10
		fmt.Printf("[COMBAT] Match invalide avec %d créature(s) ! Dégâts : %d\n", creatureCount, damage)
		c.World.Player.TakeDamage(damage, "creature_fail")

		c.World.EventBus.Publish(event.Event{
			Type:     event.PlayerDamaged,
			SourceID: "system",
			Payload: map[string]interface{}{
				"damage": damage,
				"type":   "creature_fail",
				"reason": "invalid_match",
			},
		})
	}

	if resourceCount > 0 {
		manaLoss := resourceCount * 5
		fmt.Printf("[ALCHIMIE] Match invalide avec %d ressource(s) ! Mana : -%d\n", resourceCount, manaLoss)
		c.World.Player.ConsumeMana(manaLoss)

		c.World.EventBus.Publish(event.Event{
			Type:     event.PlayerDamaged,
			SourceID: "system",
			Payload: map[string]interface{}{
				"mana_loss": manaLoss,
				"type":      "resource_fail",
				"reason":    "invalid_match",
			},
		})
	}

	// Recacher les entités avec l'animation de pente (Slope)
	plot1, _ := grid.Get(c.Pos1)
	plot2, _ := grid.Get(c.Pos2)

	_, _ = c.World.FlipTile(c.GridID, c.Pos1, plot1.Tilt.ToFlipDirection())
	_, _ = c.World.FlipTile(c.GridID, c.Pos2, plot2.Tilt.ToFlipDirection())

	c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
		entity.Position(c.Pos1), string(entity1.GetID()), c.GridID, plot1.Tilt.ToFlipDirection()))
	c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
		entity.Position(c.Pos2), string(entity2.GetID()), c.GridID, plot2.Tilt.ToFlipDirection()))

	if c.OnFailure != nil {
		c.OnFailure()
	}

	return errors.New("association échouée")
}

// --- MERGE TILES COMMAND ---

type MergeTilesCommand struct {
	World      *domain.World
	AssocEng   *domain.AssocEngine
	GridID     string
	Pos1, Pos2 board.Position
	OnSuccess  func()
	OnFailure  func()
}

func (c *MergeTilesCommand) CanExecute() bool {
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

	id1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	id2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
	e1, ok1 := c.World.Entities.Get(entity.ID(id1))
	e2, ok2 := c.World.Entities.Get(entity.ID(id2))
	if !ok1 || !ok2 {
		return false
	}

	// Uniquement si les entités sont bien révélées
	if e1.GetState()&entity.Revealed == 0 || e2.GetState()&entity.Revealed == 0 {
		return false
	}

	level := e1.GetCumulationLevel()
	maxLevel := 2
	if level >= maxLevel {
		return false
	}

	// Coût augmenté : 3 * (Niveau + 1)
	cost := 3 * (level + 1)
	if c.World.Player.Stats.Mana < cost {
		return false
	}

	return true
}

func (c *MergeTilesCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible de fusionner ces tuiles (mana insuffisant ou invalide)")
	}

	grid, _ := c.World.GetGrid(c.GridID)
	tile1, _ := grid.Get(c.Pos1)
	tile2, _ := grid.Get(c.Pos2)

	id1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	id2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
	e1, _ := c.World.Entities.Get(entity.ID(id1))
	e2, _ := c.World.Entities.Get(entity.ID(id2))

	level := e1.GetCumulationLevel()
	cost := 3 * (level + 1)
	c.World.Player.ConsumeMana(cost)

	// Refactorisation DDD : On utilise le moteur d'association
	result, err := c.AssocEng.TryAssociate(e1, e2)
	isMatch := (err == nil && result.Success)

	if isMatch {
		fmt.Printf("[ALCHIMIE] Fusion réussie en %v (Niveau %d -> %d)\n", c.Pos1, level, level+1)

		// Mise à jour de l'entité survivante
		e1.SetCumulationLevel(level + 1)
		e1.SetState(entity.Hidden) // Se recache après fusion
		e1.AddTag("cumulated")

		// On retire l'autre entité
		c.World.RemoveEntity(e2.GetID())

		// Navigation : Une fusion compte pour 1 point
		grid.MatchedTargetsCount += 1
		c.World.IsNavigationOpen(c.GridID)

		// Notification de fusion
		c.World.EventBus.Publish(event.NewTileMergedEvent(
			entity.Position(c.Pos1),
			string(e1.GetID()),
			e1.GetMatchID(),
			e1.GetType(),
			level+1,
		))

		// Animation visuelle de retournement (fermeture)
		c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
			entity.Position(c.Pos1),
			string(e1.GetID()),
			c.GridID,
			tile1.Tilt.ToFlipDirection(),
		))

		// NOUVEAU : Après une fusion, on referme TOUTES les tuiles révélées
		// (Maintenu car utile pour le gameplay memory)
		if grid, ok := c.World.GetGrid(c.GridID); ok {
			for pos, plot := range grid.Plots {
				if len(plot.EntitiesID) == 0 {
					continue
				}
				topID := plot.EntitiesID[len(plot.EntitiesID)-1]
				if ent, exists := c.World.Entities.Get(entity.ID(topID)); exists {
					if ent.GetState()&entity.Revealed != 0 {
						// On ferme avec l'animation de pente
						_, _ = c.World.FlipTile(c.GridID, pos, plot.Tilt.ToFlipDirection())
						c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
							entity.Position(pos), string(ent.GetID()), c.GridID, plot.Tilt.ToFlipDirection()))
					}
				}
			}
		}

		if c.OnSuccess != nil {
			c.OnSuccess()
		}
		return nil
	}

	// Échec : Association invalide (même punition que le match)
	damage := 0
	if e1.GetType() == entity.TypeCreature {
		damage += 10
	}
	if e2.GetType() == entity.TypeCreature {
		damage += 10
	}

	if damage > 0 {
		c.World.Player.TakeDamage(damage, "creature_fail")
	} else {
		c.World.Player.ConsumeMana(5)
	}

	// Recache les deux
	p1, _ := grid.Get(c.Pos1)
	p2, _ := grid.Get(c.Pos2)
	_, _ = c.World.FlipTile(c.GridID, c.Pos1, p1.Tilt.ToFlipDirection())
	_, _ = c.World.FlipTile(c.GridID, c.Pos2, p2.Tilt.ToFlipDirection())

	c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
		entity.Position(c.Pos1), id1, c.GridID, p1.Tilt.ToFlipDirection()))
	c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
		entity.Position(c.Pos2), id2, c.GridID, p2.Tilt.ToFlipDirection()))

	if c.OnFailure != nil {
		c.OnFailure()
	}

	return errors.New("fusion échouée")
}

// --- DISCARD TRAP COMMAND ---

type DiscardTrapCommand struct {
	World     *domain.World
	GridID    string
	Position  board.Position
	OnSuccess func()
}

func (c *DiscardTrapCommand) CanExecute() bool {
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return false
	}
	plot, err := grid.Get(c.Position)
	if err != nil || len(plot.EntitiesID) == 0 {
		return false
	}
	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, ok := c.World.Entities.Get(entity.ID(topID))
	if !ok {
		return false
	}
	return ent.GetType() == entity.TypeTrap && ent.GetState()&entity.Revealed != 0
}

func (c *DiscardTrapCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible de défausser ce piège")
	}

	c.World.Player.ConsumeMana(1)

	grid, _ := c.World.GetGrid(c.GridID)
	plot, _ := grid.Get(c.Position)
	topID := plot.EntitiesID[len(plot.EntitiesID)-1]

	c.World.RemoveEntity(entity.ID(topID))

	if c.OnSuccess != nil {
		c.OnSuccess()
	}

	return nil
}

// --- END TURN & SKIP COMMANDS ---

type EndTurnCommand struct {
	World *domain.World
}

func (c *EndTurnCommand) CanExecute() bool {
	return true
}

func (c *EndTurnCommand) Execute() error {
	if c.World.Player.Stats.Sanity > 0 {
		c.World.Player.Stats.Sanity--
	}

	c.World.EventBus.Publish(event.NewTurnEndedEvent(c.World.Turn))
	c.World.EventBus.ProcessQueue()

	return nil
}

// SkipTurnCommand ferme toutes les tuiles non appairées ouvertes (comme les séismes) et termine le tour.
type SkipTurnCommand struct {
	World *domain.World
}

func (c *SkipTurnCommand) CanExecute() bool {
	return true
}

func (c *SkipTurnCommand) Execute() error {
	gridID := c.World.CurrentGridID
	if grid, ok := c.World.GetGrid(gridID); ok {
		for pos, plot := range grid.Plots {
			if len(plot.EntitiesID) == 0 {
				continue
			}

			topID := plot.EntitiesID[len(plot.EntitiesID)-1]
			if ent, exists := c.World.Entities.Get(entity.ID(topID)); exists {
				state := ent.GetState()

				// Cache et anime toutes les tuiles révélées qui ne sont pas validées
				if state&entity.Revealed != 0 && state&entity.Matched == 0 {
					_, _ = c.World.FlipTile(gridID, pos, plot.Tilt.ToFlipDirection())

					c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
						entity.Position(pos),
						string(ent.GetID()),
						gridID,
						plot.Tilt.ToFlipDirection(),
					))
				}
			}
		}
	}

	endTurn := &EndTurnCommand{World: c.World}
	return endTurn.Execute()
}

// --- MANAGEMENT & LOOT COMMANDS ---

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
		return errors.New("grille non trouvée")
	}

	for _, e := range c.World.Entities.GetAllActive() {
		if e.GetGridID() == c.GridID {
			c.World.RemoveEntity(e.GetID())
		}
	}

	grid.InitialMatchableCount = 0
	return nil
}

type UsePortablePortalCommand struct {
	World  *domain.World
	GridID string
	Center board.Position
}

func (c *UsePortablePortalCommand) CanExecute() bool {
	if c.World == nil {
		return false
	}
	if _, ok := c.World.GetGrid(c.GridID); !ok {
		return false
	}
	return c.World.HasPortablePortal()
}

func (c *UsePortablePortalCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible de déployer le portail portable")
	}

	if c.Center.X < 0 || c.Center.Y < 0 {
		_, err := c.World.DeployPortablePortal(c.GridID)
		return err
	}

	_, err := c.World.DeployPortablePortalAt(c.GridID, c.Center)
	return err
}

type UseScannerItemCommand struct {
	World     *domain.World
	GridID    string
	ItemIndex int
}

func (c *UseScannerItemCommand) CanExecute() bool {
	if c.World == nil || c.World.Player == nil {
		return false
	}

	inv := &c.World.Player.Inventory
	if c.ItemIndex < 0 || c.ItemIndex >= len(inv.Items) {
		return false
	}

	item, err := inv.GetItem(c.ItemIndex)
	if err != nil {
		return false
	}

	if item.Name != "echo_hound" || !item.IsUsable {
		return false
	}

	return true
}

func (c *UseScannerItemCommand) Execute() error {
	item, err := c.World.Player.Inventory.GetItem(c.ItemIndex)
	if err != nil {
		return err
	}

	if !c.CanExecute() {
		return errors.New("impossible d'utiliser le scanner (item invalide ou inutilisable)")
	}

	targetGrid := c.GridID
	if targetGrid == "" {
		targetGrid = c.World.CurrentGridID
	}

	err = c.World.TriggerScannerEffect(targetGrid, item.GetCumulationLevel())
	if err != nil {
		return fmt.Errorf("échec de l'effet de scan : %w", err)
	}

	err = c.World.RemoveLootItem(c.ItemIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'objet : %w", err)
	}

	fmt.Printf("[ITEM] Le joueur a utilisé le cri de l'Echo Hound depuis l'emplacement %d\n", c.ItemIndex)
	return nil
}

type UseLootItemCommand struct {
	World     *domain.World
	ItemIndex int
}

func (c *UseLootItemCommand) CanExecute() bool {
	if c.World == nil || c.World.Player == nil {
		return false
	}

	inv := &c.World.Player.Inventory
	if c.ItemIndex < 0 || c.ItemIndex >= len(inv.Items) {
		return false
	}

	item, err := inv.GetItem(c.ItemIndex)
	if err != nil || !item.IsUsable {
		return false
	}

	if ability, exists := ItemAbilities[item.Name]; exists {
		return ability.CanExecute(c.World)
	}

	return false
}

func (c *UseLootItemCommand) Execute() error {
	item, err := c.World.Player.Inventory.GetItem(c.ItemIndex)
	if err != nil {
		return err
	}

	ability, exists := ItemAbilities[item.Name]
	if !exists {
		return fmt.Errorf("aucune capacité définie pour l'objet : %s", item.Name)
	}

	if !ability.CanExecute(c.World) {
		return fmt.Errorf("les conditions d'utilisation pour %s ne sont pas remplies (aucune pile trouvée ?)", item.Name)
	}

	message, errAbility := ability.Execute(c.World, item.GetCumulationLevel())
	if errAbility != nil {
		return errAbility
	}

	err = c.World.RemoveLootItem(c.ItemIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'objet : %w", err)
	}

	if message != "" {
		c.World.EventBus.PublishImmediate(event.NewItemMessageEvent(message))
	}

	fmt.Printf("[ITEM] %s utilisé (Ability activée) depuis l'emplacement %d\n", item.Name, c.ItemIndex)
	return nil
}

type UseDreamberryItemCommand struct {
	World     *domain.World
	ItemIndex int
}

func (c *UseDreamberryItemCommand) CanExecute() bool {
	if c.World == nil || c.World.Player == nil {
		return false
	}

	inv := &c.World.Player.Inventory
	if c.ItemIndex < 0 || c.ItemIndex >= len(inv.Items) {
		return false
	}

	item, err := inv.GetItem(c.ItemIndex)
	if err != nil {
		return false
	}

	if item.Name != "dreamberry" || !item.IsUsable {
		return false
	}

	return true
}

func (c *UseDreamberryItemCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible d'utiliser la dreamberry (item invalide ou inutilisable)")
	}

	const manaRestorationAmount = 5
	c.World.Player.RestoreMana(manaRestorationAmount)

	err := c.World.RemoveLootItem(c.ItemIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de la dreamberry : %w", err)
	}

	fmt.Printf("[ITEM] Le joueur a consommé une Dreamberry (Mana +%d) depuis l'emplacement %d\n", manaRestorationAmount, c.ItemIndex)
	return nil
}

// --- NAVIGATION & ZONE COMMANDS ---

type ClearAllBoardsCommand struct {
	World *domain.World
}

func (c *ClearAllBoardsCommand) CanExecute() bool {
	return true
}

func (c *ClearAllBoardsCommand) Execute() error {
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
		return errors.New("grille non trouvée")
	}
	c.World.SetCurrentGrid(c.GridID)

	grid, _ := c.World.GetGrid(c.GridID)
	playerPos := entity.Position{X: grid.Width / 2, Y: grid.Height / 2}
	c.World.SetPlayerPosition(playerPos)

	return nil
}

type UnlockNavigationCommand struct {
	World  *domain.World
	GridID string
}

func (c *UnlockNavigationCommand) CanExecute() bool {
	_, ok := c.World.GetGrid(c.GridID)
	return ok
}

func (c *UnlockNavigationCommand) Execute() error {
	grid, ok := c.World.GetGrid(c.GridID)
	if !ok {
		return errors.New("grille non trouvée")
	}

	grid.NavigationForcedOpen = true
	fmt.Printf("[DEBUG] Navigation forcée pour la zone %s\n", c.GridID)
	return nil
}

type SwitchZoneCommand struct {
	World     *domain.World
	Direction entity.Direction
}

func (c *SwitchZoneCommand) CanExecute() bool {
	if c.World.DreamPlane == nil {
		return false
	}
	_, ok := c.World.DreamPlane.GetConnectedZone(c.World.CurrentGridID, c.Direction)
	if !ok {
		return false
	}

	return c.World.IsNavigationOpen(c.World.CurrentGridID)
}

func (c *SwitchZoneCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("aucune zone connectée dans cette direction ou conditions non remplies")
	}

	if grid, ok := c.World.GetGrid(c.World.CurrentGridID); ok {
		grid.ExitsState[c.Direction] = [2]entity.TileState{
			entity.Revealed | entity.Matched,
			entity.Revealed | entity.Matched,
		}
	}

	targetID, _ := c.World.DreamPlane.GetConnectedZone(c.World.CurrentGridID, c.Direction)
	c.World.SetCurrentGridFrom(targetID, c.Direction)

	grid, _ := c.World.GetGrid(targetID)
	newPos := entity.Position{X: grid.Width / 2, Y: grid.Height / 2}

	switch c.Direction {
	case entity.DirNorth:
		newPos = entity.Position{X: grid.Width / 2, Y: grid.Height - 1}
	case entity.DirSouth:
		newPos = entity.Position{X: grid.Width / 2, Y: 0}
	case entity.DirEast:
		newPos = entity.Position{X: 0, Y: grid.Height / 2}
	case entity.DirWest:
		newPos = entity.Position{X: grid.Width - 1, Y: grid.Height / 2}
	}
	c.World.SetPlayerPosition(newPos)

	return nil
}

type RotateGridCommand struct {
	World  *domain.World
	GridID string
}

func (c *RotateGridCommand) CanExecute() bool {
	_, ok := c.World.GetGrid(c.GridID)
	return ok
}

func (c *RotateGridCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("grille non trouvée")
	}
	return c.World.RotateGrid(c.GridID)
}

// --- HELPERS ---

func flipToPlayerState(f domain.FlipDirection) (entity.Position, player.BorderPosition) {
	switch f {
	case domain.FlipTop:
		return entity.Position{X: 0, Y: -1}, player.BorderTop
	case domain.FlipTopRight:
		return entity.Position{X: 1, Y: -1}, player.BorderTopRight
	case domain.FlipRight:
		return entity.Position{X: 1, Y: 0}, player.BorderRight
	case domain.FlipBottomRight:
		return entity.Position{X: 1, Y: 1}, player.BorderBottomRight
	case domain.FlipBottom:
		return entity.Position{X: 0, Y: 1}, player.BorderBottom
	case domain.FlipBottomLeft:
		return entity.Position{X: -1, Y: 1}, player.BorderBottomLeft
	case domain.FlipLeft:
		return entity.Position{X: -1, Y: 0}, player.BorderLeft
	case domain.FlipTopLeft:
		return entity.Position{X: -1, Y: -1}, player.BorderTopLeft
	default:
		return entity.Position{X: 0, Y: 0}, player.BorderTop
	}
}
