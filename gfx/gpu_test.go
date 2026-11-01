//go:build gpu

package gfx

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"th10/std"

	"github.com/hajimehoshi/ebiten/v2"
)

// The GPU tests draw through a real Ebiten device and compare the frame read
// back with a CPU reference. They need a display, so they sit behind the gpu
// build tag: go test -tags gpu -count=1 ./gfx/.
//
// TestMain starts the tests on a goroutine while ebiten.RunGame owns the main
// thread, and ends the game when they finish.

var (
	gpuM    *testing.M
	gpuDone = make(chan int, 1)
	gpuCode int
)

type gpuHarness struct{ started bool }

func (h *gpuHarness) Update() error {
	if !h.started {
		h.started = true
		go func() { gpuDone <- gpuM.Run() }()
	}
	select {
	case code := <-gpuDone:
		gpuCode = code
		return ebiten.Termination
	default:
		return nil
	}
}

func (h *gpuHarness) Draw(*ebiten.Image) {}

func (h *gpuHarness) Layout(int, int) (int, int) { return Width, Height }

func TestMain(m *testing.M) {
	gpuM = m
	if err := ebiten.RunGame(&gpuHarness{}); err != nil {
		fmt.Fprintln(os.Stderr, "gfx gpu tests:", err)
		os.Exit(1)
	}
	os.Exit(gpuCode)
}

// testBG is the opaque background every test clears to.
const testBG Color = 0xff203040

// refTri is one triangle as the GPU is given it: its screen vertices, the
// state it was drawn in, and the CPU copy of its texture.
type refTri struct {
	v   [3]ebiten.Vertex
	st  state
	src *image.NRGBA
}

// trace records the triangles a test submits, in submission order. The
// reference rasterises them one at a time, so it does not depend on how the
// device batches them. The triangles are rebuilt here from the same CPU
// pipeline (fromTL or the Draw3D path, culled, ebitenVertex), without the
// device's batching.
type trace struct {
	d    *Device
	tris []refTri
}

func newTrace(d *Device) *trace { return &trace{d: d} }

// tl draws TL vertices with d and records their triangles, textured from src
// (nil when untextured).
func (tr *trace) tl(src *image.NRGBA, p Primitive, vs []TLVertex) {
	tr.d.DrawTL(p, vs)
	fog := tr.d.cur.fog
	triangles(p, len(vs), func(a, b, c int) {
		tr.add(src, fromTL(vs[a], fog), fromTL(vs[b], fog), fromTL(vs[c], fog))
	})
}

// draw3D draws untransformed vertices with d and records their triangles.
func (tr *trace) draw3D(src *image.NRGBA, p Primitive, vs []Vertex3D) {
	d := tr.d
	d.Draw3D(p, vs)
	wv := mul4(d.world, d.view)
	wvp := mul4(wv, d.proj)
	clips := make([]clipVertex, len(vs))
	for i, v := range vs {
		clips[i] = toClip(v, wvp, wv, d.cur.fog)
	}
	triangles(p, len(clips), func(a, b, c int) {
		poly := clipPoly(nil, []clipVertex{clips[a], clips[b], clips[c]})
		if len(poly) < 3 {
			return
		}
		r0 := project(poly[0], d.cur.vp)
		for k := 1; k+1 < len(poly); k++ {
			tr.add(src, r0, project(poly[k], d.cur.vp), project(poly[k+1], d.cur.vp))
		}
	})
}

// add records a screen-space triangle, unless the device's cull mode drops it.
func (tr *trace) add(src *image.NRGBA, a, b, c rasterVertex) {
	if culled(tr.d.cull, a, b, c) {
		return
	}
	tr.tris = append(tr.tris, refTri{
		v:   [3]ebiten.Vertex{ebitenVertex(a), ebitenVertex(b), ebitenVertex(c)},
		st:  tr.d.cur,
		src: src,
	})
}

// newDevice returns a device whose frame is cleared to testBG and whose depth
// buffer is cleared to 1, the far plane. Depth state is Reset's (ZENABLE,
// ZWRITEENABLE, LESSEQUAL), so the depth path runs in every case, but with
// depth 1 nothing is rejected: the cases that are not about depth are
// unaffected by it.
func newDevice(t *testing.T) *Device {
	t.Helper()
	d := NewDevice()
	d.Clear(ClearTarget|ClearZBuffer, testBG, 1)
	return d
}

// readBack flushes d and returns Frame's pixels, premultiplied RGBA.
func readBack(d *Device) []byte {
	d.Flush()
	pix := make([]byte, 4*Width*Height)
	d.Frame.ReadPixels(pix)
	return pix
}

// readDepth flushes d and returns its depth buffer as D16 values, decoded from
// the stored bytes (high byte R, low byte G) as the shader decodes them.
func readDepth(d *Device) []uint16 {
	d.Flush()
	pix := make([]byte, 4*Width*Height)
	d.depth.ReadPixels(pix)
	dep := make([]uint16, Width*Height)
	for i := range dep {
		dep[i] = uint16(pix[4*i])<<8 | uint16(pix[4*i+1])
	}
	return dep
}

// quadTL is the TL quad covering the Ebiten rectangle [x0,x1) x [y0,y1) with
// ZUN's −0.5 convention: its corners sit at (x−0.5, y−0.5), so the integer
// D3D pixel centres fall on the rectangle's pixels. uv runs from (u0, v0) at
// the top left to (u1, v1) at the bottom right.
func quadTL(x0, y0, x1, y1, u0, v0, u1, v1 float32, diffuse Color) []TLVertex {
	v := func(x, y, u, w float32) TLVertex {
		return TLVertex{X: x - 0.5, Y: y - 0.5, Z: 0, RHW: 1, Diffuse: diffuse, U: u, V: w}
	}
	tl := v(x0, y0, u0, v0)
	tr := v(x1, y0, u1, v0)
	br := v(x1, y1, u1, v1)
	bl := v(x0, y1, u0, v1)
	return []TLVertex{tl, tr, br, tl, br, bl}
}

// depthFn is a depth from a vertex's screen position.
type depthFn func(x, y float32) float32

