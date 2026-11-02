// Package supervisor is th10.exe's supervisor object (0x491c28) as far as
// drawing goes: the device, the two camera blocks, which one is current,
// the FOGENABLE cache, and the frame's opening and closing calls. The
// stage's draw chain (scene/stage) runs between BeginFrame and EndFrame.
//
// The device and the ANM manager's caches live as long as the process, as
// in the original, so there is one Supervisor (Get). Tests build their own
// with New and a recording device (supervisortest.Rec).
//
// Design: docs/research/m4_render.md §2.2; evidence: m4_render_re.md §1.4,
// §2.4, §2.5, §5.1.
package supervisor

import (
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"th10/anm"
	"th10/anm/anmdraw"
	"th10/gfx"
	"th10/std"
)

// Device is the part of *gfx.Device the supervisor and the draw chain
// drive.
type Device interface {
	anmdraw.Device
	SetViewport(v gfx.Viewport)
	SetCull(c gfx.Cull)
	SetFog(f gfx.Fog)
	SetFogEnable(on bool)
	SetAddress(a gfx.Address)
	SetZEnable(on bool)
	SetZFunc(f gfx.CmpFunc)
	SetAlphaTest(on bool, ref uint8)
	SetZWrite(on bool)
	Clear(f gfx.ClearFlags, c gfx.Color, z float32, rects ...image.Rectangle)
	Flush()
}

var _ Device = (*gfx.Device)(nil)

// Camera is the part of a camera block the supervisor reads: the 2D block
// 0x491e94 (Screen) or the game-area block 0x491d7c (Field, into which STD
// copies its camera before each use, S3).
type Camera struct {
	// The look-at camera Perspective builds the matrices from. FUN_004216f0
	// sets both blocks to Pos (0, 0, 1000), Facing zero, Up (0, 1, 0), Eye
	// zero, FOV 0x3f060a92 (π/6). +0x24..+0x2c are not traced.
	Pos    [3]float32 // +0x00
	Facing [3]float32 // +0x0c, from Pos to the point looked at
	Up     [3]float32 // +0x18
	Right  [3]float32 // +0x30, std.Right(Facing, Up), written by Perspective
	Eye    [3]float32 // +0x3c, from Pos to the eye
	FOV    float32    // +0x48

	View, Proj gfx.Mat4 // +0x4c, +0x8c, the matrices last built for it

	// Viewport is +0xcc, a D3DVIEWPORT9: X, Y, W and H are whole numbers
	// (DWORDs in the original).
	Viewport gfx.Viewport

	// Shift is +0xe8, +0xec, copied to the ANM manager's Shift whenever
	// the block's matrices are set (Flat, Perspective). Field's is the
	// screen shake: FUN_0043c5ca writes DAT_00491e64/68, draw nodes 0x24
	// (Unshake) and 0x32 (Finish) zero it. Nothing writes Screen's.
	Shift [2]float32

	// Delta and the fog fields, as std.Camera holds them; Field receives
	// STD's at every STD draw (stddraw), Screen's stay zero. Render mode 6
	// fades by Field's FogNear, FogFar and FogBytes (anmdraw.Cams).
	Delta    [3]float32 // +0xf0
	FogNear  float32    // +0xfc
	FogFar   float32    // +0x100
	FogBytes [4]float32 // +0x104..+0x110, FogColor's B, G, R, A as floats
	FogColor uint32     // +0x114, ARGB
}

