package std

import "math"

// Viewport is camera slot 0's D3DVIEWPORT9 (+0xcc, set by FUN_004216f0):
// the playfield at (32, 16), 384x448, depth 0..1.
var Viewport = struct{ X, Y, W, H float32 }{32, 16, 384, 448}

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
	view, proj := c.matrices()
	v := transformCoord(p, mul4(view, proj))
	sx := Viewport.X + float32(float32(float32(1+v[0])*Viewport.W)/2)
	sy := Viewport.Y + float32(float32(float32(1-v[1])*Viewport.H)/2)
	return sx - 224, sy - 16
}

// Matrices are the view and projection FUN_004215a0 builds for camera slot
// 0, the pair Project uses. The background draw path takes them from here,
// so the logic and the picture share one copy of the camera maths.
func (c *Camera) Matrices() (view, proj [4][4]float32) {
	v, p := c.matrices()
	return v, p
}

func (c *Camera) matrices() (view, proj mat4) {
	v, p := LookMatrices(c.Pos, c.Facing, c.Up, [3]float32{}, c.FOV, uint32(Viewport.W), uint32(Viewport.H))
	return v, p
}

// LookMatrices are the view and projection FUN_004215a0 builds from a
// camera block whose viewport is w x h (+0xd4, +0xd8):
//
//	view = LookAtLH(eye Eye + Pos, at Facing + Pos, Up)   (+0x3c, +0x0c, +0x00, +0x18)
//	proj = PerspectiveFovLH(fov, w/h, 30, 1800)           (+0x48)
//
// Facing zero makes eye and at the same point: the view's axes are zero
// (normalize3 leaves the zero vector), which only matters to 3D draws.
func LookMatrices(pos, facing, up, eye [3]float32, fov float32, w, h uint32) (view, proj [4][4]float32) {
	view = lookAtLH(add3(eye, pos), add3(facing, pos), up)
	proj = perspectiveFovLH(fov, float32(w)/float32(h), zNear, zFar)
	return view, proj
}

// Right is the side vector FUN_004215a0 stores at camera block +0x30 after
// setting the matrices: normalize(facing × up).
func Right(facing, up [3]float32) [3]float32 {
	return normalize3(cross3(facing, up))
}

// flatHalfFOV is the double at 0x470ba0, float32(π/20) widened.
const flatHalfFOV = 0.15707963705062866

// FlatMatrices are the view and projection FUN_00421480 builds for a camera
// block whose viewport is w x h (+0xd4, +0xd8):
//
//	view = LookAtLH(eye (w/2, h/2, d), at (w/2, h/2, 0), up (0, −1, 0))
//	proj = PerspectiveFovLH(0x3ea0d97c (π/10), w/h, 1, 10000)
//	d    = (h/2) / tan(0x470ba0)
//
// The half field of view is π/20, so the plane z = 0 maps one unit to one
// viewport pixel, x right and y down, and greater z is nearer the eye.
//
// FPTAN is not subject to the x87 precision control, so the tangent is
// math.Tan in float64; the quotient is rounded to float32 as FDIVR at 24-bit
// precision rounds it (once from the extended tangent in the original, so
// the last bit can differ in rare near-tie cases).
func FlatMatrices(w, h uint32) (view, proj [4][4]float32) {
	fw, fh := float32(w), float32(h)
	hw, hh := float32(fw*0.5), float32(fh*0.5)
	d := float32(float64(hh) / math.Tan(flatHalfFOV))
	view = lookAtLH([3]float32{hw, hh, d}, [3]float32{hw, hh, 0}, [3]float32{0, -1, 0})
	proj = perspectiveFovLH(math.Float32frombits(0x3ea0d97c), fw/fh, 1, 10000)
	return view, proj
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