// withDepth sets each vertex's depth to f of its position.
func withDepth(vs []TLVertex, f depthFn) []TLVertex {
	for i := range vs {
		vs[i].Z = f(vs[i].X, vs[i].Y)
	}
	return vs
}

// flat is a constant depth.
func flat(z float32) depthFn { return func(x, y float32) float32 { return z } }

// ramp is a depth that runs linearly in x from z0 at x0 to z1 at x1.
func ramp(x0, x1, z0, z1 float32) depthFn {
	return func(x, y float32) float32 { return z0 + float32((z1-z0)*(x-x0)/(x1-x0)) }
}

// distinctTexture is a w x h texture whose texels all differ, with alpha 160
// on the odd checkerboard squares so that blending shows.
func distinctTexture(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if (x+y)%2 == 1 {
				a = 160
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(40 + x*180/(w-1)),
				G: uint8(40 + y*180/(h-1)),
				B: uint8(220 - (x+y)*8),
				A: a,
			})
		}
	}
	return img
}

// holeTexture is w x h: texels left of w/2 have alpha 0, the rest alpha 255,
// and the colours all differ.
func holeTexture(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if x < w/2 {
				a = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(30*x + 10), G: uint8(30*y + 10), B: 128, A: a})
		}
	}
	return img
}

// rgbOf is a colour's RGB as 0..1 floats.
func rgbOf(c Color) [3]float32 {
	return [3]float32{float32(c>>16&0xff) / 255, float32(c>>8&0xff) / 255, float32(c&0xff) / 255}
}

// rgbInts is a colour's RGB as 0..255 integers, for wantPixel.
func rgbInts(c Color) [3]int {
	return [3]int{int(c >> 16 & 0xff), int(c >> 8 & 0xff), int(c & 0xff)}
}

// expect is the frame the GPU should show: per pixel the opaque RGB in 0..1,
// whether the pixel is checked, and how many checked pixels a triangle covers.
// depth is the D16 buffer the pixels should hold, compared where ok (verifyDepth).
type expect struct {
	rgb     [][3]float32
	ok      []bool
	covered int
	depth   []uint16
}

func newExpect() expect {
	return expect{rgb: make([][3]float32, Width*Height), ok: make([]bool, Width*Height)}
}

// edgeFn is twice the signed area of (a, b, p), positive when p lies to the
// left of a→b in screen space (y down, so the sign matches culled's).
func edgeFn(a, b ebiten.Vertex, px, py float64) float64 {
	ax, ay := float64(a.DstX), float64(a.DstY)
	return (float64(b.DstX)-ax)*(py-ay) - (float64(b.DstY)-ay)*(px-ax)
}

// nearEdge reports whether p is within 1 px of an edge of any triangle.
func nearEdge(vs []ebiten.Vertex, p [2]float64) bool {
	for t := 0; t+2 < len(vs); t += 3 {
		for k := range 3 {
			a, b := vs[t+k], vs[t+(k+1)%3]
			l := math.Hypot(float64(b.DstX-a.DstX), float64(b.DstY-a.DstY))
			if l == 0 {
				continue
			}
			if math.Abs(edgeFn(a, b, p[0], p[1]))/l < 1 {
				return true
			}
		}
	}
	return false
}

// nearInt reports whether x is within 1e-3 of an integer.
func nearInt(x float64) bool { return math.Abs(x-math.Round(x)) < 1e-3 }

// nearRounding reports whether z's D16 value lies within 0.05 of a rounding
// boundary. The GPU interpolates z to about 1e-7, far below that margin, so a
// fragment this close to a boundary might be stored one D16 step away.
func nearRounding(z float32) bool {
	q := float64(z) * 65535
	f := q - math.Floor(q)
	return f > 0.45 && f < 0.55
}

// interp returns the colour and Custom0..3 of a point with barycentric
// weights w over the triangle vs, interpolated screen-linearly as the GPU does.
func interp(vs [3]ebiten.Vertex, w [3]float64) (col, cu [4]float32) {
	col4 := func(v ebiten.Vertex) [4]float32 { return [4]float32{v.ColorR, v.ColorG, v.ColorB, v.ColorA} }
	cu4 := func(v ebiten.Vertex) [4]float32 { return [4]float32{v.Custom0, v.Custom1, v.Custom2, v.Custom3} }
	c0, c1, c2 := col4(vs[0]), col4(vs[1]), col4(vs[2])
	u0, u1, u2 := cu4(vs[0]), cu4(vs[1]), cu4(vs[2])
	for k := range 4 {
		col[k] = float32(w[0]*float64(c0[k]) + w[1]*float64(c1[k]) + w[2]*float64(c2[k]))
		cu[k] = float32(w[0]*float64(u0[k]) + w[1]*float64(u1[k]) + w[2]*float64(u2[k]))
	}
	return col, cu
}

// triRect is the pixels whose centres lie in the bounding box of vs, grown by
// g pixels and clipped to the back buffer.
func triRect(vs [3]ebiten.Vertex, g int) image.Rectangle {
	x0, y0 := math.Inf(1), math.Inf(1)
	x1, y1 := math.Inf(-1), math.Inf(-1)
	for _, v := range vs {
		x0 = min(x0, float64(v.DstX))
		y0 = min(y0, float64(v.DstY))
		x1 = max(x1, float64(v.DstX))
		y1 = max(y1, float64(v.DstY))
	}
	r := image.Rect(int(math.Floor(x0))-g, int(math.Floor(y0))-g, int(math.Ceil(x1))+g, int(math.Ceil(y1))+g)
	return r.Intersect(image.Rect(0, 0, Width, Height))
}

