package ecl

// Game opcodes 256-368 (FUN_0040e770): enemy creation, ANM slots, movement
// (motion.go), state, drops and boss phases. Bullet/laser opcodes (400-436)
// live in bullet_ops.go.
//
// Every handler reads its args in the binary's order: a var arg of -1 pops
// the data stack, so the order is part of the semantics.

import (
	"math"
	"strings"
	"sync"

	"golang.org/x/text/encoding/japanese"

	"th10/anm"
	"th10/entity/bullet"
	"th10/entity/enemy"
	"th10/gamedata"
	"th10/scene/stage"
	"th10/zun"
)

// eclActor is one enemy's ECL state (the EnemyData at enemy+0x103c).
type eclActor struct {
	e      *enemy.Enemy
	mv     motion
	main   *eclTask   // the primary context (an interrupt retargets it)
	asyncs []*eclTask // async contexts, newest first
	// vars are I0-I3 and F0-F3 (special vars -9985..-9978, +0x1138), copied
	// from the creator.
	vars [8]uint32
	// ANM state: the file ins_258 selected (+0xe8), the file of the slot-0
	// script (+0xec), the slot-0 script (+0xf0) and ins_262's script, which
	// ins_269 offsets (+0xf4).
	anmFile, bodyFile, curScript, dirScript int
	// bombScript and savedScript are ins_366's swap (+0x1448, +0x144c).
	bombScript, savedScript int
	// vms are the enemy's ANM objects in the effect world (+0xc0, visual.go).
	vms [slotCount]*anm.VM
	// dir is the way slot 0 faces (+0xf8): -1 left, 0 still, 1 right.
	dir int32
	// flash counts down the hit flash's hold-off (+0x1414).
	flash int32
	// phaseHP is the HP left before the first armed interrupt slot's
	// threshold (B+0x2404), which checkInterrupt rewrites each time it
	// reaches that slot; the hit sound reads it.
	phaseHP int32

	drops   dropTable
	score   int32               // the defeat score (enemy +0x23f8)
	deathSE int32               // sound table entry (+0x2444)
	deathFx int32               // bullet.anm script (+0x2448)
	tpl     [16]bullet.Template // bullet emitter templates (+0x2c4, bullet_ops.go)
	tplOff  [16][3]float32      // their fire offsets from the enemy (+0x1344)
	fireR   float32             // ins_346's squared no-fire radius (+0x1454)
	markers []lifeMarker

	slots   [8]interruptSlot // boss phase interrupts (interrupt.go)
	phase   eclTimer         // phase timer, reset by ins_333 and each interrupt
	invuln  eclTimer         // ins_335 invulnerability timer (+0x2458)
	bossIdx int              // the boss slot ins_332 put this enemy in
}

// dropTable is the defeat drops (+0x13cc): items[0] is the main item's kind
// (enemy +0x2408), items[k] the count of kind k for the scatter, and w, h
// the scatter ellipse (ins_328).
type dropTable struct {
	items [13]int32
	w, h  float32
}

type lifeMarker struct {
	index int
	hp    float64
	raw   int32
}

func newActor(e *enemy.Enemy) *eclActor {
	a := &eclActor{e: e}
	for i := range a.slots {
		a.slots[i].hp = -1
	}
	return a
}

// actorFor returns e's actor, registering one for an enemy the VM did not
// create (tests).
func (vm *VM) actorFor(e *enemy.Enemy) *eclActor {
	a := vm.actors[e]
	if a == nil {
		a = newActor(e)
		vm.actors[e] = a
		vm.order = append(vm.order, a)
	}
	return a
}

// syncPos publishes the final position to the enemy in world coordinates.
func (a *eclActor) syncPos() {
	a.e.X, a.e.Y = eclToWorld(float64(a.mv.final.pos[0]), float64(a.mv.final.pos[1]))
}

// boss0 is the live enemy in boss slot 0 (DAT_00477704+0x10), or nil.
func (vm *VM) boss0() *eclActor {
	if b := vm.bossSlots[0]; b != nil && b.Active {
		return vm.actors[b]
	}
	return nil
}

// boss0Pos is boss slot 0's position (enemy +0x1068), for the infinite
// lasers that follow it.
func (vm *VM) boss0Pos() ([3]float32, bool) {
	if a := vm.boss0(); a != nil {
		return a.mv.final.pos, true
	}
	return [3]float32{}, false
}

// set reports a float arg that is not the "keep the current value" sentinel.
func set(v float32) bool { return float64(v) > unsetArg }

// eclUnset is the sentinel test on an already-widened arg.
func eclUnset(v float64) bool { return v <= unsetArg }

