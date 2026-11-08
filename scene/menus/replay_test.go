package menus

import (
	"strings"
	"testing"
	"th10/game"
	"th10/input"
	"th10/replay"
)

func TestReplayEntryLabels(t *testing.T) {
	if !strings.Contains(replayEntryLabel(game.ReplaySlot{Number: 2}), "empty") {
		t.Fatal("vacant slot hidden")
	}
	r := &replay.Replay{Global: replay.GlobalHeader{User: "TEST", Stage: 8, Character: 1, ShotType: 2, Difficulty: 4}}
	label := replayEntryLabel(game.ReplaySlot{Number: 25, Replay: r})
	for _, text := range []string{"No.25", "TEST", "MarisaC", "Extra"} {
		if !strings.Contains(label, text) {
			t.Fatal("metadata missing:", label)
		}
	}
}

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
