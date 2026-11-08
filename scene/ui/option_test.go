package ui

import (
	"bytes"
	"testing"

	"th10/input"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func TestOptionNativeHintVolumeDefaultAndExit(t *testing.T) {
	t.Setenv("TH10_DATA_DIR", t.TempDir())
	g := menuGame()
	g.Config.Native.Raw[0x23] = 0x91
	g.Config.Native.SetFlags(0x3a0)
	c := NewTitleController(g, 1)
	c.SetState(TitleOption)
	var in input.State
	activatePage(t, c, &in)
	if c.Selector.Count != 6 || c.Child(1, 0x1f).Sprite != 0x2e {
		t.Fatal("native Option/On sprite differs")
	}
	menuFrame(t, c, &in, input.MenuRight)
	if g.Config.Native.Raw[0x22] != 1 || c.Child(1, 0x1f).ScriptID != 0x1f || c.Child(1, 0x1f).Sprite != 0x30 {
		t.Fatal("Hint changes sprite, not script")
	}
	menuFrame(t, c, &in, 0)
	menuFrame(t, c, &in, input.MenuRight)
	if g.Config.Native.Raw[0x22] != 2 || c.Child(1, 0x20).Sprite != 0x2d {
		t.Fatal("Auto sprite/field differs")
	}
	menuFrame(t, c, &in, input.MenuDown)
	for frame := 1; frame <= 26; frame++ {
		menuFrame(t, c, &in, input.MenuLeft)
	}
	if g.Config.BGMVolume != 0.9 || g.Config.Native.Raw[0x20] != 90 {
		t.Fatal("native 5-point/26-frame adjustment differs")
	}
	if c.Child(1, 0x25).Flags&2 != 0 || c.Child(1, 0x26).Flags&2 == 0 {
		t.Fatal("leading volume digit visibility differs")
	}
	c.Selector.Set(4)
	menuFrame(t, c, &in, input.MenuEnter)
	if g.Config.BGMVolume != 1 || g.Config.SEVolume != 0.8 || g.Config.Native.Raw[0x22] != 0 || g.Config.Native.Raw[0x23] != 0x91 || g.Config.Native.Flags() != 0x3a0 {
		t.Fatal("Default reset opaque cfg fields")
	}
	menuFrame(t, c, &in, input.MenuPause)
	if c.Selector.Value != 5 || c.Substate != 2 {
		t.Fatal("first Option cancel must select Quit")
	}
	menuFrame(t, c, &in, 0)
	menuFrame(t, c, &in, input.MenuBomb)
	menuIdle(t, c, &in, 9)
	if c.State != TitleOption || c.Timer.Cur != 10 {
		t.Fatal("Option exit before timer 10")
	}
	menuFrame(t, c, &in, 0)
	if c.State != TitleMain {
		t.Fatal("Option did not return to main")
	}
}

func TestOptionKeyBindingSwapCommitAndCancel(t *testing.T) {
	g := menuGame()
	g.Config.Native.Raw[0xc], g.Config.Native.Raw[0xd] = 0x34, 0x12
	before := append([]byte(nil), g.Config.Native.Raw[:]...)
	c := NewTitleController(g, 1)
	c.SetState(TitleKeys)
	var in input.State
	activatePage(t, c, &in)
	if !c.BindButton(0, 1) || c.keyBindings != [5]int16{1, 0, 2, 3, 4} {
		t.Fatal("native duplicate binding must swap")
	}
	if !bytes.Equal(before, g.Config.Native.Raw[:]) {
		t.Fatal("staged binding changed live cfg")
	}
	c.Selector.Set(6)
	menuFrame(t, c, &in, input.MenuBomb)
	if c.keyBindings != [5]int16{0, 1, 2, 3, 4} || !bytes.Equal(before, g.Config.Native.Raw[:]) {
		t.Fatal("cancel did not discard staged changes")
	}
	menuIdle(t, c, &in, 10)
	c.SetState(TitleKeys)
	activatePage(t, c, &in)
	in.RestoreMenu(0)
	in.SetMenuAux(1<<8, 0)
	if err := c.Calc(g, &in); err != nil {
		t.Fatal(err)
	}
	if c.keyBindings[0] != 8 {
		t.Fatal("first pressed joystick button was not bound")
	}
	c.Selector.Set(6)
	menuFrame(t, c, &in, input.MenuEnter)
	keys := g.Config.Native.JoystickBindings()
	if keys[0] != 8 || keys[4] != 0x1234 {
		t.Fatal("key commit touched reserved bindings")
	}
}

func TestOptionAndKeysHeadlessDraw(t *testing.T) {
	for _, page := range []TitlePage{TitleOption, TitleKeys} {
		c := NewTitleController(menuGame(), 1)
		c.SetState(page)
		d := &supervisortest.Rec{}
		sv := supervisor.New(d, nil)
		var in input.State
		for range 40 {
			menuFrame(t, c, &in, 0)
			c.DrawPorted(sv)
			d.Take()
		}
	}
}
