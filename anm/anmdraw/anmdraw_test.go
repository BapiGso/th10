package anmdraw

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"th10/anm"
	"th10/gfx"
)

// call is one Device call the recorder saw.
type call struct {
	op  string // "tex", "filter", "blend", "xform", "draw" or "draw3d"
	tex *gfx.Texture
	f   gfx.Filter
	b   gfx.BlendMode
	ts  gfx.TransformState
	m   gfx.Mat4
	p   gfx.Primitive
	vs  []gfx.TLVertex
	v3  []gfx.Vertex3D
}

func (c call) String() string {
	switch c.op {
	case "tex":
		return fmt.Sprintf("tex %s", texName(c.tex))
	case "filter":
		return fmt.Sprintf("filter %d", c.f)
	case "blend":
		return fmt.Sprintf("blend %d", c.b)
	case "xform":
		return fmt.Sprintf("xform %d", c.ts)
	case "draw3d":
		return fmt.Sprintf("draw3d prim %d, %d vertices", c.p, len(c.v3))
	}
	return fmt.Sprintf("draw prim %d, %d vertices", c.p, len(c.vs))
}

// rec is a Device that records its calls.
type rec struct{ calls []call }

func (d *rec) SetTexture(t *gfx.Texture) { d.calls = append(d.calls, call{op: "tex", tex: t}) }
func (d *rec) SetFilter(f gfx.Filter)    { d.calls = append(d.calls, call{op: "filter", f: f}) }
func (d *rec) SetBlend(b gfx.BlendMode)  { d.calls = append(d.calls, call{op: "blend", b: b}) }
func (d *rec) SetTransform(ts gfx.TransformState, m gfx.Mat4) {
	d.calls = append(d.calls, call{op: "xform", ts: ts, m: m})
}
func (d *rec) DrawTL(p gfx.Primitive, vs []gfx.TLVertex) {
	d.calls = append(d.calls, call{op: "draw", p: p, vs: slices.Clone(vs)})
}
func (d *rec) Draw3D(p gfx.Primitive, vs []gfx.Vertex3D) {
	d.calls = append(d.calls, call{op: "draw3d", p: p, v3: slices.Clone(vs)})
}

// take returns the calls since the last take.
func (d *rec) take() []call {
	c := d.calls
	d.calls = nil
	return c
}

var texA, texB = &gfx.Texture{W: 256, H: 256}, &gfx.Texture{W: 128, H: 128}

func texName(t *gfx.Texture) string {
	switch t {
	case texA:
		return "A"
	case texB:
		return "B"
	case nil:
		return "nil"
	}
	return "?"
}

var sprites = []Sprite{
	{Tex: texA, U0: 0.25, V0: 0.5, U1: 0.375, V1: 0.5625, W: 32, H: 16},
	{Tex: texB, U0: 0, V0: 0, U1: 1, V1: 1, W: 4, H: 2},
	{Tex: texA, U0: 0, V0: 0.5, U1: 0.5, V1: 1, W: 1, H: 1},
}

// fixture is a fresh renderer whose one file has sprites, viewing the game
// area (32, 16, 384, 448).
func fixture(t *testing.T) (*Renderer, *rec, *anm.File) {
	t.Helper()
	f := &anm.File{}
	sheet := &Sheet{Sprites: sprites}
	d := &rec{}
	r := NewRenderer(d, func(g *anm.File) *Sheet {
		if g != f {
			t.Fatalf("Sheet called with another file")
		}
		return sheet
	})
	r.View = [4]uint32{32, 16, 384, 448}
	return r, d, f
}

// obj is a drawable object of f: flag bits 0 and 1, the render mode and
// anchors, white, opaque, unscaled.
func obj(f *anm.File, sprite int, mode, ax, ay uint32) *anm.VM {
	vm := anm.New()
	vm.File = f
	vm.SetSprite(sprite)
	vm.Flags = 3 | mode<<22 | ax<<18 | ay<<20
	return vm
}

