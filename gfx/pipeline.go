package gfx

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The CPU half of the pipeline. Draw3D runs
//
//	toClip → clipPoly → project → culled → rasterVertex
//
// and DrawTL runs fromTL → culled → rasterVertex; ebitenVertex then packs
// either for the shader. Every step is a plain function so the spec tests in
// pipeline_test.go can pin it without a GPU.
//
// Float32 arithmetic follows std/project.go: every product and sum is
// wrapped in float32(), which also stops the compiler from fusing a·b+c
// into an FMA (Go may on amd64 v3, arm64 and others), so results do not
// depend on the target. A point drawn through Draw3D lands bit for bit where
// std.Camera.Project, the logic side's projection, puts it.

// clipVertex is a vertex in clip space, before the perspective divide.
// Attributes are carried unprojected: clipping interpolates them linearly in
// clip space, which is what keeps them perspective-correct.
type clipVertex struct {
	X, Y, Z, W float32
	Diffuse    Color
	U, V       float32
	Fog        float32 // vertex fog factor, 1 = no fog
}

// rasterVertex is the one vertex format the shader rasterises.
type rasterVertex struct {
	X, Y    float32 // D3D9 screen pixels: pixel centres on integers
	Z       float32 // after the viewport's MinZ..MaxZ mapping
	Q       float32 // 1/w (a TLVertex's RHW)
	Diffuse Color
	U, V    float32
	Fog     float32 // vertex fog factor, 1 = no fog
}

// texCoord is Direct3D's COUNT2 texture transform of the 2D texture
// coordinate (u, v) by m: the coordinate is taken as (u, v, 1, 0), so the
// translation is row 2 and row 3 has no effect, and the output is the first
// two components:
//
//	u' = (u·m[0][0] + v·m[1][0]) + m[2][0]
//	v' = (u·m[0][1] + v·m[1][1]) + m[2][1]
//
// in float32, each product and sum rounded.
func texCoord(m *Mat4, u, v float32) (float32, float32) {
	x := float32(float32(u*m[0][0])+float32(v*m[1][0])) + m[2][0]
	y := float32(float32(u*m[0][1])+float32(v*m[1][1])) + m[2][1]
	return x, y
}

// mul4 is D3DXMatrixMultiply (a then b): m[i][j] is
// ((a[i][0]·b[0][j] + a[i][1]·b[1][j]) + a[i][2]·b[2][j]) + a[i][3]·b[3][j].
func mul4(a, b Mat4) (m Mat4) {
	for i := range 4 {
		for j := range 4 {
			m[i][j] = float32(float32(float32(a[i][0]*b[0][j])+float32(a[i][1]*b[1][j]))+
				float32(a[i][2]*b[2][j])) + float32(a[i][3]*b[3][j])
		}
	}
	return m
}

// comp is component j of p·m for p = (x, y, z, 1), grouped as std's
// transformCoord: the three products summed left to right, then m[3][j].
func comp(m *Mat4, j int, x, y, z float32) float32 {
	return float32(float32(float32(m[0][j]*x)+float32(m[1][j]*y))+float32(m[2][j]*z)) + m[3][j]
}

// toClip is the vertex pipeline up to clip space. wvp is
// mul4(mul4(world, view), proj) and wv is mul4(world, view), both taken once
// per draw; component j of p·M is ((M[0][j]·x + M[1][j]·y) + M[2][j]·z) +
// M[3][j], as in std's transformCoord. The clip position is p·wvp (one
// product, not (p·wv)·proj, so it matches std.Camera.Project).
//
// With fog.Mode == FogVertexLinear the fog factor is fogLinear of the
// view-space z, (p·wv)[2], or of the view-space distance |p·wv| (float32
// sqrt of the float32 dot product) when fog.Range is set; otherwise it is 1.
func toClip(v Vertex3D, wvp, wv Mat4, fog Fog) clipVertex {
	c := clipVertex{
		X:       comp(&wvp, 0, v.X, v.Y, v.Z),
		Y:       comp(&wvp, 1, v.X, v.Y, v.Z),
		Z:       comp(&wvp, 2, v.X, v.Y, v.Z),
		W:       comp(&wvp, 3, v.X, v.Y, v.Z),
		Diffuse: v.Diffuse,
		U:       v.U,
		V:       v.V,
		Fog:     1,
	}
	if fog.Mode == FogVertexLinear {
		x := comp(&wv, 0, v.X, v.Y, v.Z)
		y := comp(&wv, 1, v.X, v.Y, v.Z)
		z := comp(&wv, 2, v.X, v.Y, v.Z)
		d := z
		if fog.Range {
			dot := float32(float32(x*x)+float32(y*y)) + float32(z*z)
			d = float32(math.Sqrt(float64(dot)))
		}
		c.Fog = fogLinear(fog.Start, fog.End, d)
	}
	return c
}

