package board

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

// Position et Direction mappées depuis le domaine
type Position = entity.Position
type Direction = entity.Direction

const (
	North = entity.DirNorth
	East  = entity.DirEast
	South = entity.DirSouth
	West  = entity.DirWest
)

type FlipDirection = entity.FlipDirection

const (
	FlipTop         = entity.FlipTop
	FlipTopRight    = entity.FlipTopRight
	FlipRight       = entity.FlipRight
	FlipBottomRight = entity.FlipBottomRight
	FlipBottom      = entity.FlipBottom
	FlipBottomLeft  = entity.FlipBottomLeft
	FlipLeft        = entity.FlipLeft
	FlipTopLeft     = entity.FlipTopLeft
	FlipCenter      = entity.FlipCenter
)

func CalculateFlipDirection(tileSize, localX, localY int) FlipDirection {
	return entity.CalculateFlipDirection(tileSize, localX, localY)
}

// Bearing représente l'orientation de la Grille (Cardinaux)
type Bearing int

const (
	BearingNorth Bearing = iota
	BearingEast
	BearingSouth
	BearingWest
	BearingMirror
)

// Slope représente l'inclinaison logique de la Parcelle (Vent, Courant, Piste...)
type Slope int

const (
	SlopeTop Slope = iota
	SlopeTopRight
	SlopeRight
	SlopeBottomRight
	SlopeBottom
	SlopeBottomLeft
	SlopeLeft
	SlopeTopLeft
	SlopeFlat // État neutre / Plat
)

func (s Slope) ToFlipDirection() FlipDirection {
	switch s {
	case SlopeTop:
		return FlipTop
	case SlopeTopRight:
		return FlipTopRight
	case SlopeRight:
		return FlipRight
	case SlopeBottomRight:
		return FlipBottomRight
	case SlopeBottom:
		return FlipBottom
	case SlopeBottomLeft:
		return FlipBottomLeft
	case SlopeLeft:
		return FlipLeft
	case SlopeTopLeft:
		return FlipTopLeft
	default:
		return FlipCenter
	}
}

// --- SYSTÈME ENVIRONNEMENTAL (BIOMES) ---

type BiomeType string

const (
	BiomeDefault BiomeType = "default" // Zones de départ, fin, calmes
	BiomeForest  BiomeType = "forest"  // Uniforme et aléatoire
	BiomeCave    BiomeType = "cave"    // Attraction cardinale vers la cible
	BiomeDesert  BiomeType = "desert"  // Répulsion cardinale depuis la cible
	BiomeSwamp   BiomeType = "swamp"   // Vortex en spirale concentrique
)

type Climate string

const (
	ClimateTemperate Climate = "temperate"
	ClimateHumid     Climate = "humid"
	ClimateArid      Climate = "arid"
)

type Season int

const (
	SeasonAwakening Season = iota
	SeasonZenith
	SeasonDecay
	SeasonSlumber
)

type SuccessionStage int

const (
	StagePreliminary SuccessionStage = iota
	StagePioneer
	StageClimax
)

// --- UTILITAIRES DE GÉOMÉTRIE ET LOGIQUE DES PENTES ---

// ChooseRandomGlobalSlope sélectionne une inclinaison au hasard (exclut le plat)
func ChooseRandomGlobalSlope() Slope {
	return Slope(rand.Intn(8))
}

// InvertSlope inverse une inclinaison logique à 180°
func InvertSlope(s Slope) Slope {
	if s == SlopeFlat {
		return SlopeFlat
	}
	return Slope((int(s) + 4) % 8)
}

// RotateSlope fait pivoter une pente par pas de 45° (positif = droite, négatif = gauche)
func RotateSlope(s Slope, steps int) Slope {
	if s == SlopeFlat {
		return SlopeFlat
	}
	newSlope := (int(s) + steps) % 8
	if newSlope < 0 {
		newSlope += 8
	}
	return Slope(newSlope)
}

