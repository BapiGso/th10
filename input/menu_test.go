package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"th10/replay"
)

func TestNativeMenuRepeat26And8(t *testing.T) {
	var in State
	for frame := 1; frame <= 100; frame++ {
		in.RestoreMenu(MenuDown | MenuEnter)
		m := in.Menu()
		wantRepeat := frame >= 26 && (frame-26)%8 == 0
		if got := m.Repeat&MenuDown != 0; got != wantRepeat {
			t.Fatalf("frame %d repeat=%v want %v", frame, got, wantRepeat)
		}
		if got := m.Move(MenuDown); got != (frame == 1 || wantRepeat) {
			t.Fatalf("frame %d move=%v", frame, got)
		}
		if got := m.Pressed&MenuConfirm != 0; got != (frame == 1) {
			t.Fatalf("held Enter confirms again on frame %d", frame)
		}
		if in.Menu() != m {
			t.Fatal("query advanced menu input")
		}
	}
	in.RestoreMenu(0)
	if in.Menu().Released != MenuDown|MenuEnter {
		t.Fatal("release word differs")
	}
	in.RestoreMenu(MenuDown)
	if in.Menu().Repeat != 0 || in.Menu().Pressed != MenuDown {
		t.Fatal("released counter did not reset")
	}
}

func TestNativeMenuNumpadDirectionalMapping(t *testing.T) {
	for _, test := range []struct {
		key       ebiten.Key
		direction Key
	}{
		{ebiten.KeyNumpad7, KeyUp}, {ebiten.KeyNumpad7, KeyLeft},
		{ebiten.KeyNumpad3, KeyDown}, {ebiten.KeyNumpad3, KeyRight},
		{ebiten.KeyNumpad9, KeyUp}, {ebiten.KeyNumpad9, KeyRight},
	} {
		found := false
		for _, key := range defaultKeyMap[test.direction] {
			found = found || key == test.key
		}
		if !found {
			t.Fatalf("native numpad key %v not mapped to %v", test.key, test.direction)
		}
	}
}

func TestNativeMenuEachBitAndOppositeDirections(t *testing.T) {
	var m MenuInput
	for bit := 0; bit < 16; bit++ {
		m = MenuInput{}
		mask := uint16(1) << bit
		for frame := 1; frame <= 34; frame++ {
			m.Update(mask)
			want := uint16(0)
			if frame == 26 || frame == 34 {
				want = mask
			}
			if m.Repeat != want {
				t.Fatalf("bit %d frame %d repeat=%04x", bit, frame, m.Repeat)
			}
		}
	}
	var in State
	in.RestoreMenu(0xffff)
	if in.Menu().Current != 0xffff || in.Menu().Pressed != 0xffff {
		t.Fatal("raw menu bits were resolved or masked")
	}
	if !in.Menu().Move(MenuLeft) || !in.Menu().Move(MenuRight) {
		t.Fatal("opposite keys must both reach menu code")
	}
}

func TestNativeMenuSyntheticSnapshotsAndReplayIsolation(t *testing.T) {
	var in State
	in.Restore(1 << KeyOk)
	if in.Menu().Pressed != MenuShot {
		t.Fatal("legacy synthetic OK lost")
	}
	in.RestoreMenu(MenuEnter)
	if in.IsPressed(KeyShot) || in.Menu().Current != MenuEnter {
		t.Fatal("Enter became a gameplay shot")
	}
	r := replay.Record{Keys: 0x00c1, Aux1: 0x0041, Aux2: 0x0080}
	in.RestoreReplay(r)
	if in.ReplayRecord() != r {
		t.Fatal("menu sampler changed raw replay words")
	}
	in.RestoreMenu(MenuRetry | MenuReturnTitle | MenuPause)
	if in.Menu().Current != 0x4208 {
		t.Fatal("R/Q/Esc mapping differs from FUN_0044a5f0")
	}
}
