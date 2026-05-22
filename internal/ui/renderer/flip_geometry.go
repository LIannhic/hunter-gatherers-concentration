package renderer

import (
	"image/color"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain"
	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
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

func (r *BoardRenderer) renderFlippingTile(screen *ebiten.Image, x, y float64, anim *FlipAnimation, ent entity.Entity, thicknessColor color.Color) {
	margin := (r.tileSize - ui.FaceSize) / 2
	tx, ty := float32(x+margin), float32(y+margin)
	cx, cy := float32(x+r.tileSize/2), float32(y+r.tileSize/2)

	frontImg := r.assets.GetImage("tile_hidden")
	var backImg *ebiten.Image
	if ent.GetType() == entity.TypeTrap {
		backImg = r.assets.GetImage("tile_trap")
	} else {
		backImg = r.assets.GetImage("tile_revealed")
	}

	tp := float32(smoothProgress(anim.Progress))
	showBack := anim.Progress > 0.5

	var geo thickGeometry
	dir := anim.FlipDirection
	switch {
	case dir == entity.FlipTop || dir == entity.FlipBottom:
		geo = r.generateFlipV(tx, ty, ui.FaceSize, ui.FlipThickness, dir, tp, thicknessColor)
	case dir == entity.FlipLeft || dir == entity.FlipRight || dir == entity.FlipCenter:
		geo = r.generateFlipH(tx, ty, ui.FaceSize, ui.FlipThickness, dir, tp, thicknessColor)
	default:
		geo = r.generateFlipDiag(tx, ty, ui.FaceSize, ui.FlipThickness, dir, tp, thicknessColor)
	}

	// Application de la rotation globale du plateau ("Tourner")
	r.ApplyBoardRotation(geo.V, cx, cy)

	faceImg := frontImg
	if showBack {
		faceImg = backImg
	}

	r.drawGeometryPart(screen, geo.V, geo.I[:6], faceImg)    // Face
	r.drawGeometryPart(screen, geo.V, geo.I[6:12], backImg)  // Dos
	r.drawSlices(screen, geo, dir, r.assets.GetImage("white")) // Tranches

	if showBack && ent.GetType() != entity.TypeTrap {
		// On dessine l'entité sur la face "Face" (sommets 0 à 3) qui a pivoté
		// Les sommets 0-3 conservent la couleur blanche (non teintée par l'épaisseur)
		r.renderFlippingEntityTriangles(screen, geo.V[:4], ent)
	}
}

func (r *BoardRenderer) renderFlippingEntityTriangles(screen *ebiten.Image, vFace []ebiten.Vertex, e entity.Entity) {
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
	}

	if icon == nil {
		return
	}

	// 3. Réglage des UV pour l'icône avec prise en compte de l'orientation
	w, h := icon.Size()
	fw, fh := float32(w), float32(h)

	// Orientation de l'entité
	orient := entity.DirNorth
	if e != nil {
		orient = e.GetOrientation()
	}

	// Mapping des UV selon l'orientation
	switch orient {
	case entity.DirNorth:
		vIcon[0].SrcX, vIcon[0].SrcY = 0, 0
		vIcon[1].SrcX, vIcon[1].SrcY = fw, 0
		vIcon[2].SrcX, vIcon[2].SrcY = fw, fh
		vIcon[3].SrcX, vIcon[3].SrcY = 0, fh
	case entity.DirEast:
		vIcon[0].SrcX, vIcon[0].SrcY = 0, fh
		vIcon[1].SrcX, vIcon[1].SrcY = 0, 0
		vIcon[2].SrcX, vIcon[2].SrcY = fw, 0
		vIcon[3].SrcX, vIcon[3].SrcY = fw, fh
	case entity.DirSouth:
		vIcon[0].SrcX, vIcon[0].SrcY = fw, fh
		vIcon[1].SrcX, vIcon[1].SrcY = 0, fh
		vIcon[2].SrcX, vIcon[2].SrcY = 0, 0
		vIcon[3].SrcX, vIcon[3].SrcY = fw, 0
	case entity.DirWest:
		vIcon[0].SrcX, vIcon[0].SrcY = fw, 0
		vIcon[1].SrcX, vIcon[1].SrcY = fw, fh
		vIcon[2].SrcX, vIcon[2].SrcY = 0, fh
		vIcon[3].SrcX, vIcon[3].SrcY = 0, 0
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
	}

	if label != "" {
		lx := vFace[2].DstX - 12*currentScale
		ly := vFace[2].DstY - 5*currentScale
		text.Draw(screen, label, basicfont.Face7x13, int(lx), int(ly), color.White)
	}
}

func (r *BoardRenderer) generateFlipH(tx, ty, faceSize, thickness float32, dir entity.FlipDirection, tp float32, thicknessColor color.Color) thickGeometry {
	g := r.createGeometry()
	r.initVerts(g.V)
	r.setFaceUV(g.V, faceSize)

	cx, cy := tx+faceSize/2, ty+faceSize/2
	s := float32(math.Cos(float64(tp) * math.Pi))
	if dir == entity.FlipLeft {
		s = -s
	}

	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.20
	half, height := (faceSize/2)*s*elevation, (faceSize/2)*elevation

	g.V[0].DstX, g.V[0].DstY = cx-half, cy-height
	g.V[1].DstX, g.V[1].DstY = cx+half, cy-height
	g.V[2].DstX, g.V[2].DstY = cx+half, cy+height
	g.V[3].DstX, g.V[3].DstY = cx-half, cy+height

	r.extrude(g.V, axisHorizontal, dir, false, 0, tp, thickness, thicknessColor)
	return g
}

