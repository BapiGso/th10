package std

import "th10/zun"

// The fog tween of op 9 (docs/research/m4_s2b.md, with its §7 corrections
// taking precedence): FUN_004049a0 is FUN_00404610's template (zun.Interp)
// over the 7-dword fog record instead of a vector.
//
// The wiring lives in std/vm.go:
//
//   - op 8 (0x403e7f-0x403ee4): FogColor = u32(a); FogBytes[k] =
//     float32(byte k of FogColor) for k = 0..3; FogNear = f32(a+4);
//     FogFar = f32(a+8).
//
//   - op 9 (0x403eef-0x403fbb): with dur = i32(a), mode = i32(a+4), the
//     colour bytes at a+8..a+11, near = f32(a+12), far = f32(a+16):
//
//     v.fog.Start = v.Cam.fogRec()
//     v.fog.End = fogRec{F: {near, far, float32(byte a+8), …, float32(byte a+11)},
//     C: u32(a+8)}     (FUN_00405040; its _ftol2 gives back the same bytes)
//     v.fog.Dur, v.fog.Mode = dur, mode
//     v.fog.T.Reset()
//
//     Tan0 and Tan1 are not touched (nothing writes them; they stay zero).
//
//   - tweens, after the position tween (0x40430f-0x404339): if
//     v.fog.Dur != 0 (negative too), v.Cam.setFogRec(v.fog.step(gs)).

// fogRec is the 7-dword fog record (STD+0x2b48, the camera block's
// +0xfc..+0x114): F = near, far, then the colour's bytes 0..3 as floats;
// C = the packed colour (ARGB).
type fogRec struct {
	F [6]float32
	C uint32
}

// fogRec is c's fog record: F = FogNear, FogFar, FogBytes[0..3]; C =
// FogColor.
func (c *Camera) fogRec() fogRec {
	return fogRec{
		F: [6]float32{c.FogNear, c.FogFar, c.FogBytes[0], c.FogBytes[1], c.FogBytes[2], c.FogBytes[3]},
		C: c.FogColor,
	}
}

// setFogRec stores r into c's fog fields, the inverse of fogRec (the copy
// to +0x2b48 at 0x40432c).
func (c *Camera) setFogRec(r fogRec) {
	c.FogNear, c.FogFar = r.F[0], r.F[1]
	copy(c.FogBytes[:], r.F[2:])
	c.FogColor = r.C
}

// pack is the colour the record operations store at +0x18: byte k is the
// low byte of _ftol2(f[k+2]) (truncation toward zero), k = 0..3, byte 0
// lowest. In Go: byte(int64(f[k+2])); a NaN or out-of-range value gives
// _ftol2's 0x8000000000000000, whose low byte is 0, as int64 does on amd64.
func pack(f [6]float32) uint32 {
	var c uint32
	for k := range 4 {
		c |= uint32(byte(int64(f[k+2]))) << (8 * k)
	}
	return c
}

// add7 is FUN_00404e10 with EAX = a, ECX = b: F[k] = a.F[k] + b.F[k], C =
// pack(F).
func add7(a, b fogRec) fogRec {
	var f [6]float32
	for k := range f {
		f[k] = a.F[k] + b.F[k]
	}
	return fogRec{F: f, C: pack(f)}
}

// sub7 is FUN_00404d40 with ECX = a, EAX = b: F[k] = a.F[k] − b.F[k], C =
// pack(F).
func sub7(a, b fogRec) fogRec {
	var f [6]float32
	for k := range f {
		f[k] = a.F[k] - b.F[k]
	}
	return fogRec{F: f, C: pack(f)}
}

// scale7 is FUN_00404da0 with s on the stack and EAX = a: F[k] = s·a.F[k],
// C = pack(F).
func scale7(s float32, a fogRec) fogRec {
	var f [6]float32
	for k := range f {
		f[k] = float32(s * a.F[k])
	}
	return fogRec{F: f, C: pack(f)}
}

// fogTween is the state block at STD+0xe8 that FUN_004049a0 steps.
type fogTween struct {
	Start, End, Tan0, Tan1 fogRec    // P +0xe8, Q +0x104, R1 +0x120, R2 +0x13c
	T                      zun.Timer // +0x158 (prev, cur, f; gs* +0x164, flags +0x168)
	Dur, Mode              int32     // +0x16c, +0x170
}

// step is FUN_004049a0: one frame at game speed gs, returning the new fog
// record. zun.Interp.Step's structure, record for vector:
//
//	if Dur > 0:
//	    T.Tick(gs)
//	    if Dur <= T.Cur:
//	        T.Set(Dur); Dur = 0
//	        return Start if Mode == 7, else End      (the whole record, C too)
//	switch Mode:
//	case 7:    Start = add7(End, Start); return Start
//	case 17:   s := add7(Tan1, Start); Tan1 = add7(End, Tan1); Start = s
//	           return Start
//	case 8:    h00, h01, h10, h11 = zun.HermiteWeights(T.F / float32(Dur))
//	           return add7(scale7(h11, Tan1),
//	                  add7(scale7(h10, Tan0),
//	                  add7(scale7(h01, End), scale7(h00, Start))))
//	default:   e := zun.Ease(Mode, T.F, float32(Dur))
//	           return add7(Start, scale7(e, sub7(End, Start)))
//
// Every operation rounds to float32. Mode 7 finishing on its first frame
// (Dur 1) returns Start as op 9 copied it, colour included.
func (p *fogTween) step(gs float32) fogRec {
	if p.Dur > 0 {
		p.T.Tick(gs)
		if p.Dur <= p.T.Cur {
			p.T.Set(p.Dur)
			p.Dur = 0
			if p.Mode == 7 {
				return p.Start
			}
			return p.End
		}
	}
	switch p.Mode {
	case 7:
		p.Start = add7(p.End, p.Start)
		return p.Start
	case 17:
		s := add7(p.Tan1, p.Start)
		p.Tan1 = add7(p.End, p.Tan1)
		p.Start = s
		return p.Start
	case 8:
		h00, h01, h10, h11 := zun.HermiteWeights(p.T.F / float32(p.Dur))
		return add7(scale7(h11, p.Tan1),
			add7(scale7(h10, p.Tan0),
				add7(scale7(h01, p.End), scale7(h00, p.Start))))
	}
	e := zun.Ease(p.Mode, p.T.F, float32(p.Dur))
	return add7(p.Start, scale7(e, sub7(p.End, p.Start)))
}