// CalculateSlopeDirectionCardinal oriente vers la cible en favorisant les axes horizontaux/verticaux
func CalculateSlopeDirectionCardinal(from, to Position) Slope {
	if from.X == to.X && from.Y == to.Y {
		return SlopeFlat
	}

	dx := to.X - from.X
	dy := to.Y - from.Y

	absDx := dx
	if absDx < 0 {
		absDx = -absDx
	}
	absDy := dy
	if absDy < 0 {
		absDy = -absDy
	}

	// Diagonales parfaites uniquement
	if absDx == absDy {
		if dx > 0 && dy < 0 {
			return SlopeTopRight
		}
		if dx > 0 && dy > 0 {
			return SlopeBottomRight
		}
		if dx < 0 && dy > 0 {
			return SlopeBottomLeft
		}
		return SlopeTopLeft
	}

	// Priorité aux axes cardinaux pour toutes les cases intermédiaires
	if absDx > absDy {
		if dx > 0 {
			return SlopeRight
		}
		return SlopeLeft
	}
	if dy < 0 {
		return SlopeTop
	}
	return SlopeBottom
}

// NextPeripheralPos trouve la case suivante en longeant la couronne de rayon N autour du centre
func NextPeripheralPos(current, center Position, clockwise bool) Position {
	dx := current.X - center.X
	dy := current.Y - center.Y

	absDx := dx
	if absDx < 0 {
		absDx = -absDx
	}
	absDy := dy
	if absDy < 0 {
		absDy = -absDy
	}

	maxDelta := absDx
	if absDy > maxDelta {
		maxDelta = absDy
	}

	if clockwise {
		if dx == -maxDelta && dy > -maxDelta {
			return Position{X: current.X, Y: current.Y - 1}
		}
		if dy == -maxDelta && dx < maxDelta {
			return Position{X: current.X + 1, Y: current.Y}
		}
		if dx == maxDelta && dy < maxDelta {
			return Position{X: current.X, Y: current.Y + 1}
		}
		return Position{X: current.X - 1, Y: current.Y}
	} else {
		if dx == -maxDelta && dy < maxDelta {
			return Position{X: current.X, Y: current.Y + 1}
		}
		if dy == maxDelta && dx < maxDelta {
			return Position{X: current.X + 1, Y: current.Y}
		}
		if dx == maxDelta && dy > -maxDelta {
			return Position{X: current.X, Y: current.Y - 1}
		}
		return Position{X: current.X - 1, Y: current.Y}
	}
}

func DirectionVector(d Direction) Position {
	switch d {
	case North:
		return Position{X: 0, Y: -1}
	case South:
		return Position{X: 0, Y: 1}
	case East:
		return Position{X: 1, Y: 0}
	case West:
		return Position{X: -1, Y: 0}
	}
	return Position{X: 0, Y: 0}
}

// --- STRUCTURES DU PLATEAU ---

type Plot struct {
	Position    Position
	EntitiesID  []string
	StructureID string
	Empty       bool
	LocalStage  SuccessionStage
	Tilt        Slope
	Modifier    PlotModifier
}

func (p *Plot) PushEntity(id string)         { p.EntitiesID = append(p.EntitiesID, id) }
func (p *Plot) PushEntityToBottom(id string) { p.EntitiesID = append([]string{id}, p.EntitiesID...) }
func (p *Plot) PopEntity() (string, bool) {
	if len(p.EntitiesID) == 0 {
		return "", false
	}
	lastIdx := len(p.EntitiesID) - 1
	id := p.EntitiesID[lastIdx]
	p.EntitiesID = p.EntitiesID[:lastIdx]
	return id, true
}

func (p *Plot) String() string {
	return fmt.Sprintf("Plot[%v entities=%v]", p.Position, p.EntitiesID)
}

type PlotModifier struct {
	Concealed    bool
	Obstructed   bool
	LuminousHint bool
}

type Grid struct {
	ID             string
	Width, Height  int
	Biome          BiomeType
	Climate        Climate
	CurrentSeason  Season
	SeasonProgress int
	TimeDilation   float64
	GlobalStage    SuccessionStage
	MainBearing    Bearing
	Plots          map[Position]*Plot

	InitialMatchableCount int
	ExitsState            map[Direction][2]entity.TileState
	ExitsTransform        map[Direction][2]entity.Transformation
	NavigationForcedOpen  bool
	MatchedTargetsCount   int
}

