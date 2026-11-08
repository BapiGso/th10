package stage

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"th10/entity/player"
	"th10/game"
	"th10/input"
	"th10/replay"
	"th10/scene/dialog"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

func TestPlayerFaithRectangleStageGates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		flags    uint32
		boss     bool
		noBoss   bool
		msg      bool
		noGUI    bool
		menu     MenuState
		noMenu   bool
		hold     int
		paused   bool
		dying    bool
		wantBars int
	}{
		{name: "default cfg", flags: game.DefaultNativeConfig().Flags(), hold: 130},
		{name: "enabled", flags: 0x180, hold: 130, wantBars: 2},
		{name: "unrelated flag", flags: 0x200, hold: 130},
		{name: "boss slot", flags: 0x80, boss: true, hold: 130},
		{name: "missing manager", flags: 0x80, noBoss: true, hold: 130},
		{name: "dialog", flags: 0x80, msg: true, hold: 130},
		{name: "missing GUI", flags: 0x80, noGUI: true, hold: 130},
		{name: "end menu", flags: 0x80, menu: 6, hold: 130},
		{name: "missing menu", flags: 0x80, noMenu: true, hold: 130},
		{name: "zero hold", flags: 0x80},
		{name: "host pause is not menu state", flags: 0x80, paused: true, hold: 130, wantBars: 2},
		{name: "dying", flags: 0x80, dying: true, hold: 130},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := game.NewState(game.CharReimu, game.ShotA, game.DiffNormal)
			state.Stage, state.ConfigFlags, state.FaithHold, state.Paused = 1, tc.flags, tc.hold, tc.paused
			s := NewHeadless(state, &recordScript{})
			s.ctx.Boss0 = func() ([3]float32, bool) { return [3]float32{}, tc.boss }
			if tc.noBoss {
				s.ctx.Boss0 = nil
			}
			if tc.msg {
				s.ctx.msg = &dialog.MSG{}
			}
			if tc.noGUI {
				s.ctx.GUI = nil
			}
			if tc.noMenu {
				s.ctx.Menu = nil
			} else {
				s.ctx.Menu.State = tc.menu
			}
			if tc.dying {
				s.ctx.Player.State = player.StateDying
			}
			d := &supervisortest.Rec{}
			sv := supervisor.New(d, nil)
			sv.BeginFrame()
			sv.EnterField()
			d.Take()
			s.nodePlayer(sv)
			sv.ANM.Flush()
			bars := 0
			for _, call := range d.Take() {
				if strings.HasPrefix(call, "draw 5 ") {
					bars++
				}
			}
			if bars != tc.wantBars {
				t.Fatalf("bars=%d want=%d", bars, tc.wantBars)
			}
		})
	}
}

func TestStageReplayPlayerHookRoundtrip(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".tmp", "[th10]", "replay", "th10_05.rpy"))
	if err != nil {
		t.Skipf("original replay unavailable: %v", err)
	}
	r, err := replay.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	for i, ch := range r.Chapters {
		state := game.NewState(int(r.Global.Character), int(r.Global.ShotType), int(r.Global.Difficulty))
		state.Stage, state.Power, state.Life = int(ch.Index), int(ch.Power)*5, int(ch.Lives)
		state.RNGSeed, state.HasSeed = ch.Seed, true
		s := NewHeadless(state, &recordScript{})
		if s.ctx.CaptureReplayPlayer == nil || s.ctx.RestoreReplayPlayer == nil {
			t.Fatal("NewContext did not install the live player hooks")
		}
		s.RestoreReplayChapter(ch)
		got := s.ReplaySnapshot()
		got.FrameCount, got.Records, got.SlowRates, got.Padding = ch.FrameCount, ch.Records, ch.SlowRates, ch.Padding
		h, err := replay.ChapterHeader(got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(h, ch.Raw) {
			for at := range h {
				if h[at] != ch.Raw[at] {
					t.Fatalf("chapter %d difference at %#x: %#x / %#x", i, at, h[at], ch.Raw[at])
				}
			}
		}
		before := slices.Clone(got.Raw)
		x := int32(0)
		if int32(binary.LittleEndian.Uint32(before[0x2c:])) == 0 {
			x = 450
		}
		s.ctx.Player.SetFixed(x, ch.PlayerY)
		var in input.State
		s.updatePlayer(&in)
		after := s.ReplaySnapshot().Raw
		if bytes.Equal(before[0x2c:0x34], after[0x2c:0x34]) {
			t.Fatalf("chapter %d capture still passed through opaque history instead of live movement", i)
		}
	}
}

