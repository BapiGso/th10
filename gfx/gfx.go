// Package gfx emulates the slice of the Direct3D 9 fixed-function pipeline
// th10.exe draws with, on top of Ebiten: transforms and a viewport, linear
// fog, texture addressing and filtering, the blend modes the ANM and STD
// draw paths use, the alpha test, and a D16 depth buffer. Everything is drawn
// into one 640x480 image, the original's back buffer, in the original's
// submission order; the depth buffer is a second image of the same size.
// The design is docs/research/m4_render.md §2.1.
//
// Both submission paths end as the same screen-space vertices: DrawTL takes
// pre-transformed vertices (D3DFVF_XYZRHW), Draw3D transforms, clips against
// the near and far planes, divides and maps through the viewport on the CPU.
// One Kage shader rasterises them; see shader.go for its contract and Flush
// for how depth is tested and written.
package gfx

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Mat4 is a D3DMATRIX: row vectors, v' = v·M, translation in row 3.
type Mat4 = [4][4]float32

// Identity is the identity matrix.
var Identity = Mat4{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}, {0, 0, 0, 1}}

// Color is a D3DCOLOR, 0xAARRGGBB.
type Color = uint32

// Viewport is a D3DVIEWPORT9 in back-buffer pixels.
type Viewport struct {
	X, Y, W, H float32
	MinZ, MaxZ float32
}

// FullViewport is the whole back buffer.
var FullViewport = Viewport{0, 0, Width, Height, 0, 1}

// Back-buffer size.
const (
	Width  = 640
	Height = 480
)

// TLVertex is a pre-transformed vertex (D3DFVF_XYZRHW | DIFFUSE | SPECULAR
// | TEX1): screen position in D3D9 pixel coordinates (pixel centres on
// integers), depth, reciprocal w, colours and texture coordinates. With
// vertex fog the fog factor is the specular alpha, as in Direct3D.
type TLVertex struct {
	X, Y, Z, RHW      float32
	Diffuse, Specular Color
	U, V              float32
}

// Vertex3D is an untransformed vertex (D3DFVF_XYZ | DIFFUSE | TEX1).
type Vertex3D struct {
	X, Y, Z float32
	Diffuse Color
	U, V    float32
}

// Primitive is a D3DPRIMITIVETYPE.
type Primitive uint8

const (
	TriangleList  Primitive = 4
	TriangleStrip Primitive = 5
	TriangleFan   Primitive = 6
)

// TransformState names the matrix SetTransform replaces (D3DTS_*).
type TransformState uint16

const (
	View       TransformState = 2
	Projection TransformState = 3
	World      TransformState = 256
)

// FogMode is how fog is computed: per vertex (D3DRS_FOGVERTEXMODE) or per
// pixel (D3DRS_FOGTABLEMODE), linear in both cases.
type FogMode uint8

const (
	FogNone FogMode = iota
	FogVertexLinear
	FogTableLinear
)

// Fog is the fog render state: D3DRS_FOGCOLOR, FOGSTART, FOGEND and
// RANGEFOGENABLE (distance instead of view-space z, vertex fog only).
type Fog struct {
	Mode       FogMode
	Color      Color
	Start, End float32
	Range      bool
}

// BlendMode is a D3DRS_SRCBLEND/DESTBLEND pair. More are added as the draw
// paths need them (docs/research/m4_render_re.md R3).
type BlendMode uint8

const (
	BlendAlpha BlendMode = iota // SRCALPHA, INVSRCALPHA
	BlendAdd                    // SRCALPHA, ONE
)

// ebitenBlend is the Ebiten blend that gives b's result for the shader's
// premultiplied output on an opaque target: BlendAlpha is
// ebiten.BlendSourceOver (c·a + dst·(1−a)), BlendAdd is ebiten.BlendLighter
// (c·a + dst). Alpha stays 1 under both, which keeps Frame opaque.
func (b BlendMode) ebitenBlend() ebiten.Blend {
	if b == BlendAdd {
		return ebiten.BlendLighter
	}
	return ebiten.BlendSourceOver
}

// Cull is D3DRS_CULLMODE: which screen-space winding is dropped. It applies
// to both draw paths, after clipping. Strips alternate their triangles'
// vertex order (see triangles), so a whole strip has one winding.
type Cull uint8

const (
	CullNone Cull = 1 // D3DCULL_NONE
	CullCW   Cull = 2 // D3DCULL_CW
	CullCCW  Cull = 3 // D3DCULL_CCW, Direct3D's default
)

// CmpFunc is a D3DCMPFUNC, the depth test's comparison. th10 uses only
// these two (docs/research/m4_render_re.md §4.7).
type CmpFunc uint8

