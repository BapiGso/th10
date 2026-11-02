package anmdraw

import (
	"math"
	"testing"

	"th10/anm"
	"th10/gfx"
)

// The spec of render mode 8 (mode8.go).

func eqMat(t *testing.T, what string, got, want gfx.Mat4) {
	t.Helper()
	for i := range 4 {
		for j := range 4 {
			if math.Float32bits(got[i][j]) != math.Float32bits(want[i][j]) {
				t.Fatalf("%s:\ngot  %v\nwant %v", what, got, want)
			}
		}
	}
}

func diag(x, y float32) gfx.Mat4 {
	return gfx.Mat4{{x, 0, 0, 0}, {0, y, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}}
}

// draw8One draws vm and returns the calls it made.
func draw8One(r *Renderer, d *rec, vm *anm.VM) []call {
	d.take()
	r.Draw(vm)
	return d.take()
}

// xform is the matrix of the only SetTransform(ts) call in calls.
func xform(t *testing.T, calls []call, ts gfx.TransformState) gfx.Mat4 {
	t.Helper()
	var m *gfx.Mat4
	for i := range calls {
		if calls[i].op == "xform" && calls[i].ts == ts {
			if m != nil {
				t.Fatalf("two SetTransform(%d) calls", ts)
			}
			m = &calls[i].m
		}
	}
	if m == nil {
		t.Fatalf("no SetTransform(%d) call", ts)
	}
	return *m
}

// verts3D is the vertices of the only Draw3D call in calls.
func verts3D(t *testing.T, calls []call) []gfx.Vertex3D {
	t.Helper()
	var vs []gfx.Vertex3D
	for _, c := range calls {
		if c.op == "draw3d" {
			if vs != nil {
				t.Fatalf("two Draw3D calls")
			}
			if c.p != gfx.TriangleStrip {
				t.Fatalf("Draw3D primitive %d, want TriangleStrip", c.p)
			}
			vs = c.v3
		}
	}
	if vs == nil {
		t.Fatalf("no Draw3D call")
	}
	return vs
}

func TestBaseTexBlock(t *testing.T) {
	eqMat(t, "base(sprite 0)", base(&sprites[0]), diag(0.125, 0.0625))
	eqMat(t, "base(sprite 1)", base(&sprites[1]), diag(4.0/256, 2.0/256))
	eqMat(t, "texBlock(sprite 0)", texBlock(&sprites[0]), diag(0.125, 0.0625))
	eqMat(t, "texBlock(sprite 1)", texBlock(&sprites[1]), diag(4.0/128, 2.0/128))
	// The quotient is rounded before the product: f32(f32(1/3)·5) is one
	// ulp above f32(5/3).
	odd := Sprite{Tex: &gfx.Texture{W: 3, H: 3}, W: 5, H: 7}
	eqMat(t, "texBlock(5x7 on 3x3)", texBlock(&odd), diag(math.Float32frombits(0x3fd55556), math.Float32frombits(0x40155556)))
}

func TestDraw8(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 8, 0, 0)
	vm.Pos = [3]float32{10, 20, 30}
	vm.Offset = [3]float32{1, 2, 3}
	vm.Pos2 = [3]float32{100, 200, 300}

	// Blend 0 and the linear filter are cached already; white differs
	// from the TEXTUREFACTOR cache's 1, which makes no device call here.
	calls := draw8One(r, d, vm)
	eqCalls(t, "first", calls, "xform 256", "tex A", "xform 16", "draw3d prim 5, 4 vertices")
	world := diag(0.125, 0.0625)
	world[3] = [4]float32{111, 222, 333, 1}
	eqMat(t, "world", xform(t, calls, gfx.World), world)
	tex := diag(0.125, 0.0625)
	tex[2][0], tex[2][1] = 0.25, 0.5
	eqMat(t, "texture", xform(t, calls, gfx.Texture0), tex)
	want := quad8
	for i := range want {
		want[i].Diffuse = white
	}
	if got := verts3D(t, calls); len(got) != 4 || [4]gfx.Vertex3D(got) != want {
		t.Fatalf("vertices\ngot  %+v\nwant %+v", got, want)
	}
	if vm.MatStale || vm.Mat != diag(0.125, 0.0625) {
		t.Fatalf("after the first draw: stale %v, Mat %v", vm.MatStale, vm.Mat)
	}

	// Same sprite, no horizontal scroll: only the world matrix and the draw.
	calls = draw8One(r, d, vm)
	eqCalls(t, "again", calls, "xform 256", "draw3d prim 5, 4 vertices")
	eqMat(t, "world again", xform(t, calls, gfx.World), world)
}