// tri is the six batch vertices of corners p, all with z and colour c and
// the UVs (u0, v0) to (u1, v1).
func tri(p [4][2]float32, z float32, c gfx.Color, u0, v0, u1, v1 float32) []gfx.TLVertex {
	uv := [4][2]float32{{u0, v0}, {u1, v0}, {u0, v1}, {u1, v1}}
	var q [4]gfx.TLVertex
	for i := range q {
		q[i] = gfx.TLVertex{X: p[i][0], Y: p[i][1], Z: z, RHW: 1, Diffuse: c, Specular: 0xff000000, U: uv[i][0], V: uv[i][1]}
	}
	return []gfx.TLVertex{q[0], q[1], q[2], q[1], q[2], q[3]}
}

// rect is tri for the axis-aligned quad x0..x1, y0..y1.
func rect(x0, y0, x1, y1, z float32, c gfx.Color, spr Sprite) []gfx.TLVertex {
	return tri([4][2]float32{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}}, z, c, spr.U0, spr.V0, spr.U1, spr.V1)
}

// drawOne draws vm into a flushed renderer and returns the batch it
// produced, nil if nothing was submitted.
func drawOne(t *testing.T, r *Renderer, d *rec, vm *anm.VM) []gfx.TLVertex {
	t.Helper()
	r.Flush()
	d.take()
	r.Draw(vm)
	r.Flush()
	var got []gfx.TLVertex
	for _, c := range d.take() {
		if c.op == "draw" {
			if got != nil {
				t.Fatalf("more than one draw call")
			}
			if c.p != gfx.TriangleList {
				t.Fatalf("primitive %d, want TriangleList", c.p)
			}
			got = c.vs
		}
	}
	return got
}

func eqVerts(t *testing.T, got, want []gfx.TLVertex) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d vertices, want %d\ngot  %+v\nwant %+v", len(got), len(want), got, want)
	}
	for i := range got {
		g, w := got[i], want[i]
		// Compare bits so -0 and NaN are told apart.
		same := math.Float32bits(g.X) == math.Float32bits(w.X) && math.Float32bits(g.Y) == math.Float32bits(w.Y) &&
			math.Float32bits(g.Z) == math.Float32bits(w.Z) && math.Float32bits(g.RHW) == math.Float32bits(w.RHW) &&
			g.Diffuse == w.Diffuse && g.Specular == w.Specular &&
			math.Float32bits(g.U) == math.Float32bits(w.U) && math.Float32bits(g.V) == math.Float32bits(w.V)
		if !same {
			t.Fatalf("vertex %d\ngot  %+v\nwant %+v", i, g, w)
		}
	}
}

func eqCalls(t *testing.T, step string, got []call, want ...string) {
	t.Helper()
	g := make([]string, len(got))
	for i, c := range got {
		g[i] = c.String()
	}
	if !slices.Equal(g, want) {
		t.Fatalf("%s: calls\n%q\nwant\n%q", step, g, want)
	}
}

const white gfx.Color = 0xffffffff

func TestQuad0(t *testing.T) {
	s0, s2 := sprites[0], sprites[2]
	for _, tc := range []struct {
		name       string
		sprite     int
		ax, ay     uint32
		pos, scale [2]float32
		view       [4]uint32
		x0, y0     float32 // after floor and snap
		x1, y1     float32
	}{
		// floor(100.25 − 16) = 84, floor(50.75 − 8) = 42, snapped to −0.5.
		{"centre", 0, 0, 0, [2]float32{100.25, 50.75}, [2]float32{1, 1}, [4]uint32{32, 16, 384, 448}, 83.5, 41.5, 115.5, 57.5},
		{"centre scaled", 0, 0, 0, [2]float32{100.25, 50.75}, [2]float32{2, 0.5}, [4]uint32{32, 16, 384, 448}, 67.5, 45.5, 131.5, 53.5},
		// floor, not truncation: floor(10.5 − 16) = −6.
		{"centre negative", 0, 0, 0, [2]float32{10.5, 50.75}, [2]float32{1, 1}, [4]uint32{0, 0, 640, 480}, -6.5, 41.5, 25.5, 57.5},
		// 100.25..132.25 and 50.75..66.75, snapped.
		{"top left", 0, 1, 1, [2]float32{100.25, 50.75}, [2]float32{1, 1}, [4]uint32{32, 16, 384, 448}, 99.5, 50.5, 131.5, 66.5},
		// 68.25..100.25 and 34.75..50.75, snapped.
		{"bottom right", 0, 2, 2, [2]float32{100.25, 50.75}, [2]float32{1, 1}, [4]uint32{32, 16, 384, 448}, 67.5, 34.5, 99.5, 50.5},
		// FRNDINT rounds half to even: 100.5 → 100, 101.5 → 102,
		// 61.5 → 62, 62.5 → 62.
		{"half to even", 2, 1, 1, [2]float32{100.5, 61.5}, [2]float32{1, 1}, [4]uint32{32, 16, 384, 448}, 99.5, 61.5, 101.5, 61.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, d, f := fixture(t)
			r.View = tc.view
			vm := obj(f, tc.sprite, 0, tc.ax, tc.ay)
			vm.Pos = [3]float32{tc.pos[0], tc.pos[1], 0.5}
			vm.Offset = [3]float32{0, 0, 0.25}
			vm.Pos2 = [3]float32{0, 0, 0.125}
			vm.Scale = tc.scale
			spr := s0
			if tc.sprite == 2 {
				spr = s2
			}
			eqVerts(t, drawOne(t, r, d, vm), rect(tc.x0, tc.y0, tc.x1, tc.y1, 0.875, white, spr))
		})
	}
}

