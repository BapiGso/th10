package bullet

import (
	"math"

	"th10/anm"
	"th10/zun"
)

// Calc is the bullet calc (FUN_00406770 / FUN_004065c0): update every live
// slot in slot order and rebuild the draw lists from the survivors. The
// whole calc is skipped while Env.Hold has bit 0 or 2 set, and the updates
// alone (the bullets still drawn) while it has bits 1 and 10.
//
// The draw prep of FUN_004066e0 (the sprite position in screen space and
// the heading of auto-rotating sprites) is done here as each bullet joins its
// list: it touches only that bullet, the draw follows every calc that ran,
// and nothing writes a bullet in between.
func (p *Pool) Calc(env *Env) {
	hold := env.hold()
	if hold&5 != 0 {
		return
	}
	paused := hold&2 != 0 && hold&0x400 != 0
	for i := range p.layers {
		p.layers[i] = p.layers[i][:0]
	}
	p.Count = 0
	gs := env.gs()
	for i := range p.Slots {
		b := &p.Slots[i]
		if b.State == StateFree {
			continue
		}
		if !paused && b.update(p, env) {
			continue
		}
		if l := b.Layer; l >= 0 && l < Layers {
			p.layers[l] = append(p.layers[l], b)
		}
		p.Count++
		b.T.Tick(gs)
		b.prep()
	}
}

// prep is the per-bullet part of FUN_004066e0.
func (b *Bullet) prep() {
	b.VM.Pos = [3]float32{b.Pos[0] + 224, b.Pos[1] + 16, b.Pos[2]}
	if b.VM.Flags&anm.FlagAutoRotate != 0 {
		b.VM.Rot[2] = zun.AddAngle(b.Angle, zun.HalfPi)
		b.VM.Flags |= 4
	}
}

// update is FUN_00406240. It reports whether the slot was freed.
func (b *Bullet) update(p *Pool, env *Env) bool {
	if b.Flags&FlagDelete != 0 {
		b.free()
		return true
	}
	gs := env.gs()
	switch b.State {
	case StateSpawning:
		b.halfMove(gs)
		if b.VM.I[0] == 0 {
			break
		}
		b.State = StateActive
		fallthrough
	case StateActive:
		b.active(p, env, gs)
	case StateCancelling:
		b.halfMove(gs)
	}

	if w, h, ok := b.sprite(); ok {
		if b.ExFlags&ExWrapX != 0 {
			b.wrapX(w, h, env, gs)
		}
		if b.ExFlags&ExWrapY != 0 {
			b.wrapY(w, h, env, gs)
		}
		if b.Grace < 1 && outside406160(b.Pos, w, h) {
			b.free()
			return true
		}
	}
	if b.Immune != 0 {
		b.Immune--
	}
	if b.Grace > 0 {
		b.Grace--
	}
	if b.VM.Run(env.Anm) {
		b.free()
		return true
	}
	return false
}

// active is state 1: the transforms, the move and the hit test.
func (b *Bullet) active(p *Pool, env *Env, gs float32) {
	b.runEx(p, env)
	if f := b.ExFlags; f != 0 {
		if f&ExSpawnBoost != 0 {
			b.boost(gs)
		}
		if b.ExFlags&ExAccel != 0 {
			b.accelerate(gs)
		}
		if b.ExFlags&ExAngular != 0 {
			b.angular(gs)
		}
		if b.ExFlags&ExStepRel != 0 {
			b.step(ExStepRel, env, gs)
		}
		if b.ExFlags&ExStepAbs != 0 {
			b.step(ExStepAbs, env, gs)
		}
		if b.ExFlags&ExStepAim != 0 {
			b.step(ExStepAim, env, gs)
		}
		if b.ExFlags&(ExBounce|ExBounceTop|ExBounceSides) != 0 {
			b.bounceWalls(env)
		}
		if b.ExFlags&ExHoming != 0 {
			b.homing(env, gs)
		}
		if b.ExFlags&ExWait != 0 {
			if b.waitT.Cur < 1 {
				b.ExFlags ^= ExWait
			} else {
				b.waitT.Add(-1, gs)
			}
		}
	}
	for k := range 3 {
		b.Pos[k] = float32(gs*b.Vel[k]) + b.Pos[k]
	}
	if b.Flags&FlagHittable == 0 {
		return
	}
	switch b.hitTest(env) {
	case 1:
		b.State = StateCancelling
		b.VM.Interrupt(1)
		if b.CancelScript >= 0 {
			env.effect(b.CancelScript, b.Pos)
		}
	case 2:
		if b.Flags&FlagGrazed == 0 {
			b.Flags |= FlagGrazed
			env.effect(effectGraze, b.Pos)
			env.se(seGraze, b.Pos[0], true)
			if env.Graze != nil {
				env.Graze(b)
			}
		}
	}
}

