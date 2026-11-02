package stddraw

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/gfx"
	"th10/std"
	"th10/supervisor"
	"th10/supervisor/supervisortest"
)

var texA, texB = &gfx.Texture{W: 64, H: 64}, &gfx.Texture{W: 32, H: 32}

// fixture is a new supervisor over a recorder after BeginFrame (fog cache
// off, Screen in use), with the recorder emptied. The ANM file af has
// sprite 0, 16x16 on texA, and sprite 1, 32x8 on texB.
func fixture(t *testing.T) (*supervisor.Supervisor, *supervisortest.Rec, *anm.File) {
	t.Helper()
	af := &anm.File{}
	d := &supervisortest.Rec{Names: map[*gfx.Texture]string{texA: "A", texB: "B"}}
	sv := supervisor.New(d, nil)
	sv.NewSheet = func(g *anm.File) *anmdraw.Sheet {
		if g != af {
			t.Fatalf("NewSheet called with another file")
		}
		return &anmdraw.Sheet{Sprites: []anmdraw.Sprite{
			{Tex: texA, U1: 1, V1: 1, W: 16, H: 16},
			{Tex: texB, U1: 1, V1: 1, W: 32, H: 8},
		}}
	}
	sv.BeginFrame()
	d.Take()
	return sv, d, af
}

// newSTD is f's VM with its quad VMs made (idle: no script, no sprite) and
// the fog colour 0xff102030 from 100 to 900.
func newSTD(f *std.File) *std.VM {
	v := std.New(f)
	v.Enable(&anm.File{}, nil)
	v.Cam.FogColor = 0xff102030
	v.Cam.FogNear = 100
	v.Cam.FogFar = 900
	return v
}

// drawable makes vm a visible object of af in render mode mode with the
// given sprite, inside the field, Offset (7, 7, 7) and Mat the identity.
func drawable(vm *anm.VM, af *anm.File, sprite int, mode uint32) {
	*vm = *anm.New()
	vm.File = af
	vm.Sprite = sprite
	vm.Flags = 3 | mode<<22
	vm.Pos = [3]float32{200, 200, 0}
	vm.Offset = [3]float32{7, 7, 7}
	vm.Mat = gfx.Identity
}

var (
	fieldView = []string{"transform 2", "transform 3", "viewport 32 16 384 448 0 1"}
	fogLine   = fmt.Sprintf("fog %d ff102030 100 900 false", gfx.FogVertexLinear)
	clears    = []string{"clear 2 00000000 1 []", "clear 1 00102030 1 [(32,16)-(416,464)]"}
)

func cat(parts ...[]string) []string { return slices.Concat(parts...) }

func rep(s []string, n int) []string { return slices.Repeat(s, n) }

// keep drops the ANM manager's own state calls (texture, blend, filter).
func keep(calls []string) []string {
	var out []string
	for _, c := range calls {
		if !strings.HasPrefix(c, "tex ") && !strings.HasPrefix(c, "blend ") && !strings.HasPrefix(c, "filter ") {
			out = append(out, c)
		}
	}
	return out
}

// only keeps the calls starting with prefix.
func only(calls []string, prefix string) []string {
	var out []string
	for _, c := range calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func eqCalls(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n got  %s\n want %s", what, strings.Join(got, "; "), strings.Join(want, "; "))
	}
}

// node7 is Node7's device calls with no faces, around slot, the calls of
// the slot step.
func node7(slot []string) []string {
	return cat(fieldView,
		[]string{"zwrite true", "zfunc 4", fogLine},
		clears,
		slot,
		[]string{"fogenable true"},
		rep(fieldView, 8),
		[]string{"zwrite false", "zfunc 8"})
}

