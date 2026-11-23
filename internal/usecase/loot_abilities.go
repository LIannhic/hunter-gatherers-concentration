package usecase

import (
	"errors"
	"fmt"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/creature"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
)

// LootAbility définit le contrat pour la capacité déclenchée par un objet de butin.
type LootAbility interface {
	CanExecute(world *domain.World) bool
	Execute(world *domain.World, level int) (string, error)
}

// applyLootScaling applique la formule (base * 2.2^niveau) pour l'augmentation des effets.
func applyLootScaling(base float64, level int) int {
	return int(math.Round(base * math.Pow(2.2, float64(level))))
}

// ItemAbilities associe le nom d'un objet de l'inventaire à sa capacité en jeu précise.
var ItemAbilities = map[string]LootAbility{
	player.DreamberryItemName:     &DreamberryAbility{},
	player.MoonstoneItemName:      &MoonstoneAbility{},
	player.CrystalShardItemName:   &CrystalShardAbility{},
	player.WhisperingHerbItemName: &WhisperingHerbAbility{},
	player.SpecterItemName:        &SpecterAbility{},
	player.BurrowerItemName:       &BurrowerAbility{},
	player.ShadowstalkerItemName:  &ShadowstalkerAbility{},
	player.MossMonkeyItemName:     &MossMonkeyAbility{},
	player.StonewardenItemName:    &StonewardenAbility{},
	player.FleeingSpriteName:      &FleeingSpriteAbility{},
	player.FlutterwingItemName:    &FlutterwingAbility{},
	player.MossTruffleItemName:    &MossTruffleAbility{},
	player.EchoCrystalItemName:    &EchoCrystalAbility{},
	player.VoidBloomItemName:      &VoidBloomAbility{},
	player.SandCoreItemName:       &SandCoreAbility{},
	player.LumiflyItemName:        &LumiflyAbility{},
}

// --- ABILITY : DREAMBERRY ---

type DreamberryAbility struct{}

func (a *DreamberryAbility) CanExecute(world *domain.World) bool { return true }

func (a *DreamberryAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(5, level)
	world.Player.Heal(amount)
	return fmt.Sprintf("Dreamberry consumed (Lvl.%d): +%d health.", level, amount), nil
}

// --- ABILITY : MOONSTONE ---

type MoonstoneAbility struct{}

func (a *MoonstoneAbility) CanExecute(world *domain.World) bool { return true }

func (a *MoonstoneAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(5, level)
	world.Player.RestoreSanity(amount)
	return fmt.Sprintf("Moonstone consumed (Lvl.%d): +%d sanity.", level, amount), nil
}

// --- ABILITY : CRYSTAL SHARD ---

type CrystalShardAbility struct{}

func (a *CrystalShardAbility) CanExecute(world *domain.World) bool { return true }

func (a *CrystalShardAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(5, level)
	world.Player.RestoreMana(amount)
	return fmt.Sprintf("Crystal Shard consumed (Lvl.%d): +%d mana.", level, amount), nil
}

// --- ABILITY : WHISPERING HERB ---

type WhisperingHerbAbility struct{}

func (a *WhisperingHerbAbility) CanExecute(world *domain.World) bool { return true }

func (a *WhisperingHerbAbility) Execute(world *domain.World, level int) (string, error) {
	return "A whispering herb whispers a soothing secret...", nil
}

// --- ABILITY : SPECTER ---

type SpecterAbility struct{}

func (a *SpecterAbility) CanExecute(world *domain.World) bool {
	gridID := world.CurrentGridID
	creatures := world.Entities.GetByType(entity.TypeCreature)
	count := 0
	for _, e := range creatures {
		if e.GetGridID() == gridID {
			count++
		}
	}
	return count >= 2
}