// NewGrid instancie et configure les logiques de pentes selon le biome choisi
func NewGrid(id string, width, height int, biome BiomeType) *Grid {
	g := &Grid{
		ID:             id,
		Width:          width,
		Height:         height,
		Biome:          biome,
		CurrentSeason:  SeasonAwakening,
		GlobalStage:    StagePreliminary,
		Plots:          make(map[Position]*Plot),
		ExitsState:     make(map[Direction][2]entity.TileState),
		ExitsTransform: make(map[Direction][2]entity.Transformation),
	}

	for d := entity.DirNorth; d <= entity.DirWest; d++ {
		g.ExitsState[d] = [2]entity.TileState{entity.Hidden | entity.Blocked, entity.Hidden | entity.Blocked}
		g.ExitsTransform[d] = [2]entity.Transformation{entity.TransIdentity, entity.TransIdentity}
	}

	// ÉTAPE 1 : Remplissage initial de la carte à plat
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pos := Position{X: x, Y: y}
			g.Plots[pos] = &Plot{
				Position:   pos,
				EntitiesID: []string{},
				LocalStage: StagePreliminary,
				Tilt:       SlopeFlat,
			}
		}
	}

	// Si le biome est Default, on s'arrête ici : tout reste plat.
	if biome == BiomeDefault {
		return g
	}

	// ÉTAPE 2 : Sélection de la parcelle repère secrète
	targetPos := Position{X: rand.Intn(width), Y: rand.Intn(height)}

	// Préparation de la pente uniforme pour la forêt
	forestGlobalSlope := ChooseRandomGlobalSlope()

	// ÉTAPE 3 : Application des règles directionnelles
	switch biome {
	case BiomeForest:
		for _, plot := range g.Plots {
			plot.Tilt = forestGlobalSlope
		}

	case BiomeCave:
		for pos, plot := range g.Plots {
			plot.Tilt = CalculateSlopeDirectionCardinal(pos, targetPos)
		}

	case BiomeDesert:
		for pos, plot := range g.Plots {
			plot.Tilt = InvertSlope(CalculateSlopeDirectionCardinal(pos, targetPos))
		}

	case BiomeSwamp:
		g.ApplySpiralVortex(targetPos, true, true)
	}

	return g
}

// ApplySpiralVortex propage un effet de remous en enroulant les pentes couronne par couronne (N)
func (g *Grid) ApplySpiralMountain(center Position, clockwise bool, shiftRight bool) {
	if plot, err := g.Get(center); err == nil {
		plot.Tilt = SlopeFlat // Le centre reste neutre
	}

	initialShift := -1
	if shiftRight {
		initialShift = 1
	}

	maxRadius := g.Width
	if g.Height > maxRadius {
		maxRadius = g.Height
	}

	// Voyage à travers les niveaux de périphérie (N)
	for n := 1; n <= maxRadius; n++ {
		// Case de départ du bras de la couronne N (au Nord du centre)
		startPos := Position{X: center.X, Y: center.Y - n}
		currentSlope := RotateSlope(SlopeTop, initialShift)
		currentPos := startPos

		totalSteps := 8 * n // Périmètre d'une couronne de rayon N

		for step := 0; step < totalSteps; step++ {
			if plot, err := g.Get(currentPos); err == nil {
				plot.Tilt = currentSlope
			}

			// Règle N : On répète l'inclinaison N fois avant de la faire pivoter de 45°
			if step > 0 && step%n == 0 {
				currentSlope = RotateSlope(currentSlope, initialShift)
			}

			// Avancée sur le périmètre de la couronne
			currentPos = NextPeripheralPos(currentPos, center, clockwise)
		}
	}
}

