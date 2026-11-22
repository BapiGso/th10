//go:build th10nostory

package selector

import "th10/game"

// playEnding without th10nostory's removed ending package: stage 6 goes
// straight to the completion menu the ending would have opened, Extra back.
func playEnding(g *game.Game, _ *game.GameState, back game.Scene, after func(*game.Game) game.Scene) game.Scene {
	if after != nil {
		return after(g)
	}
	return back
}
