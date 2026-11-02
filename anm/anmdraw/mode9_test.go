package anmdraw

import (
	"fmt"
	"testing"

	"th10/anm"
	"th10/gfx"
)

// ring9 is a mode 9 object on sprite s with n segments: I[0] = n and a
// ring of 2n hand-made vertices, which draw9 takes as they are.
func ring9(f *anm.File, s int, n int32) *anm.VM {
	vm := obj(f, s, 9, 0, 0)
	vm.I[0] = n
	for i := range 2 * n {
		x := float32(i)
		vm.Ring = append(vm.Ring, anm.RingVertex{
			X: 100.25 + x, Y: 50.5 - x, Z: 0.125 * x, Diffuse: 0x80402010 + uint32(i),
			U: 0.1 + 0.01*x, V: 0.7 + 0.5*x,
		})
	}
	return vm
}

// strip9 is draw9's vertices for vm on sprite spr: the first 2·I[0] ring
// vertices with the sprite's U0 (even) or U1 (odd) added to U.
func strip9(vm *anm.VM, spr Sprite) []gfx.TLVertex {
	var vs []gfx.TLVertex
	for i, v := range vm.Ring[:2*vm.I[0]] {
		u := spr.U0 + v.U
		if i%2 == 1 {
			u = spr.U1 + v.U
		}
		vs = append(vs, gfx.TLVertex{X: v.X, Y: v.Y, Z: v.Z, RHW: 1, Diffuse: v.Diffuse, Specular: 0xff000000, U: u, V: v.V})
	}
	return vs
}

func TestDraw9(t *testing.T) {
	r, d, f := fixture(t)
	// A pending 2D quad on the same texture: draw9 flushes it first and
	// then needs no texture call.
	q := obj(f, 0, 2, 1, 1)
	q.Pos = [3]float32{100, 100, 0}
	r.Draw(q)
	d.take()

	vm := ring9(f, 0, 3)
	// The ring's own colours are drawn, not the object's, and Shift and
	// the tint do not apply.
	vm.Color, vm.Alpha = [3]uint8{1, 2, 3}, 4
	r.Shift = [2]float32{7, 9}
	r.TintOn, r.Tint = true, 0x40404040
	r.Draw(vm)
	calls := d.take()
	eqCalls(t, "draw9", calls, "draw prim 4, 6 vertices", "draw prim 5, 6 vertices")
	eqVerts(t, calls[1].vs, strip9(vm, sprites[0]))
	if r.vtype != vtypeRing {
		t.Fatalf("vertex type %d, want %d", r.vtype, vtypeRing)
	}

	// Only the first 2·I[0] vertices are drawn.
	vm = ring9(f, 0, 4)
	vm.I[0] = 3
	r.Draw(vm)
	calls = d.take()
	eqCalls(t, "longer ring", calls, "draw prim 5, 6 vertices")
	eqVerts(t, calls[0].vs, strip9(vm, sprites[0]))

	// Off screen it still draws: no cull.
	vm = ring9(f, 0, 2)
	for i := range vm.Ring {
		vm.Ring[i].X = -5000
	}
	r.Draw(vm)
	calls = d.take()
	eqCalls(t, "off screen", calls, "draw prim 5, 4 vertices")
	eqVerts(t, calls[0].vs, strip9(vm, sprites[0]))
}

// TestDraw9States checks the state calls and their order: flush, texture,
// vertex type, blend and filter.
func TestDraw9States(t *testing.T) {
	r, d, f := fixture(t)
	const add, point = 1 << 4, 1 << 31

	q := obj(f, 0, 2, 1, 1)
	q.Pos = [3]float32{100, 100, 0}
	r.Draw(q)
	d.take()
	vm := ring9(f, 1, 2)
	vm.Flags |= add | point
	r.Draw(vm)
	calls := d.take()
	eqCalls(t, "texture, blend, filter", calls,
		"draw prim 4, 6 vertices", "tex B", "blend 1", "filter 0", "draw prim 5, 4 vertices")
	eqVerts(t, calls[4].vs, strip9(vm, sprites[1]))

	vm = ring9(f, 1, 2)
	r.Draw(vm)
	eqCalls(t, "cached texture", d.take(), "blend 0", "filter 1", "draw prim 5, 4 vertices")

	// A 2D quad after the ring takes its vertex type back without a call,
	// and mode 8 after it sets its own.
	r.Draw(q)
	if r.vtype != vtype2D {
		t.Fatalf("2D after ring: vertex type %d", r.vtype)
	}
	r.Draw(ring9(f, 0, 2))
	r.Draw(obj(f, 0, 8, 0, 0))
	if r.vtype != vtype3D {
		t.Fatalf("mode 8 after ring: vertex type %d", r.vtype)
	}
	d.take()
	r.Draw(ring9(f, 0, 2))
	if r.vtype != vtypeRing {
		t.Fatalf("ring after mode 8: vertex type %d", r.vtype)
	}
	eqCalls(t, "ring after mode 8", d.take(), "draw prim 5, 4 vertices")
}

// With I[0] == 1 the primitive count is 0: the state calls happen and
// nothing is drawn, whatever the ring holds (no vertex is read).
func TestDraw9Empty(t *testing.T) {
	for _, seg := range []int32{1, 0} {
		r, d, f := fixture(t)
		q := obj(f, 0, 2, 1, 1)
		q.Pos = [3]float32{100, 100, 0}
		r.Draw(q)
		d.take()
		vm := ring9(f, 1, seg)
		vm.I[0] = 1
		vm.Flags |= 1 << 4
		r.Draw(vm)
		eqCalls(t, fmt.Sprintf("I[0] 1, %d vertices", len(vm.Ring)), d.take(), "draw prim 4, 6 vertices", "tex B", "blend 1")
		if r.vtype != vtypeRing {
			t.Fatalf("vertex type %d, want %d", r.vtype, vtypeRing)
		}
	}
}

// draw9 panics before any device call where the original's
// DrawPrimitiveUP is undefined.
func TestDraw9Panics(t *testing.T) {
	r, d, f := fixture(t)
	q := obj(f, 0, 2, 1, 1)
	q.Pos = [3]float32{100, 100, 0}
	for _, tc := range []struct {
		n, ring int32 // I[0], ring segments
	}{{0, 1}, {-1, 1}, {4, 3}, {2, 0}} {
		r.Draw(q)
		d.take()
		vm := ring9(f, 1, tc.ring)
		vm.I[0] = tc.n
		mustPanic(t, fmt.Sprintf("I[0] %d, %d vertices", tc.n, len(vm.Ring)), func() { r.Draw(vm) })
		eqCalls(t, "before the panic", d.take())
		r.Flush()
		d.take()
	}
}
