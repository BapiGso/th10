package input

import "testing"

func TestUnifiedDirectionRepeatMatchesLegacyQueries(t *testing.T) {
	for _, key := range []Key{KeyUp, KeyDown, KeyLeft, KeyRight} {
		var s State
		var mask uint16
		for _, pair := range menuKeyPairs {
			if pair.key == key {
				mask = pair.bit
			}
		}
		for frame := range 300 {
			bits := uint16(1) << key
			if frame == 99 || frame == 171 {
				bits = 0
			}
			s.Restore(bits)
			n := s.HoldFrames(key)
			want := s.JustPressed(key) || n >= 26 && (n-26)%8 == 0
			if got := s.Menu().Move(mask); got != want {
				t.Fatalf("key %v frame %d native=%v legacy=%v", key, frame, got, want)
			}
		}
	}
}

func TestUnifiedDirectionRepeatPreservesOppositeKeySnapshots(t *testing.T) {
	var s State
	for frame := 1; frame <= 50; frame++ {
		s.RestoreMenu(MenuUp | MenuDown | MenuLeft | MenuRight)
		before := s.Menu()
		for _, mask := range []uint16{MenuUp, MenuDown, MenuLeft, MenuRight} {
			if got := s.Menu().Move(mask); got != (frame == 1 || frame >= 26 && (frame-26)%8 == 0) {
				t.Fatalf("mask %x frame %d repeat differs", mask, frame)
			}
		}
		if after := s.Menu(); before != after {
			t.Fatal("direction queries advanced or consumed the shared snapshot")
		}
	}
}
