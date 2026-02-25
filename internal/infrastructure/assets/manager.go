// Package assets gère le chargement et la mise en cache des ressources graphiques
package assets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Manager gère les assets du jeu
type Manager struct {
	// Cache d'images générées
	images map[string]*ebiten.Image
	colors map[string]color.Color
}

// NewManager crée un nouveau gestionnaire d'assets
func NewManager() *Manager {
	m := &Manager{
		images: make(map[string]*ebiten.Image),
		colors: map[string]color.Color{
			"hidden_bg":      color.RGBA{40, 40, 60, 255},
			"hidden_border":  color.RGBA{80, 80, 100, 255},
			"revealed_bg":    color.RGBA{60, 60, 80, 255},
			"matched_bg":     color.RGBA{40, 100, 40, 255},
			"grid_lines":     color.RGBA{100, 100, 120, 255},
			"text_default":   color.RGBA{220, 220, 220, 255},
			"highlight":      color.RGBA{255, 255, 100, 255},
			"creature":       color.RGBA{255, 100, 100, 255},
			"resource_plant": color.RGBA{100, 255, 100, 255},
			"resource_mineral": color.RGBA{150, 150, 200, 255},
		},
	}
	
	m.generatePlaceholders()
	return m
}

// GetImage retourne une image du cache (ou une image par défaut)
func (m *Manager) GetImage(name string) *ebiten.Image {
	if img, ok := m.images[name]; ok {
		return img
	}
	// Retourne une image par défaut
	return m.images["default"]
}

// GetColor retourne une couleur par nom
func (m *Manager) GetColor(name string) color.Color {
	if c, ok := m.colors[name]; ok {
		return c
	}
	return color.White
}

// generatePlaceholders crée des images placeholder pour le développement
func (m *Manager) generatePlaceholders() {
	size := 64
	
	// Image par défaut
	defaultImg := ebiten.NewImage(size, size)
	defaultImg.Fill(color.RGBA{100, 100, 100, 255})
	m.images["default"] = defaultImg
	
	// Tuile cachée
	hiddenImg := ebiten.NewImage(size, size)
	hiddenImg.Fill(m.colors["hidden_bg"])
	// Bordure
	vector.StrokeLine(hiddenImg, 0, 0, float32(size), 0, 2, m.colors["hidden_border"], true)
	vector.StrokeLine(hiddenImg, 0, 0, 0, float32(size), 2, m.colors["hidden_border"], true)
	vector.StrokeLine(hiddenImg, float32(size-1), 0, float32(size-1), float32(size), 2, m.colors["hidden_border"], true)
	vector.StrokeLine(hiddenImg, 0, float32(size-1), float32(size), float32(size-1), 2, m.colors["hidden_border"], true)
	m.images["tile_hidden"] = hiddenImg
	
	// Tuile révélée
	revealedImg := ebiten.NewImage(size, size)
	revealedImg.Fill(m.colors["revealed_bg"])
	m.images["tile_revealed"] = revealedImg
	
	// Tuile appairée
	matchedImg := ebiten.NewImage(size, size)
	matchedImg.Fill(m.colors["matched_bg"])
	m.images["tile_matched"] = matchedImg
	
	// Icônes de ressources
	plantImg := ebiten.NewImage(size, size)
	plantImg.Fill(m.colors["revealed_bg"])
	// Dessine une forme simple représentant une plante
	vector.DrawFilledCircle(plantImg, float32(size/2), float32(size/3), 15, m.colors["resource_plant"], true)
	vector.DrawFilledRect(plantImg, float32(size/2-3), float32(size/3+10), 6, 20, m.colors["resource_plant"], true)
	m.images["resource_dreamberry"] = plantImg
	
	mineralImg := ebiten.NewImage(size, size)
	mineralImg.Fill(m.colors["revealed_bg"])
	// Dessine une forme simple représentant un minerai
	vector.DrawFilledCircle(mineralImg, float32(size/2), float32(size/2), 20, m.colors["resource_mineral"], true)
	m.images["resource_moonstone"] = mineralImg
	
	// Icône de créature
	creatureImg := ebiten.NewImage(size, size)
	creatureImg.Fill(m.colors["revealed_bg"])
	// Dessine une forme simple représentant une créature
	vector.DrawFilledCircle(creatureImg, float32(size/2), float32(size/2), 18, m.colors["creature"], true)
	m.images["creature_lumifly"] = creatureImg
	
	// Autres créatures
	stalkerImg := ebiten.NewImage(size, size)
	stalkerImg.Fill(m.colors["revealed_bg"])
	vector.DrawFilledRect(stalkerImg, float32(size/2-15), float32(size/2-15), 30, 30, color.RGBA{150, 50, 50, 255}, true)
	m.images["creature_shadowstalker"] = stalkerImg
}

// GetResourceIcon retourne l'icône pour une ressource
func (m *Manager) GetResourceIcon(resourceType string) *ebiten.Image {
	key := "resource_" + resourceType
	if img, ok := m.images[key]; ok {
		return img
	}
	return m.images["default"]
}

// GetCreatureIcon retourne l'icône pour une créature
func (m *Manager) GetCreatureIcon(species string) *ebiten.Image {
	key := "creature_" + species
	if img, ok := m.images[key]; ok {
		return img
	}
	return m.images["default"]
}
