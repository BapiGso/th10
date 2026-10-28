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
	"th10/entity/item"
	"th10/gamedata"
	"th10/scene/stage"
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
	// ANM state: the file ins_258 selected (+0xe8), the slot-0 script (+0xf0)
	// and ins_262's script, which ins_269 offsets (+0xf4).
	anmFile, curScript, dirScript int
	ran                           bool // updated this frame (flag 0x400 was set)

	dropArea eclRect
	tpl      [16]bullet.Template // bullet emitter templates (+0x2c4, bullet_ops.go)
	tplOff   [16][3]float32      // their fire offsets from the enemy (+0x1344)
	fireR    float32             // ins_346's squared no-fire radius (+0x1454)
	markers  []lifeMarker

	slots   [8]interruptSlot // boss phase interrupts (interrupt.go)
	phase   eclTimer         // phase timer, reset by ins_333 and each interrupt
	invuln  eclTimer         // ins_335 invulnerability timer (+0x2458)
	bossIdx int              // the boss slot ins_332 put this enemy in
}

type eclRect struct{ x, y, w, h float64 }

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
			t.owner.AnmSlot = f
		}
	case opAnmSetSlot, opAnmSetMain:
		vm.anmSetSlot(t, in)
	case opAnmPlay, opAnmPlayAbs:
		if t.owner != nil && len(in.Params) >= 2 {
			t.owner.Hidden = false
			vm.applyEnemyAnimation(t.owner, int(vm.resolveInt(t, in.Params[1])))
		}
	case opAnmSetDir5: // the direction sprites (dirScript+5); visuals not modelled
		if t.owner != nil && len(in.Params) >= 1 {
			a := vm.actorFor(t.owner)
			if vm.resolveInt(t, in.Params[0]) == 0 {
				a.mv.size = vm.scriptSize(a.anmFile, a.dirScript+5)
			}
			t.owner.Hidden = false
		}

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
		}
	case opFlagClear:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.ECLFlags &^= int(vm.resolveInt(t, in.Params[0]))
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
	case opDropClear:
		if t.owner != nil {
			t.owner.DropPower, t.owner.DropPoint, t.owner.DropLife = 0, 0, 0
			t.owner.DropBigPower, t.owner.DropBigPoint = 0, 0
			t.owner.DropAreaW, t.owner.DropAreaH = 0, 0
			vm.actorFor(t.owner).dropArea = eclRect{}
		}
	case opDropExtra: // SS: (type, count)
		if t.owner != nil && len(in.Params) >= 2 {
			typ := int(vm.resolveInt(t, in.Params[0]))
			n := int(vm.resolveInt(t, in.Params[1]))
			vm.addDrop(t.owner, typ, n)
		}
	case opDropArea: // ff: (w, h), h read first
		if t.owner != nil && len(in.Params) >= 2 {
			a := vm.actorFor(t.owner)
			h := vm.resolveFloat(t, in.Params[1])
			w := vm.resolveFloat(t, in.Params[0])
			a.dropArea.w, a.dropArea.h = math.Abs(float64(w)), math.Abs(float64(h))
			t.owner.DropAreaW, t.owner.DropAreaH = a.dropArea.w, a.dropArea.h
		}
	case opDropMain:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.DropPower = maxInt(0, int(vm.resolveInt(t, in.Params[0])))
		}
	case opDropItems:
		vm.spawnDrops(ctx, t.owner)
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
	case opDialogRead: // 338 S: select & arm the dialog the next 339 will start
		if len(in.Params) >= 1 {
			vm.dialogID = vm.resolveInt(t, in.Params[0])
			vm.dialogArmed = true
		}
	case opDialogWait: // 339: a 338-armed 339 runs a dialog & blocks until dismissed.
		// A bare 339 (no preceding 338) is a spell-name/banner cue → not a dialog.
		switch {
		case vm.dialogFiring && vm.DialogActive != nil && vm.DialogActive():
			t.blocked = true // overlay still open: keep blocking
		case vm.dialogFiring:
			vm.dialogFiring = false // overlay closed: this event is done
		case vm.dialogArmed && vm.DialogFunc != nil && vm.DialogFunc(vm.dialogID):
			vm.dialogArmed = false
			vm.dialogFiring = true
			t.blocked = true // opened this frame: block until it closes
		default:
			vm.dialogArmed = false // bare 339 or no bridge: nothing to do
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
	case opSpellEnd: // case 0x157 leaves the bullets alone
		vm.spell.active = false // 符卡结束：撤下 HUD
	case opSpellBg, opSpell, opSpell3: // 342/357/359 SSSx: (idx, timeLimit, bonus, name)
		vm.declareSpell(t, in)
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
	case 346: // case 0x15a: 401 holds fire while the player is within r
		if t.owner != nil && len(in.Params) >= 1 {
			r := vm.resolveFloat(t, in.Params[0])
			vm.actorFor(t.owner).fireR = float32(r * r)
		}
	default:
		// Remaining no-ops with known-but-cosmetic binary effects: 272/273 spawn
		// ANM effect objects at the enemy (case 0x110/0x111), 344/345 chapter
		// bookkeeping (345 is FUN_0040e6a0), 360 stars, 361 (FUN_00405410),
		// 362 HUD flag |8, 363 (FUN_0040cea0), 365 (FUN_0041c7d0), 366 dialog
		// sprite-swap flags. UI/spell-frame ops likewise.
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
		// The stage's 3D offset (DAT_00491d7c) is taken as 0.
		pos[2] = vm.resolveFloat(t, in.Params[3])
	} else if (op == opEnmCreate || op == opEnmCreateM) && parent != nil {
		pos[0] += parent.mv.final.pos[0]
		pos[1] += parent.mv.final.pos[1]
	}
	hp := vm.resolveInt(t, in.Params[n-3])
	vm.resolveInt(t, in.Params[n-2]) // score (+0x23f8)
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
	e.DropPower = int(drop)
	e.DropPoint = 1
	if typ != enemy.TypeFairy {
		e.SpriteID = 1
	}
	e.Damaged = vm.damaged
	e.Hidden = true // until an ANM op gives it a sprite
	vm.applySpawnVisual(e, name)
	if op == opEnmCreateM || op == opEnmCreateAM {
		e.ECLFlags |= 0x800
	}
	if op == opEnmCreate3D {
		e.ECLFlags |= 0x40000
	}

	a := newActor(e)
	a.mv.layer[0].pos = pos
	if parent != nil {
		a.vars = parent.vars
	}
	a.invuln.set(2)
	a.main = &eclTask{sub: vm.prog.SubByName[name], owner: e, asyncID: -1}
	if a.main.sub != nil {
		a.main.frame = newFrame(a.main.sub, nil)
	}
	vm.actors[e] = a
	vm.updateActor(a)
	vm.order = append(vm.order, a)
	if ctx != nil {
		ctx.AddEnemy(e)
	}
}

