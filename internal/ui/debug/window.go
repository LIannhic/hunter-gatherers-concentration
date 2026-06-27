package debug

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/textutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type DebugWindow struct {
	world *domain.World
	x, y  float32
	w, h  float32

	// Caches pour éviter les allocations/tris par frame et garantir la cohérence des clics
	sortedEntities []string
	sortedShaders  []string
}

func NewDebugWindow(world *domain.World) *DebugWindow {
	dw := &DebugWindow{
		world: world,
		x:     100,
		y:     50,
		w:     1080,
		h:     620,
	}
	dw.initCaches()
	return dw
}

func (dw *DebugWindow) initCaches() {
	dw.sortedEntities = []string{
		// Creatures
		"lumifly", "shadowstalker", "burrower", "specter", "echo_hound", "fleeing_sprite", "moss_monkey", "stonewarden", "flutterwing",
		// Resources
		"dreamberry", "moonstone", "whispering_herb", "crystal_shard", "moss_truffle", "void_bloom", "echo_crystal", "sand_core",
		// Special
		"trap", "start_portal", "finish_portal", "dolmen", "obelisk", "portable_portal",
	}
	sort.Strings(dw.sortedEntities)

	// Shaders purement environnementaux (Biome)
	dw.sortedShaders = []string{"cave", "heat", "rain", "wave"}
	sort.Strings(dw.sortedShaders)
}

func (dw *DebugWindow) Render(screen *ebiten.Image) {
	if !dw.world.Debug.Visible {
		return
	}

	// Fond
	vector.DrawFilledRect(screen, dw.x, dw.y, dw.w, dw.h, color.RGBA{20, 20, 25, 240}, true)
	vector.StrokeRect(screen, dw.x, dw.y, dw.w, dw.h, 2, color.RGBA{200, 200, 255, 255}, true)

	// Titre
	textutil.Draw(screen, "DEBUG CONSOLE (F12)", int(dw.x)+20, int(dw.y)+30, color.RGBA{0, 255, 255, 255})

	// Bouton Fermer
	closeX, closeY := dw.x+dw.w-30, dw.y+10
	vector.DrawFilledRect(screen, closeX, closeY, 20, 20, color.RGBA{200, 50, 50, 255}, true)
	textutil.Draw(screen, "X", int(closeX)+6, int(closeY)+15, color.White)

	// Bouton Réinitialiser
	resetX, resetY := dw.x+dw.w-150, dw.y+10
	vector.DrawFilledRect(screen, resetX, resetY, 110, 20, color.RGBA{100, 100, 150, 255}, true)
	textutil.Draw(screen, "RESET DEFAULTS", int(resetX)+5, int(resetY)+15, color.White)

	dw.renderStats(screen)
	dw.renderDifficulty(screen)
	dw.renderCreatures(screen)
	dw.renderShaders(screen)
	dw.renderImpairments(screen)
}

func (dw *DebugWindow) renderStats(screen *ebiten.Image) {
	startX := dw.x + 20
	startY := dw.y + 70
	textutil.Draw(screen, "PLAYER STATS", int(startX), int(startY), color.RGBA{255, 255, 0, 255})

	stats := []struct {
		label string
		val   *int
		max   *int
	}{
		{"Health", &dw.world.Player.Stats.Health, &dw.world.Player.Stats.MaxHealth},
		{"Mana", &dw.world.Player.Stats.Mana, &dw.world.Player.Stats.MaxMana},
		{"Sanity", &dw.world.Player.Stats.Sanity, &dw.world.Player.Stats.MaxSanity},
	}

	for i, s := range stats {
		y := startY + 30 + float32(i*40)
		textutil.Draw(screen, fmt.Sprintf("%-7s: %d / %d", s.label, *s.val, *s.max), int(startX), int(y)+15, color.White)

		// Boutons pour Valeur
		dw.drawButton(screen, startX+150, y, "-", "v-1")
		dw.drawButton(screen, startX+180, y, "+", "v+1")
		dw.drawButton(screen, startX+210, y, "-10", "v-10")
		dw.drawButton(screen, startX+250, y, "+10", "v+10")

		// Boutons pour Max
		dw.drawButton(screen, startX+310, y, "M-10", "m-10")
		dw.drawButton(screen, startX+350, y, "M+10", "m+10")
	}
}

