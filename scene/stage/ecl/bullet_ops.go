package ecl

// Bullet/laser opcodes 400-436 (the "ET" emitter-template system) from
// FUN_0040e770. Each enemy owns 16 templates; opcodes configure a template and
// 401 emits a stacks*ways volley from it. The per-bullet angle/speed model here
// is a faithful first pass (centered fan / ring + aim + speed stacks); exact
// FUN_004067d0 semantics are refined in the danmaku task.

import (
	"math"

	"th10/entity/bullet"
	"th10/entity/item"
	"th10/scene/stage"
)

type eclTemplate struct {
	active   bool
	sprite   int
	color    int
	offX     float64
	offY     float64
	angle    float64
	angleInc float64
	speed    float64
	speedInc float64
	ways     int
	stacks   int
	aim      int
	sound    int
	ex       [22]eclEx
}

type eclEx struct {
	enabled bool
	flags   int
	delay   int
	f1, f2  float64
}

const (
	exFlagSetAccel   = 16
	exFlagAngleAccel = 32
	exFlagSetMotion  = 64
	exFlagExpire     = 8192
	exFlagClear      = -2147483648
)

func (vm *VM) execBulletOp(ctx *stage.Context, t *eclTask, in *Instr) {
	switch in.Opcode {
	case opETNew:
		slot := vm.slot(t, in, 0)
		if slot < 0 {
			return
		}
		t.actorTpl(vm)[slot] = eclTemplate{active: true, sprite: 1, ways: 1, stacks: 1, speed: 2}
	case opETSprite:
		if tp := vm.tpl(t, in); tp != nil {
			tp.sprite = int(vm.resolveInt(t, in.Params[1]))
			tp.color = int(vm.resolveInt(t, in.Params[2]))
		}
	case opETOffset:
		if tp := vm.tpl(t, in); tp != nil {
			tp.offX = float64(vm.resolveFloat(t, in.Params[1]))
			tp.offY = float64(vm.resolveFloat(t, in.Params[2]))
		}
	case opETAngle:
		if tp := vm.tpl(t, in); tp != nil {
			tp.angle = float64(vm.resolveFloat(t, in.Params[1]))
			tp.angleInc = float64(vm.resolveFloat(t, in.Params[2]))
		}
	case opETSpeed:
		if tp := vm.tpl(t, in); tp != nil {
			tp.speed = float64(vm.resolveFloat(t, in.Params[1]))
			if len(in.Params) > 2 {
				tp.speedInc = float64(vm.resolveFloat(t, in.Params[2]))
			}
		}
	case opETCount:
		if tp := vm.tpl(t, in); tp != nil {
			tp.ways = int(vm.resolveInt(t, in.Params[1]))
			tp.stacks = int(vm.resolveInt(t, in.Params[2]))
		}
	case opETAim:
		if tp := vm.tpl(t, in); tp != nil {
			tp.aim = int(vm.resolveInt(t, in.Params[1]))
		}
	case opETSound:
		if tp := vm.tpl(t, in); tp != nil && len(in.Params) >= 2 {
			tp.sound = int(vm.resolveInt(t, in.Params[1]))
		}
	case opETEx:
		vm.applyETEx(t, in)
	case opETCopy:
		if t.owner != nil && len(in.Params) >= 2 {
			tpl := t.actorTpl(vm)
			dst := int(vm.resolveInt(t, in.Params[0]))
			src := int(vm.resolveInt(t, in.Params[1]))
			if inRange(dst, len(tpl)) && inRange(src, len(tpl)) {
				tpl[dst] = tpl[src]
			}
		}
	case opETCountR3: // 425 (case 0x1a9): 3 dynamic-rank tiers of (ways,stacks)
		switch {
		case vm.rank > 511:
			vm.etCountPair(t, in, 2)
		case vm.rank >= -512:
			vm.etCountPair(t, in, 1)
		default:
			vm.etCountPair(t, in, 0)
		}
	case opETCountR5: // 426 (case 0x1aa): 5 dynamic-rank tiers of (ways,stacks)
		switch {
		case vm.rank > 599:
			vm.etCountPair(t, in, 4)
		case vm.rank > 199:
			vm.etCountPair(t, in, 3)
		case vm.rank > -201:
			vm.etCountPair(t, in, 2)
		case vm.rank >= -600:
			vm.etCountPair(t, in, 1)
		default:
			vm.etCountPair(t, in, 0)
		}
	case opETCountR2: // 427 (case 0x1ab): blend (w1,s1)->(w2,s2) linearly over rank
		if tp := vm.tpl(t, in); tp != nil && len(in.Params) >= 5 {
			w1 := vm.resolveInt(t, in.Params[1])
			s1 := vm.resolveInt(t, in.Params[2])
			w2 := vm.resolveInt(t, in.Params[3])
			s2 := vm.resolveInt(t, in.Params[4])
			tp.ways = int(w1 + rankBlend(w2-w1, vm.rank))
			tp.stacks = int(s1 + rankBlend(s2-s1, vm.rank))
		}
	case opETCountRank4: // 436 SSSSSSSSS: slot + ways[4 diff] + stacks[4 diff]
		vm.etCountRank(t, in)
	case opETSpeedRank4: // 435 Sffffffff: slot + speed[4 diff] + speed2[4 diff]
		vm.etSpeedRank(t, in)
	case opETFire:
		if tp := vm.tpl(t, in); tp != nil {
			vm.emit(ctx, t, tp)
		}
	case opETClearAll:
		// 410 (case 0x19a) = FUN_00408210 + FUN_0041c850: clear all enemy
		// bullets and lasers. It does NOT reset the emitter templates.
		if ctx != nil && ctx.Bullets != nil {
			ctx.Bullets.ClearAll()
		}
	case opLaserOnA:
		vm.fireLaserOnA(ctx, t, in)
	case opLaserStOn:
		vm.fireLaserStOn(ctx, t, in)
	case opLaserOnB, opLaserStC, opLaserStB:
		// 428 SSffSfSf / 431,433 SSffSfff: boss laser variants. Layout matches
		// 412 closely (slot, color, angle, speed, headOff, length, _, width);
		// reuse the same emission path.
		vm.fireLaserVariant(ctx, t, in)
	case opETCancel, opETClear:
		// 420/421 (case 0x1a4/0x1a5) = FUN_00408100(radius,...): cancel enemy
		// bullets within `radius` of this enemy, converting them to point
		// items (FUN_004086b0). 420 additionally spawns a per-bullet effect.
		vm.cancelBullets(ctx, t, in)
	}
}