// mirrorAngle reflects a heading about the vertical axis for a mirrored
// enemy (flag 0x800): pi/2 - norm(a - pi/2).
func mirrorAngle(a float32) float32 {
	return normAngle(halfPi32 - normAngle(a-halfPi32))
}

// execGameOp dispatches opcodes >= 256.
func (vm *VM) execGameOp(ctx *stage.Context, t *eclTask, in *Instr) {
	if in.Opcode >= 400 {
		vm.execBulletOp(ctx, t, in)
		return
	}
	switch in.Opcode {
	case opEnmCreate, opEnmCreateA, opEnmCreateM, opEnmCreateAM, 265, 266, 267, 268, opEnmCreate3D, 271:
		vm.createOp(ctx, t, in)
	case opAnmSelect:
		if t.owner != nil && len(in.Params) >= 1 {
			f := int(vm.resolveInt(t, in.Params[0]))
			vm.actorFor(t.owner).anmFile = f
		}
	case opAnmSetSlot, opAnmSetMain, opAnmSetDir5:
		vm.slotOp(ctx, t, in)
	case opAnmPlay, opAnmPlayAbs, opAnmPlayTail, opAnmPlayRot:
		vm.effectOp(ctx, t, in)

	// --- movement ---
	case opSetPos, opSetPosB:
		vm.opSetPos(t, in, layerOf(in.Opcode, opSetPos))
	case opMovePos, opMovePosB:
		vm.opMovePos(t, in, layerOf(in.Opcode, opMovePos))
	case opSetVel, opSetVelB:
		vm.opSetVel(t, in, layerOf(in.Opcode, opSetVel))
	case opMoveVel, opMoveVelB:
		vm.opMoveVel(t, in, layerOf(in.Opcode, opMoveVel))
	case opSetCircle, opSetCircleB:
		vm.opSetCircle(t, in, layerOf(in.Opcode, opSetCircle))
	case opMoveCircle, opMoveCircleB:
		vm.opMoveCircle(t, in, layerOf(in.Opcode, opMoveCircle))
	case opMoveRand, opMoveRandB:
		vm.opMoveRand(t, in, layerOf(in.Opcode, opMoveRand))
	case opPosToBoss, opPosToBossB:
		if t.owner != nil {
			if b := vm.boss0(); b != nil {
				vm.actorFor(t.owner).mv.layer[layerOf(in.Opcode, opPosToBoss)].pos = b.mv.final.pos
			}
		}
	case opPosAdd, opPosAddB: // fff
		if t.owner != nil && len(in.Params) >= 3 {
			l := &vm.actorFor(t.owner).mv.layer[layerOf(in.Opcode, opPosAdd)]
			x := vm.resolveFloat(t, in.Params[0])
			y := vm.resolveFloat(t, in.Params[1])
			z := vm.resolveFloat(t, in.Params[2])
			l.pos[0] += x
			l.pos[1] += y
			l.pos[2] += z
		}
	case opCircleCentre, opCircleCentreB: // ff
		if t.owner != nil && len(in.Params) >= 2 {
			a := vm.actorFor(t.owner)
			l := &a.mv.layer[layerOf(in.Opcode, opCircleCentre)]
			x := vm.resolveFloat(t, in.Params[0])
			y := vm.resolveFloat(t, in.Params[1])
			if set(x) {
				l.delta[0] = x
			}
			if set(y) {
				l.delta[1] = y
			}
			// Then as 294, for either layer.
			if b := vm.boss0(); b != nil {
				a.mv.layer[0].pos = b.mv.final.pos
			}
		}
	case opMoveLimit: // ffff: (x, y, w, h), flag 0x200
		if t.owner != nil && len(in.Params) >= 4 {
			a := vm.actorFor(t.owner)
			for i := range a.mv.limit {
				a.mv.limit[i] = vm.resolveFloat(t, in.Params[i])
			}
			t.owner.ECLFlags |= 0x200
		}
	case opMoveLimitR:
		if t.owner != nil {
			t.owner.ECLFlags &^= 0x200
		}

	// --- enemy state ---
	case opSetHurtbox:
		if t.owner != nil && len(in.Params) >= 2 {
			w := vm.resolveFloat(t, in.Params[0])
			h := vm.resolveFloat(t, in.Params[1])
			t.owner.HitWidth, t.owner.HitHeight = math.Abs(float64(w)), math.Abs(float64(h))
		}
	case opSetHitbox:
		if t.owner != nil && len(in.Params) >= 2 {
			w := vm.resolveFloat(t, in.Params[0])
			h := vm.resolveFloat(t, in.Params[1])
			t.owner.BodyWidth, t.owner.BodyHeight = math.Abs(float64(w)), math.Abs(float64(h))
		}
	case opFlagSet:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.ECLFlags |= int(vm.resolveInt(t, in.Params[0]))
			if t.owner.ECLFlags&0x10 != 0 {
				vm.showSlots(vm.actorFor(t.owner), false)
			}
		}
	case opFlagClear:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.ECLFlags &^= int(vm.resolveInt(t, in.Params[0]))
			if t.owner.ECLFlags&0x10 == 0 {
				vm.showSlots(vm.actorFor(t.owner), true)
			}
		}
	case opSetHP: // 331 S: HP and max HP; the life-bar markers go
		if t.owner != nil && len(in.Params) >= 1 {
			hp := int(vm.resolveInt(t, in.Params[0]))
			t.owner.HP, t.owner.MaxHP = hp, hp
			vm.actorFor(t.owner).markers = nil
		}
	case opSetBoss: // 332 S: take boss slot idx (flag 0x8000); idx < 0 leaves it
		if t.owner != nil && len(in.Params) >= 1 {
			a := vm.actorFor(t.owner)
			if idx := int(vm.resolveInt(t, in.Params[0])); idx < 0 {
				if t.owner.ECLFlags&0x8000 != 0 {
					vm.bossSlots[a.bossIdx] = nil
				}
				t.owner.ECLFlags &^= 0x8000
			} else if idx < len(vm.bossSlots) {
				t.owner.ECLFlags |= 0x8000
				vm.bossSlots[idx] = t.owner
				a.bossIdx = idx
			}
		}
	case opTimerReset: // 333: the phase timer (FUN_00405410(0))
		if t.owner != nil {
			vm.actorFor(t.owner).phase.set(0)
		}
	case opSetInvuln: // 335 S
		if t.owner != nil && len(in.Params) >= 1 {
			vm.actorFor(t.owner).invuln.set(vm.resolveInt(t, in.Params[0]))
		}
	case opDropClear: // 326: the scatter counts, not the main item
		if t.owner != nil {
			clear(vm.actorFor(t.owner).drops.items[1:])
		}
	case opDropExtra: // 327 SS: (kind, count), assigned; kind 0 is the main item
		if t.owner != nil && len(in.Params) >= 2 {
			kind := bossItem(t.owner, vm.resolveInt(t, in.Params[0]))
			n := vm.resolveInt(t, in.Params[1])
			if kind >= 0 && int(kind) < len(dropTable{}.items) {
				vm.actorFor(t.owner).drops.items[kind] = n
			}
		}
	case opDropArea: // 328 ff: (w, h), h read first
		if t.owner != nil && len(in.Params) >= 2 {
			d := &vm.actorFor(t.owner).drops
			d.h = vm.resolveFloat(t, in.Params[1])
			d.w = vm.resolveFloat(t, in.Params[0])
		}
	case opDropMain: // 330 S: the main item's kind
		if t.owner != nil && len(in.Params) >= 1 {
			vm.actorFor(t.owner).drops.items[0] = bossItem(t.owner, vm.resolveInt(t, in.Params[0]))
		}
	case opDropItems: // 329: FUN_0040c9a0
		if t.owner != nil {
			vm.dropItems(ctx, vm.actorFor(t.owner))
		}
	case opSetInterrupt: // 334 SSSm: (slot, hp, time, sub), FUN_00412720
		if t.owner != nil && len(in.Params) >= 4 {
			a := vm.actorFor(t.owner)
			time := vm.resolveInt(t, in.Params[2])
			hp := vm.resolveInt(t, in.Params[1])
			if i := int(vm.resolveInt(t, in.Params[0])); i >= 0 && i < len(a.slots) {
				s := &a.slots[i]
				if s.hp = hp; s.hp >= 0 {
					s.time = time
					s.sub, s.timeoutSub = in.Params[3].Str, in.Params[3].Str
				}
			}
		}
	case opPlaySound:
		if len(in.Params) >= 1 {
			vm.playStageSound(ctx, int(vm.resolveInt(t, in.Params[0])))
		}
	case opScreenShake: // 337 SSS: (duration, magnitude, _) — screen shake
		if len(in.Params) >= 3 {
			vm.resolveInt(t, in.Params[2])
			mag := float64(vm.resolveInt(t, in.Params[1]))
			dur := int(vm.resolveInt(t, in.Params[0]))
			if dur > 0 && mag > 0 {
				vm.shakeFrames, vm.shakeTotal, vm.shakeMag = dur, dur, mag
			}
		}
	case opMSGOpen: // 338 S (case 0x152): open .msg entry id (FUN_00415db0),
		// whose constructor cancels every bullet, then clear the rect and the
		// lasers (FUN_0041c850) and sweep the enemies.
		if len(in.Params) >= 1 {
			id := vm.resolveInt(t, in.Params[0])
			if ctx != nil {
				ctx.OpenMSG(id)
				ctx.Bullets.RectClear(false, ctx.BulletEnv)
				ctx.Lasers.ClearAll(false, ctx.LaserEnv)
			}
			vm.sweepEnemies()
		}
	case opMSGWait: // 339 (case 0x153): wait for the MSG's op 11 or its end
		if ctx != nil && ctx.MSGWaiting() {
			t.blocked = true
		}
	case opBossWait: // 340: block while boss slot 0 is taken
		if vm.boss0() != nil {
			t.blocked = true
		}
	case opSetTimeout: // 341 Sm (case 0x155): (slot, sub) run when the slot times out
		if t.owner != nil && len(in.Params) >= 2 {
			a := vm.actorFor(t.owner)
			if i := int(vm.resolveInt(t, in.Params[0])); i >= 0 && i < len(a.slots) {
				a.slots[i].timeoutSub = in.Params[1].Str
			}
		}
	case opSpellEnd: // case 0x157 (FUN_00409c00) leaves the bullets alone
		vm.card.Active = false // 符卡结束：撤下 HUD
	case opSpellBg, opSpellAlt, opSpell, opSpell2, opSpell3: // SSSx: (idx, timeLimit, bonus, name)
		vm.declareSpell(t, in)
	case opBombSwap: // case 0x16e: (armed, bomb script)
		if t.owner != nil && len(in.Params) >= 2 {
			a := vm.actorFor(t.owner)
			if vm.resolveInt(t, in.Params[0])&1 != 0 {
				t.owner.ECLFlags |= 0x100000
			} else {
				t.owner.ECLFlags &^= 0x100000
			}
			a.bombScript = int(vm.resolveInt(t, in.Params[1]))
			t.owner.ECLFlags &^= 0x200000
			a.savedScript = a.dirScript
		}
	case 362: // case 0x16a: the card is survived, not shot down (+0x378c bit 3)
		vm.card.Survival = true
	case opLifeMarker:
		vm.addLifeMarker(t, in)
	case opRankPickI:
		if len(in.Params) >= 5 {
			vm.storeVar(t, in.Params[0], intVal(vm.rankPickInt(t, in.Params[1:])))
		}
	case opRankPickF:
		if len(in.Params) >= 5 {
			vm.storeVar(t, in.Params[0], floatVal(vm.rankPickFloat(t, in.Params[1:])))
		}
	case opWaitRank: // 368 SSSS: difficulty-indexed wait (Easy,Normal,Hard,Lunatic), as 83
		if len(in.Params) > 0 {
			t.clock -= float32(vm.resolveInt(t, in.Params[vm.rankIndex(len(in.Params))]))
		}
	case 364: // case 0x16c: set/clear enemy flag bit 0x80000
		if t.owner != nil && len(in.Params) >= 1 {
			if vm.resolveInt(t, in.Params[0]) != 0 {
				t.owner.ECLFlags |= 0x80000
			} else {
				t.owner.ECLFlags &^= 0x80000
			}
		}
	case 367: // case 0x16f: the game speed (DAT_00476f78)
		if len(in.Params) >= 1 {
			vm.gameSpeed = vm.resolveFloat(t, in.Params[0])
			if ctx != nil && ctx.Anm != nil {
				ctx.Anm.GameSpeed = vm.gameSpeed
			}
		}
	case opEnemySweep: // 345 (case 0x159)
		vm.sweepEnemies()
	case 346: // case 0x15a: 401 holds fire while the player is within r
		if t.owner != nil && len(in.Params) >= 1 {
			r := vm.resolveFloat(t, in.Params[0])
			vm.actorFor(t.owner).fireR = float32(r * r)
		}
	default:
		// Remaining no-ops with known-but-cosmetic binary effects: 344 chapter
		// bookkeeping, 360 stars, 361 (FUN_00405410), 363 (FUN_0040cea0), 365
		// (FUN_0041c7d0). UI/spell-frame ops likewise.
	}
}

