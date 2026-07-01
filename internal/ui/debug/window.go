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
	shaderToBiome  map[string]string // Correspondance shader name -> biome name
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
	// Note : vortex (effet de tuile butin portail portable) n'est pas un shader d'environnement
	dw.sortedShaders = []string{"cave", "heat", "rain", "wave"}
	dw.shaderToBiome = map[string]string{
		"cave": "cave",
		"heat": "desert",
		"rain": "forest",
		"wave": "swamp",
	}
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

	dw.drawCheckbox(screen, startX, startY+30, "Override Game Settings", dw.world.Debug.OverrideDifficulty, false)

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

		dw.drawCheckbox(screen, cx, cy, e, dw.world.Debug.AllowedCreatures[e], false)
	}
}

func (dw *DebugWindow) renderShaders(screen *ebiten.Image) {
	startX := dw.x + 800
	startY := dw.y + 70
	textutil.Draw(screen, "ENVIRONMENTAL SHADERS", int(startX), int(startY), color.RGBA{255, 255, 0, 255})
	textutil.Draw(screen, "Active", int(startX)+20, int(startY)+17, color.RGBA{0, 200, 0, 255})
	textutil.Draw(screen, "Block", int(startX)+200, int(startY)+17, color.RGBA{255, 80, 80, 255})

	grid, _ := dw.world.GetCurrentGrid()
	biome := ""
	if grid != nil {
		biome = string(grid.Biome)
	}

	for i, s := range dw.sortedShaders {
		cy := startY + 30 + float32(i*30)
		isBiomeActive := biome == dw.shaderToBiome[s]
		isForceActive := dw.world.Debug.ActiveShaders[s]
		isDisabled := dw.world.Debug.DisabledShaders[s]
		isActive := (isBiomeActive || isForceActive) && !isDisabled
		dw.drawCheckbox(screen, startX, cy, s, isActive, !isBiomeActive && !isForceActive)
		dw.drawCheckbox(screen, startX+200, cy, "", isDisabled, false)
	}
}

func (dw *DebugWindow) renderImpairments(screen *ebiten.Image) {
	startX := dw.x + 800
	startY := dw.y + 250
	textutil.Draw(screen, "INFLICTIONS", int(startX), int(startY), color.RGBA{255, 255, 0, 255})
	textutil.Draw(screen, "Active", int(startX)+20, int(startY)+17, color.RGBA{0, 200, 0, 255})
	textutil.Draw(screen, "Block", int(startX)+200, int(startY)+17, color.RGBA{255, 80, 80, 255})

	p := dw.world.Player
	if p == nil {
		return
	}

	type effectInfo struct {
		key      string
		yOff     float32
		isActive bool
	}

	effects := []effectInfo{
		{"blur", 30, p.VisualEffects["blur"] > 0},
		{"bubble", 60, p.VisualEffects["bubble"] > 0},
		{"aphasia", 90, p.AphasiaTurns > 0},
		{"ataxia", 120, p.AtaxiaTurns > 0},
		{"agnosia", 150, p.AgnosiaTurns > 0},
		{"amnesia", 180, p.AmnesiaTurns > 0},
		{"vertige", 210, p.VisualEffects["vertige"] > 0},
		{"invert", 240, p.VisualEffects["invert"] > 0},
	}

	for _, e := range effects {
		isDisabled := dw.world.Debug.DisabledEffects[e.key]
		dw.drawCheckbox(screen, startX, startY+e.yOff, e.key, e.isActive && !isDisabled, false)
		dw.drawCheckbox(screen, startX+200, startY+e.yOff, "", isDisabled, false)
	}
}

func (dw *DebugWindow) drawButton(screen *ebiten.Image, x, y float32, label, id string) {
	w := float32(len(label)*8 + 10)
	h := float32(20)
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{60, 60, 80, 255}, true)
	vector.StrokeRect(screen, x, y, w, h, 1, color.RGBA{150, 150, 200, 255}, true)
	textutil.Draw(screen, label, int(x)+5, int(y)+15, color.White)
}

