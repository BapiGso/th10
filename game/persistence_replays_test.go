package game

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
	"th10/replay"
)

func storedReplay() *replay.Replay {
	return &replay.Replay{
		Global: replay.GlobalHeader{User: "TEST", Stage: 1, Difficulty: 1},
		Chapters: []replay.Chapter{{Index: 1, Seed: 1234, Faith: 5000, Lives: 2,
			Records: []replay.Record{{Keys: replay.KeyRight, Aux1: replay.KeyRight}}}},
	}
}

func TestMemoryReplayStoreAndListing(t *testing.T) {
	s := NewMemoryReplayStore()
	g := &Game{Replays: s}
	if _, err := g.LoadReplay(1); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("vacant slot error differs")
	}
	r := storedReplay()
	if err := g.SaveReplay(9, r); err != nil {
		t.Fatal(err)
	}
	slots := g.ListReplays()
	if len(slots) != 25 || slots[8].Replay == nil || slots[8].Replay.Global.User != "TEST" || slots[0].Replay != nil {
		t.Fatal("slot listing differs")
	}
	data, err := s.Read(9)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 0
	if _, err := g.LoadReplay(9); err != nil {
		t.Fatal("read aliased stored data")
	}
	if err := s.Write(2, []byte("broken")); err != nil {
		t.Fatal(err)
	}
	slots = g.ListReplays()
	if slots[1].Error == nil || slots[8].Replay == nil {
		t.Fatal("corrupt slot hid valid slots")
	}
}

func TestReplaySlotBoundsAndStoredBytes(t *testing.T) {
	s := NewMemoryReplayStore()
	g := &Game{Replays: s}
	for _, slot := range []int{-1, 0, 26, 100} {
		if g.SaveReplay(slot, storedReplay()) == nil {
			t.Fatal("accepted invalid slot")
		}
		if _, err := g.LoadReplay(slot); err == nil {
			t.Fatal("read invalid slot")
		}
	}
	r := storedReplay()
	want, err := replay.Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SaveReplay(25, r); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(25)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal("storage changed the original .rpy bytes")
	}
	if name, err := ReplayFilename(25); err != nil || name != "th10_25.rpy" {
		t.Fatal("slot filename differs")
	}
	r.Global.Character = ^uint32(0)
	if g.SaveReplay(1, r) == nil {
		t.Fatal("unsafe gameplay index accepted")
	}
}