// layerOf returns 0 (layer A) for the base opcode, 1 (B) for its variant.
func layerOf(op, base uint16) int {
	if op == base {
		return 0
	}
	return 1
}

// opSetPos is 280/282: place the layer and recompute the final position at
// once. Running tweens are left alone.
func (vm *VM) opSetPos(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 2 {
		return
	}
	a := vm.actorFor(t.owner)
	l := &a.mv.layer[i]
	x := vm.resolveFloat(t, in.Params[0])
	y := vm.resolveFloat(t, in.Params[1])
	if set(x) {
		l.pos[0] = x
	}
	if set(y) {
		l.pos[1] = y
	}
	l.circle = false
	a.mv.setFinal()
	a.syncPos()
}

// opMovePos is 281/283: tween the layer to (x, y, 0) over time frames.
func (vm *VM) opMovePos(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 4 {
		return
	}
	a := vm.actorFor(t.owner)
	l := &a.mv.layer[i]
	x := vm.resolveFloat(t, in.Params[2])
	y := vm.resolveFloat(t, in.Params[3])
	dur := vm.resolveInt(t, in.Params[0])
	mode := vm.resolveInt(t, in.Params[1])
	p := &a.mv.posI[i]
	p.Start = l.pos
	p.End = [3]float32{l.pos[0], l.pos[1], 0}
	if set(x) {
		p.End[0] = x
	}
	if set(y) {
		p.End[1] = y
	}
	p.Begin(dur, mode)
	l.circle = false
}

