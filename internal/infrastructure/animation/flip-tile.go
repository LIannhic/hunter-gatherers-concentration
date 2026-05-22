package animation

import (
	"fmt"
	"image/color"
	"math"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	tileSize     = float32(ui.TileSize) // 87.5 selon layout.go
	faceSize     = 80.0
	thickness    = 4.0
	flipDuration = 0.75
)

const (
	screenW = ui.ScreenWidth
	screenH = ui.ScreenHeight
)

type Game struct {
	Tiles []*Tile
}

type Vec2 struct {
	X, Y float32
}

type ThickGeometry struct {
	V []ebiten.Vertex
	I []uint16
}

type FlipDir int

const (
	FlipLeft FlipDir = iota
	FlipRight
	FlipTop
	FlipBottom
	FlipTopLeft
	FlipTopRight
	FlipBottomLeft
	FlipBottomRight
)

type AxisType int

const (
	AxisHorizontal AxisType = iota
	AxisVertical
	AxisDiagonal
)

type Tile struct {
	X, Y        float32
	Front, Back *ebiten.Image

	FaceFront bool

	Dir        FlipDir
	ActiveAxis AxisType

	Hover       float32
	Time        float64
	IsAnimating bool

	ImpactT    float64
	IsBouncing bool
}

func NewTile(x, y float32, front, back *ebiten.Image) *Tile {
	return &Tile{
		X:         x,
		Y:         y,
		Front:     front,
		Back:      back,
		FaceFront: true,
	}
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()

	for i, t := range g.Tiles {
		hover := pointInRect(float32(mx), float32(my), t.X, t.Y, tileSize, tileSize)

		if t.IsAnimating {
			t.Time += 1.0 / 60.0 / flipDuration

			tp := smooth(t.Time)
			elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.20
			tailleActuelle := faceSize * elevation

			fmt.Printf("[Tuile %d] Animation Flip - Longueur relative : %.2f pixels (Échelle: %.1f%%)\n",
				i, tailleActuelle, elevation*100)

			if t.Time >= 1 {
				t.Time = 0
				t.IsAnimating = false
				t.FaceFront = !t.FaceFront
				t.IsBouncing = true
				t.ImpactT = 0
			}
			continue
		}

		if t.IsBouncing {
			t.ImpactT += 0.04
			if t.ImpactT >= 1.0 {
				t.IsBouncing = false
				t.ImpactT = 0
			}
		}

		if hover {
			dir := detectDirection(float32(mx), float32(my), t.X, t.Y)
			t.Dir = dir

			switch dir {
			case FlipLeft, FlipRight:
				t.ActiveAxis = AxisHorizontal
			case FlipTop, FlipBottom:
				t.ActiveAxis = AxisVertical
			default:
				t.ActiveAxis = AxisDiagonal
			}

			t.Hover += 0.1
			if t.Hover > 1 {
				t.Hover = 1
			}

			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				if !t.IsBouncing {
					t.IsAnimating = true
					t.Time = 0
					t.Hover = 0
					t.IsBouncing = false
				}
			}
		} else {
			t.Hover -= 0.1
			if t.Hover < 0 {
				t.Hover = 0
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{24, 24, 28, 255})

	for _, t := range g.Tiles {
		cx, cy := t.X+tileSize/2, t.Y+tileSize/2

		// Repères concentriques adaptés à la nouvelle taille
		drawConcentricSquare(screen, cx, cy, faceSize, color.RGBA{60, 60, 70, 255})
		drawConcentricSquare(screen, cx, cy, faceSize*1.05, color.RGBA{0, 180, 100, 80})
		drawConcentricSquare(screen, cx, cy, faceSize*1.10, color.RGBA{255, 200, 0, 120})

		img := t.Front
		if t.IsAnimating && t.Time >= 0.5 {
			if t.FaceFront {
				img = t.Back
			} else {
				img = t.Front
			}
		} else if !t.FaceFront {
			img = t.Back
		}

		var geo ThickGeometry
		if t.IsAnimating {
			switch t.ActiveAxis {
			case AxisHorizontal:
				geo = generateFlipH(t)
			case AxisVertical:
				geo = generateFlipV(t)
			case AxisDiagonal:
				geo = generateFlipDiag(t)
			}
		} else {
			geo = generateIdle(t)
		}

		drawPart(screen, geo.V, geo.I[:6], img)
		drawPart(screen, geo.V, geo.I[6:12], t.Back)

		showRight, showLeft, showTop, showBottom := false, false, false, false
		if t.Hover > 0 || t.IsAnimating || t.IsBouncing {
			switch t.Dir {
			case FlipLeft: showLeft = true
			case FlipRight: showRight = true
			case FlipTop: showTop = true
			case FlipBottom: showBottom = true
			case FlipTopLeft: showTop, showLeft = true, true
			case FlipTopRight: showTop, showRight = true, true
			case FlipBottomLeft: showBottom, showLeft = true, true
			case FlipBottomRight: showBottom, showRight = true, true
			}
		}

		if showTop { drawPart(screen, geo.V, geo.I[12:18], t.Back) }
		if showRight { drawPart(screen, geo.V, geo.I[18:24], t.Back) }
		if showBottom { drawPart(screen, geo.V, geo.I[24:30], t.Back) }
		if showLeft { drawPart(screen, geo.V, geo.I[30:36], t.Back) }
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	return screenW, screenH
}

func drawConcentricSquare(screen *ebiten.Image, cx, cy, size float32, clr color.Color) {
	half := size / 2
	vector.StrokeRect(screen, cx-half, cy-half, size, size, 1, clr, false)
}

func createGeometry() ThickGeometry {
	return ThickGeometry{
		V: make([]ebiten.Vertex, 8),
		I: []uint16{
			0, 1, 2, 0, 2, 3, 4, 5, 6, 4, 6, 7, // Front & Back
			4, 5, 1, 4, 1, 0, 5, 6, 2, 5, 2, 1, // Top & Right
			6, 7, 3, 6, 3, 2, 7, 4, 0, 7, 0, 3, // Bottom & Left
		},
	}
}

func initVerts(v []ebiten.Vertex) {
	for i := range v {
		v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = 1, 1, 1, 1
	}
}

func setFaceUV(v []ebiten.Vertex) {
	v[0].SrcX, v[0].SrcY = 0, 0
	v[1].SrcX, v[1].SrcY = faceSize, 0
	v[2].SrcX, v[2].SrcY = faceSize, faceSize
	v[3].SrcX, v[3].SrcY = 0, faceSize
}

func generateIdle(t *Tile) ThickGeometry {
	g := createGeometry()
	initVerts(g.V)
	setFaceUV(g.V)

	margin := (tileSize - faceSize) / 2
	x, y := t.X+margin, t.Y+margin
	h := t.Hover * (faceSize * 0.08)
	cx, cy := x+faceSize/2, y+faceSize/2

	switch t.Dir {
	case FlipLeft:
		g.V[0].DstX, g.V[0].DstY = x, y+h*0.25
		g.V[1].DstX, g.V[1].DstY = x+faceSize-h*0.25, y-h*0.75
		g.V[2].DstX, g.V[2].DstY = x+faceSize-h*0.25, y+faceSize+h*0.75
		g.V[3].DstX, g.V[3].DstY = x, y+faceSize-h*0.25
	case FlipRight:
		g.V[0].DstX, g.V[0].DstY = x+h*0.25, y-h*0.75
		g.V[1].DstX, g.V[1].DstY = x+faceSize, y+h*0.25
		g.V[2].DstX, g.V[2].DstY = x+faceSize, y+faceSize-h*0.25
		g.V[3].DstX, g.V[3].DstY = x+h*0.25, y+faceSize+h*0.75
	case FlipTop:
		g.V[0].DstX, g.V[0].DstY = x+h*0.25, y
		g.V[1].DstX, g.V[1].DstY = x+faceSize-h*0.25, y
		g.V[2].DstX, g.V[2].DstY = x+faceSize+h*0.75, y+faceSize-h*0.25
		g.V[3].DstX, g.V[3].DstY = x-h*0.75, y+faceSize-h*0.25
	case FlipBottom:
		g.V[0].DstX, g.V[0].DstY = x-h*0.75, y+h*0.25
		g.V[1].DstX, g.V[1].DstY = x+faceSize+h*0.75, y+h*0.25
		g.V[2].DstX, g.V[2].DstY = x+faceSize, y+faceSize
		g.V[3].DstX, g.V[3].DstY = x, y+faceSize
	case FlipTopLeft:
		g.V[0].DstX, g.V[0].DstY = x+h*0.25, y+h*0.25
		g.V[1].DstX, g.V[1].DstY = x+faceSize-h*0.5, y-h*0.75
		g.V[2].DstX, g.V[2].DstY = x+faceSize-h*0.75, y+faceSize-h*0.75
		g.V[3].DstX, g.V[3].DstY = x-h*0.75, y+faceSize-h*0.5
	case FlipTopRight:
		g.V[0].DstX, g.V[0].DstY = x+h*0.5, y-h*0.75
		g.V[1].DstX, g.V[1].DstY = x+faceSize-h*0.25, y-h*0.25
		g.V[2].DstX, g.V[2].DstY = x+faceSize+h*0.75, y+faceSize-h*0.5
		g.V[3].DstX, g.V[3].DstY = x+h*0.75, y+faceSize-h*0.75
	case FlipBottomRight:
		g.V[0].DstX, g.V[0].DstY = x+h*0.75, y+h*0.75
		g.V[1].DstX, g.V[1].DstY = x+faceSize+h*0.75, y-h*0.5
		g.V[2].DstX, g.V[2].DstY = x+faceSize-h*0.25, y+faceSize-h*0.25
		g.V[3].DstX, g.V[3].DstY = x+h*0.5, y+faceSize+h*0.75
	case FlipBottomLeft:
		g.V[0].DstX, g.V[0].DstY = x-h*0.75, y-h*0.5
		g.V[1].DstX, g.V[1].DstY = x+faceSize-h*0.75, y+h*0.75
		g.V[2].DstX, g.V[2].DstY = x+faceSize-h*0.5, y+faceSize+h*0.75
		g.V[3].DstX, g.V[3].DstY = x+h*0.25, y+faceSize-h*0.25
	default:
		g.V[0].DstX, g.V[0].DstY = x, y
		g.V[1].DstX, g.V[1].DstY = x+faceSize, y
		g.V[2].DstX, g.V[2].DstY = x+faceSize, y+faceSize
		g.V[3].DstX, g.V[3].DstY = x, y+faceSize
	}

	var bAmp float32
	if t.IsBouncing {
		p := t.ImpactT
		switch {
		case p < 0.25: bAmp = float32(math.Sin((p/0.25)*math.Pi/2)) * (faceSize * 0.06)
		case p < 0.5:  bAmp = (1.0 - float32((p-0.25)/0.25)) * (faceSize * 0.06)
		case p < 0.75: bAmp = -float32(math.Sin(((p-0.5)/0.25)*math.Pi/2)) * (faceSize * 0.025)
		}
	}

	for i := 0; i < 4; i++ {
		dx, dy := g.V[i].DstX - cx, g.V[i].DstY - cy
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if dist > 0.0001 && bAmp != 0 {
			g.V[i].DstX += (dx / dist) * bAmp
			g.V[i].DstY += (dy / dist) * bAmp
		}
	}

	extrude(g.V, t.ActiveAxis, t.Dir, true, t.Hover, 0)
	return g
}

func generateFlipH(t *Tile) ThickGeometry {
	g := createGeometry(); initVerts(g.V); setFaceUV(g.V)
	margin := (tileSize - faceSize) / 2
	x, y := t.X+margin, t.Y+margin
	cx, cy := x+faceSize/2, y+faceSize/2
	tp := smooth(t.Time)
	s := float32(math.Cos(float64(tp) * math.Pi))
	if t.Dir == FlipLeft { s = -s }
	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.20
	half, height := (faceSize/2)*s*elevation, (faceSize/2)*elevation
	g.V[0].DstX, g.V[0].DstY = cx-half, cy-height
	g.V[1].DstX, g.V[1].DstY = cx+half, cy-height
	g.V[2].DstX, g.V[2].DstY = cx+half, cy+height
	g.V[3].DstX, g.V[3].DstY = cx-half, cy+height
	extrude(g.V, t.ActiveAxis, t.Dir, false, 0, tp)
	return g
}

func generateFlipV(t *Tile) ThickGeometry {
	g := createGeometry(); initVerts(g.V); setFaceUV(g.V)
	margin := (tileSize - faceSize) / 2
	x, y := t.X+margin, t.Y+margin
	cx, cy := x+faceSize/2, y+faceSize/2
	tp := smooth(t.Time)
	s := float32(math.Cos(float64(tp) * math.Pi))
	if t.Dir == FlipTop { s = -s }
	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.20
	half, width := (faceSize/2)*s*elevation, (faceSize/2)*elevation
	g.V[0].DstX, g.V[0].DstY = cx-width, cy-half
	g.V[1].DstX, g.V[1].DstY = cx+width, cy-half
	g.V[2].DstX, g.V[2].DstY = cx+width, cy+half
	g.V[3].DstX, g.V[3].DstY = cx-width, cy+half
	extrude(g.V, t.ActiveAxis, t.Dir, false, 0, tp)
	return g
}

func generateFlipDiag(t *Tile) ThickGeometry {
	g := createGeometry(); initVerts(g.V); setFaceUV(g.V)
	margin := (tileSize - faceSize) / 2
	x, y := t.X+margin, t.Y+margin
	cx, cy := x+faceSize/2, y+faceSize/2
	tp := smooth(t.Time)
	angle := float64(tp) * math.Pi
	halfFace := faceSize / 2
	base := [4]Vec2{{-halfFace, -halfFace}, {halfFace, -halfFace}, {halfFace, halfFace}, {-halfFace, halfFace}}
	var a float64
	if t.Dir == FlipTopRight || t.Dir == FlipBottomLeft { a = math.Pi / 4 } else { a = -math.Pi / 4 }
	cosA, sinA := math.Cos(a), math.Sin(a)
	farIdx := map[FlipDir]int{FlipTopLeft: 2, FlipTopRight: 3, FlipBottomRight: 0, FlipBottomLeft: 1}
	nearIdx := map[FlipDir]int{FlipTopLeft: 0, FlipTopRight: 1, FlipBottomRight: 2, FlipBottomLeft: 3}
	cosR := math.Cos(angle)
	elevation := 1.0 + float32(math.Sin(float64(tp)*math.Pi))*0.20
	for i, p := range base {
		xr := float64(p.X)*cosA - float64(p.Y)*sinA
		yr := float64(p.X)*sinA + float64(p.Y)*cosA
		if tp < 0.5 {
			switch {
			case i == farIdx[t.Dir]: xr *= cosR * 1.1
			case i == nearIdx[t.Dir]: xr *= cosR * 0.9
			default: xr *= cosR
			}
		} else { xr *= cosR }
		xf, yf := (xr*cosA+yr*sinA)*float64(elevation), (-xr*sinA+yr*cosA)*float64(elevation)
		g.V[i].DstX, g.V[i].DstY = cx+float32(xf), cy+float32(yf)
	}
	extrude(g.V, t.ActiveAxis, t.Dir, false, 0, tp)
	return g
}

func extrude(v []ebiten.Vertex, axis AxisType, dir FlipDir, isIdle bool, hover float32, flipTime float32) {
	baseThick := thickness
	var dx, dy float32
	switch axis {
	case AxisHorizontal: dx = baseThick; if dir == FlipLeft { dx = -baseThick }
	case AxisVertical: dy = baseThick; if dir == FlipTop { dy = -baseThick }
	case AxisDiagonal:
		diagThick := baseThick * 0.707
		switch dir {
		case FlipTopLeft: dx, dy = -diagThick, -diagThick
		case FlipTopRight: dx, dy = diagThick, -diagThick
		case FlipBottomLeft: dx, dy = -diagThick, diagThick
		case FlipBottomRight: dx, dy = diagThick, diagThick
		}
	}
	var near, far, adj1, adj2 int
	switch dir {
	case FlipTopLeft: near, far, adj1, adj2 = 0, 2, 1, 3
	case FlipTopRight: near, far, adj1, adj2 = 1, 3, 0, 2
	case FlipBottomRight: near, far, adj1, adj2 = 2, 0, 1, 3
	case FlipBottomLeft: near, far, adj1, adj2 = 3, 1, 0, 2
	}
	for i := 4; i < 8; i++ {
		vIdx := i - 4
		localDx, localDy := dx, dy
		if isIdle {
			var factor float32 = 1.0
			switch axis {
			case AxisHorizontal:
				if (dir == FlipLeft && (vIdx == 0 || vIdx == 3)) || (dir == FlipRight && (vIdx == 1 || vIdx == 2)) { factor = 1.0 - (hover * 0.6) }
			case AxisVertical:
				if (dir == FlipTop && (vIdx == 0 || vIdx == 1)) || (dir == FlipBottom && (vIdx == 2 || vIdx == 3)) { factor = 1.0 - (hover * 0.6) }
			case AxisDiagonal:
				switch vIdx {
				case near: factor = (1.0 - (hover * 0.85)) + 0.125
				case adj1, adj2: factor = 1.0 - (hover * 0.3)
				case far: factor = 1.0
				}
			}
			localDx *= factor; localDy *= factor
		} else {
			thicknessAnimi := 1.0 + float32(math.Sin(float64(flipTime)*math.Pi))*0.6
			localDx *= thicknessAnimi; localDy *= thicknessAnimi
		}
		v[i].DstX, v[i].DstY = v[vIdx].DstX+localDx, v[vIdx].DstY+localDy
		v[i].SrcX, v[i].SrcY = v[vIdx].SrcX, v[vIdx].SrcY
		v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = 0.72, 0.72, 0.72, 1
	}
}

func drawPart(screen *ebiten.Image, v []ebiten.Vertex, indices []uint16, img *ebiten.Image) {
	screen.DrawTriangles(v, indices, img, &ebiten.DrawTrianglesOptions{Filter: ebiten.FilterLinear})
}

func detectDirection(mx, my, x, y float32) FlipDir {
	lx, ly := mx-x, my-y
	s := tileSize / 3
	left, right, top, bottom := lx < s, lx > s*2, ly < s, ly > s*2
	switch {
	case top && left: return FlipTopLeft
	case top && right: return FlipTopRight
	case bottom && left: return FlipBottomLeft
	case bottom && right: return FlipBottomRight
	case left: return FlipLeft
	case right: return FlipRight
	case top: return FlipTop
	default: return FlipBottom
	}
}

func pointInRect(px, py, x, y, w, h float32) bool {
	return px >= x && px <= x+w && py >= y && py <= y+h
}

func smooth(t float64) float32 {
	t -= 0.5
	return float32(4*t*t*t + 0.5)
}

func makeFace(bg, fg color.Color) *ebiten.Image {
	s := int(faceSize)
	img := ebiten.NewImage(s, s)
	img.Fill(bg)
	for i := 0; i < s; i++ {
		img.Set(i, 0, color.White); img.Set(i, s-1, color.White)
		img.Set(0, i, color.White); img.Set(s-1, i, color.White)
	}
	return img
}