// reference is what Direct3D draws for tr, on the CPU. Each triangle is
// rasterised in submission order over its own viewport. A covered pixel is
// shaded by fragment, whose alpha and depth tests see the D16 buffer as the
// earlier triangles left it (depth0 at the start). A kept fragment is
// composited over the frame (bg at the start) with its blend mode, and where
// its state writes depth, the depth pass's output is what the buffer stores.
//
// A pixel is not checked when it is within 1 px of an edge of any triangle,
// when a covering fragment's point-sampled texel coordinate is near an
// integer, or when a covering fragment's depth is near a rounding boundary
// (nearRounding). Such a pixel is unchecked for every later triangle too.
func reference(tr *trace, bg Color, depth0 uint16) expect {
	n := Width * Height
	ex := newExpect()
	ex.depth = make([]uint16, n)
	bgRGB := rgbOf(bg)
	dst := make([][3]float32, n)
	dep := make([]uint16, n)
	hit := make([]bool, n)
	for i := range n {
		dst[i], dep[i], ex.ok[i] = bgRGB, depth0, true
	}
	for _, tri := range tr.tris {
		r := triRect(tri.v, 1)
		for py := r.Min.Y; py < r.Max.Y; py++ {
			for px := r.Min.X; px < r.Max.X; px++ {
				if nearEdge(tri.v[:], [2]float64{float64(px) + 0.5, float64(py) + 0.5}) {
					ex.ok[py*Width+px] = false
				}
			}
		}
	}
	for _, tri := range tr.tris {
		vs := tri.v
		if edgeFn(vs[0], vs[1], float64(vs[2].DstX), float64(vs[2].DstY)) == 0 {
			continue
		}
		u0 := uniformsOf(tri.st)
		u1 := u0
		u1.Pass = 1
		area := edgeFn(vs[0], vs[1], float64(vs[2].DstX), float64(vs[2].DstY))
		r := triRect(vs, 0).Intersect(viewportRect(tri.st.vp))
		for py := r.Min.Y; py < r.Max.Y; py++ {
			for px := r.Min.X; px < r.Max.X; px++ {
				i := py*Width + px
				if !ex.ok[i] {
					continue
				}
				p := [2]float64{float64(px) + 0.5, float64(py) + 0.5}
				w := [3]float64{
					edgeFn(vs[1], vs[2], p[0], p[1]) / area,
					edgeFn(vs[2], vs[0], p[0], p[1]) / area,
					edgeFn(vs[0], vs[1], p[0], p[1]) / area,
				}
				if w[0] < 0 || w[1] < 0 || w[2] < 0 {
					continue
				}
				col, cu := interp(vs, w)
				z := float32(w[0]*float64(vs[0].SrcX) + w[1]*float64(vs[1].SrcX) + w[2]*float64(vs[2].SrcX))
				if (tri.st.depthTest() || tri.st.depthWrite()) && nearRounding(z) {
					ex.ok[i] = false
					continue
				}
				if u0.Textured == 1 && tri.st.filter == FilterPoint {
					s := float64(cu[0]/cu[2]) * float64(tri.src.Rect.Dx())
					t := float64(cu[1]/cu[2]) * float64(tri.src.Rect.Dy())
					if nearInt(s) || nearInt(t) {
						ex.ok[i] = false
						continue
					}
				}
				in := fragIn{Diffuse: col, UQ: cu[0], VQ: cu[1], Q: cu[2], FQ: cu[3], Z: z, Stored: dep[i]}
				out, keep := fragment(u0, tri.src, in)
				dout, keepDepth := fragment(u1, tri.src, in)
				if keep != keepDepth {
					panic("gfx: reference: colour and depth passes disagree")
				}
				if !keep {
					continue
				}
				hit[i] = true
				for k := range 3 {
					if tri.st.blend == BlendAdd {
						dst[i][k] = min(out[k]+dst[i][k], 1)
					} else {
						dst[i][k] = out[k] + dst[i][k]*(1-out[3])
					}
				}
				if tri.st.depthWrite() {
					dep[i] = uint16(math.Round(float64(dout[0])*255))<<8 | uint16(math.Round(float64(dout[1])*255))
				}
			}
		}
	}
	for i := range n {
		ex.depth[i] = dep[i]
		if !ex.ok[i] {
			continue
		}
		ex.rgb[i] = dst[i]
		if hit[i] {
			ex.covered++
		}
	}
	return ex
}

// verify compares pix with ex. Each checked pixel must match in R, G and B
// within 2/255, and be opaque. At least minCovered checked pixels must lie in
// a triangle, so an empty draw cannot pass.
func verify(t *testing.T, name string, pix []byte, ex expect, minCovered int) {
	t.Helper()
	checked, bad := 0, 0
	var worst float32
	for i := range ex.ok {
		if !ex.ok[i] {
			continue
		}
		checked++
		px, py := i%Width, i/Width
		for k := range 3 {
			got := float32(pix[4*i+k]) / 255
			d := got - ex.rgb[i][k]
			worst = max(worst, d, -d)
			if d > 2.0/255 || -d > 2.0/255 {
				bad++
				if bad <= 5 {
					t.Errorf("%s: pixel (%d,%d) channel %d = %d/255, want %.1f/255",
						name, px, py, k, pix[4*i+k], ex.rgb[i][k]*255)
				}
			}
		}
		if pix[4*i+3] != 255 {
			bad++
			if bad <= 5 {
				t.Errorf("%s: pixel (%d,%d) alpha %d, want 255", name, px, py, pix[4*i+3])
			}
		}
	}
	t.Logf("%s: %d pixels checked, %d covered, %d mismatches, worst deviation %.2f/255",
		name, checked, ex.covered, bad, worst*255)
	if checked == 0 {
		t.Fatalf("%s: no pixels checked", name)
	}
	if ex.covered < minCovered {
		t.Fatalf("%s: %d covered pixels, want at least %d", name, ex.covered, minCovered)
	}
	if bad > 0 {
		t.Errorf("%s: %d channel mismatches", name, bad)
	}
}

// verifyDepth compares the depth buffer dep with ex.depth exactly, on the
// checked pixels.
func verifyDepth(t *testing.T, name string, dep []uint16, ex expect) {
	t.Helper()
	bad := 0
	for i := range ex.ok {
		if !ex.ok[i] || dep[i] == ex.depth[i] {
			continue
		}
		bad++
		if bad <= 5 {
			t.Errorf("%s: pixel (%d,%d) depth %d, want %d", name, i%Width, i/Width, dep[i], ex.depth[i])
		}
	}
	if bad > 0 {
		t.Errorf("%s: %d depth mismatches", name, bad)
	}
}

