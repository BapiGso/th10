package zun

import (
	"math"
	"testing"
)

// The float32 constants must carry the exact .rdata bit patterns.
func TestConstantBits(t *testing.T) {
	for _, c := range []struct {
		name string
		v    float32
		bits uint32
	}{
		{"pi (DAT_00470b18)", Pi, 0x40490fdb},
		{"2pi (DAT_00470b14)", TwoPi, 0x40c90fdb},
		{"pi/2 (DAT_00470b94)", HalfPi, 0x3fc90fdb},
	} {
		if got := math.Float32bits(c.v); got != c.bits {
			t.Errorf("%s = %#08x, want %#08x", c.name, got, c.bits)
		}
	}
}

func TestAddAngle(t *testing.T) {
	if got, want := AddAngle(3, 1), float32(4)-TwoPi; got != want {
		t.Errorf("AddAngle(3, 1) = %v, want %v", got, want)
	}
	if got := AddAngle(Pi, 0); got != Pi {
		t.Errorf("AddAngle(pi, 0) = %v, want pi unchanged", got)
	}
}

func TestAngleDiff(t *testing.T) {
	for _, c := range []struct{ a, b, want float32 }{
		{1, 0.5, 0.5},
		{3, -3, 3 - (-3 + TwoPi)},
		{-3, 3, -3 - (3 - TwoPi)},
	} {
		if got := AngleDiff(c.a, c.b); got != c.want {
			t.Errorf("AngleDiff(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestAimCoincident(t *testing.T) {
	if got := Aim(5, 5, 5, 5); got != HalfPi {
		t.Errorf("Aim at the target = %v, want pi/2", got)
	}
	if got := Aim(0, 0, 0, 10); got != HalfPi {
		t.Errorf("Aim straight down = %v, want pi/2", got)
	}
}

// Timers count whole frames at normal speed and truncate the accumulator
// otherwise.
func TestTimer(t *testing.T) {
	var tm Timer
	tm.Reset()
	tm.Tick(1)
	if tm.Prev != 0 || tm.Cur != 1 || tm.F != 1 {
		t.Errorf("after one tick: %+v, want {0 1 1}", tm)
	}
	tm.Set(10)
	if tm.Prev != 9 || tm.Cur != 10 || tm.F != 10 {
		t.Errorf("Set(10): %+v", tm)
	}
	tm.Add(-1, 1)
	if tm.Prev != 10 || tm.Cur != 9 {
		t.Errorf("Add(-1): %+v, want prev 10 cur 9", tm)
	}
	tm.Reset()
	for range 3 {
		tm.Tick(0.5)
	}
	if tm.Cur != 1 || tm.F != 1.5 {
		t.Errorf("three half-speed ticks: %+v, want cur 1 f 1.5", tm)
	}
}

// The first draws from state 1, cross-checked against the ECL VM's RNG.
func TestRNG(t *testing.T) {
	r := RNG{State: 1}
	w := uint16(1)
	var want [2]uint16
	for i := range want {
		w = (w ^ 0x9630) - 0x6553
		w = w<<2 | w>>14
		want[i] = w
	}
	if got := r.Raw(); got != uint32(want[0])<<16|uint32(want[1]) {
		t.Errorf("Raw() = %#08x, want %#04x%04x", got, want[0], want[1])
	}
	if r.Count != 2 || r.State != want[1] {
		t.Errorf("after one draw: %+v", r)
	}
	if got := (&RNG{}).Intn(0); got != 0 {
		t.Errorf("Intn(0) = %d, want 0", got)
	}
	z := RNG{State: 7}
	z.Intn(0)
	if z.State != 7 || z.Count != 0 {
		t.Errorf("Intn(0) drew: %+v", z)
	}
}

func TestInterpLinear(t *testing.T) {
	var p Interp
	p.Begin(4, 0)
	p.Start, p.End = [3]float32{0, 0, 0}, [3]float32{8, -4, 0}
	for i, want := range []float32{2, 4, 6, 8} {
		if got := p.Step(1, 3); got[0] != want || got[1] != -want/2 {
			t.Fatalf("frame %d: %v, want (%v, %v)", i+1, got, want, -want/2)
		}
	}
	if p.Dur != 0 {
		t.Errorf("dur = %d after the last frame, want 0 (finished)", p.Dur)
	}
}

// Mode 7 adds end to start each frame; the finishing frame returns start
// without the last addition.
func TestInterpMode7(t *testing.T) {
	var p Interp
	p.Begin(3, 7)
	p.Start, p.End = [3]float32{}, [3]float32{1, 2, 0}
	for i, want := range []float32{1, 2, 2} {
		if got := p.Step(1, 2); got[0] != want || got[1] != 2*want {
			t.Fatalf("frame %d: %v, want (%v, %v)", i+1, got, want, 2*want)
		}
	}
}

// With zero tangents (281/283) mode 8 is smoothstep.
func TestInterpMode8Smoothstep(t *testing.T) {
	var p Interp
	p.Begin(4, 8)
	p.Start, p.End = [3]float32{0}, [3]float32{1}
	got := []float32{p.Step(1, 3)[0], p.Step(1, 3)[0]}
	if got[0] != 0.15625 || got[1] != 0.5 {
		t.Errorf("smoothstep at 1/4, 1/2 = %v, want [0.15625 0.5]", got)
	}
}
