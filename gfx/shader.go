package gfx

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// The fragment stage. One Kage shader (shaderSrc, //kage:unit pixels) does
// all of it; shade in shader_ref_test.go is the same computation on the
// CPU, the reference the shader is tested against.
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
// fragment (shader_ref_test.go) is the whole stage; the shader is its
// transcription.

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

// depth16 is a depth as the D16 buffer stores it: ⌊float32(z′·65535) + 0.5⌋
// with z′ = z clamped to [0, 1]. Rounding to nearest is an assumption for L3
// to confirm.
func depth16(z float32) uint16 {
	z = min(max(z, 0), 1)
	return uint16(math.Floor(float64(float32(z*65535) + 0.5)))
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