// pixelAt is Frame's pixel (x, y) from a readBack.
func pixelAt(pix []byte, x, y int) [4]byte {
	o := 4 * (y*Width + x)
	return [4]byte{pix[o], pix[o+1], pix[o+2], pix[o+3]}
}

// wantPixel fails unless pixel (x, y) matches rgb (0..255) within 2.
func wantPixel(t *testing.T, name string, pix []byte, x, y int, rgb [3]int) {
	t.Helper()
	got := pixelAt(pix, x, y)
	for k := range 3 {
		if d := int(got[k]) - rgb[k]; d > 2 || d < -2 {
			t.Errorf("%s: pixel (%d,%d) = %v, want rgb %v", name, x, y, got, rgb)
			return
		}
	}
}

// Case 1: an untextured TL triangle with a solid diffuse colour.
func TestGPUSolidTriangle(t *testing.T) {
	d := newDevice(t)
	tr := newTrace(d)
	const col Color = 0xffc08040
	tr.tl(nil, TriangleList, []TLVertex{
		{X: 100, Y: 100, RHW: 1, Diffuse: col},
		{X: 300, Y: 120, RHW: 1, Diffuse: col},
		{X: 150, Y: 300, RHW: 1, Diffuse: col},
	})
	verify(t, "solid", readBack(d), reference(tr, testBG, depth16(1)), 10000)
}

// Case 2: a 4x4 textured TL quad drawn 1:1 with point filtering. Every
// covered pixel is its texel, composited SourceOver over the background, and
// every pixel outside the quad is the background.
func TestGPUTexturedQuad1to1(t *testing.T) {
	img := distinctTexture(4, 4)
	tex := NewTexture(img)
	d := newDevice(t)
	d.SetTexture(tex)
	d.SetAddress(AddressClamp)
	d.SetFilter(FilterPoint)
	const x0, y0 = 40, 60
	d.DrawTL(TriangleList, quadTL(x0, y0, x0+4, y0+4, 0, 0, 1, 1, 0xffffffff))
	pix := readBack(d)

	bgRGB := rgbOf(testBG)
	ex := newExpect()
	for y := range Height {
		for x := range Width {
			i := y*Width + x
			ex.rgb[i], ex.ok[i] = bgRGB, true
			if x < x0 || x >= x0+4 || y < y0 || y >= y0+4 {
				continue
			}
			c := img.NRGBAAt(x-x0, y-y0)
			a := float32(c.A) / 255
			for k, v := range [3]uint8{c.R, c.G, c.B} {
				ex.rgb[i][k] = float32(v)/255*a + bgRGB[k]*(1-a)
			}
			ex.covered++
		}
	}
	// The reference is not used here: every pixel of a 4 px quad lies within
	// 1 px of an edge, so it would check nothing. This explicit texel check
	// is the test.
	verify(t, "1:1 point", pix, ex, 16)
}

// Case 3: a TL quad magnified 2x (16 px over 8 texels) with linear filtering
// and wrapping: uv runs 0..2, so the texture repeats twice.
func TestGPUMagnifiedLinearWrap(t *testing.T) {
	img := distinctTexture(4, 4)
	d := newDevice(t)
	tr := newTrace(d)
	d.SetTexture(NewTexture(img))
	d.SetAddress(AddressWrap)
	d.SetFilter(FilterLinear)
	tr.tl(img, TriangleList, quadTL(100, 100, 116, 116, 0, 0, 2, 2, 0xffe0c0a0))
	// About 40 of the 196 interior pixels lie within 1 px of the diagonal
	// shared by the two triangles, so the reference checks 156.
	verify(t, "magnified linear wrap", readBack(d), reference(tr, testBG, depth16(1)), 100)
}

// Case 4: a Draw3D perspective quad, textured with linear filtering and vertex
// fog, transformed by a std.Camera and the playfield viewport. A table-fog
// variant covers the shader's per-pixel fog path.
func TestGPUPerspectiveFog(t *testing.T) {
	cam := std.Camera{
		Pos:    [3]float32{0, -300, 0},
		Facing: [3]float32{0, 300, 0},
		Up:     [3]float32{0, 0, 1},
		FOV:    0.7,
	}
	view, proj := cam.Matrices()
	img := distinctTexture(8, 8)
	quad := []Vertex3D{
		{X: -80, Y: -40, Z: -60, Diffuse: 0xffffffff, U: 0, V: 0},
		{X: 80, Y: 40, Z: -60, Diffuse: 0xffffffff, U: 1, V: 0},
		{X: 80, Y: 40, Z: 60, Diffuse: 0xffffffff, U: 1, V: 1},
		{X: -80, Y: -40, Z: 60, Diffuse: 0xffffffff, U: 0, V: 1},
	}
	tris := []Vertex3D{quad[0], quad[1], quad[2], quad[0], quad[2], quad[3]}

	for _, tc := range []struct {
		name string
		fog  Fog
	}{
		{"vertex fog", Fog{Mode: FogVertexLinear, Color: 0xff102060, Start: 200, End: 400}},
		{"table fog", Fog{Mode: FogTableLinear, Color: 0xff102060, Start: 200, End: 400}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tex := NewTexture(img)
			d := newDevice(t)
			tr := newTrace(d)
			d.SetTransform(World, Identity)
			d.SetTransform(View, view)
			d.SetTransform(Projection, proj)
			d.SetViewport(Viewport{X: std.Viewport.X, Y: std.Viewport.Y, W: std.Viewport.W, H: std.Viewport.H, MinZ: 0, MaxZ: 1})
			d.SetTexture(tex)
			d.SetAddress(AddressWrap)
			d.SetFilter(FilterLinear)
			d.SetFog(tc.fog)
			d.SetFogEnable(true)
			tr.draw3D(img, TriangleList, tris)
			verify(t, "perspective "+tc.name, readBack(d), reference(tr, testBG, depth16(1)), 1000)
		})
	}
}

