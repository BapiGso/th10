package anmdraw

import "th10/gfx"

// Rect is FUN_0043bda0, the manager's flat rectangle: the axis-aligned
// rectangle [v[0], v[2]] × [v[1], v[3]] in back-buffer pixels, filled with
// c. The player's faith-hold bars are its only caller here (FUN_00426360).
// In order:
//
//  1. Flush (0x43bdaa).
//  2. Four corners (0x43bdaf-0x43be87): (v[0], v[1]), (v[2], v[1]),
//     (v[0], v[3]), (v[2], v[3]) in that order, each with Z 0, RHW 1 and
//     Diffuse c; Specular is noSpecular and U, V are 0, as no field of the
//     original's FVF 0x44 (XYZRHW | DIFFUSE) carries them.
//  3. Dev.SetTexture(nil). The original binds nothing: it sets stage 0's
//     COLOROP and ALPHAOP to SELECTARG1 with DIFFUSE arguments, so the bound
//     texture is ignored, which is what gfx's untextured draw does. The
//     device is left untextured; the cache reset in step 6 makes the next
//     object bind its own texture first.
//  4. Dev.SetBlend(gfx.BlendAlpha): SetRenderState(DESTBLEND, INVSRCALPHA)
//     at 0x43bed7. The SRCBLEND every blend uses stays SRCALPHA.
//  5. Dev.DrawTL(gfx.TriangleStrip, the four corners): DrawPrimitiveUP(5, 2,
//     …, 0x14) at 0x43bf00. The corners are not snapped or shifted (Shift),
//     and nothing is culled.
//  6. Invalidate (0x43bf06-0x43bf30). The texture stage states restored
//     afterwards (MODULATE with TEXTURE and DIFFUSE) are gfx's only
//     combination.
//
// The corners slice handed to DrawTL must not be kept by the device, as
// with the batch.
func (r *Renderer) Rect(v [4]float32, c gfx.Color) {
	panic("anmdraw: Rect not implemented")
}
