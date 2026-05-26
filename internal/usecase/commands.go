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
	if !ok {
		return false
	}

	state := ent.GetState()
	// Si c'est un piège révélé, on peut toujours le recacher (flip inverse)
	if ent.GetType() == entity.TypeTrap && state&entity.Revealed != 0 {
		return state&entity.Matched == 0
	}

	// Sinon (caché), ne peut pas révéler une tuile déjà révélée ou appairée
	if state&entity.Revealed != 0 || state&entity.Matched != 0 {
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
		fmt.Println("Le joueur n'est pas sur cette grille")
		fmt.Printf("Joueur sur %s mais a tenté %s\n", c.World.CurrentGridID, c.GridID)
		return errors.New("le joueur n'est pas sur cette grille")
	}

	if !c.CanExecute() {
		return errors.New("impossible de révéler cette tuile")
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

	// --- NOUVEAU : Logique de Confrontation (Zone de Menace) ---
	if cre, ok := ent.(*creature.Creature); ok {
		isThreatened := cre.IsPositionThreatened(playerPos)

		fmt.Printf("[DEBUG] Reveal Créature: %s en %v (Transformation: %s)\n", cre.Species, cre.GetPosition(), cre.GetTransformation().String())
		fmt.Printf("[DEBUG] Joueur en %v (Ancre: %v) | Menacé ? %v\n", playerPos, border, isThreatened)

		activeThreats := cre.GetActiveThreatDirections()
		fmt.Printf("[DEBUG] Zones de menace actives: %v\n", activeThreats)

		if isThreatened {
			fmt.Printf("[COMBAT] Confrontation ! La créature %s attaque le joueur en %v\n", cre.Species, playerPos)
			c.World.Player.TakeDamage(10, "physical")

			// Feedback visuel de l'attaque (demi-cercle rouge)
			track := entity.NewTrack("intent_beam", 2, entity.Position{X: c.Position.X, Y: c.Position.Y}, playerPos)
			track.SetGridID(c.GridID)
			c.World.Entities.Register(track)

			// Publie un événement de dégâts pour l'UI
			c.World.EventBus.Publish(event.Event{
				Type:     event.PlayerDamaged,
				SourceID: string(cre.GetID()),
				Payload: map[string]interface{}{
					"damage": 10,
					"type":   "physical",
					"reason": "confrontation",
				},
			})
		}
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
	if c.World.Player.Stats.Mana <= 0 {
		return false
	}

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

	return true
}

func (c *MatchTilesCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible d'appairer ces tuiles")
	}

	// Consomme 1 point de mana par tentative
	c.World.Player.ConsumeMana(1)

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

	// Cas 3 : Deux pièges - ils s'annulent
	if !isMatch && entity1.GetType() == entity.TypeTrap && entity2.GetType() == entity.TypeTrap {
		isMatch = true
		matchType = "trap_neutralization"
	}

	if isMatch {
		// Succès : les entités sont marquées comme appairées
		c.World.MatchTile(c.GridID, c.Pos1)
		c.World.MatchTile(c.GridID, c.Pos2)

		// --- NOUVEAU : Logique de Match Valide (0 dégât) ---
		// (Déjà implicite car on ne fait rien ici, mais on pourrait loguer)

		// Si ce sont des ressources ou créatures, on incrémente le compteur de paires trouvées pour ouvrir les sorties
		if entity1.GetType() == entity.TypeCreature || entity1.GetType() == entity.TypeResource {
			if grid, ok := c.World.GetGrid(c.GridID); ok {
				grid.MatchedTargetsCount += 2
			}
		}

		// On récupère le nom pour le loot AVANT la suppression
		name := "unknown"
		if r, ok := entity1.(*domain.Resource); ok {
			name = r.ResourceType
		} else if c, ok := entity1.(*domain.Creature); ok {
			name = c.Species
		}

		// Publie l'événement de match (utilisé par le LootSystem)
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
				"assoc_type":  matchType,
			},
		})

		// Note: on les retire du monde
		c.World.RemoveEntity(entity1.GetID())
		c.World.RemoveEntity(entity2.GetID())

		if c.OnSuccess != nil {
			c.OnSuccess()
		}

		return nil
	} else {
		// Échec : Association invalide
		// --- NOUVEAU : Logique de Match Invalide (Dégâts par créature) ---
		creatureCount := 0
		if isCre1 {
			creatureCount++
		}
		if isCre2 {
			creatureCount++
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

		// Recacher les entités avec animation basée sur la pente (Slope)
		grid, _ := c.World.GetGrid(c.GridID)
		plot1, _ := grid.Get(c.Pos1)
		plot2, _ := grid.Get(c.Pos2)

		_, _ = c.World.FlipTile(c.GridID, c.Pos1, plot1.Tilt.ToFlipDirection())
		_, _ = c.World.FlipTile(c.GridID, c.Pos2, plot2.Tilt.ToFlipDirection())

		// Publie les événements pour le renderer (Immediate pour éviter les délais)
		c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
			entity.Position(c.Pos1), string(entity1.GetID()), c.GridID, plot1.Tilt.ToFlipDirection()))
		c.World.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
			entity.Position(c.Pos2), string(entity2.GetID()), c.GridID, plot2.Tilt.ToFlipDirection()))

		if c.OnFailure != nil {
			c.OnFailure()
		}

		return errors.New("association échouée")
	}
}

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
	// On ne peut défausser qu'un piège révélé
	return ent.GetType() == entity.TypeTrap && ent.GetState()&entity.Revealed != 0
}

