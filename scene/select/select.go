package selector

import (
	"github.com/hajimehoshi/ebiten/v2"
	"th10/game"
	"th10/input"
	"th10/scene/ending"
	"th10/scene/stage"
	"th10/scene/stage/extra"
	"th10/scene/stage/stage1"
	"th10/scene/stage/stage2"
	"th10/scene/stage/stage3"
	"th10/scene/stage/stage4"
	"th10/scene/stage/stage5"
	"th10/scene/stage/stage6"
	sceneui "th10/scene/ui"
	"th10/supervisor"
)

// Select preserves the public entry points while the actual selection
// pages are FUN_00430320/004306a0/00430a60/00430ff0 in one TitleController.
type Select struct {
	Controller      *sceneui.TitleController
	back            game.Scene
	extra, practice bool
}

func New(back game.Scene) *Select         { return &Select{back: back} }
func NewExtra(back game.Scene) *Select    { return &Select{back: back, extra: true} }
func NewPractice(back game.Scene) *Select { return &Select{back: back, practice: true} }
func (*Select) SceneMode() game.SceneID   { return game.SceneTitle }
func (s *Select) SceneEnter(g *game.Game) error {
	if s.Controller != nil {
		return nil
	}
	c := sceneui.NewTitleController(g, 1)
	s.Controller = c
	c.Practice = s.practice
	if s.extra {
		c.BeginExtra()
	}
	c.SetState(sceneui.TitleDifficulty)
	c.Selector.Count = 8
	c.Selector.Set(c.Difficulty)
	if s.extra {
		c.Selector.Set(0)
	}
	c.Spawn(0x5b)
	c.SetRunHooks(func(g *game.Game, state *game.GameState) error { return Launch(g, state, s.back) }, nil)
	return nil
}
func (s *Select) SceneLeave(g *game.Game, _ game.SceneID) {
	if s.Controller != nil {
		s.Controller.Remember(g)
		s.Controller.Dispose()
		s.Controller = nil
	}
}
func (s *Select) Update(g *game.Game) error {
	if err := s.SceneEnter(g); err != nil {
		return err
	}
	if err := s.Controller.Calc(g, &input.Global); err != nil {
		return err
	}
	if s.Controller.State == sceneui.TitleMain {
		g.GoTo(s.back)
	}
	return nil
}
func (s *Select) Draw(screen *ebiten.Image) {
	if s.Controller != nil {
		s.Controller.Draw(screen)
	}
}
func (s *Select) DrawPorted(sv *supervisor.Supervisor) {
	if s.Controller != nil {
		s.Controller.DrawPorted(sv)
	}
}

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
