package renderer

import (
	"fmt"
	"image/color"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/persistence"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// SaveMenuAction représente l'action choisie dans le menu
type SaveMenuAction struct {
	Type SaveMenuActionType
	Slot int
}

type SaveMenuActionType int

const (
	ActionNone SaveMenuActionType = iota
	ActionLoad
	ActionNew
	ActionDelete
	ActionBack
	ActionChangeDifficulty
)

// SaveMenu gère l'affichage de la sélection des sauvegardes
type SaveMenu struct {
	width, height int
	visible       bool
	metas         []persistence.Metadata

	// État interne pour la confirmation
	confirmDeleteSlot int

	// Zones cliquables
	slotRects   []Rect
	deleteRects []Rect
	diffRects   []Rect
	backRect    Rect
}

func NewSaveMenu() *SaveMenu {
	sm := &SaveMenu{
		width:  1280,
		height: 720,
		slotRects: []Rect{
			{X: 365, Y: 130, W: 500, H: 160},
			{X: 365, Y: 300, W: 500, H: 160},
			{X: 365, Y: 470, W: 500, H: 160},
		},
		deleteRects: []Rect{
			{X: 875, Y: 130, W: 40, H: 160},
			{X: 875, Y: 300, W: 40, H: 160},
			{X: 875, Y: 470, W: 40, H: 160},
		},
		diffRects: []Rect{
			{X: 745, Y: 140, W: 110, H: 30},
			{X: 745, Y: 310, W: 110, H: 30},
			{X: 745, Y: 480, W: 110, H: 30},
		},
		backRect: Rect{X: 235, Y: 60, W: 140, H: 60},
	}
	return sm
}

func (m *SaveMenu) SetVisible(v bool) {
	m.visible = v
	if !v {
		m.confirmDeleteSlot = 0
	}
}

func (m *SaveMenu) IsVisible() bool { return m.visible }

func (m *SaveMenu) UpdateMetas(metas []persistence.Metadata) {
	m.metas = metas
}

func (m *SaveMenu) Render(screen *ebiten.Image) {
	if !m.visible {
		return
	}

	// Fond semi-transparent
	overlay := ebiten.NewImage(m.width, m.height)
	overlay.Fill(color.RGBA{0, 0, 0, 220})
	screen.DrawImage(overlay, nil)

	// Titre
	text.Draw(screen, "GESTION DES PROFILS", basicfont.Face7x13, 580, 125, color.RGBA{200, 180, 100, 255})

	// Bouton Retour
	m.drawBackButton(screen)

	// Slots
	for i := 1; i <= 3; i++ {
		m.drawSlot(screen, i)
	}

	// Message de confirmation
	if m.confirmDeleteSlot > 0 {
		m.drawConfirmation(screen)
	}
}

func (m *SaveMenu) drawBackButton(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, float32(m.backRect.X), float32(m.backRect.Y), float32(m.backRect.W), float32(m.backRect.H), color.RGBA{40, 40, 60, 255}, true)
	vector.StrokeRect(screen, float32(m.backRect.X), float32(m.backRect.Y), float32(m.backRect.W), float32(m.backRect.H), 2, color.White, true)
	text.Draw(screen, "< RETOUR", basicfont.Face7x13, m.backRect.X+15, m.backRect.Y+35, color.White)
}