// fogLinear is D3DFOG_LINEAR: (end − d)/(end − start) clamped to [0, 1].
// When start == end the factor is 1 for d < end and 0 otherwise.
func fogLinear(start, end, d float32) float32 {
	if start == end {
		if d < end {
			return 1
		}
		return 0
	}
	f := (end - d) / (end - start)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// clipPoly clips the convex polygon src against D3D's near and far planes,
// z ≥ 0 and z ≤ w, appending the result to dst[:0] and returning it.
// Sutherland–Hodgman, one plane after the other (near first), keeps the
// input's cyclic order; a new vertex on edge a→b (a before b in that
// order, wrapping from last to first) is a + t·(b − a), with
// t = da/(da − db) for the signed plane distances da = a.Z, a.W − a.Z (near,
// far) and likewise db, applied to every
// field (X, Y, Z, W, U, V, Fog, and each 8-bit channel of Diffuse rounded
// to nearest). Vertices exactly on a plane are inside. The result is empty
// or has at least three vertices.
//
// The x and y planes are not clipped: the viewport scissor (Device) does
// the same job. w > 0 wherever z ≥ 0 holds for the D3DX projections th10
// uses (perspectiveFovLH, orthographic), so there is no separate w plane.
func clipPoly(dst, src []clipVertex) []clipVertex {
	// A triangle gains at most one vertex per plane, so the intermediate
	// polygon fits the stack buffer and steady-state clipping allocates nothing.
	var scratch [8]clipVertex
	mid := clipPlane(scratch[:0], src, false)
	return clipPlane(dst[:0], mid, true)
}

// clipPlane clips src against one plane, appending to dst: the near plane
// z ≥ 0 when far is false, the far plane z ≤ w when it is true.
func clipPlane(dst, src []clipVertex, far bool) []clipVertex {
	n := len(src)
	for i := range n {
		cur, prev := src[i], src[(i+n-1)%n]
		dc, dp := planeDist(cur, far), planeDist(prev, far)
		switch {
		case dc >= 0:
			if dp < 0 {
				dst = append(dst, clipEdge(prev, cur, dp, dc))
			}
			dst = append(dst, cur)
		case dp >= 0:
			dst = append(dst, clipEdge(prev, cur, dp, dc))
		}
	}
	return dst
}

// planeDist is the signed distance of v inside a plane: z for the near
// plane, w − z for the far one.
func planeDist(v clipVertex, far bool) float32 {
	if far {
		return v.W - v.Z
	}
	return v.Z
}

// clipEdge is the crossing of the edge a→b, given the plane distances da
// and db of its ends, which have opposite signs.
func clipEdge(a, b clipVertex, da, db float32) clipVertex {
	t := da / (da - db)
	return clipVertex{
		X:       lerp(a.X, b.X, t),
		Y:       lerp(a.Y, b.Y, t),
		Z:       lerp(a.Z, b.Z, t),
		W:       lerp(a.W, b.W, t),
		Diffuse: lerpColor(a.Diffuse, b.Diffuse, t),
		U:       lerp(a.U, b.U, t),
		V:       lerp(a.V, b.V, t),
		Fog:     lerp(a.Fog, b.Fog, t),
	}
}

// lerp is a + t·(b − a), with the product rounded to float32 first.
func lerp(a, b, t float32) float32 {
	return a + float32(t*(b-a))
}

// lerpColor interpolates each 8-bit channel of a and b, rounding to nearest.
func lerpColor(a, b Color, t float32) Color {
	var out Color
	for s := range 4 {
		sh := uint(8 * s)
		ca := float32((a >> sh) & 0xff)
		cb := float32((b >> sh) & 0xff)
		out |= uint32(math.Round(float64(lerp(ca, cb, t)))) << sh
	}
	return out
}

// project is the perspective divide and viewport mapping, grouped as in
// std.Camera.Project:
//
//	X = vp.X + ((1 + X/W)·vp.W)/2
//	Y = vp.Y + ((1 − Y/W)·vp.H)/2
//	Z = vp.MinZ + (Z/W)·(vp.MaxZ − vp.MinZ)
//	Q = 1/W
//
// Diffuse, U, V and Fog pass through.
func project(c clipVertex, vp Viewport) rasterVertex {
	nx := c.X / c.W
	ny := c.Y / c.W
	nz := c.Z / c.W
	return rasterVertex{
		X:       vp.X + float32(float32(float32(1+nx)*vp.W)/2),
		Y:       vp.Y + float32(float32(float32(1-ny)*vp.H)/2),
		Z:       vp.MinZ + float32(nz*(vp.MaxZ-vp.MinZ)),
		Q:       1 / c.W,
		Diffuse: c.Diffuse,
		U:       c.U,
		V:       c.V,
		Fog:     c.Fog,
	}
}

// fromTL converts a pre-transformed vertex. Direct3D takes a TL vertex's
// fog factor from its specular alpha (float32(alpha)/255) under vertex fog; under table
// fog and without fog Fog is 1.
func fromTL(v TLVertex, fog Fog) rasterVertex {
	r := rasterVertex{
		X:       v.X,
		Y:       v.Y,
		Z:       v.Z,
		Q:       v.RHW,
		Diffuse: v.Diffuse,
		U:       v.U,
		V:       v.V,
		Fog:     1,
	}
	if fog.Mode == FogVertexLinear {
		r.Fog = float32(v.Specular>>24) / 255
	}
	return r
}

// culled reports whether c drops the screen-space triangle a, b, c. Its
// winding is the sign of (b.X−a.X)·(c.Y−a.Y) − (c.X−a.X)·(b.Y−a.Y):
// positive is clockwise on screen (y grows downward). CullCW drops
// positive, CullCCW negative, CullNone nothing; a zero-area triangle is never
// dropped (it covers no pixels anyway).
func culled(c Cull, a, b, v rasterVertex) bool {
	p := float32((b.X - a.X) * (v.Y - a.Y))
	q := float32((v.X - a.X) * (b.Y - a.Y))
	s := p - q
	switch c {
	case CullCW:
		return s > 0
	case CullCCW:
		return s < 0
	}
	return false
}

// screenTri is a triangle's screen positions, the X and Y of its three
// rasterVertex values, as reserve records them.
type screenTri [3][2]float32

// conflicts reports whether a and b may cover a common pixel, so that the
// later one's depth test could depend on the earlier one's depth write:
//
//   - false if either has zero area (it covers no pixel);
//   - false if their bounding boxes, edges included, do not intersect (a
//     fast path; the edge rule below gives the same answer);
//   - false if an edge of a or b separates them: the other triangle's three
//     vertices all lie on the edge's line or on the side of it away from
//     the edge's own third vertex. The side of p for the edge e0→e1 is the
//     sign of (e1.X−e0.X)·(p.Y−e0.Y) − (p.X−e0.X)·(e1.Y−e0.Y), each product
//     rounded to float32 as in culled;
//   - true otherwise.
//
// Either winding is accepted. Triangles that only touch, like the two halves
// of a quad, do not conflict: the fill rule gives each pixel on a shared edge
// to one of them.
func conflicts(a, b screenTri) bool {
	if triArea(a) == 0 || triArea(b) == 0 {
		return false
	}
	if !triBox(a).meets(triBox(b)) {
		return false
	}
	return !separates(a, b) && !separates(b, a)
}

// edgeSide is the side of p for the edge e0→e1: the sign of
// (e1.X−e0.X)·(p.Y−e0.Y) − (p.X−e0.X)·(e1.Y−e0.Y), each product rounded to
// float32 as culled rounds its own.
func edgeSide(e0, e1, p [2]float32) float32 {
	return float32((e1[0]-e0[0])*(p[1]-e0[1])) - float32((p[0]-e0[0])*(e1[1]-e0[1]))
}

// triArea is t's signed area, zero when it is degenerate: edgeSide of its
// first edge against its third vertex.
func triArea(t screenTri) float32 {
	return edgeSide(t[0], t[1], t[2])
}

// box is an axis-aligned bounding box, min and max corners.
type box struct{ lo, hi [2]float32 }

// triBox is t's bounding box.
func triBox(t screenTri) box {
	b := box{lo: t[0], hi: t[0]}
	for _, p := range t[1:] {
		for k := range 2 {
			b.lo[k] = min(b.lo[k], p[k])
			b.hi[k] = max(b.hi[k], p[k])
		}
	}
	return b
}

// meets reports whether b and o intersect, edges included.
func (b box) meets(o box) bool {
	for k := range 2 {
		if b.lo[k] > o.hi[k] || o.lo[k] > b.hi[k] {
			return false
		}
	}
	return true
}

// separates reports whether an edge of t has every vertex of u on its line
// or on the side away from t's third vertex.
func separates(t, u screenTri) bool {
	for k := range 3 {
		e0, e1 := t[k], t[(k+1)%3]
		s := edgeSide(e0, e1, t[(k+2)%3])
		away := true
		for _, p := range u {
			v := edgeSide(e0, e1, p)
			if (s > 0 && v > 0) || (s < 0 && v < 0) {
				away = false
				break
			}
		}
		if away {
			return true
		}
	}
	return false
}

// triangles calls emit with the vertex indices of each triangle of an
// n-vertex primitive, in D3D's order:
//
//	TriangleList   (3i, 3i+1, 3i+2)
//	TriangleStrip  (i, i+1, i+2) for even i, (i+1, i, i+2) for odd i
//	TriangleFan    (0, i+1, i+2)
//
// Leftover vertices that do not complete a triangle are ignored.
func triangles(p Primitive, n int, emit func(a, b, c int)) {
	switch p {
	case TriangleList:
		for i := 0; i+2 < n; i += 3 {
			emit(i, i+1, i+2)
		}
	case TriangleStrip:
		for i := 0; i+2 < n; i++ {
			if i%2 == 0 {
				emit(i, i+1, i+2)
			} else {
				emit(i+1, i, i+2)
			}
		}
	case TriangleFan:
		for i := 0; i+2 < n; i++ {
			emit(0, i+1, i+2)
		}
	}
}

// ebitenVertex packs v for the shader (shader.go). It is the half-pixel
// rule's only site: D3D9 puts pixel centres on integers and Ebiten on
// halves, so DstX/DstY are X+0.5, Y+0.5. SrcX is Z, the depth, which
// Direct3D interpolates screen-linearly as Ebiten does every attribute; the
// shader addresses the texture itself, so SrcX is free, and an unmanaged
// texture leaves it untouched (Texture). SrcY is 0. ColorR..A is Diffuse as straight-alpha
// 0..1 floats, each float32(channel)/255. Custom0..3 are U·Q, V·Q, Q and Fog·Q, so that dividing by
// the interpolated Q undoes Ebiten's screen-linear interpolation.
//
// Diffuse is interpolated screen-linearly, not perspective-correctly; the
// draw paths seen so far give a quad one colour, where the two agree.
func ebitenVertex(v rasterVertex) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:    v.X + 0.5,
		DstY:    v.Y + 0.5,
		SrcX:    v.Z,
		ColorR:  float32((v.Diffuse>>16)&0xff) / 255,
		ColorG:  float32((v.Diffuse>>8)&0xff) / 255,
		ColorB:  float32(v.Diffuse&0xff) / 255,
		ColorA:  float32(v.Diffuse>>24) / 255,
		Custom0: v.U * v.Q,
		Custom1: v.V * v.Q,
		Custom2: v.Q,
		Custom3: v.Fog * v.Q,
	}
}