func (a *SpecterAbility) Execute(world *domain.World, level int) (string, error) {
	gridID := world.CurrentGridID
	creatures := world.Entities.GetByType(entity.TypeCreature)
	removed := 0

	for _, e := range creatures {
		if e.GetGridID() != gridID {
			continue
		}
		world.RemoveEntity(e.GetID())
		removed++
		if removed >= 2 {
			break
		}
	}

	if removed < 2 {
		return "", errors.New("specter unusable: fewer than two creatures available")
	}
	return "Specter used: a pair of creatures vanished from the board.", nil
}

// --- ABILITY : BURROWER ---

type BurrowerAbility struct{}

func (a *BurrowerAbility) CanExecute(world *domain.World) bool {
	gridID := world.CurrentGridID
	creatures := world.Entities.GetByType(entity.TypeCreature)
	for _, e := range creatures {
		if e.GetGridID() == gridID {
			return true
		}
	}
	return false
}

func (a *BurrowerAbility) Execute(world *domain.World, level int) (string, error) {
	gridID := world.CurrentGridID
	creatures := world.Entities.GetByType(entity.TypeCreature)
	marked := false

	for _, e := range creatures {
		if e.GetGridID() != gridID {
			continue
		}
		creatureEnt, ok := e.(*creature.Creature)
		if !ok || creatureEnt.MovementProfile == nil {
			continue
		}

		duration := applyLootScaling(3, level)
		creatureEnt.MovementProfile.Perception.LeavesTracks = true
		creatureEnt.MovementProfile.Perception.TrackType = "mud"
		creatureEnt.MovementProfile.Perception.TrackDuration = duration
		marked = true
		break
	}

	if !marked {
		return "", errors.New("burrower unusable: no valid creature on grid")
	}
	return fmt.Sprintf("Burrower activated (Lvl.%d): a creature will leave mud trails for %d moves.", level, applyLootScaling(3, level)), nil
}

// --- ABILITY : SHADOWSTALKER ---

type ShadowstalkerAbility struct{}

func (a *ShadowstalkerAbility) CanExecute(world *domain.World) bool { return true }

func (a *ShadowstalkerAbility) Execute(world *domain.World, level int) (string, error) {
	turns := applyLootScaling(1, level)
	world.Player.ImmunityTurns = turns
	return fmt.Sprintf("You feel evanescent for %d turns.", turns), nil
}

// --- ABILITY : MOSS MONKEY ---

type MossMonkeyAbility struct{}

func (a *MossMonkeyAbility) CanExecute(world *domain.World) bool {
	gridID := world.CurrentGridID
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return false
	}
	for x := 0; x < grid.Width; x++ {
		for y := 0; y < grid.Height; y++ {
			if plot, err := grid.Get(board.Position{X: x, Y: y}); err == nil && len(plot.EntitiesID) == 0 {
				return true
			}
		}
	}
	return false
}

func (a *MossMonkeyAbility) Execute(world *domain.World, level int) (string, error) {
	gridID := world.CurrentGridID
	grid, _ := world.GetGrid(gridID)

	trapsSpawned := 0
	for x := 0; x < grid.Width; x++ {
		for y := 0; y < grid.Height; y++ {
			pos := board.Position{X: x, Y: y}
			plot, err := grid.Get(pos)
			if err == nil && len(plot.EntitiesID) == 0 {
				trapPos := entity.Position{X: x, Y: y}
				_, errSpawn := world.SpawnTrap(gridID, trapPos)
				if errSpawn == nil {
					trapsSpawned++
				}
			}
		}
	}

	if trapsSpawned == 0 {
		return "", errors.New("moss monkey unusable: no empty square on grid")
	}
	return fmt.Sprintf("The Moss Monkey stirred! %d traps grew in empty spaces.", trapsSpawned), nil
}

// --- ABILITY : STONEWARDEN ---

type StonewardenAbility struct{}

func (a *StonewardenAbility) CanExecute(world *domain.World) bool {
	gridID := world.CurrentGridID
	grid, ok := world.GetGrid(gridID)
	if !ok {
		return false
	}

	for x := 0; x < grid.Width; x++ {
		for y := 0; y < grid.Height; y++ {
			if plot, err := grid.Get(board.Position{X: x, Y: y}); err == nil && len(plot.EntitiesID) > 1 {
				return true
			}
		}
	}
	return false
}

