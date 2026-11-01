package std

import "math"

// Viewport is camera slot 0's D3DVIEWPORT9 (+0xcc, set by FUN_004216f0):
// the playfield at (32, 16), 384x448, depth 0..1.
var viewport = struct{ X, Y, W, H float32 }{32, 16, 384, 448}

// Near and far planes FUN_004215a0 passes to D3DXMatrixPerspectiveFovLH.
const (
	zNear float32 = 30
	zFar  float32 = 1800
)

// Project is where ECL 256 puts the child of a 3D enemy (flag 0x40000,
// FUN_0040e770 case 0x100): FUN_00401d20 rebuilds camera slot 0's view and
// projection (FUN_004215a0), D3DXVec3Project maps p onto the viewport, and
// the result is moved to ECL space (DAT_00470b48/4c: x-224, y-16).
//
// The D3DX helpers follow Wine's order of operations; which code path the
// real d3dx9 takes is not traced. The eye offset at camera +0x3c (written
// only by STD op 12, which no TH10 stage uses) is taken as 0.
func (c *Camera) Project(p [3]float32) (x, y float32) {
	eye := c.Pos
	at := add3(c.Facing, c.Pos)
	view := lookAtLH(eye, at, c.Up)
	proj := perspectiveFovLH(c.FOV, float32(viewport.W/viewport.H), zNear, zFar)
	v := transformCoord(p, mul4(view, proj))
	sx := viewport.X + float32(float32(float32(1+v[0])*viewport.W)/2)
	sy := viewport.Y + float32(float32(float32(1-v[1])*viewport.H)/2)
	return sx - 224, sy - 16
}

type mat4 [4][4]float32

func add3(a, b [3]float32) [3]float32 {
	return [3]float32{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func sub3(a, b [3]float32) [3]float32 {
	return [3]float32{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func dot3(a, b [3]float32) float32 {
	return float32(float32(a[0]*b[0])+float32(a[1]*b[1])) + float32(a[2]*b[2])
}

func cross3(a, b [3]float32) [3]float32 {
	return [3]float32{
		float32(a[1]*b[2]) - float32(a[2]*b[1]),
		float32(a[2]*b[0]) - float32(a[0]*b[2]),
		float32(a[0]*b[1]) - float32(a[1]*b[0]),
	}
}

// normalize3 is D3DXVec3Normalize: v / |v|, the zero vector unchanged.
func normalize3(v [3]float32) [3]float32 {
	n := float32(math.Sqrt(float64(dot3(v, v))))
	if n == 0 {
		return [3]float32{}
	}
	return [3]float32{v[0] / n, v[1] / n, v[2] / n}
}

// lookAtLH is D3DXMatrixLookAtLH.
func lookAtLH(eye, at, up [3]float32) mat4 {
	z := normalize3(sub3(at, eye))
	x := cross3(up, z)
	y := cross3(z, x)
	x, y = normalize3(x), normalize3(y)
	return mat4{
		{x[0], y[0], z[0], 0},
		{x[1], y[1], z[1], 0},
		{x[2], y[2], z[2], 0},
		{-dot3(x, eye), -dot3(y, eye), -dot3(z, eye), 1},
	}
}

// perspectiveFovLH is D3DXMatrixPerspectiveFovLH.
func perspectiveFovLH(fovy, aspect, zn, zf float32) mat4 {
	t := float32(math.Tan(float64(fovy / 2)))
	return mat4{
		{1 / float32(aspect*t), 0, 0, 0},
		{0, 1 / t, 0, 0},
		{0, 0, zf / (zf - zn), 1},
		{0, 0, float32(zf*zn) / (zn - zf), 0},
	}
}

// mul4 is D3DXMatrixMultiply (a then b).
func mul4(a, b mat4) (m mat4) {
	for i := range 4 {
		for j := range 4 {
			m[i][j] = float32(float32(float32(a[i][0]*b[0][j])+float32(a[i][1]*b[1][j]))+
				float32(a[i][2]*b[2][j])) + float32(a[i][3]*b[3][j])
		}
	}
	return m
}

// transformCoord is D3DXVec3TransformCoord: p as (x, y, z, 1) through m,
// divided by w.
func transformCoord(p [3]float32, m mat4) (out [3]float32) {
	col := func(j int) float32 {
		return float32(float32(float32(m[0][j]*p[0])+float32(m[1][j]*p[1]))+float32(m[2][j]*p[2])) + m[3][j]
	}
	w := col(3)
	for j := range out {
		out[j] = col(j) / w
	}
	return out
}