func TestNode7(t *testing.T) {
	sv, d, _ := fixture(t)
	v := newSTD(&std.File{})
	v.Cam.Pos = [3]float32{1, 2, -600}
	v.Cam.Up = [3]float32{0.25, 1, 0}
	v.Cam.FOV = 0.75
	v.Cam.Delta = [3]float32{0.5, -1, 2}
	v.Cam.FogBytes = [4]float32{0x30, 0x20, 0x10, 0xff}
	dr := &Drawer{STD: v, Faces: 5, Culled: 6, Quads: 7}
	sv.Field.Shift = [2]float32{3, -2}
	sv.Field.Eye = [3]float32{1, 2, 3}
	sv.ANM.TintOn, sv.ANM.Tint = true, 0x12345678

	dr.Node7(sv)

	eqCalls(t, "Node7", d.Take(), node7(nil))
	if dr.Faces != 0 || dr.Culled != 0 || dr.Quads != 0 {
		t.Errorf("counters %d %d %d, want zero", dr.Faces, dr.Culled, dr.Quads)
	}
	f := &sv.Field
	if f.Pos != v.Cam.Pos || f.Facing != v.Cam.Facing || f.Up != v.Cam.Up || f.FOV != v.Cam.FOV {
		t.Errorf("Field camera %v %v %v %v, want STD's %v %v %v %v",
			f.Pos, f.Facing, f.Up, f.FOV, v.Cam.Pos, v.Cam.Facing, v.Cam.Up, v.Cam.FOV)
	}
	if f.Eye != ([3]float32{}) {
		t.Errorf("Field.Eye = %v, want zero", f.Eye)
	}
	if f.Delta != v.Cam.Delta || f.FogNear != 100 || f.FogFar != 900 || f.FogBytes != v.Cam.FogBytes || f.FogColor != 0xff102030 {
		t.Errorf("Field Delta, fog %v %v %v %v %08x, want STD's", f.Delta, f.FogNear, f.FogFar, f.FogBytes, f.FogColor)
	}
	if eye, near, far, bgr := sv.FieldFog(); eye != v.Cam.Pos || near != 100 || far != 900 || bgr != [3]float32{0x30, 0x20, 0x10} {
		t.Errorf("FieldFog() = %v %v %v %v", eye, near, far, bgr)
	}
	if right, view, proj, vp := sv.Selected(); right != f.Right || view != f.View || proj != f.Proj || vp != f.Viewport {
		t.Errorf("Selected() is not Field's block")
	}
	if f.Shift != [2]float32{3, -2} || sv.ANM.Shift != [2]float32{3, -2} {
		t.Errorf("Field.Shift %v, ANM.Shift %v, want both (3, -2)", f.Shift, sv.ANM.Shift)
	}
	if f.Viewport != (gfx.Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1}) {
		t.Errorf("Field.Viewport = %v", f.Viewport)
	}
	if sv.Cur != f || sv.Is2D {
		t.Errorf("Cur is Field %v, Is2D %v; want true, false", sv.Cur == f, sv.Is2D)
	}
	view, proj := std.LookMatrices(v.Cam.Pos, v.Cam.Facing, v.Cam.Up, [3]float32{}, v.Cam.FOV, 384, 448)
	if d.View != view || d.Proj != proj {
		t.Errorf("view, proj are not STD's camera's")
	}
	if sv.ANM.TintOn || sv.ANM.Tint != 0x80808080 {
		t.Errorf("tint %v %08x, want false 80808080", sv.ANM.TintOn, sv.ANM.Tint)
	}
}

func TestNode0A(t *testing.T) {
	sv, d, _ := fixture(t)
	dr := &Drawer{STD: newSTD(&std.File{}), Faces: 5, Culled: 6, Quads: 7}
	layer := func(l int) { d.Calls = append(d.Calls, fmt.Sprintf("layer %d", l)) }

	dr.Node0A(sv, layer)

	want := cat(fieldView,
		[]string{"fogenable true", "zwrite false", "zfunc 8", "layer 17", "zfunc 4",
			"fogenable false", "layer 18", "zfunc 4", fogLine, "zwrite false", "fogenable true"},
		rep(fieldView, 4),
		[]string{"zwrite false", "zfunc 8", "zwrite false", "zfunc 8", "fogenable false"})
	eqCalls(t, "Node0A", d.Take(), want)
	if dr.Faces != 5 || dr.Culled != 6 || dr.Quads != 7 {
		t.Errorf("counters %d %d %d, want untouched 5 6 7", dr.Faces, dr.Culled, dr.Quads)
	}
	if sv.Cur != &sv.Field || sv.Is2D {
		t.Errorf("Cur is Field %v, Is2D %v; want true, false", sv.Cur == &sv.Field, sv.Is2D)
	}
}

