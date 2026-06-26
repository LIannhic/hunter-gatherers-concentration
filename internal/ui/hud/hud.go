// HUD affiche les informations de l'interface
package hud

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/event"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	dbg "github.com/LIannhic/hunter-gatherers-concentration/internal/ui/debug"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// BoardRenderer minimal interface for HUD
type BoardRenderer interface {
	RenderInventoryLoot(target *ebiten.Image, world *domain.World, selectedIdx int, selection map[int]bool, confirmAll bool)
}

// NotificationMessage représente un message défilant
type NotificationMessage struct {
	Text        string
	X           float64
	BoxWidth    float64
	RepeatCount int
	Speed       float64
}

// ComboMessage représente un message de combo "juicy"
type ComboMessage struct {
	Text        string
	Timer       int
	MaxTimer    int
	Juiciness   int
	Count       int
	Scale       float64
	YOffset     float64
	XOffset     float64
	TurnCreated int
}

// HUD affiche les informations de jeu
type HUD struct {
	world                *domain.World
	assets               *assets.Manager
	renderer             BoardRenderer
	showDetails          bool
	showInventoryDetails bool
	showAssetsDetails    bool
	assetPage            int
	showVictory          bool
	fullFeedbackTimer    int // Timer pour le retour visuel d'inventaire plein

	inventoryOffscreen *ebiten.Image // Buffer pour le clipping de l'inventaire

	// NOUVEAU: Fenêtres de debug et difficulté
	debugWindow   *dbg.DebugWindow
	DiffSelection *DifficultySelection

	// Gestion de la suppression et sélection
	selectedLoots     map[int]bool // Indices sélectionnés
	selectedLootIndex int          // Item currently selected for usage
	confirmClearAll   bool         // vrai si on a cliqué sur X sans sélection

	// Feedback temps réel du compte à rebours
	getTimerRemaining func() float64 // Temps restant du timer
	getTimerPanic     func() bool    // true si < 3s
	pulseFrame        int            // Compteur de frames pour l'animation de pulse

	// Système de messages défilants
	queueLeft     []string
	queueRight    []string
	activeLeft    *NotificationMessage
	activeRight          *NotificationMessage
	toxicIntensity       float64 // Intensité de la toxicité pour le feedback visuel
	lastToxicTurn        int     // Dernier tour où on a reçu des dégâts de poison

	// Combo Juicy
	comboMsg *ComboMessage

	// Navigation des stades dans l'atlas des assets
	resourceStageIndex map[string]int
}

// NewHUD crée un nouveau HUD
func NewHUD(world *domain.World) *HUD {
	h := &HUD{
		world:                world,
		showDetails:          false,
		showInventoryDetails: false,
		showAssetsDetails:    false,
		assetPage:            0,
		showVictory:          false,
		fullFeedbackTimer:    0,
		inventoryOffscreen:   ebiten.NewImage(int(ui.InventoryW), 331),
		selectedLoots:        make(map[int]bool),
		selectedLootIndex:    -1,
		confirmClearAll:      false,
		debugWindow:          dbg.NewDebugWindow(world),
		DiffSelection:        NewDifficultySelection(),
		queueLeft:            make([]string, 0),
		queueRight:           make([]string, 0),
		resourceStageIndex:   make(map[string]int),
	}

	// S'abonne aux événements d'inventaire plein
	world.EventBus.SubscribeFunc("inventory_full", func(e event.Event) {
		h.fullFeedbackTimer = 60 // 1 seconde à 60 fps
	})

	// S'abonne aux messages d'item pour les effets de butin
	world.EventBus.SubscribeFunc(event.ItemMessage, func(e event.Event) {
		if msg, ok := e.Payload["message"].(string); ok {
			h.AddMessage(msg, "left")
		}
	})

	// S'abonne aux dégâts pour afficher les confrontations
	world.EventBus.SubscribeFunc(event.PlayerDamaged, func(e event.Event) {
		reason, _ := e.Payload["reason"].(string)
		damage, _ := e.Payload["damage"].(int)
		dmgType, _ := e.Payload["type"].(string)

		if dmgType == "poison" {
			stackCount, _ := e.Payload["stack_count"].(int)
			h.toxicIntensity = float64(stackCount)
			h.lastToxicTurn = h.world.Turn
		}

		msg := ""
		if reason == "confrontation" {
			msg = fmt.Sprintf("CONFRONTATION ! -%d HP", damage)
		} else if reason == "invalid_match" {
			msg = "MATCH INVALIDE !"
		} else if reason == "skipped_valid_match" {
			msg = "MATCH IGNORE !"
		} else if reason == "toxicity" {
			msg = fmt.Sprintf("TOXICITÉ ! -%d HP", damage)
		}
		if msg != "" {
			h.AddMessage(msg, "right")
		}
	})

	// S'abonne au passage de niveau
	world.EventBus.SubscribeFunc(event.LevelUp, func(e event.Event) {
		level, _ := e.Payload["level"].(int)
		h.AddMessage(fmt.Sprintf("NIVEAU %d ATTEINT !", level), "left")
	})

	// S'abonne au début d'amnésie pour afficher l'effet à droite
	world.EventBus.SubscribeFunc(event.AmnesiaStarted, func(e event.Event) {
		duration, _ := e.Payload["duration"].(int)
		h.AddMessage(fmt.Sprintf("AMNÉSIE ! %d tours.", duration), "right")
	})

	// S'abonne aux combos juicy
	world.EventBus.SubscribeFunc(event.ComboTriggered, func(e event.Event) {
		txt, _ := e.Payload["text"].(string)
		count, _ := e.Payload["count"].(int)
		juiciness, _ := e.Payload["juiciness"].(int)

		fmt.Printf("[HUD] ComboTriggered received! text=%q, count=%d, juiciness=%d\n", txt, count, juiciness)

		h.comboMsg = &ComboMessage{
			Text:        txt,
			Count:       count,
			Juiciness:   juiciness,
			Timer:       120,
			MaxTimer:    120,
			Scale:       1.0,
			XOffset:     100.0,
			TurnCreated: h.world.Turn + 1, // +1 car Turn est incrémenté dans le même Engine.Update()
		}
	})

	return h
}

// AddMessage ajoute un message à la file d'attente
func (h *HUD) AddMessage(text string, area string) {
	if area == "left" {
		h.queueLeft = append(h.queueLeft, text)
	} else {
		h.queueRight = append(h.queueRight, text)
	}
}

// Update met à jour l'état interne de l'HUD (animations, timers)
func (h *HUD) Update() {
	if h.fullFeedbackTimer > 0 {
		h.fullFeedbackTimer--
	}

	// Reset de l'intensité toxique si le tour a changé sans nouveau poison
	if h.world.Turn > h.lastToxicTurn {
		h.toxicIntensity *= 0.9 // Décroissance douce
		if h.toxicIntensity < 0.1 {
			h.toxicIntensity = 0
		}
	}

	h.updateMessageArea("left")
	h.updateMessageArea("right")

	// Update combo message
	if h.comboMsg != nil {
		// Efface si le tour a changé (fin de tour)
		if h.world.Turn > h.comboMsg.TurnCreated {
			h.comboMsg = nil
		} else {
			// Animation de slide vers le centre
			h.comboMsg.XOffset *= 0.85
			if math.Abs(h.comboMsg.XOffset) < 0.5 {
				h.comboMsg.XOffset = 0
			}

			h.comboMsg.YOffset = 0

			// Pulse initial
			t := float64(h.comboMsg.MaxTimer - h.comboMsg.Timer)
			if t < 20 {
				h.comboMsg.Scale = 1.0 + (math.Sin(t*0.2) * 0.5 * float64(h.comboMsg.Juiciness) / 3.0)
			} else {
				h.comboMsg.Scale = 1.0
			}
		}
	}

	h.pulseFrame++
}

// UpdateMessageAreas met à jour les zones de messages après le ProcessQueue.
// Doit être appelé après Engine.UpdateFrame() pour traiter les événements fraîchement dispatchés.
func (h *HUD) UpdateMessageAreas() {
	h.updateMessageArea("left")
	h.updateMessageArea("right")
}

