package ecl

import (
	"math"
	"testing"
)

// Even ring (mode 3): N ways evenly spaced 2pi/N, speed constant when 1 stack.
func TestPerBulletEvenRing(t *testing.T) {
	vm := &VM{rngState: 1}
	const n = 8
	tp := &eclTemplate{aim: 3, ways: n, stacks: 1, speed: 3, angleInc: 0}
	var prev float64
	for w := 0; w < n; w++ {
		a, sp := vm.perBullet(tp, w, 0, n, 1, 0)
		if sp != 3 {
			t.Fatalf("w=%d speed=%v want 3", w, sp)
		}
		if w > 0 {
			d := math.Mod(a-prev+2*math.Pi, 2*math.Pi)
			if math.Abs(d-2*math.Pi/n) > 1e-6 {
				t.Fatalf("w=%d spacing=%v want %v", w, d, 2*math.Pi/n)
			}
		}
		prev = a
	}
}

// Centered fan (mode 1, no aim): symmetric about the base angle, so angles sum to ways*base.
func TestPerBulletFanSymmetric(t *testing.T) {
	vm := &VM{rngState: 1}
	const n = 5
	base, inc := 1.0, 0.2
	tp := &eclTemplate{aim: 1, ways: n, stacks: 1, speed: 2, angle: base, angleInc: inc}
	sum := 0.0
	for w := 0; w < n; w++ {
		a, _ := vm.perBullet(tp, w, 0, n, 1, 0)
		sum += a
	}
	if math.Abs(sum-float64(n)*base) > 1e-6 {
		t.Fatalf("fan not symmetric: sum=%v want %v", sum, float64(n)*base)
	}
}

// Speed interpolates speed -> speed2 (tp.speedInc) across stacks.
func TestPerBulletSpeedInterp(t *testing.T) {
	vm := &VM{rngState: 1}
	tp := &eclTemplate{aim: 3, ways: 1, stacks: 4, speed: 4, speedInc: 0}
	want := []float64{4, 3, 2, 1}
	for s := 0; s < 4; s++ {
		_, sp := vm.perBullet(tp, 0, s, 1, 4, 0)
		if math.Abs(sp-want[s]) > 1e-6 {
			t.Fatalf("stack %d speed=%v want %v", s, sp, want[s])
		}
	}
}
