// Package stddraw is STD's drawing, the stage background's 3D scene. Draw
// node 7 (FUN_00403060 → FUN_00402850) draws passes 0-7; draw node 0xa
// (FUN_00403070 → FUN_00402ca0) draws the manager's layers 17 and 18, then
// passes 8-11. Each pass (FUN_00403a30) culls STD's faces against the field
// camera (FUN_00403090) and has the ANM manager draw the quads of those
// left.
//
// Dormant state: the STD object's +0x2a18 bits 0x2, 0x4 and 0x8, and what
// hangs off them, are not ported:
//
//   - the early returns on bit 0x8 and the bit-0x4 skips;
//   - the bit-0x4 transition (FUN_0043c8b0, FUN_00405410 on the timer at
//     +0x2a1c, +0x2a34's black clear);
//   - +0x1eeb and +0x1ee8's tint;
//   - +0x1ee4's counter;
//   - node 0xa's timer tail (FUN_0044bf40), which runs only while that
//     timer (+0x2a20) is positive.
//
// The constructor (FUN_00402230) sets bit 0 alone, and no writer of the
// other bits or of the timer is traced. Bit 0 is cleared only by the
// bit-0x4 transition, so it stays set and both nodes always draw their
// passes.
//
// Evidence: .tmp/ecl_dump/render/func_402850.asm, func_402ca0.asm,
// func_403a30.asm, render3/func_403090.asm, s2b_aux/func_405300.asm.
package stddraw

import (
	"image"

	"th10/gfx"
	"th10/std"
	"th10/supervisor"
)

// cullDist2 is +0x1ee0, which the constructor sets to 0x4b12a310
// (9610000 = 3100²): the squared distance beyond which an entry is culled.
const cullDist2 float32 = 9610000

// Drawer draws an STD VM's scene.
type Drawer struct {
	STD *std.VM

	// Dim is +0x1eec. When it is set, the tint tail leaves the manager
	// tinting every later quad 0xff404040 rather than untinted. Its writer
	// is not traced, so it stays false unless something sets it.
	Dim bool

	// The counters +0x2a0c, +0x2a10 and +0x2a14 hold the faces drawn, the
	// faces culled and the quads drawn by the passes since the last Node7,
	// which zeroes them (Node0A does not). Their readers are not traced.
	Faces, Culled, Quads int32
}

// Node7 is FUN_00402850(std) without the dormant paths (see the package
// comment). Each ANM.Flush below is one of the original's FUN_00442f50
// calls. They find the batch empty except where noted, but are kept. In
// order:
//
//  1. load.
//
//  2. ANM.Flush(); SetZWrite(true)                        ZWRITEENABLE 1
//     ANM.Flush(); SetZFunc(gfx.CmpLessEqual)              ZFUNC 4
//     ANM.Flush() three times, then
//     SetFog(gfx.Fog{Mode: gfx.FogVertexLinear, Color: Cam.FogColor,
//     Start: Cam.FogNear, End: Cam.FogFar}), where Cam is d.STD.Cam.
//     The original makes three render-state calls (FOGCOLOR, FOGSTART,
//     FOGEND) with a flush before each. The mode is the one Init set.
//
//  3. Clear(gfx.ClearZBuffer, 0, 1), then
//     Clear(gfx.ClearTarget, Cam.FogColor&0xffffff, 1,
//     image.Rect(32, 16, 416, 464)). The colour's alpha byte is zero.
//
//  4. Faces, Culled, Quads = 0.
//
//  5. If d.STD.Slots[0].Sprite >= 0 (slot 0's sprite pointer, VM+0x394,
//     is set; the other slots are not tested here):
//     sv.Use(&sv.Field); sv.Is2D = false         FUN_00405300 with EBX 0
//     sv.Fog(false)                              FUN_00420cd0
//     ANM.Flush(); ANM.Flush(); SetZWrite(false)
//     for each slot in order with Sprite >= 0: sv.ANM.Draw(slot)
//     ANM.Flush(); SetZWrite(true)               submits the slots' batch
//     sv.Select(&sv.Field); sv.Perspective(&sv.Field)
//     SetViewport(sv.Field.Viewport); sv.Is2D = false
//     The slots are drawn under Field's flat matrices.
//
//  6. sv.Fog(true); pass(sv, p) for p = 0..7; ANM.Flush().
//
//  7. tint.
//
//  8. ANM.Flush(); SetZWrite(false); ANM.Flush(); SetZFunc(gfx.CmpAlways).
func (d *Drawer) Node7(sv *supervisor.Supervisor) {
	d.load(sv)

	// FUN_00402850 step 2: the fog state is set with three flushes before the call.
	sv.ANM.Flush()
	sv.Dev.SetZWrite(true)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpLessEqual)
	sv.ANM.Flush()
	sv.ANM.Flush()
	sv.ANM.Flush()
	sv.Dev.SetFog(gfx.Fog{Mode: gfx.FogVertexLinear, Color: d.STD.Cam.FogColor,
		Start: d.STD.Cam.FogNear, End: d.STD.Cam.FogFar})

	// FUN_00402850 step 3.
	sv.Dev.Clear(gfx.ClearZBuffer, 0, 1)
	sv.Dev.Clear(gfx.ClearTarget, d.STD.Cam.FogColor&0xffffff, 1, image.Rect(32, 16, 416, 464))

	// FUN_00402850 step 4.
	d.Faces, d.Culled, d.Quads = 0, 0, 0

	// FUN_00402850 step 5: the slots are drawn under Field's flat matrices.
	if d.STD.Slots[0].Sprite >= 0 {
		sv.Use(&sv.Field)
		sv.Is2D = false
		sv.Fog(false)
		sv.ANM.Flush()
		sv.ANM.Flush()
		sv.Dev.SetZWrite(false)
		for i := range d.STD.Slots {
			if s := &d.STD.Slots[i]; s.Sprite >= 0 {
				sv.ANM.Draw(s)
			}
		}
		sv.ANM.Flush()
		sv.Dev.SetZWrite(true)
		sv.Select(&sv.Field)
		sv.Perspective(&sv.Field)
		sv.Dev.SetViewport(sv.Field.Viewport)
		sv.Is2D = false
	}

	// FUN_00402850 step 6.
	sv.Fog(true)
	for p := range 8 {
		d.pass(sv, p)
	}
	sv.ANM.Flush()

	// FUN_00402850 step 7.
	d.tint(sv)

	// FUN_00402850 step 8.
	sv.ANM.Flush()
	sv.Dev.SetZWrite(false)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpAlways)
}