// Supervisor is the drawing half of 0x491c28.
type Supervisor struct {
	Dev Device // +8, DAT_00491c30

	// Frame is the device's back buffer, which the chain's legacy nodes
	// draw into with Ebiten (Legacy). It is nil with a recording device.
	Frame *ebiten.Image

	// ANM is the ANM manager's drawing state (DAT_00491c10).
	ANM *anmdraw.Renderer

	// NewSheet builds a file's runtime sprites the first time ANM draws
	// from it (see sheet); New sets it to anmdraw.NewSheet.
	NewSheet func(f *anm.File) *anmdraw.Sheet

	Screen Camera // 0x491e94
	Field  Camera // 0x491d7c

	Cur  *Camera // DAT_00491fac, the camera block in use
	Is2D bool    // _DAT_00491fb0: 1 under Screen, 0 under Field; its readers are not traced

	// ClearColor is DAT_004923a8, the colour Start clears the back buffer
	// to: zero at process start, 0xff000000 from stage start (FUN_00417c80;
	// zero when DAT_00474ca0 bit 0 is set, the same picture since Clear
	// makes it opaque), the fog colour after STD's fog instructions
	// (FUN_00403c80, S2).
	ClearColor gfx.Color

	fog    uint8 // DAT_00492378 (+0x750): FOGENABLE as last set, 0xff unknown
	sheets map[*anm.File]*anmdraw.Sheet
}

var (
	once sync.Once
	the  *Supervisor
)

// Get returns the process's supervisor, made on first use with
// New(d, d.Frame) for a new gfx.Device d.
func Get() *Supervisor {
	once.Do(func() {
		d := gfx.NewDevice()
		the = New(d, d.Frame)
	})
	return the
}

// New returns a supervisor over dev, as the device creation in
// FUN_00438ad0 leaves it:
//
//   - NewSheet = anmdraw.NewSheet; ANM = anmdraw.NewRenderer(dev, s.sheet);
//   - both cameras as FUN_004216f0 leaves them: the look-at fields in
//     Camera's comment, Screen.Viewport (0, 0, 640, 480, 0, 1),
//     Field.Viewport (32, 16, 384, 448, 0, 1); Right, matrices and Shift
//     zero;
//   - Cur nil, Is2D false, ClearColor 0, fog 0xff;
//   - then Init.
func New(dev Device, frame *ebiten.Image) *Supervisor {
	// FUN_004216f0: both blocks share the look-at fields and FOV; only the
	// viewports differ.
	cam := func(vp gfx.Viewport) Camera {
		return Camera{
			Pos:      [3]float32{0, 0, 1000},
			Up:       [3]float32{0, 1, 0},
			FOV:      math.Float32frombits(0x3f060a92),
			Viewport: vp,
		}
	}
	s := &Supervisor{
		Dev:      dev,
		Frame:    frame,
		NewSheet: anmdraw.NewSheet,
		Screen:   cam(gfx.Viewport{X: 0, Y: 0, W: 640, H: 480, MinZ: 0, MaxZ: 1}),
		Field:    cam(gfx.Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1}),
		fog:      0xff,
		sheets:   map[*anm.File]*anmdraw.Sheet{},
	}
	s.ANM = anmdraw.NewRenderer(dev, s.sheet)
	s.ANM.Cams = s
	s.Init()
	return s
}

// sheet is ANM's Sheet function: f's sheet from s.sheets, built with
// NewSheet and stored on first use.
func (s *Supervisor) sheet(f *anm.File) *anmdraw.Sheet {
	sh, ok := s.sheets[f]
	if !ok {
		sh = s.NewSheet(f)
		s.sheets[f] = sh
	}
	return sh
}

// Init is FUN_00439d20, the state set after device creation (and after a
// device reset, which is not ported). In this order:
//
//	SetZEnable(true)                       ZENABLE 1
//	SetCull(gfx.CullNone)                  CULLMODE 1
//	SetBlend(gfx.BlendAlpha)               SRCBLEND 5, DESTBLEND 6
//	SetZFunc(gfx.CmpAlways)                ZFUNC 8
//	SetAlphaTest(true, 1)                  ALPHATESTENABLE 1, ALPHAREF 1, ALPHAFUNC GREATEREQUAL
//	SetFogEnable(true)                     FOGENABLE 1
//	SetFog(gfx.Fog{Mode: gfx.FogVertexLinear, Color: 0xffa0a0a0, Start: 1000, End: 5000})
//	SetAddress(gfx.AddressWrap)            ADDRESSU, ADDRESSV 1
//	SetFilter(gfx.FilterLinear)            MAGFILTER, MINFILTER 2
//
// then ANM.Invalidate() (the manager caches it resets at the end). States
// gfx does not model (lighting, shading, texture stage arguments, mip
// filter, multisampling) are left out. It does not touch the fog cache:
// BeginFrame sets that.
func (s *Supervisor) Init() {
	s.Dev.SetZEnable(true)
	s.Dev.SetCull(gfx.CullNone)
	s.Dev.SetBlend(gfx.BlendAlpha)
	s.Dev.SetZFunc(gfx.CmpAlways)
	s.Dev.SetAlphaTest(true, 1)
	s.Dev.SetFogEnable(true)
	s.Dev.SetFog(gfx.Fog{Mode: gfx.FogVertexLinear, Color: 0xffa0a0a0, Start: 1000, End: 5000})
	s.Dev.SetAddress(gfx.AddressWrap)
	s.Dev.SetFilter(gfx.FilterLinear)
	s.ANM.Invalidate()
}