const (
	CmpLessEqual CmpFunc = 4 // D3DCMP_LESSEQUAL, Direct3D's default
	CmpAlways    CmpFunc = 8 // D3DCMP_ALWAYS
)

// ClearFlags says what Clear clears (D3DCLEAR_*).
type ClearFlags uint8

const (
	ClearTarget  ClearFlags = 1 // D3DCLEAR_TARGET
	ClearZBuffer ClearFlags = 2 // D3DCLEAR_ZBUFFER
)

// Address is a sampler address mode (D3DSAMP_ADDRESSU/V).
type Address uint8

const (
	AddressWrap Address = iota
	AddressClamp
)

// Filter is a sampler min/mag filter (D3DSAMP_MINFILTER/MAGFILTER).
type Filter uint8

const (
	FilterPoint Filter = iota
	FilterLinear
)

// Texture is one whole texture. Texture coordinates address it as a unit
// square; the shader wraps or clamps them itself, since Ebiten may pack the
// image into an atlas where hardware addressing would bleed into neighbours.
//
// Image holds straight-alpha texels, not Ebiten's premultiplied ones:
// Direct3D filters and modulates straight colours, and the RGB of a
// transparent texel shows at the edges of a bilinear-filtered sprite. Image
// is only ever read by the shader; drawing it with DrawImage gives wrong
// colours.
//
// Image is unmanaged, so it is never put on an atlas and its origin is
// (0, 0). Ebiten offsets the vertices' SrcX/SrcY by Images[0]'s origin, and
// SrcX carries depth (ebitenVertex): on an atlas the float32 sum would lose
// the precision a D16 depth needs.
type Texture struct {
	Image *ebiten.Image
	W, H  int
}

// NewTexture uploads img's pixels unchanged (WritePixels stores the bytes
// as given, so they stay straight alpha).
func NewTexture(img *image.NRGBA) *Texture {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := img.Pix[:4*w*h]
	if img.Stride != 4*w || b.Min != (image.Point{}) {
		pix = make([]byte, 4*w*h)
		for y := range h {
			o := img.PixOffset(b.Min.X, b.Min.Y+y)
			copy(pix[4*w*y:4*w*(y+1)], img.Pix[o:o+4*w])
		}
	}
	dst := newUnmanaged(w, h)
	dst.WritePixels(pix)
	return &Texture{Image: dst, W: w, H: h}
}

// newUnmanaged is a w x h image off Ebiten's atlas, with origin (0, 0).
func newUnmanaged(w, h int) *ebiten.Image {
	return ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
}

// state is everything a draw depends on. A draw whose state differs from
// the pending batch's flushes it first.
type state struct {
	vp       Viewport
	fog      Fog
	fogOn    bool // D3DRS_FOGENABLE
	blend    BlendMode
	tex      *Texture
	addr     Address
	filter   Filter
	zOn      bool    // D3DRS_ZENABLE
	zWrite   bool    // D3DRS_ZWRITEENABLE
	zFunc    CmpFunc // D3DRS_ZFUNC
	alphaRef uint8   // D3DRS_ALPHAREF, 0 while the alpha test is off (SetAlphaTest)
}

// fogState is the fog draws in s get: s.fog, with Mode FogNone unless
// FOGENABLE is set.
func (s state) fogState() Fog {
	f := s.fog
	if !s.fogOn {
		f.Mode = FogNone
	}
	return f
}

// depthTest reports whether s's fragments are depth-tested: ZENABLE with
// ZFUNC LESSEQUAL (ALWAYS passes every fragment).
func (s state) depthTest() bool { return s.zOn && s.zFunc == CmpLessEqual }

// depthWrite reports whether s's fragments that pass the alpha and depth
// tests write depth: ZENABLE with ZWRITEENABLE.
func (s state) depthWrite() bool { return s.zOn && s.zWrite }

// Device is the drawing state of a frame and the entry point for draws.
type Device struct {
	// Frame is the back buffer, Width x Height, unmanaged (see Flush).
	Frame *ebiten.Image

	// depth is the depth buffer, Width x Height and unmanaged like Frame:
	// pixel (x, y) holds back-buffer pixel (x, y)'s D16 value as depthColor
	// encodes it. Its contents are undefined until a ClearZBuffer, as in
	// Direct3D. snap is scratch of the same kind, the copy of depth that a
	// batch which both tests and writes depth is tested against (Flush).
	depth, snap *ebiten.Image

	world, view, proj Mat4
	cull              Cull
	cur               state

	batch   []ebiten.Vertex
	indices []uint32
	pending state // state of the vertices in batch
	// tris are the batch's triangles while pending both tests and writes
	// depth, for reserve's conflict check; empty otherwise.
	tris []screenTri

	// Scratch for Draw3D, kept across calls so steady-state draws allocate
	// nothing: toClip's output for every vertex, and one clipped polygon.
	clips, poly []clipVertex
}