func (h *HUD) updateMessageArea(area string) {
    var active **NotificationMessage
    var queue *[]string
    var boxWidth float64

    if area == "left" {
       active = &h.activeLeft
       queue = &h.queueLeft
       boxWidth = ui.MessageBoxWLeft
    } else {
       active = &h.activeRight
       queue = &h.queueRight
       boxWidth = ui.MessageBoxWRight
    }

    if *active == nil && len(*queue) > 0 {
       msgText := (*queue)[0]
       *queue = (*queue)[1:]
       *active = &NotificationMessage{
          Text:     msgText,
          X:        boxWidth,
          BoxWidth: boxWidth,
          Speed:    h.world.Debug.MessageSpeed,
       }
    }

    if *active != nil {
       m := *active
       m.X -= m.Speed

       textWidth := float64(len(m.Text) * 7)

       if m.X < -textWidth {
          m.RepeatCount++
          if m.RepeatCount >= 2 {
             *active = nil
          } else {
             m.X = m.BoxWidth
          }
       }
    }
}

// SetTimerCallbacks injecte les accesseurs au compte à rebours temps réel.
func (h *HUD) SetTimerCallbacks(getRemaining func() float64, getPanic func() bool) {
	h.getTimerRemaining = getRemaining
	h.getTimerPanic = getPanic
}

func (h *HUD) SetBoardRenderer(r BoardRenderer) {
	h.renderer = r
}

// SetPotentialCosts reçoit le coût potentiel calculé par l'Input handler.
// Implémentation minimale : on pourrait l'afficher dans le HUD plus tard.
func (h *HUD) SetPotentialCosts(mana, health, sanity int) {
	// Pour l'instant, on n'affiche rien. Méthode présente pour liaison fonctionnelle.
}

func (h *HUD) SetAssetsManager(am *assets.Manager) {
	h.assets = am
}

func (h *HUD) GetSelectedLootItem() *player.LootItem {
	if h.selectedLootIndex < 0 || h.selectedLootIndex >= len(h.world.Player.Inventory.Items) {
		return nil
	}
	return h.world.Player.Inventory.Items[h.selectedLootIndex]
}

func (h *HUD) GetSelectedLootIndex() int {
	return h.selectedLootIndex
}

func (h *HUD) IsPortablePortalSelected() bool {
	item := h.GetSelectedLootItem()
	return item != nil && item.SourceID == player.PortablePortalItemSourceID
}

func (h *HUD) IsEchoHoundSelected() bool {
	item := h.GetSelectedLootItem()
	return item != nil && item.SourceID == player.EchoHoundItemSourceID
}

func (h *HUD) IsDreamberrySelected() bool {
	item := h.GetSelectedLootItem()
	return item != nil && item.SourceID == player.DreamberryItemSourceID
}

func (h *HUD) ClearActiveLootSelection() {
	h.selectedLootIndex = -1
}

// ClearMessages vide les files de messages défilants (gauche et droite).
func (h *HUD) ClearMessages() {
	h.queueLeft = h.queueLeft[:0]
	h.queueRight = h.queueRight[:0]
	h.activeLeft = nil
	h.activeRight = nil
}

// ToggleDetails bascule l'affichage de la fenêtre de détails
func (h *HUD) ToggleDetails() {
	h.showDetails = !h.showDetails
	if h.showDetails {
		h.showInventoryDetails = false
		h.showAssetsDetails = false
	}
}

// ToggleInventoryDetails bascule l'affichage de la liste de l'inventaire
func (h *HUD) ToggleInventoryDetails() {
	h.showInventoryDetails = !h.showInventoryDetails
	if h.showInventoryDetails {
		h.showDetails = false
		h.showAssetsDetails = false
	}
}

// ToggleAssetsDetails bascule l'affichage de l'atlas des assets
func (h *HUD) ToggleAssetsDetails() {
	h.showAssetsDetails = !h.showAssetsDetails
	if h.showAssetsDetails {
		h.showDetails = false
		h.showInventoryDetails = false
		h.showVictory = false
		if h.debugWindow != nil && h.world != nil {
			h.world.Debug.Visible = false
		}
	}
}

func (h *HUD) ToggleDebugWindow() {
	if h.debugWindow != nil && h.world != nil {
		h.world.Debug.Visible = !h.world.Debug.Visible
		if h.world.Debug.Visible {
			h.showDetails = false
			h.showInventoryDetails = false
			h.showAssetsDetails = false
			h.showVictory = false
		}
	}
}

func (h *HUD) ShowVictory() {
	h.showVictory = true
	h.showDetails = false
	h.showInventoryDetails = false
	h.showAssetsDetails = false
}

func (h *HUD) HideVictory() {
	h.showVictory = false
}

func (h *HUD) IsVictoryVisible() bool {
	return h.showVictory
}

// Render dessine le HUD complet
func (h *HUD) Render(screen *ebiten.Image) {
	h.renderPortrait(screen)
	h.renderInventory(screen)
	h.renderGauges(screen)
	h.renderMiniMap(screen)

	h.renderMessageArea(screen, "left")
	h.renderMessageArea(screen, "right")

	h.renderJuicyCombo(screen)

	if h.showDetails {
		h.renderDetailWindow(screen)
	}

	if h.showInventoryDetails {
		h.renderInventoryWindow(screen)
	}

	if h.showAssetsDetails {
		h.renderAssetsWindow(screen)
	}

	if h.showVictory {
		h.renderVictoryWindow(screen)
	}

	// NOUVEAU: Rendu des fenêtres modales
	if h.debugWindow != nil {
		h.debugWindow.Render(screen)
	}
	if h.DiffSelection != nil {
		h.DiffSelection.Render(screen)
	}
}

func (h *HUD) renderMessageArea(screen *ebiten.Image, area string) {
	var x, y, w, hBox float64
	var active *NotificationMessage

	if area == "left" {
		x, y, w, hBox = ui.MessageBoxXLeft, ui.MessageBoxYLeft, ui.MessageBoxWLeft, ui.MessageBoxHLeft
		active = h.activeLeft
	} else {
		x, y, w, hBox = ui.MessageBoxXRight, ui.MessageBoxYRight, ui.MessageBoxWRight, ui.MessageBoxHRight
		active = h.activeRight
	}

	// 1. Fond de la boîte de message
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(hBox), color.RGBA{20, 20, 30, 180}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(hBox), 1, color.RGBA{100, 100, 150, 100}, true)

	if active == nil {
		return
	}

    // 2. Texte défilant (avec clipping)
    // On crée une sous-image pour le clipping (le repère reste celui de 'screen')
    rect := image.Rect(int(x), int(y), int(x+w), int(y+hBox))
    msgImg := screen.SubImage(rect).(*ebiten.Image)

    // Calcul de la position Y centrée (relative à l'écran, donc on ajoute 'y')
    ty := y + (hBox / 2) + 5

    // On dessine sur msgImg en coordonnées absolues (repère screen)
    text.Draw(msgImg, active.Text, basicfont.Face7x13, int(x+active.X), int(ty), color.RGBA{255, 255, 230, 255})
}

