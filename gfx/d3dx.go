package gfx

import "math"

// The D3DX matrix helpers th10 calls, which it imports from the D3DX9 DLL
// (the jump thunks at 0x4521ac RotationX, 0x4521b2 Multiply, 0x4521b8
// RotationY, 0x4521be RotationZ). The DLL's CPU-specific code paths are not
// reproduced: these follow the documented formulas with float32 rounding of
// every operation, which is close but not proven bit-identical, so they are
// for drawing only.

// Mul is D3DXMatrixMultiply: a·b, with mul4's grouping.
func Mul(a, b Mat4) Mat4 { return mul4(a, b) }

// RotationX is D3DXMatrixRotationX: with c = float32(cos a) and
// s = float32(sin a), the identity with [1][1] = c, [1][2] = s,
// [2][1] = −s, [2][2] = c.
func RotationX(a float32) Mat4 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	m := Identity
	m[1][1], m[1][2] = c, s
	m[2][1], m[2][2] = -s, c
	return m
}

// RotationY is D3DXMatrixRotationY: the identity with [0][0] = c,
// [0][2] = −s, [2][0] = s, [2][2] = c (c, s as in RotationX).
func RotationY(a float32) Mat4 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	m := Identity
	m[0][0], m[0][2] = c, -s
	m[2][0], m[2][2] = s, c
	return m
}

// Translation is D3DXMatrixTranslation (thunk 0x452182): the identity with
// [3][0], [3][1], [3][2] = x, y, z.
func Translation(x, y, z float32) Mat4 {
	m := Identity
	m[3][0], m[3][1], m[3][2] = x, y, z
	return m
}

// Vec3Project is D3DXVec3Project, which D3DXVec3ProjectArray (thunk
// 0x452188) applies to each point. Wine's formula:
//
//	m = Mul(Mul(Mul(Identity, world), view), proj)
//	p = v as (x, y, z, 1) through m, each component divided by w
//	    (D3DXVec3TransformCoord; each component grouped as comp: the three
//	    products summed left to right, then m[3][j])
//	x = vp.X + ((1 + p.x)·vp.W)/2
//	y = vp.Y + ((1 − p.y)·vp.H)/2
//	z = vp.MinZ + p.z·(vp.MaxZ − vp.MinZ)
//
// with every operation rounded to float32. With world the identity, x and
// y are std.Camera.Project's before its move to ECL space.
func Vec3Project(v [3]float32, vp Viewport, proj, view, world Mat4) [3]float32 {
	m := Mul(Mul(Mul(Identity, world), view), proj)
	// The transform step of D3DXVec3ProjectArray (thunk 0x452188): the
	// four components of (v, 1)·m, each of the first three divided by w.
	w := comp(&m, 3, v[0], v[1], v[2])
	var p [3]float32
	for j := range p {
		p[j] = comp(&m, j, v[0], v[1], v[2]) / w
	}
	return [3]float32{
		vp.X + float32(float32(float32(1+p[0])*vp.W)/2),
		vp.Y + float32(float32(float32(1-p[1])*vp.H)/2),
		vp.MinZ + float32(p[2]*float32(vp.MaxZ-vp.MinZ)),
	}
}

// RotationZ is D3DXMatrixRotationZ: the identity with [0][0] = c,
// [0][1] = s, [1][0] = −s, [1][1] = c (c, s as in RotationX).
func RotationZ(a float32) Mat4 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	m := Identity
	m[0][0], m[0][1] = c, s
	m[1][0], m[1][1] = -s, c
	return m
}