func TestQuad2(t *testing.T) {
	s0 := sprites[0]
	for _, tc := range []struct {
		name           string
		ax, ay         uint32
		pos2y          float32
		x0, y0, x1, y1 float32
		z              float32
	}{
		// Z = (Pos2.y + Offset.z) + Pos.z: the ZUN bug ignores Pos2.z.
		{"centre", 0, 0, 0, 84.25, 42.75, 116.25, 58.75, 0.75},
		{"top left", 1, 1, 0, 100.25, 50.75, 132.25, 66.75, 0.75},
		{"bottom right", 2, 2, 0, 68.25, 34.75, 100.25, 50.75, 0.75},
		{"pos2.y", 1, 1, 2, 100.25, 52.75, 132.25, 68.75, 2.75},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, d, f := fixture(t)
			vm := obj(f, 0, 2, tc.ax, tc.ay)
			vm.Pos = [3]float32{100.25, 50.75, 0.5}
			vm.Offset = [3]float32{0, 0, 0.25}
			vm.Pos2 = [3]float32{0, tc.pos2y, 0.125}
			eqVerts(t, drawOne(t, r, d, vm), rect(tc.x0, tc.y0, tc.x1, tc.y1, tc.z, white, s0))
		})
	}
}

// TestGrouping pins the operand grouping of the sums near 2^24, where
// float32 spacing is 2 and (a + 1) + 1 differs from (1 + 1) + a.
func TestGrouping(t *testing.T) {
	const big = 1 << 24
	r, d, f := fixture(t)
	r.View = [4]uint32{0, 0, 1 << 26, 1 << 26}

	// Mode 2, anchors 1: x0 = Bx = (big + 1) + 1 = big;
	// x1 = ((big + 1) + 4) + 1 = big + 4; y0 = Cy = (1 + big) + 1 = big;
	// y1 = ((1 + big) + 2) + 1 = big + 4. Each rounds half to even.
	vm := obj(f, 1, 2, 1, 1)
	vm.Pos = [3]float32{big, big, 0.5}
	vm.Offset = [3]float32{1, 1, 0.25}
	vm.Pos2 = [3]float32{1, 1, 0}
	eqVerts(t, drawOne(t, r, d, vm), rect(big, big, big+4, big+4, 1.75, white, sprites[1]))

	// Mode 2, anchors 0: x0 = Ax − 2 = ((1 + 1) + big) − 2 = big;
	// y0 = Ay − 1 = big + 1 → big; x1 = 4 + x0, y1 = y0 + 2.
	vm.Flags = 3 | 2<<22
	eqVerts(t, drawOne(t, r, d, vm), rect(big, big, big+4, big+2, 1.75, white, sprites[1]))

	// Mode 0, Z = (Pos2.z + Offset.z) + Pos.z = (1 + 1) + big.
	vm = obj(f, 0, 0, 1, 1)
	vm.Pos = [3]float32{100.25, 50.75, big}
	vm.Offset = [3]float32{0, 0, 1}
	vm.Pos2 = [3]float32{0, 0, 1}
	eqVerts(t, drawOne(t, r, d, vm), rect(99.5, 50.5, 131.5, 66.5, big+2, white, sprites[0]))

	// Mode 0, anchor x 1 near 2^22 (spacing 0.5):
	// x1 = ((2^22 + 0.25) + 1) + 0.25 = 2^22 + 1, snapped to 2^22 + 0.5;
	// x0 = (2^22 + 0.25) + 0.25 = 2^22, snapped to 2^22 − 0.5.
	const mid = 1 << 22
	vm = obj(f, 2, 0, 1, 1)
	vm.Pos = [3]float32{mid, 50, 0.5}
	vm.Offset = [3]float32{0.25, 0, 0.25}
	vm.Pos2 = [3]float32{0.25, 0, 0.125}
	eqVerts(t, drawOne(t, r, d, vm), rect(mid-0.5, 49.5, mid+0.5, 50.5, 0.875, white, sprites[2]))
}