// TestDraw8Translation checks the translation row, with values that tell
// the binary's groupings from any other: 1e8 swallows 3 in float32.
func TestDraw8Translation(t *testing.T) {
	r, d, f := fixture(t)
	for _, tc := range []struct {
		ax, ay   uint32
		tx, ty   float32
		stale    bool
		wantNote string
	}{
		{0, 0, 3, 3, true, "(Pos2 + Offset) + Pos"},
		{1, 1, 0, 0, true, "((Pos2 + Offset) + Pos) − hw"},
		{2, 2, 3, 3, true, "((hw + Pos2) + Offset) + Pos"},
		{3, 3, 7, 9, false, "Mat's own row"},
		{1, 2, 0, 3, true, "mixed"},
	} {
		vm := obj(f, 0, 8, tc.ax, tc.ay)
		// hw = |32·0.1875·0.5| = 3, hh = |16·0.375·0.5| = 3.
		vm.Scale = [2]float32{0.1875, 0.375}
		vm.Pos = [3]float32{3, 3, -1e8}
		vm.Offset = [3]float32{-1e8, -1e8, 3}
		vm.Pos2 = [3]float32{1e8, 1e8, 1e8}
		mat := diag(0.125, 0.0625)
		if !tc.stale {
			vm.MatStale = false
			mat[3] = [4]float32{7, 9, 11, 1}
			vm.Mat = mat
		}
		calls := draw8One(r, d, vm)
		want := mat
		want[3][0], want[3][1], want[3][2] = tc.tx, tc.ty, 3
		eqMat(t, tc.wantNote, xform(t, calls, gfx.World), want)
	}

	// A negative scale (a flip) still gives a positive half size.
	vm := obj(f, 0, 8, 2, 1)
	vm.Scale = [2]float32{-2, -2}
	vm.Flags |= 8 // rebuild, so Mat carries the flip
	calls := draw8One(r, d, vm)
	want := diag(-0.25, -0.125)
	want[3] = [4]float32{32, -16, 0, 1}
	eqMat(t, "flipped", xform(t, calls, gfx.World), want)
}

func TestDraw8Rebuild(t *testing.T) {
	r, d, f := fixture(t)

	// Scale then X, Y, Z rotation, from base and not from Mat.
	vm := obj(f, 0, 8, 3, 3)
	vm.Flags |= 0xc
	vm.Scale = [2]float32{2, 3}
	vm.Rot = [3]float32{0.1, 0.2, 0.3}
	vm.MatStale = false
	vm.Mat = gfx.Mat4{{5, 5, 5, 5}, {5, 5, 5, 5}, {5, 5, 5, 5}, {5, 5, 5, 5}}
	m := diag(0.25, 0.1875)
	m = gfx.Mul(gfx.Mul(gfx.Mul(m, gfx.RotationX(0.1)), gfx.RotationY(0.2)), gfx.RotationZ(0.3))
	calls := draw8One(r, d, vm)
	if vm.Flags&0xc != 0 || vm.Flags&3 != 3 {
		t.Fatalf("flags after rebuild %#x", vm.Flags)
	}
	eqMat(t, "rebuilt Mat", vm.Mat, m)
	want := m
	want[3][2] = 0
	eqMat(t, "world from rebuilt Mat", xform(t, calls, gfx.World), want)

	// A zero angle is skipped, not multiplied in (only Z here).
	vm = obj(f, 0, 8, 0, 0)
	vm.Flags |= 4
	vm.Rot = [3]float32{0, 0, 0.5}
	draw8One(r, d, vm)
	eqMat(t, "Z only", vm.Mat, gfx.Mul(diag(0.125, 0.0625), gfx.RotationZ(0.5)))

	// Without a dirty bit Mat is kept as it is, scale or not.
	vm.Scale = [2]float32{4, 4}
	draw8One(r, d, vm)
	eqMat(t, "kept", vm.Mat, gfx.Mul(diag(0.125, 0.0625), gfx.RotationZ(0.5)))

	// Flags&0x4000 blocks the rebuild and keeps the dirty bits.
	vm = obj(f, 0, 8, 0, 0)
	vm.Flags |= 0x4000 | 0xc
	vm.Scale = [2]float32{2, 2}
	draw8One(r, d, vm)
	if vm.Flags&0xc != 0xc {
		t.Fatalf("0x4000: flags %#x", vm.Flags)
	}
	eqMat(t, "0x4000", vm.Mat, diag(0.125, 0.0625))
}