func (dw *DebugWindow) renderDifficulty(screen *ebiten.Image) {
	startX := dw.x + 450
	startY := dw.y + 70
	textutil.Draw(screen, "DIFFICULTY & RULES", int(startX), int(startY), color.RGBA{255, 255, 0, 255})

	settings := &dw.world.Difficulty
	if dw.world.Debug.OverrideDifficulty {
		settings = &dw.world.Debug.Difficulty
	}

	dw.drawCheckbox(screen, startX, startY+30, "Override Game Settings", dw.world.Debug.OverrideDifficulty)

	y := startY + 70
	textutil.Draw(screen, fmt.Sprintf("Turn Timer: %.1fs", settings.TurnTimerDuration), int(startX), int(y)+15, color.White)
	dw.drawButton(screen, startX+150, y, "-", "timer-")
	dw.drawButton(screen, startX+180, y, "+", "timer+")

	y += 30
	textutil.Draw(screen, fmt.Sprintf("Preview Dur: %.1fs", settings.PreviewDuration), int(startX), int(y)+15, color.White)
	dw.drawButton(screen, startX+150, y, "-", "prev-")
	dw.drawButton(screen, startX+180, y, "+", "prev+")

	y += 30
	textutil.Draw(screen, fmt.Sprintf("Nav Threshold: %.0f%%", settings.NavThreshold*100), int(startX), int(y)+15, color.White)
	dw.drawButton(screen, startX+150, y, "-", "nav-")
	dw.drawButton(screen, startX+180, y, "+", "nav+")

	y += 30
	textutil.Draw(screen, fmt.Sprintf("Msg Speed: %.1f", dw.world.Debug.MessageSpeed), int(startX), int(y)+15, color.White)
	dw.drawButton(screen, startX+150, y, "-", "msgspd-")
	dw.drawButton(screen, startX+180, y, "+", "msgspd+")
}

func (dw *DebugWindow) renderCreatures(screen *ebiten.Image) {
	startX := dw.x + 20
	startY := dw.y + 250
	textutil.Draw(screen, "ALLOWED ENTITIES (for random spawn)", int(startX), int(startY), color.RGBA{255, 255, 0, 255})

	for i, e := range dw.sortedEntities {
		row := i / 4
		col := i % 4
		cx := startX + float32(col*185)
		cy := startY + 30 + float32(row*30)

		dw.drawCheckbox(screen, cx, cy, e, dw.world.Debug.AllowedCreatures[e])
	}
}

func (dw *DebugWindow) renderShaders(screen *ebiten.Image) {
	startX := dw.x + 800
	startY := dw.y + 70
	textutil.Draw(screen, "ENVIRONMENTAL SHADERS", int(startX), int(startY), color.RGBA{255, 255, 0, 255})

	for i, s := range dw.sortedShaders {
		cy := startY + 30 + float32(i*30)
		dw.drawCheckbox(screen, startX, cy, s, dw.world.Debug.ActiveShaders[s])
	}
}

func (dw *DebugWindow) renderImpairments(screen *ebiten.Image) {
	startX := dw.x + 800
	startY := dw.y + 250
	textutil.Draw(screen, "INFLICTIONS", int(startX), int(startY), color.RGBA{255, 255, 0, 255})

	p := dw.world.Player
	if p == nil {
		return
	}

	dw.drawCheckbox(screen, startX, startY+30, "Blur (Shadowstalker)", dw.world.Debug.ActiveShaders["blur"])
	dw.drawCheckbox(screen, startX, startY+60, "Bubble (Lumifly)", dw.world.Debug.ActiveShaders["bubble"])
	dw.drawCheckbox(screen, startX, startY+90, "Aphasia (Echo Hound)", p.AphasiaTurns > 0)
	dw.drawCheckbox(screen, startX, startY+120, "Ataxia (Burrower)", p.AtaxiaTurns > 0)
	dw.drawCheckbox(screen, startX, startY+150, "Agnosia (Moss Monkey)", p.AgnosiaTurns > 0)
	dw.drawCheckbox(screen, startX, startY+180, "Amnesia (Specter)", p.AmnesiaTurns > 0)
}

func (dw *DebugWindow) drawButton(screen *ebiten.Image, x, y float32, label, id string) {
	w := float32(len(label)*8 + 10)
	h := float32(20)
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{60, 60, 80, 255}, true)
	vector.StrokeRect(screen, x, y, w, h, 1, color.RGBA{150, 150, 200, 255}, true)
	textutil.Draw(screen, label, int(x)+5, int(y)+15, color.White)
}

func (dw *DebugWindow) drawCheckbox(screen *ebiten.Image, x, y float32, label string, checked bool) {
	size := float32(16)
	vector.StrokeRect(screen, x, y, size, size, 1, color.White, true)
	if checked {
		vector.DrawFilledRect(screen, x+3, y+3, size-6, size-6, color.RGBA{0, 255, 0, 255}, true)
	}
	textutil.Draw(screen, label, int(x)+25, int(y)+13, color.White)
}