func (c *DiscardTrapCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible de défausser ce piège")
	}

	// Consomme 1 point de mana (compte comme un tour/match)
	c.World.Player.ConsumeMana(1)

	grid, _ := c.World.GetGrid(c.GridID)
	plot, _ := grid.Get(c.Position)
	topID := plot.EntitiesID[len(plot.EntitiesID)-1]

	// Supprime l'entité du monde (défausse)
	c.World.RemoveEntity(entity.ID(topID))

	// Note: On pourrait émettre un événement spécifique si besoin

	if c.OnSuccess != nil {
		c.OnSuccess()
	}

	return nil
}

type EndTurnCommand struct {
	World *domain.World
}

func (c *EndTurnCommand) CanExecute() bool {
	return true
}

func (c *EndTurnCommand) Execute() error {
	// Consomme 1 point de sanité par tour passé
	if c.World.Player.Stats.Sanity > 0 {
		c.World.Player.Stats.Sanity--
	}

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
		return errors.New("grille non trouvée")
	}

	// Supprime toutes les entités de ce grid
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
	ItemIndex int // L'index de l'objet dans la liste d'inventaire du joueur
}

func (c *UseScannerItemCommand) CanExecute() bool {
	if c.World == nil || c.World.Player == nil {
		return false
	}

	// 1. Vérifie si l'index est valide dans l'inventaire
	inv := &c.World.Player.Inventory
	if c.ItemIndex < 0 || c.ItemIndex >= len(inv.Items) {
		return false
	}

	// 2. Récupère l'objet sans le supprimer pour vérification
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
	if !c.CanExecute() {
		return errors.New("impossible d'utiliser le scanner (item invalide ou inutilisable)")
	}

	// 4. 🛠️ AJUSTEMENT : Utilise c.GridID s'il est fourni, sinon replie-toi sur c.World.CurrentGridID
	targetGrid := c.GridID
	if targetGrid == "" {
		targetGrid = c.World.CurrentGridID
	}

	// Déclenche l'effet de scan dans le monde
	err := c.World.TriggerScannerEffect(targetGrid)
	if err != nil {
		return fmt.Errorf("échec de l'effet de scan : %w", err)
	}

	// Si le scan a fonctionné, on consomme l'objet en le retirant de l'inventaire
	inv := &c.World.Player.Inventory
	err = inv.RemoveItem(c.ItemIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'objet : %w", err)
	}

	fmt.Printf("[ITEM] Le joueur a utilisé le cri de l'Echo Hound depuis l'emplacement %d\\n", c.ItemIndex)

	return nil
}

type UseLootItemCommand struct {
	World     *domain.World
	ItemIndex int // L'index de l'objet dans l'inventaire
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
	if err != nil {
		return false
	}

	if !item.IsUsable {
		return false
	}

	switch item.Name {
	case player.DreamberryItemName,
		player.MoonstoneItemName,
		player.CrystalShardItemName,
		player.WhisperingHerbItemName,
		player.SpecterItemName,
		player.BurrowerItemName:
		return true
	default:
		return false
	}
}

