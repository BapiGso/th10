package anmdraw

import (
	"th10/anm"
	"th10/gfx"
)

// draw9 is render mode 9, FUN_00444ce0 after Draw's flag and alpha test:
// op 84's ring, which anm.VM.fillRing built at the end of the object's
// last frame, as one triangle strip drawn on its own, outside the batch.
// Draw passes it buf = VM+0x358 and count = 2·I[0] (0x44523e-0x44524e),
// I[0] read now, not at the fill.
//
// Panics, before any device call, if I[0] < 1 (a primitive count of
// 2·I[0] − 2 below 0 goes to DrawPrimitiveUP as a huge UINT) or if
// I[0] ≥ 2 and 2·I[0] > len(vm.Ring) (the strip would read vertices the
// fill did not write: malloc garbage, or past the buffer). I[0] == 1 reads
// no vertex, so any vm.Ring will do. Otherwise, in order:
//
//  1. Flush (0x444d10-0x444d1a).
//  2. Texture: if spr.Tex != r.tex, r.tex = spr.Tex and SetTexture(spr.Tex)
//     (0x444d1f-0x444d42). No flush; the batch is empty.
//  3. Vertex type: if r.vtype != vtypeRing, r.vtype = vtypeRing
//     (0x444d48-0x444d64). The original also sets FVF 0x144 there.
//  4. blendFilter (0x444d6d).
//  5. The original makes DIFFUSE stage 0's ALPHAARG2 and COLORARG2
//     (0x444d80, 0x444d94), undoing mode 8's TEXTUREFACTOR: texture times
//     diffuse, the only combination gfx has, so no device call.
//  6. If I[0] ≥ 2, DrawTL(gfx.TriangleStrip, vs) (DrawPrimitiveUP with
//     2·I[0] − 2 primitives, 0x444db3), vs being the first 2·I[0]
//     vertices of vm.Ring as below. With I[0] == 1 the count is 0 and
//     nothing is drawn; steps 1-5 still happen.
//
// Each ring vertex v goes to the strip as
//
//	gfx.TLVertex{X: v.X, Y: v.Y, Z: v.Z, RHW: 1, Diffuse: v.Diffuse,
//	    Specular: noSpecular, U: spr.U0 + v.U (even index) or
//	    spr.U1 + v.U (odd index), V: v.V}
//
// Shift, the cull, the snap and the tint do not apply.
func (r *Renderer) draw9(vm *anm.VM, spr *Sprite) {
	n := vm.I[0]
	if n < 1 {
		panic("anmdraw: draw9 with I[0] < 1")
	}
	if n >= 2 && 2*int(n) > len(vm.Ring) {
		panic("anmdraw: draw9 reads past the ring")
	}

	// 1. Flush (0x444d10-0x444d1a).
	r.Flush()
	// 2. Texture (0x444d1f-0x444d42).
	if spr.Tex != r.tex {
		r.tex = spr.Tex
		r.Dev.SetTexture(spr.Tex)
	}
	// 3. Vertex type (0x444d48-0x444d64), no device call.
	if r.vtype != vtypeRing {
		r.vtype = vtypeRing
	}
	// 4. Blend and filter (0x444d6d).
	r.blendFilter(vm)
	// 5. DIFFUSE's arguments (0x444d80, 0x444d94): no device call, see above.
	// 6. The strip (0x444db3).
	if n >= 2 {
		r.strip = r.strip[:0]
		for i, v := range vm.Ring[:2*int(n)] {
			u := spr.U0 + v.U
			if i%2 == 1 {
				u = spr.U1 + v.U
			}
			r.strip = append(r.strip, gfx.TLVertex{
				X: v.X, Y: v.Y, Z: v.Z, RHW: 1, Diffuse: v.Diffuse, Specular: noSpecular,
				U: u, V: v.V,
			})
		}
		r.Dev.DrawTL(gfx.TriangleStrip, r.strip)
	}
}