// halfMove is the half-speed move of states 2 and 3.
func (b *Bullet) halfMove(gs float32) {
	for k := range 3 {
		b.Pos[k] = float32(float32(gs*b.Vel[k])*0.5) + b.Pos[k]
	}
}

// hitTest is FUN_004266b0: 1 for a hit that counts, 2 for a graze (the box
// grown by 24 on each side), 0 otherwise.
func (b *Bullet) hitTest(env *Env) int {
	x, y := b.Pos[0], b.Pos[1]
	hw, hh := float32(b.W*0.5), float32(b.H*0.5)
	l, t, r, bot := env.Box[0], env.Box[1], env.Box[2], env.Box[3]
	if hw+x < l || hh+y < t || r < x-hw || bot < y-hh {
		if l <= x+24 && t <= y+24 && x-24 <= r && y-24 <= bot {
			return 2
		}
		return 0
	}
	if env.Hit != nil && env.Hit() {
		return 1
	}
	return 0
}

// boost is FUN_004074b0: the extra speed 5 - t*0.3125 for 17 frames.
func (b *Bullet) boost(gs float32) {
	if b.boostT.Cur < 17 {
		s := float32(5-float32(b.boostT.F*0.3125)) + b.Speed
		b.Vel[0], b.Vel[1] = zun.Polar(b.Angle, s)
	} else {
		b.ExFlags ^= ExSpawnBoost
	}
	b.boostT.Tick(gs)
}

// accelerate is FUN_00407560.
func (b *Bullet) accelerate(gs float32) {
	if b.accelT.Cur < b.accelDur {
		b.Speed = float32(gs*b.accel) + b.Speed
		for k := range 3 {
			b.Vel[k] = float32(gs*b.accelVec[k]) + b.Vel[k]
		}
		vx, vy := b.Vel[0], b.Vel[1]
		if abs32(vx) > 0.0001 || abs32(vy) > 0.0001 {
			b.Angle = float32(math.Atan2(float64(vy), float64(vx)))
		}
	} else {
		b.ExFlags &^= ExAccel
	}
	b.accelT.Tick(gs)
}

// angular is FUN_004076a0.
func (b *Bullet) angular(gs float32) {
	if b.angT.Cur < b.angDur {
		b.Angle = zun.AddAngle(b.Angle, float32(gs*b.angB))
		b.Speed = float32(gs*b.angA) + b.Speed
		b.Vel[0], b.Vel[1] = zun.Polar(b.Angle, b.Speed)
	} else {
		b.ExFlags &^= ExAngular
	}
	b.angT.Tick(gs)
}

// step is FUN_00407780 (ExStepRel), FUN_004078e0 (ExStepAbs) and
// FUN_00407a30 (ExStepAim): slow linearly to a stop over stepDur frames, then
// turn, take stepSpeed and start over, stepMax times in all.
func (b *Bullet) step(kind uint32, env *Env, gs float32) {
	var s float32
	if b.stepT.Cur < b.stepDur {
		s = b.Speed - float32(float32(b.Speed*b.stepT.F)/float32(b.stepDur))
	} else {
		env.se(b.TransformSound, 0, false)
		b.stepN++
		if b.stepMax <= b.stepN {
			b.ExFlags &^= kind
		}
		switch kind {
		case ExStepRel:
			b.Angle = b.stepAngle + b.Angle
		case ExStepAbs:
			b.Angle = b.stepAngle
		case ExStepAim:
			aim := zun.Aim(b.Pos[0], b.Pos[1], env.Player[0], env.Player[1])
			b.Angle = zun.AddAngle(aim, b.stepAngle)
		}
		s = b.stepSpeed
		b.Speed = s
		b.stepT.Reset()
	}
	b.Vel[0], b.Vel[1] = zun.Polar(b.Angle, s)
	b.stepT.Tick(gs)
}