// Select makes c the camera block in use (DAT_00491fac = c) and sets
// ANM.View to c.Viewport's X, Y, W, H, which is where the manager reads
// its cull rectangle. Whoever changes the viewport of the block in use
// selects it again.
func (s *Supervisor) Select(c *Camera) {
	s.Cur = c
	v := c.Viewport
	s.ANM.View = [4]uint32{uint32(v.X), uint32(v.Y), uint32(v.W), uint32(v.H)}
}

var _ anmdraw.Cams = (*Supervisor)(nil)

// Selected is anmdraw.Cams.Selected: Cur's Right, View, Proj and
// Viewport.
func (s *Supervisor) Selected() (right [3]float32, view, proj gfx.Mat4, vp gfx.Viewport) {
	c := s.Cur
	return c.Right, c.View, c.Proj, c.Viewport
}

// FieldFog is anmdraw.Cams.FieldFog: Field's Pos, FogNear, FogFar and
// FogBytes[0:3].
func (s *Supervisor) FieldFog() (eye [3]float32, near, far float32, bgr [3]float32) {
	c := &s.Field
	return c.Pos, c.FogNear, c.FogFar, [3]float32(c.FogBytes[:3])
}

// Flat is FUN_00421480 on c: ANM.Flush(); c.View, c.Proj =
// std.FlatMatrices(c.Viewport.W, c.Viewport.H); SetTransform(gfx.View,
// c.View); SetTransform(gfx.Projection, c.Proj); ANM.Shift = c.Shift.
func (s *Supervisor) Flat(c *Camera) {
	s.ANM.Flush()
	c.View, c.Proj = std.FlatMatrices(uint32(c.Viewport.W), uint32(c.Viewport.H))
	s.Dev.SetTransform(gfx.View, c.View)
	s.Dev.SetTransform(gfx.Projection, c.Proj)
	s.ANM.Shift = c.Shift
}

// Perspective is FUN_004215a0 on c (EDI = c):
//
//	ANM.Flush()
//	c.View, c.Proj = std.LookMatrices(c.Pos, c.Facing, c.Up, c.Eye, c.FOV, c.Viewport.W, c.Viewport.H)
//	SetTransform(gfx.View, c.View); SetTransform(gfx.Projection, c.Proj)
//	c.Right = std.Right(c.Facing, c.Up)
//	ANM.Shift = c.Shift
//
// It neither selects c nor sets the viewport.
func (s *Supervisor) Perspective(c *Camera) {
	s.ANM.Flush()
	c.View, c.Proj = std.LookMatrices(c.Pos, c.Facing, c.Up, c.Eye, c.FOV, uint32(c.Viewport.W), uint32(c.Viewport.H))
	s.Dev.SetTransform(gfx.View, c.View)
	s.Dev.SetTransform(gfx.Projection, c.Proj)
	c.Right = std.Right(c.Facing, c.Up)
	s.ANM.Shift = c.Shift
}

// Use is the camera switch every switching site makes: Select(c), Flat(c),
// SetViewport(c.Viewport).
func (s *Supervisor) Use(c *Camera) {
	s.Select(c)
	s.Flat(c)
	s.Dev.SetViewport(c.Viewport)
}

