package renderer

import (
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/board"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/component"
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
		// On dessine ici les traces de mouvement génériques (pas mud qui est Under, ni claws/intent qui sont Over)
		if t.Kind != "mud" && t.Kind != "claws" && t.Kind != "intent_beam" && t.Kind != "broken_grass" && t.FromPos != t.ToPos {
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
		// claws et intent_beam sont sur la strate Over
		if t.Kind == "claws" || t.Kind == "intent_beam" {
			tr.Draw(screen, t, getCenter)
		}
	}
}

// RenderAttackThreats dessine des marqueurs blancs sur les cases menacées pendant une attaque
func (tr *TrackRenderer) RenderAttackThreats(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	attackingIDs := world.Components.QueryByComponent("attacking_animation")
	if len(attackingIDs) == 0 {
		return
	}

	whiteSprite := tr.getOrCreateSprite("intent_beam_white")
	redSprite := tr.getOrCreateSprite("intent_beam")

	for _, id := range attackingIDs {
		ent, ok := world.Entities.Get(entity.ID(id))
		if !ok || ent.GetType() != entity.TypeCreature {
			continue
		}

		comp, _ := world.Components.Get(id, "attacking_animation")
		aa := comp.(*component.AttackingAnimation)

		creature := ent.(*domain.Creature)
		threats := creature.GetActiveThreatDirections()
		startPos := creature.GetPosition()
		startX, startY := getCenter(startPos)

		for _, dir := range threats {
			targetPos := startPos.Add(dir.ToVector())

			// On dessine l'indicateur même si la case est hors de la grille (sur le tapis)
			endX, endY := getCenter(targetPos)

			// Calcul de la position "entre les cases" (milieu)
			midX := startX + (endX-startX)*0.5
			midY := startY + (endY-startY)*0.5
			angle := math.Atan2(endY-startY, endX-startX)

			op := &ebiten.DrawImageOptions{}

			// Sélection du sprite : rouge si c'est la cible touchée, blanc sinon
			sprite := whiteSprite
			if aa.HitTarget != nil && *aa.HitTarget == targetPos {
				sprite = redSprite
			}

			if sprite == nil {
				continue
			}

			w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()

			// Centrage et Rotation
			op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			op.GeoM.Rotate(angle)
			op.GeoM.Translate(midX, midY)

			// Gestion de l'alpha (fondu)
			t := float64(aa.CurrentTick) / float64(aa.DurationTicks)
			alpha := 1.0
			if t > 0.5 {
				alpha = (1.0 - t) * 2.0
			}
			op.ColorScale.ScaleAlpha(float32(alpha))

			screen.DrawImage(sprite, op)
		}
	}
}

// RenderPotentialThreats dessine des marqueurs blancs sur les cases menacées par toutes les créatures
func (tr *TrackRenderer) RenderPotentialThreats(screen *ebiten.Image, world *domain.World, getCenter func(board.Position) (float64, float64)) {
	creatures := world.Entities.GetByType(entity.TypeCreature)
	whiteSprite := tr.getOrCreateSprite("intent_beam_white")
	if whiteSprite == nil {
		return
	}

	for _, ent := range creatures {
		c, ok := ent.(*domain.Creature)
		if !ok || c.GetGridID() != world.CurrentGridID {
			continue
		}

		// On n'affiche pas si la créature est déjà en train d'animer une attaque (géré par RenderAttackThreats)
		if _, exists := world.Components.Get(string(c.GetID()), "attacking_animation"); exists {
			continue
		}

		threats := c.GetActiveThreatDirections()
		if len(threats) == 0 {
			continue
		}

		startPos := c.GetPosition()
		startX, startY := getCenter(startPos)

		for _, dir := range threats {
			targetPos := startPos.Add(dir.ToVector())
			endX, endY := getCenter(targetPos)

			midX := startX + (endX-startX)*0.5
			midY := startY + (endY-startY)*0.5
			angle := math.Atan2(endY-startY, endX-startX)

			op := &ebiten.DrawImageOptions{}
			w, h := whiteSprite.Bounds().Dx(), whiteSprite.Bounds().Dy()

			op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			op.GeoM.Rotate(angle)
			op.GeoM.Translate(midX, midY)

			// Légère transparence pour ne pas trop surcharger l'écran
			op.ColorScale.ScaleAlpha(0.6)

			screen.DrawImage(whiteSprite, op)
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

	// 2. Calcul du point milieu (pour comportement cohérent avec les intentions de menace)
	midX := startX + (intent.TargetX-startX)*0.5
	midY := startY + (intent.TargetY-startY)*0.5
	angle := math.Atan2(intent.TargetY-startY, intent.TargetX-startX)

	op := &ebiten.DrawImageOptions{}
	w, h := sprite.Bounds().Dx(), sprite.Bounds().Dy()

	// 3. Application de la matrice géométrique
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(midX, midY)

	screen.DrawImage(sprite, op)
}
