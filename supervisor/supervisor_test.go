package supervisor

import (
	"math"
	"slices"
	"strings"
	"testing"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/gfx"
	"th10/std"
	"th10/supervisor/supervisortest"
)

var _ Device = (*supervisortest.Rec)(nil)

var texA, texB = &gfx.Texture{W: 64, H: 64}, &gfx.Texture{W: 32, H: 32}

// fixture is a new supervisor over a recorder, drawing from one file f
// whose sprites 0 and 1 are on texA and texB. The recorder still holds the
// calls New made.
func fixture(t *testing.T) (*Supervisor, *supervisortest.Rec, *anm.File) {
	t.Helper()
	f := &anm.File{}
	d := &supervisortest.Rec{Names: map[*gfx.Texture]string{texA: "A", texB: "B"}}
	s := New(d, nil)
	built := 0
	s.NewSheet = func(g *anm.File) *anmdraw.Sheet {
		if g != f {
			t.Fatalf("NewSheet called with another file")
		}
		built++
		if built > 1 {
			t.Fatalf("NewSheet called twice for one file")
		}
		return &anmdraw.Sheet{Sprites: []anmdraw.Sprite{
			{Tex: texA, U1: 1, V1: 1, W: 16, H: 16},
			{Tex: texB, U1: 1, V1: 1, W: 16, H: 16},
		}}
	}
	return s, d, f
}

// obj is a drawable mode-0 object of f with the given sprite and blend,
// inside both cameras' viewports.
func obj(f *anm.File, sprite int, blend uint32) *anm.VM {
	vm := anm.New()
	vm.File = f
	vm.Sprite = sprite
	vm.Flags = 3 | blend<<4
	vm.Pos = [3]float32{200, 200, 0}
	return vm
}

func eqCalls(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n got  %s\n want %s", what, strings.Join(got, "; "), strings.Join(want, "; "))
	}
}

// draws keeps the draw and tex calls of calls.
func draws(calls []string) []string {
	var out []string
	for _, c := range calls {
		if strings.HasPrefix(c, "draw ") || strings.HasPrefix(c, "tex ") {
			out = append(out, c)
		}
	}
	return out
}

var (
	screenView = []string{"transform 2", "transform 3", "viewport 0 0 640 480 0 1"}
	fieldView  = []string{"transform 2", "transform 3", "viewport 32 16 384 448 0 1"}
)

func cat(parts ...[]string) []string { return slices.Concat(parts...) }

func TestNew(t *testing.T) {
	s, d, f := fixture(t)
	eqCalls(t, "New (FUN_00439d20)", d.Take(), []string{
		"zenable true",
		"cull 1",
		"blend 0",
		"zfunc 8",
		"alphatest true 1",
		"fogenable true",
		"fog 1 ffa0a0a0 1000 5000 false",
		"address 0",
		"filter 1",
	})
	// FUN_004216f0.
	init := Camera{Pos: [3]float32{0, 0, 1000}, Up: [3]float32{0, 1, 0}, FOV: math.Float32frombits(0x3f060a92)}
	scr, fld := init, init
	scr.Viewport = gfx.Viewport{X: 0, Y: 0, W: 640, H: 480, MinZ: 0, MaxZ: 1}
	fld.Viewport = gfx.Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1}
	if s.Screen != scr {
		t.Errorf("Screen %+v, want %+v", s.Screen, scr)
	}
	if s.Field != fld {
		t.Errorf("Field %+v, want %+v", s.Field, fld)
	}
	if s.Cur != nil || s.Is2D || s.ClearColor != 0 {
		t.Errorf("Cur %p, Is2D %v, ClearColor %08x; want nil, false, 0", s.Cur, s.Is2D, s.ClearColor)
	}

	// Init ends with ANM.Invalidate: blend cache 3, so a blend-0 object sets
	// the blend again although the device already has it.
	s.BeginFrame()
	d.Take()
	s.ANM.Draw(obj(f, 0, 0))
	s.ANM.Flush()
	eqCalls(t, "first object after New", d.Take(), []string{"tex A", "blend 0", "draw 4 6"})
}