// cancelBullets ports FUN_00408100: every bullet whose distance to the enemy
// is within radius + bulletSize/2 is removed and becomes a point item.
func (vm *VM) cancelBullets(ctx *stage.Context, t *eclTask, in *Instr) {
	if ctx == nil || ctx.Bullets == nil || t.owner == nil || len(in.Params) < 1 {
		return
	}
	radius := float64(vm.resolveFloat(t, in.Params[0]))
	if radius <= 0 {
		return
	}
	cx, cy := t.owner.X, t.owner.Y
	ctx.Bullets.Each(func(b *bullet.Bullet) {
		if !b.Active {
			return
		}
		dx, dy := b.X-cx, b.Y-cy
		r := radius + b.Radius*0.5
		if dx*dx+dy*dy <= r*r {
			b.Active = false
			if ctx.Items != nil {
				ctx.Items.Spawn(b.X, b.Y, item.PointItem)
			}
		}
	})
}

func (t *eclTask) actorTpl(vm *VM) *[16]eclTemplate {
	return &vm.actorFor(t.owner).tpl
}

// slot resolves the template index from param i, or -1 if out of range / no owner.
func (vm *VM) slot(t *eclTask, in *Instr, i int) int {
	if t.owner == nil || len(in.Params) <= i {
		return -1
	}
	s := int(vm.resolveInt(t, in.Params[i]))
	if !inRange(s, 16) {
		return -1
	}
	return s
}

// tpl returns a pointer to the template referenced by param 0, or nil.
func (vm *VM) tpl(t *eclTask, in *Instr) *eclTemplate {
	s := vm.slot(t, in, 0)
	if s < 0 {
		return nil
	}
	return &t.actorTpl(vm)[s]
}

func (vm *VM) applyETEx(t *eclTask, in *Instr) {
	if t.owner == nil || len(in.Params) < 8 {
		return
	}
	slot := int(vm.resolveInt(t, in.Params[0]))
	ex := int(vm.resolveInt(t, in.Params[1]))
	tpl := t.actorTpl(vm)
	if !inRange(slot, len(tpl)) || !inRange(ex, len(tpl[slot].ex)) {
		return
	}
	flags := int(vm.resolveInt(t, in.Params[3]))
	if flags == exFlagClear {
		tpl[slot].ex[ex] = eclEx{}
		return
	}
	tpl[slot].ex[ex] = eclEx{
		enabled: true,
		flags:   flags,
		delay:   int(vm.resolveInt(t, in.Params[4])),
		f1:      float64(vm.resolveFloat(t, in.Params[6])),
		f2:      float64(vm.resolveFloat(t, in.Params[7])),
	}
}