// TestDraw8TexMatrix checks when TEXTURE0 is set: a new sprite, or any
// horizontal scroll; a vertical scroll alone is missed.
func TestDraw8TexMatrix(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 8, 0, 0)
	draw8One(r, d, vm)

	vm.UV = [2]float32{0, 0.25}
	eqCalls(t, "vertical scroll", draw8One(r, d, vm), "xform 256", "draw3d prim 5, 4 vertices")

	vm.UV = [2]float32{0.125, 0.25}
	for _, step := range []string{"horizontal scroll", "horizontal scroll again"} {
		calls := draw8One(r, d, vm)
		eqCalls(t, step, calls, "xform 256", "xform 16", "draw3d prim 5, 4 vertices")
		tex := diag(0.125, 0.0625)
		tex[2][0], tex[2][1] = 0.375, 0.75
		eqMat(t, step, xform(t, calls, gfx.Texture0), tex)
	}

	vm.UV = [2]float32{float32(math.NaN()), 0}
	eqCalls(t, "NaN scroll", draw8One(r, d, vm), "xform 256", "xform 16", "draw3d prim 5, 4 vertices")

	// Another sprite on the same texture.
	vm2 := obj(f, 2, 8, 0, 0)
	calls := draw8One(r, d, vm2)
	eqCalls(t, "sprite 2", calls, "xform 256", "xform 16", "draw3d prim 5, 4 vertices")
	tex := diag(1.0/256, 1.0/256)
	tex[2][0], tex[2][1] = 0, 0.5
	eqMat(t, "sprite 2", xform(t, calls, gfx.Texture0), tex)

	// Invalidate and Reset forget the sprite (and the texture and blend).
	r.Invalidate()
	eqCalls(t, "after Invalidate", draw8One(r, d, vm2), "blend 0", "xform 256", "tex A", "xform 16", "draw3d prim 5, 4 vertices")
	r.Reset()
	eqCalls(t, "after Reset", draw8One(r, d, vm2), "blend 0", "filter 1", "xform 256", "tex A", "xform 16", "draw3d prim 5, 4 vertices")
}

// TestDraw8States checks FUN_004423e0's blend and TEXTUREFACTOR against
// the 2D path's, and the call order around the world matrix.
func TestDraw8States(t *testing.T) {
	r, d, f := fixture(t)
	const add, b2, b3, point = 1 << 4, 2 << 4, 3 << 4, 1 << 31
	draw := func(extra uint32) []call {
		vm := obj(f, 0, 8, 0, 0)
		vm.Flags |= extra
		return draw8One(r, d, vm)
	}
	draw(0)
	eqCalls(t, "additive point", draw(add|point), "blend 1", "filter 0", "xform 256", "draw3d prim 5, 4 vertices")
	eqCalls(t, "blend 2 is set", draw(b2), "blend 1", "filter 1", "xform 256", "draw3d prim 5, 4 vertices")
	eqCalls(t, "blend 3 is not", draw(b3), "xform 256", "draw3d prim 5, 4 vertices")
	eqCalls(t, "back to alpha", draw(0), "blend 0", "xform 256", "draw3d prim 5, 4 vertices")

	// F.3: a 2D blend 2 caches 2 without a call, and a 3D blend 2 then
	// sets nothing either. The 2D batch is flushed before anything else.
	vm2D := obj(f, 0, 2, 1, 1)
	vm2D.Pos = [3]float32{100, 100, 0}
	vm2D.Flags |= b2
	d.take()
	r.Draw(vm2D)
	eqCalls(t, "2D blend 2", d.take())
	eqCalls(t, "3D blend 2 after 2D", draw(b2), "draw prim 4, 6 vertices", "xform 256", "draw3d prim 5, 4 vertices")

	// And a 2D object after mode 8 sets its vertex type without a draw.
	d.take()
	r.Draw(vm2D)
	r.Flush()
	eqCalls(t, "2D after 3D", d.take(), "draw prim 4, 6 vertices")
}

// TestDraw8Factor checks the TEXTUREFACTOR stand-in: every vertex carries
// the device's factor, which only changes when the cache misses.
func TestDraw8Factor(t *testing.T) {
	r, d, f := fixture(t)
	diffuse := func(vm *anm.VM) gfx.Color {
		vs := verts3D(t, draw8One(r, d, vm))
		for _, v := range vs[1:] {
			if v.Diffuse != vs[0].Diffuse {
				t.Fatalf("corner colours differ: %+v", vs)
			}
		}
		return vs[0].Diffuse
	}
	// Colour 1 (op 80 with a zero alpha 2) hits the initial cache, so the
	// device keeps its default factor.
	one := obj(f, 0, 8, 0, 0)
	one.Flags |= 0x8000
	one.Color2, one.Alpha2 = [3]uint8{0, 0, 1}, 0
	if c := diffuse(one); c != 0xffffffff {
		t.Fatalf("colour 1 first: %08x, want ffffffff", c)
	}

	vm := obj(f, 0, 8, 0, 0)
	vm.Color, vm.Alpha = [3]uint8{0xff, 0x40, 0x20}, 0x80
	if c := diffuse(vm); c != 0x80ff4020 {
		t.Fatalf("colour: %08x", c)
	}
	r.TintOn, r.Tint = true, 0x80408080
	if c := diffuse(vm); c != 0x807f4020 {
		t.Fatalf("tinted: %08x, want 807f4020", c)
	}
	r.TintOn = false

	// Neither Reset nor Invalidate forgets the factor cache: colour 1 now
	// misses it.
	r.Reset()
	r.Invalidate()
	if c := diffuse(one); c != 1 {
		t.Fatalf("colour 1 after Reset: %08x, want 00000001", c)
	}
}