// Case 5: a Draw3D quad that crosses the near plane. Its part behind the
// camera must not be drawn. The reference is built by the trace's own clip,
// so a stray behind-camera triangle from the device shows up as pixels the
// reference does not expect, and the device's batch is compared vertex by
// vertex with the trace.
func TestGPUNearPlaneClip(t *testing.T) {
	cam := std.Camera{
		Pos:    [3]float32{0, -300, 150},
		Facing: [3]float32{0, 300, -150},
		Up:     [3]float32{0, 0, 1},
		FOV:    0.9,
	}
	view, proj := cam.Matrices()
	a := Vertex3D{X: -100, Y: -400, Z: 0, Diffuse: 0xffa0c0e0}
	b := Vertex3D{X: 100, Y: -400, Z: 0, Diffuse: 0xffa0c0e0}
	c := Vertex3D{X: 100, Y: 0, Z: 0, Diffuse: 0xffa0c0e0}
	dv := Vertex3D{X: -100, Y: 0, Z: 0, Diffuse: 0xffa0c0e0}

	d := newDevice(t)
	tr := newTrace(d)
	d.SetTransform(View, view)
	d.SetTransform(Projection, proj)
	d.SetViewport(Viewport{X: std.Viewport.X, Y: std.Viewport.Y, W: std.Viewport.W, H: std.Viewport.H, MinZ: 0, MaxZ: 1})
	tr.draw3D(nil, TriangleList, []Vertex3D{a, b, c, a, c, dv})
	// Both triangles are in one batch (their common edge does not conflict),
	// so the batch must hold exactly the trace's vertices, in order. Every one
	// must have 1/w > 0 (Custom2), i.e. lie in front of the near plane.
	if len(d.batch) != 3*len(tr.tris) {
		t.Fatalf("batch has %d vertices, trace %d", len(d.batch), 3*len(tr.tris))
	}
	for i, v := range d.batch {
		if v != tr.tris[i/3].v[i%3] {
			t.Fatalf("batch vertex %d = %+v, trace has %+v", i, v, tr.tris[i/3].v[i%3])
		}
		if v.Custom2 <= 0 {
			t.Fatalf("batch vertex %d has 1/w = %v: emitted from behind the camera", i, v.Custom2)
		}
	}
	verify(t, "near plane", readBack(d), reference(tr, testBG, depth16(1)), 1000)
}

// Case 6: a TL triangle larger than a small viewport writes nothing outside
// the viewport rectangle.
func TestGPUViewportScissor(t *testing.T) {
	d := newDevice(t)
	tr := newTrace(d)
	d.SetViewport(Viewport{X: 100, Y: 80, W: 120, H: 60, MinZ: 0, MaxZ: 1})
	const col Color = 0xffc0ff40
	tr.tl(nil, TriangleList, []TLVertex{
		{X: -10, Y: -10, RHW: 1, Diffuse: col},
		{X: 700, Y: -10, RHW: 1, Diffuse: col},
		{X: -10, Y: 700, RHW: 1, Diffuse: col},
	})
	pix := readBack(d)
	verify(t, "viewport scissor", pix, reference(tr, testBG, depth16(1)), 7000)
	wantPixel(t, "viewport scissor", pix, 99, 100, [3]int{0x20, 0x30, 0x40})
	wantPixel(t, "viewport scissor", pix, 100, 80, [3]int{0xc0, 0xff, 0x40})
	wantPixel(t, "viewport scissor", pix, 219, 139, [3]int{0xc0, 0xff, 0x40})
	wantPixel(t, "viewport scissor", pix, 220, 139, [3]int{0x20, 0x30, 0x40})
	wantPixel(t, "viewport scissor", pix, 219, 140, [3]int{0x20, 0x30, 0x40})
}

// Case 7: two overlapping quads with BlendAdd add up. Overlap is bg + A + B,
// clamped at 1.
func TestGPUAdditiveBlend(t *testing.T) {
	d := newDevice(t)
	tr := newTrace(d)
	d.SetBlend(BlendAdd)
	tr.tl(nil, TriangleList, quadTL(50, 50, 150, 150, 0, 0, 1, 1, 0xff643200))
	tr.tl(nil, TriangleList, quadTL(100, 100, 200, 200, 0, 0, 1, 1, 0xff003c78))
	pix := readBack(d)
	verify(t, "additive", pix, reference(tr, testBG, depth16(1)), 15000)
	// bg (32,48,64) + (100,50,0) + (0,60,120).
	wantPixel(t, "additive overlap", pix, 120, 120, [3]int{132, 158, 184})
	// Only A: bg + (100,50,0).
	wantPixel(t, "additive A only", pix, 70, 70, [3]int{132, 98, 64})
}

// Case 8: culling. The triangle wound clockwise on screen (positive winding)
// is drawn and the counter-clockwise one dropped under the default CullCCW;
// CullNone draws both and CullCW the other way round.
func TestGPUCulling(t *testing.T) {
	const red, green Color = 0xffff0000, 0xff00ff00
	// CW: positive winding. CCW: negative winding.
	cw := []TLVertex{
		{X: 100, Y: 100, RHW: 1, Diffuse: red},
		{X: 200, Y: 100, RHW: 1, Diffuse: red},
		{X: 100, Y: 200, RHW: 1, Diffuse: red},
	}
	ccw := []TLVertex{
		{X: 300, Y: 100, RHW: 1, Diffuse: green},
		{X: 300, Y: 200, RHW: 1, Diffuse: green},
		{X: 400, Y: 100, RHW: 1, Diffuse: green},
	}
	bgRGB := [3]int{0x20, 0x30, 0x40}
	for _, tc := range []struct {
		name            string
		cull            Cull
		drawCW, drawCCW bool
	}{
		{"CullCCW", CullCCW, true, false},
		{"CullNone", CullNone, true, true},
		{"CullCW", CullCW, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newDevice(t)
			d.SetCull(tc.cull)
			d.DrawTL(TriangleList, cw)
			d.DrawTL(TriangleList, ccw)
			pix := readBack(d)
			wantCW, wantCCW := bgRGB, bgRGB
			if tc.drawCW {
				wantCW = [3]int{0xff, 0, 0}
			}
			if tc.drawCCW {
				wantCCW = [3]int{0, 0xff, 0}
			}
			wantPixel(t, tc.name+" cw", pix, 120, 120, wantCW)
			wantPixel(t, tc.name+" ccw", pix, 320, 120, wantCCW)
		})
	}
}

