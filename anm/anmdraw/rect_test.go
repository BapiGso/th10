package anmdraw

import (
	"testing"

	"th10/gfx"
)

// rectStrip is the four strip vertices Rect draws for v in colour c.
func rectStrip(v [4]float32, c gfx.Color) []gfx.TLVertex {
	p := [4][2]float32{{v[0], v[1]}, {v[2], v[1]}, {v[0], v[3]}, {v[2], v[3]}}
	q := make([]gfx.TLVertex, 4)
	for i := range q {
		q[i] = gfx.TLVertex{X: p[i][0], Y: p[i][1], RHW: 1, Diffuse: c, Specular: 0xff000000}
	}
	return q
}

func TestRect(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}

	// The pending batch goes first; then the untextured alpha-blended strip.
	// Shift does not apply and nothing is snapped.
	r.Draw(vm)
	eqCalls(t, "object", d.take(), "tex A")
	r.Shift = [2]float32{3, 4}
	v := [4]float32{208.5, -7.25, 246.75, -5.25}
	r.Rect(v, 0x80000000)
	got := d.take()
	eqCalls(t, "rect", got, "draw prim 4, 6 vertices", "tex nil", "blend 0", "draw prim 5, 4 vertices")
	eqVerts(t, got[3].vs, rectStrip(v, 0x80000000))
	r.Shift = [2]float32{}

	// The caches are invalidated, so the next object sets its texture and
	// blend again; the filter cache survives.
	r.Draw(vm)
	eqCalls(t, "object after", d.take(), "tex A", "blend 0")

	// Back to back with an empty batch: no flush draw, the state set again.
	r.Flush()
	d.take()
	v = [4]float32{207, -8, 245, -6}
	r.Rect(v, 0xffffffff)
	r.Rect(v, 0xffffffff)
	got = d.take()
	eqCalls(t, "twice", got, "tex nil", "blend 0", "draw prim 5, 4 vertices", "tex nil", "blend 0", "draw prim 5, 4 vertices")
	eqVerts(t, got[2].vs, rectStrip(v, 0xffffffff))

	// A filled batch after a rect is not affected by it.
	r.Draw(vm)
	r.Flush()
	got = d.take()
	eqCalls(t, "batch after", got, "tex A", "blend 0", "draw prim 4, 6 vertices")
	eqVerts(t, got[2].vs, rect(100, 100, 132, 116, 0, white, sprites[0])) // anchor 1: left/top (FUN_00443290 0x443307, 0x4433c9)
}