func TestTint(t *testing.T) {
	for _, node := range []string{"Node7", "Node0A"} {
		sv, _, _ := fixture(t)
		dr := &Drawer{STD: newSTD(&std.File{}), Dim: true}
		if node == "Node7" {
			dr.Node7(sv)
		} else {
			dr.Node0A(sv, func(int) {})
		}
		if !sv.ANM.TintOn || sv.ANM.Tint != 0xff404040 {
			t.Errorf("%s with Dim: tint %v %08x, want true ff404040", node, sv.ANM.TintOn, sv.ANM.Tint)
		}
	}
}

// TestNode7Slots: slot 0 has a sprite, so the slots with sprites are drawn
// under Field's flat matrices with depth writes off, before the passes.
func TestNode7Slots(t *testing.T) {
	sv, d, af := fixture(t)
	v := newSTD(&std.File{})
	drawable(&v.Slots[0], af, 0, 0)
	drawable(&v.Slots[2], af, 0, 0)
	v.Slots[2].Pos = [3]float32{100, 100, 0}

	// The draw calls of the two slots drawn on their own under Field.
	ref, rd, raf := fixture(t)
	ref.Use(&ref.Field)
	rd.Take()
	for _, i := range []int{0, 2} {
		s := v.Slots[i]
		s.File = raf
		ref.ANM.Draw(&s)
	}
	ref.ANM.Flush()
	slotDraws := keep(rd.Take())
	if len(slotDraws) == 0 {
		t.Fatal("reference slots drew nothing")
	}

	(&Drawer{STD: v}).Node7(sv)

	slot := cat(fieldView, []string{"zwrite false"}, slotDraws, []string{"zwrite true"}, fieldView)
	eqCalls(t, "Node7 with slots", keep(d.Take()), node7(slot))
}

// TestNode7SlotZero: only slot 0's sprite is tested.
func TestNode7SlotZero(t *testing.T) {
	sv, d, af := fixture(t)
	v := newSTD(&std.File{})
	drawable(&v.Slots[3], af, 0, 0)

	(&Drawer{STD: v}).Node7(sv)

	eqCalls(t, "Node7 with slot 3 only", keep(d.Take()), node7(nil))
}