// Fog is the FOGENABLE cache every writer goes through: with want 1 for on
// and 0 for off, if fog != want { ANM.Flush(); fog = want;
// SetFogEnable(on) }.
func (s *Supervisor) Fog(on bool) {
	want := uint8(0)
	if on {
		want = 1
	}
	if s.fog != want {
		s.ANM.Flush()
		s.fog = want
		s.Dev.SetFogEnable(on)
	}
}

// BeginFrame is what FUN_00439390 does before the draw chain, both before
// the calc chain and at the start of the draw section, which a frame with
// no calc in between makes one sequence:
//
//	ANM.Flush()           FUN_00442f50
//	Use(&s.Screen)        DAT_00491fac = 0x491e94, FUN_00421480, SetViewport
//	Is2D = true
//	fog = 0xff; Fog(false)  DAT_00492378 = 0xff, FUN_00420cd0
//
// The original also empties the ANM batch at the draw section; the flush
// has already left it empty.
func (s *Supervisor) BeginFrame() {
	s.ANM.Flush()
	s.Use(&s.Screen)
	s.Is2D = true
	s.fog = 0xff
	s.Fog(false)
}

// Start is draw node 1 (FUN_00420000), the first node of every frame's
// draw chain:
//
//	ANM.Reset()
//	Select(&s.Screen)               DAT_00491fac = 0x491e94
//	Perspective(&s.Screen)          Screen's Facing is zero: a degenerate view
//	SetViewport(s.Screen.Viewport)
//	Is2D = true
//	Clear(gfx.ClearTarget, ClearColor, 1)
//
// The depth buffer is not cleared here; STD does that (S3).
func (s *Supervisor) Start() {
	s.ANM.Reset()
	s.Select(&s.Screen)
	s.Perspective(&s.Screen)
	s.Dev.SetViewport(s.Screen.Viewport)
	s.Is2D = true
	s.Dev.Clear(gfx.ClearTarget, s.ClearColor, 1)
}

// Unshake is what draw node 0x24 (FUN_00448740) does before its layer,
// 14: Field.Shift and ANM.Shift zero, without a flush (quads already in
// the batch were shifted when they were made). Everything after it in the
// frame is drawn unshaken.
func (s *Supervisor) Unshake() {
	s.Field.Shift = [2]float32{}
	s.ANM.Shift = [2]float32{}
}

// Finish is draw node 0x32 (FUN_004200d0), the chain's last: ANM.Flush(),
// Field.Shift zero.
func (s *Supervisor) Finish() {
	s.ANM.Flush()
	s.Field.Shift = [2]float32{}
}

// EnterField is draw node 0xf (FUN_00448620) before its layer:
// Use(&s.Field); Is2D = false; Fog(false).
func (s *Supervisor) EnterField() {
	s.Use(&s.Field)
	s.Is2D = false
	s.Fog(false)
}

// Enter2D is draw nodes 0x29, 0x2a and 0x2d (FUN_00448770, 0x4487c0,
// 0x448810) before their layer: Use(&s.Screen); Is2D = true.
func (s *Supervisor) Enter2D() {
	s.Use(&s.Screen)
	s.Is2D = true
}

// EndFrame is what FUN_00439390 does after the draw chain: ANM.Flush(),
// then Dev.SetTexture(nil) (0x104, on the device only: the manager's
// texture cache keeps the last texture until Start resets it), then
// Dev.Flush() so Frame holds the picture (EndScene).
func (s *Supervisor) EndFrame() {
	s.ANM.Flush()
	s.Dev.SetTexture(nil)
	s.Dev.Flush()
}

// Legacy is for chain nodes not yet ported, which draw into the back
// buffer with Ebiten: ANM.Flush(), Dev.Flush(), and return Frame. Ebiten
// keeps an image's draws in submission order, so the legacy picture lands
// in its place in the chain. Neither flush is the original's.
func (s *Supervisor) Legacy() *ebiten.Image {
	s.ANM.Flush()
	s.Dev.Flush()
	return s.Frame
}