// Node0A is FUN_00402ca0(std) without the dormant paths. layer(l) draws
// the manager's layer l (FUN_00448980, anmdraw.Renderer.DrawLayer). In
// order:
//
//	load
//	sv.Fog(true)
//	ANM.Flush(); SetZWrite(false)
//	ANM.Flush(); SetZFunc(gfx.CmpAlways)
//	layer(17)
//	ANM.Flush(); SetZFunc(gfx.CmpLessEqual)
//	sv.Fog(false)
//	layer(18)
//	ANM.Flush(); SetZFunc(gfx.CmpLessEqual)
//	ANM.Flush() three times; SetFog(as in Node7 step 2)
//	ANM.Flush(); SetZWrite(false)
//	sv.Fog(true)
//	pass(sv, p) for p = 8..11
//	ANM.Flush()
//	tint
//	ANM.Flush(); SetZWrite(false); ANM.Flush(); SetZFunc(gfx.CmpAlways)
//	ANM.Flush(); SetZWrite(false); ANM.Flush(); SetZFunc(gfx.CmpAlways)
//	sv.Fog(false)
//
// The ZWRITE/ZFUNC line is repeated, as in the original (the dormant timer
// tail sits between the two).
func (d *Drawer) Node0A(sv *supervisor.Supervisor, layer func(l int)) {
	d.load(sv)
	sv.Fog(true)
	sv.ANM.Flush()
	sv.Dev.SetZWrite(false)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpAlways)
	layer(17)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpLessEqual)
	sv.Fog(false)
	layer(18)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpLessEqual)
	sv.ANM.Flush()
	sv.ANM.Flush()
	sv.ANM.Flush()
	sv.Dev.SetFog(gfx.Fog{Mode: gfx.FogVertexLinear, Color: d.STD.Cam.FogColor,
		Start: d.STD.Cam.FogNear, End: d.STD.Cam.FogFar})
	sv.ANM.Flush()
	sv.Dev.SetZWrite(false)
	sv.Fog(true)
	for p := 8; p < 12; p++ {
		d.pass(sv, p)
	}
	sv.ANM.Flush()
	d.tint(sv)
	sv.ANM.Flush()
	sv.Dev.SetZWrite(false)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpAlways)
	sv.ANM.Flush()
	sv.Dev.SetZWrite(false)
	sv.ANM.Flush()
	sv.Dev.SetZFunc(gfx.CmpAlways)
	sv.Fog(false)
}

