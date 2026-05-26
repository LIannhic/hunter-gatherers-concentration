package renderer

import (
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
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
func (tr *TrackRenderer) RenderUnder(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	tracks := world.Entities.GetByType(entity.TypeTrack)
	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok || t.GetGridID() != world.CurrentGridID {
			continue
		}
		// mud et broken_grass sont sur la strate Under
		if t.Kind == "mud" || t.Kind == "broken_grass" {
			tr.Draw(screen, t, getCenter)
		} else if t.FromPos == t.ToPos {
			// Autres traces statiques par défaut
			tr.Draw(screen, t, getCenter)
		}
	}
}

// RenderNormal affiche les traces situées dans les interstices (les translations)
func (tr *TrackRenderer) RenderNormal(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	tracks := world.Entities.GetByType(entity.TypeTrack)
	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok || t.GetGridID() != world.CurrentGridID {
			continue
		}
		// On dessine ici les traces de mouvement génériques (pas mud qui est Under)
		if t.Kind != "mud" && t.Kind != "claws" && t.Kind != "broken_grass" && t.FromPos != t.ToPos {
			tr.Draw(screen, t, getCenter)
		}
	}
}

// RenderEffectsNormal dessine les lignes de visée et les lasers d'intention d'attaque
func (tr *TrackRenderer) RenderEffectsNormal(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	// À implémenter : effets d'attaque si nécessaire
}

// RenderOver affiche les indices ou effets aériens (ex: marques de griffes volantes, nuages)
func (tr *TrackRenderer) RenderOver(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	tracks := world.Entities.GetByType(entity.TypeTrack)
	for _, e := range tracks {
		t, ok := e.(*entity.Track)
		if !ok || t.GetGridID() != world.CurrentGridID {
			continue
		}
		// claws est sur la strate Over
		if t.Kind == "claws" {
			tr.Draw(screen, t, getCenter)
		}
	}
}

// =========================================================================
// MOTEUR DE RENDU TECHNIQUE (GÉOMÉTRIE ET MATRICES)
// =========================================================================

// Draw effectue le rendu d'une trace en calculant sa position (Sur, Sous, ou Entre)
func (tr *TrackRenderer) Draw(screen *ebiten.Image, t *entity.Track, getCenter func(board.Position) (float64, float64)) {
	sprite := tr.getOrCreateSprite(t.Kind)
	if sprite == nil {
		return // Sécurité si le sprite ne peut pas être généré
	}

	op := &ebiten.DrawImageOptions{}
	w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()

	// 1. Coordonnées en pixels des centres des cases de départ et d'arrivée via le callback dynamique
	startX, startY := getCenter(t.FromPos)
	endX, endY := getCenter(t.ToPos)

	var drawX, drawY float64
	var angle float64
	hasRotation := false

	// 2. Calcul du point d'ancrage selon la nature géométrique de l'indice
	if t.Kind == "broken_grass" {
		// Herbes brisées: à l'origine
		drawX = startX
		drawY = startY
	} else if t.Kind == "claws" {
		// Griffes: à la destination
		drawX = endX
		drawY = endY
	} else if t.Kind == "mud" {
		// Boue: entre les cases
		drawX = startX + (endX-startX)*0.5
		drawY = startY + (endY-startY)*0.5
		angle = math.Atan2(endY-startY, endX-startX)
		hasRotation = true
	} else if t.FromPos == t.ToPos {
		// --- INDICE STATIQUE ---
		drawX = startX
		drawY = startY
	} else {
		// --- INDICE ENTRE LES CASES (Interstice générique) ---
		drawX = startX + (endX-startX)*0.5
		drawY = startY + (endY-startY)*0.5
		angle = math.Atan2(endY-startY, endX-startX)
		hasRotation = true
	}

	// 3. Centrage, Rotation (si besoin) et translation finale unifiés
	// IMPORTANT : Toujours centrer avant de pivoter
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	if hasRotation {
		op.GeoM.Rotate(angle)
	}
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