// TestPasses runs both nodes over a file with:
//
//   - entry 0, pass 0, in view: quads of type 0 (mode 8), 1 (mode 8),
//     0 (mode 0), 0 (mode 8), 0 (mode 8, width NaN);
//   - entry 1, pass 0, 5600 from the camera (culled);
//   - entry 2, pass 9, in view, one mode-0 quad;
//   - entry 3, pass -1 (byte 0xff), never drawn;
//
// placed by faces 0 (entry 0), 1 (entry 1), 2 (entry 0 at x 10), 3 (entry
// 2) and 4 (entry 3), under STD's default camera.
func TestPasses(t *testing.T) {
	sv, d, af := fixture(t)
	nan := float32(math.NaN())
	box := [3]float32{100, 100, 100}
	at := [3]float32{0, 300, 0}
	f := &std.File{
		Entries: []std.Entry{
			{Unknown: 0x0100, Pos: at, Size: box, Quads: []std.Quad{
				{Type: 0, Pos: [3]float32{1, 2, 3}, Size: [2]float32{32, 0}},
				{Type: 1, Pos: [3]float32{9, 9, 9}, Size: [2]float32{32, 32}},
				{Type: 0, Pos: [3]float32{4, 5, 6}, Size: [2]float32{64, 64}},
				{Type: 0, Size: [2]float32{0, 16}},
				{Type: 0, Size: [2]float32{nan, 0}},
			}},
			{Unknown: 0x0100, Pos: [3]float32{0, 0, 5000}, Size: [3]float32{10, 10, 10}, Quads: []std.Quad{{}}},
			{Unknown: 0x0109, Pos: at, Size: box, Quads: []std.Quad{{}}},
			{Unknown: 0x01ff, Pos: at, Size: box, Quads: []std.Quad{{}}},
		},
		Faces: []std.Face{{Entry: 0}, {Entry: 1}, {Entry: 0, Pos: [3]float32{10, 0, 0}}, {Entry: 2}, {Entry: 3}},
	}
	v := newSTD(f)
	q := v.Quad
	drawable(q(0, 0), af, 0, 8)
	drawable(q(0, 1), af, 0, 8)
	drawable(q(0, 2), af, 1, 0)
	drawable(q(0, 3), af, 1, 8)
	drawable(q(0, 4), af, 0, 8)
	drawable(q(1, 0), af, 0, 0)
	drawable(q(2, 0), af, 0, 0)
	drawable(q(3, 0), af, 0, 0)
	dr := &Drawer{STD: v}

	dr.Node7(sv)
	calls := d.Take()

	// Fog on before the passes, then on for mode 8 and off for mode 0.
	eqCalls(t, "Node7 fog", only(calls, "fogenable"), []string{
		"fogenable true", "fogenable false", "fogenable true", "fogenable false", "fogenable true"})
	// The three drawn mode-8 quads of entry 0, for each of its two faces.
	if n := len(only(calls, "draw3d ")); n != 6 {
		t.Errorf("Node7 drew %d mode-8 quads, want 6", n)
	}
	if dr.Faces != 2 || dr.Culled != 1 || dr.Quads != 8 {
		t.Errorf("after Node7: counters %d %d %d, want 2 1 8", dr.Faces, dr.Culled, dr.Quads)
	}

	// Mode 8: Offset is quad + face position (face 2 drew last), Scale the
	// quad size over the sprite's where nonzero, and the scale-dirty bit
	// makes draw8 rebuild Mat with it.
	for _, tc := range []struct {
		name   string
		vm     *anm.VM
		offset [3]float32
		scale  [2]float32
	}{
		{"q0", q(0, 0), [3]float32{11, 2, 3}, [2]float32{2, 1}},
		{"q1 (type 1)", q(0, 1), [3]float32{7, 7, 7}, [2]float32{1, 1}},
		{"q2 (mode 0)", q(0, 2), [3]float32{7, 7, 7}, [2]float32{1, 1}},
		{"q3", q(0, 3), [3]float32{10, 0, 0}, [2]float32{1, 2}},
		{"entry 1 (culled)", q(1, 0), [3]float32{7, 7, 7}, [2]float32{1, 1}},
	} {
		if tc.vm.Offset != tc.offset || tc.vm.Scale != tc.scale {
			t.Errorf("%s: Offset %v Scale %v, want %v %v", tc.name, tc.vm.Offset, tc.vm.Scale, tc.offset, tc.scale)
		}
	}
	if s := q(0, 4).Scale; !math.IsNaN(float64(s[0])) || s[1] != 1 {
		t.Errorf("q4: Scale %v, want (NaN, 1)", s)
	}
	if m := q(0, 0).Mat; m[0][0] != 0.125 || m[1][1] != 0.0625 {
		t.Errorf("q0: Mat diagonal %v %v, want 0.125 0.0625", m[0][0], m[1][1])
	}
	if m := q(0, 3).Mat; m[0][0] != 0.125 || m[1][1] != 0.0625 {
		t.Errorf("q3: Mat diagonal %v %v, want 0.125 0.0625", m[0][0], m[1][1])
	}

	// Node0A: fog is already on, so the first change is the one after
	// layer 17; pass 9 draws entry 2's mode-0 quad. The counters carry on.
	dr.Node0A(sv, func(int) {})
	eqCalls(t, "Node0A fog", only(d.Take(), "fogenable"), []string{
		"fogenable false", "fogenable true", "fogenable false"})
	if dr.Faces != 3 || dr.Culled != 1 || dr.Quads != 9 {
		t.Errorf("after Node0A: counters %d %d %d, want 3 1 9", dr.Faces, dr.Culled, dr.Quads)
	}

	dr.Node7(sv)
	if dr.Faces != 2 || dr.Culled != 1 || dr.Quads != 8 {
		t.Errorf("after the second Node7: counters %d %d %d, want 2 1 8", dr.Faces, dr.Culled, dr.Quads)
	}
}

// identityCam sees x and y in -1..1 and z in 0..1 (view and projection
// the identity) on the field's viewport, from the origin.
func identityCam() *supervisor.Camera {
	return &supervisor.Camera{
		View:     gfx.Identity,
		Proj:     gfx.Identity,
		Viewport: gfx.Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1},
	}
}

