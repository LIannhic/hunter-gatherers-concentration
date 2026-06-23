package input

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/actionbuttons"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/usecase"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Renderer interface {
	GetTileSize() int
	GetGridOffset() (int, int)
	ScreenToGrid(screenX, screenY int, world *domain.World) (board.Position, string, bool)
	ScreenToLocalTile(screenX, screenY int, world *domain.World) (localX, localY int, gridID string, ok bool)
	RenderSelectionHighlight(screen *ebiten.Image, pos board.Position, gridID string, color color.Color, world *domain.World)
	RenderPortalPlacementPreview(screen *ebiten.Image, center board.Position, gridID string, world *domain.World)
	NotifyHover(entityID string, dir entity.FlipDirection)
	DecayHoverStates(activeThisFrame map[string]bool)
	GetTileCenter(pos board.Position, grid *board.Grid) (float64, float64)
}

type Handler struct {
	world       *domain.World
	assocEngine *domain.AssocEngine
	renderer    Renderer

	selectedTile   *board.Position
	selectedGridID string

	portablePortalMode bool

	OnTurnEnd             func()
	OnSpawnEntities       func(gridID string)
	OnSpawnAllCreatures   func(gridID string) // Shift+S: Spawn toutes les créatures
	OnSpawnRandomCreature func(gridID string) // F9: Spawn créature aléatoire
	OnClearBoard          func(gridID string)
	OnSwitchGrid          func(gridID string)
	OnRotateBoard         func(delta float64)                        // Callback pour la rotation du plateau
	OnResetRotation       func()                                     // Callback pour réinitialiser la rotation
	OnExitToMenu          func()                                     // Callback pour retourner au menu
	OnToggleDetails       func()                                     // Callback pour afficher les détails
	OnToggleInvDetails    func()                                     // Callback pour afficher l'inventaire détaillé
	OnToggleAssetsDetails func()                                     // T: Callback pour afficher l'atlas des assets
	OnFillInventory       func()                                     // Callback pour remplir l'inventaire (debug)
	OnRevealAll           func(gridID string)                        // F5: Cheat - révéler tout
	OnHideAll             func(gridID string)                        // F6: Cheat - cacher tout
	OnUnlockNavigation    func(gridID string)                        // F7: Cheat - désceller sorties
	OnClearBlocked        func(gridID string)                        // F8: Cheat - retirer état bloqué
	OnUsePortablePortal   func(gridID string, center board.Position) // P / grid placement: Déployer le portail portable
	OnVictory             func()                                     // Callback déclenché lors de l'activation du portail final
	OnForceTurn           func()                                     // KeySpace: Forcer le prochain tour
	OnTriggerQuake        func(gridID string, clockwise bool, angle float32) // Debug: déclencher l'effet séisme
	OnHoverButton         func(mana, health, sanity int)             // Feedback de coût au survol
	OnLongPress           func(pos board.Position, gridID string)    // Appui long tactile

	// Gestionnaire réactif des boutons d'action
	actionButtons *actionbuttons.Manager

	// Tactile & Gestes
	touchStartTime    time.Time
	touchStartScreenX int
	touchStartScreenY int
	isDragging        bool
	isLongPressFired  bool

	// Gestion du tour de jeu memory
	revealedTiles    []board.Position // Liste des tuiles révélées ce tour
	revealedGridIDs  []string         // GridID associé à chaque tuile révélée (pour cross-zone)
	isProcessing     bool             // Évite les clics pendant l'animation / verrouille la grille quand 2 tuiles sont retournées
	footstepTrackIDs []string         // FIFO des empreintes de pas (max 2 visibles)
	isMovedThisTurn  bool             // true si le joueur a cliqué sur une tuile ce tour

	isTransitioning bool // Bloque les entrées pendant le changement de zone
	transitionTimer int  // Frames restantes pour le blocage
	// Indique qu'un Skip a été demandé et qu'on attend la fin des animations
	skipPending bool

	// Victory timer (V0.2 : Déclenché par le déploiement du portail portable)
	victoryTimer *domain.TurnTimer

	// Direction par laquelle le joueur est entré dans la zone actuelle
	entranceDir entity.Direction
}

func NewHandler(world *domain.World, assocEng *domain.AssocEngine) *Handler {
	h := &Handler{
		world:       world,
		assocEngine: assocEng,
		entranceDir: -1, // Aucune entrée par défaut
	}

	// Bloque l'input pendant les animations (start/end)
	if world != nil && world.EventBus != nil {
		world.EventBus.SubscribeFunc(event.AnimationStarted, func(e event.Event) {
			h.isProcessing = true
		})
		world.EventBus.SubscribeFunc(event.AnimationEnded, func(e event.Event) {
			h.isProcessing = false
			// Si un skip était en attente, on reset le timer maintenant
			if h.skipPending {
				if h.world != nil && h.world.TurnTimer != nil {
					h.world.TurnTimer.Reset()
				}
				h.skipPending = false
			}
		})

		// NOUVEAU : Auto-skip forcé par l'Engine (Expiration du Timer)
		world.EventBus.SubscribeFunc(event.Type("turn_timer_expired"), func(e event.Event) {
			h.ResetTimerSkip()
		})

		// Restauration de la logique de scellage
		world.EventBus.SubscribeFunc(event.GridEntered, func(e event.Event) {
			gridID := e.Payload["grid_id"].(string)
			arrivalDir, ok := e.Payload["arrival_dir"].(entity.Direction)
			if !ok || arrivalDir < 0 {
				h.entranceDir = -1
				return
			}

			// L'entrée est la direction OPPOSÉE
			h.entranceDir = world.DreamPlane.OppositeDirection(arrivalDir)

			// Crée une empreinte de pas à la position d'arrivée
			h.spawnFootstepAtArrival(arrivalDir)

			// Si la zone n'est pas "ouverte", on lance l'animation de scellage (Révélé -> Caché)
			if !world.IsNavigationOpen(gridID) {
				fmt.Printf("[NAVIGATION] Scellage de l'entrée %v...\n", h.entranceDir)
				h.triggerSealingAnimation(gridID, h.entranceDir, true)
			}
		})

		world.EventBus.SubscribeFunc(event.NavigationOpened, func(e event.Event) {
			gridID := e.Payload["grid_id"].(string)
			grid, _ := h.world.GetGrid(gridID)
			if grid == nil {
				return
			}

			// On déscelle les sorties qui sont dans un état "ouvert" (Matched ou Revealed)
			for d := entity.DirNorth; d <= entity.DirWest; d++ {
				states := grid.ExitsState[d]
				// Si une des tuiles de la sortie est déjà "active" (pas cachée et bloquée), on anime
				if states[0]&entity.Matched != 0 || states[0]&entity.Revealed != 0 {
					h.triggerSealingAnimation(gridID, d, false)
				}
			}
		})

		world.EventBus.SubscribeFunc(event.NavigationClosed, func(e event.Event) {
			gridID := e.Payload["grid_id"].(string)
			grid, _ := h.world.GetGrid(gridID)
			if grid == nil {
				return
			}

			// On scelle les sorties qui étaient "ouvertes"
			for d := entity.DirNorth; d <= entity.DirWest; d++ {
				states := grid.ExitsState[d]
				if states[0]&entity.Matched != 0 || states[0]&entity.Revealed != 0 {
					h.triggerSealingAnimation(gridID, d, true)
				}
			}
		})
	}
	return h
}

func (h *Handler) SetRenderer(r Renderer) {
	h.renderer = r
}

func (h *Handler) SetActionButtonsManager(m *actionbuttons.Manager) {
	h.actionButtons = m
}

func (h *Handler) Update() error {
	if h.isTransitioning {
		h.transitionTimer--
		if h.transitionTimer <= 0 {
			h.isTransitioning = false
		}
		return nil
	}

	// Mise à jour du victory timer
	if h.victoryTimer != nil && h.victoryTimer.Running {
		if h.victoryTimer.Update(1.0 / 60.0) {
			if h.OnVictory != nil {
				h.OnVictory()
			}
			h.victoryTimer = nil // Stop after trigger
		}
	}

	// Feedback de coût au survol des boutons
	h.updateButtonHover()

	// Gestion du survol (Hover) pour les animations avancées
	h.updateHover()

	if err := h.handleMouse(); err != nil {
		return err
	}
	h.handleKeyboard()
	return nil
}

func (h *Handler) getInteractionPosition() (int, int) {
	tids := ebiten.TouchIDs()
	if len(tids) > 0 {
		return ebiten.TouchPosition(tids[0])
	}
	return ebiten.CursorPosition()
}

