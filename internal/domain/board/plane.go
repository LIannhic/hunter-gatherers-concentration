package board

import (
	"fmt"
)

// DiscoveryState représente l'état de découverte d'une zone sur la minimap
type DiscoveryState int

const (
	StateHidden   DiscoveryState = iota // Zone inconnue
	StateAdjacent                       // Zone limitrophe non visitée
	StateVisited                        // Zone déjà explorée
)

// DreamPlane représente le "mega-board" composé de plusieurs zones (Grids)
type DreamPlane struct {
	ID              string
	Zones           map[string]*Grid
	Coords          map[string]Position               // GridID -> Coords in the plane layout
	Connections     map[string]map[Direction]string // GridID -> Direction -> TargetGridID
	DiscoveryStates map[string]DiscoveryState         // GridID -> État de découverte
	StartZoneID     string
	EndZoneID       string
}

func NewDreamPlane(id string) *DreamPlane {
	return &DreamPlane{
		ID:              id,
		Zones:           make(map[string]*Grid),
		Coords:          make(map[string]Position),
		Connections:     make(map[string]map[Direction]string),
		DiscoveryStates: make(map[string]DiscoveryState),
	}
}

func (p *DreamPlane) AddZone(grid *Grid) {
	p.Zones[grid.ID] = grid
}

func (p *DreamPlane) Connect(fromID, toID string, dir Direction) {
	if p.Connections[fromID] == nil {
		p.Connections[fromID] = make(map[Direction]string)
	}
	p.Connections[fromID][dir] = toID

	// Connection réciproque
	opposite := p.OppositeDirection(dir)
	if p.Connections[toID] == nil {
		p.Connections[toID] = make(map[Direction]string)
	}
	p.Connections[toID][opposite] = fromID
}

func (p *DreamPlane) OppositeDirection(dir Direction) Direction {
	switch dir {
	case North:
		return South
	case South:
		return North
	case East:
		return West
	case West:
		return East
	}
	return dir
}

func (p *DreamPlane) GetConnectedZone(fromID string, dir Direction) (string, bool) {
	if conn, ok := p.Connections[fromID]; ok {
		targetID, ok := conn[dir]
		return targetID, ok
	}
	return "", false
}

func (p *DreamPlane) String() string {
	return fmt.Sprintf("DreamPlane[%s, zones=%d]", p.ID, len(p.Zones))
}
