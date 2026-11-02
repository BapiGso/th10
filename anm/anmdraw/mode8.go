package anmdraw

import (
	"math"

	"th10/anm"
	"th10/gfx"
)

// Render mode 8 (FUN_00444760, docs/research/m4_r7.md R7-A, R7-B, R7-F):
// the sprite as a square in the 3D scene, drawn on its own through the
// world matrix rather than batched. The derivation and the misreadings it
// corrects are in m4_r7.md "R7 审校更正".

// quad8 is mode 8's vertex buffer (manager+0x3ada74, FVF XYZ|TEX1, filled
// by FUN_004462f0 from +0x3ada78): a 256×256 square about the origin in
// triangle strip order. base scales it to the sprite's size.
var quad8 = [4]gfx.Vertex3D{
	{X: -128, Y: -128, U: 0, V: 0},
	{X: 128, Y: -128, U: 1, V: 0},
	{X: -128, Y: 128, U: 0, V: 1},
	{X: 128, Y: 128, U: 1, V: 1},
}

// draw8 is FUN_00444760 after Draw's flag and alpha test. In order:
//
//  1. Flush (0x4447a5-0x4447b2).
//
//  2. If vm.MatStale: vm.Mat = base(spr), vm.MatStale = false. The
//     original does this in the sprite op (anm.VM.Mat); doing it here,
//     before anything reads Mat, is the same.
//
//  3. Unless Flags&0x4000, if Flags&0xc != 0 (anm's rotation and scale
//     dirty bits), rebuild vm.Mat (0x4447cf-0x444899):
//
//     M = base(spr)
//     M[0][0] = Scale[0]·M[0][0]      M[1][1] = Scale[1]·M[1][1]
//     Flags &^= 8
//     M = gfx.Mul(M, gfx.RotationX(Rot[0])) if Rot[0] != 0
//     M = gfx.Mul(M, gfx.RotationY(Rot[1])) if Rot[1] != 0
//     M = gfx.Mul(M, gfx.RotationZ(Rot[2])) if Rot[2] != 0
//     Flags &^= 4
//
//     Each test is an FUCOMPP against 0 that rotates on a NaN, as Go's
//     != does. Otherwise vm.Mat is used as it is, from an earlier frame.
//
//  4. The world matrix T is vm.Mat with its translation row replaced.
//     With hw = |(spr.W·Scale[0])·0.5| and hh = |(spr.H·Scale[1])·0.5|:
//
//     ax 0: T[3][0] = (Pos2.x + Offset.x) + Pos.x
//     ax 1: T[3][0] = ((Pos2.x + Offset.x) + Pos.x) − hw
//     ax 2: T[3][0] = ((hw + Pos2.x) + Offset.x) + Pos.x
//     ax 3: T[3][0] = vm.Mat[3][0]
//
//     T[3][1] the same from the y components, hh and ay, and
//     T[3][2] = (Pos2.z + Pos.z) + Offset.z (0x444999-0x4449bb; the
//     store at 0x444982 is overwritten). The anchor signs are the
//     opposite of the 2D quads' (0x4448b0-0x44497e): keep them.
//
//  5. states3D.
//
//  6. SetTransform(gfx.World, T).
//
//  7. Texture: if spr.Tex != r.tex, r.tex = spr.Tex and
//     SetTexture(spr.Tex). No flush; the batch is empty.
//
//  8. Texture matrix, if spr != r.sprite or UV[0] != 0 (NaN counts):
//
//     r.sprite = spr
//     m = texBlock(spr)
//     m[2][0] = spr.U0 + UV[0]      m[2][1] = spr.V0 + UV[1]
//     SetTransform(gfx.Texture0, m)
//
//     UV[1] is not tested: the binary compares UV[0] with 0 twice
//     (0x4449fe and 0x444a10), so a vertical scroll alone does not move
//     a sprite that is already set. Keep it.
//
//  9. Vertex type: if r.vtype != vtype3D, r.vtype = vtype3D. The original
//     also sets the stream source and FVF 0x102 here and makes
//     TEXTUREFACTOR stage 0's COLORARG2 and ALPHAARG2; Flush puts DIFFUSE
//     back only when it submits a batch, which needs the vertex type to
//     change first, so the arguments are TEXTUREFACTOR exactly while
//     r.vtype is vtype3D.
//
//  10. Draw3D(gfx.TriangleStrip, quad8 with every Diffuse = r.tfDev):
//     texture times the device's TEXTUREFACTOR, which is what gfx's
//     texture times diffuse computes.
//
// Every operation rounds to float32, as in the 2D quads.
func (r *Renderer) draw8(vm *anm.VM, spr *Sprite) {
	r.Flush()

	if vm.MatStale {
		vm.Mat = base(spr)
		vm.MatStale = false
	}

	if vm.Flags&0x4000 == 0 && vm.Flags&0xc != 0 {
		m := base(spr)
		m[0][0] = vm.Scale[0] * m[0][0]
		m[1][1] = vm.Scale[1] * m[1][1]
		vm.Flags &^= 8
		if vm.Rot[0] != 0 {
			m = gfx.Mul(m, gfx.RotationX(vm.Rot[0]))
		}
		if vm.Rot[1] != 0 {
			m = gfx.Mul(m, gfx.RotationY(vm.Rot[1]))
		}
		if vm.Rot[2] != 0 {
			m = gfx.Mul(m, gfx.RotationZ(vm.Rot[2]))
		}
		vm.Flags &^= 4
		vm.Mat = m
	}

	// World matrix: the translation row from the anchors (0x444999-0x4449bb).
	hw := float32(math.Abs(float64(spr.W * vm.Scale[0] * 0.5)))
	hh := float32(math.Abs(float64(spr.H * vm.Scale[1] * 0.5)))
	t := vm.Mat
	switch vm.Flags >> 18 & 3 {
	case 0:
		t[3][0] = (vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]
	case 1:
		t[3][0] = ((vm.Pos2[0] + vm.Offset[0]) + vm.Pos[0]) - hw
	case 2:
		t[3][0] = ((hw + vm.Pos2[0]) + vm.Offset[0]) + vm.Pos[0]
	}
	switch vm.Flags >> 20 & 3 {
	case 0:
		t[3][1] = (vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]
	case 1:
		t[3][1] = ((vm.Pos2[1] + vm.Offset[1]) + vm.Pos[1]) - hh
	case 2:
		t[3][1] = ((hh + vm.Pos2[1]) + vm.Offset[1]) + vm.Pos[1]
	}
	t[3][2] = (vm.Pos2[2] + vm.Pos[2]) + vm.Offset[2]

	r.states3D(vm)
	r.Dev.SetTransform(gfx.World, t)

	// Texture, then the texture matrix for the sprite's rectangle and scroll.
	if spr.Tex != r.tex {
		r.tex = spr.Tex
		r.Dev.SetTexture(spr.Tex)
	}
	if spr != r.sprite || vm.UV[0] != 0 {
		r.sprite = spr
		m := texBlock(spr)
		m[2][0] = spr.U0 + vm.UV[0]
		m[2][1] = spr.V0 + vm.UV[1]
		r.Dev.SetTransform(gfx.Texture0, m)
	}

	// Vertex type: no device call (see the doc above).
	if r.vtype != vtype3D {
		r.vtype = vtype3D
	}

	vs := quad8
	for i := range vs {
		vs[i].Diffuse = r.tfDev
	}
	r.Dev.Draw3D(gfx.TriangleStrip, vs[:])
}