func (h *HUD) renderJuicyCombo(screen *ebiten.Image) {
    // APRÈS (style message box)
    vector.DrawFilledRect(screen, float32(ui.ComboZoneX), float32(ui.ComboZoneY),
        float32(ui.ComboZoneW), float32(ui.ComboZoneH),
        color.RGBA{20, 20, 30, 180}, true)
    vector.StrokeRect(screen, float32(ui.ComboZoneX), float32(ui.ComboZoneY),
        float32(ui.ComboZoneW), float32(ui.ComboZoneH),
        1, color.RGBA{100, 100, 150, 100}, true)

	if h.comboMsg == nil {
		return
	}

	// Position dans la combo zone (au-dessus des jauges)
	baseX := ui.ComboZoneX + ui.ComboZoneW/2
	baseY := ui.ComboZoneY + ui.ComboZoneH/2 + h.comboMsg.YOffset

	fmt.Printf("[HUD] renderJuicyCombo: text=%q, baseX=%.1f, baseY=%.1f, timer=%d, juiciness=%d\n",
		h.comboMsg.Text, float64(baseX), baseY, h.comboMsg.Timer, h.comboMsg.Juiciness)

	// Fond coloré selon juiciness
	var bgClr color.RGBA
	switch h.comboMsg.Juiciness {
	case 1:
		bgClr = color.RGBA{40, 40, 60, 220}
	case 2:
		bgClr = color.RGBA{60, 60, 20, 220}
	case 3:
		bgClr = color.RGBA{80, 50, 10, 220}
	case 4:
		bgClr = color.RGBA{100, 20, 20, 220}
	case 5:
		bgClr = color.RGBA{60, 20, 80, 220}
	}
	vector.DrawFilledRect(screen, float32(ui.ComboZoneX+2), float32(ui.ComboZoneY+2),
		float32(ui.ComboZoneW-4), float32(ui.ComboZoneH-4), bgClr, true)

	// Animation de tremblement (Shake)
	shakeX := 0.0
	shakeY := 0.0
	if h.comboMsg.Juiciness >= 3 {
		amp := float64(h.comboMsg.Juiciness-2) * 2.0
		shakeX = math.Sin(float64(h.pulseFrame)*0.8) * amp
		shakeY = math.Cos(float64(h.pulseFrame)*0.7) * amp
	}

	// Détermine la couleur en fonction du combo/juiciness
	var clr color.RGBA
	switch h.comboMsg.Juiciness {
	case 1:
		clr = color.RGBA{255, 255, 255, 255}
	case 2:
		clr = color.RGBA{255, 255, 100, 255}
	case 3:
		clr = color.RGBA{255, 200, 50, 255}
	case 4:
		clr = color.RGBA{255, 100, 100, 255}
	case 5:
		hue := float64(h.pulseFrame%60) / 60.0
		clr = hsvToRgb(hue, 1.0, 1.0)
	}

	// Effet "Pop" initial
	popScale := 0.0
	if h.comboMsg.Timer > h.comboMsg.MaxTimer-10 {
		popScale = float64(h.comboMsg.MaxTimer-h.comboMsg.Timer) * 2.0
	}

	// Particules décoratives pour juiciness élevée
	if h.comboMsg.Juiciness >= 4 {
		for i := 0; i < 4; i++ {
			angle := float64(i)*math.Pi/2 + float64(h.pulseFrame)*0.1
			dist := 30.0 + math.Sin(float64(h.pulseFrame)*0.2)*5.0
			px := float64(baseX) + math.Cos(angle)*dist
			py := float64(baseY) + math.Sin(angle)*dist
			vector.DrawFilledRect(screen, float32(px-2), float32(py-2), 4, 4, clr, true)
		}
	}

	// Texte avec outline noir (8 directions, 1px) pour lisibilité
	textStr := h.comboMsg.Text
	tw := float64(len(textStr) * 7)
	tx := float64(baseX) - tw/2 + shakeX + h.comboMsg.XOffset
	ty := float64(baseY) + shakeY
	black := color.RGBA{0, 0, 0, 220}

	// Outline noir
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		text.Draw(screen, textStr, basicfont.Face7x13, int(tx)+d[0], int(ty)+d[1], black)
	}

	// Texte coloré par-dessus
	text.Draw(screen, textStr, basicfont.Face7x13, int(tx), int(ty), clr)

	// Affiche le multiplicateur
	if h.comboMsg.Count > 1 {
		countText := fmt.Sprintf("x%d", h.comboMsg.Count)
		text.Draw(screen, countText, basicfont.Face7x13, int(tx+tw+5+popScale), int(ty+popScale), color.RGBA{255, 255, 255, 255})
	}
}

// hsvToRgb conversion simple pour l'effet arc-en-ciel
func hsvToRgb(h, s, v float64) color.RGBA {
	i := math.Floor(h * 6)
	f := h*6 - i
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	var r, g, b float64
	switch int(i) % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

// renderAssetsWindow dessine une fenêtre montrant tous les assets chargés
func (h *HUD) renderAssetsWindow(screen *ebiten.Image) {
	// Fond semi-transparent couvrant l'aire de jeu
	overlay := ebiten.NewImage(ui.ScreenWidth, ui.ScreenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 200})
	screen.DrawImage(overlay, nil)

	// Fenêtre centrale
	winW, winH := 800, 500
	x := (ui.ScreenWidth - winW) / 2
	y := (ui.ScreenHeight - winH) / 2

	vector.DrawFilledRect(screen, float32(x), float32(y), float32(winW), float32(winH), color.RGBA{30, 30, 40, 255}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(winW), float32(winH), 2, color.RGBA{100, 100, 150, 255}, true)

	// Bouton Fermer (Style identique à I et L)
	closeX, closeY := x+winW-30, y+10
	vector.DrawFilledRect(screen, float32(closeX), float32(closeY), 20, 20, color.RGBA{150, 50, 50, 255}, true)
	text.Draw(screen, "X", basicfont.Face7x13, closeX+6, closeY+15, color.White)

	// Définition des stades par ressource (pour navigation dans l'atlas)
	resourceStages := map[string][]string{
		"resource_dreamberry":      {"bourgeon", "fleur", "fruit", "gâté"},
		"resource_whispering_herb": {"graine", "pousse", "mature"},
		"resource_void_bloom":      {"graine", "éclosion", "pleine"},
		"resource_moss_truffle":    {"bourgeon", "pousse", "mature"},
		"resource_echo_crystal":    {"vibrant", "résonnant"},
		"resource_sand_core":       {"instable", "stable"},
	}

	// Liste des assets à montrer
	allAssets := []struct {
		name string
		key  string
	}{
		{"Dos Tuile Std", "tile_hidden"},
		{"Tuile Révélée", "tile_revealed"},
		{"Tuile Matchée", "tile_matched"},
		{"Tuile Piège", "tile_trap"},
		{"Tuile Bloquée", "tile_blocked"},
		{"Tuile Scellée", "tile_sealed"},
		{"Portail", "tile_portal"},
		{"Sortie", "tile_exit"},
		{"Case Vide", "square_empty"},
		{"Trace Boue", "mud"},
		{"Trace Griffes", "claws"},
		{"Herbe Cassée", "broken_grass"},
		{"Empreinte Pas", "footprints"},
		{"Rayon Attaque", "intent_beam"},
		{"Lumifly", "creature_lumifly"},
		{"Shadowstalker", "creature_shadowstalker"},
		{"Burrower", "creature_burrower"},
		{"Flutterwing", "creature_flutterwing"},
		{"Fleeing Sprite", "creature_fleeing_sprite"},
		{"Specter", "creature_specter"},
		{"Echo Hound", "creature_echo_hound"},
		{"Moss Monkey", "creature_moss_monkey"},
		{"Stonewarden", "creature_stonewarden"},
		{"Dreamberry", "resource_dreamberry"},
		{"Moonstone", "resource_moonstone"},
		{"Whisper Herb", "resource_whispering_herb"},
		{"Shard", "resource_crystal_shard"},
		{"Moss Truffle", "resource_moss_truffle"},
		{"Void Bloom", "resource_void_bloom"},
		{"Echo Crystal", "resource_echo_crystal"},
		{"Sand Core", "resource_sand_core"},
		{"Shadow Essence", "resource_shadow_essence"},
	}

	itemsPerPage := 15
	maxPage := (len(allAssets) - 1) / itemsPerPage
	if h.assetPage > maxPage {
		h.assetPage = maxPage
	}

	title := fmt.Sprintf("ATLAS DES ASSETS - Page %d/%d", h.assetPage+1, maxPage+1)
	text.Draw(screen, title, basicfont.Face7x13, x+20, y+30, color.RGBA{200, 200, 255, 255})

	// Boutons de pagination
	if h.assetPage > 0 {
		h.drawButton(screen, float32(x+20), float32(y+winH-40), 100, 25, "PRECEDENT", color.RGBA{60, 60, 80, 255})
	}
	if h.assetPage < maxPage {
		h.drawButton(screen, float32(x+winW-120), float32(y+winH-40), 100, 25, "SUIVANT", color.RGBA{60, 60, 80, 255})
	}

	colWidth := 150
	rowHeight := 140
	itemsPerRow := 5

	startIdx := h.assetPage * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > len(allAssets) {
		endIdx = len(allAssets)
	}

	assetsToDraw := allAssets[startIdx:endIdx]

	for i, asset := range assetsToDraw {
		row := i / itemsPerRow
		col := i % itemsPerRow

		ax := x + 30 + col*colWidth
		ay := y + 60 + row*rowHeight

		// Détermine la clé d'image (avec stade si applicable)
		imgKey := asset.key
		stages, hasStages := resourceStages[asset.key]
		stageIdx := h.resourceStageIndex[asset.key]
		if hasStages {
			if stageIdx < 0 || stageIdx >= len(stages) {
				stageIdx = 0
				h.resourceStageIndex[asset.key] = 0
			}
			imgKey = asset.key + "_" + stages[stageIdx]
		}

		// Cadre de l'asset
		vector.StrokeRect(screen, float32(ax), float32(ay), 88, 88, 1, color.RGBA{60, 60, 80, 255}, true)

		// Dessin de l'asset si disponible
		if h.assets != nil {
			img := h.assets.GetImage(imgKey)
			if img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(ax), float64(ay))
				sw := 88.0 / float64(img.Bounds().Dx())
				sh := 88.0 / float64(img.Bounds().Dy())
				s := sw
				if sh < s {
					s = sh
				}
				op.GeoM.Scale(s, s)
				screen.DrawImage(img, op)
			}
		}

		// Nom de l'asset
		text.Draw(screen, asset.name, basicfont.Face7x13, ax, ay+105, color.White)

		// Affichage de la clé nettoyée (sans préfixes, en majuscules)
		cleanKey := asset.key
		cleanKey = strings.TrimPrefix(cleanKey, "tile_")
		cleanKey = strings.TrimPrefix(cleanKey, "creature_")
		cleanKey = strings.TrimPrefix(cleanKey, "resource_")
		cleanKey = strings.ToUpper(cleanKey)

		text.Draw(screen, "["+cleanKey+"]", basicfont.Face7x13, ax, ay-10, color.RGBA{150, 150, 150, 255})

		// Flèches de navigation de stade (pour ressources avec stades)
		if hasStages {
			arrowColor := color.RGBA{180, 180, 200, 255}
			stageName := stages[stageIdx]
			stageNameX := ax + (88-len(stageName)*7)/2
			text.Draw(screen, stageName, basicfont.Face7x13, stageNameX, ay+85, arrowColor)

			text.Draw(screen, "<", basicfont.Face7x13, ax+2, ay+85, arrowColor)
			text.Draw(screen, ">", basicfont.Face7x13, ax+78, ay+85, arrowColor)
		}
	}
}

