package anm

import (
	"fmt"
	"math"
	"testing"

	"th10/zun"
)

// Misreadings of fillRing's contract that ringWant can make instead, so
// each fixture can show it tells them apart from the contract.
const (
	ringExact    = iota
	ringGroup    // Offset + (Pos + x) instead of (Offset + Pos) + x
	ringStored   // cos and sin rounded to float32 before the products
	ringMulAcc   // V from i·d8 rather than the running sum
	ringRawAngle // θ + step without FUN_0044bc10's wrap
	ringSwapR    // vertex A on rB, B on rA
	ringColor2   // Color2 and Alpha2 under op 80
	ringPos2     // Pos2 added to the centre
	ringAlts
)

// ringWant is fillRing's contract for N ≥ 2, written out, or the
// misreading alt.
func ringWant(vm *VM, alt int) []RingVertex {
	n, l := vm.I[0], vm.I[1]
	step := float32(6.2831855) / float32(n-1)
	d8 := float32(l) / float32(n-1)
	rA := float32(vm.Scale[0]*0.5) + vm.Scale[1]
	rB := vm.Scale[1] - float32(vm.Scale[0]*0.5)
	if alt == ringSwapR {
		rA, rB = rB, rA
	}
	a, cr, cg, cb := vm.Alpha, vm.Color[0], vm.Color[1], vm.Color[2]
	if alt == ringColor2 {
		a, cr, cg, cb = vm.Alpha2, vm.Color2[0], vm.Color2[1], vm.Color2[2]
	}
	c := uint32(a)<<24 | uint32(cr)<<16 | uint32(cg)<<8 | uint32(cb)
	polar := zun.Polar
	if alt == ringStored {
		polar = func(t, r float32) (float32, float32) {
			s, c := math.Sincos(float64(t))
			return float32(float32(c) * r), float32(float32(s) * r)
		}
	}
	pos := vm.Pos
	if alt == ringPos2 {
		for i := range pos {
			pos[i] = pos[i] + vm.Pos2[i]
		}
	}

	theta, acc := vm.Rot[2], float32(0)
	vert := func(r float32) RingVertex {
		x, y := polar(theta, r)
		v := RingVertex{
			X: (vm.Offset[0] + pos[0]) + x, Y: (vm.Offset[1] + pos[1]) + y,
			Z: vm.Offset[2] + pos[2], Diffuse: c, U: vm.UV[0], V: acc + vm.UV[1],
		}
		if alt == ringGroup {
			v.X, v.Y = vm.Offset[0]+(pos[0]+x), vm.Offset[1]+(pos[1]+y)
		}
		return v
	}
	var out []RingVertex
	for i := range n - 1 {
		if alt == ringMulAcc {
			acc = float32(float32(i) * d8)
		}
		out = append(out, vert(rA), vert(rB))
		acc = acc + d8
		if alt == ringMulAcc {
			acc = float32(float32(i+1) * d8)
		}
		if alt == ringRawAngle {
			theta = theta + step
		} else {
			theta = zun.AddAngle(theta, step)
		}
	}
	a0, b0 := out[0], out[1]
	a0.V, b0.V = acc+vm.UV[1], acc+vm.UV[1]
	return append(out, a0, b0)
}

func eqRing(t *testing.T, what string, got, want []RingVertex) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d vertices, want %d", what, len(got), len(want))
	}
	for i := range got {
		if !sameRingVertex(got[i], want[i]) {
			t.Fatalf("%s: vertex %d\ngot  %+v\nwant %+v", what, i, got[i], want[i])
		}
	}
}

// sameRingVertex compares bits, so -0 and NaN are told apart.
func sameRingVertex(g, w RingVertex) bool {
	b := math.Float32bits
	return b(g.X) == b(w.X) && b(g.Y) == b(w.Y) && b(g.Z) == b(w.Z) &&
		g.Diffuse == w.Diffuse && b(g.U) == b(w.U) && b(g.V) == b(w.V)
}

func sameRing(a, b []RingVertex) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameRingVertex(a[i], b[i]) {
			return false
		}
	}
	return true
}

// ringVM is a mode 9 object with every field fillRing reads set to
// something awkward, and Pos2, Color2 and op 80 set to show they are not.
func ringVM(n, l int32) *VM {
	vm := New()
	vm.Flags |= 0x2400000 | 0x8000
	vm.I[0], vm.I[1] = n, l
	vm.RingN = n
	vm.Rot = [3]float32{1, 2, 0.5}
	vm.Scale = [2]float32{37.3, 101.7}
	vm.Pos = [3]float32{0.3, -0.7, 0.25}
	vm.Offset = [3]float32{-0.1, 0.2, 1.5}
	vm.Pos2 = [3]float32{11, 13, 17}
	vm.UV = [2]float32{0.3, 0.7}
	vm.Color, vm.Alpha = [3]uint8{0x12, 0x34, 0x56}, 0x78
	vm.Color2, vm.Alpha2 = [3]uint8{0x9a, 0xbc, 0xde}, 0xf0
	return vm
}

func TestFillRing(t *testing.T) {
	for _, tc := range []struct{ n, l int32 }{{8, 5}, {32, 3}, {9, 8}, {2, 1}} {
		vm := ringVM(tc.n, tc.l)
		want := ringWant(vm, ringExact)
		if tc.n == 8 {
			for alt := ringExact + 1; alt < ringAlts; alt++ {
				if sameRing(ringWant(vm, alt), want) {
					t.Fatalf("fixture N %d does not tell misreading %d apart", tc.n, alt)
				}
			}
		}
		vm.fillRing()
		eqRing(t, fmt.Sprintf("N %d, L %d", tc.n, tc.l), vm.Ring, want)
	}
}

