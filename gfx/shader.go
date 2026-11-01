package gfx

import (
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// The fragment stage. One Kage shader (shaderSrc, //kage:unit pixels) does
// all of it; shade below is the same computation on the CPU, the reference
// the shader is tested against.
//
// Inputs per fragment, from ebitenVertex: color = Diffuse as straight-alpha
// 0..1; custom = (U·Q, V·Q, Q, Fog·Q), interpolated screen-linearly by Ebiten,
// so dividing by custom.z is perspective-correct; src0Pos.x = Z, the
// screen-linear depth; dstPos.xy = the back-buffer pixel centre (Flush).
//
// Uniforms (names are the Kage variable names; see uniforms):
//
//	Textured   int    0: no texture, the fragment is the diffuse colour
//	Address    int    Address of the bound texture
//	Filter     int    Filter of the bound texture
//	FogMode    int    FogMode
//	FogColor   vec3   straight 0..1, float32(channel)/255
//	FogStart   float  table fog only
//	FogEnd     float  table fog only
//	AlphaRef   int    alpha test reference, 0 = off
//	DepthTest  int    1: LESSEQUAL against Images[1]; 0: no depth test
//	Pass       int    0: colour pass; 1: depth pass (Flush)
//
// The texture is Images[0]. The shader addresses texels itself from
// imageSrc0Origin() and imageSrc0Size() and reads them with
// imageSrc0UnsafeAt at texel centres, so Ebiten's own filtering never comes
// into play. Texels are straight alpha (see Texture). The depth to test
// against is Images[1], read with imageSrc1UnsafeAt(dstPos.xy) and decoded
// as depthColor says.
//
// fragment below is the whole stage; the shader is its transcription.

// uniforms are the shader's uniform values for a batch's state.
type uniforms struct {
	Textured         int
	Address, Filter  int
	FogMode          int
	FogColor         [3]float32
	FogStart, FogEnd float32
	AlphaRef         int
	DepthTest        int
	Pass             int
}

// uniformsOf derives the colour pass's uniforms from a batch's state: the
// fog from s.fogState(), AlphaRef = s.alphaRef, DepthTest = 1 when
// s.depthTest(), Pass 0. Flush sets Pass to 1 for the depth pass.
func uniformsOf(s state) uniforms {
	fog := s.fogState()
	u := uniforms{
		Address:  int(s.addr),
		Filter:   int(s.filter),
		FogMode:  int(fog.Mode),
		FogColor: [3]float32{float32(fog.Color>>16&0xff) / 255, float32(fog.Color>>8&0xff) / 255, float32(fog.Color&0xff) / 255},
		FogStart: fog.Start,
		FogEnd:   fog.End,
		AlphaRef: int(s.alphaRef),
	}
	if s.tex != nil {
		u.Textured = 1
	}
	if s.depthTest() {
		u.DepthTest = 1
	}
	return u
}

// alpha8 is a fragment alpha as the alpha test compares it with ALPHAREF:
// ⌊float32(a·255) + 0.5⌋ clamped to 0..255. Rounding to nearest rather than
// truncating is an assumption for L3 to confirm.
func alpha8(a float32) int {
	v := int(math.Floor(float64(float32(a*255) + 0.5)))
	return min(max(v, 0), 255)
}

// depth16 is a depth as the D16 buffer stores it: ⌊float32(z′·65535) + 0.5⌋
// with z′ = z clamped to [0, 1]. Rounding to nearest is an assumption for L3
// to confirm.
func depth16(z float32) uint16 {
	z = min(max(z, 0), 1)
	return uint16(math.Floor(float64(float32(z*65535) + 0.5)))
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

// shaderSrc is the Kage source of shade, alpha8, depth16, depthColor and
// fragment (G2, G3).
const shaderSrc = `//kage:unit pixels

package main

var Textured int
var Address int
var Filter int
var FogMode int
var FogColor vec3
var FogStart float
var FogEnd float
var AlphaRef int
var DepthTest int
var Pass int

// wrapOrClamp maps a texel index into [0, n) under Address (0 wrap, 1 clamp).
// Wrap is written out rather than mod, so negative indices behave as in shade.
func wrapOrClamp(i float, n float) float {
	if Address == 0 {
		return i - n*floor(i/n)
	}
	return clamp(i, 0, n-1)
}

// fetch reads texel (i, j) of the bound texture after addressing it. org and
// size are the texture's origin and size in atlas pixels.
func fetch(org vec2, size vec2, i float, j float) vec4 {
	p := vec2(wrapOrClamp(i, size.x), wrapOrClamp(j, size.y))
	return imageSrc0UnsafeAt(org + p + 0.5)
}

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	c := color
	q := custom.z
	if Textured == 1 {
		org := imageSrc0Origin()
		size := imageSrc0Size()
		s := custom.x / q * size.x
		t := custom.y / q * size.y
		var tex vec4
		if Filter == 0 {
			tex = fetch(org, size, floor(s), floor(t))
		} else {
			x := s - 0.5
			y := t - 0.5
			i := floor(x)
			j := floor(y)
			top := mix(fetch(org, size, i, j), fetch(org, size, i+1, j), x-i)
			bot := mix(fetch(org, size, i, j+1), fetch(org, size, i+1, j+1), x-i)
			tex = mix(top, bot, y-j)
		}
		c *= tex
	}
	f := float(1)
	if FogMode == 1 {
		f = custom.w / q
	} else if FogMode == 2 {
		d := 1.0 / q
		if FogStart == FogEnd {
			if d < FogEnd {
				f = 1
			} else {
				f = 0
			}
		} else {
			f = clamp((FogEnd-d)/(FogEnd-FogStart), 0, 1)
		}
	}
	c = vec4(f*c.rgb+(1-f)*FogColor, c.a)
	if clamp(floor(c.a*255.0+0.5), 0.0, 255.0) < float(AlphaRef) {
		discard()
	}
	z := floor(clamp(src0Pos.x, 0.0, 1.0)*65535.0 + 0.5)
	if DepthTest == 1 {
		s := imageSrc1UnsafeAt(dstPos.xy)
		stored := floor(s.r*255.0+0.5)*256.0 + floor(s.g*255.0+0.5)
		if z > stored {
			discard()
		}
	}
	if Pass == 1 {
		hi := floor(z / 256.0)
		return vec4(hi/255.0, (z-hi*256.0)/255.0, 0.0, 1.0)
	}
	return vec4(c.rgb*c.a, c.a)
}
`

var (
	shaderOnce sync.Once
	shaderProg *ebiten.Shader
	shaderErr  error
)

// compiledShader compiles shaderSrc on first use and returns it thereafter.
func compiledShader() *ebiten.Shader {
	shaderOnce.Do(func() {
		shaderProg, shaderErr = ebiten.NewShader([]byte(shaderSrc))
	})
	if shaderErr != nil {
		panic("gfx: shader: " + shaderErr.Error())
	}
	return shaderProg
}