// opSetVel is 284/286: heading (mirrored under flag 0x800) and speed.
func (vm *VM) opSetVel(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 2 {
		return
	}
	l := &vm.actorFor(t.owner).mv.layer[i]
	ang := vm.resolveFloat(t, in.Params[0])
	speed := vm.resolveFloat(t, in.Params[1])
	if set(ang) {
		if t.owner.ECLFlags&0x800 != 0 {
			ang = mirrorAngle(ang)
		}
		l.angle = normAngle(ang)
	}
	if set(speed) {
		l.speed = speed
	}
	l.circle = false
}

// opMoveVel is 285/287: tween heading and speed. In mode 7 the args are
// per-frame increments (unset = 0, never mirrored). The heading takes the
// short way round when the two are pi or more apart.
func (vm *VM) opMoveVel(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 4 {
		return
	}
	a := vm.actorFor(t.owner)
	l := &a.mv.layer[i]
	ang := vm.resolveFloat(t, in.Params[2])
	speed := vm.resolveFloat(t, in.Params[3])
	mode := vm.resolveInt(t, in.Params[1])
	dur := vm.resolveInt(t, in.Params[0])
	if mode == 7 {
		if !set(ang) {
			ang = 0
		}
		if !set(speed) {
			speed = 0
		}
	} else {
		if !set(ang) {
			ang = l.angle
		} else if t.owner.ECLFlags&0x800 != 0 {
			ang = mirrorAngle(ang)
		}
		if !set(speed) {
			speed = l.speed
		}
	}
	start := l.angle
	if d := start - ang; pi32 <= float32(math.Abs(float64(d))) {
		if ang <= start {
			ang += twoPi32
		} else {
			start += twoPi32
		}
	}
	p := &a.mv.velI[i]
	p.Start = [3]float32{start, l.speed}
	p.End = [3]float32{ang, speed}
	p.Begin(dur, mode)
	l.circle = false
}

