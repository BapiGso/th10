package gfx

import (
	"image"
	"image/color"
	"math"
	"testing"

	"th10/std"

	"github.com/hajimehoshi/ebiten/v2"
)

// The spec of the CPU pipeline and the fragment reference. Expected values
// are worked out by hand from the contracts in pipeline.go and shader.go,
// picked so float32 arithmetic is exact unless a tolerance is given.

func near(a, b, eps float32) bool { return float32(math.Abs(float64(a-b))) <= eps }

func TestMul4(t *testing.T) {
	m := Mat4{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10, 11, 12}, {13, 14, 15, 16}}
	if got := mul4(Identity, m); got != m {
		t.Errorf("I·M = %v", got)
	}
	a := Mat4{{1, 2, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {3, 4, 5, 1}}
	b := Mat4{{2, 0, 0, 0}, {0, 3, 0, 0}, {0, 0, 4, 0}, {1, 1, 1, 1}}
	want := Mat4{{2, 6, 0, 0}, {0, 3, 0, 0}, {0, 0, 4, 0}, {7, 13, 21, 1}}
	if got := mul4(a, b); got != want {
		t.Errorf("A·B = %v, want %v", got, want)
	}
}

func TestToClip(t *testing.T) {
	world := Mat4{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 12, 1}}
	proj := Mat4{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 2, 1}, {0, 0, -6, 0}} // z' = 2z − 6, w = z
	wv := mul4(world, Identity)
	wvp := mul4(wv, proj)
	v := Vertex3D{X: 3, Y: 4, Z: 0, Diffuse: 0x80402010, U: 0.25, V: 0.75}

	c := toClip(v, wvp, wv, Fog{})
	want := clipVertex{X: 3, Y: 4, Z: 18, W: 12, Diffuse: 0x80402010, U: 0.25, V: 0.75, Fog: 1}
	if c != want {
		t.Errorf("no fog: %+v, want %+v", c, want)
	}
	// View-space z is 12: (26 − 12)/26.
	c = toClip(v, wvp, wv, Fog{Mode: FogVertexLinear, Start: 0, End: 26})
	if c.Fog != float32(14)/26 {
		t.Errorf("vertex fog %v, want 14/26", c.Fog)
	}
	// View-space distance is |(3, 4, 12)| = 13: (26 − 13)/26.
	c = toClip(v, wvp, wv, Fog{Mode: FogVertexLinear, Start: 0, End: 26, Range: true})
	if c.Fog != 0.5 {
		t.Errorf("range fog %v, want 0.5", c.Fog)
	}
	// Table fog is the shader's business; the vertex carries 1.
	c = toClip(v, wvp, wv, Fog{Mode: FogTableLinear, Start: 0, End: 26})
	if c.Fog != 1 {
		t.Errorf("table fog vertex factor %v, want 1", c.Fog)
	}
}

func TestTexCoord(t *testing.T) {
	// Row 3 must not count: (u, v, 1, 0).
	m := Mat4{{2, 0.5, 0, 0}, {0.25, 4, 0, 0}, {3, 5, 9, 9}, {100, 100, 100, 100}}
	if u, v := texCoord(&m, 0.5, 0.25); u != 4.0625 || v != 6.25 {
		t.Errorf("texCoord = (%v, %v), want (4.0625, 6.25)", u, v)
	}
	if u, v := texCoord(&Identity, 0.25, 0.75); u != 0.25 || v != 0.75 {
		t.Errorf("identity: (%v, %v)", u, v)
	}
	// (1 + 2⁻²⁴) + 2⁻²⁴ rounds to 1 twice; 1 + (2⁻²⁴ + 2⁻²⁴) would not.
	e := float32(math.Ldexp(1, -24))
	m = Mat4{{1, 1, 0, 0}, {e, e, 0, 0}, {e, e, 0, 0}, {0, 0, 0, 1}}
	if u, v := texCoord(&m, 1, 1); u != 1 || v != 1 {
		t.Errorf("grouping: (%v, %v), want (1, 1)", u, v)
	}
}

