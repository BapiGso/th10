package stagescript

import (
	"math"
	"reflect"
	"testing"

	"th10/scene/stage"
)

func TestCommonScriptKeepsSafetyCapAndRealFrameScroll(t *testing.T) {
	for i, scroll := range []float64{0, 2, 1.8, 2.5, 1.5, 1.2, 2} {
		s := &Script{config: Config{FrameCap: 60000}, bg: legacyBackground{kind: i + 1, scroll: scroll}}
		want := float64(0)
		for frame := 1; frame <= 60000; frame++ {
			s.Update(nil)
			want += scroll
			if s.frame != frame || s.bg.frame != frame || s.Finished() != (frame == 60000) {
				t.Fatalf("stage %d frame %d changed the old safety clock", i+1, frame)
			}
			if i == 0 {
				cam := stage01StdCamera(frame)
				if s.bg.bgX != cam.x || s.bg.bgY != cam.y {
					t.Fatal("stage1 no longer samples the legacy camera on every real frame")
				}
			} else if math.Float64bits(s.bg.bgY) != math.Float64bits(want) {
				t.Fatalf("stage %d changed float64 scrolling order", i+1)
			}
		}
	}
}

func TestCommonScriptNilVMOptionalInterfaces(t *testing.T) {
	s := &Script{}
	if x, y := s.ShakeOffset(); x != 0 || y != 0 {
		t.Fatal("nil VM has a shake offset")
	}
	if !reflect.DeepEqual(s.BossHUD(), stage.BossHUDInfo{}) {
		t.Fatal("nil VM has a Boss HUD")
	}
	s.LateUpdate(nil)
}