func (r *BoardRenderer) generateFlipV(tx, ty, faceSize, thickness float32, dir entity.FlipDirection, tp float32, thicknessColor color.Color) thickGeometry {
	g := r.createGeometry()
	r.initVerts(g.V)
	r.setFaceUV(g.V, faceSize)

	cx, cy := tx+faceSize/2, ty+faceSize/2
	s := float32(math.Cos(float64(tp) * math.Pi))
	if dir == entity.FlipTop {
		s = -s
	}

	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.20
	half, width := (faceSize/2)*s*elevation, (faceSize/2)*elevation

	g.V[0].DstX, g.V[0].DstY = cx-width, cy-half
	g.V[1].DstX, g.V[1].DstY = cx+width, cy-half
	g.V[2].DstX, g.V[2].DstY = cx+width, cy+half
	g.V[3].DstX, g.V[3].DstY = cx-width, cy+half

	r.extrude(g.V, axisVertical, dir, false, 0, tp, thickness, thicknessColor)
	return g
}

func (r *BoardRenderer) generateFlipDiag(tx, ty, faceSize, thickness float32, dir entity.FlipDirection, tp float32, thicknessColor color.Color) thickGeometry {
	g := r.createGeometry()
	r.initVerts(g.V)
	r.setFaceUV(g.V, faceSize)

	cx, cy := tx+faceSize/2, ty+faceSize/2
	tp64 := float64(tp)
	angle := tp64 * math.Pi
	halfFace := faceSize / 2

	type vec2 struct {
		X, Y float32
	}
	base := [4]vec2{{-halfFace, -halfFace}, {halfFace, -halfFace}, {halfFace, halfFace}, {-halfFace, halfFace}}

	var a float64
	if dir == entity.FlipTopRight || dir == entity.FlipBottomLeft {
		a = math.Pi / 4
	} else {
		a = -math.Pi / 4
	}
	cosA, sinA := math.Cos(a), math.Sin(a)

	farIdx := map[entity.FlipDirection]int{entity.FlipTopLeft: 2, entity.FlipTopRight: 3, entity.FlipBottomRight: 0, entity.FlipBottomLeft: 1}
	nearIdx := map[entity.FlipDirection]int{entity.FlipTopLeft: 0, entity.FlipTopRight: 1, entity.FlipBottomRight: 2, entity.FlipBottomLeft: 3}

	cosR := math.Cos(angle)
	elevation := 1.0 + float32(math.Sin(tp64*math.Pi))*0.20

	for i, p := range base {
		xr := float64(p.X)*cosA - float64(p.Y)*sinA
		yr := float64(p.X)*sinA + float64(p.Y)*cosA

		if tp < 0.5 {
			switch i {
			case farIdx[dir]:
				xr *= cosR * 1.1
			case nearIdx[dir]:
				xr *= cosR * 0.9
			default:
				xr *= cosR
			}
		} else {
			xr *= cosR
		}

		xf, yf := (xr*cosA+yr*sinA)*float64(elevation), (-xr*sinA+yr*cosA)*float64(elevation)
		g.V[i].DstX, g.V[i].DstY = cx+float32(xf), cy+float32(yf)
	}

	r.extrude(g.V, axisDiagonal, dir, false, 0, tp, thickness, thicknessColor)
	return g
}

func (r *BoardRenderer) extrude(v []ebiten.Vertex, axis axisType, dir entity.FlipDirection, isIdle bool, hover float32, flipTime float32, baseThick float32, clr color.Color) {
	var dx, dy float32
	switch axis {
	case axisHorizontal:
		dx = baseThick
		if dir == entity.FlipLeft {
			dx = -baseThick
		}
	case axisVertical:
		dy = baseThick
		if dir == entity.FlipTop {
			dy = -baseThick
		}
	case axisDiagonal:
		diagThick := baseThick * 0.707
		switch dir {
		case entity.FlipTopLeft:
			dx, dy = -diagThick, -diagThick
		case entity.FlipTopRight:
			dx, dy = diagThick, -diagThick
		case entity.FlipBottomLeft:
			dx, dy = -diagThick, diagThick
		case entity.FlipBottomRight:
			dx, dy = diagThick, diagThick
		}
	}

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

	cr, cg, cb, ca := clr.RGBA()
	fR := float32(cr) / 0xffff
	fG := float32(cg) / 0xffff
	fB := float32(cb) / 0xffff
	fA := float32(ca) / 0xffff

	for i := 4; i < 8; i++ {
		vIdx := i - 4
		localDx, localDy := dx, dy
		if isIdle {
			var factor float32 = 1.0
			switch axis {
			case axisHorizontal:
				isNear := (dir == entity.FlipLeft && (vIdx == 0 || vIdx == 3)) || (dir == entity.FlipRight && (vIdx == 1 || vIdx == 2))
				if isNear {
					factor = 1.0 - (hover * 0.6)
				}
			case axisVertical:
				isNear := (dir == entity.FlipTop && (vIdx == 0 || vIdx == 1)) || (dir == entity.FlipBottom && (vIdx == 2 || vIdx == 3))
				if isNear {
					factor = 1.0 - (hover * 0.6)
				}
			case axisDiagonal:
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
			thicknessAnimi := 1.0 + float32(math.Sin(float64(flipTime)*math.Pi))*0.6
			localDx *= thicknessAnimi
			localDy *= thicknessAnimi
		}
		v[i].DstX, v[i].DstY = v[vIdx].DstX+localDx, v[vIdx].DstY+localDy
		v[i].SrcX, v[i].SrcY = v[vIdx].SrcX, v[vIdx].SrcY
		v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = fR, fG, fB, fA
	}
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
