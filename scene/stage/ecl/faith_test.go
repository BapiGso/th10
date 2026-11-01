package ecl

import "testing"

// TestEnemyFrameSetsDecayRateOne: an enemy that runs its frame and is not
// removed sets the faith decay rate to 1 (FUN_0040dc80 0x40e26b). Nothing
// else in stepFrame touches the rate, so it stays 18 until an enemy runs. The
// spawn frame's own update leaves 0x400 set, so the next list pass returns at
// once; the pass after that is the first full frame.
func TestEnemyFrameSetsDecayRateOne(t *testing.T) {
	ctx := newTestContext()
	vm := NewStage01VM()
	for i := 0; i < 400 && activeEnemies(ctx) == 0; i++ {
		stepFrame(vm, ctx)
	}
	if activeEnemies(ctx) == 0 {
		t.Fatal("no enemy appeared in the first 400 frames")
	}
	ctx.State.FaithRate = 18
	stepFrame(vm, ctx) // 0x400 still set from the spawn: early return
	stepFrame(vm, ctx) // full frame
	if ctx.State.FaithRate != 1 {
		t.Fatalf("faith decay rate %d after an enemy frame, want 1", ctx.State.FaithRate)
	}
}
