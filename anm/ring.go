package anm

import "th10/zun"

// Render mode 9 (op 84): a ring of quads around the object, built by the
// VM at the end of each frame and drawn as one triangle strip by
// anmdraw (FUN_00444ce0). Only bullet.anm uses it: the bomb children of
// scripts 400, 407 and 441 (N 32, L 3) and the spell card declaration's
// 417 (N 32, L 3) and 427 (N 9, L 8), docs/research/m4_modes.md.

// RingVertex is one vertex of the ring buffer (VM+0x358), the original's
// TL vertex (FVF 0x144, 0x1c bytes, RHW always 1) without RHW, and with U
// short of the sprite's offset: the original stores spr.U0 + UV[0] for
// even vertices and spr.U1 + UV[0] for odd ones, from the sprite record
// the VM does not have, so U holds UV[0] and the renderer adds the
// sprite's U0 or U1 (one float32 addition, as in the original).
type RingVertex struct {
	X, Y, Z float32
	Diffuse uint32 // 0xAARRGGBB
	U, V    float32
}

// fillRing is the mode 9 step of FUN_0043ee30's frame tail
// (0x44115d-0x44133e), which endRun calls after the tweens and before the
// timer tick when Flags&0x3c00000 == 0x2400000. It writes vm.Ring.
//
// With N = I[0] ($10000, VM+0x30c), L = I[1] ($10001, VM+0x310):
//
// Panics if the writes pass op 84's allocation, malloc(RingN·0x38), room
// for 2·RingN vertices: if N ≥ 2 and N > RingN, or N ≤ 1 and RingN < 1.
// RingN 0 also covers mode 9 set by op 67 with no op 84, where the
// buffer pointer is the reset's null.
//
// N ≤ 1: the loop is skipped (JLE at 0x4411b7 on TEST ESI,ESI at
// 0x44117e; nothing between touches EFLAGS) and only the closing pair
// runs, copying vertices 0 and 1 onto themselves: vm.Ring has length 2,
// each vertex what it was (the zero vertex where vm.Ring was shorter),
// with V = 0 + UV[1]. Nothing draws these: the draw's primitive count
// 2·N − 2 is 0 or less (anmdraw draw9).
//
// N ≥ 2: vm.Ring has length 2N. With every operation rounded to float32
// in the order written (x87 at 24-bit precision; wrap products in
// float32() so they do not fuse):
//
//	step = 2π / float32(N−1)       0x470b14 (6.2831855), 0x441184-0x4411a5
//	d8   = float32(L) / float32(N−1)                       0x4411a9-0x4411b1
//	rA   = Scale[0]·0.5 + Scale[1]                         0x441200-0x44120c
//	rB   = Scale[1] − Scale[0]·0.5                         0x441288-0x441291
//	c    = Alpha<<24 | Color[0]<<16 | Color[1]<<8 | Color[2]
//	θ    = Rot[2], acc = 0
//
// FILD loads N−1 and L exactly, and float32() of them is exact below
// 2^24, which both shipped values are far under. c is the DWORD at
// VM+0x2fc as anmdraw's colour lays it out (m4_r7.md unresolved #5): never
// Color2, whatever op 80 says, and no tint.
//
// Then for i = 0 .. N−2, with (x, y) = zun.Polar(θ, r) (FUN_00441ef0:
// FSINCOS extended, each product rounded, nothing stored between):
//
//	Ring[2i]   = {X: (Offset[0] + Pos[0]) + x, Y: (Offset[1] + Pos[1]) + y,
//	              Z: Offset[2] + Pos[2], Diffuse: c, U: UV[0],
//	              V: acc + UV[1]}                     r = rA, 0x4411d0-0x441259
//	Ring[2i+1] = the same with r = rB                    0x44125c-0x4412e9
//	acc = acc + d8                                       0x4412ec-0x4412f4
//	θ   = zun.AddAngle(θ, step)              FUN_0044bc10, 0x4412f8-0x4412fe
//
// and the closing pair (0x441308-0x44133b), which joins the strip's end
// to its start:
//
//	Ring[2N−2] = Ring[0] with V = acc + UV[1]
//	Ring[2N−1] = Ring[1] with V = acc + UV[1]
//
// Pos2 is not used, and Offset is whatever the frame tail left it (flag
// 0x2000's scroll is not ported, see endRun).
func (vm *VM) fillRing() {
	n := vm.I[0]
	if (n >= 2 && n > vm.RingN) || (n <= 1 && vm.RingN < 1) {
		panic("anm: fillRing writes past op 84's buffer")
	}
	if n <= 1 {
		// Loop skipped; closing pair only (0x441308-0x44133b), on the
		// vertices as they were.
		for len(vm.Ring) < 2 {
			vm.Ring = append(vm.Ring, RingVertex{})
		}
		vm.Ring = vm.Ring[:2]
		vm.Ring[0].V = 0 + vm.UV[1]
		vm.Ring[1].V = 0 + vm.UV[1]
		return
	}

	cnt := 2 * int(n)
	if cap(vm.Ring) >= cnt {
		vm.Ring = vm.Ring[:cnt]
	} else {
		vm.Ring = make([]RingVertex, cnt)
	}
	// Step, d8, rA, rB and the colour (0x441184-0x441291).
	step := float32(6.2831855) / float32(n-1)
	d8 := float32(vm.I[1]) / float32(n-1)
	rA := float32(vm.Scale[0]*0.5) + vm.Scale[1]
	rB := vm.Scale[1] - float32(vm.Scale[0]*0.5)
	c := uint32(vm.Alpha)<<24 | uint32(vm.Color[0])<<16 | uint32(vm.Color[1])<<8 | uint32(vm.Color[2])
	theta, acc := vm.Rot[2], float32(0)
	vert := func(r float32) RingVertex {
		x, y := zun.Polar(theta, r)
		return RingVertex{
			X: (vm.Offset[0] + vm.Pos[0]) + x, Y: (vm.Offset[1] + vm.Pos[1]) + y,
			Z: vm.Offset[2] + vm.Pos[2], Diffuse: c, U: vm.UV[0], V: acc + vm.UV[1],
		}
	}
	// Segments (0x4411d0-0x4412fe).
	for i := range int(n) - 1 {
		vm.Ring[2*i] = vert(rA)
		vm.Ring[2*i+1] = vert(rB)
		acc = acc + d8
		theta = zun.AddAngle(theta, step)
	}
	// Closing pair (0x441308-0x44133b): the strip's end joins its start.
	a, b := vm.Ring[0], vm.Ring[1]
	a.V, b.V = acc+vm.UV[1], acc+vm.UV[1]
	vm.Ring[cnt-2], vm.Ring[cnt-1] = a, b
}
