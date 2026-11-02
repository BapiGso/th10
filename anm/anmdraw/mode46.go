package anmdraw

import (
	"math"

	"th10/anm"
	"th10/gfx"
)

// Render modes 4 and 6: a sprite-sized billboard projected from the
// object's position through the selected camera, drawn as a 2D quad, mode
// 6 also fading with distance from the Field camera
// (docs/research/m4_r7.md R7-C, R7-D). STD quads of stages 1 and 5 use
// mode 6.

// Cams is what modes 4 and 6 read from the supervisor's camera blocks,
// which the original reads straight from memory. *supervisor.Supervisor
// implements it.
type Cams interface {
	// Selected returns Right (+0x30), View (+0x4c), Proj (+0x8c) and
	// Viewport (+0xcc) of the selected block, the one DAT_00491fac
	// points at.
	Selected() (right [3]float32, view, proj gfx.Mat4, vp gfx.Viewport)

	// FieldFog returns the Field block's (0x491d7c) Pos, fog near and
	// far (+0xfc, +0x100: DAT_00491e78, DAT_00491e7c) and fog bytes 0-2
	// (+0x104, +0x108, +0x10c: B, G, R, as floats).
	FieldFog() (eye [3]float32, near, far float32, bgr [3]float32)
}

// corners is FUN_00443b60, which stages the corners of modes 4 and 6.
// With Ax, Ay as in the quad notation, Az = (Pos2.z + Offset.z) + Pos.z
// and right, view, proj, vp = r.Cams.Selected(), it projects the object's
// origin and its right vector:
//
//	world = gfx.Translation(Ax, Ay, Az)
//	o = gfx.Vec3Project({0, 0, 0}, vp, proj, view, world)
//	if o.z < 0 or o.z > 1: return false
//	q = gfx.Vec3Project(right, vp, proj, view, world)
//
// A rejected object writes nothing to r.quad. A NaN o.z passes both tests
// (0x443c9b-0x443cbb branch on ordered results only). Then the projected
// half length of right scales the sprite:
//
//	dx, dy, dz = q.x − o.x, q.y − o.y, q.z − o.z
//	f = sqrt((dz·dz + dy·dy) + dx·dx)·0.5
//	w = (spr.W·Scale[0])·f     h = (spr.H·Scale[1])·f
//
// sqrt is float32(math.Sqrt(float64(·))), FSQRT at 24-bit precision. With
// c, s and the edges L, R, T, B for w and h as in rotated, the corners
// are
//
//	P0 = ((L·c − T·s) + o.x, (T·c + L·s) + o.y)
//	P1 = ((R·c − T·s) + o.x, (R·s + T·c) + o.y)
//	P2 = ((L·c − B·s) + o.x, (B·c + L·s) + o.y)
//	P3 = ((R·c − B·s) + o.x, (B·c + R·s) + o.y)
//
// with Z = o.z for all four, and corners returns true. RHW is not written.
// An anchor of 3 reads uninitialised stack, so corners panics on it, but
// only after the depth test: a rejected object never reads its anchors.
//
// Addresses: world 0x443b69-0x443c92 (X, Y, Z at 0x443b7e, 0x443bac,
// 0x443be5), depth tests 0x443c97-0x443cbb, q 0x443cc1-0x443ce6,
// differences 0x443ceb-0x443d33, f 0x443d37-0x443d5d, w and h
// 0x443d63-0x443d8d, Z 0x443d3f-0x443d88, FSINCOS 0x443d93, x edges
// 0x443da1-0x443e18, y edges 0x443e1c-0x443e8b, corners 0x443e8f-0x443f5d.
func (r *Renderer) corners(vm *anm.VM, spr *Sprite) bool {
	// Ax, Ay, Az with Az = (Pos2.z + Offset.z) + Pos.z: world 0x443b69-0x443c92
	// (X, Y, Z at 0x443b7e, 0x443bac, 0x443be5).
	right, view, proj, vp := r.Cams.Selected()
	Ax := (vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]
	Ay := (vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]
	Az := (vm.Pos2[2] + vm.Offset[2]) + vm.Pos[2]
	world := gfx.Translation(Ax, Ay, Az)
	o := gfx.Vec3Project([3]float32{0, 0, 0}, vp, proj, view, world)
	// Depth test 0x443c97-0x443cbb: an ordered test, so a NaN o.z passes.
	if o[2] < 0 || o[2] > 1 {
		return false
	}
	// Anchor 3 reads uninitialised stack in the original, so it panics here,
	// after the depth test (the anchors are read only past it).
	if vm.Flags>>18&3 == 3 || vm.Flags>>20&3 == 3 {
		panic("anmdraw: corners anchor 3 reads uninitialised stack")
	}
	ax := vm.Flags >> 18 & 3
	ay := vm.Flags >> 20 & 3
	// q 0x443cc1-0x443ce6.
	q := gfx.Vec3Project(right, vp, proj, view, world)
	// Differences 0x443ceb-0x443d33.
	dx, dy, dz := q[0]-o[0], q[1]-o[1], q[2]-o[2]
	// f 0x443d37-0x443d5d: sqrt((dz·dz + dy·dy) + dx·dx)·0.5.
	sum := (float32(dz*dz) + float32(dy*dy)) + float32(dx*dx)
	f := float32(float32(math.Sqrt(float64(sum))) * 0.5)
	// w and h 0x443d63-0x443d8d.
	w := float32(float32(spr.W*vm.Scale[0]) * f)
	h := float32(float32(spr.H*vm.Scale[1]) * f)
	// FSINCOS 0x443d93: c and s, stored as float32 before use, as in rotated.
	c := float32(math.Cos(float64(vm.Rot[2])))
	s := float32(math.Sin(float64(vm.Rot[2])))
	// Edges as in rotated: x edges 0x443da1-0x443e18, y edges 0x443e1c-0x443e8b.
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
	// Corners 0x443e8f-0x443f5d, each sum grouped as the doc writes it.
	r.quad[0].X = (float32(l*c) - float32(tp*s)) + o[0]
	r.quad[0].Y = (float32(tp*c) + float32(l*s)) + o[1]
	r.quad[1].X = (float32(rr*c) - float32(tp*s)) + o[0]
	r.quad[1].Y = (float32(rr*s) + float32(tp*c)) + o[1]
	r.quad[2].X = (float32(l*c) - float32(b*s)) + o[0]
	r.quad[2].Y = (float32(b*c) + float32(l*s)) + o[1]
	r.quad[3].X = (float32(rr*c) - float32(b*s)) + o[0]
	r.quad[3].Y = (float32(b*c) + float32(rr*s)) + o[1]
	// Z = o.z for all four corners (0x443d3f-0x443d88). RHW, U, V and
	// Diffuse are not written.
	for i := range r.quad {
		r.quad[i].Z = o[2]
	}
	return true
}

