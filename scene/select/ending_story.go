//go:build !th10nostory

package selector

import (
	"th10/game"
	"th10/scene/ending"
)

// playEnding runs the ending and staff roll; after (stage 6 only) opens the
// completion menu once they finish.
func playEnding(_ *game.Game, st *game.GameState, back game.Scene, after func(*game.Game) game.Scene) game.Scene {
	e := ending.New(st, st.Stage == game.StageExtra, back)
	e.After = after
	return game.NumberScene(game.SceneEnding, e) // preserve e.After while numbering request 14
}
