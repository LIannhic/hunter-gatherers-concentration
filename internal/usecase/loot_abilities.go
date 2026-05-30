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

// LootAbility définit le contrat pour la capacité déclenchée par un objet de butin.
type LootAbility interface {
	CanExecute(world *domain.World) bool
	Execute(world *domain.World) (string, error)
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
}

// --- ABILITY : DREAMBERRY ---

type DreamberryAbility struct{}

func (a *DreamberryAbility) CanExecute(world *domain.World) bool { return true }

func (a *DreamberryAbility) Execute(world *domain.World) (string, error) {
	const healthRestoration = 5
	world.Player.Heal(healthRestoration)
	return fmt.Sprintf("Dreamberry consommée : +%d santé.", healthRestoration), nil
}

// --- ABILITY : MOONSTONE ---

type MoonstoneAbility struct{}

func (a *MoonstoneAbility) CanExecute(world *domain.World) bool { return true }

func (a *MoonstoneAbility) Execute(world *domain.World) (string, error) {
	const sanityRestoration = 5
	world.Player.RestoreSanity(sanityRestoration)
	return fmt.Sprintf("Moonstone consommée : +%d sanité.", sanityRestoration), nil
}

// --- ABILITY : CRYSTAL SHARD ---

type CrystalShardAbility struct{}

func (a *CrystalShardAbility) CanExecute(world *domain.World) bool { return true }

func (a *CrystalShardAbility) Execute(world *domain.World) (string, error) {
	const manaRestoration = 5
	world.Player.RestoreMana(manaRestoration)
	return fmt.Sprintf("Crystal Shard consommée : +%d mana.", manaRestoration), nil
}

// --- ABILITY : WHISPERING HERB ---

type WhisperingHerbAbility struct{}

func (a *WhisperingHerbAbility) CanExecute(world *domain.World) bool { return true }

func (a *WhisperingHerbAbility) Execute(world *domain.World) (string, error) {
	return "Une herbe chuchotante murmure un secret apaisant...", nil
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

func (a *SpecterAbility) Execute(world *domain.World) (string, error) {
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
		return "", errors.New("spectre inutilisable : moins de deux créatures disponibles")
	}
	return "Spectre utilisé : une paire de créatures a disparu du plateau.", nil
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

func (a *BurrowerAbility) Execute(world *domain.World) (string, error) {
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
		creatureEnt.MovementProfile.Perception.LeavesTracks = true
		creatureEnt.MovementProfile.Perception.TrackType = "mud"
		creatureEnt.MovementProfile.Perception.TrackDuration = 3
		marked = true
		break
	}

	if !marked {
		return "", errors.New("burrower inutilisable : aucune créature valide sur la grille")
	}
	return "Burrower activé : une créature laissera bientôt des traces de boue.", nil
}

// --- ABILITY : SHADOWSTALKER ---

type ShadowstalkerAbility struct{}

func (a *ShadowstalkerAbility) CanExecute(world *domain.World) bool { return true }

func (a *ShadowstalkerAbility) Execute(world *domain.World) (string, error) {
	world.Player.ImmunityTurns = 1
	return "Vous vous sentez évanescent.", nil
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

func (a *MossMonkeyAbility) Execute(world *domain.World) (string, error) {
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
		return "", errors.New("singe de mousse inutilisable : aucune case vide sur la grille")
	}
	return fmt.Sprintf("Le Singe de Mousse s'est agité ! %d pièges ont poussé dans les espaces vides.", trapsSpawned), nil
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

func (a *StonewardenAbility) Execute(world *domain.World) (string, error) {
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
       return "", errors.New("gardien de pierre inutilisable : aucune pile de tuiles trouvée")
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

    return fmt.Sprintf("[GARDIEN] Séisme en (%d,%d) ! Dispersion de la pile effectuée.", sourcePos.X, sourcePos.Y), nil
}

// --- ABILITY : FLEEING SPRITE ---

type FleeingSpriteAbility struct{}

func (a *FleeingSpriteAbility) CanExecute(world *domain.World) bool { return true }

func (a *FleeingSpriteAbility) Execute(world *domain.World) (string, error) {
	fmt.Println("[ABILITY] Activation du Fleeing Sprite - ThreatVisionTurns set to 1")
	world.Player.ThreatVisionTurns = 1
	return "L'éclat du Fleeing Sprite révèle les intentions de vos prédateurs.", nil
}