func TestBeginFrame(t *testing.T) {
	s, d, f := fixture(t)
	s.Screen.Shift = [2]float32{1.5, -2}
	s.Field.Shift = [2]float32{7, 7}
	d.Take()

	// A pending batch is submitted before anything changes (FUN_00442f50,
	// and FUN_00421480's own flush). The object needs a cull rectangle,
	// which New leaves zero: the original never draws before Start.
	s.Select(&s.Field)
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()
	s.BeginFrame()
	eqCalls(t, "BeginFrame", d.Take(), cat([]string{"draw 4 6"}, screenView, []string{"fogenable false"}))

	view, proj := std.FlatMatrices(640, 480)
	if d.View != view || d.Proj != proj {
		t.Errorf("matrices are not FlatMatrices(640, 480)")
	}
	if s.Screen.View != view || s.Screen.Proj != proj {
		t.Errorf("Screen's matrices not stored in the block")
	}
	if s.Cur != &s.Screen || !s.Is2D {
		t.Errorf("Cur %p (Screen %p), Is2D %v", s.Cur, &s.Screen, s.Is2D)
	}
	if s.ANM.View != [4]uint32{0, 0, 640, 480} {
		t.Errorf("ANM.View %v", s.ANM.View)
	}
	if s.ANM.Shift != s.Screen.Shift {
		t.Errorf("ANM.Shift %v, want Screen's %v", s.ANM.Shift, s.Screen.Shift)
	}

	// The fog cache is reset to unknown every frame, so FOGENABLE 0 is set
	// even when the last frame left it off.
	s.BeginFrame()
	eqCalls(t, "second BeginFrame", d.Take(), cat(screenView, []string{"fogenable false"}))
}

func TestFog(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	d.Take()

	s.Fog(false)
	eqCalls(t, "Fog(false) while off", d.Take(), nil)
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()
	s.Fog(true)
	eqCalls(t, "Fog(true) with a pending quad", d.Take(), []string{"draw 4 6", "fogenable true"})
	s.Fog(true)
	eqCalls(t, "Fog(true) while on", d.Take(), nil)
	s.Fog(false)
	eqCalls(t, "Fog(false) while on", d.Take(), []string{"fogenable false"})
}

func TestEnterField(t *testing.T) {
	s, d, f := fixture(t)
	s.Field.Shift = [2]float32{3, 4}
	s.BeginFrame()
	s.Fog(true)
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()

	s.EnterField()
	eqCalls(t, "EnterField with fog on", d.Take(), cat([]string{"draw 4 6"}, fieldView, []string{"fogenable false"}))
	view, proj := std.FlatMatrices(384, 448)
	if d.View != view || d.Proj != proj {
		t.Errorf("matrices are not FlatMatrices(384, 448)")
	}
	if s.Cur != &s.Field || s.Is2D {
		t.Errorf("Cur %p (Field %p), Is2D %v", s.Cur, &s.Field, s.Is2D)
	}
	if s.ANM.View != [4]uint32{32, 16, 384, 448} {
		t.Errorf("ANM.View %v", s.ANM.View)
	}
	if s.ANM.Shift != [2]float32{3, 4} {
		t.Errorf("ANM.Shift %v, want Field's", s.ANM.Shift)
	}

	s.EnterField()
	eqCalls(t, "EnterField with fog off", d.Take(), fieldView)
}

func TestEnter2D(t *testing.T) {
	s, d, _ := fixture(t)
	s.BeginFrame()
	s.EnterField()
	s.Fog(true)
	d.Take()

	// The 2D nodes leave the fog alone.
	s.Enter2D()
	eqCalls(t, "Enter2D", d.Take(), screenView)
	if s.Cur != &s.Screen || !s.Is2D {
		t.Errorf("Cur %p (Screen %p), Is2D %v", s.Cur, &s.Screen, s.Is2D)
	}
	if s.ANM.View != [4]uint32{0, 0, 640, 480} {
		t.Errorf("ANM.View %v", s.ANM.View)
	}
}

func TestPerspective(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	c := &s.Field
	c.Pos = [3]float32{0, -50, 400}
	c.Facing = [3]float32{0, 300, -400}
	c.Up = [3]float32{0, 0, 1}
	c.Eye = [3]float32{0, 0, 10}
	c.FOV = 0.9
	c.Shift = [2]float32{-1, 0.5}
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()

	s.Perspective(c)
	eqCalls(t, "Perspective", d.Take(), []string{"draw 4 6", "transform 2", "transform 3"})
	view, proj := std.LookMatrices(c.Pos, c.Facing, c.Up, c.Eye, c.FOV, 384, 448)
	if d.View != view || d.Proj != proj {
		t.Errorf("Perspective did not set LookMatrices of the block")
	}
	if c.View != view || c.Proj != proj {
		t.Errorf("Perspective did not store the matrices in the block")
	}
	if c.Right != std.Right(c.Facing, c.Up) {
		t.Errorf("Right %v, want %v", c.Right, std.Right(c.Facing, c.Up))
	}
	if s.ANM.Shift != c.Shift {
		t.Errorf("ANM.Shift %v, want %v", s.ANM.Shift, c.Shift)
	}
	if s.Cur != &s.Screen || s.ANM.View != [4]uint32{0, 0, 640, 480} {
		t.Errorf("Perspective changed Cur or the cull rectangle")
	}
}