func (h *Handler) updateButtonHover() {
	if h.actionButtons == nil || h.OnHoverButton == nil {
		return
	}

    x, y := ebiten.CursorPosition()
    // SÉCURITÉ MOBILE : Pas de survol si la souris n'est pas détectée
    if x == -1 && y == -1 {
        return
    }

	states := h.actionButtons.ComputeStates()
	btnID, ok := h.actionButtons.HitTest(x, y, states)

	if !ok {
		h.OnHoverButton(0, 0, 0)
		return
	}

	mana, hp, sanity := 0, 0, 0

	// Calcul du coût potentiel basé sur l'état actuel (2 tuiles révélées)
	if len(h.revealedTiles) == 2 {
		grid, _ := h.world.GetGrid(h.GetCurrentGridID())
		if grid != nil {
			tile1, _ := grid.Get(h.revealedTiles[0])
			tile2, _ := grid.Get(h.revealedTiles[1])
			if tile1 != nil && tile2 != nil && len(tile1.EntitiesID) > 0 && len(tile2.EntitiesID) > 0 {
				id1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
				id2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
				e1, _ := h.world.Entities.Get(entity.ID(id1))
				e2, _ := h.world.Entities.Get(entity.ID(id2))

				if e1 != nil && e2 != nil {
					level := e1.GetCumulationLevel()

					switch btnID {
					case actionbuttons.BtnMatch:
						mana = 1
					case actionbuttons.BtnMerge:
						mana = 3 * (level + 1)
					case actionbuttons.BtnSkip, actionbuttons.BtnEndTurn:
						// Estimation du risque maximum (les deux tuiles sont des créatures ou ressources)
						hp = 20     // 2 créatures x 10
						mana = 10   // 2 ressources x 5
					}
				}
			}
		}
	} else if btnID == actionbuttons.BtnEndTurn {
		// Pas de tuiles révélées, mais End Turn consomme toujours 1 SN par défaut
		sanity = 1
	}

	h.OnHoverButton(mana, hp, sanity)
}

func (h *Handler) updateHover() {
	if h.renderer == nil || h.world == nil {
		return
	}

	activeThisFrame := make(map[string]bool)
	mx, my := h.getInteractionPosition()

	// SÉCURITÉ MOBILE : Pas de survol si aucun doigt/souris n'est actif
    if mx == -1 && my == -1 {
        h.renderer.DecayHoverStates(activeThisFrame)
        return
    }

	pos, gridID, ok := h.renderer.ScreenToGrid(mx, my, h.world)

	if ok {
		hoverable := h.world.GetHoverableAt(gridID, pos)
		if hoverable != nil && hoverable.IsHoverAllowed() {
			id := hoverable.GetHoverID()
			activeThisFrame[id] = true

			localX, localY, _, _ := h.renderer.ScreenToLocalTile(mx, my, h.world)
			tileSize := h.renderer.GetTileSize()
			// Pour les sorties ou l'inventaire, on utilise une taille fixe de 88 si nécessaire
			if gridID == board.InventoryGridID || strings.HasPrefix(gridID, "exit_") {
				tileSize = int(math.Round(ui.TileSize))
			}
			dir := entity.CalculateFlipDirection(tileSize, localX, localY)
			h.renderer.NotifyHover(id, dir)
		}
	}

	h.renderer.DecayHoverStates(activeThisFrame)
}

func (h *Handler) calculateExitFlipDirection(px, py float64, dir entity.Direction) entity.FlipDirection {
	// Détermine la direction de flip vers l'extérieur du plateau ou vers le curseur
	return entity.CalculateFlipDirection(int(88), int(math.Mod(px, 88)), int(math.Mod(py, 88)))
}

func (h *Handler) Draw(screen *ebiten.Image) {
	if h.renderer == nil {
		return
	}
	h.renderHighlights(screen)
}

// getEntityInfo retourne une description texte de l'entité pour la console
func (h *Handler) getEntityInfo(ent entity.Entity) string {
	if ent == nil {
		return "Vide"
	}
	switch e := ent.(type) {
	case *domain.Creature:
		return fmt.Sprintf("Créature:%s", e.Species)
	case *domain.Resource:
		return fmt.Sprintf("Ressource:%s", e.ResourceType)
	default:
		switch ent.GetType() {
		case entity.TypeTrap:
			return "Piège"
		case entity.TypeTrack:
			return "Trace"
		case entity.TypeStructure:
			return "Structure"
		case entity.TypeArtefact:
			return "Artefact"
		case entity.TypeLoot:
			return "Butin"
		default:
			return ent.GetType().String()
		}
	}
}

func (h *Handler) handleMouse() error {
    // Clic droit : Désélection (Maintenu pour Desktop)
    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
       if h.selectedTile != nil {
          fmt.Printf("[SÉLECTION] Tuile en %v désélectionnée (clic droit)\n", *h.selectedTile)
          h.ClearSelection()
       }
       return nil
    }

    // 1. Détection du début (Appui)
    justPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
    var tids []ebiten.TouchID
    tids = inpututil.AppendJustPressedTouchIDs(tids)
    if len(tids) > 0 {
       justPressed = true
    }

    if justPressed {
       h.touchStartTime = time.Now()
       h.isLongPressFired = false
       h.isDragging = false
       h.touchStartScreenX, h.touchStartScreenY = h.getInteractionPosition()
       return nil
    }

    // 2. Pendant la pression (Drag, Long Press)
    isDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.TouchIDs()) > 0
    if isDown {
       currX, currY := h.getInteractionPosition()

       // Si on est sur mobile et qu'on récupère (-1, -1) alors que c'est pressé,
       // on ignore cette frame pour éviter de casser les calculs
       if currX == -1 && currY == -1 {
          return nil
       }

       // Détection du Drag
       dist := math.Hypot(float64(currX-h.touchStartScreenX), float64(currY-h.touchStartScreenY))
       if dist > 15.0 { // Augmenté légèrement à 15 pour la sensibilité mobile
          h.isDragging = true
       }

       if h.isDragging {
          // Défilement de l'inventaire
          mx, my := h.getInteractionPosition()
          _, gridID, ok := h.renderer.ScreenToGrid(mx, my, h.world)
          if ok && gridID == board.InventoryGridID {
             dy := float64(h.touchStartScreenY - currY)
             h.world.Player.Inventory.ScrollOffset += dy
             h.touchStartScreenY = currY // Update pour le prochain delta

             // Bornage du scroll
             inv := &h.world.Player.Inventory
             totalRows := float64((inv.MaxSize + ui.LootSlotsPerRow - 1) / ui.LootSlotsPerRow)
             rowH := ui.LootSlotSize + ui.LootSlotPadding
             totalHeight := totalRows * rowH
             viewportHeight := 331.0
             maxScroll := totalHeight - viewportHeight
             if maxScroll < 0 {
                maxScroll = 0
             }
             if inv.ScrollOffset < 0 {
                inv.ScrollOffset = 0
             }
             if inv.ScrollOffset > maxScroll {
                inv.ScrollOffset = maxScroll
             }
          }
       } else if !h.isLongPressFired {
          // Détection de l'appui long (500ms)
          if time.Since(h.touchStartTime) > 500*time.Millisecond {
             h.isLongPressFired = true
             h.handleLongPress()
          }
       }
       return nil
    }

    // 3. Relâchement (Action finale)
    var rtids []ebiten.TouchID
    rtids = inpututil.AppendJustReleasedTouchIDs(rtids)
    justReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) || len(rtids) > 0

    if justReleased {
       if !h.isDragging && !h.isLongPressFired {
          // SUR MOBILE : Au relâchement, getInteractionPosition() renvoie (-1, -1).
          // On utilise donc touchStartScreenX/Y qui contiennent la position initiale du clic valide !
          execX, execY := h.getInteractionPosition()
          if execX == -1 && execY == -1 {
             execX = h.touchStartScreenX
             execY = h.touchStartScreenY
          }
          return h.executePrimaryActionAt(execX, execY)
       }
       h.isDragging = false
    }

    return nil
}

func (h *Handler) handleLongPress() {
	x, y := h.getInteractionPosition()
	pos, gridID, ok := h.renderer.ScreenToGrid(x, y, h.world)
	if !ok {
		return
	}

	if h.OnLongPress != nil {
		h.OnLongPress(pos, gridID)
	}
}

