// Package zun holds the arithmetic primitives th10.exe shares between its
// subsystems: the angle helpers, the frame timer and the RNG.
//
// th10.exe creates its D3D9 device without FPU_PRESERVE, so the x87 runs at
// 24-bit precision: every add, subtract, multiply and divide rounds like
// float32, while fsin/fcos/fpatan stay extended. Products are wrapped in
// float32() so the compiler cannot fuse them into an FMA.
package zun

import "math"

// float32 constants, bit-identical to th10.exe .rdata (see zun_test.go).
const (
	Pi     float32 = math.Pi     // DAT_00470b18
	TwoPi  float32 = 2 * math.Pi // DAT_00470b14
	HalfPi float32 = math.Pi / 2 // DAT_00470b94
)

// NormAngle is FUN_0044bc70: fold into [-pi, pi] by whole turns, at most 34
// steps in all (the counter is shared by both loops).
func NormAngle(a float32) float32 {
	i := 0
	for a > Pi {
		a -= TwoPi
		ok := i < 0x21
		i++
		if !ok {
			break
		}
	}
	for a < -Pi {
		a += TwoPi
		ok := i < 0x21
		i++
		if !ok {
			break
		}
	}
	return a
}

// AddAngle is FUN_0044bc10: NormAngle(a + b).
func AddAngle(a, b float32) float32 { return NormAngle(a + b) }

// Polar is FUN_0044c5d0 / FUN_00408750: (cos a * r, sin a * r), the cosine
// and sine extended, the products rounded.
func Polar(a, r float32) (float32, float32) {
	s, c := math.Sincos(float64(a))
	return float32(c * float64(r)), float32(s * float64(r))
}

// Ellipse is FUN_00413270: (cos a * w, sin a * h), fsincos extended, the
// products rounded.
func Ellipse(a, w, h float32) (float32, float32) {
	s, c := math.Sincos(float64(a))
	return float32(c * float64(w)), float32(s * float64(h))
}

// Aim is the fpatan of (dy, dx) from (x, y) to (tx, ty), pi/2 when the points
// coincide (FUN_00426660 and the emitter's aim in FUN_004073e0).
func Aim(x, y, tx, ty float32) float32 {
	dx, dy := tx-x, ty-y
	if dx == 0 && dy == 0 {
		return HalfPi
	}
	return float32(math.Atan2(float64(dy), float64(dx)))
}

// AngleDiff is FUN_00408660: a - b taken the short way round.
func AngleDiff(a, b float32) float32 {
	if a-b > Pi {
		return a - (b + TwoPi)
	}
	if b-a > Pi {
		return a - (b - TwoPi)
	}
	return a - b
}

// Ease is FUN_0044c350: the tween curve for mode at num/den.
func Ease(mode int32, num, den float32) float32 {
	t := num / den
	switch mode {
	case 1:
		return float32(t * t)
	case 2:
		return float32(float32(t*t) * t)
	case 3:
		return float32(float32(float32(t*t)*t) * t)
	case 4:
		return 1 - float32((1-t)*(1-t))
	case 5:
		u := 1 - t
		return 1 - float32(float32(u*u)*u)
	case 6:
		u := 1 - t
		return 1 - float32(float32(float32(u*u)*u)*u)
	case 9:
		u := t + t
		if 1 <= u {
			return float32((2 - float32((2-u)*(2-u))) * 0.5)
		}
		return float32(float32(u*u) * 0.5)
	case 10:
		u := t + t
		if u < 1 {
			return float32(float32(float32(u*u)*u) * 0.5)
		}
		v := 2 - u
		return float32((2 - float32(float32(v*v)*v)) * 0.5)
	case 11:
		u := t + t
		if u < 1 {
			return float32(float32(float32(float32(u*u)*u)*u) * 0.5)
		}
		v := 2 - u
		return float32((2 - float32(float32(float32(v*v)*v)*v)) * 0.5)
	case 12:
		u := t + t
		if 1 <= u {
			return float32(float32((u-1)*(u-1))*0.5) + 0.5
		}
		return 0.5 - float32(float32((1-u)*(1-u))*0.5)
	case 13:
		u := t + t
		if 1 <= u {
			w := u - 1
			return float32(float32(float32(w*w)*w)*0.5) + 0.5
		}
		w := 1 - u
		return 0.5 - float32(float32(float32(w*w)*w)*0.5)
	case 14:
		u := t + t
		if 1 <= u {
			w := u - 1
			return float32(float32(float32(float32(w*w)*w)*w)*0.5) + 0.5
		}
		w := 1 - u
		return 0.5 - float32(float32(float32(float32(w*w)*w)*w)*0.5)
	case 15:
		return 0
	case 16:
		return 1
	}
	return t
}