// Case 9: straight alpha. testTexture has one transparent white texel; with
// linear filtering the texels around a point mix as straight colours, so the
// white shows. The 1 px quad samples the texture centre (0.5, 0.5), which by
// hand is (0.5, 0.5, 0.5, 0.75): premultiplied, then SourceOver the background.
func TestGPUStraightAlpha(t *testing.T) {
	img := testTexture()
	tex := NewTexture(img)
	bgRGB := rgbOf(testBG)

	t.Run("centre", func(t *testing.T) {
		d := newDevice(t)
		d.SetTexture(tex)
		d.SetAddress(AddressClamp)
		d.SetFilter(FilterLinear)
		d.DrawTL(TriangleList, quadTL(100, 100, 101, 101, 0, 0, 1, 1, 0xffffffff))
		pix := readBack(d)
		got := pixelAt(pix, 100, 100)
		// Premultiplied source (0.375, 0.375, 0.375, 0.75) over bg.
		for k := range 3 {
			want := 0.375 + bgRGB[k]*0.25
			if g := float32(got[k]) / 255; g-want > 2.0/255 || want-g > 2.0/255 {
				t.Errorf("centre channel %d = %d/255, want %.1f/255", k, got[k], want*255)
			}
		}
		wantPixel(t, "centre neighbour", pix, 101, 100, [3]int{0x20, 0x30, 0x40})
	})

	t.Run("linear quad", func(t *testing.T) {
		d := newDevice(t)
		tr := newTrace(d)
		d.SetTexture(tex)
		d.SetAddress(AddressClamp)
		d.SetFilter(FilterLinear)
		tr.tl(img, TriangleList, quadTL(100, 100, 164, 164, 0, 0, 1, 1, 0xffffffff))
		verify(t, "straight alpha linear", readBack(d), reference(tr, testBG, depth16(1)), 3000)
	})
}

// Case (a) occlusion: two overlapping quads, near at z 0.25 and far at z 0.75,
// drawn in both orders, as two calls and as one DrawTL call. The one call
// submits both quads together, so reserve must split the batch where they
// overlap. Near-then-far in one call is what a missing split gets wrong.
func TestGPUOcclusion(t *testing.T) {
	const near, far Color = 0xffc08040, 0xff4080c0
	nearQ := withDepth(quadTL(60, 60, 140, 140, 0, 0, 1, 1, near), flat(0.25))
	farQ := withDepth(quadTL(100, 100, 180, 180, 0, 0, 1, 1, far), flat(0.75))
	for _, o := range []struct {
		name string
		qs   [2][]TLVertex
	}{
		{"near first", [2][]TLVertex{nearQ, farQ}},
		{"far first", [2][]TLVertex{farQ, nearQ}},
	} {
		for _, one := range []bool{false, true} {
			name := o.name + ", two calls"
			if one {
				name = o.name + ", one call"
			}
			t.Run(name, func(t *testing.T) {
				d := newDevice(t)
				tr := newTrace(d)
				if one {
					tr.tl(nil, TriangleList, append(append([]TLVertex{}, o.qs[0]...), o.qs[1]...))
				} else {
					tr.tl(nil, TriangleList, o.qs[0])
					tr.tl(nil, TriangleList, o.qs[1])
				}
				pix := readBack(d)
				ex := reference(tr, testBG, depth16(1))
				verify(t, name, pix, ex, 5000)
				verifyDepth(t, name, readDepth(d), ex)
				// The overlap [100,140)² shows the near quad.
				wantPixel(t, name, pix, 120, 120, rgbInts(near))
				wantPixel(t, name, pix, 70, 70, rgbInts(near))
				wantPixel(t, name, pix, 170, 170, rgbInts(far))
			})
		}
	}
}

// Case (b) depth intersection: quad a's depth runs from 0.1 to 0.9 across the
// quad and crosses quad b's flat 0.4 at x ≈ 89.5. Left of the crossing a is
// nearer, right of it b. Both orders, as two calls and as one.
func TestGPUDepthIntersection(t *testing.T) {
	const a, b Color = 0xffc08040, 0xff4080c0
	qa := withDepth(quadTL(60, 60, 140, 140, 0, 0, 1, 1, a), ramp(59.5, 139.5, 0.1, 0.9))
	qb := withDepth(quadTL(60, 60, 140, 140, 0, 0, 1, 1, b), flat(0.4))
	for _, o := range []struct {
		name string
		qs   [2][]TLVertex
	}{
		{"a first", [2][]TLVertex{qa, qb}},
		{"b first", [2][]TLVertex{qb, qa}},
	} {
		for _, one := range []bool{false, true} {
			name := o.name + ", two calls"
			if one {
				name = o.name + ", one call"
			}
			t.Run(name, func(t *testing.T) {
				d := newDevice(t)
				tr := newTrace(d)
				if one {
					tr.tl(nil, TriangleList, append(append([]TLVertex{}, o.qs[0]...), o.qs[1]...))
				} else {
					tr.tl(nil, TriangleList, o.qs[0])
					tr.tl(nil, TriangleList, o.qs[1])
				}
				pix := readBack(d)
				ex := reference(tr, testBG, depth16(1))
				verify(t, name, pix, ex, 5000)
				verifyDepth(t, name, readDepth(d), ex)
				wantPixel(t, name, pix, 70, 100, rgbInts(a))
				wantPixel(t, name, pix, 120, 100, rgbInts(b))
			})
		}
	}
}