// The closing pair repeats the first two vertices with the final V.
func TestFillRingClose(t *testing.T) {
	vm := ringVM(9, 8)
	vm.fillRing()
	r := vm.Ring
	if len(r) != 18 {
		t.Fatalf("%d vertices, want 18", len(r))
	}
	for i := range 2 {
		a, z := r[i], r[16+i]
		if z.V == a.V {
			t.Fatalf("closing vertex %d keeps V %v", i, a.V)
		}
		z.V = a.V
		if !sameRingVertex(a, z) {
			t.Fatalf("closing vertex %d\ngot  %+v\nwant %+v", i, r[16+i], a)
		}
	}
	// L 8 over 8 segments: d8 is 1 and the sum is exact.
	if r[16].V != 8+vm.UV[1] {
		t.Fatalf("closing V %v, want %v", r[16].V, 8+vm.UV[1])
	}
}

// With N ≤ 1 only the closing pair runs: two vertices, as they were, with
// V = 0 + UV[1].
func TestFillRingShort(t *testing.T) {
	for _, n := range []int32{1, 0, -3} {
		vm := ringVM(3, 1)
		vm.fillRing()
		old := vm.Ring[:2:2]
		old = append([]RingVertex(nil), old...)
		vm.I[0] = n
		vm.UV[1] = 0.25
		vm.fillRing()
		if len(vm.Ring) != 2 {
			t.Fatalf("N %d: %d vertices, want 2", n, len(vm.Ring))
		}
		for i := range 2 {
			w := old[i]
			w.V = 0.25
			if !sameRingVertex(vm.Ring[i], w) {
				t.Fatalf("N %d: vertex %d\ngot  %+v\nwant %+v", n, i, vm.Ring[i], w)
			}
		}

		// Nothing written before: zero vertices. 0 + -0 is +0.
		vm = ringVM(n, 1)
		vm.RingN = 1
		vm.UV[1] = float32(math.Copysign(0, -1))
		vm.fillRing()
		if len(vm.Ring) != 2 || !sameRingVertex(vm.Ring[0], RingVertex{}) || !sameRingVertex(vm.Ring[1], RingVertex{}) {
			t.Fatalf("N %d, fresh: %+v, want two zero vertices", n, vm.Ring)
		}
	}
}

// The fill panics where the original writes past op 84's buffer.
func TestFillRingOverflow(t *testing.T) {
	for _, tc := range []struct{ n, ringN int32 }{{5, 4}, {2, 1}, {2, 0}, {1, 0}, {0, 0}, {-1, 0}} {
		vm := ringVM(tc.n, 1)
		vm.RingN = tc.ringN
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("N %d, RingN %d: no panic", tc.n, tc.ringN)
				}
			}()
			vm.fillRing()
		}()
	}
	// N = RingN fills the buffer exactly.
	vm := ringVM(4, 1)
	vm.fillRing()
	if len(vm.Ring) != 8 {
		t.Fatalf("N = RingN: %d vertices, want 8", len(vm.Ring))
	}
}

func f32arg(x float32) int32 { return int32(math.Float32bits(x)) }

// endRun fills the ring in mode 9, after the tweens: the ring is what
// fillRing makes of the state the frame ends in.
func TestFillRingRun(t *testing.T) {
	f := synth(
		ins(0, 6, 1, 10000, 4),
		ins(0, 6, 1, 10001, 2),
		ins(0, 84, 1, 10000),
		ins(0, 58, 0, 4, 0, 0),
		ins(0, 56, 0, 4, 0, f32arg(40), f32arg(80), f32arg(0)),
		ins(1000, 2, 0),
	)
	env := testEnv()
	vm := New()
	vm.Start(f, 0, env)
	vm.Scale = [2]float32{8, 20}
	for frame := range 2 {
		vm.Run(env)
		if vm.Flags>>22&0xf != 9 || vm.RingN != 4 {
			t.Fatalf("frame %d: mode %d, RingN %d", frame, vm.Flags>>22&0xf, vm.RingN)
		}
		if vm.Alpha == 0xff || vm.Pos == ([3]float32{}) {
			t.Fatalf("frame %d: tweens did not step (alpha %d, pos %v)", frame, vm.Alpha, vm.Pos)
		}
		c := *vm
		c.Ring = nil
		c.fillRing()
		eqRing(t, "after Run", vm.Ring, c.Ring)
	}

	// Not in mode 9: no ring.
	g := synth(ins(0, 6, 1, 10000, 4), ins(1000, 2, 0))
	vm = New()
	vm.Start(g, 0, env)
	vm.Run(env)
	if vm.Ring != nil {
		t.Fatalf("mode 0 filled a ring: %+v", vm.Ring)
	}
}

// Op 67 with mode 10 would start the particles, which are not ported; the
// other modes only set the mode.
func TestMode10Panics(t *testing.T) {
	for _, mode := range []int32{8, 10, 11, 0x1a} {
		f := synth(ins(0, 67, 0, mode), ins(1000, 2, 0))
		vm := New()
		panicked := func() (p bool) {
			defer func() { p = recover() != nil }()
			vm.Start(f, 0, testEnv())
			return false
		}()
		if want := mode&0xf == 10; panicked != want {
			t.Errorf("op 67(%#x): panicked %v, want %v", mode, panicked, want)
		}
	}
}
