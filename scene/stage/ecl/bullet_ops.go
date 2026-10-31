package ecl

// Bullet and laser opcodes 400-436 (FUN_0040e770 cases 400-0x1b4). Each enemy
// owns 16 emitter templates (enemy +0x2c4, bullet.Template) and their fire
// offsets (+0x1344); the ops set template fields and 401 fires one through
// the bullet manager (FUN_004073e0, entity/bullet).
//
// Every handler resolves exactly the params the binary resolves, in its
// order: a var arg of -1 pops the data stack, so a tier op that reads only
// the chosen pair must not read the others.
//
// The laser ops 412-434 build a line or infinite laser template and fire it
// through the laser manager (FUN_0041c510, entity/laser), or look a laser up
// by its handle (FUN_0040cf90) and write one of its fields.

import (
	"math"

	"th10/entity/bullet"
	"th10/entity/laser"
	"th10/scene/stage"
)

// newTemplate is the template op 400 leaves.
func newTemplate() bullet.Template {
	return bullet.Template{Count1: 1, Count2: 1, Speed1: 2, Sound: 7, TransformSound: 0x18, Flags: 0x203}
}

// template returns template s, or nil past the 16 slots (which the binary
// does not check).
func (a *eclActor) template(s int32) *bullet.Template {
	if s < 0 || int(s) >= len(a.tpl) {
		return nil
	}
	return &a.tpl[s]
}

func (vm *VM) execBulletOp(ctx *stage.Context, t *eclTask, in *Instr) {
	if t.owner == nil || len(in.Params) < len(th10Format(in.Opcode)) {
		return
	}
	a := vm.actorFor(t.owner)
	p := in.Params
	ri := func(i int) int32 { return vm.resolveInt(t, p[i]) }
	rf := func(i int) float32 { return vm.resolveFloat(t, p[i]) }

	switch in.Opcode {
	case opETNew:
		s := ri(0)
		if tp := a.template(s); tp != nil {
			*tp = newTemplate()
			a.tplOff[s][0], a.tplOff[s][1] = 0, 0
		}
	case opETFire:
		vm.fire(ctx, a, ri(0))
	case opETSprite:
		s := ri(0)
		typ, col := ri(1), ri(2)
		if tp := a.template(s); tp != nil {
			tp.Type, tp.Colour = int16(typ), int16(col)
		}
	case opETOffset:
		s := ri(0)
		x, y := rf(1), rf(2)
		if a.template(s) != nil {
			a.tplOff[s][0], a.tplOff[s][1] = x, y
		}
	case opETAngle:
		s := ri(0)
		a1, a2 := rf(1), rf(2)
		if tp := a.template(s); tp != nil {
			tp.Angle1, tp.Angle2 = a1, a2
		}
	case opETSpeed:
		vm.setSpeeds(t, a, ri(0), p[1], p[2])
	case opETCount:
		vm.setCounts(t, a, ri(0), p[1], p[2])
	case opETAim:
		s := ri(0)
		aim := ri(1)
		if tp := a.template(s); tp != nil {
			tp.Aim = uint16(aim)
		}
	case opETSound:
		s := ri(0)
		snd, tsnd := ri(1), ri(2)
		if tp := a.template(s); tp != nil {
			tp.Sound, tp.TransformSound = snd, tsnd
		}
	case opETEx:
		s, idx := ri(0), ri(1)
		e := bullet.Ex{Async: ri(2), Type: uint32(ri(3)), C: ri(4), D: ri(5)}
		e.A, e.B = rf(6), rf(7)
		// The binary writes entry slot*22+idx: idx 18-21 land in the
		// template's tail and the next slot, which are not modelled.
		if tp := a.template(s); tp != nil && idx >= 0 && idx < bullet.ExCount {
			tp.Ex[idx] = e
		}
	case opETClearAll: // FUN_00408210(1), FUN_0041c850
		if ctx != nil && ctx.Bullets != nil {
			ctx.Bullets.RectClear(true, ctx.BulletEnv)
			ctx.Lasers.ClearAll(true, ctx.LaserEnv)
		}
	case opETCopy: // the source is resolved first
		src := ri(1)
		dst := ri(0)
		if s, d := a.template(src), a.template(dst); s != nil && d != nil {
			*d = *s
		}
	case opETCancel, opETClear: // FUN_00408100(r, items, 0), FUN_0041c800
		r := rf(0)
		if ctx != nil && ctx.Bullets != nil {
			pos := a.mv.final.pos
			ctx.Bullets.CircleClear(pos[0], pos[1], r, in.Opcode == opETCancel, false, ctx.BulletEnv)
			ctx.Lasers.CircleClear(pos, r, in.Opcode == opETCancel, ctx.LaserEnv)
		}
	case opETSpeedR3:
		s := ri(0)
		switch {
		case vm.rank > 511:
			vm.setSpeeds(t, a, s, p[5], p[6])
		case vm.rank >= -512:
			vm.setSpeeds(t, a, s, p[3], p[4])
		default:
			vm.setSpeeds(t, a, s, p[1], p[2])
		}
	case opETSpeedR5:
		s := ri(0)
		i := rankTier5(vm.rank)
		vm.setSpeeds(t, a, s, p[1+2*i], p[2+2*i])
	case opETSpeedR2:
		s := ri(0)
		lo1, lo2, hi1, hi2 := rf(1), rf(2), rf(3), rf(4)
		if tp := a.template(s); tp != nil {
			tp.Speed1 = speedBlend(lo1, hi1, vm.rank)
			tp.Speed2 = speedBlend(lo2, hi2, vm.rank)
		}
	case opETCountR3:
		s := ri(0)
		switch {
		case vm.rank > 511:
			vm.setCounts(t, a, s, p[5], p[6])
		case vm.rank >= -512:
			vm.setCounts(t, a, s, p[3], p[4])
		default:
			vm.setCounts(t, a, s, p[1], p[2])
		}
	case opETCountR5:
		s := ri(0)
		i := rankTier5(vm.rank)
		vm.setCounts(t, a, s, p[1+2*i], p[2+2*i])
	case opETCountR2:
		s := ri(0)
		lo1, lo2, hi1, hi2 := ri(1), ri(2), ri(3), ri(4)
		if tp := a.template(s); tp != nil {
			tp.Count1 = int16(rankBlend(hi1-lo1, vm.rank)) + int16(lo1)
			tp.Count2 = int16(rankBlend(hi2-lo2, vm.rank)) + int16(lo2)
		}
	case opETSpeedDiff:
		s := ri(0)
		d := diffColumn(vm.difficulty)
		vm.setSpeeds(t, a, s, p[1+d], p[5+d])
	case opETCountDiff:
		s := ri(0)
		d := diffColumn(vm.difficulty)
		vm.setCounts(t, a, s, p[1+d], p[5+d])
	case opAimAt: // the angle from (x, y) to the player into a float var
		x := rf(1)
		y := rf(2)
		dx, dy := vm.playerECL[0]-x, vm.playerECL[1]-y
		if p[0].IsVar {
			vm.storeVar(t, p[0], floatVal(float32(math.Atan2(float64(dy), float64(dx)))))
		}

	case opLaserOnA, opLaserOnB, opLaserOnAEx, opLaserOnBEx:
		vm.fireLaserLine(ctx, t, a, in)
	case opLaserStOn, opLaserStB, opLaserStOnEx, opLaserStBEx:
		vm.fireLaserInfinite(ctx, t, a, in)
	case opLaserPos, opLaserMove, opLaser416, opLaser417, opLaser418, opLaser419:
		vm.changeLaser(ctx, t, in)
	}
}

