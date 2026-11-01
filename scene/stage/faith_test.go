package stage

import (
	"testing"

	"th10/entity/item"
)

// bossScript stands in for the stage script: the faith gate asks it only
// whether boss slot 0 is busy. The embedded nil Script is never called.
type bossScript struct {
	Script
	busy bool
}

func (b bossScript) BossSlot0Busy() bool { return b.busy }

// TestFaithGate follows FUN_00418190's test: the stage clock past 89 (the
// zero-based frame, so Frame-1), and no boss in slot 0.
func TestFaithGate(t *testing.T) {
	s := powerTestStage()
	s.ctx.State.Frame = 90 // zero-based frame 89
	if s.faithGate() {
		t.Fatal("gate open at zero-based frame 89")
	}
	s.ctx.State.Frame = 91 // zero-based frame 90
	if !s.faithGate() {
		t.Fatal("gate closed at zero-based frame 90")
	}
	s.script = bossScript{busy: true}
	if s.faithGate() {
		t.Fatal("gate open with a boss in slot 0")
	}
	s.script = bossScript{}
	if !s.faithGate() {
		t.Fatal("gate closed with no boss")
	}
}

// TestCollectedItemsHoldFaith: each kind adds the frames FUN_0041afd0 gives
// it through FUN_00412ff0 (60 power, 20 big power, 100 point and full-point,
// 120 big faith, 60 faith; the 1-up gives none). The cancel item (kind 8)
// waits out its spawn delay before it can be collected, so it is not here.
func TestCollectedItemsHoldFaith(t *testing.T) {
	for _, tc := range []struct {
		kind int32
		hold int
	}{
		{item.KindPower, 60}, {item.KindBigPower, 20}, {item.KindPoint, 100},
		{item.KindBigFaith, 120}, {item.KindFullPoint, 100},
		{item.KindFaith, 60}, {item.KindLife, 0},
	} {
		s := powerTestStage()
		dropOnPlayer(s, tc.kind)
		s.ctx.UpdateItems(false)
		if s.ctx.Items.ActiveCount() != 0 {
			t.Fatalf("kind %d: not collected", tc.kind)
		}
		if s.ctx.State.FaithHold != tc.hold {
			t.Fatalf("kind %d: hold %d, want %d", tc.kind, s.ctx.State.FaithHold, tc.hold)
		}
	}
}