func (m *SaveMenu) drawSlot(screen *ebiten.Image, id int) {
	r := m.slotRects[id-1]
	var slotMeta *persistence.Metadata
	for _, meta := range m.metas {
		if meta.SlotID == id {
			slotMeta = &meta
			break
		}
	}

	// Fond du slot
	bgColor := color.RGBA{30, 30, 35, 255}
	borderColor := color.RGBA{80, 80, 90, 255}
	if slotMeta != nil {
		borderColor = color.RGBA{100, 150, 255, 255}
	}

	vector.DrawFilledRect(screen, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), bgColor, true)
	vector.StrokeRect(screen, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), 2, borderColor, true)

	if slotMeta == nil {
		text.Draw(screen, fmt.Sprintf("SLOT %d : VIDE", id), basicfont.Face7x13, r.X+20, r.Y+85, color.Gray{150})
		text.Draw(screen, "[ CLIQUEZ POUR NOUVEAU PROFIL ]", basicfont.Face7x13, r.X+200, r.Y+85, color.White)
	} else {
		// Infos slot
		titleColor := color.RGBA{100, 150, 255, 255}
		text.Draw(screen, fmt.Sprintf("SLOT %d - %s", id, slotMeta.Difficulty), basicfont.Face7x13, r.X+20, r.Y+30, titleColor)

		createdStr := slotMeta.CreatedAt.Format("02/01/2006")
		text.Draw(screen, "Cree le : "+createdStr, basicfont.Face7x13, r.X+20, r.Y+55, color.Gray{150})

		dateStr := slotMeta.UpdatedAt.Format("02/01/2006 15:04")
		text.Draw(screen, "Dernier jeu : "+dateStr, basicfont.Face7x13, r.X+20, r.Y+80, color.White)

		playtimeStr := formatDuration(slotMeta.TotalPlaytime)
		text.Draw(screen, "Temps de jeu : "+playtimeStr, basicfont.Face7x13, r.X+20, r.Y+105, color.RGBA{200, 200, 100, 255})

		stats := fmt.Sprintf("Expeditions : %d | Morts : %d", slotMeta.SessionCount, slotMeta.DeathCount)
		text.Draw(screen, stats, basicfont.Face7x13, r.X+20, r.Y+130, color.Gray{180})

		scoreStr := fmt.Sprintf("Score Max : %d | Dernier : %d", slotMeta.MaxScore, slotMeta.LastScore)
		text.Draw(screen, scoreStr, basicfont.Face7x13, r.X+250, r.Y+130, color.RGBA{100, 200, 100, 255})

		// Bouton Difficulté
		dr := m.diffRects[id-1]
		vector.DrawFilledRect(screen, float32(dr.X), float32(dr.Y), float32(dr.W), float32(dr.H), color.RGBA{60, 60, 100, 255}, true)
		vector.StrokeRect(screen, float32(dr.X), float32(dr.Y), float32(dr.W), float32(dr.H), 1, color.RGBA{200, 200, 255, 255}, true)
		text.Draw(screen, "DIFFICULTE", basicfont.Face7x13, dr.X+15, dr.Y+20, color.White)

		// Bouton Supprimer (X)
		dlr := m.deleteRects[id-1]
		vector.DrawFilledRect(screen, float32(dlr.X), float32(dlr.Y), float32(dlr.W), float32(dlr.H), color.RGBA{80, 30, 30, 255}, true)
		text.Draw(screen, "X", basicfont.Face7x13, dlr.X+15, dlr.Y+85, color.White)
	}
}

func formatDuration(seconds float64) string {
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	return fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
}

func (m *SaveMenu) drawConfirmation(screen *ebiten.Image) {
	// Simple overlay box
	cw, ch := 400, 150
	cx, cy := (m.width-cw)/2, (m.height-ch)/2
	vector.DrawFilledRect(screen, float32(cx), float32(cy), float32(cw), float32(ch), color.RGBA{40, 20, 20, 255}, true)
	vector.StrokeRect(screen, float32(cx), float32(cy), float32(cw), float32(ch), 2, color.RGBA{255, 0, 0, 255}, true)

	msg := fmt.Sprintf("Supprimer le profil %d ?", m.confirmDeleteSlot)
	text.Draw(screen, msg, basicfont.Face7x13, cx+100, cy+50, color.White)
	text.Draw(screen, "Cliquez a nouveau sur X pour confirmer", basicfont.Face7x13, cx+60, cy+80, color.Gray{150})
	text.Draw(screen, "Ou n'importe ou ailleurs pour annuler", basicfont.Face7x13, cx+60, cy+100, color.Gray{150})
}

// HandleClick traite le clic et retourne l'action
func (m *SaveMenu) HandleClick(x, y int) SaveMenuAction {
	if !m.visible {
		return SaveMenuAction{Type: ActionNone}
	}

	// Si confirmation en cours
	if m.confirmDeleteSlot > 0 {
		dr := m.deleteRects[m.confirmDeleteSlot-1]
		if x >= dr.X && x <= dr.X+dr.W && y >= dr.Y && y <= dr.Y+dr.H {
			slot := m.confirmDeleteSlot
			m.confirmDeleteSlot = 0
			return SaveMenuAction{Type: ActionDelete, Slot: slot}
		}
		m.confirmDeleteSlot = 0
		return SaveMenuAction{Type: ActionNone}
	}

	// Bouton Retour
	if m.backRect.Contains(x, y) {
		return SaveMenuAction{Type: ActionBack}
	}

	// Clic sur Slots
	for i := 1; i <= 3; i++ {
		// Supprimer ?
		dlr := m.deleteRects[i-1]
		if dlr.Contains(x, y) {
			// Vérifier si le slot existe
			for _, meta := range m.metas {
				if meta.SlotID == i {
					m.confirmDeleteSlot = i
					return SaveMenuAction{Type: ActionNone}
				}
			}
		}

		// Difficulté ?
		diffr := m.diffRects[i-1]
		if diffr.Contains(x, y) {
			// Vérifier si le slot existe
			for _, meta := range m.metas {
				if meta.SlotID == i {
					return SaveMenuAction{Type: ActionChangeDifficulty, Slot: i}
				}
			}
		}

		// Charger / Nouveau ?
		sr := m.slotRects[i-1]
		if sr.Contains(x, y) {
			exists := false
			for _, meta := range m.metas {
				if meta.SlotID == i {
					exists = true
					break
				}
			}
			if exists {
				return SaveMenuAction{Type: ActionLoad, Slot: i}
			}
			return SaveMenuAction{Type: ActionNew, Slot: i}
		}
	}

	return SaveMenuAction{Type: ActionNone}
}
