package stage

import (
	"bytes"
	"testing"

	"th10/game"
	"th10/input"
	"th10/scene/dialog"

	"github.com/hajimehoshi/ebiten/v2"
)

type clearNextScene struct{}

func (*clearNextScene) Update(*game.Game) error { return nil }
func (*clearNextScene) Draw(*ebiten.Image)      {}

func clearControllerFixture(t *testing.T, stageNum, difficulty int, flags uint32, replay bool) (*Stage, *game.Game) {
	t.Helper()
	prior := input.Global
	input.Global = input.State{}
	t.Cleanup(func() { input.Global = prior })
	g := &game.Game{Config: game.DefaultConfig(), Scores: game.NewMemoryScoreStore()}
	g.LoadProgress()
	state := game.NewState(0, 0, difficulty)
	state.Stage, state.RunFlags, state.Power = stageNum, flags, 400
	if replay {
		state.StageMode, state.ReplayMode = 1, 1
	} else {
		g.BeginRun(state)
	}
	s := New(g, state, &recordScript{})
	m := dialog.NewMSG(1, []dialog.Instr{{Time: 0, Op: 20}, {Time: 10000, Op: 0}}, dialog.Hooks{Clear: s.ctx.clearStage})
	s.ctx.GUI.attachMSG(m, stageNum, 0)
	s.ctx.msg = m
	return s, g
}

func TestGUIFinalClearRequestsSceneAfter120Frames(t *testing.T) {
	s, g := clearControllerFixture(t, 6, game.DiffNormal, 0, false)
	var shake [5]int32
	s.ctx.RequestShake = func(kind, frames, x, y, z int32) { shake = [5]int32{kind, frames, x, y, z} }
	called := 0
	s.NextScene = func(g *game.Game, state *game.GameState) game.Scene {
		called++
		g.MarkClear(state)
		return &clearNextScene{}
	}
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	ctrl := s.ctx.GUIController
	if ctrl.Flags != 0x30 || ctrl.ClearTimer != 0 || ctrl.SceneRequest != 0 || !s.FinishedHeadless() || s.ctx.State.Score != 63000000 || s.ctx.State.Life != 2 || shake != [5]int32{5, 120, 0, 0, 0} {
		t.Fatal("MSG final clear did not set original flags/bonus/timer/hook")
	}
	for range 119 {
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
	}
	if called != 0 || ctrl.ClearTimer != 119 || ctrl.SceneRequest != 0 {
		t.Fatal("stage switched before the original 120-frame GUI wait")
	}
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	if called != 1 || ctrl.ClearTimer != 120 || ctrl.SceneRequest != 14 || g.Save.Progress[0].Clears[game.DiffNormal] != 1 || s.ctx.State.ReplayState9C != 0 {
		t.Fatal("GUI request/clear count/extend inhibition differs")
	}
	// Keep t5's post-ending menu entry usable on the same Stage instance.
	s.OpenCompletion(g)
	if s.ctx.Menu.State != MenuPrompt || !s.ctx.Menu.Cleared {
		t.Fatal("post-ending completion entry stopped working")
	}
}

func TestPracticeExtraAndReplayClearRoutes(t *testing.T) {
	practice, pg := clearControllerFixture(t, 3, game.DiffNormal, game.RunFlagPractice, false)
	if err := practice.Update(pg); err != nil {
		t.Fatal(err)
	}
	if practice.ctx.Menu.State != MenuPrompt || practice.ctx.State.Score != 0 || practice.ctx.GUIController.Flags != 0 || practice.ctx.GUI.notice[0] != nil || pg.Save.ExtraUnlocked || pg.Save.Progress[0].Stages[1][2].Cleared != 1 || pg.Save.Progress[0].Frames != 0 {
		t.Fatal("practice did not open only its original clear menu/flags")
	}
	extra, eg := clearControllerFixture(t, 7, game.DiffExtra, 0, false)
	if err := extra.Update(eg); err != nil {
		t.Fatal(err)
	}
	if extra.ctx.Menu.State != MenuPrompt || extra.ctx.GUIController.Flags != 0x20 || extra.ctx.GUIController.ClearTimer != 0 || extra.ctx.State.Score != 117000000 || eg.Save.Progress[0].Clears[4] != 1 {
		t.Fatal("Extra incorrectly used the normal ending wait/bonus")
	}
	replayed, rg := clearControllerFixture(t, 6, game.DiffNormal, 0, true)
	if err := rg.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	before, err := rg.Scores.Read()
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	replayed.UpdateHeadless(&in)
	if err := rg.SaveProgress(); err != nil {
		t.Fatal(err)
	}
	after, err := rg.Scores.Read()
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ctx.GUIController.Flags != 0x20 || replayed.ctx.GUIController.SceneRequest != 4 || replayed.ctx.Menu.Active() || replayed.ctx.State.Score != 63000000 || !bytes.Equal(before, after) {
		t.Fatal("replay final clear did not exit without persisted changes")
	}
}

func TestClearSceneRequestOverrideAndOrdinaryTransition(t *testing.T) {
	for _, tc := range []struct {
		flags uint32
		want  int32
	}{{0, 14}, {0x1000, 2}} {
		ctrl := &GUIController{Flags: 0x30}
		for range 120 {
			ctrl.TickClear(tc.flags)
		}
		if ctrl.SceneRequest != tc.want {
			t.Fatal("DAT_00491ff4 scene override differs")
		}
	}
	s, g := clearControllerFixture(t, 2, game.DiffNormal, 0, false)
	called := 0
	s.NextScene = func(*game.Game, *game.GameState) game.Scene { called++; return &clearNextScene{} }
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	if called != 1 || s.ctx.GUIController.SceneRequest != 0xb || s.ctx.State.Score != 0 || s.ctx.Menu.Active() || g.Save.Progress[0].Stages[1][1].Available != 1 {
		t.Fatal("ordinary stage did not request its original transition")
	}
	final, fg := clearControllerFixture(t, 6, game.DiffNormal, 0, false)
	if err := final.Update(fg); err != nil {
		t.Fatal(err)
	}
	for range 119 {
		if err := final.Update(fg); err != nil {
			t.Fatal(err)
		}
	}
	fg.SystemFlags = 0x1000 // change the application word after the stage was initialized
	if err := final.Update(fg); err != nil {
		t.Fatal(err)
	}
	if final.ctx.GUIController.SceneRequest != 2 {
		t.Fatal("GUI used a stale stage-start copy of the global override")
	}
}