func TestCulled(t *testing.T) {
	nan := float32(math.NaN())
	half := [3]float32{1, 1, 0.5}
	unit := [3]float32{1, 1, 1}
	// skew makes projected z = 4x + 2y + z + c, so that only the first
	// corners land in 0..1.
	skew := func(c float32) func(*supervisor.Camera) {
		return func(cam *supervisor.Camera) {
			cam.Proj[0][2], cam.Proj[1][2], cam.Proj[2][2], cam.Proj[3][2] = 4, 2, 1, c
		}
	}
	for _, tc := range []struct {
		name      string
		pos, size [3]float32
		at        [3]float32
		cam       func(*supervisor.Camera)
		dist2     float32
		want      bool
	}{
		{"in view", [3]float32{0, 0, 0.5}, half, [3]float32{}, nil, cullDist2, false},
		{"past z 1", [3]float32{0, 0, 5}, unit, [3]float32{}, nil, cullDist2, true},
		{"z 1 counts", [3]float32{0, 0, 1.5}, unit, [3]float32{}, nil, cullDist2, false},
		{"just past z 1", [3]float32{0, 0, 1.5}, [3]float32{1, 1, 0.9921875}, [3]float32{}, nil, cullDist2, true},
		{"z 0 counts", [3]float32{0, 0, -0.5}, unit, [3]float32{}, nil, cullDist2, false},
		{"face position moves it out", [3]float32{}, half, [3]float32{0, 0, 5}, nil, cullDist2, true},
		{"face position moves it in", [3]float32{}, half, [3]float32{0, 0, 0.5}, nil, cullDist2, false},
		{"minX 416", [3]float32{1.5, 0, 0.5}, half, [3]float32{}, nil, cullDist2, false},
		{"minX 416.75", [3]float32{1.50390625, 0, 0.5}, half, [3]float32{}, nil, cullDist2, true},
		{"maxX 32", [3]float32{-1.5, 0, 0.5}, half, [3]float32{}, nil, cullDist2, false},
		{"maxY 16", [3]float32{0, 1.5, 0.5}, half, [3]float32{}, nil, cullDist2, false},
		{"minY 464", [3]float32{0, -1.5, 0.5}, half, [3]float32{}, nil, cullDist2, false},
		{"lone corner (else-if)", [3]float32{}, unit, [3]float32{}, skew(-2.75), cullDist2, true},
		{"two corners at z 1 and 0", [3]float32{}, unit, [3]float32{}, skew(-2.5), cullDist2, false},
		{"too far", [3]float32{0, 0, 0.5}, half, [3]float32{},
			func(c *supervisor.Camera) { c.Pos = [3]float32{0, 0, -1} }, 1, true},
		{"exactly at distance", [3]float32{0, 0, 0.5}, half, [3]float32{},
			func(c *supervisor.Camera) { c.Eye = [3]float32{0, 0, -0.5} }, 1, false},
		{"eye plus position", [3]float32{0, 0, 0.5}, half, [3]float32{},
			func(c *supervisor.Camera) { c.Eye, c.Pos = [3]float32{0, 0, -0.25}, [3]float32{0, 0, -0.25} }, 1, false},
		{"NaN position", [3]float32{nan, 0, 0.5}, half, [3]float32{}, nil, cullDist2, true},
		{"NaN distance", [3]float32{0, 0, 0.5}, half, [3]float32{},
			func(c *supervisor.Camera) { c.Pos = [3]float32{nan, 0, 0} }, cullDist2, false},
	} {
		cam := identityCam()
		if tc.cam != nil {
			tc.cam(cam)
		}
		e := &std.Entry{Pos: tc.pos, Size: tc.size}
		if got := culled(e, tc.at, cam, tc.dist2); got != tc.want {
			t.Errorf("%s: culled = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestVec3ProjectMatchesStd: with the identity world, Vec3Project is
// std.Camera.Project before its move to ECL space, bit for bit.
func TestVec3ProjectMatchesStd(t *testing.T) {
	c := std.New(&std.File{}).Cam
	view, proj := c.Matrices()
	vp := gfx.Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1}
	for _, p := range [][3]float32{{0, 300, 0}, {50, 250, 100}, {-120, 400, -50}} {
		x, y := c.Project(p)
		q := gfx.Vec3Project(p, vp, proj, view, gfx.Identity)
		if q[0]-224 != x || q[1]-16 != y {
			t.Errorf("%v: Vec3Project %v %v, Project %v %v", p, q[0]-224, q[1]-16, x, y)
		}
	}
}