// ApplySpiralVortex propage un effet de remous en enroulant les pentes couronne par couronne (N).
func (g *Grid) ApplySpiralVortex(center Position, clockwise bool, shiftRight bool) {
	// 1. Le centre du vortex reste plat
	if centerPlot, ok := g.Plots[center]; ok {
		centerPlot.Tilt = SlopeFlat
	}

	// Détermination du décalage initial (1 pour horaire/droite, -1 pour anti-horaire)
	initialShift := -1
	if shiftRight {
		initialShift = 1
	}

	// Rayon maximum pour couvrir toute la grille, même si le centre est excentré
	maxRadius := g.Width
	if g.Height > maxRadius {
		maxRadius = g.Height
	}

	// 2. Voyage à travers les niveaux de périphérie (couronne N)
	for n := 1; n <= maxRadius; n++ {
		// On commence chaque couronne au Nord du centre, à une distance N
		currentPos := Position{X: center.X, Y: center.Y - n}
		currentSlope := RotateSlope(SlopeTop, initialShift)

		// Périmètre exact d'une couronne carrée de rayon N
		totalSteps := 8 * n
		stepsInCurrentSlope := 0

		for step := 0; step < totalSteps; step++ {
			// On cible la position dans la map du plateau
			if plot, ok := g.Plots[currentPos]; ok {
				plot.Tilt = currentSlope
			}

			// Progression sur le périmètre de la couronne actuelle
			currentPos = NextPeripheralPos(currentPos, center, clockwise)
			stepsInCurrentSlope++

			// RÈGLE N : On a répété la pente N fois ? On pivote l'inclinaison de 45°
			if stepsInCurrentSlope == n {
				stepsInCurrentSlope = 0
				currentSlope = RotateSlope(currentSlope, initialShift)
			}
		}
	}
}

// --- MÉTHODES CLASSIQUES DE LA GRILLE ---

func (g *Grid) IsValid(pos Position) bool {
	return pos.X >= 0 && pos.X < g.Width && pos.Y >= 0 && pos.Y < g.Height
}

func (g *Grid) Get(pos Position) (*Plot, error) {
	if !g.IsValid(pos) {
		return nil, fmt.Errorf("position %v hors limites", pos)
	}
	p, ok := g.Plots[pos]
	if !ok {
		return nil, errors.New("parcelle inexistante")
	}
	return p, nil
}

func (g *Grid) PlaceEntity(pos Position, entityID string) error {
	plot, err := g.Get(pos)
	if err != nil {
		return err
	}
	plot.PushEntity(entityID)
	return nil
}

func (g *Grid) PlaceEntityAtBottom(pos Position, entityID string) error {
	plot, err := g.Get(pos)
	if err != nil {
		return err
	}
	plot.PushEntityToBottom(entityID)
	return nil
}

func (g *Grid) RemoveEntity(pos Position, entityID string) (string, error) {
	plot, err := g.Get(pos)
	if err != nil {
		return "", err
	}

	foundIdx := -1
	for i, id := range plot.EntitiesID {
		if id == entityID {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return "", fmt.Errorf("entité %s non trouvée à la position %v", entityID, pos)
	}

	plot.EntitiesID = append(plot.EntitiesID[:foundIdx], plot.EntitiesID[foundIdx+1:]...)
	return entityID, nil
}

func (g *Grid) GetNeighbors(pos Position) []*Plot {
	var neighbors []*Plot
	dirs := []Position{
		{X: 0, Y: -1}, {X: 0, Y: 1}, {X: 1, Y: 0}, {X: -1, Y: 0},
		{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1},
	}
	for _, d := range dirs {
		targetPos := pos.Add(d)
		if plot, err := g.Get(targetPos); err == nil {
			neighbors = append(neighbors, plot)
		}
	}
	return neighbors
}

func (g *Grid) GetTileAt(x, y int) (*Plot, error) {
	return g.Get(Position{X: x, Y: y})
}

func (g *Grid) RotateClockwise() {
	g.MainBearing = Bearing((int(g.MainBearing) + 1) % 4)

	newPlots := make(map[Position]*Plot)
	for oldPos, plot := range g.Plots {
		newPos := g.TransformPosition(oldPos)
		plot.Position = newPos

		// Note de cohérence : Si vos Slopes logiques dépendent de l'orientation absolue de la grille,
		// appliquez ici un pivot sur plot.Tilt (ex: plot.Tilt = RotateSlope(plot.Tilt, 2))

		newPlots[newPos] = plot
	}
	g.Plots = newPlots

	newExitsState := make(map[Direction][2]entity.TileState)
	newExitsState[East] = g.ExitsState[North]
	newExitsState[South] = g.ExitsState[East]
	newExitsState[West] = g.ExitsState[South]
	newExitsState[North] = g.ExitsState[West]
	g.ExitsState = newExitsState
}

func (g *Grid) TransformPosition(pos Position) Position {
	return Position{
		X: g.Height - 1 - pos.Y,
		Y: pos.X,
	}
}
