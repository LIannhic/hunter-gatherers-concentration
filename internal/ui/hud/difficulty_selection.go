package hud

import (
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui/textutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DifficultySelection est une petite modal pour choisir la difficulté.
type DifficultySelection struct {
	visible    bool
	x, y, w, h float32
	options    []meta.DifficultyLevel
	OnSelected func(level meta.DifficultyLevel)
}

// NewDifficultySelection crée la modal par défaut.
func NewDifficultySelection() *DifficultySelection {
	return &DifficultySelection{
		visible: false,
		x:       440,
		y:       200,
		w:       400,
		h:       250,
		options: []meta.DifficultyLevel{meta.LevelEasy, meta.LevelNormal, meta.LevelHard, meta.LevelInsane},
	}
}

func (d *DifficultySelection) Render(screen *ebiten.Image) {
	if !d.visible {
		return
	}
	// Fond semi-transparent couvrant tout l'écran pour l'aspect modal
	overlay := ebiten.NewImage(1280, 720)
	overlay.Fill(color.RGBA{0, 0, 0, 180})
	screen.DrawImage(overlay, nil)

	// Fenêtre
	vector.DrawFilledRect(screen, d.x, d.y, d.w, d.h, color.RGBA{20, 20, 35, 255}, true)
	vector.StrokeRect(screen, d.x, d.y, d.w, d.h, 2, color.RGBA{100, 100, 200, 255}, true)

	title := "CHOISISSEZ VOTRE DESTIN"
	textutil.Draw(screen, title, int(d.x)+100, int(d.y)+35, color.RGBA{255, 220, 100, 255})

	mx, my := ebiten.CursorPosition()

	// Options
	baseY := int(d.y) + 70
	for i, o := range d.options {
		oy := baseY + i*40
		rect := Rect{X: int(d.x) + 50, Y: oy, W: 300, H: 30}

		isHovered := mx >= rect.X && mx <= rect.X+rect.W && my >= rect.Y && my <= rect.Y+rect.H

		bgColor := color.RGBA{40, 40, 60, 255}
		textColor := color.RGBA{200, 200, 255, 255}
		if isHovered {
			bgColor = color.RGBA{80, 80, 120, 255}
			textColor = color.RGBA{255, 255, 255, 255}
		}

		vector.DrawFilledRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), bgColor, true)
		if isHovered {
			vector.StrokeRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), 1, color.White, true)
		}

		textutil.Draw(screen, string(o), rect.X+110, rect.Y+20, textColor)
	}
}

// Rect est une structure utilitaire pour les zones cliquables
type Rect struct {
	X, Y, W, H int
}

// HandleClick retourne (level, ok) si une option est cliquée.
func (d *DifficultySelection) HandleClick(mx, my int) (meta.DifficultyLevel, bool) {
	if !d.visible {
		return "", false
	}
	fx, fy := float32(mx), float32(my)
	if fx < d.x || fx > d.x+d.w || fy < d.y || fy > d.y+d.h {
		return "", false
	}

	// Options positions
	baseY := int(d.y) + 70
	for i, o := range d.options {
		oy := baseY + i*40
		rect := Rect{X: int(d.x) + 50, Y: oy, W: 300, H: 30}

		if mx >= rect.X && mx <= rect.X+rect.W && my >= rect.Y && my <= rect.Y+rect.H {
			if d.OnSelected != nil {
				d.OnSelected(o)
			}
			return o, true
		}
	}
	return "", false
}

func (d *DifficultySelection) SetVisible(v bool) { d.visible = v }

func (d *DifficultySelection) IsVisible() bool { return d.visible }