func (c *UseLootItemCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible d'utiliser cet objet de butin")
	}

	inv := &c.World.Player.Inventory
	item, err := inv.GetItem(c.ItemIndex)
	if err != nil {
		return err
	}

	var message string

	switch item.Name {
	case player.DreamberryItemName:
		const healthRestoration = 5
		c.World.Player.Heal(healthRestoration)
		message = fmt.Sprintf("Dreamberry consommée : +%d santé.", healthRestoration)

	case player.MoonstoneItemName:
		const sanityRestoration = 5
		c.World.Player.RestoreSanity(sanityRestoration)
		message = fmt.Sprintf("Moonstone consommée : +%d sanité.", sanityRestoration)

	case player.CrystalShardItemName:
		const manaRestoration = 5
		c.World.Player.RestoreMana(manaRestoration)
		message = fmt.Sprintf("Crystal Shard consommée : +%d mana.", manaRestoration)

	case player.WhisperingHerbItemName:
		message = "Une herbe chuchotante murmure un secret apaisant..."

	case player.SpecterItemName:
		gridID := c.World.CurrentGridID
		creatures := c.World.Entities.GetByType(entity.TypeCreature)
		removed := 0
		for _, e := range creatures {
			if e.GetGridID() != gridID {
				continue
			}
			c.World.RemoveEntity(e.GetID())
			removed++
			if removed >= 2 {
				break
			}
		}
		if removed < 2 {
			return errors.New("spectre inutilisable : moins de deux créatures disponibles")
		}
		message = "Spectre utilisé : une paire de créatures a disparu du plateau."

	case player.BurrowerItemName:
		gridID := c.World.CurrentGridID
		creatures := c.World.Entities.GetByType(entity.TypeCreature)
		marked := false
		for _, e := range creatures {
			if e.GetGridID() != gridID {
				continue
			}
			creatureEnt, ok := e.(*creature.Creature)
			if !ok {
				continue
			}
			if creatureEnt.MovementProfile == nil {
				continue
			}
			creatureEnt.MovementProfile.Perception.LeavesTracks = true
			creatureEnt.MovementProfile.Perception.TrackType = "mud"
			creatureEnt.MovementProfile.Perception.TrackDuration = 3
			marked = true
			break
		}
		if !marked {
			return errors.New("burrower inutilisable : aucune créature sur la grille")
		}
		message = "Burrower activé : une créature laissera bientôt des traces de boue."

	default:
		return errors.New("objet de butin non pris en charge")
	}

	err = inv.RemoveItem(c.ItemIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'objet : %w", err)
	}

	if message != "" {
		c.World.EventBus.PublishImmediate(event.NewItemMessageEvent(message))
	}

	fmt.Printf("[ITEM] %s utilisé depuis l'emplacement %d\n", item.Name, c.ItemIndex)
	return nil
}

// --- COMMANDE DÉDIÉE À LA DREAMBERRY ---
type UseDreamberryItemCommand struct {
	World     *domain.World
	ItemIndex int // L'index de l'objet dans l'inventaire
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

	// La commande ne s'occupe QUE de la dreamberry utilisable
	if item.Name != "dreamberry" || !item.IsUsable {
		return false
	}

	return true
}

func (c *UseDreamberryItemCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("impossible d'utiliser la dreamberry (item invalide ou inutilisable)")
	}

	// 1. Applique l'effet de restauration de mana
	const manaRestorationAmount = 5 // À ajuster selon l'équilibrage
	c.World.Player.RestoreMana(manaRestorationAmount)

	// 2. Consomme l'objet en le retirant de l'inventaire
	inv := &c.World.Player.Inventory
	err := inv.RemoveItem(c.ItemIndex)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de la dreamberry : %w", err)
	}

	fmt.Printf("[ITEM] Le joueur a consommé une Dreamberry (Mana +%d) depuis l'emplacement %d\\n", manaRestorationAmount, c.ItemIndex)
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
		return errors.New("grille non trouvée")
	}
	c.World.SetCurrentGrid(c.GridID)

	// Update player position to center of new grid
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
	fmt.Printf("[DEBUG] Navigation forcée pour la zone %s\\n", c.GridID)
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

	// La navigation n'est possible que si le seuil de complétion est atteint
	return c.World.IsNavigationOpen(c.World.CurrentGridID)
}

func (c *SwitchZoneCommand) Execute() error {
	if !c.CanExecute() {
		return errors.New("aucune zone connectée dans cette direction ou conditions non remplies")
	}

	// Marque les sorties comme révélées et appairées lors de la transition (pour cohérence visuelle)
	if grid, ok := c.World.GetGrid(c.World.CurrentGridID); ok {
		grid.ExitsState[c.Direction] = [2]entity.TileState{
			entity.Revealed | entity.Matched,
			entity.Revealed | entity.Matched,
		}
	}

	targetID, _ := c.World.DreamPlane.GetConnectedZone(c.World.CurrentGridID, c.Direction)
	c.World.SetCurrentGrid(targetID)

	// Positionne le joueur de l'autre côté de la grille (entrée logique)
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

// flipToPlayerState convertit une direction de flip en offset de position et en ancrage pour le joueur.
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
		return entity.Position{X: 0, Y: 0}, player.BorderTop // Fallback
	}
}