func (h *Handler) executePrimaryActionAt(x, y int) error {
    // SÉCURITÉ MOBILE : Si les coordonnées reçues suite au relâchement du doigt sont invalides
    if x == -1 && y == -1 {
        x = h.touchStartScreenX
        y = h.touchStartScreenY
    }

// Priorité : gestion des clics sur les boutons d'action (même si isProcessing)
	if h.actionButtons != nil {
	   states := h.actionButtons.ComputeStates()
	   if btnID, ok := h.actionButtons.HitTest(x, y, states); ok {
		  h.handleActionButtonClick(btnID)
		  return nil
	   }
	}

	// Priorité : mode portail portable (même si isProcessing)
	if h.portablePortalMode && h.OnUsePortablePortal != nil {
		if pos, gridID, ok := h.renderer.ScreenToGrid(x, y, h.world); ok {
			grid, _ := h.world.GetGrid(gridID)
			if grid != nil && h.isValidPortalPreviewPosition(grid, pos) {
				h.OnUsePortablePortal(gridID, pos)
				return nil
			}
		}
	}

	if h.isProcessing {
		// EXCEPTION : On autorise le clic sur un portail déjà révélé pour la victoire
		// (sinon le verrouillage du mode memory bloque la fin de partie)
		if pos, gridID, ok := h.renderer.ScreenToGrid(x, y, h.world); ok {
			grid, _ := h.world.GetGrid(gridID)
			if grid != nil {
				if plot, err := grid.Get(pos); err == nil && len(plot.EntitiesID) > 0 {
					topID := plot.EntitiesID[len(plot.EntitiesID)-1]
					if ent, ok := h.world.Entities.Get(entity.ID(topID)); ok {
						if ent.GetState()&entity.Revealed != 0 {
							isFinish := ent.HasTag("finish_portal")
							isPortable := ent.HasTag("portable_portal")
							if isFinish || isPortable {
								// On traite le clic ici car il est bloqué plus bas
								h.handlePortalClick(ent, gridID, pos)
								return nil
							}
						}
					}
				}
			}
		}

		fmt.Println("[INPUT] Traitement en cours, veuillez patienter...")
		return nil
	}

	// Gestion des clics sur les sorties (navigation zone par zone)
	if dir, index, ok := h.checkExitClickFromCoords(x, y); ok {
		// Vérifie si la sortie existe
		if _, hasExit := h.world.DreamPlane.GetConnectedZone(h.world.CurrentGridID, dir); !hasExit {
			return nil
		}

		// Vérifie si la navigation est ouverte
		if !h.world.IsNavigationOpen(h.world.CurrentGridID) {
			fmt.Println("[NAVIGATION] Sortie scellée. Trouvez plus de paires !")
			return nil
		}

		grid, _ := h.world.GetGrid(h.world.CurrentGridID)

		// État actuel des tuiles de cette sortie
		states := grid.ExitsState[dir]
		currentState := states[index]

		// 1. Si la sortie est déjà appairée, on change de zone
		if currentState&entity.Matched != 0 {
			cmd := &usecase.SwitchZoneCommand{World: h.world, Direction: dir}
			if err := cmd.Execute(); err == nil {
				fmt.Printf("[NAVIGATION] Passage vers le %v -> %s\n", dir, h.world.CurrentGridID)
				h.ClearSelection()
				h.isTransitioning = true
				h.transitionTimer = 30
			}
			return nil
		}

		// 2. Sinon, on révèle la tuile si elle est cachée
		if currentState&entity.Hidden != 0 {
			px := float64(x) - ui.PlaymatX
			py := float64(y) - ui.PlaymatY
			flipDir := h.calculateExitFlipDirection(px, py, dir)

			// Révélation de la première moitié
			states[index] = (currentState & ^entity.Hidden) | entity.Revealed
			fmt.Printf("[NAVIGATION] Tuile de sortie %v #%d révélée\n", dir, index)
			grid.ExitsState[dir] = states

			// Publie l'événement pour l'animation
			h.world.EventBus.Publish(event.Event{
				Type:     event.TileRevealed,
				SourceID: "player",
				Payload: map[string]interface{}{
					"position":       entity.Position{}, // Position virtuelle
					"entity_id":      fmt.Sprintf("exit_%s_%d", board.DirectionToName(dir), index),
					"grid_id":        h.world.CurrentGridID,
					"flip_direction": flipDir,
					"reason":         "player_action",
				},
			})

			// 3. Vérification de l'association via le DOMAINE
			otherIndex := 1 - index
			if states[otherIndex]&entity.Revealed != 0 {
				dirName := "north"
				switch dir {
				case board.South:
					dirName = "south"
				case board.East:
					dirName = "east"
				case board.West:
					dirName = "west"
				}

				// On crée des wrappers Matchable pour le moteur d'association
				m1 := &exitMatchable{id: fmt.Sprintf("exit_%s_%d", dirName, index)}
				m2 := &exitMatchable{id: fmt.Sprintf("exit_%s_%d", dirName, otherIndex)}

				result, err := h.assocEngine.TryAssociate(m1, m2)
				if err == nil && result.Success {
					states[0] |= entity.Matched
					states[1] |= entity.Matched
					grid.ExitsState[dir] = states
					fmt.Printf("[NAVIGATION] %s (Moteur Domaine)\n", result.Message)
				}
			}
		}
		return nil
	}

	pos, gridID, ok := h.renderer.ScreenToGrid(x, y, h.world)
	if !ok {
		return nil
	}

	grid, _ := h.world.GetGrid(gridID)
	plot, err := grid.Get(pos)
	if err != nil || len(plot.EntitiesID) == 0 {
		return nil
	}

	topID := plot.EntitiesID[len(plot.EntitiesID)-1]
	ent, hasEntity := h.world.Entities.Get(entity.ID(topID))
	if !hasEntity {
		return nil
	}

	state := ent.GetState()
	if state&entity.Hidden != 0 {
		// Vérifie si on a déjà révélé 2 tuiles ce tour
		if len(h.revealedTiles) >= 2 {
			fmt.Println("[INPUT] Déjà 2 tuiles révélées ce tour. Veuillez attendre la fin du traitement.")
			return nil
		}

		// Ne permet pas de révéler une tuile bloquée
		if state&entity.Blocked != 0 {
			fmt.Println("[INPUT] Cette tuile est scellée ou bloquée.")
			return nil
		}

		// Calcule la direction de flip basée sur la position du clic dans la tuile
		flipDir := h.calculateFlipDirectionAt(x, y, gridID)

		cmd := &usecase.RevealTileCommand{
			World:         h.world,
			GridID:        gridID,
			Position:      pos,
			FlipDirection: flipDir,
		}
		if err := cmd.Execute(); err == nil {
			h.isMovedThisTurn = true
			fmt.Printf("[INPUT] Clic en %v sur %s. Position logique du joueur : %v\n", pos, gridID, h.world.GetPlayerPosition())
			info := h.getEntityInfo(ent)
			num := len(h.revealedTiles) + 1
			fmt.Printf("[SÉLECTION] Choix #%d : Tuile révélée en %v sur %s -> %s\n", num, pos, gridID, info)
			h.revealedTiles = append(h.revealedTiles, pos)
			h.revealedGridIDs = append(h.revealedGridIDs, gridID)
			// Action volontaire : reset du compte à rebours temps réel
			if h.world.TurnTimer != nil {
				h.world.TurnTimer.Reset()
			}

			// Crée une empreinte de pas sur le bord extérieur de la tuile
			h.spawnFootstepTrack(x, y, pos, gridID)
		}

		// On met à jour le gridID pour la résolution du match
		h.selectedGridID = gridID

		// On sélectionne la tuile pour le match
		h.selectedTile = &pos

		// Si on a révélé 2 tuiles, verrouille la grille et active les boutons Match/Skip
		if len(h.revealedTiles) == 2 {
			h.isProcessing = true
			fmt.Println("[MATCH] 2 tuiles révélées. Grille verrouillée. Choisissez Match ou Skip.")
		}

	} else if state&entity.Revealed != 0 {
		// On délègue à une méthode dédiée pour gérer la victoire et les logs
		if h.handlePortalClick(ent, gridID, pos) {
			return nil
		}

		if ent.GetType() == entity.TypeTrap {
			// NOUVELLE LOGIQUE : Si c'est le seul révélé, on peut s'en débarrasser (Match)
			if len(h.revealedTiles) == 1 && h.revealedTiles[0] == pos {
				fmt.Printf("[ACTION] Défausse du piège en %v (Compte comme un match)\n", pos)
				cmd := &usecase.DiscardTrapCommand{
					World:    h.world,
					GridID:   gridID,
					Position: pos,
					OnSuccess: func() {
					h.revealedTiles = nil
					h.revealedGridIDs = nil
					h.isProcessing = false
					h.ClearSelection()
					h.endTurn()
				},
			}
			if err := cmd.Execute(); err != nil {
				fmt.Printf("[ACTION] Erreur défausse : %v\n", err)
			}
				return nil
			}

			// Sinon (si 0 ou 2 déjà révélés, ou clic sur un autre), flip normal pour recacher
			flipDir := h.calculateFlipDirectionAt(x, y, gridID)
			cmd := &usecase.RevealTileCommand{
				World:         h.world,
				GridID:        gridID,
				Position:      pos,
				FlipDirection: flipDir,
			}
			if err := cmd.Execute(); err == nil {
				fmt.Printf("[ACTION] Piège en %v recaché\n", pos)
				for i, p := range h.revealedTiles {
					if p == pos {
						h.revealedTiles = append(h.revealedTiles[:i], h.revealedTiles[i+1:]...)
						if i < len(h.revealedGridIDs) {
							h.revealedGridIDs = append(h.revealedGridIDs[:i], h.revealedGridIDs[i+1:]...)
						}
						break
					}
				}
				if h.selectedTile != nil && *h.selectedTile == pos {
					h.ClearSelection()
				}
			}
			return nil
		}

		if h.selectedTile != nil && h.selectedGridID == gridID && *h.selectedTile == pos {
			// Déjà sélectionnée : on ne fait rien (le clic gauche ne peut pas désélectionner)
			info := h.getEntityInfo(ent)
			fmt.Printf("[SÉLECTION] Tuile en %v déjà sélectionnée : %s\n", pos, info)
		} else {
			info := h.getEntityInfo(ent)
			fmt.Printf("[SÉLECTION] Tuile en %v sur %s sélectionnée : %s\n", pos, gridID, info)
			h.selectedTile = &pos
			h.selectedGridID = gridID

			// Réinitialise le timer si on clique sur une structure révélée
			// (Dolmen, Obélisque, Portail)
			if ent.GetType() == entity.TypeStructure && h.world.TurnTimer != nil {
				fmt.Println("[ACTION] Structure activée : Timer réinitialisé")
				h.world.TurnTimer.Reset()
			}
		}
	}
	return nil
}

