// Package anmdraw draws ANM objects as th10.exe's ANM manager (DAT_00491c10)
// does: FUN_004451c0 turns one object into a quad, and the quads are
// gathered into one triangle-list batch of pre-transformed vertices that is
// submitted whenever the texture or a render state has to change.
//
// This file covers render modes 0-3, the 2D quads (A1a in
// docs/research/m4_render.md §4); mode8.go covers mode 8, the 3D sprite,
// mode46.go modes 4 and 6, the projected billboards, and mode9.go mode 9,
// op 84's ring. Modes 5 and 7 are not ported: no embedded .anm uses them
// (docs/research/m4_modes.md).
//
// The manager caches the device state it last set (texture, blend, filter,
// vertex type, texture factor, texture matrix) and only touches the device when an object needs a different
// value; every change first submits the pending batch. Renderer keeps the same
// caches and makes the same calls in the same order, so a gfx.Device fed by
// it sees exactly the original's state sequence. Code that draws through the
// device directly must Flush first and Invalidate afterwards, as the
// original's direct draws do (FUN_0043bda0).
//
// Arithmetic is float32 with the binary's operand grouping: th10 runs the
// x87 at 24-bit precision (see package zun), so each add, subtract and
// multiply rounds like float32, and a different grouping of the same sum
// can give a different result. The doc comments write every sum with
// explicit brackets; keep them, and wrap products in float32() so the
// compiler cannot fuse them into an FMA.
package anmdraw

import (
	"fmt"
	"image"
	"math"

	"th10/anm"
	"th10/gfx"
)

// Sprite is a runtime sprite, the 0x44-byte record the sprite op points an
// object at (FUN_0043e5a0 sets VM+0x394 to it).
type Sprite struct {
	Tex *gfx.Texture // +0x04, the texture the sprite lives on

	// U0, V0, U1, V1 are the sprite's rectangle in texture coordinates
	// (+0x20, +0x24, +0x28, +0x2c).
	U0, V0, U1, V1 float32

	// W and H are its size in pixels (+0x34, +0x30). The sprite op copies
	// them to VM+0x4c and +0x50, which is what the quad functions read;
	// until R6-C finds another writer of those, Draw reads them here.
	W, H float32
}

// Sheet is one ANM file's runtime sprites, indexed by sprite number.
type Sheet struct {
	Sprites []Sprite
}

// NewSheet builds f's runtime sprites and their textures as the ANM loader
// does (FUN_00447470 for each entry, FUN_00447940 for each sprite;
// docs/research/m4_r6.md R6-A):
//
//   - Every entry gets a texture of its own, entries sharing a name too:
//     gfx.NewTexture(entryImage(e)).
//   - Sprites[i] is the record of the sprite with id i; len(Sprites) is
//     len(f.Sprites). Ids are file order 0..n-1 in every TH10 .anm, which is
//     what the original indexes by; NewSheet panics on an id outside that.
//
// A record is computed in float32 from the file's sprite S, its entry's
// Width, Height (We, He) and its texture's (Wd, Hd; equal, since every TH10
// entry is a power of two and D3DX pads nothing):
//
//	f1 = Wd/We                 f2 = Hd/He
//	x0 = S.X·f1                y0 = S.Y·f2
//	x1 = (S.W + S.X)·f1        y1 = (S.H + S.Y)·f2
//	U0 = x0/Wd  U1 = x1/Wd     V0 = y0/Hd  V1 = y1/Hd
//	W = (x1 − x0)/f1           H = (y1 − y0)/f2
//
// There is no half-texel offset or epsilon.
func NewSheet(f *anm.File) *Sheet {
	texs := make([]*gfx.Texture, len(f.Entries))
	for i := range f.Entries {
		texs[i] = gfx.NewTexture(entryImage(&f.Entries[i]))
	}
	s := &Sheet{Sprites: make([]Sprite, len(f.Sprites))}
	for id, sp := range f.Sprites {
		if id < 0 || id >= len(f.Sprites) {
			panic(fmt.Sprintf("anmdraw: sprite id %d outside 0..%d", id, len(f.Sprites)-1))
		}
		e := &f.Entries[sp.Entry]
		tex := texs[sp.Entry]
		wd, hd := float32(tex.W), float32(tex.H)
		we, he := float32(e.Width), float32(e.Height)
		f1, f2 := wd/we, hd/he
		x0, y0 := float32(sp.X*f1), float32(sp.Y*f2)
		x1, y1 := float32((sp.W+sp.X)*f1), float32((sp.H+sp.Y)*f2)
		s.Sprites[id] = Sprite{
			Tex: tex,
			U0:  x0 / wd, V0: y0 / hd, U1: x1 / wd, V1: y1 / hd,
			W: (x1 - x0) / f1, H: (y1 - y0) / f2,
		}
	}
	return s
}

