package gfx

import (
	"image"
	"math"
)

// The CPU reference of the fragment stage that shaderSrc transcribes; see
// the comment at the top of shader.go.

// alpha8 is a fragment alpha as the alpha test compares it with ALPHAREF:
// ⌊float32(a·255) + 0.5⌋ clamped to 0..255. Rounding to nearest rather than
// truncating is an assumption for L3 to confirm.
func alpha8(a float32) int {
	v := int(math.Floor(float64(float32(a*255) + 0.5)))
	return min(max(v, 0), 255)
}

// depthColor is how the depth image holds the D16 value d, and what the
// depth pass outputs for it: (⌊d/256⌋/255, (d mod 256)/255, 0, 1), each
// a float32 quotient. Alpha 1 keeps Ebiten's premultiplied storage from
// altering R and G, so the stored bytes are d>>8 and d&0xff; the shader
// decodes a texel (r, g, …) as ⌊r·255 + 0.5⌋·256 + ⌊g·255 + 0.5⌋.
func depthColor(d uint16) [4]float32 {
	return [4]float32{float32(d>>8) / 255, float32(d&0xff) / 255, 0, 1}
}

// fragIn is one fragment's interpolated inputs and the depth buffer's value
// at its pixel.
type fragIn struct {
	Diffuse       [4]float32
	UQ, VQ, Q, FQ float32
	Z             float32 // screen-linear depth (src0Pos.x)
	Stored        uint16  // D16 value at the pixel, read only when DepthTest is set
}

// fragment is the fragment stage of one pass:
//
//  1. c = shade(u, tex, in.Diffuse, in.UQ, in.VQ, in.Q, in.FQ).
//  2. Alpha test, GREATEREQUAL: discard if alpha8(c[3]) < AlphaRef. c[3] is
//     the modulated alpha, which fog and premultiplication leave alone.
//  3. Depth test, LESSEQUAL: discard if DepthTest is set and
//     depth16(in.Z) > in.Stored.
//  4. Pass 0 outputs c; Pass 1 outputs depthColor(depth16(in.Z)).
//
// keep is false for a discarded fragment, and out is then zero.
func fragment(u uniforms, tex *image.NRGBA, in fragIn) (out [4]float32, keep bool) {
	c := shade(u, tex, in.Diffuse, in.UQ, in.VQ, in.Q, in.FQ)
	if alpha8(c[3]) < u.AlphaRef {
		return [4]float32{}, false
	}
	z := depth16(in.Z)
	if u.DepthTest == 1 && z > in.Stored {
		return [4]float32{}, false
	}
	if u.Pass == 1 {
		return depthColor(z), true
	}
	return c, true
}

// texel samples tex at (u, v) as Direct3D 9 does, in straight-alpha 0..1
// floats (R, G, B, A). With s = u·W and t = v·H:
//
//   - FilterPoint reads texel (⌊s⌋, ⌊t⌋).
//   - FilterLinear reads the four texels around (s − 0.5, t − 0.5): with
//     i = ⌊s − 0.5⌋, a = (s − 0.5) − i and likewise j, b for t, it is
//     lerp(lerp(T(i,j), T(i+1,j), a), lerp(T(i,j+1), T(i+1,j+1), a), b),
//     per channel, with lerp(x, y, a) = x + (y − x)·a.
//
// Every texel index is addressed before the read: AddressWrap takes it
// modulo the size (into 0..size−1, negative indices too), AddressClamp
// clamps it to 0..size−1.
func texel(tex *image.NRGBA, u, v float32, a Address, f Filter) [4]float32 {
	s := float32(u * float32(tex.Rect.Dx()))
	t := float32(v * float32(tex.Rect.Dy()))
	if f == FilterPoint {
		return texelAt(tex, a, int(math.Floor(float64(s))), int(math.Floor(float64(t))))
	}
	x := s - 0.5
	y := t - 0.5
	fx := float32(math.Floor(float64(x)))
	fy := float32(math.Floor(float64(y)))
	i, j := int(fx), int(fy)
	ax, by := x-fx, y-fy
	top := lerp4(texelAt(tex, a, i, j), texelAt(tex, a, i+1, j), ax)
	bot := lerp4(texelAt(tex, a, i, j+1), texelAt(tex, a, i+1, j+1), ax)
	return lerp4(top, bot, by)
}

// texelAt reads texel (i, j) after addressing it, as straight-alpha floats.
func texelAt(tex *image.NRGBA, a Address, i, j int) [4]float32 {
	i = addressIndex(i, tex.Rect.Dx(), a)
	j = addressIndex(j, tex.Rect.Dy(), a)
	o := tex.PixOffset(tex.Rect.Min.X+i, tex.Rect.Min.Y+j)
	p := tex.Pix[o : o+4 : o+4]
	return [4]float32{float32(p[0]) / 255, float32(p[1]) / 255, float32(p[2]) / 255, float32(p[3]) / 255}
}

// addressIndex maps a texel index into [0, n) under the address mode.
func addressIndex(i, n int, a Address) int {
	if a == AddressClamp {
		return min(max(i, 0), n-1)
	}
	i %= n
	if i < 0 {
		i += n
	}
	return i
}

// lerp4 is lerp(x, y, a) per channel.
func lerp4(x, y [4]float32, a float32) [4]float32 {
	var r [4]float32
	for k := range r {
		r[k] = x[k] + float32((y[k]-x[k])*a)
	}
	return r
}

// shade is a fragment's colour, step 1 of fragment, given the interpolated
// inputs:
//
//  1. c = diffuse, times texel(uq/q, vq/q) when textured (D3DTOP_MODULATE on
//     colour and alpha).
//  2. Fog factor: vertex fog f = fq/q; table fog f = fogLinear(FogStart,
//     FogEnd, 1/q), Direct3D's w-based table fog; no fog f = 1. Then
//     c.rgb = f·c.rgb + (1 − f)·FogColor; alpha is untouched.
//  3. The result is premultiplied: (c.r·c.a, c.g·c.a, c.b·c.a, c.a).
//
// tex is ignored when the uniforms say untextured.
func shade(u uniforms, tex *image.NRGBA, diffuse [4]float32, uq, vq, q, fq float32) [4]float32 {
	c := diffuse
	if u.Textured != 0 {
		t := texel(tex, uq/q, vq/q, Address(u.Address), Filter(u.Filter))
		for k := range c {
			c[k] = float32(c[k] * t[k])
		}
	}
	f := float32(1)
	switch FogMode(u.FogMode) {
	case FogVertexLinear:
		f = fq / q
	case FogTableLinear:
		f = fogLinear(u.FogStart, u.FogEnd, 1/q)
	}
	for k := range 3 {
		c[k] = float32(f*c[k]) + float32((1-f)*u.FogColor[k])
	}
	return [4]float32{float32(c[0] * c[3]), float32(c[1] * c[3]), float32(c[2] * c[3]), c[3]}
}
