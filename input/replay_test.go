package input

import (
	"testing"
	"th10/replay"
)

func TestLiveReplayWords(t *testing.T) {
	var in State
	in.Restore(1<<KeyLeft | 1<<KeyRight | 1<<KeyPause | 1<<KeyOk)
	r := in.ReplayRecord()
	if r != (replay.Record{Keys: replay.KeyLeft | replay.KeyRight, Aux1: replay.KeyLeft | replay.KeyRight}) {
		t.Fatalf("first words: %+v", r)
	}
	in.Restore(1<<KeyRight | 1<<KeyShot | 1<<KeyFocus | 1<<KeySkip)
	r = in.ReplayRecord()
	keys := replay.KeyRight | replay.KeyShot | replay.KeyFocus | replay.KeySkip
	if r != (replay.Record{Keys: keys, Aux1: keys &^ replay.KeyRight, Aux2: replay.KeyLeft}) {
		t.Fatalf("second words: %+v", r)
	}
}

func TestRestoreReplayPreservesRawWords(t *testing.T) {
	var in State
	r := replay.Record{Keys: replay.KeyLeft | replay.KeyRight | replay.KeyShot, Aux1: 0x4321, Aux2: 0x1234}
	in.RestoreReplay(r)
	if in.ReplayRecord() != r {
		t.Fatal("raw words changed")
	}
	if !in.IsPressed(KeyLeft) || in.IsPressed(KeyRight) || !in.IsPressed(KeyOk) {
		t.Fatal("direction/shot mapping differs from sim")
	}
	in.Restore(0)
	if in.ReplayRecord().Keys != 0 {
		t.Fatal("physical restore retained playback words")
	}
}

func TestReplayAutoFocusOption(t *testing.T) {
	var in State
	for i := 1; i <= 8; i++ {
		in.Restore(1 << KeyShot)
		in.ApplyReplayOptions(0x200)
		if in.IsPressed(KeyFocus) != (i >= 8) {
			t.Fatalf("frame %d focus differs", i)
		}
	}
	in.RestoreReplay(replay.Record{Keys: replay.KeyShot})
	in.ApplyReplayOptions(0x200)
	if in.IsPressed(KeyFocus) {
		t.Fatal("host options altered recorded input")
	}
}