// etCountPair reads the pair-th (ways, stacks) argument pair of a rank-tier
// count opcode (425/426: slot, w0,s0, w1,s1, ...).
func (vm *VM) etCountPair(t *eclTask, in *Instr, pair int) {
	tp := vm.tpl(t, in)
	if tp == nil || len(in.Params) < 3+pair*2 {
		return
	}
	tp.ways = int(vm.resolveInt(t, in.Params[1+pair*2]))
	tp.stacks = int(vm.resolveInt(t, in.Params[2+pair*2]))
}

// rankBlend scales a delta by (rank+1024)/2048 with the binary's
// round-toward-zero shift (FUN case 0x1ab).
func rankBlend(delta int32, rank int) int32 {
	v := delta * int32(rank+1024)
	if v < 0 {
		v += 0x7ff
	}
	return v >> 11
}

// etCountRank sets ways/stacks from per-difficulty arg groups (opcode 436:
// slot + ways[4 difficulties] + stacks[4 difficulties]).
func (vm *VM) etCountRank(t *eclTask, in *Instr) {
	tp := vm.tpl(t, in)
	if tp == nil {
		return
	}
	// args after slot are (ways..., stacks...) per difficulty; use the rank index.
	rest := in.Params[1:]
	half := len(rest) / 2
	if half >= 1 {
		tp.ways = int(vm.resolveInt(t, rest[vm.rankIndex(half)]))
	}
	if len(rest) > half {
		tp.stacks = int(vm.resolveInt(t, rest[half+vm.rankIndex(len(rest)-half)]))
	}
}

// etSpeedRank sets speed/speed2 from per-difficulty arg groups (opcode 435:
// slot + speed[4 difficulties] + speed2[4 difficulties]), mirroring the
// difficulty-pick of etCountRank but for the speed pair.
func (vm *VM) etSpeedRank(t *eclTask, in *Instr) {
	tp := vm.tpl(t, in)
	if tp == nil {
		return
	}
	rest := in.Params[1:]
	half := len(rest) / 2
	if half >= 1 {
		tp.speed = float64(vm.resolveFloat(t, rest[vm.rankIndex(half)]))
	}
	if len(rest) > half {
		tp.speedInc = float64(vm.resolveFloat(t, rest[half+vm.rankIndex(len(rest)-half)]))
	}
}

// emit fires a stacks*ways volley from the template, porting the per-bullet
// angle/speed math of FUN_004067d0 (which FUN_004073e0 calls once per
// way/stack). Constants are the real ones from .tmp/ecl_dump/consts.txt:
// 0.5 (even-fan centering), 2pi (ring), pi (half-ring offset), -1 (side flip).
//
// Field mapping: tp.aim is the template MODE (unaff_EBX[0xfc]); tp.angle/angleInc
// the base angle/step (+0x10/+0x14); tp.speed/speedInc are the binary's
// speed/speed2 (+0x18/+0x1c) — speed2 is the speed the LAST stack approaches.
func (vm *VM) emit(ctx *stage.Context, t *eclTask, tp *eclTemplate) {
	if ctx == nil || ctx.Bullets == nil || t.owner == nil || !t.owner.Active {
		return
	}
	ways := tp.ways
	if ways < 1 {
		ways = 1
	}
	stacks := tp.stacks
	if stacks < 1 {
		stacks = 1
	}
	x := t.owner.X + tp.offX
	y := t.owner.Y + tp.offY
	// FUN_004073e0 aims from the fire origin (enemy + template offset), not the
	// enemy center; exactly pi/2 (down) when it coincides with the player.
	aim := vm.aimAngleAt(x, y)
	btype := mapBulletType(tp.sprite)
	// Loop order matches FUN_004073e0: stacks (outer) x ways (inner).
	for s := 0; s < stacks; s++ {
		for w := 0; w < ways; w++ {
			angle, speed := vm.perBullet(tp, w, s, ways, stacks, aim)
			b := ctx.Bullets.Fire(x, y, speed, angle, btype, tp.color, false)
			if b != nil {
				if r := spriteHitRadius(tp.sprite); r > 0 {
					b.Radius = r // official DAT_004741e0 size, not the shape default
				}
			}
			applyEx(b, tp)
		}
	}
	if tp.sound != 0 {
		vm.playStageSound(ctx, tp.sound)
	}
}

