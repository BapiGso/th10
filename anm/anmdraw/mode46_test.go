package anmdraw

import (
	"math"
	"testing"

	"th10/anm"
	"th10/gfx"
)

// cams is a fixed Cams.
type cams struct {
	right      [3]float32
	view, proj gfx.Mat4
	vp         gfx.Viewport
	eye        [3]float32
	near, far  float32
	bgr        [3]float32
}

func (c *cams) Selected() ([3]float32, gfx.Mat4, gfx.Mat4, gfx.Viewport) {
	return c.right, c.view, c.proj, c.vp
}

func (c *cams) FieldFog() ([3]float32, float32, float32, [3]float32) {
	return c.eye, c.near, c.far, c.bgr
}

// billboard is fixture with an identity camera on the viewport
// (100, 100, 2, 2): a position A projects to (101 + A.x, 101 − A.y, A.z),
// and right (2, 0, 0) gives f = 1. The Field camera sits 150 units from
// bill's E along x, with fog from 100 to 200 and fog bytes
// (10.9, 251.7, 0).
func billboard(t *testing.T) (*Renderer, *rec, *anm.File, *cams) {
	t.Helper()
	r, d, f := fixture(t)
	c := &cams{
		right: [3]float32{2, 0, 0},
		view:  gfx.Identity,
		proj:  gfx.Identity,
		vp:    gfx.Viewport{X: 100, Y: 100, W: 2, H: 2, MinZ: 0, MaxZ: 1},
		eye:   [3]float32{-150, -3, 0.5},
		near:  100,
		far:   200,
		bgr:   [3]float32{10.9, 251.7, 0},
	}
	r.Cams = c
	return r, d, f, c
}

// bill is a mode-mode object of f on sprite 1 (4x2) with colour
// 0xffc86432. Its position parts make corners' Ax = (1e8 + −1e8) + 1 = 1
// but draw6's Ex = (−1e8 + 1) + 1e8 = 0, so a swapped grouping moves the
// quad by a pixel or the fade distance from 150 to 151: corners projects
// to o = (102, 104, 0.5), and E = (0, −3, 0.5).
func bill(f *anm.File, mode, ax, ay uint32) *anm.VM {
	vm := obj(f, 1, mode, ax, ay)
	vm.Pos2 = [3]float32{1e8, 0, 0}
	vm.Offset = [3]float32{-1e8, 0, 0}
	vm.Pos = [3]float32{1, -3, 0.5}
	vm.Color = [3]uint8{200, 100, 50}
	vm.Alpha = 255
	return vm
}

// edges are the anchor tables for a w x h quad (rotated's).
func edges(w, h float32, ax, ay uint32) (l, r, tp, b float32) {
	switch ax {
	case 0:
		l, r = float32(-0.5*w), float32(w*0.5)
	case 1:
		l, r = 0, w
	case 2:
		l, r = -w, 0
	}
	switch ay {
	case 0:
		tp, b = float32(h*-0.5), float32(h*0.5)
	case 1:
		tp, b = 0, h
	case 2:
		tp, b = -h, 0
	}
	return
}

// eqCorners compares r.quad's X, Y, Z with p and z, and checks RHW is
// still 1.
func eqCorners(t *testing.T, what string, r *Renderer, p [4][2]float32, z float32) {
	t.Helper()
	for i, q := range r.quad {
		if math.Float32bits(q.X) != math.Float32bits(p[i][0]) || math.Float32bits(q.Y) != math.Float32bits(p[i][1]) ||
			math.Float32bits(q.Z) != math.Float32bits(z) || q.RHW != 1 {
			t.Fatalf("%s: P%d = (%v, %v, %v) RHW %v, want (%v, %v, %v) RHW 1", what, i, q.X, q.Y, q.Z, q.RHW, p[i][0], p[i][1], z)
		}
	}
}

func TestCorners(t *testing.T) {
	r, _, f, _ := billboard(t)
	for ax := range uint32(3) {
		for ay := range uint32(3) {
			vm := bill(f, 4, ax, ay)
			if !r.corners(vm, &sprites[1]) {
				t.Fatalf("anchor %d %d: rejected", ax, ay)
			}
			l, rr, tp, b := edges(4, 2, ax, ay)
			eqCorners(t, "unrotated", r, [4][2]float32{
				{102 + l, 104 + tp}, {102 + rr, 104 + tp}, {102 + l, 104 + b}, {102 + rr, 104 + b},
			}, 0.5)
		}
	}
}