// fire is 401: place template s at the enemy plus the slot's offset, and fire
// it unless the player is within the fire radius op 346 set.
func (vm *VM) fire(ctx *stage.Context, a *eclActor, s int32) {
	tp := a.template(s)
	if tp == nil {
		return
	}
	for k := range tp.Pos {
		tp.Pos[k] = a.tplOff[s][k] + a.mv.final.pos[k]
	}
	if r := a.fireR; r != 0 {
		dy := tp.Pos[1] - vm.playerECL[1]
		dx := tp.Pos[0] - vm.playerECL[0]
		if r > float32(float32(dy*dy)+float32(dx*dx)) {
			return
		}
	}
	if ctx != nil && ctx.Bullets != nil {
		ctx.Bullets.Volley(tp, ctx.BulletEnv)
	}
}

// setSpeeds resolves a speed pair into template s.
func (vm *VM) setSpeeds(t *eclTask, a *eclActor, s int32, p1, p2 Param) {
	v1 := vm.resolveFloat(t, p1)
	v2 := vm.resolveFloat(t, p2)
	if tp := a.template(s); tp != nil {
		tp.Speed1, tp.Speed2 = v1, v2
	}
}

// setCounts resolves a count pair into template s.
func (vm *VM) setCounts(t *eclTask, a *eclActor, s int32, p1, p2 Param) {
	v1 := vm.resolveInt(t, p1)
	v2 := vm.resolveInt(t, p2)
	if tp := a.template(s); tp != nil {
		tp.Count1, tp.Count2 = int16(v1), int16(v2)
	}
}

// rankTier5 is the pair 423 and 426 pick by rank: 0 below -600 up to 4 above
// 599.
func rankTier5(rank int) int {
	switch {
	case rank > 599:
		return 4
	case rank > 199:
		return 3
	case rank > -201:
		return 2
	case rank >= -600:
		return 1
	}
	return 0
}

// diffColumn is the column 435 and 436 pick: Easy, Normal, Hard, else the
// fourth.
func diffColumn(d int) int {
	if d >= 0 && d <= 2 {
		return d
	}
	return 3
}

// speedBlend is 424's (hi - lo) * (rank + 1024) / 2048 + lo, in float32
// steps (0x470d4c, 0x470d48).
func speedBlend(lo, hi float32, rank int) float32 {
	return float32(float32(float32(hi-lo)*(float32(rank)+1024))*0.00048828125) + lo
}