// NewDevice returns a device drawing into a new back buffer.
func NewDevice() *Device {
	d := &Device{
		Frame: newUnmanaged(Width, Height),
		depth: newUnmanaged(Width, Height),
		snap:  newUnmanaged(Width, Height),
	}
	d.Reset()
	return d
}

// Reset restores Direct3D's defaults for a device with an automatic depth
// buffer: identity transforms, the full viewport, CCW culling, alpha
// blending, no texture, FOGENABLE off (the fog parameters zero), ZENABLE and
// ZWRITEENABLE on with ZFUNC LESSEQUAL, the alpha test off. It does not touch
// Frame or the depth buffer. th10's own initial state (FUN_00439d20,
// docs/research/m4_render_re.md §2.4) is the draw chain's to apply.
func (d *Device) Reset() {
	d.world, d.view, d.proj = Identity, Identity, Identity
	d.cull = CullCCW
	d.cur = state{vp: FullViewport, zOn: true, zWrite: true, zFunc: CmpLessEqual}
}

// SetTransform replaces a matrix.
func (d *Device) SetTransform(t TransformState, m Mat4) {
	switch t {
	case World:
		d.world = m
	case View:
		d.view = m
	case Projection:
		d.proj = m
	}
}

// SetViewport replaces the viewport.
func (d *Device) SetViewport(v Viewport) { d.cur.vp = v }

// SetCull replaces the cull mode.
func (d *Device) SetCull(c Cull) { d.cull = c }

// SetFog replaces the fog parameters. They take effect only while FOGENABLE
// is set (SetFogEnable): th10 toggles FOGENABLE per STD quad and leaves the
// parameters alone.
func (d *Device) SetFog(f Fog) { d.cur.fog = f }

// SetFogEnable sets D3DRS_FOGENABLE.
func (d *Device) SetFogEnable(on bool) { d.cur.fogOn = on }

// SetBlend replaces the blend mode.
func (d *Device) SetBlend(b BlendMode) { d.cur.blend = b }

// SetTexture binds t to stage 0, nil for untextured draws. The sampler
// state is separate (SetAddress, SetFilter), as in Direct3D: th10 rebinds
// textures without touching it.
func (d *Device) SetTexture(t *Texture) { d.cur.tex = t }

// SetAddress sets D3DSAMP_ADDRESSU and ADDRESSV of stage 0 to a.
func (d *Device) SetAddress(a Address) { d.cur.addr = a }

// SetFilter sets D3DSAMP_MAGFILTER and MINFILTER of stage 0 to f, as
// FUN_004425a0 does (it always sets the two together).
func (d *Device) SetFilter(f Filter) { d.cur.filter = f }

// SetZEnable sets D3DRS_ZENABLE. While it is off nothing is depth-tested or
// written, whatever ZWRITEENABLE and ZFUNC say.
func (d *Device) SetZEnable(on bool) { d.cur.zOn = on }

// SetZWrite sets D3DRS_ZWRITEENABLE.
func (d *Device) SetZWrite(on bool) { d.cur.zWrite = on }

// SetZFunc sets D3DRS_ZFUNC. Only CmpLessEqual and CmpAlways are emulated;
// any other value panics.
func (d *Device) SetZFunc(f CmpFunc) {
	if f != CmpLessEqual && f != CmpAlways {
		panic("gfx: unsupported ZFUNC")
	}
	d.cur.zFunc = f
}

// SetAlphaTest sets D3DRS_ALPHATESTENABLE and ALPHAREF, with ALPHAFUNC
// GREATEREQUAL, the only function th10 uses (docs/research/m4_render_re.md
// §2.4). A fragment whose alpha8 is below ref is discarded: it writes neither
// colour nor depth. GREATEREQUAL 0 passes every fragment, so a disabled test
// is kept as ref 0.
func (d *Device) SetAlphaTest(on bool, ref uint8) {
	if !on {
		ref = 0
	}
	d.cur.alphaRef = ref
}

