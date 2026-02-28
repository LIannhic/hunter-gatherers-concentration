// Package assets gère le chargement et la mise en cache des ressources graphiques
//
// ⚠️  ASSETS TEMPORAIRES / PLACEHOLDERS
// Ce package génère tous les assets procéduralement. Ce sont des placeholders
// temporaires pour faciliter le développement sans dépendances externes.
//
// Les assets seront remplacés par :
// - Sprites/Pixel art pour les tuiles
// - Illustrations pour les ressources et créatures
// - Animations pour les effets de flip
//
// TODO: Intégrer le chargement d'assets externes avant release
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
	themes map[string]TileTheme
}

// NewManager crée un nouveau gestionnaire d'assets
func NewManager() *Manager {
	m := &Manager{
		images: make(map[string]*ebiten.Image),
		colors: map[string]color.Color{
			"hidden_bg":        color.RGBA{40, 40, 60, 255},
			"hidden_border":    color.RGBA{80, 80, 100, 255},
			"revealed_bg":      color.RGBA{60, 60, 80, 255},
			"matched_bg":       color.RGBA{40, 100, 40, 255},
			"grid_lines":       color.RGBA{100, 100, 120, 255},
			"text_default":     color.RGBA{220, 220, 220, 255},
			"highlight":        color.RGBA{255, 255, 100, 255},
			"creature":         color.RGBA{255, 100, 100, 255},
			"resource_plant":   color.RGBA{100, 255, 100, 255},
			"resource_mineral": color.RGBA{150, 150, 200, 255},
		},
		themes: map[string]TileTheme{
			"default": ThemeDefault,
			"forest":  ThemeForest,
			"cave":    ThemeCave,
			"swamp":   ThemeSwamp,
		},
	}

	m.generateAllAssets()
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

// GetTheme retourne un thème par nom
func (m *Manager) GetTheme(name string) TileTheme {
	if theme, ok := m.themes[name]; ok {
		return theme
	}
	return ThemeDefault
}

// generateAllAssets génère tous les assets du jeu
func (m *Manager) generateAllAssets() {
	size := 64

	// Image par défaut
	defaultImg := ebiten.NewImage(size, size)
	defaultImg.Fill(color.RGBA{100, 100, 100, 255})
	m.images["default"] = defaultImg

	// === TUILES AVEC THEMES ===
	for themeName, theme := range m.themes {
		// Tuile cachée avec motif décoratif
		hiddenImg := generateTileHidden(size, theme)
		m.images["tile_hidden_"+themeName] = hiddenImg

		// Tuile révélée
		revealedImg := generateTileRevealed(size, theme)
		m.images["tile_revealed_"+themeName] = revealedImg

		// Tuile appairée
		matchedImg := generateTileMatched(size, theme)
		m.images["tile_matched_"+themeName] = matchedImg
	}

	// Tuiles par défaut (sans suffixe de thème)
	m.images["tile_hidden"] = m.images["tile_hidden_default"]
	m.images["tile_revealed"] = m.images["tile_revealed_default"]
	m.images["tile_matched"] = m.images["tile_matched_default"]

	// === RESSOURCES ===
	// Dreamberry
	m.images["resource_dreamberry"] = generateDreamberry(size, DreamberryPalette)
	// Moonstone
	m.images["resource_moonstone"] = generateMoonstone(size, MoonstonePalette)
	// Whispering Herb
	m.images["resource_whispering_herb"] = generateWhisperingHerb(size, WhisperingHerbPalette)
	// Shadow Essence
	m.images["resource_shadow_essence"] = generateShadowEssence(size, ShadowEssencePalette)
	// Crystal Shard
	m.images["resource_crystal_shard"] = generateCrystalShard(size, CrystalShardPalette)

	// === CRÉATURES ===
	// Lumifly
	m.images["creature_lumifly"] = generateLumifly(size, LumiflyPalette)
	// Shadowstalker
	m.images["creature_shadowstalker"] = generateShadowstalker(size, ShadowstalkerPalette)
	// Burrower
	m.images["creature_burrower"] = generateBurrower(size, BurrowerPalette)
	// Flutterwing
	m.images["creature_flutterwing"] = generateFlutterwing(size, FlutterwingPalette)

	// === EFFETS DE FLIP ===
	m.images["flip_overlay_top"] = generateFlipEffectOverlay(size, "top")
	m.images["flip_overlay_bottom"] = generateFlipEffectOverlay(size, "bottom")
	m.images["flip_overlay_left"] = generateFlipEffectOverlay(size, "left")
	m.images["flip_overlay_right"] = generateFlipEffectOverlay(size, "right")
	m.images["flip_overlay_center"] = generateFlipEffectOverlay(size, "center")

	// === INDICATEURS DE DIRECTION ===
	m.images["direction_top"] = generateDirectionIndicator(size, "top")
	m.images["direction_bottom"] = generateDirectionIndicator(size, "bottom")
	m.images["direction_left"] = generateDirectionIndicator(size, "left")
	m.images["direction_right"] = generateDirectionIndicator(size, "right")

	// === ICÔNES DE COMPORTEMENT ===
	m.generateBehaviorIcons(size)

	// === ICÔNES D'UI ===
	m.generateUIIcons(size)
}

// generateBehaviorIcons crée les icônes pour les états de comportement des créatures
func (m *Manager) generateBehaviorIcons(size int) {
	// Chasse (hunting) - Rouge
	huntingImg := ebiten.NewImage(size/2, size/2)
	vector.DrawFilledCircle(huntingImg, float32(size/4), float32(size/4), float32(size/8), color.RGBA{255, 80, 80, 255}, true)
	m.images["behavior_hunting"] = huntingImg

	// Fuite (fleeing) - Bleu
	fleeingImg := ebiten.NewImage(size/2, size/2)
	vector.DrawFilledCircle(fleeingImg, float32(size/4), float32(size/4), float32(size/8), color.RGBA{80, 100, 255, 255}, true)
	m.images["behavior_fleeing"] = fleeingImg

	// Pollinisation (pollinating) - Vert
	pollinatingImg := ebiten.NewImage(size/2, size/2)
	vector.DrawFilledCircle(pollinatingImg, float32(size/4), float32(size/4), float32(size/8), color.RGBA{100, 255, 100, 255}, true)
	m.images["behavior_pollinating"] = pollinatingImg

	// Inactif (idle) - Gris
	idleImg := ebiten.NewImage(size/2, size/2)
	vector.DrawFilledCircle(idleImg, float32(size/4), float32(size/4), float32(size/8), color.RGBA{180, 180, 180, 255}, true)
	m.images["behavior_idle"] = idleImg
}

// generateUIIcons crée les icônes pour l'interface utilisateur
func (m *Manager) generateUIIcons(size int) {
	// Icône de rotation
	rotationImg := ebiten.NewImage(size/2, size/2)
	center := float32(size / 4)
	// Cercle
	vector.StrokeCircle(rotationImg, center, center, float32(size/6), 3, color.RGBA{200, 200, 200, 255}, true)
	// Flèche de rotation
	vector.StrokeLine(rotationImg, center+5, center-8, center+12, center-12, 3, color.RGBA{200, 200, 200, 255}, true)
	vector.StrokeLine(rotationImg, center+12, center-12, center+8, center-5, 3, color.RGBA{200, 200, 200, 255}, true)
	m.images["ui_rotation"] = rotationImg

	// Icône de flip
	flipImg := ebiten.NewImage(size/2, size/2)
	// Rectangle
	vector.StrokeRect(flipImg, 8, 8, float32(size/2-16), float32(size/2-16), 2, color.RGBA{200, 200, 200, 255}, true)
	// Flèche de flip
	vector.StrokeLine(flipImg, float32(size/4), 4, float32(size/4), 12, 3, color.RGBA{255, 255, 100, 255}, true)
	m.images["ui_flip"] = flipImg
}

// GetResourceIcon retourne l'icône pour une ressource
func (m *Manager) GetResourceIcon(resourceType string) *ebiten.Image {
	key := "resource_" + resourceType
	if img, ok := m.images[key]; ok {
		return img
	}
	// Génère une icône générique si non trouvée
	return generateGenericResource(64, ResourcePalette{
		Primary:   color.RGBA{150, 150, 150, 255},
		Secondary: color.RGBA{100, 100, 100, 255},
		Accent:    color.RGBA{200, 200, 200, 255},
		Bg:        color.RGBA{60, 60, 60, 255},
	})
}

// GetCreatureIcon retourne l'icône pour une créature
func (m *Manager) GetCreatureIcon(species string) *ebiten.Image {
	key := "creature_" + species
	if img, ok := m.images[key]; ok {
		return img
	}
	// Génère une icône générique si non trouvée
	return generateGenericCreature(64, CreaturePalette{
		Body:      color.RGBA{150, 150, 150, 255},
		Highlight: color.RGBA{200, 200, 200, 255},
		Shadow:    color.RGBA{100, 100, 100, 255},
		Eye:       color.RGBA{50, 50, 50, 255},
		Bg:        color.RGBA{60, 60, 60, 255},
	})
}

// GetTileImage retourne l'image de tuile avec le thème spécifié
func (m *Manager) GetTileImage(tileType string, themeName string) *ebiten.Image {
	key := "tile_" + tileType + "_" + themeName
	if img, ok := m.images[key]; ok {
		return img
	}
	// Retourne l'image par défaut
	return m.images["tile_"+tileType]
}

// GetFlipOverlay retourne l'overlay d'effet de flip pour une direction
func (m *Manager) GetFlipOverlay(direction string) *ebiten.Image {
	key := "flip_overlay_" + direction
	if img, ok := m.images[key]; ok {
		return img
	}
	return m.images["flip_overlay_center"]
}

// GetDirectionIndicator retourne l'indicateur de direction
func (m *Manager) GetDirectionIndicator(direction string) *ebiten.Image {
	key := "direction_" + direction
	if img, ok := m.images[key]; ok {
		return img
	}
	return m.images["default"]
}