// rotatedWant restates the corner formulas of rotated.
func rotatedWant(vm *anm.VM, spr Sprite, ax, ay int) ([4][2]float32, float32) {
	c := float32(math.Cos(float64(vm.Rot[2])))
	s := float32(math.Sin(float64(vm.Rot[2])))
	w := float32(spr.W * vm.Scale[0])
	h := float32(spr.H * vm.Scale[1])
	cx := (vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]
	cy := (vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]
	var l, rr, tp, b float32
	switch ax {
	case 0:
		l, rr = float32(-0.5*w), float32(w*0.5)
	case 1:
		l, rr = 0, w
	case 2:
		l, rr = -w, 0
	}
	switch ay {
	case 0:
		tp, b = float32(h*-0.5), float32(h*0.5)
	case 1:
		tp, b = 0, h
	case 2:
		tp, b = -h, 0
	}
	p := [4][2]float32{
		{(float32(l*c) - float32(tp*s)) + cx, (float32(l*s) + float32(tp*c)) + cy},
		{(float32(rr*c) - float32(tp*s)) + cx, (float32(tp*c) + float32(rr*s)) + cy},
		{(float32(l*c) - float32(b*s)) + cx, (float32(l*s) + float32(c*b)) + cy},
		{(float32(rr*c) - float32(b*s)) + cx, (float32(rr*s) + float32(b*c)) + cy},
	}
	return p, (vm.Pos2[2] + vm.Offset[2]) + vm.Pos[2]
}

func TestRotated(t *testing.T) {
	s0 := sprites[0]
	for _, mode := range []uint32{1, 3} {
		for ax := range 3 {
			for ay := range 3 {
				t.Run(fmt.Sprintf("mode %d anchor %d,%d", mode, ax, ay), func(t *testing.T) {
					r, d, f := fixture(t)
					vm := obj(f, 0, mode, uint32(ax), uint32(ay))
					vm.Pos = [3]float32{200, 100, 0.5}
					vm.Offset = [3]float32{0.25, 0.5, 0.25}
					vm.Pos2 = [3]float32{1, 2, 0.125}
					vm.Scale = [2]float32{1.5, 0.75}
					vm.Rot[2] = 0.5
					p, z := rotatedWant(vm, s0, ax, ay)
					eqVerts(t, drawOne(t, r, d, vm), tri(p, z, white, s0.U0, s0.V0, s0.U1, s0.V1))
				})
			}
		}
	}
}

// TestUnrotated checks that modes 1 and 3 with Rot[2] == 0 are quad2.
func TestUnrotated(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 0, 0)
	vm.Pos = [3]float32{100.25, 50.75, 0.5}
	vm.Offset = [3]float32{0, 0, 0.25}
	vm.Pos2 = [3]float32{0, 3, 0.125}
	want := drawOne(t, r, d, vm)
	for _, mode := range []uint32{1, 3} {
		vm.Flags = 3 | mode<<22
		eqVerts(t, drawOne(t, r, d, vm), want)
	}
}