// opSetCircle is 288/290: circle the current position (if not circling
// already) with the given angle, angular speed, radius and radial speed.
func (vm *VM) opSetCircle(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 4 {
		return
	}
	l := &vm.actorFor(t.owner).mv.layer[i]
	ang := vm.resolveFloat(t, in.Params[0])
	speed := vm.resolveFloat(t, in.Params[1])
	r := vm.resolveFloat(t, in.Params[2])
	rv := vm.resolveFloat(t, in.Params[3])
	if !l.circle {
		l.delta = l.pos
	}
	if set(ang) {
		l.angle = normAngle(ang)
	}
	if set(speed) {
		l.speed = speed
	}
	if set(r) {
		l.radius = r
	}
	if set(rv) {
		l.radVel = rv
	}
	l.circle = true
}

// opMoveCircle is 289/291: tween the circle's angle/angular speed and
// radius/radial speed, circling the current position.
func (vm *VM) opMoveCircle(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 6 {
		return
	}
	a := vm.actorFor(t.owner)
	l := &a.mv.layer[i]
	ang := vm.resolveFloat(t, in.Params[2])
	speed := vm.resolveFloat(t, in.Params[3])
	r := vm.resolveFloat(t, in.Params[4])
	rv := vm.resolveFloat(t, in.Params[5])
	if !set(speed) {
		speed = l.speed
	}
	if !set(ang) {
		ang = l.angle
	}
	if !set(rv) {
		rv = l.radVel
	}
	if !set(r) {
		r = l.radius
	}
	dur := vm.resolveInt(t, in.Params[0])
	mode := vm.resolveInt(t, in.Params[1])
	v := &a.mv.velI[i]
	v.Start, v.End = [3]float32{l.angle, l.speed}, [3]float32{ang, speed}
	v.Begin(dur, mode)
	c := &a.mv.circI[i]
	c.Start, c.End = [3]float32{l.radius, l.radVel}, [3]float32{r, rv}
	c.Begin(dur, mode)
	l.delta = l.pos
	l.circle = true
}