// entryImage is the texture FUN_00446f40 creates for e: e.Width x e.Height,
// e.Texture.Image() at the top left and zero texels elsewhere
// (D3DX_FILTER_NONE copies the THTX block without scaling; some blocks are
// smaller than their entry). Zero is opaque black in format 3 (R5G6B5 has
// no alpha) and transparent black otherwise. "@" entries, which have no
// Texture and are filled at run time by game code not ported yet, are all
// zero. Formats 1, 3 and 5 are the only ones in TH10's 46 .anm files.
func entryImage(e *anm.Entry) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, e.Width, e.Height))
	if e.Format == anm.FormatRGB565 {
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 0xff
		}
	}
	if e.Texture != nil {
		// Copy the overlap; a block can be smaller than its entry.
		src := e.Texture.Image()
		w := min(e.Width, src.Rect.Dx())
		h := min(e.Height, src.Rect.Dy())
		for y := range h {
			copy(img.Pix[img.PixOffset(0, y):], src.Pix[src.PixOffset(0, y):src.PixOffset(w, y)])
		}
	}
	return img
}

// Device is the part of *gfx.Device the manager drives.
type Device interface {
	SetTexture(t *gfx.Texture)
	SetFilter(f gfx.Filter)
	SetBlend(b gfx.BlendMode)
	SetTransform(t gfx.TransformState, m gfx.Mat4)
	DrawTL(p gfx.Primitive, vs []gfx.TLVertex)
	Draw3D(p gfx.Primitive, vs []gfx.Vertex3D)
}

// Vertex type cache values (manager +0x3ada6a).
const (
	vtypeNone uint8 = 0    // FUN_00445900: nothing set yet
	vtype2D   uint8 = 1    // the TL batch of FUN_00442670
	vtype3D   uint8 = 2    // mode 8's vertex buffer (FUN_00444760)
	vtypeRing uint8 = 3    // mode 9's ring (FUN_00444ce0)
	vtypeAny  uint8 = 0xff // Invalidate: unknown
)

// noSpecular is the specular colour of every TL vertex. The original's
// vertices (FVF 0x144) have no specular; gfx takes the fog factor from the
// specular alpha, and 0xff (no fog) is the assumption docs/research/
// m4_render.md §2.1 G3 records for such vertices.
const noSpecular gfx.Color = 0xff000000

// Renderer is the ANM manager's 2D drawing state.
type Renderer struct {
	Dev Device

	// Sheet returns the runtime sprites of f.
	Sheet func(f *anm.File) *Sheet

	// View is the cull rectangle, the viewport the supervisor last set
	// (DAT_00491fac+0xcc, +0xd0, +0xd4, +0xd8: X, Y, W, H, unsigned).
	View [4]uint32

	// Cams gives modes 4 and 6 the supervisor's camera blocks (see Cams).
	Cams Cams

	// Shift is added to every corner of every quad before snapping
	// (manager +0x5c, +0x60). R6-B finds who sets it.
	Shift [2]float32

	// While TintOn (manager +0x73245c is nonzero), every quad's colour is
	// modulated by Tint (+0x732458), see emit.
	TintOn bool
	Tint   gfx.Color

	// Device state caches, as FUN_00445900 leaves them: no texture, blend
	// 0, vertex type none, linear filtering.
	tex   *gfx.Texture // +0x3ada64
	blend uint8        // +0x3ada68
	vtype uint8        // +0x3ada6a
	// point is the filter cache (+0x3ada6e): 0 linear, 1 point, 0xff unknown
	// (Reset). blendFilter compares it with the object's bit 31 as a byte, so
	// 0xff always differs.
	point uint8

	// tfactor is the TEXTUREFACTOR cache (+0x3ada60), which FUN_00445900
	// sets to 1 and nothing but states3D changes: neither Reset nor
	// Invalidate touches it. tfDev is the device's TEXTUREFACTOR,
	// Direct3D's default 0xffffffff until states3D first sets it; the two
	// differ until then, so a first object whose colour is 1 (alpha 0,
	// which only op 80 or the tint can give a drawn object) draws with
	// 0xffffffff. gfx has no TEXTUREFACTOR, see draw8.
	tfactor gfx.Color
	tfDev   gfx.Color

	// sprite is the texture matrix cache (+0x3ada70): the sprite draw8 last
	// set TEXTURE0 for. Reset and Invalidate clear it.
	sprite *Sprite

	// quad is the staging quad DAT_004978c0: corners P0 (top left), P1
	// (top right), P2 (bottom left), P3 (bottom right) in that order. It
	// persists between draws: anchor 3 leaves corners as the previous quad
	// left them. FUN_00445900 sets RHW 1 and UVs (0,0), (1,0), (0,1),
	// (1,1); everything else starts at zero. NewRenderer applies that, and
	// gives every corner Specular noSpecular, which nothing changes after.
	quad [4]gfx.TLVertex

	// batch is the pending triangle list (+0x72dad0 up to +0x72dacc).
	batch []gfx.TLVertex

	// strip is draw9's scratch: the ring as TL vertices, rebuilt for every
	// mode 9 object. Like batch, Dev.DrawTL must not keep it.
	strip []gfx.TLVertex
}