// rankBlend is 427's integer form of the same blend: the product shifted
// right by 11, rounded toward zero.
func rankBlend(delta int32, rank int) int32 {
	v := delta * int32(rank+1024)
	if v < 0 {
		v += 0x7ff
	}
	return v >> 11
}

// laserPos is where the enemy's lasers start: the enemy plus slot 0's fire
// offset.
func (a *eclActor) laserPos() [3]float32 {
	var p [3]float32
	for k := range p {
		p[k] = a.tplOff[0][k] + a.mv.final.pos[k]
	}
	return p
}

// fireLaserLine is 412, 428, 431 and 433 (FUN_0041c510(0)): a line laser
// from the enemy plus slot 0's offset, drawn additively by 412 and 431. 431
// and 433 take slot 0's transform entries.
func (vm *VM) fireLaserLine(ctx *stage.Context, t *eclTask, a *eclActor, in *Instr) {
	p := in.Params
	var tpl laser.LineTemplate
	if in.Opcode == opLaserOnAEx || in.Opcode == opLaserOnBEx {
		tpl.Ex = a.tpl[0].Ex
	}
	tpl.Pos = a.laserPos()
	tpl.Type = int16(vm.resolveInt(t, p[0]))
	tpl.Colour = int16(vm.resolveInt(t, p[1]))
	tpl.Angle = vm.resolveFloat(t, p[2])
	tpl.Speed = vm.resolveFloat(t, p[3])
	tpl.Init = vm.resolveFloat(t, p[4])
	tpl.Full = vm.resolveFloat(t, p[5])
	tpl.MaxDist = vm.resolveFloat(t, p[6])
	tpl.Width = vm.resolveFloat(t, p[7])
	if in.Opcode == opLaserOnA || in.Opcode == opLaserOnAEx {
		tpl.Flags |= 1
	}
	if ctx != nil && ctx.Lasers != nil {
		ctx.Lasers.FireLine(&tpl, ctx.LaserEnv)
	}
}

// fireLaserInfinite is 413, 429, 432 and 434 (FUN_0041c510(1)): an infinite
// laser from the enemy plus slot 0's offset, its handle stored into the var
// param 0 names. 413 and 432 draw it additively; 432 and 434 take slot 0's
// transform entries.
func (vm *VM) fireLaserInfinite(ctx *stage.Context, t *eclTask, a *eclActor, in *Instr) {
	p := in.Params
	tpl := laser.InfiniteTemplate{Speed: 8}
	if in.Opcode == opLaserStOnEx || in.Opcode == opLaserStBEx {
		tpl.Ex = a.tpl[0].Ex
	}
	tpl.Pos = a.laserPos()
	tpl.Type = int16(vm.resolveInt(t, p[1]))
	tpl.Colour = int16(vm.resolveInt(t, p[2]))
	tpl.Angle = vm.resolveFloat(t, p[3])
	tpl.Init = vm.resolveFloat(t, p[4])
	tpl.Max = vm.resolveFloat(t, p[5])
	tpl.Delay = vm.resolveInt(t, p[6])
	tpl.Expand = vm.resolveInt(t, p[7])
	tpl.Hold = vm.resolveInt(t, p[8])
	tpl.Shrink = vm.resolveInt(t, p[9])
	tpl.Width = vm.resolveFloat(t, p[10])
	tpl.Flags = uint32(vm.resolveInt(t, p[11]))
	if in.Opcode == opLaserStOn || in.Opcode == opLaserStOnEx {
		tpl.Flags |= 2
	} else {
		tpl.Flags &^= 2
	}
	var id int32
	if ctx != nil && ctx.Lasers != nil {
		id = ctx.Lasers.FireInfinite(&tpl, ctx.LaserEnv)
	}
	if p[0].IsVar {
		vm.storeVar(t, p[0], intVal(id))
	}
}

// changeLaser is 414-419: when the handle names a laser, resolve the rest
// and write it: 414 the tail (+0x24), 415 the template's +0x430 (FUN_0040cf60),
// 416 the speed, 417 the width, 418 the angle, 419 the template's +0x440.
func (vm *VM) changeLaser(ctx *stage.Context, t *eclTask, in *Instr) {
	id := vm.resolveInt(t, in.Params[0])
	if ctx == nil || ctx.Lasers == nil {
		return
	}
	l := ctx.Lasers.Find(id)
	if l == nil {
		return
	}
	b := l.Base()
	switch in.Opcode {
	case opLaserPos, opLaserMove: // y is resolved first
		y := vm.resolveFloat(t, in.Params[2])
		x := vm.resolveFloat(t, in.Params[1])
		if in.Opcode == opLaserPos {
			b.Pos = [3]float32{x, y, 0}
		} else {
			l.SetTpl430([3]float32{x, y, 0})
		}
	case opLaser416:
		b.Speed = vm.resolveFloat(t, in.Params[1])
	case opLaser417:
		b.Width = vm.resolveFloat(t, in.Params[1])
	case opLaser418:
		b.Angle = vm.resolveFloat(t, in.Params[1])
	case opLaser419:
		l.SetTpl440(vm.resolveFloat(t, in.Params[1]))
	}
}
