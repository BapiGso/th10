package player

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"th10/game"
	"th10/input"
	"th10/replay"
)

func replayPair(header []byte, at int) [2]int32 {
	return [2]int32{int32(binary.LittleEndian.Uint32(header[at:])), int32(binary.LittleEndian.Uint32(header[at+4:]))}
}

func putReplayPair(header []byte, at int, pair [2]int32) {
	binary.LittleEndian.PutUint32(header[at:], uint32(pair[0]))
	binary.LittleEndian.PutUint32(header[at+4:], uint32(pair[1]))
}

func TestReplayPlayerPairLayoutAndClockReset(t *testing.T) {
	p := testPlayer(game.CharReimu)
	p.GS.Power = 400
	p.State, p.focus = StateDeathbomb, true
	p.StateTimer.Set(30)
	p.initTimer.Set(20)
	p.Invinc.Set(51)
	p.shot.timer.Set(12)
	h := bytes.Repeat([]byte{0xa5}, 0x1c4)
	for i := range 33 {
		putReplayPair(h, 0x2c+i*8, [2]int32{int32(i) * -731, int32(i) * 919})
	}
	for col := range 4 {
		for i := range 4 {
			putReplayPair(h, 0x134+col*0x20+i*8, [2]int32{int32(col*1000 + i*37), int32(-col*2000 - i*59)})
		}
	}
	p.RestoreReplayPlayer(h)
	if p.State != StateNormal || p.StateTimer.Cur != 0 || p.StateTimer.Prev != -1 || p.initTimer.Cur != 0 ||
		p.Invinc.Cur != 51 || p.shot.timer.Cur != -1 || !p.focus || p.optionCount != 4 {
		t.Fatal("replay restore reset the wrong player clocks/state")
	}
	for i, o := range p.options {
		if o.snap || o.state != 2 || o.target != replayPair(h, 0x134+i*8) || o.pos != replayPair(h, 0x154+i*8) ||
			o.fast != replayPair(h, 0x174+i*8) || o.slow != replayPair(h, 0x194+i*8) {
			t.Fatalf("option %d layout/snap mismatch: %+v", i, o)
		}
	}
	out := bytes.Repeat([]byte{0xa5}, 0x1c4)
	p.CaptureReplayPlayer(out)
	if !bytes.Equal(out, h) {
		t.Fatal("capture/restore changed a pair or a field owned by the stage")
	}
}

func TestReplayPlayerOriginalChaptersRoundtrip(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".tmp", "[th10]", "replay", "th10_05.rpy"))
	if err != nil {
		t.Skipf("original replay unavailable: %v", err)
	}
	r, err := replay.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Chapters) != 6 {
		t.Fatalf("chapters %d, want 6", len(r.Chapters))
	}
	for i, ch := range r.Chapters {
		gs := game.NewState(int(r.Global.Character), int(r.Global.ShotType), int(r.Global.Difficulty))
		gs.Power = int(ch.Power) * 5
		p := New(gs, game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
		p.SetFixed(ch.PlayerX, ch.PlayerY)
		p.RestoreReplayFocus(ch.Focus)
		p.RestoreReplayPlayer(ch.Raw)
		h := slices.Clone(ch.Raw)
		clear(h[0x2c:0x1b4]) // prove these bytes are recreated from live fields, not passed through
		p.CaptureReplayPlayer(h)
		if !bytes.Equal(h, ch.Raw) {
			for at := range h {
				if h[at] != ch.Raw[at] {
					t.Fatalf("chapter %d first difference at %#x: %#x / %#x", i, at, h[at], ch.Raw[at])
				}
			}
		}
	}
}

func TestReplayOptionsDoNotSnapOnFirstMove(t *testing.T) {
	p, _, _ := optionsPlayer(game.CharReimu, game.ShotA, 100, nil)
	p.SetFixed(1000, 20000)
	h := make([]byte, 0x1c4)
	putReplayPair(h, 0x154, [2]int32{-2000, 15000})
	putReplayPair(h, 0x174, [2]int32{800, 3200})
	putReplayPair(h, 0x194, [2]int32{-500, 2100})
	p.RestoreReplayPlayer(h)
	var in input.State
	p.move(&in)
	want := [2]int32{-2000 + (1800+2000)*30/100, 15000 + (23200-15000)*30/100}
	if p.options[0].pos != want || p.options[0].target != [2]int32{1800, 23200} {
		t.Fatalf("restored option snapped: pos=%v want=%v", p.options[0].pos, want)
	}
}

func TestReplayMarisaCLockHomeNormalisation(t *testing.T) {
	p := New(game.NewState(game.CharMarisa, game.ShotC, game.DiffNormal), game.FieldLeft, game.FieldTop, game.FieldRight, game.FieldBottom)
	p.GS.Power = 400
	h := make([]byte, 0x1c4)
	for i := range 4 {
		putReplayPair(h, 0x134+i*8, [2]int32{int32(i) * 123, 32100})
		putReplayPair(h, 0x194+i*8, [2]int32{999, 111})
	}
	p.RestoreReplayPlayer(h)
	for i, o := range p.options {
		if o.slow != replayPair(h, 0x134+i*8) {
			t.Fatal("Marisa C restore must take its absolute lock home from the target column")
		}
	}
}

func TestReplayPlayerShortHeader(t *testing.T) {
	p := testPlayer(game.CharReimu)
	before := p.positionHistory
	for _, n := range []int{0, 4, 0x1c3} {
		h := bytes.Repeat([]byte{0x5a}, n)
		p.CaptureReplayPlayer(h)
		if !bytes.Equal(h, bytes.Repeat([]byte{0x5a}, n)) {
			t.Fatal("short capture changed its caller's bytes")
		}
		p.RestoreReplayPlayer(h)
		if p.positionHistory != before {
			t.Fatal("short synthetic chapter replaced the history")
		}
	}
}