func TestAnchor3(t *testing.T) {
	s0, s1 := sprites[0], sprites[1]
	r, d, f := fixture(t)
	r.View = [4]uint32{0, 0, 640, 480}

	// The first quad sees FUN_00445900's staging quad: every corner at 0.
	vm := obj(f, 1, 2, 3, 3)
	vm.Pos = [3]float32{100, 100, 0.5}
	eqVerts(t, drawOne(t, r, d, vm), rect(0, 0, 0, 0, 0.5, white, s1))

	// Afterwards an anchor of 3 keeps that axis from the quad before.
	vm = obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100.25, 50.75, 0.5}
	eqVerts(t, drawOne(t, r, d, vm), rect(100.25, 50.75, 132.25, 66.75, 0.5, white, s0))
	vm = obj(f, 1, 2, 3, 1)
	vm.Pos = [3]float32{300, 200, 0.5}
	eqVerts(t, drawOne(t, r, d, vm), rect(100.25, 200, 132.25, 202, 0.5, white, s1))
	vm = obj(f, 1, 2, 1, 3)
	vm.Pos = [3]float32{10, 20, 0.5}
	eqVerts(t, drawOne(t, r, d, vm), rect(10, 200, 14, 202, 0.5, white, s1))

	// The rotated path would read uninitialised stack slots.
	vm = obj(f, 0, 1, 3, 0)
	vm.Rot[2] = 0.5
	mustPanic(t, "rotated anchor 3", func() { r.Draw(vm) })
}

func mustPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s: no panic", what)
		}
	}()
	fn()
}

func TestSkip(t *testing.T) {
	r, d, f := fixture(t)
	for _, tc := range []struct {
		name  string
		setup func(vm *anm.VM)
	}{
		{"no bit 0", func(vm *anm.VM) { vm.Flags &^= 1 }},
		{"no bit 1", func(vm *anm.VM) { vm.Flags &^= 2 }},
		{"alpha 0", func(vm *anm.VM) { vm.Alpha = 0 }},
		{"alpha 0 under colour 2", func(vm *anm.VM) { vm.Alpha = 0; vm.Alpha2 = 255; vm.Flags |= 0x8000 }},
		{"mode 10", func(vm *anm.VM) { vm.Flags |= 10 << 22 }},
		{"mode 15", func(vm *anm.VM) { vm.Flags |= 15 << 22 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vm := obj(f, 0, 0, 1, 1)
			vm.Pos = [3]float32{100, 100, 0}
			tc.setup(vm)
			r.Draw(vm)
			r.Flush()
			eqCalls(t, tc.name, d.take())
		})
	}

	for _, mode := range []uint32{5, 7} {
		vm := obj(f, 0, mode, 1, 1)
		vm.Pos = [3]float32{100, 100, 0}
		mustPanic(t, fmt.Sprintf("mode %d", mode), func() { r.Draw(vm) })
	}
	vm := obj(f, -1, 0, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}
	mustPanic(t, "no sprite", func() { r.Draw(vm) })
}

func TestColour(t *testing.T) {
	s0 := sprites[0]
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}
	vm.Color = [3]uint8{0x12, 0x34, 0x56}
	vm.Alpha = 0x78
	vm.Color2 = [3]uint8{0x9a, 0xbc, 0xde}
	vm.Alpha2 = 0xf0
	eqVerts(t, drawOne(t, r, d, vm), rect(100, 100, 132, 116, 0, 0x78123456, s0))

	vm.Flags |= 0x8000
	eqVerts(t, drawOne(t, r, d, vm), rect(100, 100, 132, 116, 0, 0xf09abcde, s0))

	// Colour 1's alpha gates the draw; colour 2's is what is drawn.
	vm.Alpha, vm.Alpha2 = 1, 0
	eqVerts(t, drawOne(t, r, d, vm), rect(100, 100, 132, 116, 0, 0x009abcde, s0))
}

func TestTint(t *testing.T) {
	s0 := sprites[0]
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}
	vm.Color = [3]uint8{0x90, 0x81, 0xff}
	vm.Alpha = 0xc8
	r.Tint = 0x80ff4003
	eqVerts(t, drawOne(t, r, d, vm), rect(100, 100, 132, 116, 0, 0xc890_81ff, s0))

	// A: 0xc8·0x80>>7 = 0xc8. R: 0x90·0xff>>7 = 286, clamped to 0xff.
	// G: 0x81·0x40>>7 = 0x40. B: 0xff·0x03>>7 = 5.
	r.TintOn = true
	eqVerts(t, drawOne(t, r, d, vm), rect(100, 100, 132, 116, 0, 0xc8ff4005, s0))
}

