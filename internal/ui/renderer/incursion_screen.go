package renderer

import (
	"fmt"
	"image/color"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// IncursionScreen affiche l'UI en jeu (mode incursion) pour une résolution 1280x720
type IncursionScreen struct{}

// NewIncursionScreen crée un nouvel écran d'incursion
func NewIncursionScreen() *IncursionScreen {
	return &IncursionScreen{}
}

// Render dessine l'ensemble de l'UI en jeu
func (is *IncursionScreen) Render(screen *ebiten.Image, world *domain.World, boardRenderer *BoardRenderer) {
	// --- PANNEAU GAUCHE HAUT : PORTRAIT ---
	is.drawPanel(screen, 10, 10, 270, 270, color.RGBA{40, 40, 40, 255}, color.RGBA{100, 100, 100, 255})
	// Menu icon
	is.drawPanel(screen, 15, 15, 43.75, 43.75, color.RGBA{60, 60, 60, 255}, color.RGBA{150, 150, 150, 255})
	mx := float64(15) + 43.75/2 - 4
	my := float64(15) + 43.75/2 + 4
	text.Draw(screen, "M", basicfont.Face7x13, int(mx), int(my), color.White)
	// Portrait placeholder
	is.drawPanel(screen, 15, 65, 260, 210, color.RGBA{30, 30, 30, 255}, color.RGBA{80, 80, 80, 255})
	text.Draw(screen, "PORTRAIT", basicfont.Face7x13, int(10+270/2-30), int(10+270/2+4), color.RGBA{150, 150, 150, 255})

	// --- PANNEAU GAUCHE BAS : INVENTAIRE ---
	is.drawPanel(screen, 10, 290, 270, 420, color.RGBA{40, 40, 40, 255}, color.RGBA{100, 100, 100, 255})
	is.drawLootGrid(screen, 10, 290)
	// Loot counter
	is.drawPanel(screen, 15, 661, 43.75, 43.75, color.RGBA{60, 60, 60, 255}, color.RGBA{150, 150, 150, 255})
	cx := float64(15) + 43.75/2 - 4
	cy := float64(661) + 43.75/2 + 4
	text.Draw(screen, "0", basicfont.Face7x13, int(cx), int(cy), color.White)
	// Delete loot icon
	is.drawPanel(screen, 231, 661, 43.75, 43.75, color.RGBA{80, 40, 40, 255}, color.RGBA{180, 80, 80, 255})
	dx := float64(231) + 43.75/2 - 3
	dy := float64(661) + 43.75/2 + 4
	text.Draw(screen, "X", basicfont.Face7x13, int(dx), int(dy), color.White)

	// --- PLAYMAT (CENTRE) ---
	biomeColor := is.biomeColor(world)
	is.drawPanel(screen, 290, 10, 700, 700, biomeColor, color.RGBA{80, 80, 80, 255})

	// Boutons d'action x4
	actions := []string{"Action", "Skip", "Match", "Other"}
	btnPositions := [][2]float64{{300, 20}, {760, 20}, {300, 661}, {760, 661}}
	for i, label := range actions {
		x, y := btnPositions[i][0], btnPositions[i][1]
		is.drawButton(screen, x, y, 219.67, 39.17, label)
	}

	// Sorties
	is.drawExit(screen, 552.5, 10, 175, 87.5, "N")
	is.drawExit(screen, 902.5, 272.5, 87.5, 175, "E")
	is.drawExit(screen, 552.5, 622.5, 175, 87.5, "S")
	is.drawExit(screen, 290, 272.5, 87.5, 175, "W")

	// Plateau de jeu (tuiles + entités)
	boardRenderer.Render(screen, world)

	// --- PANNEAU DROIT HAUT : JAUGES ---
	is.drawPanel(screen, 1000, 10, 270, 420, color.RGBA{40, 40, 40, 255}, color.RGBA{100, 100, 100, 255})
	p := world.Player
	is.drawGauge(screen, 1010, 20, 76.67, 400, p.Stats.Health, p.Stats.MaxHealth, "HP", color.RGBA{200, 60, 60, 255})
	is.drawGauge(screen, 1097.65, 20, 76.67, 400, p.Stats.Mana, p.Stats.MaxMana, "MP", color.RGBA{60, 100, 200, 255})
	is.drawGauge(screen, 1183.32, 20, 76.67, 400, p.Stats.Sanity, p.Stats.MaxSanity, "SN", color.RGBA{180, 80, 180, 255})

	// --- PANNEAU DROIT BAS : MINIMAP ---
	is.drawPanel(screen, 1000, 440, 270, 270, color.RGBA{20, 20, 20, 255}, color.RGBA{100, 100, 100, 255})
	is.drawMinimap(screen, 1000, 440, 270, 270, world)
}

func (is *IncursionScreen) drawPanel(screen *ebiten.Image, x, y, w, h float64, fill, stroke color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), fill, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1, stroke, true)
}