// TestDraw3DTexture0 checks that Draw3D runs U, V through the TEXTURE0
// matrix before anything else: with identity world, view and projection
// (w = 1) the batch's Custom0/Custom1 are the transformed coordinates.
func TestDraw3DTexture0(t *testing.T) {
	var d Device
	d.Reset()
	d.SetZEnable(false)
	d.SetCull(CullNone)
	d.SetTransform(Texture0, Mat4{{0.5, 0, 0, 0}, {0, 0.25, 0, 0}, {0.125, 0.75, 1, 0}, {0, 0, 0, 1}})
	d.Draw3D(TriangleList, []Vertex3D{
		{X: -0.5, Y: -0.5, Z: 0.5, U: 0, V: 0},
		{X: 0.5, Y: -0.5, Z: 0.5, U: 1, V: 0},
		{X: -0.5, Y: 0.5, Z: 0.5, U: 0, V: 1},
	})
	want := [][2]float32{{0.125, 0.75}, {0.625, 0.75}, {0.125, 1}}
	if len(d.batch) != 3 {
		t.Fatalf("%d batch vertices, want 3", len(d.batch))
	}
	for i, w := range want {
		if v := d.batch[i]; v.Custom2 != 1 || v.Custom0 != w[0] || v.Custom1 != w[1] {
			t.Errorf("vertex %d: u·q %v, v·q %v, q %v; want %v, q 1", i, v.Custom0, v.Custom1, v.Custom2, w)
		}
	}
}

func TestFogLinear(t *testing.T) {
	for _, tc := range []struct{ start, end, d, want float32 }{
		{100, 500, 100, 1},
		{100, 500, 300, 0.5},
		{100, 500, 500, 0},
		{100, 500, 600, 0},
		{100, 500, 50, 1},
		{10, 10, 9, 1},
		{10, 10, 10, 0},
		{10, 10, 11, 0},
	} {
		if got := fogLinear(tc.start, tc.end, tc.d); got != tc.want {
			t.Errorf("fogLinear(%v, %v, %v) = %v, want %v", tc.start, tc.end, tc.d, got, tc.want)
		}
	}
}