func (dw *DebugWindow) HandleClick(mx, my int) bool {
	if !dw.world.Debug.Visible {
		return false
	}

	fx, fy := float32(mx), float32(my)
	if fx < dw.x || fx > dw.x+dw.w || fy < dw.y || fy > dw.y+dw.h {
		return false
	}

	// Bouton Fermer
	if fx >= dw.x+dw.w-30 && fx <= dw.x+dw.w-10 && fy >= dw.y+10 && fy <= dw.y+30 {
		dw.world.Debug.Visible = false
		return true
	}

	// Bouton RESET
	if fx >= dw.x+dw.w-150 && fx <= dw.x+dw.w-40 && fy >= dw.y+10 && fy <= dw.y+30 {
		dw.ResetDefaults()
		return true
	}

	// Stats
	dw.handleClickStats(fx, fy)
	// Difficulty
	dw.handleClickDifficulty(fx, fy)
	// Creatures
	dw.handleClickCreatures(fx, fy)
	// Shaders
	dw.handleClickShaders(fx, fy)
	// Impairments
	dw.handleClickImpairments(fx, fy)

	return true
}

func (dw *DebugWindow) handleClickStats(mx, my float32) {
	startX := dw.x + 20
	startY := dw.y + 70

	stats := []struct {
		val *int
		max *int
	}{
		{&dw.world.Player.Stats.Health, &dw.world.Player.Stats.MaxHealth},
		{&dw.world.Player.Stats.Mana, &dw.world.Player.Stats.MaxMana},
		{&dw.world.Player.Stats.Sanity, &dw.world.Player.Stats.MaxSanity},
	}

	for i, s := range stats {
		y := startY + 30 + float32(i*40)

		// v-1
		if dw.isInside(mx, my, startX+150, y, 20, 20) { *s.val-- }
		// v+1
		if dw.isInside(mx, my, startX+180, y, 20, 20) { *s.val++ }
		// v-10
		if dw.isInside(mx, my, startX+210, y, 35, 20) { *s.val -= 10 }
		// v+10
		if dw.isInside(mx, my, startX+250, y, 35, 20) { *s.val += 10 }
		// m-10
		if dw.isInside(mx, my, startX+310, y, 35, 20) { *s.max -= 10 }
		// m+10
		if dw.isInside(mx, my, startX+350, y, 35, 20) { *s.max += 10 }

		if *s.val < 0 { *s.val = 0 }
		if *s.val > *s.max { *s.val = *s.max }
	}
}

func (dw *DebugWindow) handleClickDifficulty(mx, my float32) {
	startX := dw.x + 450
	startY := dw.y + 70

	// Checkbox Override
	if dw.isInside(mx, my, startX, startY+30, 16, 16) {
		dw.world.Debug.OverrideDifficulty = !dw.world.Debug.OverrideDifficulty
		if dw.world.Debug.OverrideDifficulty {
			dw.world.Debug.Difficulty = dw.world.Difficulty
		}
		return
	}

	if !dw.world.Debug.OverrideDifficulty {
		return
	}

	settings := &dw.world.Debug.Difficulty
	y := startY + 70
	if dw.isInside(mx, my, startX+150, y, 20, 20) { settings.TurnTimerDuration -= 0.5 }
	if dw.isInside(mx, my, startX+180, y, 20, 20) { settings.TurnTimerDuration += 0.5 }

	y += 30
	if dw.isInside(mx, my, startX+150, y, 20, 20) { settings.PreviewDuration -= 0.5 }
	if dw.isInside(mx, my, startX+180, y, 20, 20) { settings.PreviewDuration += 0.5 }

	y += 30
	if dw.isInside(mx, my, startX+150, y, 20, 20) { settings.NavThreshold -= 0.05 }
	if dw.isInside(mx, my, startX+180, y, 20, 20) { settings.NavThreshold += 0.05 }

	if settings.TurnTimerDuration < 1 { settings.TurnTimerDuration = 1 }
	if settings.PreviewDuration < 0 { settings.PreviewDuration = 0 }
	if settings.NavThreshold < 0 { settings.NavThreshold = 0 }
	if settings.NavThreshold > 1 { settings.NavThreshold = 1 }

	y += 30
	if dw.isInside(mx, my, startX+150, y, 20, 20) { dw.world.Debug.MessageSpeed -= 0.2 }
	if dw.isInside(mx, my, startX+180, y, 20, 20) { dw.world.Debug.MessageSpeed += 0.2 }
	if dw.world.Debug.MessageSpeed < 0.2 { dw.world.Debug.MessageSpeed = 0.2 }
	if dw.world.Debug.MessageSpeed > 5.0 { dw.world.Debug.MessageSpeed = 5.0 }
}