func (h *HUD) drawButton(screen *ebiten.Image, x, y, w, buttonH float32, label string, bg color.RGBA) {
	vector.DrawFilledRect(screen, x, y, w, buttonH, bg, true)
	vector.StrokeRect(screen, x, y, w, buttonH, 1, color.RGBA{150, 150, 200, 255}, true)

	// Centrage sommaire du texte
	tw := len(label) * 7
	tx := int(x) + (int(w)-tw)/2
	ty := int(y) + (int(buttonH)+10)/2

	text.Draw(screen, label, basicfont.Face7x13, tx, ty, color.White)
}

// renderVictoryWindow dessine l'écran de victoire
func (h *HUD) renderVictoryWindow(screen *ebiten.Image) {
	overlay := ebiten.NewImage(ui.ScreenWidth, ui.ScreenHeight)
	overlay.Fill(color.RGBA{0, 0, 0, 220})
	screen.DrawImage(overlay, nil)

	winW, winH := 600, 400
	x := (ui.ScreenWidth - winW) / 2
	y := (ui.ScreenHeight - winH) / 2

	vector.DrawFilledRect(screen, float32(x), float32(y), float32(winW), float32(winH), color.RGBA{20, 40, 20, 255}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(winW), float32(winH), 3, color.RGBA{100, 255, 100, 255}, true)

	text.Draw(screen, "VICTOIRE !", basicfont.Face7x13, x+250, y+50, color.RGBA{100, 255, 100, 255})
	text.Draw(screen, "Vous avez franchi le portail final.", basicfont.Face7x13, x+150, y+80, color.White)
	text.Draw(screen, "Appuyez sur ESC pour retourner au menu.", basicfont.Face7x13, x+130, y+100, color.RGBA{150, 150, 150, 255})

	// Calcul des gains
	totalValue := 0
	resourceCount := 0
	creatureCount := 0
	for _, item := range h.world.Player.Inventory.Items {
		switch item.OriginalType {
		case entity.TypeResource:
			resourceCount++
			totalValue += 100 // Valeur arbitraire pour l'instant
		case entity.TypeCreature:
			creatureCount++
			totalValue += 250
		}
	}

	statsY := y + 150
	text.Draw(screen, fmt.Sprintf("Ressources collectées : %d", resourceCount), basicfont.Face7x13, x+100, statsY, color.White)
	text.Draw(screen, fmt.Sprintf("Créatures capturées : %d", creatureCount), basicfont.Face7x13, x+100, statsY+30, color.White)
	text.Draw(screen, fmt.Sprintf("Valeur totale du butin : %d XP", totalValue), basicfont.Face7x13, x+100, statsY+70, color.RGBA{255, 255, 100, 255})

	// Boutons
	btnW, btnH := 160, 40

	// Bouton REJOUER
	bx1 := x + 100
	by := y + 300
	vector.DrawFilledRect(screen, float32(bx1), float32(by), float32(btnW), float32(btnH), color.RGBA{40, 80, 40, 255}, true)
	vector.StrokeRect(screen, float32(bx1), float32(by), float32(btnW), float32(btnH), 1, color.White, true)
	text.Draw(screen, "REJOUER", basicfont.Face7x13, bx1+50, by+25, color.White)

	// Bouton MENU
	bx2 := x + 340
	vector.DrawFilledRect(screen, float32(bx2), float32(by), float32(btnW), float32(btnH), color.RGBA{80, 40, 40, 255}, true)
	vector.StrokeRect(screen, float32(bx2), float32(by), float32(btnW), float32(btnH), 1, color.White, true)
	text.Draw(screen, "MENU", basicfont.Face7x13, bx2+60, by+25, color.White)

	text.Draw(screen, "Appuyez sur ESC pour retourner au menu.", basicfont.Face7x13, x+130, y+360, color.RGBA{150, 150, 150, 255})
}

func (h *HUD) HandleVictoryClick(mx, my int) string {
	if !h.showVictory {
		return ""
	}

	winW, winH := 600, 400
	x := (ui.ScreenWidth - winW) / 2
	y := (ui.ScreenHeight - winH) / 2

	btnW, btnH := 160, 40
	by := y + 300

	// REJOUER
	bx1 := x + 100
	if mx >= bx1 && mx <= bx1+btnW && my >= by && my <= by+btnH {
		return "replay"
	}

	// MENU
	bx2 := x + 340
	if mx >= bx2 && mx <= bx2+btnW && my >= by && my <= by+btnH {
		return "menu"
	}

	return ""
}

func (h *HUD) HandleGameOverClick(mx, my int) string {
	winW, winH := 600, 400
	x := (ui.ScreenWidth - winW) / 2
	y := (ui.ScreenHeight - winH) / 2

	btnW, btnH := 160, 40
	by := y + 300

	// REJOUER
	bx1 := x + 100
	if mx >= bx1 && mx <= bx1+btnW && my >= by && my <= by+btnH {
		return "replay"
	}

	// MENU
	bx2 := x + 340
	if mx >= bx2 && mx <= bx2+btnW && my >= by && my <= by+btnH {
		return "menu"
	}

	return ""
}

// getGridDetailedCounts retourne le nombre d'entités par type/espèce pour une grille donnée
func (h *HUD) getGridDetailedCounts(gridID string) map[string]int {
	counts := make(map[string]int)

	// Ressources
	for _, e := range h.world.Entities.GetByType(entity.TypeResource) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			if r, ok := e.(*domain.Resource); ok {
				counts[r.ResourceType]++
			}
		}
	}
	// Créatures
	for _, e := range h.world.Entities.GetByType(entity.TypeCreature) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			if c, ok := e.(*domain.Creature); ok {
				counts[c.Species]++
			}
		}
	}
	// Structures & Portails
	for _, e := range h.world.Entities.GetByType(entity.TypeStructure) {
		if e.GetGridID() == gridID {
			label := "structure"
			if e.HasTag("start_portal") {
				label = "portail_entree"
			} else if e.HasTag("finish_portal") {
				label = "portail_sortie"
			} else if e.HasTag("dolmen") {
				label = "dolmen"
			} else if e.HasTag("obelisk") {
				label = "obelisque"
			}
			counts[label]++
		}
	}
	return counts
}

// getGridEntityCounts retourne le nombre total de ressources, créatures et structures sur un grid
func (h *HUD) getGridEntityCounts(gridID string) (resCount, creCount, structCount int) {
	for _, e := range h.world.Entities.GetByType(entity.TypeResource) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			resCount++
		}
	}
	for _, e := range h.world.Entities.GetByType(entity.TypeCreature) {
		if e.GetGridID() == gridID && e.GetState() != entity.Matched {
			creCount++
		}
	}
	for _, e := range h.world.Entities.GetByType(entity.TypeStructure) {
		if e.GetGridID() == gridID {
			structCount++
		}
	}
	return
}

