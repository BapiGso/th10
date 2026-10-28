package ecl

// The movement half of the enemy update FUN_0040dc80, in the binary's own
// arithmetic. th10.exe creates its D3D9 device without FPU_PRESERVE, so the
// x87 runs at 24-bit precision: every add, subtract, multiply and divide
// rounds like float32, while fsin/fcos stay extended. Products are wrapped in
// float32() so the compiler cannot fuse them into an FMA.
//
// An enemy's position is the sum of two layers (EnemyData+0x58 "A" and +0x84
// "B"), each a posBlock that either moves by polar(angle, speed) per frame or
// circles its centre. Timed changes run through interp tweens: one for the
// position (281/283), one for angle/speed (285/287/289/291/292/293) and one
// for radius/radius-speed (289/291) per layer.

import (
	"math"

	"th10/zun"
)

// float32 constants, bit-identical to th10.exe .rdata (see motion_test.go).
const (
	pi32      float32 = math.Pi     // DAT_00470b18
	twoPi32   float32 = 2 * math.Pi // DAT_00470b14
	halfPi32  float32 = math.Pi / 2 // DAT_00470b94
	thirdPi32 float32 = math.Pi / 3 // DAT_00470d50
	third32   float32 = 1.0 / 3     // DAT_00470cc4

	// unsetArg is DAT_00470d58 (a double): a float arg at or below it means
	// "keep the current value".
	unsetArg = -999999.0
)

// normAngle is FUN_0044bc70 (zun.NormAngle).
func normAngle(a float32) float32 { return zun.NormAngle(a) }

// polar is FUN_0044c5d0 (zun.Polar).
func polar(a, r float32) (float32, float32) { return zun.Polar(a, r) }

// round01 snaps a coordinate down to the 0.01 grid: floor(v*100) * 0.01
// (FUN_0044c2a0 with the CRT floor FUN_00452ff0).
func round01(v float32) float32 {
	return float32(float32(math.Floor(float64(float32(v*100)))) * 0.01)
}

// ease is FUN_0044c350 (zun.Ease).
func ease(mode int32, num, den float32) float32 { return zun.Ease(mode, num, den) }

// interp is the shared tween zun.Interp (FUN_00412ac0 / FUN_00404610).
type interp = zun.Interp

// posBlock is one 11-dword position record: a movement layer, or the final
// position and its previous-frame copy.
type posBlock struct {
	pos   [3]float32
	delta [3]float32 // per-frame step; the centre while circling
	speed float32    // linear speed, or angular speed while circling
	angle float32
	// radius and radVel only matter while circling.
	radius, radVel float32
	circle         bool // flags bit 0 (288-291)
}

// advance is FUN_0044c2a0: step (or place on the circle), then snap x and y
// to the 0.01 grid.
func (b *posBlock) advance() {
	if !b.circle {
		for i := range b.pos {
			b.pos[i] += b.delta[i]
		}
	} else {
		c, s := polar(b.angle, b.radius)
		b.pos[0] = c + b.delta[0]
		b.pos[1] = s + b.delta[1]
		b.pos[2] = b.delta[2]
	}
	b.pos[0] = round01(b.pos[0])
	b.pos[1] = round01(b.pos[1])
}

// motion is an enemy's movement state (EnemyData from +0x0).
type motion struct {
	prev, final posBlock
	layer       [2]posBlock // A, B
	posI        [2]interp   // 281/283: position (3 components)
	velI        [2]interp   // angle, speed
	circI       [2]interp   // radius, radius speed
	// limit is ins_324's box (x, y, w, h), enforced while flag 0x200 is set.
	limit [4]float32
	// size is the extent of the slot-0 sprite (+0x13a4/+0x13a8), used by the
	// offscreen test. Zero until an ANM op sets slot 0.
	size [2]float32
}

// setFinal recomputes the final position as B + A at once, as 280/282 do.
func (m *motion) setFinal() {
	for i := range m.final.pos {
		m.final.pos[i] = m.layer[1].pos[i] + m.layer[0].pos[i]
	}
}

// Field bounds of the offscreen test (DAT_00470b40/3c/04/38).
const (
	fieldMinX float32 = -192
	fieldMaxX float32 = 192
	fieldMinY float32 = 0
	fieldMaxY float32 = 448
)

// step is the movement part of FUN_0040dc80: tweens, the per-layer step,
// rounding, the B+A sum, the ins_324 clamp and the offscreen test. flags is
// the enemy's ECL flag word; it returns false when an enemy that had been
// inside the field (0x100) has left it and 0x4 does not keep it.
func (m *motion) step(flags *int, speed float32) bool {
	m.prev = m.final
	for i := range m.layer {
		l := &m.layer[i]
		if m.velI[i].Dur != 0 {
			o := m.velI[i].Step(speed, 2)
			l.angle, l.speed = normAngle(o[0]), o[1]
		}
		if m.circI[i].Dur != 0 {
			o := m.circI[i].Step(speed, 2)
			l.radius, l.radVel = o[0], o[1]
		}
	}
	for i := range m.layer {
		l := &m.layer[i]
		switch {
		case m.posI[i].Dur != 0:
			o := m.posI[i].Step(speed, 3)
			for k := range o {
				l.delta[k] = o[k] - l.pos[k]
			}
		case l.circle:
			l.radius = l.radVel + l.radius
			l.angle = normAngle(l.speed + l.angle)
		default:
			l.delta[0], l.delta[1] = polar(l.angle, l.speed)
			l.delta[2] = 0
		}
	}
	a, b, f := &m.layer[0], &m.layer[1], &m.final
	a.advance()
	// Flag 0x40000 (enemies made by 270/271) also adds DAT_00491e6c..74 to B
	// here: a stage 3D offset not traced yet, taken as 0.
	b.advance()
	for k := range f.delta {
		f.delta[k] = (b.pos[k] + a.pos[k]) - f.pos[k]
	}
	f.advance()

	fl := *flags
	if fl&0x200 != 0 {
		clampAxis(&f.pos[0], m.limit[0], m.limit[2])
		clampAxis(&f.pos[1], m.limit[1], m.limit[3])
		for k := range a.pos {
			a.pos[k] = f.pos[k] - b.pos[k]
		}
	}
	hw, hh := float32(m.size[0]*0.5), float32(m.size[1]*0.5)
	x, y := f.pos[0], f.pos[1]
	if hw+x < fieldMinX || fieldMaxX < x-hw || hh+y < fieldMinY || fieldMaxY < y-hh {
		if fl&0x100 != 0 && fl&0x4 == 0 {
			return false
		}
	} else {
		*flags = fl | 0x100
	}
	return true
}

// clampAxis keeps *v within centre ± size/2, testing the low edge first.
func clampAxis(v *float32, centre, size float32) {
	if lo := centre - float32(size*0.5); lo <= *v {
		if hi := float32(size*0.5) + centre; hi < *v {
			*v = hi
		}
	} else {
		*v = lo
	}
}
