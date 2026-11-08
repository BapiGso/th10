package stage

import (
	"testing"

	"th10/game"
	"th10/gamedata"
	"th10/input"
	"th10/replay"
)

func TestPlaybackNativePauseDoesNotConsumeRecordedInput(t *testing.T) {
	data, err := gamedata.ReadFile("misc/demo1.rpy")
	if err != nil {
		t.Fatal(err)
	}
	r, err := replay.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g := &game.Game{Config: game.DefaultConfig(), Save: game.DefaultSaveData()}
	p, err := NewPlayback(g, r, 0, func(int) Script { return &recordScript{} }, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 40 {
		if err := p.UpdateHeadless(g); err != nil {
			t.Fatal(err)
		}
	}
	frame, record := p.stage.ctx.State.Frame, p.in.ReplayRecord()
	old := input.Global
	defer func() { input.Global = old }()
	input.Global = input.State{}
	input.Global.RestoreMenu(input.MenuPause)
	if err := p.Update(g); err != nil {
		t.Fatal(err)
	}
	for range 30 {
		input.Global.RestoreMenu(0)
		if err := p.Update(g); err != nil {
			t.Fatal(err)
		}
	}
	if p.stage.ctx.State.Frame != frame || p.in.ReplayRecord() != record || p.stage.ctx.Menu.pauseSelector.Count != 2 {
		t.Fatal("pause consumed replay records or offered retry")
	}
	input.Global.RestoreMenu(input.MenuPause)
	if err := p.Update(g); err != nil {
		t.Fatal(err)
	}
	for range 12 {
		input.Global.RestoreMenu(0)
		if err := p.Update(g); err != nil {
			t.Fatal(err)
		}
	}
	if p.stage.ctx.State.Frame != frame+1 {
		t.Fatal("replay did not resume on the native unpause calc")
	}
}