func (h *HUD) renderPortrait(screen *ebiten.Image) {
	// Portrait Holder
	vector.StrokeRect(screen, ui.PortraitX, ui.PortraitY, ui.PortraitW, ui.PortraitH, 1, color.RGBA{100, 100, 100, 255}, true)

	// Menu Icon / Button (M)
	mx := ui.PortraitX + ui.MenuIconRelativeX
	my := ui.PortraitY + ui.MenuIconRelativeY
	btnMColor := color.RGBA{150, 150, 150, 255}
	// On pourrait ajouter un effet de survol ici si on avait accès à la souris
	vector.DrawFilledRect(screen, float32(mx), float32(my), float32(ui.MenuIconSize), float32(ui.MenuIconSize), btnMColor, true)
	text.Draw(screen, "M", basicfont.Face7x13, int(mx)+15, int(my)+25, color.Black)

	// Turn and Difficulty (aligned)
	infoX := int(ui.PortraitX) + 60
	infoY := int(ui.PortraitY) + 25
	text.Draw(screen, fmt.Sprintf("T:%d", h.world.Turn), basicfont.Face7x13, infoX, infoY, color.White)

	diffLabel := fmt.Sprintf("D:%s", h.world.Difficulty.Level)
	dx := infoX + 60
	text.Draw(screen, diffLabel, basicfont.Face7x13, dx, infoY, color.RGBA{255, 200, 100, 255})

	// Score (XP Totale)
	scoreLabel := fmt.Sprintf("S:%d", h.world.Player.Stats.TotalExperience)
	sx := dx + 85
	text.Draw(screen, scoreLabel, basicfont.Face7x13, sx, infoY, color.RGBA{100, 255, 100, 255})

	// Zone de clic pour changer la difficulté (Cartouche)
	// (0,0) est PortraitX, PortraitY. On capture le clic dans HandleClick

	// --- COLUMN LEFT: CONTROLS ---
	y := int(ui.PortraitY) + 85
	text.Draw(screen, "ACTION:", basicfont.Face7x13, int(ui.PortraitX)+10, y-15, color.RGBA{100, 200, 255, 255})

	controls := []string{
		"CLIC: Ouvrir",
		"M: Matcher",
		"I: Zones",
		"L: Liste Inv",
		"P: Portail",
		"B: Remplir Inv",
		"ZQSD: Naviguer",
		"ESPACE: Fin",
		"F1-F4: Diff",
		"ESC: Menu",
	}
	for _, c := range controls {
		text.Draw(screen, c, basicfont.Face7x13, int(ui.PortraitX)+10, y, color.RGBA{160, 160, 160, 255})
		y += 18
	}

	// --- COLUMN RIGHT: CURRENT GRID DETAILS ---
	rx := int(ui.PortraitX) + 125
	ry := int(ui.PortraitY) + 85
	text.Draw(screen, "SUR ZONE:", basicfont.Face7x13, rx, ry-15, color.RGBA{100, 200, 255, 255})

	counts := h.getGridDetailedCounts(h.world.CurrentGridID)

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		label := strings.Title(strings.ReplaceAll(k, "_", " "))
		if len(label) > 12 {
			label = label[:12]
		}
		info := fmt.Sprintf("%s:%d", label, counts[k])
		text.Draw(screen, info, basicfont.Face7x13, rx, ry, color.White)
		ry += 16
		if ry > int(ui.PortraitY)+int(ui.PortraitH)-20 {
			break
		}
	}

	if len(keys) == 0 {
		text.Draw(screen, "(Zone vide)", basicfont.Face7x13, rx, ry, color.RGBA{150, 150, 150, 255})
	}

	// Menu Icon
	mxIcon := ui.PortraitX + ui.MenuIconRelativeX
	myIcon := ui.PortraitY + ui.MenuIconRelativeY
	vector.DrawFilledRect(screen, float32(mxIcon), float32(myIcon), float32(ui.MenuIconSize), float32(ui.MenuIconSize), color.RGBA{150, 150, 150, 255}, true)
	text.Draw(screen, "M", basicfont.Face7x13, int(mxIcon)+15, int(myIcon)+25, color.Black)

	// Fullscreen Icon
	fxIcon := ui.PortraitX + ui.FullscreenIconRelativeX
	fyIcon := ui.PortraitY + ui.FullscreenIconRelativeY
	vector.DrawFilledRect(screen, float32(fxIcon), float32(fyIcon), float32(ui.FullscreenIconSize), float32(ui.FullscreenIconSize), color.RGBA{150, 150, 150, 255}, true)
	fsLabel := "F"
	if ebiten.IsFullscreen() {
		fsLabel = "W"
	}
	text.Draw(screen, fsLabel, basicfont.Face7x13, int(fxIcon)+15, int(fyIcon)+25, color.Black)
}

func (h *HUD) renderInventory(screen *ebiten.Image) {
	// Inventory Panel border
	panelClr := color.RGBA{100, 100, 100, 255}
	inv := h.world.Player.Inventory

	// Orange if full
	if inv.IsFull() {
		panelClr = color.RGBA{255, 165, 0, 255} // Orange
	}

	if h.fullFeedbackTimer > 0 {
		if h.fullFeedbackTimer%10 < 5 {
			panelClr = color.RGBA{255, 100, 0, 255} // Flash Orange/Darker Orange
		}
	}
	vector.StrokeRect(screen, ui.InventoryX, ui.InventoryY, ui.InventoryW, ui.InventoryH, 1, panelClr, true)
	text.Draw(screen, "INVENTORY", basicfont.Face7x13, int(ui.InventoryX)+10, int(ui.InventoryY)+20, color.RGBA{100, 200, 255, 255})

	items := inv.Items

	// 1. Dessiner les slots dans le buffer offscreen via le BoardRenderer pour le TILT
	h.inventoryOffscreen.Fill(color.Transparent)
	if h.renderer != nil {
		h.renderer.RenderInventoryLoot(h.inventoryOffscreen, h.world, h.selectedLootIndex, h.selectedLoots, h.confirmClearAll)
	}

	// 2. Afficher le buffer
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ui.InventoryX, ui.InventoryY+40)
	screen.DrawImage(h.inventoryOffscreen, op)

	// 3. Loot counter et bouton X
	lcx := ui.InventoryX + ui.LootCounterRelativeX
	lcy := ui.InventoryY + ui.LootCounterRelativeY
	vector.DrawFilledRect(screen, float32(lcx), float32(lcy), float32(ui.LootCounterSize), float32(ui.LootCounterSize), color.RGBA{50, 50, 50, 255}, true)
	text.Draw(screen, fmt.Sprintf("%d", len(items)), basicfont.Face7x13, int(lcx)+15, int(lcy)+25, color.White)

	dlx := ui.InventoryX + ui.DeleteLootRelativeX
	dly := ui.InventoryY + ui.DeleteLootRelativeY

	btnXClr := color.RGBA{150, 50, 50, 255}
	if len(h.selectedLoots) > 0 || h.confirmClearAll {
		btnXClr = color.RGBA{255, 50, 50, 255} // Bright red if active
	}
	vector.DrawFilledRect(screen, float32(dlx), float32(dly), float32(ui.DeleteLootSize), float32(ui.DeleteLootSize), btnXClr, true)
	text.Draw(screen, "X", basicfont.Face7x13, int(dlx)+15, int(dly)+25, color.White)

	if inv.IsFull() {
		text.Draw(screen, "FULL", basicfont.Face7x13, int(ui.InventoryX)+ui.InventoryW/2-15, int(ui.InventoryY)+ui.InventoryH-10, color.RGBA{255, 165, 0, 255})
	}
}