// bounceWalls is FUN_00407be0.
func (b *Bullet) bounceWalls(env *Env) {
	x, y := b.Pos[0], b.Pos[1]
	if x > -192 && x < 192 && y > 0 && y < 448 {
		return
	}
	env.se(b.TransformSound, 0, false)
	bounced := false
	if x < -192 || x >= 192 {
		b.Angle = zun.AddAngle(-b.Angle-zun.Pi, 0)
		bounced = true
		if x >= -192 {
			b.Pos[0] = 384 - x
		} else {
			b.Pos[0] = -384 - x
		}
	}
	if b.ExFlags&ExBounceSides == 0 && (y < 0 || (y >= 448 && b.ExFlags&ExBounce != 0)) {
		bounced = true
		b.Angle = -b.Angle
		if y >= 0 {
			b.Pos[1] = (448 - y) + 448
		} else {
			b.Pos[1] = -y
		}
	}
	if b.bounceSpeed > -990 {
		b.Speed = b.bounceSpeed
	}
	b.Vel[0], b.Vel[1] = zun.Polar(b.Angle, b.Speed)
	if bounced {
		b.bounceN++
	}
	if b.bounce <= b.bounceN {
		b.ExFlags &= 0xf7fff3ff
	}
}

// wrapX is FUN_00407da0: once fully off a side, come back in at the other.
func (b *Bullet) wrapX(w, h float32, env *Env, gs float32) {
	if !outside4061d0(b.Pos, w, h) {
		return
	}
	if x := b.Pos[0]; x < -192 || x > 192 {
		if x < -192 {
			b.Pos[0] = (w + x) + 384
		} else {
			b.Pos[0] = x - (w + 384)
		}
		b.wrapXT.Add(-1, gs)
		env.se(b.TransformSound, 0, false)
	}
	if b.wrapXT.Cur < 1 {
		b.ExFlags ^= ExWrapX
	}
}

// wrapY is FUN_00407e40, wrapX for the top and bottom.
func (b *Bullet) wrapY(w, h float32, env *Env, gs float32) {
	if !outside4061d0(b.Pos, w, h) {
		return
	}
	if y := b.Pos[1]; y < 0 || y > 448 {
		if y < 0 {
			b.Pos[1] = (h + y) + 448
		} else {
			b.Pos[1] = y - (h + 448)
		}
		b.wrapYT.Add(-1, gs)
		env.se(b.TransformSound, 0, false)
	}
	if b.wrapYT.Cur < 1 {
		b.ExFlags ^= ExWrapY
	}
}

// homing is FUN_00407ef0: turn toward the player plus homeB by homeA of the
// difference each frame.
func (b *Bullet) homing(env *Env, gs float32) {
	if b.homeT.Cur < b.homeDur {
		aim := zun.Aim(b.Pos[0], b.Pos[1], env.Player[0], env.Player[1])
		d := zun.AngleDiff(zun.AddAngle(b.homeB, aim), b.Angle)
		b.Angle = zun.AddAngle(b.Angle, float32(float32(d*b.homeA)*gs))
		b.Vel[0], b.Vel[1] = zun.Polar(b.Angle, b.Speed)
	} else {
		b.ExFlags &^= ExHoming
	}
	b.homeT.Tick(gs)
}

// outside4061d0 is FUN_004061d0: whether a w*h box at pos has left the
// playfield entirely.
func outside4061d0(pos [3]float32, w, h float32) bool {
	hw, hh := float32(w*0.5), float32(h*0.5)
	return !(hw+pos[0] > -192 && pos[0]-hw < 192 && hh+pos[1] > 0 && pos[1]-hh < 448)
}

// outside406160 is FUN_00406160, the offscreen deletion test: as
// outside4061d0 with 64 more pixels above the top.
func outside406160(pos [3]float32, w, h float32) bool {
	hw, hh := float32(w*0.5), float32(h*0.5)
	return !(hw+pos[0] > -192 && pos[0]-hw < 192 && hh+pos[1] > -64 && pos[1]-hh < 448)
}

// outside4086b0 is FUN_004086b0: whether pos is off the playfield by more
// than the margins mx, my.
func outside4086b0(pos [3]float32, mx, my float32) bool {
	return !(pos[0]+mx > -192 && pos[0]-mx < 192 && pos[1]+my > 0 && pos[1]-my < 448)
}

func abs32(v float32) float32 {
	return math.Float32frombits(math.Float32bits(v) &^ (1 << 31))
}