func TestUVScroll(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}
	vm.UV = [2]float32{0.125, 0.25}
	p := [4][2]float32{{100, 100}, {132, 100}, {100, 116}, {132, 116}}
	eqVerts(t, drawOne(t, r, d, vm), tri(p, 0, white, 0.375, 0.75, 0.5, 0.8125))
}

func TestShift(t *testing.T) {
	s0, s2 := sprites[0], sprites[2]
	r, d, f := fixture(t)
	r.Shift = [2]float32{0.5, 0.25}

	// Shift comes before the snap: 100.75 → 101, 132.75 → 133, 51, 67.
	vm := obj(f, 0, 0, 1, 1)
	vm.Pos = [3]float32{100.25, 50.75, 0}
	eqVerts(t, drawOne(t, r, d, vm), rect(100.5, 50.5, 132.5, 66.5, 0, white, s0))

	vm.Flags = 3 | 2<<22 | 1<<18 | 1<<20
	eqVerts(t, drawOne(t, r, d, vm), rect(100.75, 51, 132.75, 67, 0, white, s0))

	// And before the cull: 30..31 is outside x >= 32, 31..32 is not.
	vm = obj(f, 2, 2, 1, 1)
	vm.Pos = [3]float32{30, 100, 0}
	r.Shift = [2]float32{}
	if got := drawOne(t, r, d, vm); got != nil {
		t.Fatalf("unshifted quad drawn")
	}
	r.Shift = [2]float32{1, 0}
	eqVerts(t, drawOne(t, r, d, vm), rect(31, 100, 32, 101, 0, white, s2))
}

func TestCull(t *testing.T) {
	// View 32, 16, 384, 448: drawn while maxX >= 32, maxY >= 16,
	// minX <= 416, minY <= 464. Sprite 2 is 1x1, anchored top left. A NaN
	// position makes NaN bounds, which never cull (see TestCulled).
	nan := math.Float32frombits(0x7fc00000)
	for _, tc := range []struct {
		x, y  float32
		drawn bool
	}{
		{31, 100, true}, {30.5, 100, false},
		{416, 100, true}, {416.25, 100, false},
		{100, 15, true}, {100, 14.5, false},
		{100, 464, true}, {100, 464.25, false},
		{nan, 100, true}, {100, nan, true},
	} {
		t.Run(fmt.Sprintf("%v,%v", tc.x, tc.y), func(t *testing.T) {
			r, d, f := fixture(t)
			vm := obj(f, 2, 2, 1, 1)
			vm.Pos = [3]float32{tc.x, tc.y, 0}
			r.Draw(vm)
			r.Flush()
			calls := d.take()
			if !tc.drawn {
				// A culled quad changes nothing: not even the texture.
				eqCalls(t, "culled", calls)
				vm.Pos = [3]float32{100, 100, 0}
				r.Draw(vm)
				eqCalls(t, "next", d.take(), "tex A")
				return
			}
			eqCalls(t, "drawn", calls, "tex A", "draw prim 4, 6 vertices")
		})
	}
}

// TestCulled checks culled's selections on NaN directly: each keeps its
// second operand, so a NaN P1 becomes the bound and a NaN P0, P2 or P3 is
// passed over.
func TestCulled(t *testing.T) {
	nan := math.Float32frombits(0x7fc00000)
	// Corners in P0, P1, P2, P3 order; only X varies, Y is inside the view.
	for _, tc := range []struct {
		name   string
		x      [4]float32
		culled bool
	}{
		{"left of view", [4]float32{10, 20, 10, 20}, true},
		{"right of view", [4]float32{420, 430, 420, 430}, true},
		// P0 NaN: both bounds start at P1 = 20 and the quad is left of the view.
		{"P0 NaN", [4]float32{nan, 20, 10, 20}, true},
		// P1 NaN: both bounds are NaN from the start and stay NaN.
		{"P1 NaN", [4]float32{10, nan, 10, 20}, false},
		{"P2 NaN", [4]float32{10, 20, nan, 20}, true},
		{"P3 NaN", [4]float32{420, 430, 420, nan}, true},
		{"all NaN", [4]float32{nan, nan, nan, nan}, false},
		// Edges are inclusive: maxX == 32 and minX == 416 are drawn.
		{"touch left", [4]float32{31, 32, 31, 32}, false},
		{"touch right", [4]float32{416, 417, 416, 417}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, _ := fixture(t)
			for i := range r.quad {
				r.quad[i].X = tc.x[i]
				r.quad[i].Y = 100
			}
			if got := r.culled(); got != tc.culled {
				t.Errorf("culled() = %v, want %v", got, tc.culled)
			}
		})
	}
}

