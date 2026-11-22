package title

import (
	"bytes"
	"fmt"
	"testing"

	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/replay"
	"th10/scene/stage"
	sceneui "th10/scene/ui"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func TestTitleEmbeddedDemoEntryAndReadOnlyProgress(t *testing.T) {
	old := input.Global
	defer func() { input.Global = old }()
	input.Global = input.State{}
	g := &game.Game{Config: game.DefaultConfig(), Save: game.DefaultSaveData()}
	g.Save.Native = game.DefaultScoreFile(0)
	before, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil {
		t.Fatal(err)
	}
	title := New()
	g.SetScene(title)
	title.Controller.SetState(sceneui.TitleMain)
	title.Controller.SetSubstate(2)
	title.Controller.Idle = 899
	input.Global.RestoreMenu(0)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if g.RequestedSceneID() != game.ScenePlayback || g.Selection.DemoIndex != 1 {
		t.Fatal("900-frame demo did not request native mode 12")
	}
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	p, ok := g.CurrentScene().(*stage.Playback)
	if !ok || p.Stage().SceneMode() != game.SceneStage || g.RunState.StageMode != 1 || g.RunState.ReplayMode != 1 || g.RunState.RunFlags&game.RunFlagDemo == 0 || g.RunState.Recording != nil {
		t.Fatal("demo was not initialized as mode 1/1 with bit 0x20 before stage creation")
	}
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	for range 12 {
		if err := p.UpdateHeadless(g); err != nil {
			t.Fatal(err)
		}
		p.DrawPorted(sv)
		d.Take()
	}
	after, err := game.EncodeScoreFile(g.Save.Native)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("demo changed the persisted native score")
	}
	input.Global.RestoreMenu(input.MenuEnter)
	if err := p.Update(g); err != nil {
		t.Fatal(err)
	}
	if !p.Finished() || g.RequestedSceneID() != game.SceneTitle {
		t.Fatal("current held native input did not abort demo")
	}
}

func TestTitleDemoResourcesAndPageHooks(t *testing.T) {
	for _, number := range [...]int{1, 2, 3, 0} {
		data, err := gamedata.ReadFile(fmt.Sprintf("misc/demo%d.rpy", number))
		if err != nil {
			t.Fatal(err)
		}
		r, err := replay.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.ValidatePlayback(); err != nil {
			t.Fatal(err)
		}
	}
	g := &game.Game{Config: game.DefaultConfig(), Save: game.DefaultSaveData()}
	title := New()
	called := false
	g.SetScene(title)
	title.Controller.Hooks[sceneui.TitleMusicRoom] = sceneui.TitlePageHooks{Calc: func(c *sceneui.TitleController, _ *game.Game, _ *input.State) error {
		called = true
		c.SetState(sceneui.TitleMain)
		return nil
	}}
	title.Controller.SetState(sceneui.TitleMusicRoom)
	if err := g.CalcScene(); err != nil {
		t.Fatal(err)
	}
	if !called || title.Controller.State != sceneui.TitleMain || g.CurrentSceneID() != game.SceneTitle {
		t.Fatal("t11 hook reconstructed the title instead of using shared controller")
	}
}