func (a *StonewardenAbility) Execute(world *domain.World, level int) (string, error) {
	gridID := world.CurrentGridID
	grid, _ := world.GetGrid(gridID)

	var targetPlot *board.Plot
	var sourcePos board.Position
	maxHeight := 1

	// 1. Recherche de la pile la plus haute
	for x := 0; x < grid.Width; x++ {
		for y := 0; y < grid.Height; y++ {
			pos := board.Position{X: x, Y: y}
			plot, err := grid.Get(pos)
			if err == nil && len(plot.EntitiesID) > maxHeight {
				maxHeight = len(plot.EntitiesID)
				targetPlot = plot
				sourcePos = pos
			}
		}
	}

	if targetPlot == nil {
		return "", errors.New("stonewarden unusable: no tile stack found")
	}

	directions := []board.Position{
		{X: 0, Y: 1},   // Bas
		{X: 1, Y: 1},   // Bas-Droite
		{X: 1, Y: 0},   // Droite
		{X: 1, Y: -1},  // Haut-Droite
		{X: 0, Y: -1},  // Haut
		{X: -1, Y: -1}, // Haut-Gauche
		{X: -1, Y: 0},  // Gauche
		{X: -1, Y: 1},  // Bas-Gauche
	}

	dirAttempts := 0
	maxAttempts := len(directions) * 5

	// 2. DISPERSION LOGIQUE
	for len(targetPlot.EntitiesID) > 0 && dirAttempts < maxAttempts {
		offset := directions[dirAttempts%len(directions)]
		dirAttempts++

		adjPos := board.Position{X: sourcePos.X + offset.X, Y: sourcePos.Y + offset.Y}
		adjPlot, err := grid.Get(adjPos)
		if err != nil {
			continue
		}

		currentAllowedHeight := 1 + (dirAttempts / len(directions))
		if len(adjPlot.EntitiesID) > currentAllowedHeight {
			continue
		}

		topEntityID := targetPlot.EntitiesID[len(targetPlot.EntitiesID)-1]

		// A. Notification visuelle de départ avec le mode unique "earthquake"
		world.EventBus.PublishImmediate(event.NewCreatureMovedEvent(
			topEntityID,
			entity.Position(sourcePos),
			entity.Position(adjPos),
			"earthquake",
			false,
			false,
		))

		// B. Déplacement immédiat en mémoire vers la case adjacente cible
		targetPlot.EntitiesID = targetPlot.EntitiesID[:len(targetPlot.EntitiesID)-1]
		if ent, exists := world.Entities.Get(entity.ID(topEntityID)); exists {
			ent.SetPosition(entity.Position(adjPos))
			// L'état de l'entité (Revealed / Hidden) n'est PLUS modifié ici.
			// Il reste intact pour préserver la logique du jeu après le vol.
		}
		adjPlot.EntitiesID = append(adjPlot.EntitiesID, topEntityID)
		grid.Plots[adjPos] = adjPlot
	}

	// Sauvegarde de l'état final de la parcelle d'origine
	grid.Plots[sourcePos] = targetPlot

	return fmt.Sprintf("[STONEWARDEN] Earthquake at (%d,%d)! Stack scatter complete.", sourcePos.X, sourcePos.Y), nil
}

// --- ABILITY : FLEEING SPRITE ---

type FleeingSpriteAbility struct{}

func (a *FleeingSpriteAbility) CanExecute(world *domain.World) bool { return true }

func (a *FleeingSpriteAbility) Execute(world *domain.World, level int) (string, error) {
	fmt.Println("[ABILITY] Activation du Fleeing Sprite - ThreatVisionTurns set based on level")
	turns := applyLootScaling(1, level)
	world.Player.ThreatVisionTurns = turns
	return fmt.Sprintf("The Fleeing Sprite's glow reveals predator intentions for %d turns.", turns), nil
}