// TestCornersScale: right (0, 3, 4) projects to (0, −3, 4) from o, so
// f = sqrt(16 + 9 + 0)·0.5 = 2.5, and Scale (1.5, 3) makes the 4x2 sprite
// 15x15.
func TestCornersScale(t *testing.T) {
	r, _, f, c := billboard(t)
	c.right = [3]float32{0, 3, 4}
	vm := bill(f, 4, 1, 1)
	vm.Scale = [2]float32{1.5, 3}
	if !r.corners(vm, &sprites[1]) {
		t.Fatal("rejected")
	}
	eqCorners(t, "scaled", r, [4][2]float32{{102, 104}, {117, 104}, {102, 119}, {117, 119}}, 0.5)
}

func TestCornersRotated(t *testing.T) {
	r, _, f, _ := billboard(t)
	for _, a := range [][2]uint32{{0, 0}, {1, 2}, {2, 1}} {
		vm := bill(f, 4, a[0], a[1])
		vm.Rot[2] = 0.7
		if !r.corners(vm, &sprites[1]) {
			t.Fatalf("anchor %v: rejected", a)
		}
		c := float32(math.Cos(float64(vm.Rot[2])))
		s := float32(math.Sin(float64(vm.Rot[2])))
		l, rr, tp, b := edges(4, 2, a[0], a[1])
		var ox, oy float32 = 102, 104
		eqCorners(t, "rotated", r, [4][2]float32{
			{(float32(l*c) - float32(tp*s)) + ox, (float32(tp*c) + float32(l*s)) + oy},
			{(float32(rr*c) - float32(tp*s)) + ox, (float32(rr*s) + float32(tp*c)) + oy},
			{(float32(l*c) - float32(b*s)) + ox, (float32(b*c) + float32(l*s)) + oy},
			{(float32(rr*c) - float32(b*s)) + ox, (float32(b*c) + float32(rr*s)) + oy},
		}, 0.5)
	}
}

// sentinel fills r.quad with values corners never computes here.
func sentinel(r *Renderer) [4]gfx.TLVertex {
	for i := range r.quad {
		r.quad[i].X, r.quad[i].Y, r.quad[i].Z = 7, 8, 9
		r.quad[i].Diffuse = 0x01020304
	}
	return r.quad
}

func TestCornersDepth(t *testing.T) {
	r, _, f, _ := billboard(t)
	for _, z := range []float32{-0.25, 1.25, 2} {
		want := sentinel(r)
		vm := bill(f, 4, 1, 1)
		vm.Pos[2] = z
		if r.corners(vm, &sprites[1]) {
			t.Errorf("z %v: accepted", z)
		}
		if r.quad != want {
			t.Errorf("z %v: rejected object wrote the quad", z)
		}
	}
	for _, z := range []float32{0, 1} {
		vm := bill(f, 4, 1, 1)
		vm.Pos[2] = z
		if !r.corners(vm, &sprites[1]) {
			t.Errorf("z %v: rejected", z)
		}
		if r.quad[3].Z != z {
			t.Errorf("z %v: Z = %v", z, r.quad[3].Z)
		}
	}
	// A NaN depth passes both tests.
	vm := bill(f, 4, 1, 1)
	vm.Pos[2] = float32(math.NaN())
	if !r.corners(vm, &sprites[1]) {
		t.Errorf("NaN z rejected")
	}
	if !math.IsNaN(float64(r.quad[0].Z)) {
		t.Errorf("NaN z: Z = %v", r.quad[0].Z)
	}
}

func TestCornersAnchor3(t *testing.T) {
	r, _, f, _ := billboard(t)
	mustPanic(t, "ax 3", func() { r.corners(bill(f, 4, 3, 0), &sprites[1]) })
	mustPanic(t, "ay 3", func() { r.corners(bill(f, 4, 0, 3), &sprites[1]) })
	// A rejected object never reads its anchors.
	vm := bill(f, 4, 3, 3)
	vm.Pos[2] = -1
	if r.corners(vm, &sprites[1]) {
		t.Errorf("z -1 accepted")
	}
}

// TestDraw4: corners, then emit with 0: Shift applies, nothing snaps, the
// colour is the object's.
func TestDraw4(t *testing.T) {
	r, d, f, _ := billboard(t)
	r.Shift = [2]float32{10, 20}
	vm := bill(f, 4, 0, 0)
	vm.Pos[0] = 1.25
	vm.Color = [3]uint8{0x12, 0x34, 0x56}
	vm.Alpha = 0x78
	got := drawOne(t, r, d, vm)
	eqVerts(t, got, tri([4][2]float32{{110.25, 123}, {114.25, 123}, {110.25, 125}, {114.25, 125}}, 0.5, 0x78123456, 0, 0, 1, 1))
}