// DrawTL draws pre-transformed primitives, IDirect3DDevice9::DrawPrimitiveUP
// with an XYZRHW vertex format: fromTL, cull, then into the batch. vs is not
// retained.
func (d *Device) DrawTL(p Primitive, vs []TLVertex) {
	triangles(p, len(vs), func(a, b, c int) {
		ra := fromTL(vs[a], d.cur.fog)
		rb := fromTL(vs[b], d.cur.fog)
		rc := fromTL(vs[c], d.cur.fog)
		if culled(d.cull, ra, rb, rc) {
			return
		}
		d.reserve(ra, rb, rc)
		d.batch = append(d.batch, ebitenVertex(ra), ebitenVertex(rb), ebitenVertex(rc))
	})
}

// reserve readies the batch for the triangle a, b, c in the current state.
// The pending batch is flushed first when
//
//   - its state differs from cur,
//   - three more vertices would overflow Ebiten's vertex limit, or
//   - cur both tests and writes depth and the triangle conflicts with one
//     in tris (conflicts).
//
// The last rule exists because such a batch is tested against the depth from
// before the batch (Flush), so none of its triangles may cover a pixel
// another one covers. Afterwards pending is cur and, while cur both tests and
// writes depth, the triangle's screen positions are appended to tris.
func (d *Device) reserve(a, b, c rasterVertex) {
	t := screenTri{{a.X, a.Y}, {b.X, b.Y}, {c.X, c.Y}}
	both := d.cur.depthTest() && d.cur.depthWrite()
	if len(d.batch) > 0 && (d.cur != d.pending || len(d.batch)+3 > ebiten.MaxVertexCount || (both && d.clashes(t))) {
		d.Flush()
	}
	d.pending = d.cur
	if both {
		d.tris = append(d.tris, t)
	}
}

// clashes reports whether t conflicts with a triangle of the pending batch.
func (d *Device) clashes(t screenTri) bool {
	for _, o := range d.tris {
		if conflicts(o, t) {
			return true
		}
	}
	return false
}

// Draw3D draws untransformed primitives through world·view·proj: toClip
// with the product taken once per call, then per triangle clipPoly, project,
// a fan over the clipped polygon, cull, and into the batch. vs is not
// retained.
func (d *Device) Draw3D(p Primitive, vs []Vertex3D) {
	wv := mul4(d.world, d.view)
	wvp := mul4(wv, d.proj)
	d.clips = d.clips[:0]
	for _, v := range vs {
		d.clips = append(d.clips, toClip(v, wvp, wv, d.cur.fog))
	}
	vp := d.cur.vp
	triangles(p, len(d.clips), func(a, b, c int) {
		src := [3]clipVertex{d.clips[a], d.clips[b], d.clips[c]}
		d.poly = clipPoly(d.poly[:0], src[:])
		if len(d.poly) < 3 {
			return
		}
		r0 := project(d.poly[0], vp)
		for k := 1; k+1 < len(d.poly); k++ {
			r1 := project(d.poly[k], vp)
			r2 := project(d.poly[k+1], vp)
			if culled(d.cull, r0, r1, r2) {
				continue
			}
			d.reserve(r0, r1, r2)
			d.batch = append(d.batch, ebitenVertex(r0), ebitenVertex(r1), ebitenVertex(r2))
		}
	})
}

// Clear is IDirect3DDevice9::Clear: each of rects (back-buffer pixels)
// intersected with the viewport is cleared, or the viewport itself when
// rects is empty. ClearTarget fills Frame with c made opaque: the back
// buffer (X8R8G8B8) has no alpha, and keeping Frame opaque is what lets the
// blend modes in shader.go match Direct3D's. ClearZBuffer fills the depth
// buffer with depthColor(depth16(z)). The pending batch is flushed first.
func (d *Device) Clear(f ClearFlags, c Color, z float32, rects ...image.Rectangle) {
	d.Flush()
	vp := viewportRect(d.cur.vp)
	if len(rects) == 0 {
		rects = []image.Rectangle{vp}
	}
	dv := depth16(z)
	for _, r := range rects {
		r = r.Intersect(vp)
		if r.Empty() {
			continue
		}
		if f&ClearTarget != 0 {
			if dst := sub(d.Frame, r); dst != nil {
				dst.Fill(color.RGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 255})
			}
		}
		if f&ClearZBuffer != 0 {
			// depthColor(dv) as bytes: d>>8 and d&0xff, opaque.
			if dst := sub(d.depth, r); dst != nil {
				dst.Fill(color.RGBA{R: uint8(dv >> 8), G: uint8(dv), A: 255})
			}
		}
	}
}

// viewportRect is the pixels vp covers.
func viewportRect(vp Viewport) image.Rectangle {
	return image.Rect(int(vp.X), int(vp.Y), int(vp.X+vp.W), int(vp.Y+vp.H))
}

