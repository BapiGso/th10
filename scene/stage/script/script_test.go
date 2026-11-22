package stagescript

import (
	"reflect"
	"testing"

	"th10/scene/stage"
)

func TestCommonScriptKeepsSafetyCap(t *testing.T) {
	s := &Script{config: Config{FrameCap: 60000}}
	for frame := 1; frame <= 60000; frame++ {
		s.Update(nil)
		if s.frame != frame || s.Finished() != (frame == 60000) {
			t.Fatalf("frame %d changed the old safety clock", frame)
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