// anmSetSlot is 259/262 (cases 0x103/0x106): play script in an ANM slot of
// the ins_258 file. Slot 0 is the body: its sprite sizes the offscreen test,
// and 262 also makes the script the base 269 offsets.
func (vm *VM) anmSetSlot(t *eclTask, in *Instr) {
	if t.owner == nil || len(in.Params) < 2 {
		return
	}
	a := vm.actorFor(t.owner)
	slot := int(vm.resolveInt(t, in.Params[0]))
	script := int(vm.resolveInt(t, in.Params[1]))
	if in.Opcode == opAnmSetSlot && script < 0 {
		return
	}
	if slot == 0 {
		a.curScript = script
		a.mv.size = vm.scriptSize(a.anmFile, script)
		if in.Opcode == opAnmSetMain {
			t.owner.ECLFlags |= 0x1000
			a.dirScript = script
			t.owner.MainScript = script
		}
	}
	t.owner.Hidden = false
	vm.applyEnemyVisual(t.owner, script)
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

// scriptSize is the slot-0 extent (+0x13a4/+0x13a8) for script in ANM file
// index file (0 = bullet.anm, i = the ECL's i-th ANM): the sprite and scale
// the script sets at time 0 (ANM ops 3 and 50), as (h*sy, w*sx). Zero if
// the file is not embedded.
func (vm *VM) scriptSize(file, script int) [2]float32 {
	name := "bullet.anm"
	if file > 0 {
		if file > len(vm.prog.AnimNames) {
			return [2]float32{}
		}
		name = vm.prog.AnimNames[file-1]
	}
	f := loadANM(name)
	if f == nil || f.Scripts[script] == nil {
		return [2]float32{}
	}
	var w, h float32
	sx, sy := float32(1), float32(1)
	for _, ins := range f.Scripts[script].Instrs {
		if ins.Time > 0 {
			break
		}
		switch ins.Op {
		case 3:
			if sp, ok := f.Sprites[int(ins.Arg(0))]; ok {
				w, h = sp.W, sp.H
			}
		case 50:
			sx = math.Float32frombits(uint32(ins.Arg(0)))
			sy = math.Float32frombits(uint32(ins.Arg(1)))
		}
	}
	return [2]float32{float32(h * sy), float32(w * sx)}
}

func (vm *VM) applyEnemyVisual(e *enemy.Enemy, id int) {
	if e.Type != enemy.TypeFairy {
		if id >= 0 && id <= 7 {
			e.SpriteID = maxInt(1, id)
		}
		return
	}
	switch id {
	case 0:
		e.FairyKind = 0
	case 40:
		e.FairyKind = 5
	case 45:
		e.FairyKind = 2
	case 46:
		e.FairyKind = 1
	case 47:
		e.FairyKind = 3
	case 48:
		e.FairyKind = 4
	default:
		if id >= 0 && id < 8 {
			e.FairyKind = id
		}
	}
}

func (vm *VM) applyEnemyAnimation(e *enemy.Enemy, id int) {
	if e == nil {
		return
	}
	if e.Type == enemy.TypeFairy {
		vm.applyEnemyVisual(e, id)
		return
	}
	if id >= 0 && id <= 7 {
		e.SpriteID = maxInt(1, id)
	}
}

func (vm *VM) applySpawnVisual(e *enemy.Enemy, name string) {
	if e.Type != enemy.TypeFairy {
		return
	}
	switch {
	case strings.Contains(name, "RGirl"):
		e.FairyKind = 1
	case strings.Contains(name, "GGirl"):
		e.FairyKind = 3
	case strings.Contains(name, "YGirl"):
		e.FairyKind = 4
	case strings.Contains(name, "BGirl"):
		e.FairyKind = 2
	default:
		e.FairyKind = 0
	}
}

func (vm *VM) addDrop(e *enemy.Enemy, dropType, count int) {
	if e == nil || count <= 0 {
		return
	}
	switch dropType {
	case 1:
		e.DropPower += count
	case 2:
		e.DropPoint += count
	case 4:
		e.DropBigPower += count
	case 5:
		e.DropBigPoint += count
	case 7:
		e.DropLife += count
	default:
		e.DropPoint += count
	}
}

func (vm *VM) spawnDrops(ctx *stage.Context, e *enemy.Enemy) {
	if ctx == nil || ctx.Items == nil || e == nil {
		return
	}
	a := vm.actors[e]
	w, h := 0.0, 0.0
	if a != nil {
		w, h = a.dropArea.w, a.dropArea.h
	}
	spawn := func(n int, it item.ItemType) {
		for i := 0; i < n; i++ {
			x, y := e.X, e.Y
			if w > 0 || h > 0 {
				x += (float64(vm.randFloat()) - 0.5) * w
				y += (float64(vm.randFloat()) - 0.5) * h
			}
			ctx.Items.Spawn(x, y, it)
		}
	}
	spawn(e.DropPower, item.PowerItem)
	spawn(e.DropPoint, item.PointItem)
	spawn(e.DropLife, item.LifeFragment)
	spawn(e.DropBigPower, item.BigPower)
	spawn(e.DropBigPoint, item.BigPoint)
	e.DropPower, e.DropPoint, e.DropLife = 0, 0, 0
	e.DropBigPower, e.DropBigPoint = 0, 0
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

// declareSpell handles 342/357/359: (idx, timeLimit, bonus, name). It records
// the boss spell card for the HUD; the time limit is the interrupt slot's.
func (vm *VM) declareSpell(t *eclTask, in *Instr) {
	if len(in.Params) < 4 {
		return
	}
	vm.resolveInt(t, in.Params[0])
	bonus := vm.resolveInt(t, in.Params[2])
	vm.resolveInt(t, in.Params[1])
	name := in.Params[3].Str
	if utf, err := spellSJISDec.String(name); err == nil && utf != "" {
		name = utf
	}
	vm.spell = spellState{
		active: true,
		name:   name,
		bonus:  int64(bonus),
		owner:  t.owner,
	}
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