func (h *Handler) checkExitClickFromCoords(x, y int) (entity.Direction, int, bool) {
	// Coordonnées relatives au Playmat
	px := float64(x) - ui.PlaymatX
	py := float64(y) - ui.PlaymatY

	return h.checkExitClick(px, py)
}

func (h *Handler) calculateFlipDirectionAt(x, y int, gridID string) domain.FlipDirection {
	if h.renderer == nil {
		return usecase.DefaultFlipDirection
	}

	// Récupère la position locale du clic dans la tuile
	localX, localY, gID, ok := h.renderer.ScreenToLocalTile(x, y, h.world)
	if !ok || gID != gridID {
		return usecase.DefaultFlipDirection
	}

	tileSize := h.renderer.GetTileSize()
	return entity.CalculateFlipDirection(tileSize, localX, localY)
}

// handleActionButtonClick traite les clics sur les boutons d'action du Playmat.
func (h *Handler) handleActionButtonClick(btnID actionbuttons.ButtonID) {
	switch btnID {
	case actionbuttons.BtnMatch:
		fmt.Println("[ACTION] Bouton Match activé")
		h.processMatchAttempt()
		if h.world.TurnTimer != nil {
			h.world.TurnTimer.Reset()
		}
	case actionbuttons.BtnSkip:
		fmt.Println("[ACTION] Bouton Skip activé")
		h.skipPending = true
		h.isProcessing = true
		h.processSkip()
		// Si aucune animation n'est lancée immédiatement, on reset tout de suite
		if h.world.TurnTimer != nil {
			if len(h.world.Components.QueryByComponent("moving_animation")) == 0 {
				h.world.TurnTimer.Reset()
				h.skipPending = false
				h.isProcessing = false
			}
		}
	case actionbuttons.BtnEndTurn:
		fmt.Println("[ACTION] Bouton End Turn activé")
		// Si le portail est déployé, ce bouton agit comme un instant victory
		if h.victoryTimer != nil && h.victoryTimer.Running {
			if h.OnVictory != nil {
				h.OnVictory()
			}
			h.victoryTimer = nil
			return
		}

		h.skipPending = true
		h.isProcessing = true
		h.processSkip()

		// Si aucune animation n'est lancée immédiatement, on reset tout de suite
		if h.world.TurnTimer != nil {
			if len(h.world.Components.QueryByComponent("moving_animation")) == 0 {
				h.world.TurnTimer.Reset()
				h.skipPending = false
				h.isProcessing = false
			}
		}
	case actionbuttons.BtnMerge:
		fmt.Println("[ACTION] Bouton Merge activé")
		h.processMergeAttempt()
		if h.world.TurnTimer != nil {
			h.world.TurnTimer.Reset()
		}
	}
}

// endTurn synchronise l'état du joueur sur le plateau puis termine le tour.
func (h *Handler) endTurn() {
	h.world.SetPlayerOnBoard(h.isMovedThisTurn)
	if h.OnTurnEnd != nil {
		h.OnTurnEnd()
	}
	h.isMovedThisTurn = false
}

// processSkip recache les tuiles révélées et termine le tour.
func (h *Handler) processSkip() {
	// Si rien à cacher, on se contente d'appeler endTurn (le reset du timer est géré ailleurs)
	if len(h.revealedTiles) == 0 {
		h.endTurn()
		return
	}

	// --- NOUVEAU : Logique de Skip (Pénalité si Match Valide manqué) ---
	if len(h.revealedTiles) == 2 {
		pos1 := h.revealedTiles[0]
		pos2 := h.revealedTiles[1]
		gridID1 := h.revealedGridIDs[0]
		gridID2 := h.revealedGridIDs[1]
		if gridID1 == "" {
			gridID1 = h.world.CurrentGridID
		}
		if gridID2 == "" {
			gridID2 = h.world.CurrentGridID
		}

		grid1, _ := h.world.GetGrid(gridID1)
		grid2, _ := h.world.GetGrid(gridID2)
		if grid1 != nil && grid2 != nil {
			tile1, _ := grid1.Get(pos1)
			tile2, _ := grid2.Get(pos2)

			if len(tile1.EntitiesID) > 0 && len(tile2.EntitiesID) > 0 {
				id1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
				id2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
				e1, _ := h.world.Entities.Get(entity.ID(id1))
				e2, _ := h.world.Entities.Get(entity.ID(id2))

				if e1 != nil && e2 != nil {
					// On vérifie si c'était un match valide
					isMatch := false
					res1, isRes1 := e1.(*domain.Resource)
					res2, isRes2 := e2.(*domain.Resource)
					cre1, isCre1 := e1.(*domain.Creature)
					cre2, isCre2 := e2.(*domain.Creature)

					if isRes1 && isRes2 {
						res, err := h.assocEngine.TryAssociate(res1, res2)
						if err == nil && res.Success {
							isMatch = true
						}
					} else if isCre1 && isCre2 && cre1.Species == cre2.Species {
						isMatch = true
					}

					if isMatch {
						creatureCount := 0
						if isCre1 {
							creatureCount++
						}
						if isCre2 {
							creatureCount++
						}

						resourceCount := 0
						if isRes1 {
							resourceCount++
						}
						if isRes2 {
							resourceCount++
						}

						if creatureCount > 0 {
							fmt.Printf("[DEBUG] SKIP d'une paire valide contenant des créatures (Pos: %v, %v)\n", pos1, pos2)
							damage := creatureCount * 10
							fmt.Printf("[COMBAT] Skip d'un match VALIDE avec %d créature(s) ! Dégâts : %d\n", creatureCount, damage)
							h.world.Player.TakeDamage(damage, "creature_fail")

							h.world.EventBus.Publish(event.NewPlayerDamagedEvent(
								"system",
								damage,
								"creature_fail",
								"skipped_valid_match",
							))
						}

						if resourceCount > 0 {
							manaLoss := resourceCount * 5
							fmt.Printf("[ALCHIMIE] Skip d'un match VALIDE avec %d ressource(s) ! Mana : -%d\n", resourceCount, manaLoss)
							h.world.Player.ConsumeMana(manaLoss)

							h.world.EventBus.Publish(event.NewPlayerDamagedEvent(
								"system",
								manaLoss,
								"resource_fail",
								"skipped_valid_match",
							))
						}
					}
				}
			}
		}
	}

	// On cache les tuiles révélées puis déclenche la fin de tour.
	h.hideAllTilesInGrid()
	h.ClearSelection()

	h.endTurn()
}