// base is the matrix the sprite op leaves at VM+0x23c (FUN_0043e5a0,
// 0x43e638-0x43e6ce): the identity with [0][0] = spr.W·2⁻⁸ and
// [1][1] = spr.H·2⁻⁸ (0x470b7c is 1/256), so that it scales quad8 to the
// sprite's size.
func base(spr *Sprite) gfx.Mat4 {
	m := gfx.Identity
	m[0][0] = spr.W * (1.0 / 256)
	m[1][1] = spr.H * (1.0 / 256)
	return m
}

// texBlock is the texture matrix the sprite op leaves at VM+0x2bc
// (0x43e64e-0x43e6f5): the identity with
//
//	[0][0] = (f1/Wd)·spr.W      [1][1] = (f2/Hd)·spr.H
//
// the quotient rounded first, where f1, f2 are the sprite record's +0x38,
// +0x3c (NewSheet's f1, f2: 1 for every TH10 sheet, as Wd = We) and Wd, Hd
// its +0x1c, +0x18 (float32(spr.Tex.W), float32(spr.Tex.H)). With 1 for f1
// and f2 that is float32(float32(1/Wd)·spr.W) and float32(float32(1/Hd)·
// spr.H). It maps quad8's 0..1 to the sprite's width and height in
// texture coordinates; draw8 adds the offset.
func texBlock(spr *Sprite) gfx.Mat4 {
	wd, hd := float32(spr.Tex.W), float32(spr.Tex.H)
	m := gfx.Identity
	m[0][0] = float32(1) / wd * spr.W
	m[1][1] = float32(1) / hd * spr.H
	return m
}

// states3D is FUN_004423e0, blendFilter's 3D twin, in its order:
//
//  1. Blend (0x4423e4-0x44244c): with b = Flags>>4 & 3, if b != r.blend:
//     Flush, r.blend = b, then SetBlend(gfx.BlendAlpha) for b 0,
//     SetBlend(gfx.BlendAdd) for b 1 and 2, nothing for b 3. Unlike
//     blendFilter, b 2 is set; after a 2D object with b 2 the shared cache
//     says 2 while the device still blends as before, and a 3D object
//     with b 2 then sets nothing (m4_r7.md F.3). Keep it.
//  2. TEXTUREFACTOR (0x442448-0x442527): with c = r.colour(vm), if
//     c != r.tfactor: Flush, r.tfactor = c, r.tfDev = c.
//  3. filter (0x44252d-0x442592).
//
// Its flushes find the batch empty (draw8 flushed first) but are kept.
func (r *Renderer) states3D(vm *anm.VM) {
	// Blend: 2 is set here, unlike blendFilter.
	b := uint8(vm.Flags >> 4 & 3)
	if b != r.blend {
		r.Flush()
		r.blend = b
		switch b {
		case 0:
			r.Dev.SetBlend(gfx.BlendAlpha)
		case 1, 2:
			r.Dev.SetBlend(gfx.BlendAdd)
		}
	}
	// TEXTUREFACTOR.
	if c := r.colour(vm); c != r.tfactor {
		r.Flush()
		r.tfactor = c
		r.tfDev = c
	}
	r.filter(vm)
}
