package zun

// Timer is the binary's frame timer {prev, cur, f, gs*, flags}: the previous
// and current whole-frame counts and the float accumulator. The game-speed
// pointer is passed to Tick/Add instead of stored.
type Timer struct {
	Prev, Cur int32
	F         float32
}

// normalSpeed reports a game speed within 1% of 1, where timers count whole
// frames instead of accumulating.
func normalSpeed(gs float32) bool { return gs > 0.99 && gs < 1.01 }

// Reset restarts the timer at 0 (prev = -1).
func (t *Timer) Reset() { t.Prev, t.Cur, t.F = -1, 0, 0 }

// Set is FUN_00405410.
func (t *Timer) Set(n int32) { t.Prev, t.Cur, t.F = n-1, n, float32(n) }

// Tick advances by the game speed gs.
func (t *Timer) Tick(gs float32) {
	t.Prev = t.Cur
	if normalSpeed(gs) {
		t.Cur++
		t.F++
		return
	}
	t.F += gs
	t.Cur = int32(t.F)
}

// Add is FUN_0044bf40: add d frames at the game speed gs.
func (t *Timer) Add(d, gs float32) {
	t.Prev = t.Cur
	if normalSpeed(gs) {
		t.F += d
	} else {
		t.F += float32(d * gs)
	}
	t.Cur = int32(t.F)
}