// opMoveRand is 292/293 (case 0x124): a random heading from one RNG draw,
// fanned by where the final position sits in the ins_324 box (x bands of
// ±w/4) and which side the player is on, then pushed back toward the box's
// ±h/4 y band; the speed tweens from the arg down to 0.
func (vm *VM) opMoveRand(t *eclTask, in *Instr, i int) {
	if t.owner == nil || len(in.Params) < 3 {
		return
	}
	a := vm.actorFor(t.owner)
	m := &a.mv
	fx, fy := m.final.pos[0], m.final.pos[1]
	lx, ly := m.limit[0], m.limit[1]
	var ang float32
	q := float32(m.limit[2] * 0.25)
	switch {
	case fx < lx-q: // past the left margin: fan to the right
		ang = float32(vm.randSigned() * thirdPi32)
	case q+lx < fx: // past the right margin: fan to the left
		ang = normAngle(float32(float32(vm.randSigned()*pi32)*third32) + pi32)
	case fx < vm.playerECL[0]: // player to the right
		ang = float32(vm.randSigned() * halfPi32)
	default:
		ang = normAngle(float32(float32(vm.randSigned()*pi32)*0.5) + pi32)
	}
	qh := float32(m.limit[3] * 0.25)
	switch {
	case fy < ly-qh:
		ang = float32(math.Abs(float64(ang)))
	case qh+ly < fy:
		ang = -float32(math.Abs(float64(ang)))
	}
	mode := vm.resolveInt(t, in.Params[1])
	speed := vm.resolveFloat(t, in.Params[2])
	dur := vm.resolveInt(t, in.Params[0])
	p := &m.velI[i]
	p.Start = [3]float32{ang, speed}
	p.End = [3]float32{ang, 0}
	p.Begin(dur, mode)
	m.layer[i].circle = false
}

// createOp is 256-261/265-268/270/271 and FUN_0040cfb0. The 265-268/271
// forms only create while boss slot 0 is empty. The new enemy is updated
// once on the spot (movement, then its ECL), before it joins the list, so
// what its sub does at time 0 is in place before the creator's next
// instruction.
func (vm *VM) createOp(ctx *stage.Context, t *eclTask, in *Instr) {
	op := in.Opcode
	switch op {
	case 265, 266, 267, 268, 271:
		if vm.boss0() != nil {
			return
		}
		op = map[uint16]uint16{265: 256, 266: 257, 267: 260, 268: 261, 271: 270}[op]
	}
	n := 6
	if op == opEnmCreate3D {
		n = 7
	}
	if len(in.Params) < n {
		return
	}
	parent := vm.actors[t.owner]
	name := in.Params[0].Str
	var pos [3]float32
	pos[0] = vm.resolveFloat(t, in.Params[1])
	pos[1] = vm.resolveFloat(t, in.Params[2])
	if op == opEnmCreate3D {
		// 270 adds the STD camera's x and y (DAT_00491d7c/80), not its z.
		if ctx != nil && ctx.STD != nil {
			pos[0] += ctx.STD.Cam.Pos[0]
			pos[1] += ctx.STD.Cam.Pos[1]
		}
		pos[2] = vm.resolveFloat(t, in.Params[3])
	} else if op == opEnmCreate && parent != nil && parent.e.ECLFlags&0x40000 != 0 && ctx != nil && ctx.STD != nil {
		// A 3D creator's child starts where the creator shows on screen.
		p := parent.mv.final.pos
		p[0] += pos[0]
		p[1] += pos[1]
		pos[0], pos[1] = ctx.STD.Cam.Project(p)
	} else if (op == opEnmCreate || op == opEnmCreateM) && parent != nil {
		pos[0] += parent.mv.final.pos[0]
		pos[1] += parent.mv.final.pos[1]
	}
	hp := vm.resolveInt(t, in.Params[n-3])
	score := vm.resolveInt(t, in.Params[n-2])
	drop := vm.resolveInt(t, in.Params[n-1])

	typ := enemy.TypeFairy
	switch {
	case strings.Contains(name, "MBoss"):
		typ = enemy.TypeMidBoss
	case strings.Contains(name, "Boss"):
		typ = enemy.TypeBoss
	}
	wx, wy := eclToWorld(float64(pos[0]), float64(pos[1]))
	e := enemy.New(wx, wy, int(hp), typ)
	e.Damaged = vm.damaged
	if op == opEnmCreateM || op == opEnmCreateAM {
		e.ECLFlags |= 0x800
	}
	if op == opEnmCreate3D {
		e.ECLFlags |= 0x40000
	}

	a := newActor(e)
	a.mv.layer[0].pos = pos
	a.score = score
	a.drops.items[0] = drop
	if parent != nil {
		a.vars = parent.vars
	}
	a.invuln.set(2)
	a.main = &eclTask{sub: vm.prog.SubByName[name], owner: e, asyncID: -1}
	if a.main.sub != nil {
		a.main.frame = newFrame(a.main.sub, nil)
	}
	vm.actors[e] = a
	vm.updateActor(a) // FUN_0040cfb0 ignores its result
	a.drops.items[0] = bossItem(e, a.drops.items[0])
	a.deathSE = 2 + vm.created&1
	a.deathFx = deathEffect(a)
	vm.created++
	vm.order = append(vm.order, a)
	if ctx != nil {
		ctx.AddEnemy(e)
	}
}

