package renderer

import (
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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
	sprites map[string]*ebiten.Image
}

func NewTrackRenderer(sprites map[string]*ebiten.Image) *TrackRenderer {
	return &TrackRenderer{
		sprites: sprites,
	}
}

// =========================================================================
// INTERFACES DU PIPELINE DE RENDU (STRATES)
// =========================================================================

// RenderUnder gère les indices enfouis ou sous les cases (ex: boue profonde, galeries)
func (tr *TrackRenderer) RenderUnder(screen *ebiten.Image, world *domain.World) {
	// Exemple : si tu as des traces immobiles de type "mud", elles restent au sol
	// Tu peux itérer sur tes traces ici si ton world ou tes grids les exposent.
}

// RenderBetween affiche les traces situées dans les interstices (les translations)
func (tr *TrackRenderer) RenderBetween(screen *ebiten.Image, world *domain.World) {
	// C'est ici que tu appelles Draw pour les traces qui vont d'une case A à une case B
}

// RenderEffectsBetween dessine les lignes de visée et les lasers d'intention d'attaque
func (tr *TrackRenderer) RenderEffectsBetween(screen *ebiten.Image, world *domain.World) {
	// On raccorde l'ancienne méthode DrawAttackIntent ici si un effet est actif
	// Exemple hypothétique en attendant ton implémentation finale :
	// if world.CurrentAttackIntent != nil {
	//     tr.DrawAttackIntent(screen, world.CurrentAttackIntent, 64, 0)
	// }
}

// RenderOver affiche les indices ou effets aériens (ex: marques de griffes volantes, nuages)
func (tr *TrackRenderer) RenderOver(screen *ebiten.Image, world *domain.World) {
	// Évolutions futures pour les traces au-dessus des tuiles physiques
}

// =========================================================================
// MOTEUR DE RENDU TECHNIQUE (GÉOMÉTRIE ET MATRICES)
// =========================================================================

// Draw effectue le rendu d'une trace en calculant sa position (Sur, Sous, ou Entre)
func (tr *TrackRenderer) Draw(screen *ebiten.Image, t *entity.Trace, tileSize, spacing float64) {
	sprite, exists := tr.sprites[t.Kind]
	if !exists || sprite == nil {
		return // Évite un crash si le sprite de l'indice n'est pas chargé
	}

	op := &ebiten.DrawImageOptions{}
	w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()

	// 1. Coordonnées en pixels des centres des cases de départ et d'arrivée
	startX := float64(t.FromPos.X)*(tileSize+spacing) + (tileSize / 2)
	startY := float64(t.FromPos.Y)*(tileSize+spacing) + (tileSize / 2)

	endX := float64(t.ToPos.X)*(tileSize+spacing) + (tileSize / 2)
	endY := float64(t.ToPos.Y)*(tileSize+spacing) + (tileSize / 2)

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

	sprite := tr.sprites["intent_beam"]
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