// renderGauges dessine les jauges HP/Mana/Sanity
func (h *HUD) renderGauges(screen *ebiten.Image) {
	// Gauges Holder
	vector.StrokeRect(screen, float32(ui.GaugesX), float32(ui.GaugesY), float32(ui.GaugesW), float32(ui.GaugesH), 1, color.RGBA{100, 100, 100, 255}, true)

	p := h.world.Player
	if p == nil {
		return
	}

	// Health gauge
	hpColor := color.RGBA{R: 255, G: 50, B: 50, A: 255}
	if h.toxicIntensity > 0 {
		// Interpolation vers le rose (255, 100, 200)
		factor := h.toxicIntensity / 5.0 // Capé à 5 stacks pour le max feedback
		if factor > 1.0 {
			factor = 1.0
		}
		hpColor.G = uint8(50 + factor*50)
		hpColor.B = uint8(50 + factor*150)
	}
	h.drawVerticalGauge(screen, ui.GaugesX+ui.HealthGaugeRelativeX, ui.GaugesY+ui.HealthGaugeRelativeY, "HP", p.Stats.Health, p.Stats.MaxHealth, hpColor)

	// Mana gauge
	h.drawVerticalGauge(screen, ui.GaugesX+ui.ManaGaugeRelativeX, ui.GaugesY+ui.ManaGaugeRelativeY, "MN", p.Stats.Mana, p.Stats.MaxMana, color.RGBA{R: 50, G: 50, B: 255, A: 255})

	// Sanity gauge (avec pulse si panique)
	var sanityX float64 = ui.GaugesX + ui.SanityGaugeRelativeX
	var sanityY float64 = ui.GaugesY + ui.SanityGaugeRelativeY
	if h.getTimerPanic != nil && h.getTimerPanic() {
		// Phase de panique : la jauge de santé mentale tremble de plus en plus fort
		offset := h.computePanicOffset()
		sanityX += offset
	}
	h.drawVerticalGauge(screen, sanityX, sanityY, "SN", p.Stats.Sanity, p.Stats.MaxSanity, color.RGBA{R: 50, G: 255, B: 50, A: 255})
}

// computePanicOffset calcule un décalage oscillant dont l'amplitude augmente
// à mesure que le timer approche de 0 (entre 3s et 0s).
func (h *HUD) computePanicOffset() float64 {
	if h.getTimerRemaining == nil {
		return 0
	}
	remaining := h.getTimerRemaining()
	if remaining <= 0 || remaining >= 3.0 {
		return 0
	}
	// Intensité inversement proportionnelle au temps restant
	intensity := (3.0 - remaining) / 3.0 // 0.0 → 1.0
	maxAmp := 6.0 * intensity            // amplitude max 6 px
	freq := 0.4 + (intensity * 0.6)      // fréquence accélère
	return math.Sin(float64(h.pulseFrame)*freq) * maxAmp
}

func (h *HUD) drawVerticalGauge(screen *ebiten.Image, x, y float64, label string, val, max int, clr color.Color) {
	// Gauge Holder background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(ui.GaugeW), float32(ui.GaugeH), color.RGBA{30, 30, 30, 255}, true)

	// Les jauges prennent désormais toute la hauteur disponible (ui.GaugeH)
	totalPx := float32(ui.GaugeH)

	var fillHeight float32
	if max > 0 {
		fillHeight = (float32(val) / float32(max)) * totalPx
	}

	// Draw background of the actual gauge (fond gris clair pour la partie vide)
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(ui.GaugeW), totalPx, color.RGBA{50, 50, 50, 255}, true)

	// Fill from bottom
	vector.DrawFilledRect(screen, float32(x), float32(y+float64(totalPx-fillHeight)), float32(ui.GaugeW), fillHeight, clr, true)

	// Label
	text.Draw(screen, label, basicfont.Face7x13, int(x)+25, int(y)+int(ui.GaugeH)+15, color.White)
}

func (h *HUD) renderMiniMap(screen *ebiten.Image) {
	// Minimap Holder (270x270 at 1000, 440)
	vector.StrokeRect(screen, float32(ui.MinimapX), float32(ui.MinimapY), float32(ui.MinimapW), float32(ui.MinimapH), 1, color.RGBA{100, 100, 100, 255}, true)

	if h.world.DreamPlane == nil {
		return
	}

	plane := h.world.DreamPlane
	currentGridID := h.world.CurrentGridID
	currentCoords, ok := plane.Coords[currentGridID]
	if !ok {
		return
	}

	const (
		cellSize = 30
		nodeSize = 26
		padding  = 2
		halfGrid = 4 // Centre d'une grille 9x9
	)

	// Couleurs
	adjColor := color.RGBA{60, 60, 65, 255}
	visitedColor := color.RGBA{40, 40, 50, 255}
	currentColor := color.RGBA{220, 200, 50, 255}
	lineColor := color.RGBA{70, 70, 80, 255}

	// 1. Dessiner d'abord les connexions (edges) pour qu'elles soient sous les nodes
	for id, coords := range plane.Coords {
		state := plane.DiscoveryStates[id]
		if state == board.StateHidden {
			continue
		}

		localX := halfGrid + (coords.X - currentCoords.X)
		localY := halfGrid + (coords.Y - currentCoords.Y)

		if localX < 0 || localX >= 9 || localY < 0 || localY >= 9 {
			continue
		}

		// Connexions sortantes
		if conns, ok := plane.Connections[id]; ok {
			for _, targetID := range conns {
				targetState := plane.DiscoveryStates[targetID]
				if targetState == board.StateHidden {
					continue
				}

				targetCoords := plane.Coords[targetID]
				targetLocalX := halfGrid + (targetCoords.X - currentCoords.X)
				targetLocalY := halfGrid + (targetCoords.Y - currentCoords.Y)

				if targetLocalX < 0 || targetLocalX >= 9 || targetLocalY < 0 || targetLocalY >= 9 {
					continue
				}

				// On ne dessine l'edge qu'une seule fois (par ID)
				if id < targetID {
					x1 := float32(ui.MinimapX + float64(localX*cellSize) + 15)
					y1 := float32(ui.MinimapY + float64(localY*cellSize) + 15)
					x2 := float32(ui.MinimapX + float64(targetLocalX*cellSize) + 15)
					y2 := float32(ui.MinimapY + float64(targetLocalY*cellSize) + 15)

					vector.StrokeLine(screen, x1, y1, x2, y2, 2, lineColor, true)
				}
			}
		}
	}

	// 2. Dessiner les nodes
	for id, coords := range plane.Coords {
		state := plane.DiscoveryStates[id]
		if state == board.StateHidden {
			continue
		}

		localX := halfGrid + (coords.X - currentCoords.X)
		localY := halfGrid + (coords.Y - currentCoords.Y)

		if localX < 0 || localX >= 9 || localY < 0 || localY >= 9 {
			continue
		}

		nx := float32(ui.MinimapX + float64(localX*cellSize) + padding)
		ny := float32(ui.MinimapY + float64(localY*cellSize) + padding)

		isCurrent := id == currentGridID

		// Fond du node
		var bgColor color.Color
		if isCurrent {
			bgColor = color.RGBA{45, 40, 20, 255}
		} else if state == board.StateVisited {
			bgColor = visitedColor
		} else {
			bgColor = adjColor
		}

		vector.DrawFilledRect(screen, nx, ny, nodeSize, nodeSize, bgColor, true)

		// Bordure
		bColor := lineColor
		bWidth := float32(1)
		if isCurrent {
			bColor = currentColor
			bWidth = 2
			// Effet de pulsation (Halo doré)
			pulse := float32(math.Sin(float64(h.pulseFrame)*0.1)*1.5 + 1.5)
			vector.StrokeRect(screen, nx-pulse/2, ny-pulse/2, nodeSize+pulse, nodeSize+pulse, 1, color.RGBA{255, 220, 50, 100}, true)
		}
		vector.StrokeRect(screen, nx, ny, nodeSize, nodeSize, bWidth, bColor, true)

		// Icône du biome (si Visited ou Current)
		if state == board.StateVisited || isCurrent {
			grid, ok := h.world.GetGrid(id)
			if ok && h.assets != nil {
				icon := h.assets.GetImage("minimap_" + string(grid.Biome))
				if icon != nil {
					op := &ebiten.DrawImageOptions{}
					// Centrer pour la rotation
					op.GeoM.Translate(-nodeSize/2, -nodeSize/2)
					// Rotation basée sur MainBearing
					op.GeoM.Rotate(float64(grid.MainBearing) * (math.Pi / 2))
					// Replacer
					op.GeoM.Translate(nodeSize/2, nodeSize/2)

					op.GeoM.Translate(float64(nx), float64(ny))
					if isCurrent {
						op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 180, 255})
					} else {
						op.ColorScale.ScaleWithColor(color.RGBA{160, 160, 180, 255})
					}
					screen.DrawImage(icon, op)
				}
			}
		}
	}
}

