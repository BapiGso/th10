package ui

import (
	"reflect"
	"strings"
	"testing"

	"th10/game"
	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func TestNativeSelectorWrapDisabledAndStack(t *testing.T) {
	s := Selector{Count: 8, Wrap: true, Disabled: []int{1}}
	if s.Move(1) != 2 || s.Move(-1) != 0 || s.Move(-1) != 7 {
		t.Fatal("FUN_0044bea0 skipped/wrapped wrong")
	}
	s.Push()
	s.Count, s.Value = 4, 2
	s.Push()
	s.Count, s.Value = 3, 1
	s.Pop()
	if s.Count != 4 || s.Value != 2 || s.Disabled != nil {
		t.Fatal("nested selector restore differs")
	}
	s.Pop()
	if s.Count != 8 || s.Value != 7 {
		t.Fatal("parent selector lost")
	}
	s.Wrap = false
	if s.Move(1) != 7 {
		t.Fatal("non-wrapping selector overshot")
	}
	s.Count = 0
	s.Set(10)
	if s.Value != 10 {
		t.Fatal("zero-count FUN_0040ad20 rejected value")
	}
	for range 20 {
		s.Push()
	}
	if s.depth != 15 {
		t.Fatal("native selector stack depth must clamp to 15")
	}
}

func TestNativeFadePreTickAndLifetime(t *testing.T) {
	f := NewFade(5, 32, 0)
	for frame := int32(0); frame <= 34; frame++ {
		f.Calc(1, false, 0)
		want := min(frame*255/32, 255)
		if f.Alpha != want {
			t.Fatalf("out frame %d alpha=%d want %d", frame, f.Alpha, want)
		}
		if f.Active != (frame < 34) {
			t.Fatalf("out frame %d active=%v", frame, f.Active)
		}
	}
	f = NewFade(0, 32, 0)
	for frame := int32(0); frame <= 32; frame++ {
		f.Calc(1, false, 0)
		want := int32(float32(255) - float32(frame)*255/32)
		if f.Alpha != want || f.Active != (frame < 32) {
			t.Fatalf("in frame %d alpha=%d active=%v", frame, f.Alpha, f.Active)
		}
	}
	f = NewFade(5, 32, 0)
	f.Calc(1, false, 4)
	if f.Timer.Cur != 0 {
		t.Fatal("stage freeze did not hold fade-out timer")
	}
	f.Calc(0.5, false, 0)
	f.Calc(0.5, false, 0)
	if f.Timer.F != 1 || f.Timer.Cur != 1 || f.Alpha != 3 {
		t.Fatal("fractional fade accumulator differs")
	}
}

func TestNativeTitleHandlesAndHeadlessDraw(t *testing.T) {
	g := &game.Game{}
	c := NewTitleController(g, 0)
	c.Spawn(0)
	if c.Child(0, 3) == nil {
		t.Fatal("title root's script-3 child is missing")
	}
	c.Interrupt(0, 17)
	var in input.State
	if err := c.Calc(g, &in); err != nil {
		t.Fatal(err)
	}
	d := &supervisortest.Rec{}
	sv := supervisor.New(d, nil)
	for range 4 {
		c.DrawPorted(sv)
	}
	if !strings.Contains(strings.Join(d.Calls, "\n"), "draw ") {
		t.Fatal("headless title made no ANM draw calls")
	}
	c.ShowLoading()
	marker := c.LoadingMarker
	c.ShowLoading()
	if c.LoadingMarker != marker {
		t.Fatal("loading marker was duplicated")
	}
	c.LeaveScript(0)
	if c.Root(0) != nil {
		t.Fatal("leave did not clear native root handle")
	}
	c.Dispose()
	if c.World.Len() != 0 {
		t.Fatal("title destructor leaked world objects")
	}
}

func TestNativeTitleSharedPageHooks(t *testing.T) {
	g := &game.Game{}
	c := NewTitleController(g, 0)
	var states []TitlePage
	c.Hooks[TitleReplay] = TitlePageHooks{Calc: func(c *TitleController, _ *game.Game, _ *input.State) error {
		states = append(states, c.State)
		c.SetState(TitlePlayerData)
		return nil
	}}
	c.Hooks[TitlePlayerData] = TitlePageHooks{Calc: func(c *TitleController, _ *game.Game, _ *input.State) error {
		states = append(states, c.State)
		return nil
	}}
	c.SetState(TitleReplay)
	var in input.State
	if err := c.Calc(g, &in); err != nil {
		t.Fatal(err)
	}
	if c.Timer.Cur != 1 || c.Substate != 0 {
		t.Fatal("page change did not reset before shared tail tick")
	}
	if err := c.Calc(g, &in); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(states, []TitlePage{TitleReplay, TitlePlayerData}) {
		t.Fatal("t11 page hooks did not share dispatch")
	}
}
