package hud

import (
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/meta"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
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
		visible: true,
		x:       300,
		y:       200,
		w:       400,
		h:       220,
		options: []meta.DifficultyLevel{meta.LevelEasy, meta.LevelNormal, meta.LevelHard},
	}
}

func (d *DifficultySelection) Render(screen *ebiten.Image) {
	if !d.visible {
		return
	}
	// Fond
	vector.DrawFilledRect(screen, d.x, d.y, d.w, d.h, color.RGBA{10, 10, 20, 220}, true)
	vector.StrokeRect(screen, d.x, d.y, d.w, d.h, 2, color.RGBA{150, 150, 200, 255}, true)
	text.Draw(screen, "SELECT DIFFICULTY", basicfont.Face7x13, int(d.x)+20, int(d.y)+30, color.White)

	// Options
	y := int(d.y) + 60
	for i, o := range d.options {
		text.Draw(screen, string(o), basicfont.Face7x13, int(d.x)+40, y+(i*30), color.RGBA{200, 200, 255, 255})
	}
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
	// Options y positions
	baseY := int(d.y) + 60
	for i, o := range d.options {
		oy := baseY + i*30
		if mx >= int(d.x)+40 && mx <= int(d.x)+200 && my >= oy-12 && my <= oy+6 {
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