func (dw *DebugWindow) drawCheckbox(screen *ebiten.Image, x, y float32, label string, checked bool, disabled bool) {
	size := float32(16)
	borderColor := color.RGBA{100, 100, 100, 255}
	labelColor := color.RGBA{120, 120, 120, 255}
	if !disabled {
		borderColor = color.RGBA{255, 255, 255, 255}
		labelColor = color.RGBA{255, 255, 255, 255}
	}
	vector.StrokeRect(screen, x, y, size, size, 1, borderColor, true)
	if checked {
		vector.DrawFilledRect(screen, x+3, y+3, size-6, size-6, color.RGBA{0, 255, 0, 255}, true)
	}
	textutil.Draw(screen, label, int(x)+25, int(y)+13, labelColor)
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
		if dw.isInside(mx, my, startX+200, cy, 20, 20) {
			if dw.world.Debug.DisabledShaders[s] {
				delete(dw.world.Debug.DisabledShaders, s)
				fmt.Printf("[DEBUG] Shader débloqué : %s\n", s)
			} else {
				dw.world.Debug.DisabledShaders[s] = true
				fmt.Printf("[DEBUG] Shader bloqué : %s\n", s)
			}
		} else if dw.isInside(mx, my, startX, cy, 200, 20) {
			if dw.world.Debug.ActiveShaders[s] {
				delete(dw.world.Debug.ActiveShaders, s)
				fmt.Printf("[DEBUG] Shader désactivé : %s\n", s)
			} else {
				dw.world.Debug.ActiveShaders[s] = true
				fmt.Printf("[DEBUG] Shader activé : %s\n", s)
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

	type effectDef struct {
		key    string
		yOff   float32
		active func()
		clear  func()
	}

	effects := []effectDef{
		{"blur", 30, func() { p.VisualEffects["blur"] = 999 }, func() { p.VisualEffects["blur"] = 0 }},
		{"bubble", 60, func() { p.VisualEffects["bubble"] = 999 }, func() { p.VisualEffects["bubble"] = 0 }},
		{"aphasia", 90, func() { p.AphasiaTurns = 10 }, func() { p.AphasiaTurns = 0 }},
		{"ataxia", 120, func() { p.AtaxiaTurns = 10 }, func() { p.AtaxiaTurns = 0 }},
		{"agnosia", 150, func() { p.AgnosiaTurns = 10 }, func() { p.AgnosiaTurns = 0 }},
		{"amnesia", 180, func() { p.AmnesiaTurns = 10 }, func() { p.AmnesiaTurns = 0 }},
		{"vertige", 210, func() { p.VisualEffects["vertige"] = 999 }, func() { p.VisualEffects["vertige"] = 0 }},
		{"invert", 240, func() { p.VisualEffects["invert"] = 999 }, func() { p.VisualEffects["invert"] = 0 }},
	}

	for _, e := range effects {
		if dw.isInside(mx, my, startX+200, startY+e.yOff, 20, 20) {
			if dw.world.Debug.DisabledEffects[e.key] {
				delete(dw.world.Debug.DisabledEffects, e.key)
				fmt.Printf("[DEBUG] Effet débloqué : %s\n", e.key)
			} else {
				dw.world.Debug.DisabledEffects[e.key] = true
				e.clear()
				fmt.Printf("[DEBUG] Effet bloqué : %s\n", e.key)
			}
		} else if dw.isInside(mx, my, startX, startY+e.yOff, 200, 20) {
			isActive := e.key == "blur" && p.VisualEffects["blur"] > 0 ||
				e.key == "bubble" && p.VisualEffects["bubble"] > 0 ||
				e.key == "aphasia" && p.AphasiaTurns > 0 ||
				e.key == "ataxia" && p.AtaxiaTurns > 0 ||
				e.key == "agnosia" && p.AgnosiaTurns > 0 ||
				e.key == "amnesia" && p.AmnesiaTurns > 0 ||
				e.key == "vertige" && p.VisualEffects["vertige"] > 0 ||
				e.key == "invert" && p.VisualEffects["invert"] > 0

			if isActive {
				e.clear()
				fmt.Printf("[DEBUG] Effet désactivé : %s\n", e.key)
			} else {
				e.active()
				fmt.Printf("[DEBUG] Effet activé : %s\n", e.key)
			}
		}
	}
}

func (dw *DebugWindow) isInside(mx, my, x, y, w, h float32) bool {
	return mx >= x && mx <= x+w && my >= y && my <= y+h
}

func (dw *DebugWindow) ResetDefaults() {
	dw.world.Debug.OverrideDifficulty = false
	dw.world.Debug.ActiveShaders = make(map[string]bool)
	dw.world.Debug.DisabledShaders = make(map[string]bool)
	dw.world.Debug.DisabledEffects = make(map[string]bool)
	dw.world.Debug.MessageSpeed = 1.0
	if dw.world.Player != nil {
		dw.world.Player.AphasiaTurns = 0
		dw.world.Player.AtaxiaTurns = 0
		dw.world.Player.AgnosiaTurns = 0
		dw.world.Player.AmnesiaTurns = 0
		dw.world.Player.VertigoTurns = 0
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