// hideRevealedTiles remet l'état Hidden sur toutes les tuiles de revealedTiles (toute la pile).
func (h *Handler) hideRevealedTiles() {
	for i, pos := range h.revealedTiles {
		gridID := h.revealedGridIDs[i]
		if gridID == "" {
			gridID = h.world.CurrentGridID
		}
		grid, ok := h.world.GetGrid(gridID)
		if !ok {
			continue
		}

		plot, err := grid.Get(pos)
		if err != nil || len(plot.EntitiesID) == 0 {
			continue
		}

		// On cache TOUTE la pile d'entités sur cette parcelle
		for _, id := range plot.EntitiesID {
			if ent, ok := h.world.Entities.Get(entity.ID(id)); ok {
				if ent.GetState()&entity.Revealed != 0 {
					flipDir := plot.Tilt.ToFlipDirection()
					// On ne flip réellement (animation) que l'entité au sommet pour éviter le chaos visuel,
					// mais on change l'état logique de toutes les entités de la pile.
					topID := plot.EntitiesID[len(plot.EntitiesID)-1]

					if id == topID {
						_, _ = h.world.FlipTile(gridID, pos, flipDir, "system_hide")
						h.world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
							entity.Position(pos), id, gridID, flipDir,
							map[string]interface{}{"reason": "system_hide"}))
					} else {
						ent.SetState(entity.Hidden)
					}
				}
			}
		}
	}
	h.revealedTiles = nil
	h.revealedGridIDs = nil
}

// hideAllTilesInGrid parcourt toute la grille et passe TOUTES les entités de TOUTES les piles en état Hidden.
func (h *Handler) hideAllTilesInGrid() {
	// 1. D'abord on cache spécifiquement les tuiles sélectionnées (peut traverser plusieurs grilles)
	h.hideRevealedTiles()

	// 2. Puis on cache le reste de la grille actuelle (comportement de sécurité standard)
	gridID := h.world.CurrentGridID
	grid, ok := h.world.GetGrid(gridID)
	if !ok {
		return
	}

	for _, plot := range grid.Plots {
		if len(plot.EntitiesID) == 0 {
			continue
		}

		topID := plot.EntitiesID[len(plot.EntitiesID)-1]

		for _, id := range plot.EntitiesID {
			if ent, ok := h.world.Entities.Get(entity.ID(id)); ok {
				if ent.GetState()&entity.Revealed != 0 {
					flipDir := plot.Tilt.ToFlipDirection()

					if id == topID {
						_, _ = h.world.FlipTile(gridID, plot.Position, flipDir, "system_hide")
						h.world.EventBus.PublishImmediate(event.NewEntityRevealedEvent(
							entity.Position(plot.Position), id, gridID, flipDir,
							map[string]interface{}{"reason": "system_hide"}))
					} else {
						ent.SetState(entity.Hidden)
					}
				}
			}
		}
	}
}