// --- ABILITY : FLUTTERWING ---

type FlutterwingAbility struct{}

func (a *FlutterwingAbility) CanExecute(world *domain.World) bool { return true }

func (a *FlutterwingAbility) Execute(world *domain.World, level int) (string, error) {
	sanityRestoration := applyLootScaling(10, level)
	turns := applyLootScaling(3, level)
	world.Player.RestoreSanity(sanityRestoration)
	world.Player.GraceTurns = turns
	return fmt.Sprintf("A soothing breeze enfolds you (+%d Sanity). Creatures will hesitate to attack you for %d turns.", sanityRestoration, turns), nil
}

// --- ABILITY : MOSS TRUFFLE ---
type MossTruffleAbility struct{}

func (a *MossTruffleAbility) CanExecute(world *domain.World) bool { return true }
func (a *MossTruffleAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(15, level)
	world.Player.Heal(amount)
	return fmt.Sprintf("Moss truffle consumed (Lvl.%d): a woody flavor regenerates your body (+%d HP).", level, amount), nil
}

// --- ABILITY : ECHO CRYSTAL ---
type EchoCrystalAbility struct{}

func (a *EchoCrystalAbility) CanExecute(world *domain.World) bool { return true }
func (a *EchoCrystalAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(15, level)
	world.Player.RestoreMana(amount)
	return fmt.Sprintf("Echo crystal absorbed (Lvl.%d): resonates with your inner energy (+%d Mana).", level, amount), nil
}

// --- ABILITY : VOID BLOOM ---
type VoidBloomAbility struct{}

func (a *VoidBloomAbility) CanExecute(world *domain.World) bool { return true }
func (a *VoidBloomAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(15, level)
	world.Player.RestoreSanity(amount)
	return fmt.Sprintf("Void bloom consumed (Lvl.%d): an ethereal essence calms your mind (+%d Sanity).", level, amount), nil
}

// --- ABILITY : SAND CORE ---
type SandCoreAbility struct{}

func (a *SandCoreAbility) CanExecute(world *domain.World) bool { return true }
func (a *SandCoreAbility) Execute(world *domain.World, level int) (string, error) {
	amount := applyLootScaling(5, level)
	world.Player.Heal(amount)
	world.Player.RestoreMana(amount)
	world.Player.RestoreSanity(amount)
	return fmt.Sprintf("Sand core absorbed (Lvl.%d): balanced energy (+%d to all stats).", level, amount), nil
}

// --- ABILITY : LUMIFLY ---
type LumiflyAbility struct{}

func (a *LumiflyAbility) CanExecute(world *domain.World) bool {
	if world.CurrentGridID == "" {
		return false
	}
	// Vérifie qu'au moins un lumifly est sur la grille
	for _, e := range world.Entities.GetByType(entity.TypeCreature) {
		if e.GetGridID() == world.CurrentGridID {
			if c, ok := e.(*creature.Creature); ok && c.Species == "lumifly" {
				return true
			}
		}
	}
	return false
}

func (a *LumiflyAbility) Execute(world *domain.World, level int) (string, error) {
	duration := 0.5 + float64(level)*0.25
	radius := (0.5 + float64(level)/6.0) * math.Sqrt(2)

	// Trouve tous les lumifly sur la grille courante
	centers := make([]entity.Position, 0)
	for _, e := range world.Entities.GetByType(entity.TypeCreature) {
		if e.GetGridID() != world.CurrentGridID {
			continue
		}
		if c, ok := e.(*creature.Creature); ok && c.Species == "lumifly" {
			centers = append(centers, e.GetPosition())
		}
	}

	if len(centers) == 0 {
		return "", fmt.Errorf("no lumifly on grid")
	}

	world.TriggerLumiflyEffect(centers, radius, duration)

	return fmt.Sprintf("Lumifly released %d golden wave(s) (Lvl.%d)", len(centers), level), nil
}