// anmFiles caches parsed ANM files by name (nil: not embedded).
var anmFiles sync.Map

func loadANM(name string) *anm.File {
	if v, ok := anmFiles.Load(name); ok {
		return v.(*anm.File)
	}
	f, err := gamedata.ANM(name)
	if err != nil {
		f = nil
	}
	anmFiles.Store(name, f)
	return f
}

// bossItem is the boss remap of a power item kind (flag 0x8000, ins_327,
// ins_330 and FUN_0040cfb0): small and big power become kinds 10 and 11.
func bossItem(e *enemy.Enemy, kind int32) int32 {
	if e.ECLFlags&0x8000 != 0 {
		switch kind {
		case 1:
			return 10
		case 4:
			return 11
		}
	}
	return kind
}

// deathEffect is the bullet.anm script FUN_0040cfb0 picks for the defeat
// effect after the first update: by the slot-0 script when it comes from
// the stage's first ANM file, else 0x167.
func deathEffect(a *eclActor) int32 {
	if a.bodyFile == 1 {
		switch a.curScript {
		case 5, 0x19, 0x32:
			return 0x164
		case 10, 0x1e, 0x33:
			return 0x16a
		case 0xf, 0x23, 0x34:
			return 0x16d
		}
	}
	return 0x167
}

// The launch of every dropped item (FUN_0040c9a0, FUN_0040c9d0): straight
// up at 2.2.
const (
	dropAngle float32 = -zun.HalfPi
	dropSpeed float32 = 2.2
)

// kill is the defeat in FUN_0040dc80: the score (FUN_00409d90 adds a tenth
// of it to the stored score, which shows times 10), then FUN_0040e5f0: the
// death sound, the death effect, the drops, then the faith gauge's hold timer
// gains 10 frames (FUN_0040e5f0 0x40e681).
func (vm *VM) kill(a *eclActor) {
	ctx := vm.ctx
	if ctx == nil {
		vm.dropItems(nil, a)
		return
	}
	if ctx.State != nil {
		ctx.State.AddPoints(int64(a.score))
	}
	if a.deathSE >= 0 {
		ctx.PlaySEID(int(a.deathSE))
	}
	vm.spawnDeathFx(ctx, a)
	vm.dropItems(ctx, a)
	if ctx.State != nil {
		ctx.State.AddFaithHold(10) // FUN_0040e5f0: FUN_00412ff0(10)
	}
}

// spawnDeathFx is the death effect of FUN_0040e5f0 (0x40e613) and of the
// sweep FUN_0040e6a0 (0x40e6e8): bullet.anm's deathFx script on layer 0 at the
// enemy's final position, from the main RNG.
func (vm *VM) spawnDeathFx(ctx *stage.Context, a *eclActor) {
	if a.deathFx >= 0 && ctx != nil && ctx.World != nil && ctx.BulletEnv != nil {
		ctx.World.Spawn(ctx.BulletEnv.File, int(a.deathFx), a.mv.final.pos, ctx.Anm)
	}
}