// load is how both nodes start, with Cam = d.STD.Cam:
//
//	ANM.Flush()
//	sv.Field.Pos, Facing, Up, FOV = Cam.Pos, Facing, Up, FOV
//	sv.Field.Delta, FogNear, FogFar, FogBytes, FogColor = Cam's
//	sv.Field.Eye = zero
//	sv.Select(&sv.Field); sv.Perspective(&sv.Field)
//	SetViewport(sv.Field.Viewport); sv.Is2D = false
//
// The original saves Field's Shift into STD's camera block (+0x2b34,
// +0x2b38) and copies the whole block over Field (0x46 dwords to
// 0x491d7c). The block began as a copy of Field (FUN_00402230), and its
// viewport is never written, so the copy changes Field's Pos, Facing, Up,
// Eye (+0x3c, written only by op 12, which no TH10 .std uses), FOV,
// Delta and the fog fields. Its View, Proj and Right are what Perspective
// rebuilds next.
func (d *Drawer) load(sv *supervisor.Supervisor) {
	cam := &d.STD.Cam
	sv.ANM.Flush()
	sv.Field.Pos, sv.Field.Facing, sv.Field.Up, sv.Field.FOV = cam.Pos, cam.Facing, cam.Up, cam.FOV
	sv.Field.Delta, sv.Field.FogNear, sv.Field.FogFar = cam.Delta, cam.FogNear, cam.FogFar
	sv.Field.FogBytes, sv.Field.FogColor = cam.FogBytes, cam.FogColor
	sv.Field.Eye = [3]float32{}
	sv.Select(&sv.Field)
	sv.Perspective(&sv.Field)
	sv.Dev.SetViewport(sv.Field.Viewport)
	sv.Is2D = false
}

// pass is FUN_00403a30(std, p):
//
//  1. sv.Select(&sv.Field); sv.Perspective(&sv.Field);
//     SetViewport(sv.Field.Viewport); sv.Is2D = false. The original also
//     sets manager +0x3ada6c to 1, which has no reader here.
//
//  2. For each face f of d.STD.File().Faces in order, with
//     e = &Entries[f.Entry]:
//
//     a. Skip f unless int8(e.Unknown) == p (entry byte +2, sign-extended).
//
//     b. If culled(e, f.Pos, &sv.Field, cullDist2): Culled++, next face.
//
//     c. (The original ORs 2 into entry byte +3. No reader is known, so it
//     is not modelled.)
//
//     d. For each quad j of e in order with q.Type == 0 (other types are
//     stepped over: nothing is touched or counted), with
//     vm = d.STD.Quad(f.Entry, j) and mode = vm.Flags>>22 & 0xf:
//
//     if mode >= 4:
//     vm.Offset[k] = q.Pos[k] + f.Pos[k] for k = 0..2     (+0x340..+0x348)
//     spr = sv.ANM.Sheet(vm.File).Sprites[vm.Sprite]
//     if q.Size[0] != 0: vm.Scale[0] = q.Size[0] / spr.W; vm.Flags |= 8
//     if q.Size[1] != 0: vm.Scale[1] = q.Size[1] / spr.H; vm.Flags |= 8
//     sv.Fog(mode == 8)
//     sv.ANM.Draw(vm)
//     Quads++
//
//     Each Size test is an FUCOMPP against 0 that divides on a NaN, as
//     Go's != does. Flags bit 8 is anm's scale-dirty bit: it makes mode 8
//     rebuild vm.Mat with the new scale. Mode 8 is drawn with fog and
//     every other mode without.
//
//     e. Faces++.
//
// A face's entry is not range-checked, and neither is a quad VM's sprite
// (the original dereferences it), so either one out of range panics.
func (d *Drawer) pass(sv *supervisor.Supervisor, p int) {
	// FUN_00403a30 step 1.
	sv.Select(&sv.Field)
	sv.Perspective(&sv.Field)
	sv.Dev.SetViewport(sv.Field.Viewport)
	sv.Is2D = false

	// FUN_00403a30 step 2.
	file := d.STD.File()
	for _, f := range file.Faces {
		e := &file.Entries[f.Entry]
		// FUN_00403a30 step 2a.
		if int(int8(e.Unknown)) != p {
			continue
		}
		// FUN_00403a30 step 2b.
		if culled(e, f.Pos, &sv.Field, cullDist2) {
			d.Culled++
			continue
		}
		// FUN_00403a30 step 2c (entry byte +3 |= 2) is not modelled: no reader is known.
		// FUN_00403a30 step 2d.
		for j := range e.Quads {
			q := &e.Quads[j]
			if q.Type != 0 {
				continue
			}
			vm := d.STD.Quad(int(f.Entry), j)
			mode := vm.Flags >> 22 & 0xf
			if mode >= 4 {
				for k := range 3 {
					vm.Offset[k] = q.Pos[k] + f.Pos[k]
				}
				spr := sv.ANM.Sheet(vm.File).Sprites[vm.Sprite]
				if q.Size[0] != 0 {
					vm.Scale[0] = q.Size[0] / spr.W
					vm.Flags |= 8
				}
				if q.Size[1] != 0 {
					vm.Scale[1] = q.Size[1] / spr.H
					vm.Flags |= 8
				}
			}
			sv.Fog(mode == 8)
			sv.ANM.Draw(vm)
			d.Quads++
		}
		// FUN_00403a30 step 2e.
		d.Faces++
	}
}