// cyclicEqual reports whether got is want rotated.
func cyclicEqual(got, want []clipVertex) bool {
	if len(got) != len(want) {
		return false
	}
	if len(got) == 0 {
		return true
	}
	for r := range got {
		ok := true
		for i := range want {
			if got[(i+r)%len(got)] != want[i] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func TestClipPoly(t *testing.T) {
	inside := []clipVertex{
		{X: 0, Y: 0, Z: 0, W: 1, Fog: 1}, // on the near plane: inside
		{X: 2, Y: 0, Z: 1, W: 3, Fog: 1},
		{X: 0, Y: 2, Z: 3, W: 3, Fog: 1}, // on the far plane: inside
	}
	buf := make([]clipVertex, 1, 8)
	got := clipPoly(buf, inside)
	if !cyclicEqual(got, inside) {
		t.Errorf("inside: %+v", got)
	}
	if &got[0] != &buf[0] {
		t.Error("clipPoly did not append to dst[:0]")
	}

	// A is behind the near plane (z = −1). A→B crosses at t = 0.5; C→A at
	// t = 1/(1 − (−1)) = 0.5 from C.
	a := clipVertex{X: 0, Y: 0, Z: -1, W: 1, Diffuse: 0xff000000, U: 0, V: 0, Fog: 1}
	b := clipVertex{X: 2, Y: 0, Z: 1, W: 3, Diffuse: 0xffc80000, U: 1, V: 0, Fog: 0}
	c := clipVertex{X: 0, Y: 2, Z: 1, W: 3, Diffuse: 0xff0000c8, U: 0, V: 1, Fog: 1}
	iab := clipVertex{X: 1, Y: 0, Z: 0, W: 2, Diffuse: 0xff640000, U: 0.5, V: 0, Fog: 0.5}
	ica := clipVertex{X: 0, Y: 1, Z: 0, W: 2, Diffuse: 0xff000064, U: 0, V: 0.5, Fog: 1}
	if got := clipPoly(nil, []clipVertex{a, b, c}); !cyclicEqual(got, []clipVertex{iab, b, c, ica}) {
		t.Errorf("near: %+v", got)
	}

	// A is beyond the far plane (w − z = −1). A→B: t = −1/(−1 − 3) = 0.25;
	// C→A: t = 3/(3 + 1) = 0.75 from C.
	a = clipVertex{X: 0, Y: 0, Z: 5, W: 4, Fog: 1}
	b = clipVertex{X: 4, Y: 0, Z: 1, W: 4, Fog: 1}
	c = clipVertex{X: 0, Y: 4, Z: 1, W: 4, Fog: 1}
	iab = clipVertex{X: 1, Y: 0, Z: 4, W: 4, Fog: 1}
	ica = clipVertex{X: 0, Y: 1, Z: 4, W: 4, Fog: 1}
	if got := clipPoly(nil, []clipVertex{a, b, c}); !cyclicEqual(got, []clipVertex{iab, b, c, ica}) {
		t.Errorf("far: %+v", got)
	}

	behind := []clipVertex{{Z: -1, W: 1}, {X: 1, Z: -2, W: 1}, {Y: 1, Z: -0.5, W: 1}}
	if got := clipPoly(nil, behind); len(got) != 0 {
		t.Errorf("behind: %+v", got)
	}
	beyond := []clipVertex{{Z: 2, W: 1}, {X: 1, Z: 3, W: 1}, {Y: 1, Z: 1.5, W: 1}}
	if got := clipPoly(nil, beyond); len(got) != 0 {
		t.Errorf("beyond: %+v", got)
	}
}

func TestProject(t *testing.T) {
	vp := Viewport{32, 16, 384, 448, 0, 1}
	got := project(clipVertex{X: 0, Y: 0, Z: 1, W: 2, Diffuse: 0x12345678, U: 0.25, V: 0.75, Fog: 0.5}, vp)
	want := rasterVertex{X: 224, Y: 240, Z: 0.5, Q: 0.5, Diffuse: 0x12345678, U: 0.25, V: 0.75, Fog: 0.5}
	if got != want {
		t.Errorf("centre: %+v, want %+v", got, want)
	}
	got = project(clipVertex{X: -4, Y: 4, Z: 0, W: 4}, vp)
	if got.X != 32 || got.Y != 16 || got.Z != 0 || got.Q != 0.25 {
		t.Errorf("top left: %+v", got)
	}
	got = project(clipVertex{X: 4, Y: -4, Z: 2, W: 4}, Viewport{0, 0, 640, 480, 0.25, 0.75})
	if got.X != 640 || got.Y != 480 || got.Z != 0.5 {
		t.Errorf("bottom right, depth range: %+v", got)
	}
}

// TestProjectMatchesCamera: the background is drawn where the logic thinks
// it is. With an identity world matrix, Draw3D's toClip and project put a
// point exactly where std.Camera.Project (ECL 256's projection) does.
func TestProjectMatchesCamera(t *testing.T) {
	cam := std.Camera{
		Pos:    [3]float32{10, -150, 300},
		Facing: [3]float32{0.3, 400, -200},
		Up:     [3]float32{0, 0, 1},
		FOV:    0.7,
	}
	view, proj := cam.Matrices()
	wv := mul4(Identity, view)
	wvp := mul4(wv, proj)
	vp := Viewport{std.Viewport.X, std.Viewport.Y, std.Viewport.W, std.Viewport.H, 0, 1}
	for _, p := range [][3]float32{{0, 0, 0}, {100, 50, 0}, {-80, 300, -40}, {37.5, 12.25, 99}} {
		r := project(toClip(Vertex3D{X: p[0], Y: p[1], Z: p[2]}, wvp, wv, Fog{}), vp)
		x, y := cam.Project(p)
		if r.X-224 != x || r.Y-16 != y {
			t.Errorf("%v: drawn at (%v, %v), logic has (%v, %v)", p, r.X-224, r.Y-16, x, y)
		}
	}
}

func TestFromTL(t *testing.T) {
	v := TLVertex{X: 1, Y: 2, Z: 0.5, RHW: 0.25, Diffuse: 0xff102030, Specular: 0x80ffffff, U: 0.5, V: 0.125}
	want := rasterVertex{X: 1, Y: 2, Z: 0.5, Q: 0.25, Diffuse: 0xff102030, U: 0.5, V: 0.125, Fog: float32(0x80) / 255}
	if got := fromTL(v, Fog{Mode: FogVertexLinear}); got != want {
		t.Errorf("vertex fog: %+v, want %+v", got, want)
	}
	want.Fog = 1
	if got := fromTL(v, Fog{Mode: FogTableLinear}); got != want {
		t.Errorf("table fog: %+v, want %+v", got, want)
	}
	if got := fromTL(v, Fog{}); got != want {
		t.Errorf("no fog: %+v, want %+v", got, want)
	}
}

func TestTriangles(t *testing.T) {
	for _, tc := range []struct {
		p    Primitive
		n    int
		want [][3]int
	}{
		{TriangleList, 7, [][3]int{{0, 1, 2}, {3, 4, 5}}},
		{TriangleStrip, 5, [][3]int{{0, 1, 2}, {2, 1, 3}, {2, 3, 4}}},
		{TriangleFan, 5, [][3]int{{0, 1, 2}, {0, 2, 3}, {0, 3, 4}}},
		{TriangleStrip, 2, nil},
		{TriangleFan, 0, nil},
	} {
		var got [][3]int
		triangles(tc.p, tc.n, func(a, b, c int) { got = append(got, [3]int{a, b, c}) })
		if len(got) != len(tc.want) {
			t.Errorf("%d/%d: %v, want %v", tc.p, tc.n, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%d/%d: %v, want %v", tc.p, tc.n, got, tc.want)
				break
			}
		}
	}
}

func TestCulled(t *testing.T) {
	a := rasterVertex{X: 0, Y: 0}
	b := rasterVertex{X: 10, Y: 0}
	c := rasterVertex{X: 0, Y: 10} // a, b, c is clockwise on screen
	for _, tc := range []struct {
		mode    Cull
		cw, ccw bool // dropped
	}{
		{CullNone, false, false},
		{CullCW, true, false},
		{CullCCW, false, true},
	} {
		if culled(tc.mode, a, b, c) != tc.cw || culled(tc.mode, a, c, b) != tc.ccw {
			t.Errorf("mode %d: cw dropped %v, ccw dropped %v; want %v, %v", tc.mode,
				culled(tc.mode, a, b, c), culled(tc.mode, a, c, b), tc.cw, tc.ccw)
		}
		if culled(tc.mode, a, b, rasterVertex{X: 20, Y: 0}) {
			t.Errorf("mode %d dropped a zero-area triangle", tc.mode)
		}
	}
}

func TestEbitenVertex(t *testing.T) {
	got := ebitenVertex(rasterVertex{X: 10, Y: 20, Z: 0.375, Q: 0.5, Diffuse: 0x80ff4000, U: 0.25, V: 0.75, Fog: 0.5})
	want := ebiten.Vertex{
		DstX: 10.5, DstY: 20.5, SrcX: 0.375,
		ColorR: 1, ColorG: float32(0x40) / 255, ColorB: 0, ColorA: float32(0x80) / 255,
		Custom0: 0.125, Custom1: 0.375, Custom2: 0.5, Custom3: 0.25,
	}
	if got != want {
		t.Errorf("%+v, want %+v", got, want)
	}
}

// testTexture is 2x2: red, green / blue, transparent white.
func testTexture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	img.SetNRGBA(1, 0, color.NRGBA{0, 255, 0, 255})
	img.SetNRGBA(0, 1, color.NRGBA{0, 0, 255, 255})
	img.SetNRGBA(1, 1, color.NRGBA{255, 255, 255, 0})
	return img
}

func TestTexel(t *testing.T) {
	tex := testTexture()
	red := [4]float32{1, 0, 0, 1}
	green := [4]float32{0, 1, 0, 1}
	redGreen := [4]float32{0.5, 0.5, 0, 1}
	for _, tc := range []struct {
		u, v float32
		a    Address
		f    Filter
		want [4]float32
	}{
		{0.25, 0.25, AddressWrap, FilterPoint, red},
		{0.75, 0.25, AddressWrap, FilterPoint, green},
		{1.25, 0.25, AddressWrap, FilterPoint, red},
		{1.25, 0.25, AddressClamp, FilterPoint, green},
		{-0.25, 0.25, AddressWrap, FilterPoint, green},
		{-0.25, 0.25, AddressClamp, FilterPoint, red},
		{0.25, 0.25, AddressWrap, FilterLinear, red},
		{0.5, 0.25, AddressWrap, FilterLinear, redGreen},
		{0, 0.25, AddressWrap, FilterLinear, redGreen},
		{0, 0.25, AddressClamp, FilterLinear, red},
		// All four texels, straight alpha: the transparent texel's white
		// counts in the colour, as in Direct3D.
		{0.5, 0.5, AddressWrap, FilterLinear, [4]float32{0.5, 0.5, 0.5, 0.75}},
	} {
		if got := texel(tex, tc.u, tc.v, tc.a, tc.f); got != tc.want {
			t.Errorf("texel(%v, %v, addr %d, filter %d) = %v, want %v", tc.u, tc.v, tc.a, tc.f, got, tc.want)
		}
	}
}

func TestShade(t *testing.T) {
	tex := testTexture()
	for _, tc := range []struct {
		name          string
		u             uniforms
		diffuse       [4]float32
		uq, vq, q, fq float32
		want          [4]float32
	}{
		{"untextured", uniforms{}, [4]float32{1, 0.5, 0, 0.5}, 0, 0, 1, 1, [4]float32{0.5, 0.25, 0, 0.5}},
		{"modulate", uniforms{Textured: 1}, [4]float32{0.5, 1, 1, 1}, 0.5, 0.5, 2, 2, [4]float32{0.5, 0, 0, 1}},
		{"vertex fog", uniforms{FogMode: int(FogVertexLinear), FogColor: [3]float32{0, 0, 1}},
			[4]float32{1, 1, 1, 1}, 0, 0, 0.5, 0.25, [4]float32{0.5, 0.5, 1, 1}},
		{"table fog", uniforms{FogMode: int(FogTableLinear), FogColor: [3]float32{0, 0, 1}, FogStart: 2, FogEnd: 6},
			[4]float32{1, 1, 1, 1}, 0, 0, 0.25, 0, [4]float32{0.5, 0.5, 1, 1}},
		{"fog keeps alpha", uniforms{FogMode: int(FogVertexLinear), FogColor: [3]float32{0, 0, 1}},
			[4]float32{1, 1, 1, 0.5}, 0, 0, 1, 0.5, [4]float32{0.25, 0.25, 0.5, 0.5}},
	} {
		if got := shade(tc.u, tex, tc.diffuse, tc.uq, tc.vq, tc.q, tc.fq); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestUniformsOf(t *testing.T) {
	s := state{
		tex: &Texture{}, addr: AddressClamp, filter: FilterLinear,
		fog: Fog{Mode: FogTableLinear, Color: 0xff336699, Start: 10, End: 20}, fogOn: true,
		zOn: true, zFunc: CmpLessEqual, alphaRef: 1,
	}
	want := uniforms{
		Textured: 1, Address: int(AddressClamp), Filter: int(FilterLinear),
		FogMode: int(FogTableLinear), FogColor: [3]float32{float32(0x33) / 255, float32(0x66) / 255, float32(0x99) / 255},
		FogStart: 10, FogEnd: 20, AlphaRef: 1, DepthTest: 1,
	}
	if got := uniformsOf(s); got != want {
		t.Errorf("%+v, want %+v", got, want)
	}
	if got := uniformsOf(state{}); got.Textured != 0 || got.DepthTest != 0 || got.AlphaRef != 0 {
		t.Errorf("zero state: %+v", got)
	}
	noFog := s
	noFog.fogOn = false
	if got := uniformsOf(noFog); got.FogMode != int(FogNone) {
		t.Errorf("FOGENABLE off: FogMode = %d", got.FogMode)
	}
	always := s
	always.zFunc = CmpAlways
	if got := uniformsOf(always); got.DepthTest != 0 {
		t.Errorf("ZFUNC ALWAYS: DepthTest = %d", got.DepthTest)
	}
	zOff := s
	zOff.zOn = false
	if got := uniformsOf(zOff); got.DepthTest != 0 {
		t.Errorf("ZENABLE off: DepthTest = %d", got.DepthTest)
	}
}

func TestStateDepth(t *testing.T) {
	for _, tc := range []struct {
		name        string
		s           state
		test, write bool
	}{
		{"zero", state{}, false, false},
		{"lessequal", state{zOn: true, zFunc: CmpLessEqual}, true, false},
		{"always", state{zOn: true, zFunc: CmpAlways, zWrite: true}, false, true},
		{"lessequal write", state{zOn: true, zFunc: CmpLessEqual, zWrite: true}, true, true},
		{"zenable off", state{zFunc: CmpLessEqual, zWrite: true}, false, false},
	} {
		if got := tc.s.depthTest(); got != tc.test {
			t.Errorf("%s: depthTest = %v", tc.name, got)
		}
		if got := tc.s.depthWrite(); got != tc.write {
			t.Errorf("%s: depthWrite = %v", tc.name, got)
		}
	}
	f := Fog{Mode: FogVertexLinear, Color: 0xffa0a0a0, Start: 1000, End: 5000}
	if got := (state{fog: f}).fogState(); got != (Fog{Color: 0xffa0a0a0, Start: 1000, End: 5000}) {
		t.Errorf("fogState without FOGENABLE: %+v", got)
	}
	if got := (state{fog: f, fogOn: true}).fogState(); got != f {
		t.Errorf("fogState with FOGENABLE: %+v", got)
	}
}

// TestDeviceState covers the setters and Reset, which touch no image.
func TestDeviceState(t *testing.T) {
	var d Device
	d.Reset()
	if want := (state{vp: FullViewport, zOn: true, zWrite: true, zFunc: CmpLessEqual}); d.cur != want || d.cull != CullCCW {
		t.Errorf("Reset: %+v cull %d", d.cur, d.cull)
	}
	if d.world != Identity || d.view != Identity || d.proj != Identity || d.tex0 != Identity {
		t.Errorf("Reset: transforms %v %v %v %v", d.world, d.view, d.proj, d.tex0)
	}
	m := Mat4{{1, 2, 0, 0}, {3, 4, 0, 0}, {5, 6, 1, 0}, {0, 0, 0, 1}}
	d.SetTransform(Texture0, m)
	if d.tex0 != m || d.world != Identity {
		t.Errorf("SetTransform(Texture0): tex0 %v world %v", d.tex0, d.world)
	}
	d.Reset()
	if d.tex0 != Identity {
		t.Errorf("Reset after SetTransform(Texture0): %v", d.tex0)
	}
	d.SetAlphaTest(true, 1)
	if d.cur.alphaRef != 1 {
		t.Errorf("SetAlphaTest(true, 1): ref %d", d.cur.alphaRef)
	}
	d.SetAlphaTest(false, 5)
	if d.cur.alphaRef != 0 {
		t.Errorf("SetAlphaTest(false, 5): ref %d", d.cur.alphaRef)
	}
	d.SetZEnable(false)
	d.SetZWrite(false)
	d.SetZFunc(CmpAlways)
	d.SetFogEnable(true)
	if d.cur.zOn || d.cur.zWrite || d.cur.zFunc != CmpAlways || !d.cur.fogOn {
		t.Errorf("setters: %+v", d.cur)
	}
	defer func() {
		if recover() == nil {
			t.Error("SetZFunc(D3DCMP_LESS) did not panic")
		}
	}()
	d.SetZFunc(2)
}

func TestAlpha8(t *testing.T) {
	for _, tc := range []struct {
		a    float32
		want int
	}{
		{0, 0}, {1, 255}, {0.6 / 255, 1}, {0.4 / 255, 0}, {128.0 / 255, 128}, {-0.25, 0}, {1.5, 255},
	} {
		if got := alpha8(tc.a); got != tc.want {
			t.Errorf("alpha8(%v) = %d, want %d", tc.a, got, tc.want)
		}
	}
}

func TestDepth16(t *testing.T) {
	for _, tc := range []struct {
		z    float32
		want uint16
	}{
		{0, 0}, {1, 65535}, {0.25, 16384}, {0.75, 49151}, {100.4 / 65535, 100}, {100.6 / 65535, 101}, {-1, 0}, {2, 65535},
	} {
		if got := depth16(tc.z); got != tc.want {
			t.Errorf("depth16(%v) = %d, want %d", tc.z, got, tc.want)
		}
	}
}

func TestDepthColor(t *testing.T) {
	if got, want := depthColor(0x1234), [4]float32{float32(0x12) / 255, float32(0x34) / 255, 0, 1}; got != want {
		t.Errorf("depthColor(0x1234) = %v, want %v", got, want)
	}
	// The shader's decoding gives every value back.
	for _, d := range []uint16{0, 1, 255, 256, 0x1234, 0xfffe, 0xffff} {
		c := depthColor(d)
		r := math.Floor(float64(float32(c[0]*255) + 0.5))
		g := math.Floor(float64(float32(c[1]*255) + 0.5))
		if got := uint16(r*256 + g); got != d {
			t.Errorf("decode(depthColor(%#x)) = %#x", d, got)
		}
	}
}

func TestFragment(t *testing.T) {
	tiny := float32(1) / 255
	in := func(diffuse [4]float32, z float32, stored uint16) fragIn {
		return fragIn{Diffuse: diffuse, Q: 1, FQ: 1, Z: z, Stored: stored}
	}
	half := [4]float32{1, 0.5, 0, 0.5}
	clear := [4]float32{1, 1, 1, 0}
	white := [4]float32{1, 1, 1, 1}
	for _, tc := range []struct {
		name string
		u    uniforms
		in   fragIn
		want [4]float32
		keep bool
	}{
		{"no tests", uniforms{}, in(half, 0.5, 0), [4]float32{0.5, 0.25, 0, 0.5}, true},
		{"alpha test off keeps alpha 0", uniforms{}, in(clear, 0, 0), [4]float32{}, true},
		{"alpha 0 fails ref 1", uniforms{AlphaRef: 1}, in(clear, 0, 0), [4]float32{}, false},
		{"alpha 1/255 passes ref 1", uniforms{AlphaRef: 1}, in([4]float32{1, 1, 1, tiny}, 0, 0), [4]float32{tiny, tiny, tiny, tiny}, true},
		{"equal depth passes", uniforms{DepthTest: 1}, in(white, 100.0/65535, 100), white, true},
		{"farther depth fails", uniforms{DepthTest: 1}, in(white, 100.0/65535, 99), [4]float32{}, false},
		{"no depth test", uniforms{}, in(white, 100.0/65535, 99), white, true},
		{"depth pass", uniforms{Pass: 1}, in(half, float32(0x1234)/65535, 0), depthColor(0x1234), true},
		{"depth pass alpha fail", uniforms{Pass: 1, AlphaRef: 1}, in(clear, 0.5, 0), [4]float32{}, false},
		{"depth pass depth fail", uniforms{Pass: 1, DepthTest: 1}, in(white, 200.0/65535, 100), [4]float32{}, false},
	} {
		got, keep := fragment(tc.u, nil, tc.in)
		if got != tc.want || keep != tc.keep {
			t.Errorf("%s: %v %v, want %v %v", tc.name, got, keep, tc.want, tc.keep)
		}
	}
}

func TestConflicts(t *testing.T) {
	a := screenTri{{0, 0}, {10, 0}, {0, 10}}
	aCW := screenTri{{0, 0}, {0, 10}, {10, 0}}
	for _, tc := range []struct {
		name string
		a, b screenTri
		want bool
	}{
		{"same", a, a, true},
		{"quad halves", a, screenTri{{10, 0}, {10, 10}, {0, 10}}, false},
		{"shared edge, below", a, screenTri{{0, 0}, {5, -5}, {10, 0}}, false},
		{"far apart", a, screenTri{{20, 0}, {30, 0}, {20, 10}}, false},
		{"touch at a vertex", a, screenTri{{10, 0}, {20, 0}, {20, 10}}, false},
		{"inside", a, screenTri{{1, 1}, {3, 1}, {1, 3}}, true},
		{"inside, other winding", aCW, screenTri{{1, 1}, {3, 1}, {1, 3}}, true},
		{"star", screenTri{{0, 0}, {10, 0}, {5, 10}}, screenTri{{0, 7}, {10, 7}, {5, -3}}, true},
		{"crossing, no vertex inside", a, screenTri{{2, -1}, {3, -1}, {2.5, 20}}, true},
		{"boxes overlap, hypotenuse separates", a, screenTri{{6, 6}, {10, 6}, {6, 10}}, false},
		{"hypotenuse separates, other winding", aCW, screenTri{{6, 6}, {6, 10}, {10, 6}}, false},
		{"zero area", a, screenTri{{0, 0}, {5, 5}, {10, 10}}, false},
	} {
		if got := conflicts(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: conflicts(a, b) = %v, want %v", tc.name, got, tc.want)
		}
		if got := conflicts(tc.b, tc.a); got != tc.want {
			t.Errorf("%s: conflicts(b, a) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestEbitenBlend(t *testing.T) {
	if BlendAlpha.ebitenBlend() != ebiten.BlendSourceOver {
		t.Error("BlendAlpha is not BlendSourceOver")
	}
	if BlendAdd.ebitenBlend() != ebiten.BlendLighter {
		t.Error("BlendAdd is not BlendLighter")
	}
}

func TestRotation(t *testing.T) {
	a := float32(0.5)
	c, s := float32(math.Cos(0.5)), float32(math.Sin(0.5))
	for _, tc := range []struct {
		name string
		got  Mat4
		want Mat4
	}{
		{"X", RotationX(a), Mat4{{1, 0, 0, 0}, {0, c, s, 0}, {0, -s, c, 0}, {0, 0, 0, 1}}},
		{"Y", RotationY(a), Mat4{{c, 0, -s, 0}, {0, 1, 0, 0}, {s, 0, c, 0}, {0, 0, 0, 1}}},
		{"Z", RotationZ(a), Mat4{{c, s, 0, 0}, {-s, c, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}}},
	} {
		if tc.got != tc.want {
			t.Errorf("Rotation%s(0.5) = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	if RotationZ(0) != Identity {
		t.Errorf("RotationZ(0) = %v", RotationZ(0))
	}
}

func TestTranslation(t *testing.T) {
	want := Mat4{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {1.5, -2, 3, 1}}
	if got := Translation(1.5, -2, 3); got != want {
		t.Errorf("Translation(1.5, -2, 3) = %v, want %v", got, want)
	}
}

// TestVec3Project checks Vec3Project's formula on values exact in float32.
func TestVec3Project(t *testing.T) {
	vp := Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1}
	persp := Identity // w = z
	persp[2][3], persp[3][3] = 1, 0
	for _, tc := range []struct {
		name              string
		v                 [3]float32
		vp                Viewport
		proj, view, world Mat4
		want              [3]float32
	}{
		{"identity", [3]float32{0.5, -0.5, 0.25}, vp, Identity, Identity, Identity, [3]float32{320, 352, 0.25}},
		{"world", [3]float32{0, 0, 0}, vp, Identity, Identity, Translation(1, 2, 3), [3]float32{416, -208, 3}},
		{"view", [3]float32{0, 0, 0}, vp, Identity, Translation(1, 2, 3), Identity, [3]float32{416, -208, 3}},
		{"divide", [3]float32{2, 4, 4}, vp, persp, Identity, Identity, [3]float32{320, 16, 1}},
		{"depth range", [3]float32{0, 0, 0.5}, Viewport{X: 0, Y: 0, W: 640, H: 480, MinZ: 0.25, MaxZ: 0.75},
			Identity, Identity, Identity, [3]float32{320, 240, 0.5}},
	} {
		if got := Vec3Project(tc.v, tc.vp, tc.proj, tc.view, tc.world); got != tc.want {
			t.Errorf("%s: Vec3Project(%v) = %v, want %v", tc.name, tc.v, got, tc.want)
		}
	}
}
