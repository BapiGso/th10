package ui

import (
	"reflect"
	"testing"

	"th10/game"
	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func menuGame() *game.Game {
	save := game.DefaultSaveData()
	save.Native = game.DefaultScoreFile(0)
	return &game.Game{Config: game.DefaultConfig(), Save: save}
}

func menuFrame(t *testing.T, c *TitleController, in *input.State, bits uint16) {
	t.Helper()
	in.RestoreMenu(bits)
	if err := c.Calc(c.Game, in); err != nil {
		t.Fatal(err)
	}
}
func menuIdle(t *testing.T, c *TitleController, in *input.State, frames int) {
	t.Helper()
	for range frames {
		menuFrame(t, c, in, 0)
	}
}
func activatePage(t *testing.T, c *TitleController, in *input.State) {
	t.Helper()
	for frame := 0; c.Substate != 2 && frame < 20; frame++ {
		menuFrame(t, c, in, 0)
	}
	if c.Substate != 2 {
		t.Fatalf("page %d did not become active", c.State)
	}
}

func TestTitleIntroAndMainNativeFrameGates(t *testing.T) {
	c := NewTitleController(menuGame(), 0)
	var in input.State
	menuFrame(t, c, &in, 0)
	if c.State != TitleIntro || c.Timer.Cur != 1 {
		t.Fatal("initial title dispatch differs")
	}
	for frame := 1; frame <= 11; frame++ {
		menuFrame(t, c, &in, input.MenuShot)
	}
	if c.Substate != 2 {
		t.Fatal("intro did not open at timer >10")
	}
	menuFrame(t, c, &in, input.MenuShot)
	if c.Substate != 2 {
		t.Fatal("held pre-gate confirm became a new press")
	}
	menuFrame(t, c, &in, 0)
	menuFrame(t, c, &in, input.MenuEnter)
	if c.Substate != 4 || c.Timer.Cur != 1 {
		t.Fatal("intro confirm did not start native 30-frame exit")
	}
	menuIdle(t, c, &in, 29)
	if c.State != TitleIntro || c.Timer.Cur != 30 {
		t.Fatal("intro left before pre-tick frame 30")
	}
	menuFrame(t, c, &in, 0)
	if c.State != TitleMain {
		t.Fatal("intro failed to leave on frame 30")
	}
	activatePage(t, c, &in)
	if c.Root(0) == nil || c.Selector.Count != 8 {
		t.Fatal("main menu root/count differs")
	}
}

func TestTitleMainRepeatLockedExtraAndCancel(t *testing.T) {
	c := NewTitleController(menuGame(), 1)
	var in input.State
	activatePage(t, c, &in)
	for frame := 1; frame <= 34; frame++ {
		menuFrame(t, c, &in, input.MenuDown)
		want := 2
		if frame >= 26 {
			want = 3
		}
		if frame >= 34 {
			want = 4
		}
		if c.Selector.Value != want {
			t.Fatalf("frame %d cursor=%d want %d", frame, c.Selector.Value, want)
		}
	}
	menuFrame(t, c, &in, input.MenuPause)
	if c.Selector.Value != 7 || c.Substate != 2 {
		t.Fatal("first cancel must only select Quit")
	}
	menuIdle(t, c, &in, 1)
	menuFrame(t, c, &in, input.MenuBomb)
	if c.Substate != 4 {
		t.Fatal("second cancel did not start exit")
	}
	menuIdle(t, c, &in, 19)
	if c.State != TitleMain || c.Timer.Cur != 20 {
		t.Fatal("main exit is not 20 pre-tick frames")
	}
	menuFrame(t, c, &in, 0)
	menuFrame(t, c, &in, 0)
	if c.Game.RequestedSceneID() != game.SceneExit {
		t.Fatal("Quit did not request the native exit mode")
	}
}

func TestTitleNormalSelectionOrderTimingAndModeFields(t *testing.T) {
	c := NewTitleController(menuGame(), 1)
	c.SetState(TitleDifficulty)
	c.Selector.Count = 4
	c.Selector.Set(game.DiffNormal)
	var in input.State
	activatePage(t, c, &in)
	menuFrame(t, c, &in, input.MenuEnter)
	menuIdle(t, c, &in, 13)
	if c.State != TitleDifficulty || c.Timer.Cur != 14 {
		t.Fatal("difficulty confirmed too early")
	}
	menuFrame(t, c, &in, 0)
	if c.State != TitleCharacter {
		t.Fatal("difficulty did not lead to character")
	}
	activatePage(t, c, &in)
	menuFrame(t, c, &in, input.MenuRight)
	menuFrame(t, c, &in, input.MenuShot)
	menuIdle(t, c, &in, 14)
	if c.State != TitleShot || c.Character != game.CharMarisa || c.Selector.Value != 0 {
		t.Fatal("character/shot transition differs")
	}
	activatePage(t, c, &in)
	menuFrame(t, c, &in, input.MenuDown)
	menuFrame(t, c, &in, 0)
	menuFrame(t, c, &in, input.MenuDown)
	var started *game.GameState
	c.SetRunHooks(func(_ *game.Game, s *game.GameState) error { started = s; return nil }, nil)
	menuFrame(t, c, &in, input.MenuEnter)
	menuIdle(t, c, &in, 9)
	if len(c.Fades) != 0 || c.Timer.Cur != 10 {
		t.Fatal("selection fade started before timer 10")
	}
	menuFrame(t, c, &in, 0)
	if len(c.Fades) != 1 || c.LoadingMarker == nil || c.Fades[0].Alpha != 0 {
		t.Fatal("frame-10 native fade/marker missing")
	}
	menuIdle(t, c, &in, 29)
	if started != nil || c.Timer.Cur != 40 {
		t.Fatal("stage was requested before frame 40")
	}
	menuFrame(t, c, &in, 0)
	if started == nil || started.Character != 1 || started.ShotType != 2 || started.Difficulty != 1 || started.Stage != 1 || started.StartStage != 1 || started.StageMode != 0 || started.ReplayMode != 0 || !started.FreshRun || started.RunFlags != 0 || started.Rank != 0 || started.Life != 2 || started.Power != 0 || started.Faith != 50000 {
		t.Fatalf("normal launch fields differ: %+v", started)
	}
}

func TestTitlePracticeReadsClearedRecordAndNumericLives(t *testing.T) {
	g := menuGame()
	g.Save.Progress[1].Stages[2][2].Cleared = 1
	g.Save.Progress[1].Stages[2][5].Cleared = 1
	g.Save.Progress[1].Stages[2][3].Available = 1 // availability alone is not native +0x4e9
	c := NewTitleController(g, 1)
	c.Character, c.Shot, c.Difficulty, c.Practice, c.PracticeStage = 0, 1, 2, true, 4
	c.SetState(TitlePractice)
	var in input.State
	activatePage(t, c, &in)
	if !reflect.DeepEqual(c.AvailableStages(), []int{3, 6}) || c.Selector.Value != 2 {
		t.Fatal("practice unlock/filter policy differs")
	}
	menuFrame(t, c, &in, input.MenuDown)
	if c.Selector.Value != 5 {
		t.Fatal("practice cursor used a row index instead of native stage index")
	}
	var started *game.GameState
	c.SetRunHooks(func(_ *game.Game, s *game.GameState) error { started = s; return nil }, nil)
	in.RestoreMenu(input.MenuEnter)
	in.SetMenuAux(0, 4)
	if err := c.Calc(g, &in); err != nil {
		t.Fatal(err)
	}
	menuIdle(t, c, &in, 40)
	if started == nil || started.Stage != 6 || started.StartStage != 6 || started.Life != 3 || started.PracticeLives != 4 || started.Power != 400 || started.Rank != 0 || started.RunFlags != game.RunFlagPractice || started.StageMode != 0 || started.ReplayMode != 0 || !started.FreshRun || !started.SingleStage {
		t.Fatalf("practice launch differs: %+v", started)
	}
}

func TestTitleExtraUsesPerShotSTBytes(t *testing.T) {
	g := menuGame()
	g.Save.Native.Stats[0x1c+4] = 1
	if g.Save.ExtraUnlocked {
		t.Fatal("fixture must not use aggregate unlock")
	}
	c := NewTitleController(g, 1)
	c.BeginExtra()
	c.SetState(TitleCharacter)
	var in input.State
	activatePage(t, c, &in)
	if c.Selector.Value != 1 || !reflect.DeepEqual(c.Selector.Disabled, []int{0}) {
		t.Fatal("Extra character gate ignored ST bytes")
	}
	menuFrame(t, c, &in, input.MenuEnter)
	menuIdle(t, c, &in, 14)
	activatePage(t, c, &in)
	if c.Character != 1 || c.Selector.Value != 1 || !reflect.DeepEqual(c.Selector.Disabled, []int{0, 2}) {
		t.Fatal("Extra shot gate ignored ST bytes")
	}
	var started *game.GameState
	c.SetRunHooks(func(_ *game.Game, s *game.GameState) error { started = s; return nil }, nil)
	menuFrame(t, c, &in, input.MenuEnter)
	menuIdle(t, c, &in, 40)
	if started == nil || started.Stage != 7 || started.StartStage != 7 || started.Difficulty != 4 || started.Character != 1 || started.ShotType != 1 || started.RunFlags != 0 || started.Power != 400 {
		t.Fatalf("Extra launch differs: %+v", started)
	}
}

func TestTitleIdle900AndNativeDemoRotation(t *testing.T) {
	c := NewTitleController(menuGame(), 1)
	var in input.State
	activatePage(t, c, &in)
	c.Idle = 0
	var demos []int
	c.SetRunHooks(nil, func(_ *game.Game, index int) error { demos = append(demos, index); return nil })
	for range 899 {
		menuFrame(t, c, &in, input.MenuRight|input.MenuFocus)
	}
	if len(demos) != 0 {
		t.Fatal("demo started before 900 held-mask-free frames")
	}
	menuFrame(t, c, &in, input.MenuRight|input.MenuFocus)
	if !reflect.DeepEqual(demos, []int{0}) {
		t.Fatal("direction/focus incorrectly reset idle")
	}
	menuFrame(t, c, &in, input.MenuSpecial)
	if c.Idle != 0 {
		t.Fatal("S did not reset native idle mask")
	}
	for range 4 {
		menuIdle(t, c, &in, 900)
	}
	if !reflect.DeepEqual(demos, []int{0, 1, 2, 3, 0}) {
		t.Fatalf("demo rotation indices %v", demos)
	}
}

func TestTitleAllOwnedPagesHeadlessDraw(t *testing.T) {
	for _, page := range []TitlePage{TitleIntro, TitleMain, TitleDifficulty, TitleCharacter, TitleShot, TitlePractice} {
		t.Run(string(rune('A'+page)), func(t *testing.T) {
			g := menuGame()
			for stage := range 6 {
				g.Save.Progress[0].Stages[1][stage].Cleared = 1
			}
			c := NewTitleController(g, 1)
			c.SetState(page)
			c.Selector.Count = 4
			d := &supervisortest.Rec{}
			sv := supervisor.New(d, nil)
			var in input.State
			for range 40 {
				menuFrame(t, c, &in, 0)
				c.DrawPorted(sv)
				d.Take()
			}
		})
	}
}