// tint is the tail both nodes share (0x402c34, 0x402f21): sv.ANM.TintOn =
// false and sv.ANM.Tint = 0x80808080, or TintOn = true and Tint =
// 0xff404040 if d.Dim. There is no flush: quads already in the batch keep
// their colour.
func (d *Drawer) tint(sv *supervisor.Supervisor) {
	if d.Dim {
		sv.ANM.TintOn, sv.ANM.Tint = true, 0xff404040
	} else {
		sv.ANM.TintOn, sv.ANM.Tint = false, 0x80808080
	}
}

// culled is FUN_00403090(e, at, c, dist2): whether entry e placed at at is
// out of sight of camera c. In this order:
//
//  1. Distance: d[k] = (e.Pos[k] + at[k]) − (c.Eye[k] + c.Pos[k]). Culled
//     if ((d[2]·d[2] + d[1]·d[1]) + d[0]·d[0]) > dist2. A NaN sum is not
//     culled here.
//
//  2. The box's corners, h[k] = e.Size[k]·0.5, in the order
//     p0 (+,+,+), p1 (+,+,−), p2 (+,−,+), p3 (+,−,−),
//     p4 (−,+,+), p5 (−,+,−), p6 (−,−,+), p7 (−,−,−),
//     where + is h[k] + e.Pos[k] and − is e.Pos[k] − h[k]. Each corner is
//     projected as gfx.Vec3Project(p, c.Viewport, c.Proj, c.View,
//     gfx.Translation(at[0], at[1], at[2])). The original projects them
//     in one D3DXVec3ProjectArray call.
//
//  3. Bounds: start from minX 424, maxX 24, minY 472, maxY 8. For each
//     projected point in order with 0 <= z <= 1 (z == 1 counts, a NaN z
//     does not):
//
//     if x < minX { minX = x } else if x > maxX { maxX = x }
//     if y < minY { minY = y } else if y > maxY { maxY = y }
//
//     The else is the original's (0x40341d-0x40347f and its seven
//     repeats). A point that lowers minX is never tested against maxX, so a
//     lone counted point leaves maxX at 24 and the box is culled. Keep it.
//
//  4. Visible, so not culled, iff maxX >= 32 && minX <= 416 &&
//     maxY >= 16 && minY <= 464. Any NaN culls.
//
// Every operation rounds to float32.
func culled(e *std.Entry, at [3]float32, c *supervisor.Camera, dist2 float32) bool {
	// FUN_00403090 step 1.
	var dv [3]float32
	for k := range dv {
		dv[k] = (e.Pos[k] + at[k]) - (c.Eye[k] + c.Pos[k])
	}
	if float32(float32(float32(dv[2]*dv[2])+float32(dv[1]*dv[1]))+float32(dv[0]*dv[0])) > dist2 {
		return true
	}

	// FUN_00403090 step 2: plus[i][k] is whether corner i takes +h on axis k.
	var h [3]float32
	for k := range h {
		h[k] = float32(e.Size[k] * 0.5)
	}
	plus := [8][3]bool{
		{true, true, true}, {true, true, false}, {true, false, true}, {true, false, false},
		{false, true, true}, {false, true, false}, {false, false, true}, {false, false, false},
	}
	tr := gfx.Translation(at[0], at[1], at[2])

	// FUN_00403090 step 3. The else-if is the original's (0x40341d-0x40347f): a point
	// that lowers minX is not tested against maxX, and likewise for y.
	minX, maxX, minY, maxY := float32(424), float32(24), float32(472), float32(8)
	for _, pl := range plus {
		var p [3]float32
		for k := range p {
			if pl[k] {
				p[k] = h[k] + e.Pos[k]
			} else {
				p[k] = e.Pos[k] - h[k]
			}
		}
		q := gfx.Vec3Project(p, c.Viewport, c.Proj, c.View, tr)
		x, y, z := q[0], q[1], q[2]
		if !(0 <= z && z <= 1) {
			continue
		}
		if x < minX {
			minX = x
		} else if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		} else if y > maxY {
			maxY = y
		}
	}

	// FUN_00403090 step 4: visible iff the box meets the field rectangle; a NaN fails the test.
	return !(maxX >= 32 && minX <= 416 && maxY >= 16 && minY <= 464)
}