func (h *HUD) renderDetailWindow(screen *ebiten.Image) {
	// Position et taille de la fenêtre
	winW, winH := 320, 450
	winX := (ui.ScreenWidth - winW) / 2
	winY := (ui.ScreenHeight - winH) / 2

	// Fond translucide
	vector.DrawFilledRect(screen, float32(winX), float32(winY), float32(winW), float32(winH), color.RGBA{10, 10, 20, 230}, true)
	vector.StrokeRect(screen, float32(winX), float32(winY), float32(winW), float32(winH), 2, color.RGBA{100, 100, 150, 255}, true)

	// Titre
	text.Draw(screen, "STATISTIQUES DES ZONES", basicfont.Face7x13, winX+20, winY+30, color.RGBA{100, 200, 255, 255})

	// Icone fermer (X)
	closeX := winX + winW - 30
	closeY := winY + 10
	vector.DrawFilledRect(screen, float32(closeX), float32(closeY), 20, 20, color.RGBA{150, 50, 50, 255}, true)
	text.Draw(screen, "X", basicfont.Face7x13, closeX+6, closeY+15, color.White)

	// Liste des zones (L'ancienne liste du portrait)
	dy := winY + 70
	for _, gridID := range h.world.GridOrder {
		resCount, creCount, structCount := h.getGridEntityCounts(gridID)

		info := fmt.Sprintf("%-15s R:%d C:%d S:%d", gridID, resCount, creCount, structCount)
		var clr color.Color = color.White
		if gridID == h.world.CurrentGridID {
			clr = color.RGBA{255, 255, 0, 255}
		}
		text.Draw(screen, info, basicfont.Face7x13, winX+30, dy, clr)
		dy += 20
	}
}

func (h *HUD) renderInventoryWindow(screen *ebiten.Image) {
	// Position et taille de la fenêtre (identique à renderDetailWindow)
	winW, winH := 320, 450
	winX := (ui.ScreenWidth - winW) / 2
	winY := (ui.ScreenHeight - winH) / 2

	// Fond translucide
	vector.DrawFilledRect(screen, float32(winX), float32(winY), float32(winW), float32(winH), color.RGBA{10, 20, 10, 230}, true)
	vector.StrokeRect(screen, float32(winX), float32(winY), float32(winW), float32(winH), 2, color.RGBA{100, 150, 100, 255}, true)

	// Titre
	text.Draw(screen, "CONTENU DE L'INVENTAIRE", basicfont.Face7x13, winX+20, winY+30, color.RGBA{100, 255, 100, 255})

	// Icone fermer (X)
	closeX := winX + winW - 30
	closeY := winY + 10
	vector.DrawFilledRect(screen, float32(closeX), float32(closeY), 20, 20, color.RGBA{150, 50, 50, 255}, true)
	text.Draw(screen, "X", basicfont.Face7x13, closeX+6, closeY+15, color.White)

	// Liste des items
	inv := h.world.Player.Inventory
	dy := winY + 70

	if len(inv.Items) == 0 {
		text.Draw(screen, "(Inventaire vide)", basicfont.Face7x13, winX+30, dy, color.RGBA{150, 150, 150, 255})
		return
	}

	// Groupement par nom pour la liste détaillée
	counts := make(map[string]int)
	for _, item := range inv.Items {
		counts[item.Name]++
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		info := fmt.Sprintf("%-20s x%d", k, counts[k])
		text.Draw(screen, info, basicfont.Face7x13, winX+30, dy, color.White)
		dy += 20
		if dy > winY+winH-20 {
			break
		}
	}
}

