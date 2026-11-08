package stage

import (
	"testing"

	"th10/game"
	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func pauseFixture(t *testing.T) (*Stage, *game.Game, *input.State, func(uint16) MenuAction) {
	t.Helper()
	state := game.NewState(0, 0, 1)
	state.Stage, state.Frame, state.DebugMode = 1, 30, true
	s := NewHeadless(state, &recordScript{})
	g := &game.Game{Config: game.DefaultConfig(), Save: game.DefaultSaveData(), Replays: game.NewMemoryReplayStore()}
	in := &input.State{}
	tick := func(raw uint16) MenuAction {
		in.RestoreMenu(raw)
		action := s.ctx.Menu.Calc(g, in)
		s.ctx.Menu.world.Run(s.ctx.Menu.env)
		return action
	}
	return s, g, in, tick
}

func TestPauseNativeGateFreezeAndResumeClock(t *testing.T) {
	s, g, in, tick := pauseFixture(t)
	m := s.ctx.Menu
	s.ctx.State.Frame = 29
	in.RestoreMenu(input.MenuPause)
	if m.TryPause(g, in) {
		t.Fatal("pause opened before native stage timer >29")
	}
	s.ctx.State.Frame = 30
	in.RestoreMenu(0)
	in.RestoreMenu(input.MenuPause)
	s.ctx.Anm.GameSpeed = 0.5
	if !m.TryPause(g, in) || !s.ctx.State.Paused || s.SceneFlags()&0x10 == 0 || s.ctx.Anm.GameSpeed != 1 {
		t.Fatal("pause freeze/speed flags differ")
	}
	for range 10 {
		tick(0)
	}
	if m.State != MenuPauseOpening || m.Timer.Cur != 10 {
		t.Fatal("opening gate is not ten pre-tick frames")
	}
	tick(0)
	if m.State != MenuPauseChoices || m.Timer.Cur != 11 || m.pauseSelector.Count != 3 {
		t.Fatal("choice activation incorrectly reset timer/count")
	}
	tick(input.MenuEnter)
	for range 11 {
		tick(0)
	}
	if m.State != MenuPauseLeaving || m.Timer.Cur != 12 {
		t.Fatal("resume resolved before frame 12")
	}
	tick(0)
	if m.State != MenuDormant || s.ctx.State.Paused || s.SceneFlags()&0x10 != 0 || s.ctx.Anm.GameSpeed != 0.5 || m.Timer.Cur != 12 || m.Timer.F != 12.5 {
		t.Fatal("unpause did not restore speed without resetting the menu clock")
	}
}

func TestPauseNativeConfirmationAndRetry(t *testing.T) {
	s, g, _, tick := pauseFixture(t)
	m := s.ctx.Menu
	m.OpenPause(g)
	for range 11 {
		tick(0)
	}
	tick(input.MenuDown)
	tick(0)
	tick(input.MenuDown)
	if m.Selection != 2 {
		t.Fatal("Retry is not the third choice")
	}
	tick(input.MenuEnter)
	for range 19 {
		tick(0)
	}
	if m.State != MenuPauseConfirm || m.Timer.Cur != 20 {
		t.Fatal("confirmation opened too early")
	}
	tick(input.MenuUp) // still before input-enable frame 30
	if m.Selection != 1 || m.pauseSelector.Count != 2 {
		t.Fatal("confirmation default is No, with input disabled")
	}
	for range 9 {
		tick(0)
	}
	tick(input.MenuUp)
	if m.Selection != 0 {
		t.Fatal("confirmation did not enable input on timer 30")
	}
	tick(input.MenuEnter)
	for range 20 {
		tick(0)
	}
	if m.State != MenuPauseLeaving || m.Selection != 2 {
		t.Fatal("Yes did not restore the paused selector")
	}
	for range 11 {
		tick(0)
	}
	if got := tick(0); got != MenuRetry {
		t.Fatalf("Retry action %d", got)
	}
}

func TestPauseNativeNoAndShortcuts(t *testing.T) {
	s, g, _, tick := pauseFixture(t)
	m := s.ctx.Menu
	m.OpenPause(g)
	for range 11 {
		tick(0)
	}
	tick(input.MenuDown)
	tick(input.MenuEnter)
	for range 30 {
		tick(0)
	}
	tick(input.MenuEnter)
	for range 20 {
		tick(0)
	}
	if m.State != MenuPauseChoices || m.Selection != 1 {
		t.Fatal("No did not restore Title selection")
	}
	tick(input.MenuRetry)
	if m.State != MenuPauseLeaving || m.Selection != 2 {
		t.Fatal("R shortcut did not bypass confirmation")
	}
	for range 11 {
		tick(0)
	}
	if tick(0) != MenuRetry {
		t.Fatal("R did not dispatch initial-stage retry")
	}
	m.OpenPause(g)
	for range 11 {
		tick(0)
	}
	tick(input.MenuReturnTitle)
	for range 11 {
		tick(0)
	}
	if tick(0) != MenuTitle {
		t.Fatal("Q did not dispatch title")
	}
}

func TestPauseReplayTwoChoicesAndDemoSuppression(t *testing.T) {
	s, g, in, tick := pauseFixture(t)
	m := s.ctx.Menu
	s.ctx.State.StageMode, s.ctx.State.ReplayMode = 1, 1
	m.OpenPause(g)
	for range 11 {
		tick(0)
	}
	if m.pauseSelector.Count != 2 {
		t.Fatal("replay should not offer Retry")
	}
	tick(input.MenuDown)
	tick(input.MenuEnter)
	if m.State != MenuPauseLeaving {
		t.Fatal("replay title exit incorrectly asked confirmation")
	}
	for range 11 {
		tick(0)
	}
	if tick(0) != MenuTitle {
		t.Fatal("replay title action lost")
	}
	s.ctx.State.RunFlags = game.RunFlagDemo
	in.RestoreMenu(0)
	in.RestoreMenu(input.MenuPause)
	if m.TryPause(g, in) {
		t.Fatal("title demo allowed pause")
	}
}

func TestPauseHeadlessDrawAndLiveWorldFreeze(t *testing.T) {
	s, g, _, _ := pauseFixture(t)
	old := input.Global
	defer func() { input.Global = old }()
	input.Global = input.State{}
	input.Global.RestoreMenu(input.MenuPause)
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	frame := s.ctx.State.Frame
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	for range 40 {
		input.Global.RestoreMenu(0)
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
		s.runChain(sv, false)
		d.Take()
	}
	if s.ctx.State.Frame != frame || s.ctx.Menu.State != MenuPauseChoices {
		t.Fatal("live pause advanced the game world")
	}
	input.Global.RestoreMenu(input.MenuPause)
	if err := s.Update(g); err != nil {
		t.Fatal(err)
	}
	for range 12 {
		input.Global.RestoreMenu(0)
		if err := s.Update(g); err != nil {
			t.Fatal(err)
		}
	}
	if s.ctx.State.Frame != frame+1 {
		t.Fatal("world must resume in the same calc that clears native freeze")
	}
}