// Case (c) alpha test and depth: a textured quad whose left half has alpha 0,
// drawn first at z 0.25, then a green quad at z 0.75 over it. With the alpha
// test on, the transparent texels write no depth, so green shows through them
// and is blocked by the opaque half. With it off, the transparent texels write
// depth too, and green is blocked everywhere.
func TestGPUAlphaTestDepth(t *testing.T) {
	const green Color = 0xff00ff00
	img := holeTexture(8, 8)
	for _, on := range []bool{true, false} {
		name := "alpha test on"
		if !on {
			name = "alpha test off"
		}
		t.Run(name, func(t *testing.T) {
			d := newDevice(t)
			tr := newTrace(d)
			d.SetTexture(NewTexture(img))
			d.SetAddress(AddressClamp)
			d.SetFilter(FilterPoint)
			d.SetAlphaTest(on, 1)
			tr.tl(img, TriangleList, withDepth(quadTL(40, 40, 104, 104, 0, 0, 1, 1, 0xffffffff), flat(0.25)))
			d.SetTexture(nil)
			d.SetAddress(AddressClamp)
			d.SetFilter(FilterPoint)
			tr.tl(nil, TriangleList, withDepth(quadTL(40, 40, 104, 104, 0, 0, 1, 1, green), flat(0.75)))
			pix := readBack(d)
			ex := reference(tr, testBG, depth16(1))
			verify(t, name, pix, ex, 3000)
			verifyDepth(t, name, readDepth(d), ex)
			// Texel (6, 3) is opaque in both runs: colour (190, 100, 128).
			wantPixel(t, name, pix, 90, 70, [3]int{190, 100, 128})
			if on {
				wantPixel(t, name, pix, 50, 70, rgbInts(green))
			} else {
				wantPixel(t, name, pix, 50, 70, rgbInts(testBG))
			}
		})
	}
}

// Case (d) test-only: a writing quad W, then two test-only quads T1 (z 0.4)
// and T2 (z 0.72) that overlap each other. T1 is nearer than T2, so if T1 had
// written depth T2 would be rejected where they overlap. Neither may be tested
// against the other, only against W.
func TestGPUTestOnly(t *testing.T) {
	const grey, red, green Color = 0xff808080, 0xffff0000, 0xff00ff00
	d := newDevice(t)
	tr := newTrace(d)
	tr.tl(nil, TriangleList, withDepth(quadTL(40, 40, 100, 100, 0, 0, 1, 1, grey), flat(0.6)))
	d.SetZWrite(false)
	tr.tl(nil, TriangleList, withDepth(quadTL(80, 80, 140, 140, 0, 0, 1, 1, red), flat(0.4)))
	tr.tl(nil, TriangleList, withDepth(quadTL(60, 60, 160, 160, 0, 0, 1, 1, green), flat(0.72)))
	pix := readBack(d)
	ex := reference(tr, testBG, depth16(1))
	verify(t, "test only", pix, ex, 5000)
	verifyDepth(t, "test only", readDepth(d), ex)
	wantPixel(t, "test only", pix, 110, 110, rgbInts(green)) // T1 and T2 overlap, outside W: T2 drawn over T1
	wantPixel(t, "test only", pix, 85, 85, rgbInts(red))     // in W: T1 passes W, T2 fails W
	wantPixel(t, "test only", pix, 50, 50, rgbInts(grey))    // W only
	wantPixel(t, "test only", pix, 95, 70, rgbInts(grey))    // in W and T2 only: T2 fails W
	wantPixel(t, "test only", pix, 70, 120, rgbInts(green))  // T2 only
}

// Case (e) clear: each combination of flags, with and without rectangles and
// with a viewport, changes exactly the pixels it should. Depth is read back
// through readDepth.
func TestGPUClear(t *testing.T) {
	const c0, c1 Color = 0xff102030, 0xffa0b0c0
	const z0, z1 float32 = 0.35, 0.8
	both := ClearTarget | ClearZBuffer
	vp := Viewport{X: 100, Y: 80, W: 120, H: 60, MinZ: 0, MaxZ: 1}
	for _, tc := range []struct {
		name  string
		vp    Viewport
		flags ClearFlags
		rects []image.Rectangle
	}{
		{"target, rect", FullViewport, ClearTarget, []image.Rectangle{image.Rect(20, 20, 60, 60)}},
		{"zbuffer, rect", FullViewport, ClearZBuffer, []image.Rectangle{image.Rect(20, 20, 60, 60)}},
		{"both, rect", FullViewport, both, []image.Rectangle{image.Rect(20, 20, 60, 60)}},
		{"both, two rects", FullViewport, both, []image.Rectangle{image.Rect(0, 0, 10, 10), image.Rect(300, 300, 320, 320)}},
		{"target, viewport", vp, ClearTarget, nil},
		{"zbuffer, viewport", vp, ClearZBuffer, nil},
		{"rect clipped to viewport", vp, both, []image.Rectangle{image.Rect(50, 50, 200, 200)}},
		{"rect outside viewport", vp, both, []image.Rectangle{image.Rect(0, 0, 50, 50)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDevice()
			d.Clear(both, c0, z0)
			d.SetViewport(tc.vp)
			d.Clear(tc.flags, c1, z1, tc.rects...)
			pix := readBack(d)
			dep := readDepth(d)

			// The model: the rectangles clipped to the viewport, or the
			// viewport when there are none.
			vpr := viewportRect(tc.vp)
			region := []image.Rectangle{vpr}
			if len(tc.rects) != 0 {
				region = region[:0]
				for _, r := range tc.rects {
					region = append(region, r.Intersect(vpr))
				}
			}
			bad := 0
			for y := range Height {
				for x := range Width {
					inside := false
					for _, r := range region {
						inside = inside || image.Pt(x, y).In(r)
					}
					f, dv := c0, depth16(z0)
					if inside && tc.flags&ClearTarget != 0 {
						f = c1
					}
					if inside && tc.flags&ClearZBuffer != 0 {
						dv = depth16(z1)
					}
					i := y*Width + x
					got := pixelAt(pix, x, y)
					if want := rgbInts(f); int(got[0]) != want[0] || int(got[1]) != want[1] || int(got[2]) != want[2] || got[3] != 255 {
						if bad++; bad <= 5 {
							t.Errorf("pixel (%d,%d) = %v, want rgb %v", x, y, got, want)
						}
					}
					if dep[i] != dv {
						if bad++; bad <= 5 {
							t.Errorf("pixel (%d,%d) depth %d, want %d", x, y, dep[i], dv)
						}
					}
				}
			}
			if bad > 0 {
				t.Errorf("%d mismatched pixels", bad)
			}
		})
	}
}