// HandleClick gère les clics sur les éléments de l'HUD
func (h *HUD) HandleClick(x, y int) bool {
	// NOUVEAU: Interception par les fenêtres de debug/difficulté
	if h.debugWindow != nil && h.world != nil && h.world.Debug.Visible {
		if h.debugWindow.HandleClick(x, y) {
			return true
		}
	}
	if h.DiffSelection != nil && h.DiffSelection.visible {
		if level, ok := h.DiffSelection.HandleClick(x, y); ok {
			// On applique le changement
			h.world.Difficulty = meta.GetSettings(level)

			// Si un callback est enregistré (ex: pour démarrer la partie), on l'appelle
			if h.DiffSelection.OnSelected != nil {
				h.DiffSelection.OnSelected(level)
			}

			h.DiffSelection.SetVisible(false)
			return true
		}
		return true // On consomme le clic pour éviter d'interagir avec le plateau sous la modale
	}

	// Interception par l'Atlas des Assets
	if h.showAssetsDetails {
		winW, winH := 800, 500
		wx := (ui.ScreenWidth - winW) / 2
		wy := (ui.ScreenHeight - winH) / 2
		fx, fy := float32(x), float32(y)

		// Bouton Fermer (X)
		if fx >= float32(wx+winW-30) && fx <= float32(wx+winW-10) && fy >= float32(wy+10) && fy <= float32(wy+30) {
			h.showAssetsDetails = false
			return true
		}

		// Boutons Pagination
		allAssetsCount := 32 // Nombre total d'assets dans la liste
		itemsPerPage := 15
		maxPage := (allAssetsCount - 1) / itemsPerPage

		// Bouton Précédent
		if h.assetPage > 0 {
			if fx >= float32(wx+20) && fx <= float32(wx+120) && fy >= float32(wy+winH-40) && fy <= float32(wy+winH-15) {
				h.assetPage--
				return true
			}
		}

		// Bouton Suivant
		if h.assetPage < maxPage {
			if fx >= float32(wx+winW-120) && fx <= float32(wx+winW-20) && fy >= float32(wy+winH-40) && fy <= float32(wy+winH-15) {
				h.assetPage++
				return true
			}
		}

		// Flèches de navigation de stade dans les cellules
		resourceStages := map[string][]string{
			"resource_dreamberry":      {"bourgeon", "fleur", "fruit", "gâté"},
			"resource_whispering_herb": {"graine", "pousse", "mature"},
			"resource_void_bloom":      {"graine", "éclosion", "pleine"},
			"resource_moss_truffle":    {"bourgeon", "pousse", "mature"},
			"resource_echo_crystal":    {"vibrant", "résonnant"},
			"resource_sand_core":       {"instable", "stable"},
		}

		colWidth := 150
		rowHeight := 140
		itemsPerRow := 5

		startIdx := h.assetPage * itemsPerPage
		endIdx := startIdx + itemsPerPage
		if endIdx > allAssetsCount {
			endIdx = allAssetsCount
		}

		allAssets := []struct {
			name string
			key  string
		}{
			{"Dos Tuile Std", "tile_hidden"},
			{"Tuile Révélée", "tile_revealed"},
			{"Tuile Matchée", "tile_matched"},
			{"Tuile Piège", "tile_trap"},
			{"Tuile Bloquée", "tile_blocked"},
			{"Tuile Scellée", "tile_sealed"},
			{"Portail", "tile_portal"},
			{"Sortie", "tile_exit"},
			{"Case Vide", "square_empty"},
			{"Trace Boue", "mud"},
			{"Trace Griffes", "claws"},
			{"Herbe Cassée", "broken_grass"},
			{"Empreinte Pas", "footprints"},
			{"Rayon Attaque", "intent_beam"},
			{"Lumifly", "creature_lumifly"},
			{"Shadowstalker", "creature_shadowstalker"},
			{"Burrower", "creature_burrower"},
			{"Flutterwing", "creature_flutterwing"},
			{"Fleeing Sprite", "creature_fleeing_sprite"},
			{"Specter", "creature_specter"},
			{"Echo Hound", "creature_echo_hound"},
			{"Moss Monkey", "creature_moss_monkey"},
			{"Stonewarden", "creature_stonewarden"},
			{"Dreamberry", "resource_dreamberry"},
			{"Moonstone", "resource_moonstone"},
			{"Whisper Herb", "resource_whispering_herb"},
			{"Shard", "resource_crystal_shard"},
			{"Moss Truffle", "resource_moss_truffle"},
			{"Void Bloom", "resource_void_bloom"},
			{"Echo Crystal", "resource_echo_crystal"},
			{"Sand Core", "resource_sand_core"},
			{"Shadow Essence", "resource_shadow_essence"},
		}

		for i := startIdx; i < endIdx; i++ {
			asset := allAssets[i]
			stages, hasStages := resourceStages[asset.key]
			if !hasStages {
				continue
			}

			localIdx := i - startIdx
			row := localIdx / itemsPerRow
			col := localIdx % itemsPerRow

			ax := wx + 30 + col*colWidth
			ay := wy + 60 + row*rowHeight

			// Zone de la flèche gauche
			leftArrowX := ax + 2
			leftArrowY := ay + 78
			if int(fx) >= leftArrowX && int(fx) <= leftArrowX+10 && int(fy) >= leftArrowY-10 && int(fy) <= leftArrowY+5 {
				idx := h.resourceStageIndex[asset.key]
				idx--
				if idx < 0 {
					idx = len(stages) - 1
				}
				h.resourceStageIndex[asset.key] = idx
				return true
			}

			// Zone de la flèche droite
			rightArrowX := ax + 78
			rightArrowY := ay + 78
			if int(fx) >= rightArrowX && int(fx) <= rightArrowX+10 && int(fy) >= rightArrowY-10 && int(fy) <= rightArrowY+5 {
				idx := h.resourceStageIndex[asset.key]
				idx++
				if idx >= len(stages) {
					idx = 0
				}
				h.resourceStageIndex[asset.key] = idx
				return true
			}
		}

		return true // Consomme tout clic dans la fenêtre
	}

	// Clic sur l'icône Menu (M) dans le portrait
	mxIcon := ui.PortraitX + ui.MenuIconRelativeX
	myIcon := ui.PortraitY + ui.MenuIconRelativeY
	fx, fy := float64(x), float64(y)
	if fx >= float64(mxIcon) && fx <= float64(mxIcon)+ui.MenuIconSize &&
		fy >= float64(myIcon) && fy <= float64(myIcon)+ui.MenuIconSize {
		// On ne peut pas appeler ReturnToMenu ici car HUD ne connaît pas app.
		// On laisse app.go gérer via ses propres callbacks (Input.OnExitToMenu déjà lié à l'icône M)
		// On retourne juste true pour dire que le clic a été consommé.
		return true
	}

	// Clic sur l'icône Fullscreen (F/W)
	fsxIcon := ui.PortraitX + ui.FullscreenIconRelativeX
	fsyIcon := ui.PortraitY + ui.FullscreenIconRelativeY
	if fx >= float64(fsxIcon) && fx <= float64(fsxIcon)+ui.FullscreenIconSize &&
		fy >= float64(fsyIcon) && fy <= float64(fsyIcon)+ui.FullscreenIconSize {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return true
	}

	if h.showDetails {
		winW, winH := 320, 450
		winX := (ui.ScreenWidth - winW) / 2
		winY := (ui.ScreenHeight - winH) / 2
		closeX := winX + winW - 30
		closeY := winY + 10

		if x >= closeX && x <= closeX+20 && y >= closeY && y <= closeY+20 {
			h.showDetails = false
			return true
		}
	}

	if h.showInventoryDetails {
		winW, winH := 320, 450
		winX := (ui.ScreenWidth - winW) / 2
		winY := (ui.ScreenHeight - winH) / 2
		closeX := winX + winW - 30
		closeY := winY + 10

		if x >= closeX && x <= closeX+20 && y >= closeY && y <= closeY+20 {
			h.showInventoryDetails = false
			return true
		}
	}

	// Gestion du clic sur l'inventaire
	if float64(x) >= ui.InventoryX && float64(x) <= ui.InventoryX+ui.InventoryW &&
		float64(y) >= ui.InventoryY && float64(y) <= ui.InventoryY+ui.InventoryH {

		// 1. Détection clic sur les slots (Zone slots commence à InventoryY + 40)
		slotZoneY := float64(ui.InventoryY + 40)
		if float64(y) >= slotZoneY && float64(y) <= slotZoneY+331 {
			localY := float64(y) - slotZoneY + h.world.Player.Inventory.ScrollOffset
			localX := float64(x) - float64(ui.InventoryX) - 5

			rowH := ui.LootSlotSize + ui.LootSlotPadding
			row := int(localY / rowH)
			col := int(localX / rowH)

			if col >= 0 && col < ui.LootSlotsPerRow {
				idx := row*ui.LootSlotsPerRow + col
				if idx >= 0 && idx < len(h.world.Player.Inventory.Items) {
					// Clic Gauche = Sélection pour USAGE (Bordure bleue)
					h.selectedLootIndex = idx
					h.confirmClearAll = false
					return true
				}
			}
		}

		// 2. Bouton Delete (X)
		dlx := float64(ui.InventoryX + ui.DeleteLootRelativeX)
		dly := float64(ui.InventoryY + ui.DeleteLootRelativeY)
		if float64(x) >= dlx && float64(x) <= dlx+float64(ui.DeleteLootSize) &&
			float64(y) >= dly && float64(y) <= dly+float64(ui.DeleteLootSize) {

			if len(h.selectedLoots) > 0 {
				// Suppression de toutes les tuiles sélectionnées
				// On trie les indices par ordre décroissant pour ne pas décaler les suivants
				indices := make([]int, 0, len(h.selectedLoots))
				for idx := range h.selectedLoots {
					indices = append(indices, idx)
				}
				sort.Sort(sort.Reverse(sort.IntSlice(indices)))

				for _, idx := range indices {
					// Vérifie si l'item est supprimable
					if idx < len(h.world.Player.Inventory.Items) && h.world.Player.Inventory.Items[idx].IsDeletable {
						_ = h.world.RemoveLootItem(idx)
					}
				}
				h.selectedLoots = make(map[int]bool)
			} else if !h.confirmClearAll {
				// Première étape : Sélectionner tout (Confirmation)
				// On ne sélectionne visuellement que ce qui est supprimable
				if len(h.world.Player.Inventory.Items) > 0 {
					h.confirmClearAll = true
				}
			} else {
				// Deuxième étape : Supprimer tout ce qui est supprimable
				// On parcourt à l'envers pour garder les indices valides
				for i := len(h.world.Player.Inventory.Items) - 1; i >= 0; i-- {
					if h.world.Player.Inventory.Items[i].IsDeletable {
						_ = h.world.RemoveLootItem(i)
					}
				}
				h.confirmClearAll = false
			}
			return true
		}
	} else {
		// Clic en dehors de l'inventaire désélectionne tout
		h.selectedLoots = make(map[int]bool)
		h.selectedLootIndex = -1
		h.confirmClearAll = false
	}

	return false
}

// HandleRightClick gère les clics droits (Sélection pour suppression)
func (h *HUD) HandleRightClick(x, y int) bool {
	if float64(x) >= ui.InventoryX && float64(x) <= ui.InventoryX+ui.InventoryW &&
		float64(y) >= ui.InventoryY && float64(y) <= ui.InventoryY+ui.InventoryH {

		// Détection clic sur les slots pour la suppression (Rouge)
		slotZoneY := float64(ui.InventoryY + 40)
		if float64(y) >= slotZoneY && float64(y) <= slotZoneY+331 {
			localY := float64(y) - slotZoneY + h.world.Player.Inventory.ScrollOffset
			localX := float64(x) - float64(ui.InventoryX) - 5

			rowH := ui.LootSlotSize + ui.LootSlotPadding
			row := int(localY / rowH)
			col := int(localX / rowH)

			if col >= 0 && col < ui.LootSlotsPerRow {
				idx := row*ui.LootSlotsPerRow + col
				if idx >= 0 && idx < len(h.world.Player.Inventory.Items) {
					// Clic Droit = Toggle sélection pour SUPPRESSION (Bordure rouge)
					if h.selectedLoots[idx] {
						delete(h.selectedLoots, idx)
					} else {
						h.selectedLoots[idx] = true
					}
					h.confirmClearAll = false
					return true
				}
			}
		}

		// Clic droit ailleurs dans l'inventaire = Tout désélectionner
		h.selectedLoots = make(map[int]bool)
		h.selectedLootIndex = -1
		h.confirmClearAll = false
		return true
	}
	return false
}

// HandleScroll gère le scroll sur l'inventaire
func (h *HUD) HandleScroll(x, y int) {
	if float64(x) >= ui.InventoryX && float64(x) <= ui.InventoryX+ui.InventoryW &&
		float64(y) >= ui.InventoryY && float64(y) <= ui.InventoryY+ui.InventoryH {
		_, dy := ebiten.Wheel()
		if dy != 0 {
			inv := &h.world.Player.Inventory
			// Défilement de 20 pixels par cran de molette
			inv.ScrollOffset -= dy * 20

			// Calcul de la hauteur totale du contenu
			totalRows := float64((inv.MaxSize + ui.LootSlotsPerRow - 1) / ui.LootSlotsPerRow)
			rowH := ui.LootSlotSize + ui.LootSlotPadding
			totalHeight := totalRows * rowH
			viewportHeight := 331.0

			// Clamp scroll
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
	}
}
