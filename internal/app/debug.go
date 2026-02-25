package app

import (
	"fmt"
	"time"
)

// Debug stats pour suivre ce qui se passe
type DebugStats struct {
	lastPrint    time.Time
	frameCount   int
	spawnCount   int
	actionCount  int
}

func NewDebugStats() *DebugStats {
	return &DebugStats{
		lastPrint: time.Now(),
	}
}

func (d *DebugStats) Frame() {
	d.frameCount++
	if time.Since(d.lastPrint) > time.Second {
		fmt.Printf("[DEBUG] FPS: %d, Spawns: %d, Actions: %d\n", 
			d.frameCount, d.spawnCount, d.actionCount)
		d.frameCount = 0
		d.spawnCount = 0
		d.actionCount = 0
		d.lastPrint = time.Now()
	}
}

func (d *DebugStats) Spawn() {
	d.spawnCount++
}

func (d *DebugStats) Action() {
	d.actionCount++
}
