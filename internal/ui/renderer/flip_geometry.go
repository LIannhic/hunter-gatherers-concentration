package renderer

import (
	"image/color"
	"math"
	"strings"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/player"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

// Constantes pour l'animation de flip
const (
	flipDuration = 0.75
)

type thickGeometry struct {
	V []ebiten.Vertex
	I []uint16
}

type axisType int

const (
	axisHorizontal axisType = iota
	axisVertical
	axisDiagonal
)

// ApplyBoardRotation applique la rotation globale du plateau ("Tourner") à un set de sommets.
// Elle pivote les points autour du centre de la tuile (cx, cy).
func (r *BoardRenderer) ApplyBoardRotation(v []ebiten.Vertex, cx, cy float32) {
	if r.boardRotation == 0 {
		return
	}
	angle := r.boardRotation * math.Pi / 180
	cosA, sinA := float32(math.Cos(angle)), float32(math.Sin(angle))

	for i := range v {
		relX := v[i].DstX - cx
		relY := v[i].DstY - cy
		v[i].DstX = cx + relX*cosA - relY*sinA
		v[i].DstY = cy + relX*sinA + relY*cosA
	}
}

// GetTransformationGeometry retourne les coordonnées des 4 coins d'une tuile (0,0 à 1,1)
// pour une transformation D4 donnée. Ordre: TL, TR, BR, BL.
func GetTransformationGeometry(t entity.Transformation) [4][2]float32 {
	switch t {
	case entity.TransIdentity:
		return [4][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	case entity.TransRot90:
		// Correction : rotation 90° horaire (v, 1-u)
		return [4][2]float32{{0, 1}, {0, 0}, {1, 0}, {1, 1}}
	case entity.TransRot180:
		return [4][2]float32{{1, 1}, {0, 1}, {0, 0}, {1, 0}}
	case entity.TransRot270:
		// Correction : rotation 270° horaire (1-v, u)
		return [4][2]float32{{1, 0}, {1, 1}, {0, 1}, {0, 0}}
	case entity.TransMirrorH:
		return [4][2]float32{{1, 0}, {0, 0}, {0, 1}, {1, 1}}
	case entity.TransMirrorD1:
		return [4][2]float32{{0, 0}, {0, 1}, {1, 1}, {1, 0}}
	case entity.TransMirrorV:
		return [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
	case entity.TransMirrorD2:
		return [4][2]float32{{1, 1}, {1, 0}, {0, 0}, {0, 1}}
	}
	return [4][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
}

func (r *BoardRenderer) renderFlippingTile(screen *ebiten.Image, x, y float64, anim *FlipAnimation, ent entity.Entity, themeName string, thicknessColor color.Color) {
	margin := (r.tileSize - ui.FaceSize) / 2
	tx, ty := float32(x+margin), float32(y+margin)
	cx, cy := float32(x+r.tileSize/2), float32(y+r.tileSize/2)

	hiddenImg := r.assets.GetTileImage("hidden", themeName)
	revealedImg := r.getEntityRevealedImage(ent, themeName)

	if ent == nil && strings.HasPrefix(anim.EntityID, "exit_") {
		revealedImg = r.assets.GetTileImage("exit", themeName)
	}

	tp := float32(smoothProgress(anim.Progress))
	isHiding := anim.TileState&entity.Hidden != 0

	g := r.createGeometry()
	r.initVerts(g.V)

	// Elevation 3D (effet de zoom)
	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.40

	// Facteur d'écrasement de la tuile (va de 1.0 à -1.0)
	scaleAnim := 1.0 - 2.0*tp

	for i := 0; i < 4; i++ {
		var vx, vy float32
		switch i {
		case 0:
			vx, vy = tx, ty
		case 1:
			vx, vy = tx+ui.FaceSize, ty
		case 2:
			vx, vy = tx+ui.FaceSize, ty+ui.FaceSize
		case 3:
			vx, vy = tx, ty+ui.FaceSize
		}

		relX := vx - cx
		relY := vy - cy

		switch anim.FlipDirection {
		case entity.FlipLeft, entity.FlipRight:
			vx = cx + relX*scaleAnim
		case entity.FlipTop, entity.FlipBottom:
			vy = cy + relY*scaleAnim
		case entity.FlipTopRight, entity.FlipBottomLeft:
			u := (relX + relY) * 0.5
			v := (relX - relY) * 0.5
			v *= scaleAnim
			vx = cx + u + v
			vy = cy + u - v
		case entity.FlipTopLeft, entity.FlipBottomRight:
			u := (relX - relY) * 0.5
			v := (relX + relY) * 0.5
			v *= scaleAnim
			vx = cx + u + v
			vy = cy - u + v
		}

		g.V[i].DstX = cx + (vx-cx)*elevation
		g.V[i].DstY = cy + (vy-cy)*elevation
	}

	// Une seule orientation fixe tout le long du flip
	var currentFace, currentBack *ebiten.Image

	// On se base uniquement sur l'état de DÉPART pour verrouiller la texture initiale
	if isHiding {
		currentFace = revealedImg
		currentBack = hiddenImg
	} else {
		currentFace = hiddenImg
		currentBack = revealedImg
	}

	// On utilise STRICTEMENT StartTransform pour projeter la texture de départ
	trans := anim.StartTransform

	uvCoords := GetTransformationGeometry(trans)
	for i := 0; i < 4; i++ {
		g.V[i].SrcX = uvCoords[i][0] * ui.FaceSize
		g.V[i].SrcY = uvCoords[i][1] * ui.FaceSize
	}

	// Épaisseur et rotation globale du plateau
	r.extrude(g.V, anim.FlipDirection, false, 0, tp, thicknessColor)
	r.ApplyBoardRotation(g.V, cx, cy)

	// Déterminer quelle face est vers l'avant selon la rotation (scaleAnim)
	frontFacing := scaleAnim > 0

	if frontFacing {
		// On dessine le DOS d'abord, puis la FACE
		r.drawGeometryPart(screen, g.V, g.I[6:12], currentBack)
		r.drawGeometryPart(screen, g.V, g.I[:6], currentFace)
	} else {
		// On dessine la FACE d'abord, puis le DOS (qui est maintenant devant)
		r.drawGeometryPart(screen, g.V, g.I[:6], currentFace)
		r.drawGeometryPart(screen, g.V, g.I[6:12], currentBack)
	}

	r.drawSlices(screen, g, anim.FlipDirection, r.assets.GetImage("white"))

	shouldShowIcon := (!isHiding && anim.Progress >= 0.5) || (isHiding && anim.Progress < 0.5)

	if shouldShowIcon && ent != nil {
		r.renderFlippingEntityTriangles(screen, g.V[:4], ent, trans)
	}
}

func (r *BoardRenderer) renderFlippingEntityTriangles(screen *ebiten.Image, vFace []ebiten.Vertex, e entity.Entity, trans entity.Transformation) {
	// 1. Calcul du centre de la face (sommets 0-3 fournis)
	var cx, cy float32
	for i := 0; i < 4; i++ {
		cx += vFace[i].DstX
		cy += vFace[i].DstY
	}
	cx /= 4
	cy /= 4

	// 2. Interpolation des coins pour une échelle de 75%
	const scale = 0.75
	vIcon := make([]ebiten.Vertex, 4)

	for i := 0; i < 4; i++ {
		vIcon[i] = vFace[i]
		vIcon[i].DstX = cx + (vFace[i].DstX-cx)*scale
		vIcon[i].DstY = cy + (vFace[i].DstY-cy)*scale
	}

	var icon *ebiten.Image
	var label string
	var behaviorState string

	switch ent := e.(type) {
	case *domain.Creature:
		icon = r.assets.GetCreatureIcon(ent.Species)
		behaviorState = ent.Behavior.State
	case *domain.Resource:
		stageName := ent.Lifecycle.GetCurrentStageName()
		icon = r.assets.GetResourceIcon(ent.ResourceType, stageName)
		if len(stageName) > 0 {
			label = string(stageName[0])
		}
	case *player.LootItem:
		// On tente de récupérer l'icône appropriée selon le type d'origine ou le SourceID
		if ent.OriginalType == entity.TypeCreature {
			icon = r.assets.GetCreatureIcon(ent.SourceID)
		} else if ent.OriginalType == entity.TypeResource {
			icon = r.assets.GetResourceIcon(ent.SourceID, "")
		} else {
			// Fallback sur le SourceID pour les artefacts etc.
			icon = r.assets.GetImage("resource_" + ent.SourceID)
		}
		if icon == nil {
			// Dernier recours : première lettre du nom
			if len(ent.Name) > 0 {
				label = string(ent.Name[0])
			}
		}
	}

	if icon == nil {
		return
	}

	// 3. Réglage des UV pour l'icône avec prise en compte de la transformation diédrique fournie
	iw, ih := icon.Size()
	fw, fh := float32(iw), float32(ih)

	// On utilise la géométrie de transformation pour mapper les UV
	iconUvCoords := GetTransformationGeometry(trans)
	for i := 0; i < 4; i++ {
		vIcon[i].SrcX = iconUvCoords[i][0] * fw
		vIcon[i].SrcY = iconUvCoords[i][1] * fh
	}

	// 4. Rendu de l'icône avec les triangles
	indices := []uint16{0, 1, 2, 0, 2, 3}
	r.drawGeometryPart(screen, vIcon, indices, icon)

	// 5. Indicateurs additionnels
	// Calcul de l'échelle relative actuelle pour les feedbacks 2D
	dist01 := math.Sqrt(math.Pow(float64(vFace[1].DstX-vFace[0].DstX), 2) + math.Pow(float64(vFace[1].DstY-vFace[0].DstY), 2))
	currentScale := float32(dist01) / ui.FaceSize

	if behaviorState != "" {
		behaviorColor := color.RGBA{200, 200, 200, 255}
		switch behaviorState {
		case "hunting":
			behaviorColor = color.RGBA{255, 100, 100, 255}
		case "fleeing":
			behaviorColor = color.RGBA{100, 100, 255, 255}
		case "pollinating":
			behaviorColor = color.RGBA{100, 255, 100, 255}
		}

		tx := (vFace[0].DstX + vFace[1].DstX) / 2
		ty := (vFace[0].DstY + vFace[1].DstY) / 2
		bx := tx + (cx-tx)*0.25
		by := ty + (cy-ty)*0.25
		vector.DrawFilledCircle(screen, bx, by, 4*currentScale, behaviorColor, true)

		// --- NOUVEAU: BARRE D'AGRESSIVITÉ (Feedback Visuel) ---
		if cre, ok := e.(*domain.Creature); ok && cre.Behavior.Aggression > 0 {
			agg := cre.Behavior.Aggression
			// Barre en bas de la tuile
			barW := 40.0 * currentScale
			barH := 4.0 * currentScale
			barX := cx - barW/2
			barY := vFace[2].DstY - 12*currentScale

			// Couleur dégradée : Orange (Peu énervé) -> Rouge (Très énervé)
			rVal := uint8(200 + (agg * 55 / 100))
			gVal := uint8(150 - (agg * 150 / 100))
			barColor := color.RGBA{rVal, gVal, 0, 255}

			// Fond de la barre
			vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{20, 20, 20, 180}, true)
			// Remplissage
			fillW := barW * float32(agg) / 100.0
			vector.DrawFilledRect(screen, barX, barY, fillW, barH, barColor, true)
		}
	}

	if label != "" {
		lx := vFace[2].DstX - 12*currentScale
		ly := vFace[2].DstY - 5*currentScale
		text.Draw(screen, label, basicfont.Face7x13, int(lx), int(ly), color.White)
	}
}

func (r *BoardRenderer) extrude(v []ebiten.Vertex, dir entity.FlipDirection, isIdle bool, hover float32, flipTime float32, clr color.Color) {
	baseThick := float32(ui.FlipThickness)
	var dx, dy float32
	diagThick := baseThick * 0.707
	switch dir {
	case entity.FlipLeft:
		dx = -baseThick
	case entity.FlipRight:
		dx = baseThick
	case entity.FlipTop:
		dy = -baseThick
	case entity.FlipBottom:
		dy = baseThick
	case entity.FlipTopLeft:
		dx, dy = -diagThick, -diagThick
	case entity.FlipTopRight:
		dx, dy = diagThick, -diagThick
	case entity.FlipBottomLeft:
		dx, dy = -diagThick, diagThick
	case entity.FlipBottomRight:
		dx, dy = diagThick, diagThick
	}

	cr, cg, cb, ca := clr.RGBA()
	fR, fG, fB, fA := float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff

	var near, far, adj1, adj2 int
	switch dir {
	case entity.FlipTopLeft:
		near, far, adj1, adj2 = 0, 2, 1, 3
	case entity.FlipTopRight:
		near, far, adj1, adj2 = 1, 3, 0, 2
	case entity.FlipBottomRight:
		near, far, adj1, adj2 = 2, 0, 1, 3
	case entity.FlipBottomLeft:
		near, far, adj1, adj2 = 3, 1, 0, 2
	}

	for i := 4; i < 8; i++ {
		vIdx := i - 4
		localDx, localDy := dx, dy

		if isIdle {
			var factor float32 = 1.0
			// Utilise la direction de survol pour réduire l'épaisseur sur le côté incliné (perspective)
			switch dir {
			case entity.FlipLeft:
				if vIdx == 0 || vIdx == 3 {
					factor = 1.0 - (hover * 0.6)
				}
			case entity.FlipRight:
				if vIdx == 1 || vIdx == 2 {
					factor = 1.0 - (hover * 0.6)
				}
			case entity.FlipTop:
				if vIdx == 0 || vIdx == 1 {
					factor = 1.0 - (hover * 0.6)
				}
			case entity.FlipBottom:
				if vIdx == 2 || vIdx == 3 {
					factor = 1.0 - (hover * 0.6)
				}
			default: // Diagonales
				switch vIdx {
				case near:
					factor = (1.0 - (hover * 0.85)) + 0.125
				case adj1, adj2:
					factor = 1.0 - (hover * 0.3)
				case far:
					factor = 1.0
				}
			}
			localDx *= factor
			localDy *= factor
		} else {
			// Élévation dynamique de la tranche pendant le flip
			thicknessAnim := 1.0 + float32(math.Sin(float64(flipTime)*math.Pi))*0.6
			localDx *= thicknessAnim
			localDy *= thicknessAnim
		}

		v[i].DstX = v[vIdx].DstX + localDx
		v[i].DstY = v[vIdx].DstY + localDy
		v[i].SrcX, v[i].SrcY = v[vIdx].SrcX, v[vIdx].SrcY
		v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = fR, fG, fB, fA
	}
}

func (r *BoardRenderer) generateIdleGeometry(tx, ty float32, id string, thicknessColor color.Color) thickGeometry {
	g := r.createGeometry()
	r.initVerts(g.V)
	r.setFaceUV(g.V, ui.FaceSize)

	// Centre de la tuile pour les calculs de pivot/impact
	cx, cy := tx+ui.FaceSize/2, ty+ui.FaceSize/2

	// 1. Récupération des états d'animation
	hover, hasHover := r.hoverStates[id]
	bounce, hasBounce := r.bounceStates[id]

	// 2. Application du HOVER (inclinaison levier)
	if hasHover && hover.Progress > 0 {
		h := hover.Progress * 8.0 // Amplitude de l'inclinaison
		switch hover.Dir {
		case entity.FlipLeft:
			g.V[0].DstX, g.V[0].DstY = tx, ty+h*0.25
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize-h*0.25, ty-h*0.75
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize-h*0.25, ty+ui.FaceSize+h*0.75
			g.V[3].DstX, g.V[3].DstY = tx, ty+ui.FaceSize-h*0.25
		case entity.FlipRight:
			g.V[0].DstX, g.V[0].DstY = tx+h*0.25, ty-h*0.75
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize, ty+h*0.25
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize, ty+ui.FaceSize-h*0.25
			g.V[3].DstX, g.V[3].DstY = tx+h*0.25, ty+ui.FaceSize+h*0.75
		case entity.FlipTop:
			g.V[0].DstX, g.V[0].DstY = tx+h*0.25, ty
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize-h*0.25, ty
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize+h*0.75, ty+ui.FaceSize-h*0.25
			g.V[3].DstX, g.V[3].DstY = tx-h*0.75, ty+ui.FaceSize-h*0.25
		case entity.FlipBottom:
			g.V[0].DstX, g.V[0].DstY = tx-h*0.75, ty+h*0.25
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize+h*0.75, ty+h*0.25
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize, ty+ui.FaceSize
			g.V[3].DstX, g.V[3].DstY = tx, ty+ui.FaceSize
		case entity.FlipTopLeft:
			g.V[0].DstX, g.V[0].DstY = tx+h*0.25, ty+h*0.25
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize-h*0.5, ty-h*0.75
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize-h*0.75, ty+ui.FaceSize-h*0.75
			g.V[3].DstX, g.V[3].DstY = tx-h*0.75, ty+ui.FaceSize-h*0.5
		case entity.FlipTopRight:
			g.V[0].DstX, g.V[0].DstY = tx+h*0.5, ty-h*0.75
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize-h*0.25, ty-h*0.25
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize+h*0.75, ty+ui.FaceSize-h*0.5
			g.V[3].DstX, g.V[3].DstY = tx+h*0.75, ty+ui.FaceSize-h*0.75
		case entity.FlipBottomRight:
			g.V[0].DstX, g.V[0].DstY = tx+h*0.75, ty+h*0.75
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize+h*0.75, ty-h*0.5
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize-h*0.25, ty+ui.FaceSize-h*0.25
			g.V[3].DstX, g.V[3].DstY = tx+h*0.5, ty+ui.FaceSize+h*0.75
		case entity.FlipBottomLeft:
			g.V[0].DstX, g.V[0].DstY = tx-h*0.75, ty-h*0.5
			g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize-h*0.75, ty+h*0.75
			g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize-h*0.5, ty+ui.FaceSize+h*0.75
			g.V[3].DstX, g.V[3].DstY = tx+h*0.25, ty+ui.FaceSize-h*0.25
		}
	} else {
		// Position standard au repos
		g.V[0].DstX, g.V[0].DstY = tx, ty
		g.V[1].DstX, g.V[1].DstY = tx+ui.FaceSize, ty
		g.V[2].DstX, g.V[2].DstY = tx+ui.FaceSize, ty+ui.FaceSize
		g.V[3].DstX, g.V[3].DstY = tx, ty+ui.FaceSize
	}

	// 3. Application du BOUNCE (impact élastique)
	if hasBounce {
		var bAmp float32
		p := bounce.ImpactT
		switch {
		case p < 0.25:
			bAmp = float32(math.Sin((float64(p)/0.25)*math.Pi/2)) * 6.0
		case p < 0.5:
			bAmp = (1.0 - float32((p-0.25)/0.25)) * 6.0
		case p < 0.75:
			bAmp = -float32(math.Sin(((float64(p)-0.5)/0.25)*math.Pi/2)) * 2.5
		}

		if bAmp != 0 {
			for i := 0; i < 4; i++ {
				dx := g.V[i].DstX - cx
				dy := g.V[i].DstY - cy
				dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
				if dist > 0.0001 {
					g.V[i].DstX += (dx / dist) * bAmp
					g.V[i].DstY += (dy / dist) * bAmp
				}
			}
		}
	}

	// 4. Extrude avec prise en compte du survol pour la perspective
	hProgress := float32(0)
	hDir := entity.FlipTop
	if hasHover {
		hProgress = hover.Progress
		hDir = hover.Dir
	} else if hasBounce {
		hDir = bounce.Dir
	}
	r.extrude(g.V, hDir, true, hProgress, 0, thicknessColor)

	return g
}

func (r *BoardRenderer) createGeometry() thickGeometry {
	return thickGeometry{
		V: make([]ebiten.Vertex, 8),
		I: []uint16{
			0, 1, 2, 0, 2, 3, // FRONT
			4, 5, 6, 4, 6, 7, // BACK
			4, 5, 1, 4, 1, 0, // TOP
			5, 6, 2, 5, 2, 1, // RIGHT
			6, 7, 3, 6, 3, 2, // BOTTOM
			7, 4, 0, 7, 0, 3, // LEFT
		},
	}
}

func (r *BoardRenderer) initVerts(v []ebiten.Vertex) {
	for i := range v {
		v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = 1, 1, 1, 1
	}
}

func (r *BoardRenderer) setFaceUV(v []ebiten.Vertex, size float32) {
	v[0].SrcX, v[0].SrcY = 0, 0
	v[1].SrcX, v[1].SrcY = size, 0
	v[2].SrcX, v[2].SrcY = size, size
	v[3].SrcX, v[3].SrcY = 0, size
}

func smoothProgress(t float64) float64 {
	t -= 0.5
	return 4*t*t*t + 0.5
}

func (r *BoardRenderer) drawGeometryPart(screen *ebiten.Image, v []ebiten.Vertex, indices []uint16, img *ebiten.Image) {
	screen.DrawTriangles(v, indices, img, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}

func (r *BoardRenderer) drawSlices(screen *ebiten.Image, geo thickGeometry, dir entity.FlipDirection, backImg *ebiten.Image) {
	showRight, showLeft, showTop, showBottom := false, false, false, false
	switch dir {
	case entity.FlipLeft:
		showLeft = true
	case entity.FlipRight:
		showRight = true
	case entity.FlipTop:
		showTop = true
	case entity.FlipBottom:
		showBottom = true
	case entity.FlipTopLeft:
		showTop, showLeft = true, true
	case entity.FlipTopRight:
		showTop, showRight = true, true
	case entity.FlipBottomLeft:
		showBottom, showLeft = true, true
	case entity.FlipBottomRight:
		showBottom, showRight = true, true
	}

	// FIX UV BLEEDING: Set all vertices used by slices to UV (0,0) for the white pixel image
	for i := range geo.V {
		geo.V[i].SrcX = 0
		geo.V[i].SrcY = 0
	}

	// On utilise l'image blanche pour avoir la couleur pure définie dans les sommets
	sliceImg := r.assets.GetImage("white")

	if showTop {
		r.drawGeometryPart(screen, geo.V, geo.I[12:18], sliceImg)
	}
	if showRight {
		r.drawGeometryPart(screen, geo.V, geo.I[18:24], sliceImg)
	}
	if showBottom {
		r.drawGeometryPart(screen, geo.V, geo.I[24:30], sliceImg)
	}
	if showLeft {
		r.drawGeometryPart(screen, geo.V, geo.I[30:36], sliceImg)
	}
}

func (r *BoardRenderer) renderEarthquakeTile360(screen *ebiten.Image, x, y float64, progress float64, ent entity.Entity, themeName string, thicknessColor color.Color, flipDir entity.FlipDirection) {
	margin := (r.tileSize - ui.FaceSize) / 2
	tx, ty := float32(x+margin), float32(y+margin)
	cx, cy := float32(x+r.tileSize/2), float32(y+r.tileSize/2)

	hiddenImg := r.assets.GetTileImage("hidden", themeName)
	revealedImg := r.getEntityRevealedImage(ent, themeName)
	visualState := ent.GetState()
	frontImg := revealedImg
	backImg := hiddenImg

	// Lissage du progrès pour la fluidité 3D
	tp := float32(smoothProgress(progress))

	g := r.createGeometry()
	r.initVerts(g.V)

	// 1. Élévation 3D : la tuile s'élève et redescend en parabole parfaite
	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.50

	// 2. Rotation 360° avec sens constant
	angle := float32(tp * 2 * math.Pi)
	cosAngle := math.Cos(float64(angle))
	scale := float32(math.Abs(cosAngle))

	for i := 0; i < 4; i++ {
		var vx, vy float32
		switch i {
		case 0:
			vx, vy = tx, ty
		case 1:
			vx, vy = tx+ui.FaceSize, ty
		case 2:
			vx, vy = tx+ui.FaceSize, ty+ui.FaceSize
		case 3:
			vx, vy = tx, ty+ui.FaceSize
		}

		relX := vx - cx
		relY := vy - cy

		switch flipDir {
		case entity.FlipLeft, entity.FlipRight:
			vx = cx + relX*scale
		case entity.FlipTop, entity.FlipBottom:
			vy = cy + relY*scale
		case entity.FlipTopRight, entity.FlipBottomLeft:
			u := (relX + relY) * 0.5
			v := (relX - relY) * 0.5
			v *= scale
			vx = cx + u + v
			vy = cy + u - v
		case entity.FlipTopLeft, entity.FlipBottomRight:
			u := (relX - relY) * 0.5
			v := (relX + relY) * 0.5
			v *= scale
			vx = cx + u + v
			vy = cy - u + v
		default:
			vx = cx + relX*scale
		}

		g.V[i].DstX = cx + (vx-cx)*elevation
		g.V[i].DstY = cy + (vy-cy)*elevation
	}

	// 3. Mapping des UV : On part de la transformation par défaut (TransIdentity)
	uvCoords := GetTransformationGeometry(entity.TransIdentity)
	for i := 0; i < 4; i++ {
		g.V[i].SrcX = uvCoords[i][0] * ui.FaceSize
		g.V[i].SrcY = uvCoords[i][1] * ui.FaceSize
	}

	// 4. Tranche et rotation globale du plateau
	r.extrude(g.V, flipDir, false, 0, tp, thicknessColor)
	r.ApplyBoardRotation(g.V, cx, cy)

	// 5. Logique des Faces (360°)
	// Un cosinus positif signifie que la position géométrique correspond à la face avant.
	// Si l'entité est masquée, on inverse la visibilité pour démarrer par le dos.
	isFrontVisible := cosAngle > 0
	if visualState&entity.Revealed == 0 && visualState&entity.Matched == 0 {
		isFrontVisible = !isFrontVisible
	}

	if isFrontVisible {
		// On dessine d'abord l'arrière, puis la face avant.
		r.drawGeometryPart(screen, g.V, g.I[6:12], backImg)
		r.drawGeometryPart(screen, g.V, g.I[:6], frontImg)
	} else {
		// On dessine d'abord la face avant, puis l'arrière.
		r.drawGeometryPart(screen, g.V, g.I[:6], frontImg)
		r.drawGeometryPart(screen, g.V, g.I[6:12], backImg)
	}

	// Dessin des tranches
	r.drawSlices(screen, g, flipDir, r.assets.GetImage("white"))

	// 6. Affichage de l'icône de l'entité uniquement au milieu du vol (quand la face est visible)
	showIcon := isFrontVisible && ent != nil && ent.GetType() != entity.TypeTrap
	if showIcon {
		r.renderFlippingEntityTriangles(screen, g.V[:4], ent, entity.TransIdentity)
	}
}

// drawElasticFilament dessine le rectangle blanc uni qui s'amincit sur le bon axe (X ou Y) selon la direction
func (r *BoardRenderer) drawElasticFilament(screen *ebiten.Image, px, py, cx, cy float64, progress float64, themeName string) {
	if progress >= 0.85 {
		return
	}

	dx := cx - px
	dy := cy - py
	isVertical := math.Abs(dy) > math.Abs(dx)

	whiteImg := r.assets.GetImage("white")
	if whiteImg == nil {
		return
	}

	alpha := float32(1.0 - (progress / 0.85))
	var p1X, p1Y, p2X, p2Y, c1X, c1Y, c2X, c2Y float64

	if isVertical {
		// Mouvement vertical : le filament est une bande horizontale qui s'amincit en HAUTEUR (Y)
		// L'amincissement se produit surtout en phase 2 (quand la traîne remonte)
		thickness := r.tileSize * 0.4
		if progress > 0.5 {
			thickness *= (1.0 - progress) * 2.0
		}

		// On trace le rectangle entre la frontière des deux cases
		midY := (py + cy) / 2
		p1X, p1Y = px-r.tileSize*0.4, midY-thickness*0.5
		p2X, p2Y = px+r.tileSize*0.4, midY-thickness*0.5
		c1X, c1Y = px+r.tileSize*0.4, midY+thickness*0.5
		c2X, c2Y = px-r.tileSize*0.4, midY+thickness*0.5
	} else {
		// Mouvement horizontal : le filament est une bande verticale qui s'amincit en LARGEUR (X)
		thickness := r.tileSize * 0.4
		if progress > 0.5 {
			thickness *= (1.0 - progress) * 2.0
		}

		midX := (px + cx) / 2
		p1X, p1Y = midX-thickness*0.5, py-r.tileSize*0.4
		p2X, p2Y = midX+thickness*0.5, py-r.tileSize*0.4
		c1X, c1Y = midX+thickness*0.5, py+r.tileSize*0.4
		c2X, c2Y = midX-thickness*0.5, py+r.tileSize*0.4
	}

	vs := []ebiten.Vertex{
		{DstX: float32(p1X), DstY: float32(p1Y), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
		{DstX: float32(p2X), DstY: float32(p2Y), SrcX: 1, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
		{DstX: float32(c1X), DstY: float32(c1Y), SrcX: 1, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
		{DstX: float32(c2X), DstY: float32(c2Y), SrcX: 0, SrcY: 1, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha},
	}
	indices := []uint16{0, 1, 2, 0, 2, 3}
	screen.DrawTriangles(vs, indices, whiteImg, nil)
}

// renderPropagatingChildTile anime les 4 angles de la tuile de manière asynchrone (Tête puis Traîne)
func (r *BoardRenderer) renderPropagatingChildTile(screen *ebiten.Image, x, y, px, py float64, progress float64, world *domain.World, entityID string, themeName string) {
	theme := r.assets.GetTheme(themeName)

	// On génère la géométrie de départ sur la case PARENTE (px, py)
	margin := (r.tileSize - ui.FaceSize) / 2
	parentTileX := px - r.tileSize/2
	parentTileY := py - r.tileSize/2
	geo := r.generateIdleGeometry(float32(parentTileX+margin), float32(parentTileY+margin), entityID, theme.HiddenBorder)

	// Calcul du vecteur total de déplacement de la tuile
	totalDx := float32(x - parentTileX)
	totalDy := float32(y - parentTileY)

	isVertical := math.Abs(float64(totalDy)) > math.Abs(float64(totalDx))

	// Définition des taux de progression pour les deux vagues (Tête et Traîne)
	var tHead, tTail float32
	if progress < 0.5 {
		tHead = float32(progress * 2.0) // La tête fait 100% de son trajet entre progress 0 et 0.5
		tTail = 0.0                     // La traîne ne bouge pas encore
	} else {
		tHead = 1.0
		tTail = float32((progress - 0.5) * 2.0) // La traîne fait son trajet entre progress 0.5 et 1.0
	}

	// Indices standards des sommets générés par generateIdleGeometry :
	// geo.V[0] = Haut-Gauche (HG), geo.V[1] = Haut-Droit (HD)
	// geo.V[2] = Bas-Droit (BD),   geo.V[3] = Bas-Gauche (BG)

	if isVertical {
		if totalDy < 0 { // Déplacement vers le HAUT (Nord)
			// Tête = HG (0) et HD (1)
			geo.V[0].DstX += totalDx * tHead
			geo.V[0].DstY += totalDy * tHead
			geo.V[1].DstX += totalDx * tHead
			geo.V[1].DstY += totalDy * tHead
			// Traîne = BD (2) et BG (3)
			geo.V[2].DstX += totalDx * tTail
			geo.V[2].DstY += totalDy * tTail
			geo.V[3].DstX += totalDx * tTail
			geo.V[3].DstY += totalDy * tTail
		} else { // Déplacement vers le BAS (Sud)
			// Tête = BD (2) et BG (3)
			geo.V[2].DstX += totalDx * tHead
			geo.V[2].DstY += totalDy * tHead
			geo.V[3].DstX += totalDx * tHead
			geo.V[3].DstY += totalDy * tHead
			// Traîne = HG (0) et HD (1)
			geo.V[0].DstX += totalDx * tTail
			geo.V[0].DstY += totalDy * tTail
			geo.V[1].DstX += totalDx * tTail
			geo.V[1].DstY += totalDy * tTail
		}
	} else {
		if totalDx > 0 { // Déplacement vers la DROITE (Est)
			// Tête = HD (1) et BD (2)
			geo.V[1].DstX += totalDx * tHead
			geo.V[1].DstY += totalDy * tHead
			geo.V[2].DstX += totalDx * tHead
			geo.V[2].DstY += totalDy * tHead
			// Traîne = HG (0) et BG (3)
			geo.V[0].DstX += totalDx * tTail
			geo.V[0].DstY += totalDy * tTail
			geo.V[3].DstX += totalDx * tTail
			geo.V[3].DstY += totalDy * tTail
		} else { // Déplacement vers la GAUCHE (Ouest)
			// Tête = HG (0) et BG (3)
			geo.V[0].DstX += totalDx * tHead
			geo.V[0].DstY += totalDy * tHead
			geo.V[3].DstX += totalDx * tHead
			geo.V[3].DstY += totalDy * tHead
			// Traîne = HD (1) et BD (2)
			geo.V[1].DstX += totalDx * tTail
			geo.V[1].DstY += totalDy * tTail
			geo.V[2].DstX += totalDx * tTail
			geo.V[2].DstY += totalDy * tTail
		}
	}

	// Application de la rotation globale autour du centre actuel de la forme
	cx := float32(parentTileX + r.tileSize/2 + float64(totalDx)*float64(progress))
	cy := float32(parentTileY + r.tileSize/2 + float64(totalDy)*float64(progress))
	r.ApplyBoardRotation(geo.V, cx, cy)

	// Rendu en mode DOS (Caché)
	backImg := r.assets.GetTileImage("hidden", themeName)
	if backImg == nil {
		backImg = r.assets.GetImage("tile_hidden")
	}

	r.drawGeometryPart(screen, geo.V, geo.I[6:12], backImg)
	r.drawGeometryPart(screen, geo.V, geo.I[:6], backImg)
}