// sub is the part of img that r covers, or nil if it covers nothing. SubImage
// keeps img's coordinates, so drawing into it is scissored to r.
func sub(img *ebiten.Image, r image.Rectangle) *ebiten.Image {
	if r.Empty() {
		return nil
	}
	s, _ := img.SubImage(r).(*ebiten.Image)
	return s
}

// target is the part of img (Frame, depth or snap) that vp covers, or nil if
// it covers nothing. SubImage keeps img's coordinates, so drawing into it is
// scissored to vp.
func (d *Device) target(img *ebiten.Image, vp Viewport) *ebiten.Image {
	return sub(img, viewportRect(vp))
}

// Flush submits the pending batch. With s the batch's state, in this order:
//
//  1. If s both tests and writes depth: copy depth into snap over s.vp.
//  2. Colour pass: one DrawTrianglesShader32 onto target(Frame, s.vp) with
//     s.blend, Images (s.tex, depth) and uniformsOf(s).
//  3. If s writes depth: depth pass, the same vertices and indices onto
//     target(depth, s.vp) with ebiten.BlendCopy, Images (s.tex, snap) and
//     uniformsOf(s) with Pass 1. Every fragment that passes the alpha and
//     depth tests stores depthColor of its depth; the others are discarded
//     and leave depth alone.
//
// The target scissors to the viewport, as Direct3D does. Both passes test
// against the depth from before the batch: the colour pass reads depth
// before the depth pass writes it, and the depth pass reads the copy (Ebiten
// cannot read the image it draws to). That gives Direct3D's per-triangle
// result because reserve keeps such a batch free of triangles that cover a
// common pixel. A batch that writes depth without testing it (ZFUNC ALWAYS)
// needs neither the copy nor the rule: BlendCopy keeps the last fragment
// drawn, as Direct3D does.
//
// Frame, depth and snap are unmanaged, so their origin is (0, 0) and a
// fragment's dstPos is its back-buffer pixel in all three: the shader reads
// Images[1] at dstPos.
//
// Since a draw in another state flushes first, submission order is always
// the original's draw order. Call Flush before reading Frame.
func (d *Device) Flush() {
	d.tris = d.tris[:0]
	if len(d.batch) == 0 {
		return
	}
	s := d.pending
	n := len(d.batch)
	for len(d.indices) < n {
		d.indices = append(d.indices, uint32(len(d.indices)))
	}
	indices := d.indices[:n]
	if s.depthTest() && s.depthWrite() {
		d.copyDepth(s.vp)
	}
	u := uniformsOf(s)
	if dst := d.target(d.Frame, s.vp); dst != nil {
		d.submit(dst, u, s.blend.ebitenBlend(), s.tex, d.depth, indices)
	}
	if s.depthWrite() {
		if dst := d.target(d.depth, s.vp); dst != nil {
			u.Pass = 1
			d.submit(dst, u, ebiten.BlendCopy, s.tex, d.snap, indices)
		}
	}
	d.batch = d.batch[:0]
}

// copyDepth copies depth into snap over vp. DrawImage draws a sub-image from
// the destination's origin, not from its own position, so the GeoM moves the
// copy back to vp's top left, where it came from.
func (d *Device) copyDepth(vp Viewport) {
	// SubImage clips to the back buffer, so the translation must use the
	// clipped rectangle's origin too.
	r := viewportRect(vp).Intersect(image.Rect(0, 0, Width, Height))
	dst, src := sub(d.snap, r), sub(d.depth, r)
	if dst == nil || src == nil {
		return
	}
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	dst.DrawImage(src, op)
}

// submit draws the batch's vertices onto dst with u, the texture in Images[0]
// and second in Images[1]. The shader reads Images[1] only to test depth, so it
// matters only when u.DepthTest is set.
func (d *Device) submit(dst *ebiten.Image, u uniforms, blend ebiten.Blend, tex *Texture, second *ebiten.Image, indices []uint32) {
	opts := &ebiten.DrawTrianglesShaderOptions{
		Blend:  blend,
		Images: [4]*ebiten.Image{nil, second},
		Uniforms: map[string]any{
			"Textured":  u.Textured,
			"Address":   u.Address,
			"Filter":    u.Filter,
			"FogMode":   u.FogMode,
			"FogColor":  u.FogColor,
			"FogStart":  u.FogStart,
			"FogEnd":    u.FogEnd,
			"AlphaRef":  u.AlphaRef,
			"DepthTest": u.DepthTest,
			"Pass":      u.Pass,
		},
	}
	if tex != nil {
		opts.Images[0] = tex.Image
	}
	dst.DrawTrianglesShader32(d.batch, indices, compiledShader(), opts)
}