// processMergeAttempt tente de fusionner les 2 tuiles révélées
func (h *Handler) processMergeAttempt() {
	if len(h.revealedTiles) != 2 {
		h.isProcessing = false
		return
	}

	pos1 := h.revealedTiles[0]
	pos2 := h.revealedTiles[1]

	gridID := h.selectedGridID
	if gridID == "" {
		gridID = h.world.CurrentGridID
	}

	cmd := &usecase.MergeTilesCommand{
		World:    h.world,
		AssocEng: h.assocEngine,
		GridID:   gridID,
		Pos1:     pos1,
		Pos2:     pos2,
		OnSuccess: func() {
			fmt.Printf("[MERGE] ✅ Succès !\n")
			// Après fusion, la commande UseCase a déjà refermé les tuiles logiquement.
			// On vide la mémoire tampon de l'input handler.
			h.revealedTiles = nil
						h.revealedGridIDs = nil
			h.isProcessing = false
			h.ClearSelection()

			h.endTurn()
		},
		OnFailure: func() {
			fmt.Printf("[MERGE] ❌ Échec !\n")
			h.revealedTiles = nil
						h.revealedGridIDs = nil
			h.isProcessing = false
			h.ClearSelection()
			h.endTurn()
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Printf("[MERGE] %v\n", err)
		h.revealedTiles = nil
						h.revealedGridIDs = nil
		h.isProcessing = false
	}
}

// processMatchAttempt tente d'associer les 2 tuiles révélées
func (h *Handler) processMatchAttempt() {
	if len(h.revealedTiles) != 2 {
		h.isProcessing = false
		return
	}

	pos1 := h.revealedTiles[0]
	pos2 := h.revealedTiles[1]

	// Grid IDs par tuile (supporte le cross-zone matching)
	gridID1 := h.revealedGridIDs[0]
	gridID2 := h.revealedGridIDs[1]
	if gridID1 == "" {
		gridID1 = h.world.CurrentGridID
	}
	if gridID2 == "" {
		gridID2 = h.world.CurrentGridID
	}

	grid1, ok1 := h.world.GetGrid(gridID1)
	grid2, ok2 := h.world.GetGrid(gridID2)
	if !ok1 || !ok2 {
		fmt.Printf("[MATCH] Erreur : Grid %s ou %s non trouvé\n", gridID1, gridID2)
		h.revealedTiles = nil
		h.revealedGridIDs = nil
		h.isProcessing = false
		return
	}

	tile1, _ := grid1.Get(pos1)
	tile2, _ := grid2.Get(pos2)

	if len(tile1.EntitiesID) == 0 || len(tile2.EntitiesID) == 0 {
		h.revealedTiles = nil
		h.revealedGridIDs = nil
		h.isProcessing = false
		return
	}

	id1 := tile1.EntitiesID[len(tile1.EntitiesID)-1]
	id2 := tile2.EntitiesID[len(tile2.EntitiesID)-1]
	e1, _ := h.world.Entities.Get(entity.ID(id1))
	e2, _ := h.world.Entities.Get(entity.ID(id2))

	if e1 == nil || e2 == nil {
		h.revealedTiles = nil
		h.revealedGridIDs = nil
		h.isProcessing = false
		return
	}

	// CAS VICTOIRE PAR MATCH : Tout duo de portails révélés (Même grille)
	isP1Portal := e1.HasTag("finish_portal") || e1.HasTag("portable_portal")
	isP2Portal := e2.HasTag("finish_portal") || e2.HasTag("portable_portal")

	if isP1Portal && isP2Portal {
		// On autorise : Finish + Portable OU Finish + Finish (si duplicata) OU Portable + Portable (si duplicata)
		// L'important est d'avoir deux portails "actifs" ensemble.
		fmt.Println("[MATCH] ✅ La paire de portails est réunie ! VICTOIRE.")
		if h.OnVictory != nil {
			h.OnVictory()
		}
		h.revealedTiles = nil
						h.revealedGridIDs = nil
		h.isProcessing = false
		h.ClearSelection()
		return
	}

	fmt.Printf("[MATCH] Comparaison de la paire : %s vs %s\n", h.getEntityInfo(e1), h.getEntityInfo(e2))

	cmd := &usecase.MatchTilesCommand{
		World:    h.world,
		AssocEng: h.assocEngine,
		GridID:   gridID1,
		GridID2:  gridID2,
		Pos1:     pos1,
		Pos2:     pos2,
		OnSuccess: func() {
			fmt.Printf("[MATCH] ✅ Succès ! Paire de %s trouvée.\n", h.getEntityInfo(e1))
			h.revealedTiles = nil
			h.revealedGridIDs = nil
			h.isProcessing = false
			h.ClearSelection()
			h.endTurn()
		},
		OnFailure: func() {
			fmt.Printf("[MATCH] ❌ Échec ! %s et %s ne correspondent pas.\n", h.getEntityInfo(e1), h.getEntityInfo(e2))
			h.revealedTiles = nil
			h.revealedGridIDs = nil
			h.isProcessing = false
			h.ClearSelection()
			h.endTurn()
		},
	}

	if err := cmd.Execute(); err != nil {
		fmt.Printf("[MATCH] %v\n", err)
		h.revealedTiles = nil
						h.revealedGridIDs = nil
		h.isProcessing = false
	}
}

func (h *Handler) handleKeyboard() {
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		if h.OnToggleDetails != nil {
			h.OnToggleDetails()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		if h.OnToggleInvDetails != nil {
			h.OnToggleInvDetails()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		if h.OnToggleAssetsDetails != nil {
			h.OnToggleAssetsDetails()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if h.OnFillInventory != nil {
			h.OnFillInventory()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if h.OnUsePortablePortal != nil {
			center := board.Position{X: -1, Y: -1}
			if hovered, _, ok := h.getHoveredTile(); ok {
				center = hovered
			}
			h.OnUsePortablePortal(h.GetCurrentGridID(), center)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		if len(h.revealedTiles) == 2 {
			fmt.Println("[ACTION] Raccourci clavier : Match")
			h.processMatchAttempt()
			if h.world.TurnTimer != nil {
				h.world.TurnTimer.Reset()
			}
		} else {
			h.tryMatchSelected()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		fmt.Println("[ACTION] Touche Espace activée")
		h.skipPending = true
		h.isProcessing = true
		h.processSkip()

		// Si aucune animation n'est lancée immédiatement, on reset tout de suite
		if h.world.TurnTimer != nil {
			if len(h.world.Components.QueryByComponent("moving_animation")) == 0 {
				h.world.TurnTimer.Reset()
				h.skipPending = false
				h.isProcessing = false
			}
		}
	}

	// Navigation entre zones (Dream Plane)
	h.handleNavigationKeys()

	// Changement de difficulté (Touches F1, F2, F3, F4)
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		h.setDifficulty(meta.LevelEasy)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		h.setDifficulty(meta.LevelNormal)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		h.setDifficulty(meta.LevelHard)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		h.setDifficulty(meta.LevelInsane)
	}

	// S: Spawn entités via debug list, Shift+S: Spawn TOUTES les créatures
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		if h.OnSpawnAllCreatures != nil {
			h.OnSpawnAllCreatures(h.GetCurrentGridID())
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyS) && h.world.Debug.Visible {
		// Spawn via la liste de debug si la fenêtre est ouverte
		if h.OnSpawnEntities != nil {
			h.OnSpawnEntities(h.GetCurrentGridID())
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		if h.OnSpawnRandomCreature != nil {
			h.OnSpawnRandomCreature(h.GetCurrentGridID())
		}
	}

	// F5: Cheat - révéler toutes les tuiles
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if h.OnRevealAll != nil {
			h.OnRevealAll(h.GetCurrentGridID())
		}
	}

	// F6: Cheat - cacher toutes les tuiles
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		if h.OnHideAll != nil {
			h.OnHideAll(h.GetCurrentGridID())
		}
	}

	// F7: Cheat - désceller les sorties
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		if h.OnUnlockNavigation != nil {
			h.OnUnlockNavigation(h.GetCurrentGridID())
		}
	}

	// F8: Cheat - retirer l'état bloqué des tuiles
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		if h.OnClearBlocked != nil {
			h.OnClearBlocked(h.GetCurrentGridID())
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		fmt.Println("[KEY] C : Nettoyage du plateau")
		if h.OnClearBoard != nil {
			h.OnClearBoard(h.GetCurrentGridID())
		}
	}

	for i := 0; i < 9; i++ {
		key := ebiten.Key(i + int(ebiten.Key1))
		if inpututil.IsKeyJustPressed(key) {
			if i < len(h.world.GridOrder) {
				gridID := h.world.GridOrder[i]
				fmt.Printf("[INPUT] Changement de grille : -> %s\n", gridID)
				if h.OnSwitchGrid != nil {
					h.OnSwitchGrid(gridID)
					h.ClearSelection()
				}
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		fmt.Println("[INPUT] Abandon de la partie")
		if h.OnExitToMenu != nil {
			h.OnExitToMenu()
		} else {
			h.ClearSelection()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		fmt.Println("[ACTION] Réinitialisation de la rotation")
		rotations := 0
		if grid, ok := h.world.GetCurrentGrid(); ok {
			for int(grid.MainBearing) != 0 {
				_ = h.world.RotateGrid(grid.ID)
				rotations++
			}
		}
		if h.OnTriggerQuake != nil && rotations > 0 {
			angle := float32(rotations) * float32(math.Pi/2)
			h.OnTriggerQuake(h.GetCurrentGridID(), true, angle)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPEqual) {
		fmt.Println("[ACTION] Rotation horaire (+90°)")
		cmd := &usecase.RotateGridCommand{World: h.world, GridID: h.GetCurrentGridID()}
		_ = cmd.Execute()
		if h.OnTriggerQuake != nil {
			h.OnTriggerQuake(h.GetCurrentGridID(), true, float32(math.Pi/2))
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		fmt.Println("[ACTION] Rotation anti-horaire (-90°)")
		// 3 rotations horaires = 1 rotation anti-horaire
		for i := 0; i < 3; i++ {
			cmd := &usecase.RotateGridCommand{World: h.world, GridID: h.GetCurrentGridID()}
			_ = cmd.Execute()
		}
		if h.OnTriggerQuake != nil {
			h.OnTriggerQuake(h.GetCurrentGridID(), false, float32(math.Pi/2))
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackslash) {
		fmt.Println("[KEY] \\ : Retour au menu")
		if h.OnExitToMenu != nil {
			h.OnExitToMenu()
		}
	}
}

func (h *Handler) handleNavigationKeys() {
	var dir entity.Direction
	var pressed bool

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		dir = board.North
		pressed = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		// Sud : Uniquement si Shift n'est pas pressé ET que la fenêtre de debug n'est pas ouverte
		if !ebiten.IsKeyPressed(ebiten.KeyShift) && !h.world.Debug.Visible {
			dir = board.South
			pressed = true
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		dir = board.West
		pressed = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		dir = board.East
		pressed = true
	}

	if pressed {
		cmd := &usecase.SwitchZoneCommand{World: h.world, Direction: dir}
		if err := cmd.Execute(); err == nil {
			fmt.Printf("[NAVIGATION] Passage à la zone: %s\n", h.world.CurrentGridID)
			h.ClearSelection()
			h.isTransitioning = true
			h.transitionTimer = 30
		}
	}
}

func (h *Handler) setDifficulty(level meta.DifficultyLevel) {
	settings := meta.GetSettings(level)
	h.world.Difficulty = settings
	fmt.Printf("[DIFFICULTÉ] Niveau changé pour : %s\n", level)
	// Synchronise la durée du timer temps réel
	if h.world.TurnTimer != nil {
		h.world.TurnTimer.SetMaxTime(settings.TurnTimerDuration)
	}
	h.world.EventBus.PublishImmediate(domain.Event{
		Type:     domain.EventType("difficulty_changed"),
		SourceID: "player",
		Payload: map[string]interface{}{
			"level": string(level),
		},
	})
}

func (h *Handler) tryMatchSelected() {
	if h.selectedTile == nil {
		fmt.Println("[MATCH] Erreur : Aucune tuile sélectionnée")
		return
	}

	grid, ok := h.world.GetGrid(h.selectedGridID)
	if !ok {
		return
	}

	for _, plot := range grid.Plots {
		if plot.Position.X == h.selectedTile.X && plot.Position.Y == h.selectedTile.Y {
			continue
		}

		if len(plot.EntitiesID) == 0 {
			continue
		}

		topID := plot.EntitiesID[len(plot.EntitiesID)-1]
		ent, ok := h.world.Entities.Get(entity.ID(topID))
		if !ok {
			continue
		}

		if ent.GetState() == entity.Revealed {
			id1 := ""
			if tile1, err := grid.Get(*h.selectedTile); err == nil && len(tile1.EntitiesID) > 0 {
				id1 = tile1.EntitiesID[len(tile1.EntitiesID)-1]
			}
			e1, _ := h.world.Entities.Get(entity.ID(id1))

			fmt.Printf("[MATCH] Comparaison manuelle : %s vs %s\\n", h.getEntityInfo(e1), h.getEntityInfo(ent))

			cmd := &usecase.MatchTilesCommand{
				World:    h.world,
				AssocEng: h.assocEngine,
				GridID:   h.selectedGridID,
				Pos1:     *h.selectedTile,
				Pos2:     plot.Position,
				OnSuccess: func() {
					fmt.Println("[MATCH] ✅ Succès !")
					h.ClearSelection()
					h.endTurn()
				},
				OnFailure: func() {
					fmt.Println("[MATCH] ❌ Échec !")
					h.ClearSelection()
					h.endTurn()
				},
			}

			if err := cmd.Execute(); err != nil {
				fmt.Printf("[MATCH] Erreur : %v\n", err)
			}
			return
		}
	}
}

func (h *Handler) getHoveredTile() (board.Position, string, bool) {
    if h.renderer == nil {
       return board.Position{}, "", false
    }
    // FIX MOBILE : Remplacer CursorPosition par getInteractionPosition
    x, y := h.getInteractionPosition()

    // Si aucun contact/souris, on s'arrête là
    if x == -1 && y == -1 {
       return board.Position{}, "", false
    }
    return h.renderer.ScreenToGrid(x, y, h.world)
}

func (h *Handler) getClickedExit() (entity.Direction, int, bool) {
    // FIX MOBILE : Remplacer CursorPosition par getInteractionPosition
    x, y := h.getInteractionPosition()
    if x == -1 && y == -1 {
       return 0, 0, false
    }

    // Coordonnées relatives au Playmat
    px := float64(x) - ui.PlaymatX
    py := float64(y) - ui.PlaymatY

    return h.checkExitClick(px, py)
}

// calculateFlipDirection détermine la direction de flip basée sur la position du clic dans la tuile
func (h *Handler) calculateFlipDirection(gridID string) domain.FlipDirection {
    if h.renderer == nil {
       return usecase.DefaultFlipDirection
    }

    // FIX MOBILE : Récupère la position tactile ou souris (getInteractionPosition)
    cursorX, cursorY := h.getInteractionPosition()
    if cursorX == -1 && cursorY == -1 {
       return usecase.DefaultFlipDirection
    }

    localX, localY, gID, ok := h.renderer.ScreenToLocalTile(cursorX, cursorY, h.world)
    if !ok || gID != gridID {
       return usecase.DefaultFlipDirection
    }

    tileSize := h.renderer.GetTileSize()
    return entity.CalculateFlipDirection(tileSize, localX, localY)
}

func (h *Handler) checkExitClick(px, py float64) (entity.Direction, int, bool) {
	if px >= ui.ExitNorthX && px < ui.ExitNorthX+ui.ExitNorthW && py >= ui.ExitNorthY && py < ui.ExitNorthY+ui.ExitNorthH {
		index := 0
		if px >= ui.ExitNorthX+ui.TileSize {
			index = 1
		}
		return board.North, index, true
	}
	if px >= ui.ExitEastX && px < ui.ExitEastX+ui.ExitEastW && py >= ui.ExitEastY && py < ui.ExitEastY+ui.ExitEastH {
		index := 0
		if py >= ui.ExitEastY+ui.TileSize {
			index = 1
		}
		return board.East, index, true
	}
	if px >= ui.ExitSouthX && px < ui.ExitSouthX+ui.ExitSouthW && py >= ui.ExitSouthY && py < ui.ExitSouthY+ui.ExitSouthH {
		index := 0
		if px >= ui.ExitSouthX+ui.TileSize {
			index = 1
		}
		return board.South, index, true
	}
	if px >= ui.ExitWestX && px < ui.ExitWestX+ui.ExitWestW && py >= ui.ExitWestY && py < ui.ExitWestY+ui.ExitWestH {
		index := 0
		if py >= ui.ExitWestY+ui.TileSize {
			index = 1
		}
		return board.West, index, true
	}
	return 0, 0, false
}

func (h *Handler) renderHighlights(screen *ebiten.Image) {
	// Portail portable preview (doit s'afficher même sur cases vides)
	if h.portablePortalMode {
		if hovered, gridID, ok := h.getHoveredTile(); ok {
			if gridID == board.InventoryGridID {
				// Ne pas afficher l'aperçu quand on survole l'inventaire
			} else {
				grid, ok := h.world.GetGrid(gridID)
				if ok && h.isValidPortalPreviewPosition(grid, hovered) {
					h.renderer.RenderPortalPlacementPreview(screen, hovered, gridID, h.world)
				}
			}
		}
	}

	// Highlight de la tuile survolée (ne s'affiche que si entité présente ET pas inventaire)
	if hovered, gridID, ok := h.getHoveredTile(); ok {
		if gridID == board.InventoryGridID {
			return
		}
		grid, ok := h.world.GetGrid(gridID)
		if !ok {
			return
		}

		tile, err := grid.Get(hovered)
		if err != nil {
			return
		}

		if len(tile.EntitiesID) == 0 {
			return
		}

		topID := tile.EntitiesID[len(tile.EntitiesID)-1]
		ent, ok := h.world.Entities.Get(entity.ID(topID))
		if !ok {
			return
		}

		var highlightColor color.Color
		state := ent.GetState()
		if state&entity.Hidden != 0 {
			highlightColor = color.RGBA{255, 255, 0, 100}
		} else if state&entity.Revealed != 0 {
			highlightColor = color.RGBA{0, 255, 255, 100}
		} else {
			highlightColor = color.RGBA{255, 255, 255, 50}
		}

		h.renderer.RenderSelectionHighlight(screen, hovered, gridID, highlightColor, h.world)
	}

	if h.selectedTile != nil {
		h.renderer.RenderSelectionHighlight(
			screen,
			*h.selectedTile,
			h.selectedGridID,
			color.RGBA{255, 0, 0, 150},
			h.world,
		)
	}
}

func (h *Handler) isValidPortalPreviewPosition(grid *board.Grid, center board.Position) bool {
	return center.X >= 1 && center.Y >= 1 && center.X <= grid.Width-2 && center.Y <= grid.Height-2
}

func (h *Handler) GetCurrentGridID() string {
	if h.selectedGridID != "" {
		return h.selectedGridID
	}
	return h.world.CurrentGridID
}

// GetRevealedTiles retourne les tuiles révélées pendant le tour courant.
// Utilisé par le gestionnaire de boutons d'action pour le calcul réactif.
func (h *Handler) GetRevealedTiles() []board.Position {
	return h.revealedTiles
}

func (h *Handler) ClearSelection() {
	h.selectedTile = nil
	h.selectedGridID = ""
}

func (h *Handler) SetPortablePortalMode(active bool) {
	h.portablePortalMode = active
}

func (h *Handler) StartVictoryTimer(duration float64) {
	h.victoryTimer = domain.NewTurnTimer(duration)
	h.victoryTimer.Reset()
}

func (h *Handler) GetVictoryTimerProgress() float64 {
	if h.victoryTimer == nil {
		return 0
	}
	return h.victoryTimer.Progress()
}

func (h *Handler) IsVictoryTimerActive() bool {
	return h.victoryTimer != nil && h.victoryTimer.Running
}

func (h *Handler) IsPortablePortalMode() bool {
	return h.portablePortalMode
}

// handlePortalClick gère les interactions avec les portails révélés.
// Retourne true si l'entrée a été consommée (victoire).
func (h *Handler) handlePortalClick(ent entity.Entity, gridID string, pos board.Position) bool {
	state := ent.GetState()

	// --- LOGS CONSOLE POUR LES PORTAILS ---
	if ent.HasTag("start_portal") {
		fmt.Println("[PORTAIL] Les portails de départ ne peuvent pas être utilisés.")
	} else if ent.HasTag("portable_portal") {
		stateStr := h.formatState(state)
		fmt.Printf("[PORTAIL] Zone: %s, Position: (%d,%d), Etat: %s\n", gridID, pos.X, pos.Y, stateStr)
	} else if ent.HasTag("finish_portal") {
		stateStr := h.formatState(state)
		fmt.Printf("[PORTAIL] Position: (%d,%d), Etat: %s\n", pos.X, pos.Y, stateStr)
	}

	// CAS VICTOIRE : Portail de fin ou Portail portable déployé
	isFinish := ent.HasTag("finish_portal")
	isPortable := ent.HasTag("portable_portal")

	if isFinish || isPortable {
		// On cherche l'autre partie du duo sur N'IMPORTE QUELLE grille
		targetTag := "portable_portal"
		if isPortable {
			targetTag = "finish_portal"
		}

		foundOther := false
		for _, e := range h.world.Entities.GetAllActive() {
			// On vérifie que l'autre portail existe et est RÉVÉLÉ
			if e.HasTag(targetTag) && e.GetState()&entity.Revealed != 0 {
				foundOther = true
				break
			}
		}

		if foundOther {
			fmt.Println("[VICTOIRE] Portail final activé !")
			if h.OnVictory != nil {
				h.OnVictory()
			}
			return true
		}
	}
	return false
}

func (h *Handler) formatState(state entity.TileState) string {
	parts := []string{}
	if state&entity.Hidden != 0 {
		parts = append(parts, "Caché")
	}
	if state&entity.Revealed != 0 {
		parts = append(parts, "Révélé")
	}
	if state&entity.Matched != 0 {
		parts = append(parts, "Appairé")
	}
	if state&entity.Blocked != 0 {
		parts = append(parts, "Bloqué")
	}
	if len(parts) == 0 {
		return "Inconnu"
	}
	return "" + parts[0] // On retourne le premier état majeur pour la lisibilité
}

// ResetTimerSkip est appelé par l'app quand le timer temps réel expire.
// Il simule un Skip sans reset du timer (le reset est fait côté app).
func (h *Handler) ResetTimerSkip() {
	if len(h.revealedTiles) > 0 {
		h.hideRevealedTiles()
	}
	h.isProcessing = false
	h.ClearSelection()
	h.endTurn()
}

// ResetGameState réinitialise l'état du jeu (pour retour au menu)
func (h *Handler) ResetGameState() {
	h.selectedTile = nil
	h.selectedGridID = ""
	h.revealedTiles = nil
	h.revealedGridIDs = nil
	h.isProcessing = false
	h.victoryTimer = nil
	h.entranceDir = -1
	h.portablePortalMode = false
	h.isMovedThisTurn = false
	h.ClearFootsteps()
	if h.world != nil {
		h.world.SetPlayerOnBoard(false)
		h.world.ActiveAnimationCount = 0
	}
}

// ClearFootsteps supprime les empreintes de pas actives du monde.
// Si le joueur s'est déplacé ce tour, garde la dernière trace.
func (h *Handler) ClearFootsteps() {
	if h.isMovedThisTurn && len(h.footstepTrackIDs) > 1 {
		// Garde seulement la dernière trace
		toRemove := h.footstepTrackIDs[:len(h.footstepTrackIDs)-1]
		for _, id := range toRemove {
			h.world.RemoveEntity(entity.ID(id))
		}
		h.footstepTrackIDs = h.footstepTrackIDs[len(h.footstepTrackIDs)-1:]
	} else {
		// Pas déplacé : tout supprimer
		for _, id := range h.footstepTrackIDs {
			h.world.RemoveEntity(entity.ID(id))
		}
		h.footstepTrackIDs = h.footstepTrackIDs[:0]
	}
}

// triggerSealingAnimation gère l'animation de bascule des tuiles d'entrée
func (h *Handler) triggerSealingAnimation(gridID string, dir entity.Direction, isSealing bool) {
	grid, ok := h.world.GetGrid(gridID)
	if !ok {
		return
	}

	// Direction de flip : vers l'extérieur du plateau
	var flipDir entity.FlipDirection
	switch dir {
	case board.North:
		flipDir = entity.FlipTop
	case board.South:
		flipDir = entity.FlipBottom
	case board.East:
		flipDir = entity.FlipRight
	case board.West:
		flipDir = entity.FlipLeft
	}

	// Animation inverse pour le déscellage
	if !isSealing {
		flipDir = h.invertFlipDirection(flipDir)
	}

	for i := 0; i < 2; i++ {
		entityID := fmt.Sprintf("exit_%s_%d", board.DirectionToName(dir), i)

		// 1. Détermine les états
		var endState entity.TileState
		if isSealing {
			// Révélé -> Caché + Bloqué
			endState = entity.Hidden | entity.Matched | entity.Blocked
		} else {
			// Caché + Bloqué -> Révélé
			endState = entity.Revealed | entity.Matched
		}

		// 2. Mise à jour de la grille (via une copie du tableau car map index n'est pas adressable directement)
		states := grid.ExitsState[dir]
		states[i] = endState
		grid.ExitsState[dir] = states

		// 3. Déclenche l'animation visuelle
		h.world.EventBus.PublishImmediate(event.Event{
			Type:     event.TileRevealed,
			SourceID: "system",
			Payload: map[string]interface{}{
				"position":       entity.Position{}, // Position virtuelle
				"entity_id":      entityID,
				"grid_id":        gridID,
				"flip_direction": flipDir,
				"reason":         "system_action",
			},
		})
	}
}

// invertFlipDirection retourne la direction opposée pour le déscellage
func (h *Handler) invertFlipDirection(d entity.FlipDirection) entity.FlipDirection {
	switch d {
	case entity.FlipTop:
		return entity.FlipBottom
	case entity.FlipBottom:
		return entity.FlipTop
	case entity.FlipLeft:
		return entity.FlipRight
	case entity.FlipRight:
		return entity.FlipLeft
	}
	return d
}

// spawnFootstepTrack crée une empreinte de pas sur le bord extérieur de la tuile cliquée.
// La direction du bord est déterminée par la position du clic par rapport au centre de la tuile.
func (h *Handler) spawnFootstepTrack(clickX, clickY int, tilePos board.Position, gridID string) {
	// On récupère les coordonnées locales LOGIQUES (déjà corrigées par le renderer)
	localX, localY, _, ok := h.renderer.ScreenToLocalTile(clickX, clickY, h.world)
	if !ok {
		return
	}

	tileSize := float64(h.renderer.GetTileSize())

	// Direction du centre vers le clic LOGIQUE
	dirX := float64(localX) - (tileSize / 2)
	dirY := float64(localY) - (tileSize / 2)
	dist := math.Sqrt(dirX*dirX + dirY*dirY)

	if dist < 1.0 {
		dirX, dirY = 0, 1
		dist = 1.0
	}
	dirX /= dist
	dirY /= dist

	// Position sur le bord extérieur de la tuile (rayon = tileSize/2 + petit décalage)
	edgeDist := tileSize/2 + 4
	offsetX := dirX * edgeDist
	offsetY := dirY * edgeDist

	// Angle de rotation : orienté vers le centre de la tuile (opposé à la direction du bord)
	angle := math.Atan2(-dirY, -dirX)

	// Crée le track d'empreinte de pas
	track := entity.NewTrack("footprints", 3, tilePos, tilePos)
	track.SetGridID(gridID)
	track.OffsetX = offsetX
	track.OffsetY = offsetY
	track.Angle = angle
	h.world.Entities.Register(track)

	// FIFO : max 2 empreintes visibles
	h.footstepTrackIDs = append(h.footstepTrackIDs, string(track.GetID()))
	for len(h.footstepTrackIDs) > 2 {
		oldID := entity.ID(h.footstepTrackIDs[0])
		h.footstepTrackIDs = h.footstepTrackIDs[1:]
		h.world.RemoveEntity(oldID)
	}
}

// spawnFootstepAtArrival crée une empreinte de pas sur le bord de la tuile d'arrivée
// quand le joueur entre dans une nouvelle zone.
func (h *Handler) spawnFootstepAtArrival(arrivalDir entity.Direction) {
	if !h.world.IsPlayerOnBoard() {
		return
	}
	pos := h.world.GetPlayerPosition()
	grid, ok := h.world.GetGrid(h.world.CurrentGridID)
	if !ok || grid == nil {
		return
	}

	tileSize := float64(h.renderer.GetTileSize())

	// Direction vers l'intérieur du plateau (opposée à l'arrivée)
	var dirX, dirY float64
	switch arrivalDir {
	case entity.DirNorth:
		dirX, dirY = 0, 1
	case entity.DirSouth:
		dirX, dirY = 0, -1
	case entity.DirEast:
		dirX, dirY = -1, 0
	case entity.DirWest:
		dirX, dirY = 1, 0
	default:
		dirX, dirY = 0, 1
	}

	edgeDist := tileSize/2 + 4
	offsetX := dirX * edgeDist
	offsetY := dirY * edgeDist
	angle := math.Atan2(-dirY, -dirX)

	track := entity.NewTrack("footprints", 3, pos, pos)
	track.SetGridID(h.world.CurrentGridID)
	track.OffsetX = offsetX
	track.OffsetY = offsetY
	track.Angle = angle
	h.world.Entities.Register(track)
	h.footstepTrackIDs = append(h.footstepTrackIDs, string(track.GetID()))
}

// exitMatchable est un wrapper pour soumettre les sorties au moteur d'association
type exitMatchable struct {
	id string
}

func (m *exitMatchable) GetMatchID() string      { return m.id }
func (m *exitMatchable) GetLogicKey() string     { return "" }
func (m *exitMatchable) GetElement() string      { return "" }
func (m *exitMatchable) GetNarrativeTag() string { return "" }
func (m *exitMatchable) GetMatchTypes() []string { return []string{"orientation"} }
func (m *exitMatchable) GetCumulationLevel() int { return 0 }
func (m *exitMatchable) IsCumulated() bool       { return false }
func (m *exitMatchable) GetCategory() string     { return "navigation" }