// Case (f) offset viewport: depth is tested at dstPos, so an off-by-one
// lookup would compare each pixel with a neighbour's depth. Pass 1 writes a
// plane whose slope differs in x and y. Pass 2 is the same plane offset by a
// term that is negative left of x 224 and positive right of it, so the left
// part passes and the right part fails. Both passes cover the whole back
// buffer, so the scissor is tested as well.
func TestGPUViewportDepth(t *testing.T) {
	const p1, p2 Color = 0xffa04020, 0xff20a0ff
	vp := Viewport{X: 32, Y: 16, W: 384, H: 448, MinZ: 0, MaxZ: 1}
	d := newDevice(t)
	d.SetViewport(vp)
	tr := newTrace(d)
	plane1 := func(x, y float32) float32 {
		return 0.2 + float32(0.3*(x-32)/384) + float32(0.1*(y-16)/448)
	}
	tr.tl(nil, TriangleList, withDepth(quadTL(0, 0, Width, Height, 0, 0, 1, 1, p1), plane1))
	tr.tl(nil, TriangleList, withDepth(quadTL(0, 0, Width, Height, 0, 0, 1, 1, p2), func(x, y float32) float32 {
		return plane1(x, y) + float32(0.0003*(x-224)/192)
	}))
	pix := readBack(d)
	ex := reference(tr, testBG, depth16(1))
	verify(t, "offset viewport", pix, ex, 100000)
	verifyDepth(t, "offset viewport", readDepth(d), ex)
	wantPixel(t, "offset viewport", pix, 100, 100, rgbInts(p2))
	wantPixel(t, "offset viewport", pix, 300, 300, rgbInts(p1))
	wantPixel(t, "offset viewport", pix, 10, 10, rgbInts(testBG))
	wantPixel(t, "offset viewport", pix, 420, 300, rgbInts(testBG))
}

// Case (g) D16 quantisation: 20000.1/65535 and that value plus 2e-6 both store
// D16 20000, so the later quad wins under LESSEQUAL in either order. Five D16
// steps farther, the nearer quad wins in either order.
func TestGPUDepthQuantisation(t *testing.T) {
	const red, green Color = 0xffff0000, 0xff00ff00
	zNear := float32(20000.1 / 65535)
	zSame := zNear + float32(2e-6)
	zFar := float32(20005.1 / 65535)
	if depth16(zNear) != 20000 || depth16(zSame) != 20000 || depth16(zFar) != 20005 {
		t.Fatalf("test depths store D16 %d, %d, %d; want 20000, 20000, 20005",
			depth16(zNear), depth16(zSame), depth16(zFar))
	}
	for _, tc := range []struct {
		name         string
		z0, z1       float32
		c0, c1, want Color
	}{
		{"same D16, later wins", zNear, zSame, red, green, green},
		{"same D16, later wins, reversed", zSame, zNear, green, red, red},
		{"five D16 steps, nearer wins", zFar, zNear, red, green, green},
		{"five D16 steps, nearer wins, reversed", zNear, zFar, green, red, green},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newDevice(t)
			tr := newTrace(d)
			tr.tl(nil, TriangleList, withDepth(quadTL(100, 100, 200, 200, 0, 0, 1, 1, tc.c0), flat(tc.z0)))
			tr.tl(nil, TriangleList, withDepth(quadTL(140, 140, 240, 240, 0, 0, 1, 1, tc.c1), flat(tc.z1)))
			pix := readBack(d)
			ex := reference(tr, testBG, depth16(1))
			verify(t, tc.name, pix, ex, 5000)
			verifyDepth(t, tc.name, readDepth(d), ex)
			wantPixel(t, tc.name, pix, 170, 170, rgbInts(tc.want))
		})
	}
}

// Case (h) FOGENABLE: with the fog parameters set and FOGENABLE off, a quad is
// unfogged. With it on, table fog at w = 1 is fogLinear(0, 2, 1) = ½, so the
// colour is halved and mixed with FogColor.
func TestGPUFogEnable(t *testing.T) {
	const c, fogC Color = 0xffc08040, 0xff102030
	d := newDevice(t)
	tr := newTrace(d)
	d.SetFog(Fog{Mode: FogTableLinear, Color: fogC, Start: 0, End: 2})
	d.SetFogEnable(false)
	tr.tl(nil, TriangleList, quadTL(40, 40, 80, 80, 0, 0, 1, 1, c))
	d.SetFogEnable(true)
	tr.tl(nil, TriangleList, quadTL(100, 100, 140, 140, 0, 0, 1, 1, c))
	pix := readBack(d)
	verify(t, "fog enable", pix, reference(tr, testBG, depth16(1)), 2000)
	wantPixel(t, "fog off", pix, 60, 60, rgbInts(c))
	// 0.5·(192, 128, 64) + 0.5·(16, 32, 48).
	wantPixel(t, "fog on", pix, 120, 120, [3]int{104, 80, 56})
}

// TestGPUTraceMatchesBatch is a check on the trace itself: for a single-batch
// draw, the triangles it records are exactly the vertices Device.DrawTL batches.
func TestGPUTraceMatchesBatch(t *testing.T) {
	d := newDevice(t)
	tr := newTrace(d)
	tr.tl(nil, TriangleList, quadTL(20, 20, 60, 60, 0, 0, 1, 1, 0xffffffff))
	if len(d.batch) != len(tr.tris)*3 {
		t.Fatalf("batch %d vertices, trace %d triangles", len(d.batch), len(tr.tris))
	}
	for i, v := range d.batch {
		if v != tr.tris[i/3].v[i%3] {
			t.Fatalf("vertex %d: batch %+v, trace %+v", i, v, tr.tris[i/3].v[i%3])
		}
	}
}