func stageReplayPair(h []byte, at int) [2]int32 {
	return [2]int32{int32(binary.LittleEndian.Uint32(h[at:])), int32(binary.LittleEndian.Uint32(h[at+4:]))}
}

func TestRecordingNextChapterCapturesLivePlayerFields(t *testing.T) {
	state := game.NewState(game.CharReimu, game.ShotA, game.DiffNormal)
	state.Stage, state.Power = 1, 400
	s := NewHeadless(state, &recordScript{})
	s.beginRecording()
	first := slices.Clone(state.Recording.Replay().Chapters[0].Raw)
	var in input.State
	in.Restore(1 << input.KeyRight)
	for range 16 {
		s.UpdateHeadless(&in)
	}
	// FUN_0042a450 snapshots the live player at the chapter boundary, before
	// any input in the next chapter. The old chapter's stored header is frozen.
	state.Stage, state.Frame = 2, 0
	// select.go 的切关路径会调用它；t8 起 fresh-run 位取自 FUN_00428f60 bit 0（DAT_00491fc4）
	state.BeginStageTransition()
	if err := s.StartRecording(state.Recording); err != nil {
		t.Fatal(err)
	}
	chapters := state.Recording.Replay().Chapters
	if len(chapters) != 2 || !bytes.Equal(chapters[0].Raw, first) {
		t.Fatal("chapter transition changed the preceding capture")
	}
	h := chapters[1].Raw
	for i := range 33 {
		want := [2]int32{max(0, 16-int32(i)) * 450, 40000}
		if got := stageReplayPair(h, 0x2c+i*8); got != want {
			t.Fatalf("history %d=%v want=%v", i, got, want)
		}
	}
	fast := [4][2]int32{{-3200, 800}, {3200, 800}, {-1600, 3200}, {1600, 3200}}
	slow := [4][2]int32{{-1600, 2000}, {1600, 2000}, {-800, 3000}, {800, 3000}}
	for i := range 4 {
		cx, cy := fast[i][0], 40000+fast[i][1]
		for frame := int32(1); frame <= 16; frame++ {
			tx := frame*450 + fast[i][0]
			if frame == 1 {
				cx = tx
			} else {
				cx += (tx - cx) * 30 / 100
			}
		}
		if got := stageReplayPair(h, 0x134+i*8); got != [2]int32{7200 + fast[i][0], cy} {
			t.Fatalf("option %d target=%v", i, got)
		}
		if got := stageReplayPair(h, 0x154+i*8); got != [2]int32{cx, cy} {
			t.Fatalf("option %d current=%v want=%v", i, got, [2]int32{cx, cy})
		}
		if stageReplayPair(h, 0x174+i*8) != fast[i] || stageReplayPair(h, 0x194+i*8) != slow[i] {
			t.Fatalf("option %d saved offsets are not the original fixed-point fields", i)
		}
	}
	if chapters[1].PlayerX != 7200 || chapters[1].PlayerY != 40000 || chapters[1].Flags&1 != 0 {
		t.Fatal("subsequent chapter captured a fresh-run marker or the wrong player position")
	}
	state.Recording.Append(replay.Record{Keys: replay.KeyLeft}, 60)
	encoded, err := replay.Encode(state.Recording.PrepareSave())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := replay.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Chapters) != 2 || !bytes.Equal(decoded.Chapters[1].Raw[0x2c:0x1b4], h[0x2c:0x1b4]) ||
		decoded.Chapters[1].PlayerX != 7200 || decoded.Chapters[1].PlayerY != 40000 {
		t.Fatal("native replay writer did not serialize the live chapter-boundary capture")
	}
}