func (is *IncursionScreen) drawButton(screen *ebiten.Image, x, y, w, h float64, label string) {
	is.drawPanel(screen, x, y, w, h, color.RGBA{50, 50, 50, 255}, color.RGBA{120, 120, 120, 255})
	// Texte centré approximativement
	textX := int(x + 10)
	textY := int(y + 8.65 + 14)
	text.Draw(screen, label, basicfont.Face7x13, textX, textY, color.White)
	// Icon placeholder
	iconX := x + 180.5
	iconY := y + 5
	is.drawPanel(screen, iconX, iconY, 29.17, 29.17, color.RGBA{80, 80, 80, 255}, color.RGBA{150, 150, 150, 255})
}

func (is *IncursionScreen) drawExit(screen *ebiten.Image, x, y, w, h float64, label string) {
	is.drawPanel(screen, x, y, w, h, color.RGBA{60, 60, 60, 255}, color.RGBA{120, 120, 120, 255})
	cx := int(x + w/2 - 3)
	cy := int(y + h/2 + 4)
	text.Draw(screen, label, basicfont.Face7x13, cx, cy, color.RGBA{200, 200, 200, 255})
}

func (is *IncursionScreen) drawLootGrid(screen *ebiten.Image, panelX, panelY float64) {
	slotSize := 87.5
	padding := 1.875
	cols := 3
	maxRows := 4
	startX := panelX + padding
	startY := panelY + padding
	for row := 0; row < maxRows; row++ {
		for col := 0; col < cols; col++ {
			x := startX + float64(col)*(slotSize+padding)
			y := startY + float64(row)*(slotSize+padding)
			is.drawPanel(screen, x, y, slotSize, slotSize, color.RGBA{50, 50, 50, 255}, color.RGBA{90, 90, 90, 255})
		}
	}
}

func (is *IncursionScreen) drawGauge(screen *ebiten.Image, x, y, w, h float64, value, max int, label string, fillColor color.Color) {
	// Fond
	is.drawPanel(screen, x, y, w, h, color.RGBA{30, 30, 30, 255}, color.RGBA{80, 80, 80, 255})

	// Calcul de la hauteur remplie
	var fillHeight float64
	if max <= 0 {
		fillHeight = 0
	} else {
		maxGaugeHeight := h
		if max <= 200 {
			maxGaugeHeight = float64(max) * 2
			if maxGaugeHeight > h {
				maxGaugeHeight = h
			}
		}
		fillHeight = float64(value) / float64(max) * maxGaugeHeight
	}
	if fillHeight < 0 {
		fillHeight = 0
	}
	if fillHeight > h {
		fillHeight = h
	}

	// Remplissage depuis le bas
	if fillHeight > 0 {
		vector.DrawFilledRect(screen, float32(x), float32(y+h-fillHeight), float32(w), float32(fillHeight), fillColor, true)
	}

	// Label
	text.Draw(screen, label, basicfont.Face7x13, int(x+5), int(y+14), color.White)
	// Valeur
	valStr := fmt.Sprintf("%d", value)
	text.Draw(screen, valStr, basicfont.Face7x13, int(x+5), int(y+h-10), color.RGBA{220, 220, 220, 255})
}

func (is *IncursionScreen) drawMinimap(screen *ebiten.Image, x, y, w, h float64, world *domain.World) {
	// Fond déjà dessiné
	padding := 10.0
	mapW := w - padding*2
	mapH := h - padding*2
	originX := x + padding
	originY := y + padding

	// Dessine les grids connus
	gridCount := len(world.GridOrder)
	if gridCount == 0 {
		return
	}

	// Simple layout en grille pour la minimap
	cols := int(math.Ceil(math.Sqrt(float64(gridCount))))
	rows := (gridCount + cols - 1) / cols
	cellW := mapW / float64(cols)
	cellH := mapH / float64(rows)
	cellSize := cellW
	if cellH < cellSize {
		cellSize = cellH
	}

	for i, gridID := range world.GridOrder {
		col := i % cols
		row := i / cols
		cx := originX + float64(col)*cellW + cellW/2
		cy := originY + float64(row)*cellH + cellH/2
		gx := cx - cellSize/2
		gy := cy - cellSize/2

		gridColor := color.RGBA{60, 60, 60, 255}
		if gridID == world.CurrentGridID {
			gridColor = color.RGBA{100, 180, 100, 255}
		}
		vector.DrawFilledRect(screen, float32(gx), float32(gy), float32(cellSize-2), float32(cellSize-2), gridColor, true)
		text.Draw(screen, gridID[:1], basicfont.Face7x13, int(gx+4), int(gy+12), color.White)
	}
}

func (is *IncursionScreen) biomeColor(world *domain.World) color.Color {
	grid, ok := world.GetCurrentGrid()
	if !ok {
		return color.RGBA{30, 30, 30, 255}
	}
	switch grid.Biome {
	case domain.BiomeForest:
		return color.RGBA{34, 60, 34, 255}
	case domain.BiomeCave:
		return color.RGBA{40, 35, 50, 255}
	case domain.BiomeDesert:
		return color.RGBA{60, 50, 30, 255}
	default:
		return color.RGBA{30, 30, 30, 255}
	}
}
