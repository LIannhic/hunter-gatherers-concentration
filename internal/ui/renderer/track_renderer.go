package renderer

import (
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/infrastructure/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

// AttackIntent représente une intention d'attaque canalisée vers le curseur.
type AttackIntent struct {
	SourcePos entity.Position
	TargetX   float64
	TargetY   float64
	Intensity float64
	IsActive  bool
}

type TrackRenderer struct {
	spriteCache map[string]*ebiten.Image // Cache des assets générés
	tileSize    float64
}

func NewTrackRenderer(tileSize float64) *TrackRenderer {
	return &TrackRenderer{
		spriteCache: make(map[string]*ebiten.Image),
		tileSize:    tileSize,
	}
}

// getOrCreateSprite retourne le sprite d'une trace, en le générant si nécessaire
func (tr *TrackRenderer) getOrCreateSprite(kind string) *ebiten.Image {
	if sprite, exists := tr.spriteCache[kind]; exists {
		return sprite
	}

	// Génère l'asset et le met en cache
	sprite := assets.GenerateTrackAsset(kind, int(tr.tileSize))
	tr.spriteCache[kind] = sprite
	return sprite
}

// =========================================================================
// INTERFACES DU PIPELINE DE RENDU (STRATES)
// =========================================================================

// RenderUnder gère les indices enfouis ou sous les cases (ex: boue profonde, galeries)
func (tr *TrackRenderer) RenderUnder(screen *ebiten.Image, world *domain.World, offsetX, offsetY, tileSize, spacingX, spacingY float64) {
	tracks := world.Entities.GetByType(entity.TypeTrack)
	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok || t.GetGridID() != world.CurrentGridID {
			continue
		}
		// On ne dessine ici que les traces "ancrées" (FromPos == ToPos)
		if t.FromPos == t.ToPos {
			tr.Draw(screen, t, offsetX, offsetY, tileSize, spacingX, spacingY)
		}
	}
}

// RenderBetween affiche les traces situées dans les interstices (les translations)
func (tr *TrackRenderer) RenderBetween(screen *ebiten.Image, world *domain.World, offsetX, offsetY, tileSize, spacingX, spacingY float64) {
	tracks := world.Entities.GetByType(entity.TypeTrack)
	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok || t.GetGridID() != world.CurrentGridID {
			continue
		}
		// On dessine ici les traces de mouvement (interstices)
		if t.FromPos != t.ToPos {
			tr.Draw(screen, t, offsetX, offsetY, tileSize, spacingX, spacingY)
		}
	}
}

// RenderEffectsBetween dessine les lignes de visée et les lasers d'intention d'attaque
func (tr *TrackRenderer) RenderEffectsBetween(screen *ebiten.Image, world *domain.World, offsetX, offsetY, tileSize, spacingX, spacingY float64) {
	// À implémenter : effets d'attaque si nécessaire
	// if world.CurrentAttackIntent != nil {
	//     tr.DrawAttackIntent(screen, world.CurrentAttackIntent, tileSize, spacingX)
	// }
}

// RenderOver affiche les indices ou effets aériens (ex: marques de griffes volantes, nuages)
func (tr *TrackRenderer) RenderOver(screen *ebiten.Image, world *domain.World, offsetX, offsetY, tileSize, spacingX, spacingY float64) {
	// Évolutions futures pour les traces au-dessus des tuiles physiques
}

// =========================================================================
// MOTEUR DE RENDU TECHNIQUE (GÉOMÉTRIE ET MATRICES)
// =========================================================================

// Draw effectue le rendu d'une trace en calculant sa position (Sur, Sous, ou Entre)
func (tr *TrackRenderer) Draw(screen *ebiten.Image, t *entity.Track, offsetX, offsetY, tileSize, spacingX, spacingY float64) {
	sprite := tr.getOrCreateSprite(t.Kind)
	if sprite == nil {
		return // Sécurité si le sprite ne peut pas être généré
	}

	op := &ebiten.DrawImageOptions{}
	w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()

	// 1. Coordonnées en pixels des centres des cases de départ et d'arrivée
	startX := offsetX + float64(t.FromPos.X)*(tileSize+spacingX) + (tileSize / 2)
	startY := offsetY + float64(t.FromPos.Y)*(tileSize+spacingY) + (tileSize / 2)

	endX := offsetX + float64(t.ToPos.X)*(tileSize+spacingX) + (tileSize / 2)
	endY := offsetY + float64(t.ToPos.Y)*(tileSize+spacingY) + (tileSize / 2)

	var drawX, drawY float64

	// 2. Calcul du point d'ancrage selon la nature géométrique de l'indice
	if t.FromPos == t.ToPos {
		// --- INDICE SUR OU SOUS LA CASE ---
		drawX = startX
		drawY = startY
	} else {
		// --- INDICE ENTRE LES CASES (Interstice) ---
		drawX = startX + (endX-startX)*0.5
		drawY = startY + (endY-startY)*0.5

		// Orienter l'indice dans la direction de la translation
		angle := math.Atan2(endY-startY, endX-startX)
		op.GeoM.Rotate(angle)
	}

	// 3. Centrage et translation finale unifiés
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Translate(drawX, drawY)

	screen.DrawImage(sprite, op)
}

// DrawAttackIntent étire et pivote une lueur d'attaque vers le curseur cible à 360°
func (tr *TrackRenderer) DrawAttackIntent(screen *ebiten.Image, intent *AttackIntent, tileSize, spacing float64) {
	if !intent.IsActive {
		return
	}

	sprite := tr.getOrCreateSprite("intent_beam")
	if sprite == nil {
		return
	}

	// 1. Centre de la case de départ de la créature
	startX := float64(intent.SourcePos.X)*(tileSize+spacing) + (tileSize / 2)
	startY := float64(intent.SourcePos.Y)*(tileSize+spacing) + (tileSize / 2)

	// 2. Calcul des deltas, de l'angle et de la distance géométrique vers la souris
	dx := intent.TargetX - startX
	dy := intent.TargetY - startY
	angle := math.Atan2(dy, dx)
	distance := math.Sqrt(dx*dx + dy*dy)

	op := &ebiten.DrawImageOptions{}
	w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()

	// 3. Application STRICTEMENT ordonnée de la matrice géométrique d'Ebiten
	op.GeoM.Translate(0, -float64(h)/2)

	scaleX := distance / float64(w)
	op.GeoM.Scale(scaleX, 1.0)

	op.GeoM.Rotate(angle)
	op.GeoM.Translate(startX, startY)

	screen.DrawImage(sprite, op)
}
