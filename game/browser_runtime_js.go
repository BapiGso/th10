//go:build js

package game

import (
	"th10/pacing"
	"time"
)

// StartBrowserClock discards host time spent downloading/waiting for a gesture.
// It does not reset a scene, input stream, score, or either original RNG.
func (g *Game) StartBrowserClock() {
	g.start = time.Now()
	g.pace = pacing.Pacer{}
	g.fresh, g.fps, g.fpsFrames, g.fpsT = false, 0, 0, 0
}
