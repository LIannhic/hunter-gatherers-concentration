package creature

import (
	"testing"

	"github.com/LIannhic/hunter-gatherers-concentration/internal/domain/entity"
)

type MockWorldQuery struct {
	WorldQuery
	ValidMove bool
}

func (m *MockWorldQuery) IsValidMove(pos entity.Position) bool {
	return m.ValidMove
}

func (m *MockWorldQuery) IsWalkable(c *Creature, pos entity.Position) bool {
	return m.ValidMove
}

func TestNavRelativeProgression(t *testing.T) {
	// Routine: Forward, Right, Backward, Left
	pattern := []entity.Position{
		{X: 0, Y: -1}, // Forward
		{X: 1, Y: 0},  // Right
		{X: 0, Y: 1},  // Backward
		{X: -1, Y: 0}, // Left
	}

	nl := &NavigationLogic{
		Type:        NavRelative,
		PatrolRoute: pattern,
		PatrolIndex: 0,
	}

	c := New("test", entity.Position{X: 5, Y: 5})
	c.SetOrientation(entity.DirNorth)

	world := &MockWorldQuery{ValidMove: false} // Blocked!

	// 1st move attempt (Forward)
	dir1 := nl.relative(world, c)
	if nl.PatrolIndex != 1 {
		t.Errorf("Expected PatrolIndex 1 after first attempt, got %d", nl.PatrolIndex)
	}
	if dir1.X != 0 || dir1.Y != -1 {
		t.Errorf("Expected direction {0, -1}, got %v", dir1)
	}

	// 2nd move attempt (Right)
	dir2 := nl.relative(world, c)
	if nl.PatrolIndex != 2 {
		t.Errorf("Expected PatrolIndex 2 after second attempt, got %d", nl.PatrolIndex)
	}
	if dir2.X != 1 || dir2.Y != 0 {
		t.Errorf("Expected direction {1, 0}, got %v", dir2)
	}

	// Now change orientation to East
	c.SetOrientation(entity.DirEast)
	// 3rd move in pattern is Backward {0, 1}
	// Relative to East: Backward is West {-1, 0}
	dir3 := nl.relative(world, c)
	if nl.PatrolIndex != 3 {
		t.Errorf("Expected PatrolIndex 3 after third attempt, got %d", nl.PatrolIndex)
	}
	if dir3.X != -1 || dir3.Y != 0 {
		t.Errorf("Expected direction {-1, 0} (Backward relative to East), got %v", dir3)
	}
}
