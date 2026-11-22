package selector

import (
	"th10/game"
	"th10/scene/stage"
	stagescript "th10/scene/stage/script"
)

// Launch delays the native scene-7 constructor until the request is consumed.
// Existing OnContinue, NextScene and post-ending menu callbacks stay intact.
func Launch(g *game.Game, state *game.GameState, back game.Scene) error {
	g.GoToFactory(game.SceneStage, func(g *game.Game) (game.Scene, error) {
		g.BeginRun(state)
		return buildStage(g, state, back), nil
	})
	return nil
}

func buildStage(g *game.Game, state *game.GameState, back game.Scene) *stage.Stage {
	s := stage.New(g, state, stagescript.New(state.Stage))
	s.OnGameOver = func(_ *game.Game, _ *game.GameState) game.Scene { return back }
	s.OnContinue = func(g *game.Game, st *game.GameState) game.Scene {
		st.RestartStage() // FUN_004218d0 request 13, t5/t8's native fresh-run path
		g.BeginRun(st)
		return buildStage(g, st, back)
	}
	s.OnRetry = func(g *game.Game, st *game.GameState) game.Scene {
		st.RetryInitialStage()
		g.BeginRun(st)
		return buildStage(g, st, back)
	}
	s.NextScene = func(g *game.Game, st *game.GameState) game.Scene {
		if st.SingleStage {
			return back
		}
		next := st.NextStage()
		if next == 0 {
			if st.Stage != game.StageExtra {
				g.MarkClear(st)
			}
			var after func(*game.Game) game.Scene
			if st.Stage == game.StageCount {
				after = func(g *game.Game) game.Scene {
					s.NextScene = func(*game.Game, *game.GameState) game.Scene { return back }
					s.OpenCompletion(g)
					return s
				}
			}
			return playEnding(g, st, back, after)
		}
		st.Stage, st.Frame = next, 0
		st.BeginStageTransition()
		return buildStage(g, st, back)
	}
	return s
}

func NewDebugStageWithPower(g *game.Game, back game.Scene, stageNum, character, shotType, difficulty, power int) game.Scene {
	state := g.NewState(character, shotType, difficulty)
	state.Power = max(0, min(power, game.PowerMax))
	state.Stage, state.SingleStage, state.DebugMode, state.ExtraUnlocked = stageNum, true, true, true
	if stageNum == game.StageExtra {
		state.Difficulty = game.DiffExtra
	}
	g.BeginRun(state)
	return buildStage(g, state, back)
}
