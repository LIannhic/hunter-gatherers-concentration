package board

import (
	"math/rand"
)

// ChooseRandomGlobalSlope sélectionne une inclinaison au hasard (exclut le plat).
func ChooseRandomGlobalSlope() Slope {
	return Slope(rand.Intn(8))
}

// InvertSlope inverse une inclinaison logique à 180°.
func InvertSlope(s Slope) Slope {
	if s == SlopeFlat {
		return SlopeFlat
	}
	return Slope((int(s) + 4) % 8)
}

// RotateSlope fait pivoter une pente par pas de 45°.
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

// CalculateSlopeDirectionCardinal oriente vers la cible en favorisant les axes horizontaux/verticaux.
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

// NextPeripheralPos trouve la case suivante en longeant la couronne de rayon N autour du centre.
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

// DirectionVector convertit une direction abstraite en vecteur de position relative.
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

// ApplySpiralVortex propage un effet de remous en enroulant les pentes couronne par couronne.
func ApplySpiralVortex(plots map[Position]*Plot, center Position, width, height int, clockwise bool, shiftRight bool) {
	if centerPlot, ok := plots[center]; ok {
		centerPlot.Tilt = SlopeFlat
	}

	initialShift := -1
	if shiftRight {
		initialShift = 1
	}

	maxRadius := width
	if height > maxRadius {
		maxRadius = height
	}

	for n := 1; n <= maxRadius; n++ {
		currentPos := Position{X: center.X, Y: center.Y - n}
		currentSlope := RotateSlope(SlopeTop, initialShift)

		totalSteps := 8 * n
		stepsInCurrentSlope := 0

		for step := 0; step < totalSteps; step++ {
			if plot, ok := plots[currentPos]; ok {
				plot.Tilt = currentSlope
			}

			currentPos = NextPeripheralPos(currentPos, center, clockwise)
			stepsInCurrentSlope++

			if stepsInCurrentSlope == n {
				stepsInCurrentSlope = 0
				currentSlope = RotateSlope(currentSlope, initialShift)
			}
		}
	}
}

// ApplySpiralMountain applique un relief en spirale (utilisé pour certains biomes).
func ApplySpiralMountain(plots map[Position]*Plot, center Position, width, height int, clockwise bool, shiftRight bool) {
	initialShift := -1
	if shiftRight {
		initialShift = 1
	}

	maxRadius := width
	if height > maxRadius {
		maxRadius = height
	}

	for n := 1; n <= maxRadius; n++ {
		startPos := Position{X: center.X, Y: center.Y - n}
		currentSlope := RotateSlope(SlopeTop, initialShift)
		currentPos := startPos

		totalSteps := 8 * n

		for step := 0; step < totalSteps; step++ {
			if plot, ok := plots[currentPos]; ok {
				plot.Tilt = currentSlope
			}

			if step > 0 && step%n == 0 {
				currentSlope = RotateSlope(currentSlope, initialShift)
			}

			currentPos = NextPeripheralPos(currentPos, center, clockwise)
		}
	}
}