// NewRenderer returns a renderer in the state FUN_00445900 leaves the
// manager in (see the cache fields and quad).
func NewRenderer(dev Device, sheet func(*anm.File) *Sheet) *Renderer {
	r := &Renderer{Dev: dev, Sheet: sheet, vtype: vtypeNone, tfactor: 1, tfDev: 0xffffffff}
	uv := [4][2]float32{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	for i := range r.quad {
		r.quad[i] = gfx.TLVertex{RHW: 1, Specular: noSpecular, U: uv[i][0], V: uv[i][1]}
	}
	return r
}

// Draw is FUN_004451c0. Nothing is drawn unless Flags bits 0 and 1 are
// both set and vm.Alpha (+0x2ff, the alpha of Color, even when op 80
// selects Color2) is nonzero. Then, by mode = Flags>>22 & 0xf:
//
//   - 0: quad0.
//   - 1 and 3 (one code path): quad2 if Rot[2] == 0, otherwise rotated.
//   - 2: quad2.
//   - 4: draw4.
//   - 6: draw6.
//   - 8: draw8.
//   - 9: draw9.
//   - 5 and 7: panic (not ported; no embedded .anm uses them,
//     docs/research/m4_modes.md).
//   - 10-15: nothing.
//
// The object's sprite is r.Sheet(vm.File).Sprites[vm.Sprite]. An object
// that gets this far without a sprite would make the original dereference
// a null pointer, so Draw panics.
func (r *Renderer) Draw(vm *anm.VM) {
	if vm.Flags&3 != 3 || vm.Alpha == 0 {
		return
	}
	mode := vm.Flags >> 22 & 0xf
	switch {
	case mode >= 10:
		return
	case mode == 5 || mode == 7:
		panic(fmt.Sprintf("anmdraw: render mode %d is not ported", mode))
	}
	spr := &r.Sheet(vm.File).Sprites[vm.Sprite]
	switch mode {
	case 4:
		r.draw4(vm, spr)
	case 6:
		r.draw6(vm, spr)
	case 8:
		r.draw8(vm, spr)
	case 9:
		r.draw9(vm, spr)
	case 0:
		r.quad0(vm, spr)
	case 2:
		r.quad2(vm, spr)
	case 1, 3:
		if vm.Rot[2] == 0 {
			r.quad2(vm, spr)
		} else {
			r.rotated(vm, spr)
		}
	}
}

// DrawLayer is FUN_00448980: Draw for each object in w's layer l, in list
// order, skipping those with anm.FlagDelete. The original first calls the
// object's pre-draw hook (VM+0x39c) when it has one. Only mode 10's
// particles set one (R6-G; the other writer, FUN_0040b110, has no callers
// and its table's one entry has no hook), and the VM panics on mode 10, so
// no object here has a hook.
func (r *Renderer) DrawLayer(w *anm.World, l int) {
	w.Each(l, func(vm *anm.VM) {
		if vm.Flags&anm.FlagDelete == 0 {
			r.Draw(vm)
		}
	})
}

// Flush is FUN_00442f50: if the batch holds anything, submit it as one
// DrawTL(gfx.TriangleList, batch) and empty it. The original also sets
// COLORARG2 and ALPHAARG2 of stage 0 to DIFFUSE and the FVF to 0x144;
// texture times diffuse is the only combination gfx has. An empty batch
// makes no device call.
func (r *Renderer) Flush() {
	if len(r.batch) > 0 {
		r.Dev.DrawTL(gfx.TriangleList, r.batch)
		r.batch = r.batch[:0]
	}
}

// Invalidate forgets the cached device state after someone else has drawn,
// as FUN_0043bda0 and FUN_0043bfa0 do: texture nil, blend 3, vertex type
// vtypeAny, texture matrix sprite nil. It does not flush, and the filter
// and TEXTUREFACTOR caches are kept (neither site resets +0x3ada6e or
// +0x3ada60).
//
// Blend 3 matches no mode that sets the blend, so the next object with
// blend 0, 1 (or 2 in 3D) sets it again; one with blend 3 does not.
func (r *Renderer) Invalidate() {
	r.tex = nil
	r.blend = 3
	r.vtype = vtypeAny
	r.sprite = nil
}

// Reset is the cache reset draw node 1 (FUN_00420000) makes at the start
// of every frame's draw chain: what Invalidate resets, and also the filter
// cache to unknown (+0x3ada6e = 0xff, so the next object sets the filter
// whichever it wants: the cache becomes three-valued, 0 linear, 1 point,
// 0xff unknown), TintOn false (+0x73245c = 0), Tint 0x80808080
// (+0x732458) and Shift zero (+0x5c, +0x60). It does not flush. The other
// fields it resets (+0x3ada69, +0x3ada6b, +0x3ada6c) have no reader here.
func (r *Renderer) Reset() {
	r.Invalidate()
	r.point = 0xff
	r.TintOn = false
	r.Tint = 0x80808080
	r.Shift = [2]float32{}
}

// Notation for the quad functions, all on vm and its sprite spr:
//
//	w = spr.W·Scale[0], h = spr.H·Scale[1]
//	ax = Flags>>18 & 3, ay = Flags>>20 & 3 (op 65)
//	Ax = (Pos2.x + Offset.x) + Pos.x     Ay = (Pos2.y + Offset.y) + Pos.y
//	Bx = (Pos.x + Offset.x) + Pos2.x     Cy = (Pos2.y + Pos.y) + Offset.y
//
// Each sets x0, x1 (P0.x = P2.x = x0, P1.x = P3.x = x1), y0, y1 (P0.y =
// P1.y = y0, P2.y = P3.y = y1) and Z of all four corners, then calls emit.
// An anchor of 3 sets none of that axis's corners, so they keep the
// previous quad's values.

// quad0 is FUN_00443080, mode 0. floor is float32(math.Floor(float64(·)))
// (FUN_00452ff0).
//
//	ax 0: x0 = floor(Ax − w·0.5)      x1 = x0 + w
//	ax 1: x0 = Bx                      x1 = ((Pos.x + Offset.x) + w) + Pos2.x
//	ax 2: x0 = Bx − w                  x1 = Bx
//	ay 0: y0 = floor(Ay − h·0.5)      y1 = y0 + h
//	ay 1: y0 = Ay                      y1 = Ay + h
//	ay 2: y0 = Ay − h                  y1 = Ay
//	Z = (Pos2.z + Offset.z) + Pos.z
//
// Then emit with emitSnap.
func (r *Renderer) quad0(vm *anm.VM, spr *Sprite) {
	w := float32(spr.W * vm.Scale[0])
	h := float32(spr.H * vm.Scale[1])
	ax := vm.Flags >> 18 & 3
	ay := vm.Flags >> 20 & 3
	Ax := (vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]
	Ay := (vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]
	Bx := (vm.Pos[0] + vm.Offset[0]) + vm.Pos2[0]
	// FUN_00443080 branches: x ax 0 0x443125-0x443166, ax 1 0x4430f0-0x443123,
	// ax 2 0x4430bb-0x4430ee; y ay 0 0x4431fa-0x443240, ay 1 0x4431c8-0x4431f8,
	// ay 2 0x443192-0x4431c6; Z 0x443246-0x443272; emit with snap at 0x443278.
	if ax != 3 {
		var x0, x1 float32
		switch ax {
		case 0:
			x0 = float32(math.Floor(float64(Ax - float32(w*0.5))))
			x1 = x0 + w
		case 1:
			x0 = Bx
			x1 = ((vm.Pos[0] + vm.Offset[0]) + w) + vm.Pos2[0]
		case 2:
			x0 = Bx - w
			x1 = Bx
		}
		r.quad[0].X, r.quad[2].X = x0, x0
		r.quad[1].X, r.quad[3].X = x1, x1
	}
	if ay != 3 {
		var y0, y1 float32
		switch ay {
		case 0:
			y0 = float32(math.Floor(float64(Ay - float32(h*0.5))))
			y1 = y0 + h
		case 1:
			y0 = Ay
			y1 = Ay + h
		case 2:
			y0 = Ay - h
			y1 = Ay
		}
		r.quad[0].Y, r.quad[1].Y = y0, y0
		r.quad[2].Y, r.quad[3].Y = y1, y1
	}
	z := (vm.Pos2[2] + vm.Offset[2]) + vm.Pos[2]
	for i := range r.quad {
		r.quad[i].Z = z
	}
	r.emit(vm, spr, emitSnap)
}

// quad2 is FUN_00443290, modes 2 and unrotated 1 and 3.
//
//	ax 0: x0 = Ax − w·0.5             x1 = w + x0
//	ax 1: x0 = Bx                      x1 = ((Pos.x + Offset.x) + w) + Pos2.x
//	ax 2: x0 = Bx − w                  x1 = Bx
//	ay 0: y0 = Ay − h·0.5             y1 = y0 + h
//	ay 1: y0 = Cy                      y1 = ((Pos2.y + Pos.y) + h) + Offset.y
//	ay 2: y0 = Cy − h                  y1 = Cy
//	Z = (Pos2.y + Offset.z) + Pos.z
//
// Z really reads Pos2.y, not Pos2.z (+0x350 at 0x0044343b): a ZUN bug, keep
// it. Then emit with 0.
func (r *Renderer) quad2(vm *anm.VM, spr *Sprite) {
	w := float32(spr.W * vm.Scale[0])
	h := float32(spr.H * vm.Scale[1])
	ax := vm.Flags >> 18 & 3
	ay := vm.Flags >> 20 & 3
	Ax := (vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]
	Ay := (vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]
	Bx := (vm.Pos[0] + vm.Offset[0]) + vm.Pos2[0]
	Cy := (vm.Pos2[1] + vm.Pos[1]) + vm.Offset[1]
	// FUN_00443290 branches: x ax 0 0x443335-0x443375, ax 1 0x443307-0x443333
	// (tail 0x4432f3-0x443305), ax 2 0x4432c7-0x443305; y ay 0 0x4433fe-0x443435,
	// ay 1 0x4433c9-0x4433fc, ay 2 0x443394-0x4433c7; Z 0x44343b-0x443470.
	if ax != 3 {
		var x0, x1 float32
		switch ax {
		case 0:
			x0 = Ax - float32(w*0.5)
			x1 = w + x0
		case 1:
			x0 = Bx
			x1 = ((vm.Pos[0] + vm.Offset[0]) + w) + vm.Pos2[0]
		case 2:
			x0 = Bx - w
			x1 = Bx
		}
		r.quad[0].X, r.quad[2].X = x0, x0
		r.quad[1].X, r.quad[3].X = x1, x1
	}
	if ay != 3 {
		var y0, y1 float32
		switch ay {
		case 0:
			y0 = Ay - float32(h*0.5)
			y1 = y0 + h
		case 1:
			y0 = Cy
			y1 = ((vm.Pos2[1] + vm.Pos[1]) + h) + vm.Offset[1]
		case 2:
			y0 = Cy - h
			y1 = Cy
		}
		r.quad[0].Y, r.quad[1].Y = y0, y0
		r.quad[2].Y, r.quad[3].Y = y1, y1
	}
	// ZUN bug: Z reads Pos2.y (+0x350), not Pos2.z (0x44343b).
	z := (vm.Pos2[1] + vm.Offset[2]) + vm.Pos[2]
	for i := range r.quad {
		r.quad[i].Z = z
	}
	r.emit(vm, spr, 0)
}

// rotated is the mode 1 and 3 path of FUN_004451c0 when Rot[2] != 0. With
// c = float32(cos Rot[2]) and s = float32(sin Rot[2]) (FSINCOS, both stored
// as float32 before use) and the edges
//
//	ax 0: L = −0.5·w, R = w·0.5     ay 0: T = h·(−0.5), B = h·0.5
//	ax 1: L = 0, R = w              ay 1: T = 0, B = h
//	ax 2: L = −w, R = 0             ay 2: T = −h, B = 0
//
// the corners are
//
//	P0 = ((L·c − T·s) + Ax, (L·s + T·c) + Ay)
//	P1 = ((R·c − T·s) + Ax, (T·c + R·s) + Ay)
//	P2 = ((L·c − B·s) + Ax, (L·s + c·B) + Ay)
//	P3 = ((R·c − B·s) + Ax, (R·s + B·c) + Ay)
//
// and Z = (Pos2.z + Offset.z) + Pos.z for all four. An anchor of 3 reads
// uninitialised stack slots in the original, so rotated panics on it. Then
// emit with 0.
func (r *Renderer) rotated(vm *anm.VM, spr *Sprite) {
	if vm.Flags>>18&3 == 3 || vm.Flags>>20&3 == 3 {
		panic("anmdraw: rotated quad with anchor 3 reads uninitialised stack")
	}
	// FSINCOS at 0x4436ed; the stored float32 values are c and s.
	c := float32(math.Cos(float64(vm.Rot[2])))
	s := float32(math.Sin(float64(vm.Rot[2])))
	w := float32(spr.W * vm.Scale[0])
	h := float32(spr.H * vm.Scale[1])
	ax := vm.Flags >> 18 & 3
	ay := vm.Flags >> 20 & 3
	Ax := (vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]
	Ay := (vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]
	// The mode 1 copy at 0x4436c0 (mode 3's at 0x443910 is identical):
	// Ax 0x4436f7-0x44370e, Ay 0x44371a-0x443726, FSINCOS 0x4436ed.
	// Edges: ax 0 L,R 0x443774-0x44378e, ax 1 0x44375e-0x443772,
	// ax 2 0x443744-0x443756, ax 3 0x443790 (uninitialised). ay 0 T,B
	// 0x4437e3-0x443803, ay 1 0x4437c5-0x4437e1, ay 2 0x4437a7-0x4437c3,
	// ay 3 0x443805 (uninitialised). Corners P0.X 0x44380d-0x443821,
	// P0.Y 0x443827-0x44383b, P1.X 0x443841-0x443853, P1.Y 0x44385b-0x44386b,
	// P2.X 0x443871-0x443883, P2.Y 0x443889-0x44389b, P3.X 0x4438a1-0x4438b5,
	// P3.Y 0x4438bb-0x4438cf; Z 0x4438db-0x4438ff; emit with snap 0 at 0x443905.
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
	r.quad[0].X = (float32(l*c) - float32(tp*s)) + Ax
	r.quad[0].Y = (float32(l*s) + float32(tp*c)) + Ay
	r.quad[1].X = (float32(rr*c) - float32(tp*s)) + Ax
	r.quad[1].Y = (float32(tp*c) + float32(rr*s)) + Ay
	r.quad[2].X = (float32(l*c) - float32(b*s)) + Ax
	r.quad[2].Y = (float32(l*s) + float32(c*b)) + Ay
	r.quad[3].X = (float32(rr*c) - float32(b*s)) + Ax
	r.quad[3].Y = (float32(rr*s) + float32(b*c)) + Ay
	z := (vm.Pos2[2] + vm.Offset[2]) + vm.Pos[2]
	for i := range r.quad {
		r.quad[i].Z = z
	}
	r.emit(vm, spr, 0)
}

// emit's argument bits, FUN_00442670's stack argument: mode 0 passes
// emitSnap, modes 1-5 pass 0, and modes 6 and 7 pass emitKeep
// (docs/research/m4_r6.md R6-E).
const (
	emitSnap uint8 = 1 // bit 0: snap the corners to the pixel grid
	emitKeep uint8 = 2 // bit 1: keep the colour the caller staged
)

// emit is FUN_00442670 with argument arg:
//
//  1. Add Shift to every corner's X and Y.
//  2. If arg&emitSnap: x0 = RoundToEven(P0.X) − 0.5, x1 = RoundToEven(P1.X) − 0.5,
//     y0 = RoundToEven(P0.Y) − 0.5, y1 = RoundToEven(P2.Y) − 0.5 (FRNDINT
//     in the default rounding mode), and the corners become the axis-aligned
//     rectangle x0, x1, y0, y1.
//  3. UVs: P0 (U0 + UV[0], V0 + UV[1]), P1 (U1 + UV[0], V0 + UV[1]),
//     P2 (U0 + UV[0], V1 + UV[1]), P3 (U1 + UV[0], V1 + UV[1]), each sum
//     sprite value plus scroll.
//  4. Cull, see culled. A culled quad returns here: no cache, device or
//     batch change.
//  5. Texture: if spr.Tex != r.tex, Flush, r.tex = spr.Tex,
//     SetTexture(spr.Tex).
//  6. Vertex type: if r.vtype != vtype2D, Flush, r.vtype = vtype2D.
//  7. Colour: r.colour(vm) on all four corners, unless arg&emitKeep
//     (0x442a16), which keeps the Diffuse the caller staged in r.quad.
//  8. blendFilter.
//  9. push.
//
// The staged corners and UVs stay in r.quad whether or not the quad is
// culled.
func (r *Renderer) emit(vm *anm.VM, spr *Sprite, arg uint8) {
	// 1. Shift every corner (0x44267e-0x4426f4).
	for i := range r.quad {
		r.quad[i].X = r.quad[i].X + r.Shift[0]
		r.quad[i].Y = r.quad[i].Y + r.Shift[1]
	}
	// 2. Snap to the pixel grid, rounding half to even (FRNDINT,
	// 0x4426fc-0x44275e).
	if arg&emitSnap != 0 {
		x0 := float32(math.RoundToEven(float64(r.quad[0].X))) - 0.5
		x1 := float32(math.RoundToEven(float64(r.quad[1].X))) - 0.5
		y0 := float32(math.RoundToEven(float64(r.quad[0].Y))) - 0.5
		y1 := float32(math.RoundToEven(float64(r.quad[2].Y))) - 0.5
		r.quad[0].X, r.quad[2].X = x0, x0
		r.quad[1].X, r.quad[3].X = x1, x1
		r.quad[0].Y, r.quad[1].Y = y0, y0
		r.quad[2].Y, r.quad[3].Y = y1, y1
	}
	// 3. Texture coordinates, scrolled by UV (0x442764-0x4427be).
	u0, v0 := spr.U0+vm.UV[0], spr.V0+vm.UV[1]
	u1, v1 := spr.U1+vm.UV[0], spr.V1+vm.UV[1]
	r.quad[0].U, r.quad[0].V = u0, v0
	r.quad[1].U, r.quad[1].V = u1, v0
	r.quad[2].U, r.quad[2].V = u0, v1
	r.quad[3].U, r.quad[3].V = u1, v1

	// 4. Cull.
	if r.culled() {
		return
	}

	// 5. Texture. Only a change flushes and sets the device (0x4429ce-0x4429fb).
	if spr.Tex != r.tex {
		r.Flush()
		r.tex = spr.Tex
		r.Dev.SetTexture(spr.Tex)
	}
	// 6. Vertex type, which the batch needs (no device call).
	if r.vtype != vtype2D {
		r.Flush()
		r.vtype = vtype2D
	}
	// 7. Colour (0x442a1f-0x442a94), skipped by emitKeep (0x442a16).
	if arg&emitKeep == 0 {
		col := r.colour(vm)
		for i := range r.quad {
			r.quad[i].Diffuse = col
		}
	}
	// 8. Blend and filter state.
	r.blendFilter(vm)
	// 9. Append the two triangles (FUN_00442fe0).
	r.push()
}

// colour is the D3DCOLOR an object draws with: Color2 and Alpha2 if
// Flags&0x8000 (op 80), else Color and Alpha, as 0xAARRGGBB. While TintOn
// each of its four bytes c becomes min((c·t)>>7, 255) with t the same byte
// of Tint (FUN_004423c0). emit's DIFFUSE (0x442a1f-0x442a94) and states3D's
// TEXTUREFACTOR (0x44244e-0x442505) are both this.
func (r *Renderer) colour(vm *anm.VM) gfx.Color {
	a, cr, cg, cb := vm.Alpha, vm.Color[0], vm.Color[1], vm.Color[2]
	if vm.Flags&0x8000 != 0 {
		a, cr, cg, cb = vm.Alpha2, vm.Color2[0], vm.Color2[1], vm.Color2[2]
	}
	col := uint32(a)<<24 | uint32(cr)<<16 | uint32(cg)<<8 | uint32(cb)
	if r.TintOn {
		// FUN_004423c0: each byte is modulated by the same byte of Tint.
		var m uint32
		for i := range 4 {
			sh := uint(8 * i)
			c := col >> sh & 0xff
			t := uint32(r.Tint) >> sh & 0xff
			m |= min((c*t)>>7, 255) << sh
		}
		col = m
	}
	return col
}

// culled is step 4 of emit (0x4427c4-0x4429c8), on the shifted, snapped
// corners of r.quad:
//
//	maxX = P0.X > P1.X ? P0.X : P1.X, then P2.X, then P3.X where greater
//	minX = P0.X < P1.X ? P0.X : P1.X, then P2.X, then P3.X where less
//
// and maxY, minY alike. The quad is culled if maxX < X, Y > maxY,
// X+W < minX or Y+H < minY, with X, Y, W, H the uint32s of View, X+W and
// Y+H summed as uint32, and the integers compared exactly (FILD). These
// are the binary's comparisons and branch directions: its x87 branches are
// taken only on an ordered result, so as in Go a comparison with NaN is
// false. On NaN each selection keeps its second operand (P1 in the first,
// the bound so far after that), and a NaN bound never culls.
func (r *Renderer) culled() bool {
	q := &r.quad
	// Selections 0x4427c4-0x442921: TEST AH,0x41 / JNZ keeps the second
	// operand unless the first is greater; TEST AH,5 / JP keeps it unless
	// the first is less.
	maxX, minX := q[1].X, q[1].X
	if q[0].X > q[1].X {
		maxX = q[0].X
	}
	if q[0].X < q[1].X {
		minX = q[0].X
	}
	maxY, minY := q[1].Y, q[1].Y
	if q[0].Y > q[1].Y {
		maxY = q[0].Y
	}
	if q[0].Y < q[1].Y {
		minY = q[0].Y
	}
	for _, c := range q[2:] {
		if c.X > maxX {
			maxX = c.X
		}
		if c.X < minX {
			minX = c.X
		}
		if c.Y > maxY {
			maxY = c.Y
		}
		if c.Y < minY {
			minY = c.Y
		}
	}
	// Tests 0x44292c-0x4429c8, each a jump to the exit at 0x442abb.
	x, y := float64(r.View[0]), float64(r.View[1])
	x1, y1 := float64(r.View[0]+r.View[2]), float64(r.View[1]+r.View[3])
	return float64(maxX) < x || y > float64(maxY) || x1 < float64(minX) || y1 < float64(minY)
}

// blendFilter is FUN_004425a0. With b = Flags>>4 & 3 (op 66): if b !=
// r.blend, Flush, r.blend = b, and SetBlend(gfx.BlendAlpha) for b 0 or
// SetBlend(gfx.BlendAdd) for b 1; b 2 and 3 make no call and leave the
// device's blend as it was (the original only writes DESTBLEND, and only
// for 0 and 1). Then filter.
func (r *Renderer) blendFilter(vm *anm.VM) {
	b := uint8(vm.Flags >> 4 & 3)
	if b != r.blend {
		r.Flush()
		r.blend = b
		switch b {
		case 0:
			r.Dev.SetBlend(gfx.BlendAlpha)
		case 1:
			r.Dev.SetBlend(gfx.BlendAdd)
		}
	}
	r.filter(vm)
}

// filter is the sampler half of FUN_004425a0 and FUN_004423e0, the same in
// both: with p = Flags bit 31 as a byte (0 or 1), if p != r.point, Flush,
// r.point = p, SetFilter(gfx.FilterPoint) if p else
// SetFilter(gfx.FilterLinear). A cache of 0xff (Reset) always differs.
func (r *Renderer) filter(vm *anm.VM) {
	var p uint8
	if vm.Flags&(1<<31) != 0 {
		p = 1
	}
	if p != r.point {
		r.Flush()
		r.point = p
		if p == 1 {
			r.Dev.SetFilter(gfx.FilterPoint)
		} else {
			r.Dev.SetFilter(gfx.FilterLinear)
		}
	}
}

// push is FUN_00442fe0: append P0, P1, P2, P1, P2, P3 of r.quad to the
// batch. The original's batch is a fixed buffer it never checks; the
// slice grows.
func (r *Renderer) push() {
	q := &r.quad
	r.batch = append(r.batch, q[0], q[1], q[2], q[1], q[2], q[3])
}