// perBullet computes one bullet's (angle, speed) for way w / stack s, a faithful
// port of FUN_004067d0's switch(mode) at .tmp/ecl_dump/helpers/func_4067d0.c.
func (vm *VM) perBullet(tp *eclTemplate, w, s, ways, stacks int, aim float64) (angle, speed float64) {
	const twoPi = 2 * math.Pi

	// Speed: stack 0 = speed, linearly interpolating toward speed2 over stacks.
	speed = tp.speed
	if stacks >= 2 {
		speed = tp.speed - (tp.speed-tp.speedInc)*float64(s)/float64(stacks)
	}

	switch tp.aim {
	case 0, 1: // centered fan; mode 0 also aims at the player
		var m float64
		if ways&1 == 1 { // odd way count: one bullet on center
			m = float64((w + 1) / 2)
		} else { // even: bullets straddle the center (half-step offset)
			m = float64(w/2) + 0.5
		}
		angle = m * tp.angleInc
		if w&1 == 1 { // odd index goes to the mirror side
			angle = -angle
		}
		if tp.aim == 0 {
			angle += aim
		}
		angle += tp.angle
	case 2, 3: // even ring of `ways`, each stack rotated by angleInc; mode 2 aims
		angle = float64(s)*tp.angleInc + float64(w)*twoPi/float64(ways) + tp.angle
		if tp.aim == 2 {
			angle += aim
		}
	case 4, 5: // ring offset by half a spacing (pi/ways); mode 4 aims
		angle = float64(s)*tp.angleInc + float64(w)*twoPi/float64(ways) + math.Pi/float64(ways) + tp.angle
		if tp.aim == 4 {
			angle += aim
		}
	case 7: // ring angle, random speed (RNG behavioral per Pareto)
		angle = float64(s)*tp.angleInc + float64(w)*twoPi/float64(ways) + tp.angle
		speed = vm.randSpeed(tp)
	case 8: // random angle + random speed
		angle = tp.angle + (float64(vm.randFloat())-0.5)*tp.angleInc*float64(ways)
		speed = vm.randSpeed(tp)
	default: // case 6: random angle
		angle = tp.angle + (float64(vm.randFloat())-0.5)*tp.angleInc*float64(ways)
	}
	return angle, speed
}

// randSpeed picks a speed in [speed2, speed] (FUN_004067d0 case 7/8:
// rand*range + speed2). RNG is behavioral, not ZUN's exact generator.
func (vm *VM) randSpeed(tp *eclTemplate) float64 {
	lo, hi := tp.speedInc, tp.speed
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo + float64(vm.randFloat())*(hi-lo)
}

func applyEx(b *bullet.Bullet, tp *eclTemplate) {
	if b == nil {
		return
	}
	for _, ex := range tp.ex {
		if !ex.enabled {
			continue
		}
		start := ex.delay
		if start < 0 {
			start = 0
		}
		if ex.flags&exFlagExpire != 0 {
			b.AddAction(bullet.Action{Kind: bullet.ActionExpire, Start: start})
		}
		if ex.flags&exFlagSetAccel != 0 {
			a := bullet.Action{Kind: bullet.ActionSetAccel, Start: start}
			if !eclUnset(ex.f1) {
				a.AccelS = ex.f1
			}
			if !eclUnset(ex.f2) {
				a.Angle, a.HasAngle = ex.f2, true
			}
			b.AddAction(a)
		}
		if ex.flags&exFlagAngleAccel != 0 {
			a := bullet.Action{Kind: bullet.ActionSetAngleAccel, Start: start}
			if !eclUnset(ex.f2) {
				a.AccelA = ex.f2
			}
			b.AddAction(a)
		}
		if ex.flags&exFlagSetMotion != 0 {
			a := bullet.Action{Kind: bullet.ActionSetMotion, Start: start}
			if !eclUnset(ex.f1) {
				a.Angle, a.HasAngle = ex.f1, true
			}
			if !eclUnset(ex.f2) {
				a.Speed, a.HasSpeed = ex.f2, true
			}
			b.AddAction(a)
		}
	}
}