// TestDraw6 checks the fade on bill's quad at distance 150. With fog 100
// to 200, f = 0.5 and colour 0xffc86432 gives
//
//	B: 50 − (50 − 10)·0.5 = 30           G: 100 − ftol((100 − 251)·0.5) = 100 + 75 = 175
//	R: 200 − (200 − 0)·0.5 = 100         A: ftol(0.5·255) = 127
//
// G truncates toward zero, fog byte 251.7 truncates to 251, and a fade
// distance of 151 (corners' grouping for E) would give R 98 and G 177.
func TestDraw6(t *testing.T) {
	quad := [4][2]float32{{100, 103}, {104, 103}, {100, 105}, {104, 105}}
	for _, tc := range []struct {
		name  string
		setup func(r *Renderer, c *cams, vm *anm.VM)
		want  gfx.Color
	}{
		{"f 0.5", func(*Renderer, *cams, *anm.VM) {}, 0x7f64af1e},
		// f = −50/50 = −1: B 50 + 40 = 90, G 100 − 151 wraps to 205,
		// R 200 + 200 wraps to 144, A 2·255 = 510 keeps its low byte 254.
		{"f -1 wraps", func(_ *Renderer, c *cams, _ *anm.VM) { c.far = 50 }, 0xfe90cd5a},
		// The tint applies first: A 127, R 255, G 100, B 50, then the fade.
		{"tint then fade", func(r *Renderer, _ *cams, _ *anm.VM) { r.TintOn, r.Tint = true, 0x40ff8080 }, 0x3f80af1e},
		{"colour 2", func(_ *Renderer, _ *cams, vm *anm.VM) {
			vm.Flags |= 0x8000
			vm.Color2, vm.Alpha2 = vm.Color, vm.Alpha
			vm.Color, vm.Alpha = [3]uint8{1, 2, 3}, 9
		}, 0x7f64af1e},
		{"near = d", func(_ *Renderer, c *cams, _ *anm.VM) { c.near = 150 }, 0xffc86432},
		{"near NaN", func(_ *Renderer, c *cams, _ *anm.VM) { c.near = float32(math.NaN()) }, 0xffc86432},
		{"tint, no fade", func(r *Renderer, c *cams, _ *anm.VM) {
			c.near = 150
			r.TintOn, r.Tint = true, 0x40ff8080
		}, 0x7fff6432},
		// f NaN: every ftol of a NaN is 0x8000000000000000, low byte 0.
		{"f NaN", func(_ *Renderer, c *cams, _ *anm.VM) { c.far = float32(math.NaN()) }, 0x00c86432},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, d, f, c := billboard(t)
			vm := bill(f, 6, 0, 0)
			tc.setup(r, c, vm)
			got := drawOne(t, r, d, vm)
			eqVerts(t, got, tri(quad, 0.5, tc.want, 0, 0, 1, 1))
		})
	}
}

// TestDraw6Cull: f >= 1 culls after corners staged the quad, before any
// colour is written.
func TestDraw6Cull(t *testing.T) {
	for _, far := range []float32{150, 120} {
		r, d, f, c := billboard(t)
		c.far = far
		sentinel(r)
		if got := drawOne(t, r, d, bill(f, 6, 0, 0)); got != nil {
			t.Errorf("far %v: drew %v", far, got)
		}
		if r.quad[0].X != 100 || r.quad[3].Y != 105 {
			t.Errorf("far %v: corners not staged: %+v", far, r.quad)
		}
		for i := range r.quad {
			if r.quad[i].Diffuse != 0x01020304 {
				t.Errorf("far %v: P%d colour %08x written", far, i, r.quad[i].Diffuse)
			}
		}
	}
}

func TestDraw6Rejected(t *testing.T) {
	r, d, f, _ := billboard(t)
	want := sentinel(r)
	vm := bill(f, 6, 0, 0)
	vm.Pos[2] = -1
	if got := drawOne(t, r, d, vm); got != nil {
		t.Errorf("drew %v", got)
	}
	if r.quad != want {
		t.Errorf("rejected object wrote the quad")
	}
}

func TestFtol(t *testing.T) {
	const indefinite = math.MinInt64
	for _, tc := range []struct {
		x    float32
		want int64
	}{
		{2.9, 2}, {-2.9, -2}, {0.5, 0}, {-0.5, 0}, {255.99, 255}, {-75.5, -75},
		{-9.223372e18, math.MinInt64}, // -2^63 itself is in range
		{9.2e18, int64(float32(9.2e18))},
		{9.223372e18, indefinite}, // 2^63
		{-1e19, indefinite},
		{float32(math.Inf(1)), indefinite},
		{float32(math.Inf(-1)), indefinite},
		{float32(math.NaN()), indefinite},
	} {
		if got := ftol(tc.x); got != tc.want {
			t.Errorf("ftol(%v) = %d, want %d", tc.x, got, tc.want)
		}
	}
}
