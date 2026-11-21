package menus

import (
	"testing"
	"th10/game"
	"th10/input"
	"th10/replay"
)

func TestReplayMenuLoadsStorage(t *testing.T) {
	g := &game.Game{Replays: game.NewMemoryReplayStore()}
	r := &replay.Replay{Global: replay.GlobalHeader{User: "TEST", Stage: 1},
		Chapters: []replay.Chapter{{Index: 1, Records: []replay.Record{{}}, PlayerY: 40000}}}
	if err := g.SaveReplay(1, r); err != nil {
		t.Fatal(err)
	}
	previous := input.Global
	t.Cleanup(func() { input.Global = previous })
	input.Global = input.State{}
	m := NewReplay(nil)
	if err := m.Update(g); err != nil {
		t.Fatal(err)
	}
	if len(m.entries) != 25 || m.entries[0].slot.Replay == nil || m.entries[0].slot.Replay.Global.User != "TEST" {
		t.Fatal("stored replay did not populate the existing menu")
	}
}