func (vm *VM) fireLaserOnA(ctx *stage.Context, t *eclTask, in *Instr) {
	if ctx == nil || ctx.Bullets == nil || t.owner == nil || len(in.Params) < 8 {
		return
	}
	color := int(vm.resolveInt(t, in.Params[1]))
	angle := float64(vm.resolveFloat(t, in.Params[2]))
	speed := float64(vm.resolveFloat(t, in.Params[3]))
	headOff := float64(vm.resolveFloat(t, in.Params[4]))
	length := float64(vm.resolveFloat(t, in.Params[5]))
	width := float64(vm.resolveFloat(t, in.Params[7]))
	x, y := vm.laserOrigin(t, headOff, angle)
	ctx.Bullets.FireLaser(x, y, speed, angle, length, width, color, 240)
}

// fireLaserVariant handles the boss laser opcodes 428 (SSffSfSf) and 431/433
// (SSffSfff), whose param layout mirrors 412 laserOnA:
// (slot, color, angle, speed, headOff, length, _, width).
func (vm *VM) fireLaserVariant(ctx *stage.Context, t *eclTask, in *Instr) {
	if ctx == nil || ctx.Bullets == nil || t.owner == nil || len(in.Params) < 8 {
		return
	}
	color := int(vm.resolveInt(t, in.Params[1]))
	angle := float64(vm.resolveFloat(t, in.Params[2]))
	speed := float64(vm.resolveFloat(t, in.Params[3]))
	headOff := float64(vm.resolveFloat(t, in.Params[4]))
	length := float64(vm.resolveFloat(t, in.Params[5]))
	width := float64(vm.resolveFloat(t, in.Params[7]))
	if length <= 0 {
		length = 256
	}
	if width <= 0 {
		width = 16
	}
	x, y := vm.laserOrigin(t, headOff, angle)
	ctx.Bullets.FireLaser(x, y, speed, angle, length, width, color, 240)
}

func (vm *VM) fireLaserStOn(ctx *stage.Context, t *eclTask, in *Instr) {
	if ctx == nil || ctx.Bullets == nil || t.owner == nil || len(in.Params) < 12 {
		return
	}
	color := int(vm.resolveInt(t, in.Params[2]))
	angle := float64(vm.resolveFloat(t, in.Params[3]))
	headOff := float64(vm.resolveFloat(t, in.Params[4]))
	length := float64(vm.resolveFloat(t, in.Params[5]))
	width := float64(vm.resolveFloat(t, in.Params[10]))
	lifetime := 0
	for _, i := range []int{6, 7, 8, 9} {
		lifetime += maxInt(0, int(vm.resolveInt(t, in.Params[i])))
	}
	if lifetime == 0 {
		lifetime = 180
	}
	x, y := vm.laserOrigin(t, headOff, angle)
	ctx.Bullets.FireLaser(x, y, 0, angle, length, width, color, lifetime)
}

func (vm *VM) laserOrigin(t *eclTask, headOff, angle float64) (float64, float64) {
	x, y := t.owner.X, t.owner.Y
	tpl := t.actorTpl(vm)
	x += tpl[0].offX
	y += tpl[0].offY
	if !eclUnset(headOff) && headOff != 0 {
		x += math.Cos(angle) * headOff
		y += math.Sin(angle) * headOff
	}
	return x, y
}

// spriteSize is DAT_004741e0 from th10.exe: the official per-sprite bullet
// size (diameter-ish; FUN_00408100 uses value*0.5 as the radius) for sprite
// ids 0-27. Id 17 is the bubble (28px).
var spriteSize = [28]float64{
	4, 6, 6, 4, 4, 4, 4, 4, 4, 0, 4, 6, 10, 8,
	8, 8, 8, 28, 6, 10, 8, 8, 8, 6, 4, 6, 4, 0,
}

// mapBulletType picks the closest renderer shape for a sprite id by its
// official size family (the Go renderer has 4 round shapes; exact per-sprite
// art needs the bullet ANM sheet, tracked separately).
func mapBulletType(sprite int) bullet.Type {
	if sprite < 0 || sprite >= len(spriteSize) || spriteSize[sprite] == 0 {
		return bullet.TypeSmall
	}
	switch s := spriteSize[sprite]; {
	case s <= 4:
		return bullet.TypeRice
	case s <= 6:
		return bullet.TypeSmall
	case s <= 8:
		return bullet.TypeMiddle
	default:
		return bullet.TypeLarge
	}
}

// spriteHitRadius returns the official hit radius (size*0.5), or 0 if unknown.
func spriteHitRadius(sprite int) float64 {
	if sprite < 0 || sprite >= len(spriteSize) {
		return 0
	}
	return spriteSize[sprite] * 0.5
}

func inRange(i, n int) bool { return i >= 0 && i < n }