// draw4 is FUN_00443f80, mode 4: if corners, emit with 0.
func (r *Renderer) draw4(vm *anm.VM, spr *Sprite) {
	// emit with 0 writes r.colour to the corners.
	if r.corners(vm, spr) {
		r.emit(vm, spr, 0)
	}
}

// draw6 is FUN_00443fb0, mode 6. If corners returns false, nothing else
// happens. Otherwise, with
//
//	col = r.colour(vm)
//	eye, near, far, bgr = r.Cams.FieldFog()
//	Ex = (Offset.x + Pos.x) + Pos2.x    (Ey, Ez alike)
//	dx, dy, dz = Ex − eye.x, Ey − eye.y, Ez − eye.z
//	d = sqrt((dz·dz + dy·dy) + dx·dx)
//
// (E's grouping is not corners' Ax, Ay, Az), the colour is col unless
// near < d, an ordered test: a NaN keeps col. Otherwise
//
//	f = (near − d) / (near − far)
//	if f >= 1: return
//
// culls the object: no colour written, no emit (a NaN f goes on). The
// colour is then col with byte k = 0, 1, 2 (B, G, R), of old value c,
// replaced by
//
//	c − uint8(ftol(float32(int32(c) − int32(ftol(bgr[k])))·f))
//
// and byte 3 (A), of old value a, by uint8(ftol((1 − f)·float32(a))).
// Each byte wraps: the subtraction is a byte one (SUB BL,AL) and every
// ftol keeps its low byte, so 1 − f > 1 or a negative product can carry
// a byte past 255. The int32 difference converts exactly for any fog byte
// below 2^24 (FILD is exact; the product rounds once).
//
// The colour, faded or not, goes to all four corners' Diffuse; then emit
// with emitKeep.
//
// Addresses: colour selection 0x443fc5-0x443fef and tint
// 0x444064-0x444126 (both r.colour), E 0x443fef-0x44402e, d
// 0x444034-0x444094, near < d 0x44412c-0x444139, f 0x44413f-0x444149,
// f >= 1 0x44414b-0x444163, B 0x444164-0x44418b, G 0x444191-0x4441b6, R
// 0x4441bc-0x4441ec, A 0x4441d9-0x4441fb, corners' colour
// 0x444200-0x444221, emit 0x444223.
func (r *Renderer) draw6(vm *anm.VM, spr *Sprite) {
	// A rejected object gets nothing else.
	if !r.corners(vm, spr) {
		return
	}
	// Colour: colour selection 0x443fc5-0x443fef and tint 0x444064-0x444126.
	col := r.colour(vm)
	eye, near, far, bgr := r.Cams.FieldFog()
	// E = (Offset + Pos) + Pos2, not corners' Ax grouping: 0x443fef-0x44402e.
	Ex := (vm.Offset[0] + vm.Pos[0]) + vm.Pos2[0]
	Ey := (vm.Offset[1] + vm.Pos[1]) + vm.Pos2[1]
	Ez := (vm.Offset[2] + vm.Pos[2]) + vm.Pos2[2]
	// d 0x444034-0x444094: sqrt((dz·dz + dy·dy) + dx·dx), without the 0.5.
	dx, dy, dz := Ex-eye[0], Ey-eye[1], Ez-eye[2]
	sum := (float32(dz*dz) + float32(dy*dy)) + float32(dx*dx)
	d := float32(math.Sqrt(float64(sum)))
	c := col
	// near < d (ordered: a NaN keeps col) 0x44412c-0x444139.
	if near < d {
		// f 0x44413f-0x444149; f >= 1 culls 0x44414b-0x444163 (a NaN goes on).
		f := (near - d) / (near - far)
		if f >= 1 {
			return
		}
		ch := [4]uint8{uint8(col), uint8(col >> 8), uint8(col >> 16), uint8(col >> 24)}
		// B, G, R for k = 0, 1, 2: 0x444164-0x44418b, 0x444191-0x4441b6,
		// 0x4441bc-0x4441ec. Each byte wraps (SUB BL,AL), and so does ftol's.
		for k := range 3 {
			old := ch[k]
			t := int32(ftol(bgr[k]))
			x := float32(float32(int32(old)-t) * f)
			ch[k] = old - uint8(ftol(x))
		}
		// A 0x4441d9-0x4441fb: uint8(ftol((1 − f)·a)). ch[3] is still the old
		// alpha, since the loop wrote only bytes 0-2.
		ch[3] = uint8(ftol(float32(float32(1-f) * float32(ch[3]))))
		c = gfx.Color(ch[0]) | gfx.Color(ch[1])<<8 | gfx.Color(ch[2])<<16 | gfx.Color(ch[3])<<24
	}
	// The colour, faded or not, goes to all four corners (0x444200-0x444221),
	// then emit with emitKeep (0x444223).
	for i := range r.quad {
		r.quad[i].Diffuse = c
	}
	r.emit(vm, spr, emitKeep)
}

// ftol is _ftol2 (FUN_00463b2c): x truncated toward zero, as an int64. A
// NaN or a value outside the int64 range gives 0x8000000000000000, the x87
// integer indefinite. Go leaves those conversions implementation-defined
// (amd64 happens to agree, wasm does not), so ftol checks for them itself.
func ftol(x float32) int64 {
	// _ftol2 (FUN_00463b2c) gives the x87 integer indefinite for a NaN and
	// for x outside [−2^63, 2^63). 2^63 is exact in float32, so the bounds
	// below are exact too, and −2^63 itself is in range.
	if math.IsNaN(float64(x)) || x >= 1<<63 || x < -(1<<63) {
		return math.MinInt64
	}
	// In range, the conversion truncates toward zero, as _ftol2 does.
	return int64(x)
}