// dropItems is FUN_0040c9a0 (ins_329, and the tail of a defeat): the main
// item at the enemy, then the scatter; the main item is used up.
func (vm *VM) dropItems(ctx *stage.Context, a *eclActor) {
	pos := a.mv.final.pos
	d := &a.drops
	if d.items[0] > 0 && ctx != nil {
		ctx.SpawnItem(pos[0], pos[1], d.items[0], dropAngle, dropSpeed)
	}
	vm.scatterDrops(ctx, d, pos)
	d.items[0] = 0
}

// scatterDrops is FUN_0040c9d0: kinds 1-11 drop round the drop ellipse from
// a random start, each at 0.5-1 of the radius, the next a quarter turn on
// give or take an eighth; then every count is cleared. It draws at least
// once. The two-step draws inside repeat their second step (zun.RNG.Dup).
func (vm *VM) scatterDrops(ctx *stage.Context, d *dropTable, pos [3]float32) {
	angle := float32(vm.randSigned() * zun.Pi)
	for kind := int32(1); kind <= 11; kind++ {
		for range d.items[kind] {
			px, py := zun.Ellipse(angle, d.w, d.h)
			f := float32(float32(vm.rng.Dup())*0x1p-33) + 0.5
			x := float32(px*f) + pos[0]
			y := float32(py*f) + pos[1]
			if ctx != nil {
				ctx.SpawnItem(x, y, kind, dropAngle, dropSpeed)
			}
			r := float32(float32(vm.rng.Dup())*0x1p-31) - 1
			angle = zun.NormAngle(float32(float32(r*(zun.Pi/4))+angle) + zun.HalfPi)
		}
	}
	clear(d.items[1:])
}

// playStageSound plays entry id of the sound table (0x4749c8).
func (vm *VM) playStageSound(ctx *stage.Context, id int) {
	if ctx != nil {
		ctx.PlaySEID(id)
	}
}

func (vm *VM) rankPickInt(t *eclTask, args []Param) int32 {
	return vm.resolveInt(t, args[vm.rankIndex(len(args))])
}

func (vm *VM) rankPickFloat(t *eclTask, args []Param) float32 {
	return vm.resolveFloat(t, args[vm.rankIndex(len(args))])
}

// rankIndex maps difficulty to the arg index for the 5-way rank-pick opcodes
// (Easy, Normal, Hard, Lunatic, Extra).
func (vm *VM) rankIndex(n int) int {
	idx := vm.difficulty
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return idx
}

// spellSJIS decodes a spell-card name. The ECL loader already XOR-decrypts the
// 'x' string; here we convert its Shift-JIS bytes to UTF-8 for display.
var spellSJISDec = japanese.ShiftJIS.NewDecoder()

// declareSpell handles 342/348/357-359 (cases 0x156, 0x15c, 0x165-0x167):
// (idx, timeLimit, bonus, name). The card's index is idx, plus the difficulty
// for 357, less one for 358 and two for 359 (FUN_00409280's param); its timer
// restarts, the capture is open and the survival bit (362) clears. The time
// limit is the interrupt slot's.
func (vm *VM) declareSpell(t *eclTask, in *Instr) {
	if len(in.Params) < 4 {
		return
	}
	id := vm.resolveInt(t, in.Params[0])
	switch in.Opcode {
	case opSpell:
		id += int32(vm.difficulty)
	case opSpell2:
		id += int32(vm.difficulty) - 1
	case opSpell3:
		id += int32(vm.difficulty) - 2
	}
	bonus := vm.resolveInt(t, in.Params[2])
	vm.resolveInt(t, in.Params[1])
	name := in.Params[3].Str
	if utf, err := spellSJISDec.String(name); err == nil && utf != "" {
		name = utf
	}
	vm.card.Active, vm.card.Capture = true, true
	vm.card.Survival = false
	vm.card.ID = id
	vm.card.Timer.Reset()
	vm.card.Bonus = int64(bonus)
	vm.spell = spellState{name: name, owner: t.owner}
}

// addLifeMarker is 347 SfS: (index, hp, colour), read last to first.
func (vm *VM) addLifeMarker(t *eclTask, in *Instr) {
	if t.owner == nil || len(in.Params) < 3 {
		return
	}
	raw := vm.resolveInt(t, in.Params[2])
	hp := vm.resolveFloat(t, in.Params[1])
	marker := lifeMarker{index: int(vm.resolveInt(t, in.Params[0])), hp: float64(hp), raw: raw}
	a := vm.actorFor(t.owner)
	for i := range a.markers {
		if a.markers[i].index == marker.index {
			a.markers[i] = marker
			return
		}
	}
	a.markers = append(a.markers, marker)
}
