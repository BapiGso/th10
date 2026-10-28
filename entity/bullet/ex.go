package bullet

import "th10/zun"

// runEx is FUN_00406d90: run the transform chain from ExIdx until an empty
// entry, or a synchronous one while a handler is still active. Entries that
// stay active set their bit in ExFlags; the rest act at once.
func (b *Bullet) runEx(p *Pool, env *Env) {
	for b.ExIdx >= 0 && b.ExIdx < ExCount {
		e := &b.Ex[b.ExIdx]
		if e.Type == 0 {
			return
		}
		// The binary compares the int as a float: only +0 and -0 bits are zero.
		if uint32(e.Async)&0x7fffffff == 0 && b.ExFlags != 0 {
			return
		}
		switch e.Type {
		case ExNop:
		case ExSpawnBoost:
			b.ExFlags |= ExSpawnBoost
			b.boostT.Set(0)
		case ExAccel:
			b.ExFlags |= ExAccel
			b.accel = e.A
			b.accelAng = b.angleRule(e.B, env)
			b.accelT.Set(0)
			b.accelDur = e.C
			b.accelVec[0], b.accelVec[1] = zun.Polar(b.accelAng, b.accel)
			if b.ExIdx != 0 {
				env.se(b.TransformSound, 0, false)
			}
		case ExAngular:
			b.ExFlags |= ExAngular
			b.angA, b.angB = e.A, e.B
			b.angT.Set(0)
			b.angDur = e.C
			if b.ExIdx != 0 {
				env.se(b.TransformSound, 0, false)
			}
		case ExStepRel, ExStepAim, ExStepAbs:
			b.ExFlags |= e.Type
			b.stepAngle = b.angleRule(e.A, env)
			if e.B <= -999 {
				b.stepSpeed = b.Speed
			} else {
				b.stepSpeed = e.B
			}
			b.stepT.Reset()
			b.stepDur, b.stepMax, b.stepN = e.C, e.D, 0
		case ExBounce, ExBounceTop, ExBounceSides:
			b.ExFlags |= e.Type
			b.bounceSpeed = e.A
			b.bounce, b.bounceN = e.C, 0
		case ExImmune:
			b.Immune = e.C
		case ExGrace:
			b.Grace = e.C
		case ExScript:
			b.startScript(scriptBase[typeIndex(int16(e.C))]+e.D, env)
		case ExWait:
			b.ExFlags |= ExWait
			b.waitT.Set(e.C)
		case ExCancel:
			b.Cancel(env)
		case ExSound:
			env.se(e.C, b.Pos[0], true)
		case ExWrapX:
			b.ExFlags |= ExWrapX
			b.wrapXT.Set(e.C)
		case ExWrapY:
			b.ExFlags |= ExWrapY
			b.wrapYT.Set(e.C)
		case ExVolley:
			b.volley(p, e, env)
			if uint32(e.C)&0x80000000 == 0 {
				continue
			}
			b.Cancel(env)
		case ExValue:
			b.Value = e.C
		case ExGoto:
			b.ExIdx = e.C
			continue
		case ExHoming:
			b.ExFlags |= ExHoming
			b.homeA, b.homeB = e.A, e.B
			b.homeT.Reset()
			b.homeDur = e.C
		}
		b.ExIdx++
	}
}

// angleRule is the angle argument of the accel and step entries: below -990
// keeps the heading, 990 and above aims at the player.
func (b *Bullet) angleRule(x float32, env *Env) float32 {
	switch {
	case x <= -990:
		return b.Angle
	case x >= 990:
		return zun.Aim(b.Pos[0], b.Pos[1], env.Player[0], env.Player[1])
	}
	return x
}

// volley is the ExVolley entry: a volley from the bullet, its type, colour,
// first entry and count per ring packed in this entry and its angles, ring
// count and flags in the next. It leaves ExIdx past both entries.
func (b *Bullet) volley(p *Pool, e *Ex, env *Env) {
	t := Template{
		Pos:            b.Pos,
		Colour:         int16(int8(e.C >> 8)),
		StartEx:        e.C & 0xff,
		Count1:         int16(e.D),
		Speed1:         e.A,
		Speed2:         e.B,
		TransformSound: -1,
	}
	b.ExIdx++
	var next Ex
	if b.ExIdx < ExCount {
		next = b.Ex[b.ExIdx]
	}
	t.Angle1, t.Angle2 = next.A, next.B
	t.Count2 = int16(next.C)
	t.Flags = uint32(next.D)
	t.Ex = b.Ex
	p.Volley(&t, env)
	b.ExIdx++
}