func (dw *DebugWindow) handleClickCreatures(mx, my float32) {
	startX := dw.x + 20
	startY := dw.y + 250

	for i, e := range dw.sortedEntities {
		row := i / 4
		col := i % 4
		cx := startX + float32(col*185)
		cy := startY + 30 + float32(row*30)

		// Clique sur le checkbox OU son label (cx à cx+180)
		if dw.isInside(mx, my, cx, cy, 180, 20) {
			dw.world.Debug.AllowedCreatures[e] = !dw.world.Debug.AllowedCreatures[e]
		}
	}
}

func (dw *DebugWindow) handleClickShaders(mx, my float32) {
	startX := dw.x + 800
	startY := dw.y + 70

	for i, s := range dw.sortedShaders {
		cy := startY + 30 + float32(i*30)
		// Clique sur le checkbox OU son label
		if dw.isInside(mx, my, startX, cy, 200, 20) {
			dw.world.Debug.ActiveShaders[s] = !dw.world.Debug.ActiveShaders[s]
			// Sync with player visual effects for immediate feedback
			if dw.world.Player != nil {
				if dw.world.Debug.ActiveShaders[s] {
					dw.world.Player.VisualEffects[s] = 999
				} else {
					dw.world.Player.VisualEffects[s] = 0
				}
			}
		}
	}
}

func (dw *DebugWindow) handleClickImpairments(mx, my float32) {
	startX := dw.x + 800
	startY := dw.y + 250
	p := dw.world.Player
	if p == nil {
		return
	}

	// Blur
	if dw.isInside(mx, my, startX, startY+30, 200, 20) {
		dw.world.Debug.ActiveShaders["blur"] = !dw.world.Debug.ActiveShaders["blur"]
		if dw.world.Player != nil {
			if dw.world.Debug.ActiveShaders["blur"] {
				dw.world.Player.VisualEffects["blur"] = 999
			} else {
				dw.world.Player.VisualEffects["blur"] = 0
			}
		}
	}
	// Bubble
	if dw.isInside(mx, my, startX, startY+60, 200, 20) {
		dw.world.Debug.ActiveShaders["bubble"] = !dw.world.Debug.ActiveShaders["bubble"]
		if dw.world.Player != nil {
			if dw.world.Debug.ActiveShaders["bubble"] {
				dw.world.Player.VisualEffects["bubble"] = 999
			} else {
				dw.world.Player.VisualEffects["bubble"] = 0
			}
		}
	}
	// Aphasia
	if dw.isInside(mx, my, startX, startY+90, 200, 20) {
		if p.AphasiaTurns > 0 { p.AphasiaTurns = 0 } else { p.AphasiaTurns = 10 }
	}
	// Ataxia
	if dw.isInside(mx, my, startX, startY+120, 200, 20) {
		if p.AtaxiaTurns > 0 { p.AtaxiaTurns = 0 } else { p.AtaxiaTurns = 10 }
	}
	// Agnosia
	if dw.isInside(mx, my, startX, startY+150, 200, 20) {
		if p.AgnosiaTurns > 0 { p.AgnosiaTurns = 0 } else { p.AgnosiaTurns = 10 }
	}
	// Amnesia
	if dw.isInside(mx, my, startX, startY+180, 200, 20) {
		if p.AmnesiaTurns > 0 { p.AmnesiaTurns = 0 } else { p.AmnesiaTurns = 10 }
	}
}

func (dw *DebugWindow) isInside(mx, my, x, y, w, h float32) bool {
	return mx >= x && mx <= x+w && my >= y && my <= y+h
}

func (dw *DebugWindow) ResetDefaults() {
	dw.world.Debug.OverrideDifficulty = false
	dw.world.Debug.ActiveShaders = make(map[string]bool)
	dw.world.Debug.MessageSpeed = 1.0
	if dw.world.Player != nil {
		dw.world.Player.AphasiaTurns = 0
		dw.world.Player.AtaxiaTurns = 0
		dw.world.Player.AgnosiaTurns = 0
		dw.world.Player.AmnesiaTurns = 0
		dw.world.Player.VisualEffects = make(map[string]int)
	}
	dw.world.Debug.AllowedCreatures = map[string]bool{
		"lumifly":         true,
		"shadowstalker":   true,
		"burrower":        true,
		"specter":         true,
		"echo_hound":      true,
		"fleeing_sprite":  true,
		"moss_monkey":     true,
		"stonewarden":     true,
		"flutterwing":     true,
		"dreamberry":      true,
		"moonstone":       true,
		"whispering_herb": true,
		"crystal_shard":   true,
		"moss_truffle":    true,
		"void_bloom":      true,
		"echo_crystal":    true,
		"sand_core":       true,
		"trap":            true,
		"start_portal":    true,
		"finish_portal":   true,
		"dolmen":          true,
		"obelisk":         true,
		"portable_portal": true,
	}
}
