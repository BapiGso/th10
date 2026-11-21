package selector

import (
	"th10/game"
	"th10/scene/ending"
	"th10/scene/stage"
	"th10/scene/stage/extra"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
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

func stageScript(stageNum int) stage.Script {
	switch stageNum {
	case 1:
		return stage1.NewScript()
	case 2:
		return stage2.NewScript()
	case 3:
		return stage3.NewScript()
	case 4:
		return stage4.NewScript()
	case 5:
		return stage5.NewScript()
	case 6:
		return stage6.NewScript()
	case game.StageExtra:
		return extra.NewScript()
	default:
		return stage1.NewScript()
	}
}
func ReplayScript(stageNum int) stage.Script { return stageScript(stageNum) }

func buildStage(g *game.Game, state *game.GameState, back game.Scene) *stage.Stage {
	s := stage.New(g, state, stageScript(state.Stage))
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
			e := ending.New(st, st.Stage == game.StageExtra, back)
			if st.Stage == game.StageCount {
				e.After = func(g *game.Game) game.Scene {
					s.NextScene = func(*game.Game, *game.GameState) game.Scene { return back }
					s.OpenCompletion(g)
					return s
				}
			}
			return game.NumberScene(game.SceneEnding, e) // preserve e.After while numbering request 14
		}
		st.Stage, st.Frame = next, 0
		st.BeginStageTransition()
		return buildStage(g, st, back)
	}
	return s
}

func NewDebugStage(g *game.Game, back game.Scene, stageNum, character, shotType, difficulty int) game.Scene {
	return NewDebugStageWithPower(g, back, stageNum, character, shotType, difficulty, 0)
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