func TestStateSequence(t *testing.T) {
	r, d, f := fixture(t)
	draw := func(sprite int, extra uint32) {
		vm := obj(f, sprite, 2, 1, 1)
		vm.Pos = [3]float32{100, 100, 0}
		vm.Flags |= extra
		r.Draw(vm)
	}
	const add, b2, b3, point = 1 << 4, 2 << 4, 3 << 4, 1 << 31

	// FUN_00445900 leaves blend 0 and linear filtering cached, so only the
	// texture is set; the vertex type change flushes an empty batch.
	draw(0, 0)
	eqCalls(t, "first", d.take(), "tex A")
	draw(2, 0)
	eqCalls(t, "same texture", d.take())
	draw(1, 0)
	eqCalls(t, "texture B", d.take(), "draw prim 4, 12 vertices", "tex B")
	draw(1, add)
	eqCalls(t, "additive", d.take(), "draw prim 4, 6 vertices", "blend 1")
	draw(1, add|point)
	eqCalls(t, "point", d.take(), "draw prim 4, 6 vertices", "filter 0")
	r.Flush()
	eqCalls(t, "flush", d.take(), "draw prim 4, 6 vertices")
	r.Flush()
	eqCalls(t, "empty flush", d.take())

	// Blend 2 and 3 flush but set nothing.
	draw(1, b2|point)
	eqCalls(t, "blend 2", d.take())
	draw(1, b3|point)
	eqCalls(t, "blend 3", d.take(), "draw prim 4, 6 vertices")

	// Blend first, then the filter, and the second change finds the batch
	// already empty.
	draw(1, 0)
	eqCalls(t, "back to alpha", d.take(), "draw prim 4, 6 vertices", "blend 0", "filter 1")
	r.Flush()
	eqCalls(t, "final flush", d.take(), "draw prim 4, 6 vertices")
}

func TestInvalidate(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}
	vm.Flags |= 1 << 31

	r.Draw(vm)
	eqCalls(t, "first", d.take(), "tex A", "filter 0")

	// Invalidate does not flush.
	r.Invalidate()
	eqCalls(t, "invalidate", d.take())
	r.Flush()
	eqCalls(t, "flush", d.take(), "draw prim 4, 6 vertices")

	// Texture and blend are set again; the filter cache survives.
	r.Draw(vm)
	eqCalls(t, "after", d.take(), "tex A", "blend 0")
	r.Flush()
	d.take()

	// Blend 3 is what Invalidate caches, so it sets nothing.
	r.Invalidate()
	vm.Flags |= 3 << 4
	r.Draw(vm)
	eqCalls(t, "blend 3", d.take(), "tex A")
}

func TestReset(t *testing.T) {
	r, d, f := fixture(t)
	vm := obj(f, 0, 2, 1, 1)
	vm.Pos = [3]float32{100, 100, 0}
	r.Draw(vm)
	eqCalls(t, "first", d.take(), "tex A")
	r.TintOn, r.Tint, r.Shift = true, 0x12345678, [2]float32{3, 4}

	// Reset does not flush.
	r.Reset()
	eqCalls(t, "reset", d.take())
	if r.TintOn || r.Tint != 0x80808080 || r.Shift != [2]float32{} {
		t.Errorf("after Reset: TintOn %v, Tint %08x, Shift %v", r.TintOn, r.Tint, r.Shift)
	}
	r.Flush()
	eqCalls(t, "flush", d.take(), "draw prim 4, 6 vertices")

	// Texture, blend and filter are all set again, though the device
	// already has each of them.
	r.Draw(vm)
	eqCalls(t, "after", d.take(), "tex A", "blend 0", "filter 1")
	r.Flush()
	d.take()

	// The filter cache holds a value again.
	r.Draw(vm)
	eqCalls(t, "again", d.take())

	// A point-filtered first object after Reset sets the point filter.
	r.Flush()
	d.take()
	r.Reset()
	vm.Flags |= 1 << 31
	r.Draw(vm)
	eqCalls(t, "point after reset", d.take(), "tex A", "blend 0", "filter 0")
}