func TestStart(t *testing.T) {
	s, d, f := fixture(t)
	s.ClearColor = 0xff000000
	s.BeginFrame()
	s.Field.Shift = [2]float32{5, 6}
	s.EnterField()
	s.Fog(true)
	s.ANM.Draw(obj(f, 0, 0))
	s.ANM.Flush()
	s.ANM.TintOn = true
	d.Take()

	// The batch is empty (BeginFrame flushed it), so the flush in
	// Perspective submits nothing; the fog is left alone.
	s.Start()
	eqCalls(t, "Start", d.Take(), cat(screenView, []string{"clear 1 ff000000 1 []"}))
	view, proj := std.LookMatrices(s.Screen.Pos, s.Screen.Facing, s.Screen.Up, s.Screen.Eye, s.Screen.FOV, 640, 480)
	if d.View != view || d.Proj != proj {
		t.Errorf("Start's matrices are not Screen's LookMatrices")
	}
	if s.Cur != &s.Screen || !s.Is2D || s.ANM.View != [4]uint32{0, 0, 640, 480} {
		t.Errorf("Cur %p (Screen %p), Is2D %v, ANM.View %v", s.Cur, &s.Screen, s.Is2D, s.ANM.View)
	}
	if s.ANM.TintOn || s.ANM.Shift != s.Screen.Shift {
		t.Errorf("ANM.TintOn %v, ANM.Shift %v: Reset not applied", s.ANM.TintOn, s.ANM.Shift)
	}

	// ANM.Reset, not Invalidate: the filter is set again too.
	s.ANM.Draw(obj(f, 0, 0))
	s.ANM.Flush()
	eqCalls(t, "first object after Start", d.Take(), []string{"tex A", "blend 0", "filter 1", "draw 4 6"})
}

func TestUnshake(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	s.Field.Shift = [2]float32{2, -3}
	s.EnterField()
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()

	s.Unshake()
	eqCalls(t, "Unshake", d.Take(), nil)
	if s.Field.Shift != ([2]float32{}) || s.ANM.Shift != ([2]float32{}) {
		t.Errorf("Field.Shift %v, ANM.Shift %v; want zero", s.Field.Shift, s.ANM.Shift)
	}
}

func TestFinish(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	s.Field.Shift = [2]float32{2, -3}
	s.ANM.Shift = [2]float32{1, 1}
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()

	s.Finish()
	eqCalls(t, "Finish", d.Take(), []string{"draw 4 6"})
	if s.Field.Shift != ([2]float32{}) {
		t.Errorf("Field.Shift %v; want zero", s.Field.Shift)
	}
	if s.ANM.Shift != [2]float32{1, 1} {
		t.Errorf("Finish changed ANM.Shift to %v", s.ANM.Shift)
	}
}

func TestEndFrame(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()

	s.EndFrame()
	eqCalls(t, "EndFrame", d.Take(), []string{"draw 4 6", "tex nil", "flush"})

	// The manager still thinks texture A is set (Start is what resets it):
	// without a Start, the next frame's first quad on A sets no texture,
	// one on B does.
	s.BeginFrame()
	d.Take()
	s.ANM.Draw(obj(f, 0, 0))
	s.ANM.Draw(obj(f, 1, 0))
	s.ANM.Flush()
	eqCalls(t, "next frame", draws(d.Take()), []string{"draw 4 6", "tex B", "draw 4 6"})
}

func TestLegacy(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	s.ANM.Draw(obj(f, 0, 0))
	d.Take()
	if got := s.Legacy(); got != s.Frame {
		t.Errorf("Legacy returned %p, want Frame %p", got, s.Frame)
	}
	eqCalls(t, "Legacy", d.Take(), []string{"draw 4 6", "flush"})
}

func TestSheetCache(t *testing.T) {
	s, d, f := fixture(t)
	s.BeginFrame()
	d.Take()
	// fixture's NewSheet fails the test on a second call for f.
	for range 3 {
		s.ANM.Draw(obj(f, 0, 0))
	}
	s.ANM.Flush()
	if got := draws(d.Take()); len(got) != 2 {
		t.Errorf("three quads on one texture: %v", got)
	}
}
