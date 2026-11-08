package stage

import "th10/game"

// SceneMode keeps the original supervisor's gameplay number (7), including
// mode 1 playback. The pending requests 10..13 normalize to this number.
func (*Stage) SceneMode() game.SceneID    { return game.SceneStage }
func (*Playback) SceneMode() game.SceneID { return game.SceneStage }

func (s *Stage) SceneFlags() uint32 { return s.sceneFlags }

// SceneEnter's construction has already run through New/OnContinue;
// do not reset a post-ending Stage returned by the preserved Ending.After.
func (*Stage) SceneEnter(*game.Game) error { return nil }

// SceneLeave is the host's FUN_00418150 -> FUN_00417c80 VM teardown.
// Ending.After intentionally retains the completed stage's frozen picture
// for t5's post-ending capture/menu; every other leave releases its worlds.
func (s *Stage) SceneLeave(_ *game.Game, next game.SceneID) {
	if next == game.SceneEnding {
		return
	}
	s.ctx.World.Clear()
	s.ctx.Menu.world.Clear()
}

func (*Playback) SceneEnter(*game.Game) error                  { return nil }
func (p *Playback) SceneLeave(g *game.Game, next game.SceneID) { p.stage.SceneLeave(g, next) }
