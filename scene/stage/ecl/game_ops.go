package ecl

// Game opcodes 256-368: enemy lifecycle, visuals, drops, and the two-layer
// movement system, ported from FUN_0040e770. Bullet/laser opcodes (400-436)
// live in bullet_ops.go.

import (
	"math"
	"strings"

	"golang.org/x/text/encoding/japanese"

	"th10/audio"
	"th10/entity/enemy"
	"th10/entity/item"
	"th10/scene/stage"
)

// eclActor holds the ECL movement/visual state for one enemy. Final position is
// the sum of two independent movement layers (enemy+0x58 / +0x84 in the binary);
// most enemies only use layer A.
type eclActor struct {
	layers    [2]moveLayer
	moveLimit eclRect
	hasLimit  bool
	entered   bool // has been inside the field once (FUN_0040dc80 flag 0x100)
	dropArea  eclRect
	tpl       [16]eclTemplate // bullet emitter templates (bullet_ops.go)
	markers   []lifeMarker
}

type eclRect struct{ x, y, w, h float64 }

type lifeMarker struct {
	index int
	hp    float64
	raw   int32
}

type moveLayer struct {
	x, y     float64 // position contribution, ECL coords (center/top relative)
	speed    float64
	angle    float64
	accel    float64 // mode-7 continuous speed acceleration
	angleVel float64 // continuous heading rotation (moveCircle 288/290), rad/frame
	flip     bool    // mirror X movement (enemy spawned by a "flip" create opcode)
	pos      *posTween
	vel      *velTween
}

type posTween struct {
	sx, sy, tx, ty float64
	t, total       int
	mode           int
}

type velTween struct {
	sa, ss, ta, ts float64
	t, total       int
	mode           int
}

func (vm *VM) actorFor(e *enemy.Enemy) *eclActor {
	a := vm.actors[e]
	if a == nil {
		a = &eclActor{}
		vm.actors[e] = a
	}
	return a
}

// integrate advances both movement layers (tweens advance by `ticks` script
// ticks; velocity integrates every real frame) and writes the enemy's world
// position. With moveLimit active (flag 0x200) the final position is clamped
// into the box each frame (FUN_0040dc80), the correction going into layer A.
func (a *eclActor) integrate(e *enemy.Enemy, ticks int) {
	for i := range a.layers {
		a.layers[i].step(ticks)
	}
	sx := a.layers[0].x + a.layers[1].x
	sy := a.layers[0].y + a.layers[1].y
	if a.hasLimit {
		hw, hh := math.Abs(a.moveLimit.w)*0.5, math.Abs(a.moveLimit.h)*0.5
		nx := clampF(sx, a.moveLimit.x-hw, a.moveLimit.x+hw)
		ny := clampF(sy, a.moveLimit.y-hh, a.moveLimit.y+hh)
		a.layers[0].x += nx - sx
		a.layers[0].y += ny - sy
		sx, sy = nx, ny
	}
	e.X, e.Y = eclToWorld(sx, sy)

	// Offscreen despawn (FUN_0040dc80): the ECL field is x in [-192,192],
	// y in [0,448] (DAT_00470b3c/b40/b04/b38). Once an enemy has entered,
	// leaving by more than its half sprite size deletes it silently, unless
	// ECL flag bit2 (0x4, set via ins_322) keeps it alive. Bosses never
	// despawn this way (they sit inside a moveLimit anyway).
	const half = 16.0
	outside := sx+half < -192 || sx-half > 192 || sy+half < 0 || sy-half > 448
	if !outside {
		a.entered = true
	} else if a.entered && e.ECLFlags&0x4 == 0 && e.Type == enemy.TypeFairy {
		e.Active = false
	}
}

func (l *moveLayer) step(ticks int) {
	for k := 0; k < ticks; k++ {
		if l.pos != nil {
			l.pos.t++
			f := easeFrac(l.pos.t, l.pos.total, l.pos.mode)
			l.x = lerp(l.pos.sx, l.pos.tx, f)
			l.y = lerp(l.pos.sy, l.pos.ty, f)
			if l.pos.t >= l.pos.total {
				l.x, l.y = l.pos.tx, l.pos.ty
				l.pos = nil
			}
			continue
		}
		if l.vel != nil {
			l.vel.t++
			f := easeFrac(l.vel.t, l.vel.total, l.vel.mode)
			l.angle = l.vel.sa + angDelta(l.vel.sa, l.vel.ta)*f
			l.speed = lerp(l.vel.ss, l.vel.ts, f)
			if l.vel.t >= l.vel.total {
				l.angle, l.speed = l.vel.ta, l.vel.ts
				l.vel = nil
			}
		}
		if l.accel != 0 {
			l.speed += l.accel
		}
	}
	if l.pos != nil {
		return // a position tween owns the position; no velocity integration
	}
	if l.angleVel != 0 {
		l.angle += l.angleVel // moveCircle: heading rotates → curved path
	}
	dx := math.Cos(l.angle) * l.speed
	if l.flip {
		dx = -dx // mirrored enemy: negate X displacement (angle -> pi-angle)
	}
	l.x += dx
	l.y += math.Sin(l.angle) * l.speed
}

func (l *moveLayer) setVel(angle, speed float64) {
	l.angle, l.speed, l.accel, l.angleVel = angle, speed, 0, 0
	l.pos, l.vel = nil, nil
}

// execGameOp dispatches opcodes >= 256.
func (vm *VM) execGameOp(ctx *stage.Context, t *eclTask, in *Instr) {
	if in.Opcode >= 400 {
		vm.execBulletOp(ctx, t, in)
		return
	}
	switch in.Opcode {
	case opEnmCreate, opEnmCreateM, opEnmCreateAbs, opEnmCreateAbsM, 265, 266, 267, 268, 270, 271:
		vm.spawn(ctx, t, in)
	case opAnmSelect:
		// Selects which loaded ANM file subsequent anm ops target (enemy->0xe8).
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.AnmSlot = int(vm.resolveInt(t, in.Params[0]))
		}
	case opAnmSetMain:
		// anmSetMain(layer, scriptID): play an ANM script on a layer. Layer 0 is
		// the main body — record its enemy.anm script id for faithful rendering.
		if t.owner != nil && len(in.Params) >= 2 {
			layer := int(vm.resolveInt(t, in.Params[0]))
			scriptID := int(vm.resolveInt(t, in.Params[1]))
			if layer == 0 {
				t.owner.MainScript = scriptID
			}
			vm.applyEnemyVisual(t.owner, scriptID)
		}
	case opAnmSetSprite:
		if t.owner != nil && len(in.Params) >= 2 {
			vm.applyEnemyVisual(t.owner, int(vm.resolveInt(t, in.Params[1])))
		}
	case opAnmPlay, opAnmPlayAbs:
		if t.owner != nil && len(in.Params) >= 2 {
			vm.applyEnemyAnimation(t.owner, int(vm.resolveInt(t, in.Params[1])))
		}
	case opAnmSelectPlay:
		if t.owner != nil && len(in.Params) >= 1 {
			vm.applyEnemyAnimation(t.owner, int(vm.resolveInt(t, in.Params[0])))
		}

	// --- movement ---
	case opSetPos, opSetPosB:
		vm.opSetPos(t, in, layerOf(in.Opcode, opSetPos))
	case opMovePosTime, opMovePosTimeB:
		vm.opMovePosTime(t, in, layerOf(in.Opcode, opMovePosTime))
	case opSetVel, opSetVelB:
		vm.opSetVel(t, in, layerOf(in.Opcode, opSetVel))
	case opMoveVelTime, opMoveVelTimeB:
		vm.opMoveVelTime(t, in, layerOf(in.Opcode, opMoveVelTime))
	case opMoveRand, opMoveRandB:
		vm.opMoveRand(t, in, layerOf(in.Opcode, opMoveRand))
	case opMoveAdd, opMoveAddB:
		if t.owner != nil && len(in.Params) >= 2 {
			a := vm.actorFor(t.owner)
			l := &a.layers[layerOf(in.Opcode, opMoveAdd)]
			dx := float64(vm.resolveFloat(t, in.Params[0]))
			if l.flip {
				dx = -dx
			}
			l.x += dx
			l.y += float64(vm.resolveFloat(t, in.Params[1]))
		}
	case opMoveCircle, opMoveCircleB:
		vm.opMoveCircle(t, in, layerOf(in.Opcode, opMoveCircle))
	case opMoveStop: // 294: halt all motion on layer A (keep position)
		if t.owner != nil {
			l := &vm.actorFor(t.owner).layers[0]
			l.speed, l.accel, l.angleVel = 0, 0, 0
			l.pos, l.vel = nil, nil
		}
	case opMoveVel299: // 299 ff: set layer-B heading/speed instantly (B-form of 297/298 family)
		if t.owner != nil && len(in.Params) >= 2 {
			l := &vm.actorFor(t.owner).layers[1]
			angle := float64(vm.resolveFloat(t, in.Params[0]))
			speed := float64(vm.resolveFloat(t, in.Params[1]))
			if !eclUnset(angle) {
				l.angle = angle
			}
			if !eclUnset(speed) {
				l.speed = speed
			}
			l.accel, l.angleVel = 0, 0
			l.pos, l.vel = nil, nil
		}
	case opMoveLimit:
		if t.owner != nil && len(in.Params) >= 4 {
			a := vm.actorFor(t.owner)
			a.moveLimit = eclRect{
				x: float64(vm.resolveFloat(t, in.Params[0])),
				y: float64(vm.resolveFloat(t, in.Params[1])),
				w: float64(vm.resolveFloat(t, in.Params[2])),
				h: float64(vm.resolveFloat(t, in.Params[3])),
			}
			a.hasLimit = true
		}
	case opMoveLimitR:
		if t.owner != nil {
			vm.actorFor(t.owner).hasLimit = false
		}

	// --- enemy state ---
	case opSetHurtbox:
		if t.owner != nil && len(in.Params) >= 2 {
			t.owner.HitWidth = math.Abs(float64(vm.resolveFloat(t, in.Params[0])))
			t.owner.HitHeight = math.Abs(float64(vm.resolveFloat(t, in.Params[1])))
		}
	case opSetHitbox:
		if t.owner != nil && len(in.Params) >= 2 {
			t.owner.BodyWidth = math.Abs(float64(vm.resolveFloat(t, in.Params[0])))
			t.owner.BodyHeight = math.Abs(float64(vm.resolveFloat(t, in.Params[1])))
		}
	case opFlagSet:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.ECLFlags |= int(vm.resolveInt(t, in.Params[0]))
		}
	case opFlagClear:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.ECLFlags &^= int(vm.resolveInt(t, in.Params[0]))
		}
	case opSetHP:
		if t.owner != nil && len(in.Params) >= 1 {
			if hp := int(vm.resolveInt(t, in.Params[0])); hp > 0 {
				t.owner.HP, t.owner.MaxHP = hp, hp
				vm.actorFor(t.owner).markers = nil
			}
		}
	case opSetBoss:
		// Boss/MidBoss type is decided at spawn; nothing extra needed here.
	case opTimerReset:
		if t.owner != nil {
			t.owner.Age = 0
		}
	case opSetInvuln:
		if t.owner != nil && len(in.Params) >= 1 {
			n := int(vm.resolveInt(t, in.Params[0]))
			if n > t.owner.InvulnFrames {
				t.owner.InvulnFrames = n
			}
		}
	case opDropClear:
		if t.owner != nil {
			t.owner.DropPower, t.owner.DropPoint, t.owner.DropLife, t.owner.DropBomb = 0, 0, 0, 0
			t.owner.DropAreaW, t.owner.DropAreaH = 0, 0
			vm.actorFor(t.owner).dropArea = eclRect{}
		}
	case opDropExtra:
		if t.owner != nil && len(in.Params) >= 2 {
			vm.addDrop(t.owner, int(vm.resolveInt(t, in.Params[0])), int(vm.resolveInt(t, in.Params[1])))
		}
	case opDropArea:
		if t.owner != nil && len(in.Params) >= 2 {
			a := vm.actorFor(t.owner)
			a.dropArea.w = math.Abs(float64(vm.resolveFloat(t, in.Params[0])))
			a.dropArea.h = math.Abs(float64(vm.resolveFloat(t, in.Params[1])))
			t.owner.DropAreaW, t.owner.DropAreaH = a.dropArea.w, a.dropArea.h
		}
	case opDropMain:
		if t.owner != nil && len(in.Params) >= 1 {
			t.owner.DropPower = maxInt(0, int(vm.resolveInt(t, in.Params[0])))
		}
	case opDropItems:
		vm.spawnDrops(ctx, t.owner)
	case opDelayedCall:
		if t.owner != nil && len(in.Params) >= 4 {
			delay := int(vm.resolveInt(t, in.Params[1]))
			if delay <= 0 {
				delay = int(vm.resolveInt(t, in.Params[2]))
			}
			name := in.Params[3].Str
			if name != "" {
				if nt := vm.startTask(name, t.owner, nil, -1); nt != nil && delay > 0 {
					nt.wait = delay
				}
			}
		}
	case opPlaySound:
		if len(in.Params) >= 1 {
			vm.playStageSound(ctx, int(vm.resolveInt(t, in.Params[0])))
		}
	case opScreenShake: // 337 SSS: (duration, magnitude, _) — screen shake
		if len(in.Params) >= 2 {
			dur := int(vm.resolveInt(t, in.Params[0]))
			mag := float64(vm.resolveInt(t, in.Params[1]))
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
			t.wait = 1
			t.pc-- // overlay still open: keep blocking
		case vm.dialogFiring:
			vm.dialogFiring = false // overlay closed: this event is done
		case vm.dialogArmed && vm.DialogFunc != nil && vm.DialogFunc(vm.dialogID):
			vm.dialogArmed = false
			vm.dialogFiring = true
			t.wait = 1
			t.pc-- // opened this frame: block until it closes
		default:
			vm.dialogArmed = false // bare 339 or no bridge: nothing to do
		}
	case opDeathWait:
		if vm.hasBlockingEnemies(ctx, t.owner) {
			t.wait = 1
			// re-run this instruction next time it wakes
			t.pc--
		}
	case opSetTimeout: // 341 Sm (case 0x155): register the timeout escape sub
		if len(in.Params) >= 2 && in.Params[1].Str != "" {
			vm.escapeSub = in.Params[1].Str
			vm.escapeOwner = t.owner
		}
	case opSpellEnd:
		if ctx != nil && ctx.Bullets != nil {
			ctx.Bullets.ClearAll()
		}
		vm.spell.active = false // 符卡结束：撤下 HUD
		vm.escapeSub = ""       // 阶段被打掉：注册的超时逃跑不再触发
	case opSpellBg, opSpell, opSpell3: // 342/357/359 SSSx: (idx, timeLimit, bonus, name)
		vm.declareSpell(t, in)
	case opLifeMarker:
		vm.addLifeMarker(t, in)
	case opRankPickI:
		if len(in.Params) >= 5 {
			vm.storeVar(t, in.Params[0], intVal(vm.rankPickInt(in.Params[1:])))
		}
	case opRankPickF:
		if len(in.Params) >= 5 {
			vm.storeVar(t, in.Params[0], floatVal(vm.rankPickFloat(t, in.Params[1:])))
		}
	case opWaitRank: // 368 SSSS: difficulty-indexed wait (Easy,Normal,Hard,Lunatic)
		if len(in.Params) > 0 {
			n := int(vm.resolveInt(t, in.Params[vm.rankIndex(len(in.Params))]))
			if n > 0 {
				t.wait = n
			}
		}
	case 364: // case 0x16c: set/clear enemy flag bit 0x80000
		if t.owner != nil && len(in.Params) >= 1 {
			if vm.resolveInt(t, in.Params[0]) != 0 {
				t.owner.ECLFlags |= 0x80000
			} else {
				t.owner.ECLFlags &^= 0x80000
			}
		}
	case 367: // case 0x16f: set the global Timer time scale (DAT_00476f78)
		if len(in.Params) >= 1 {
			if s := float64(vm.resolveFloat(t, in.Params[0])); s > 0 {
				vm.timeScale = s
			}
		}
	default:
		// Remaining no-ops with known-but-cosmetic binary effects: 272/273 spawn
		// ANM effect objects at the enemy (case 0x110/0x111), 344/345 chapter
		// bookkeeping, 346 (FUN_0040e6a0), 360 stars, 361 (FUN_00405410),
		// 362 HUD flag |8, 363 (FUN_0040cea0), 365 (FUN_0041c7d0), 366 dialog
		// sprite-swap flags. UI/spell-frame ops likewise.
	}
}

// layerOf returns 0 for the base opcode, 1 for its "+2" B-layer variant.
func layerOf(op, base uint16) int {
	if op == base {
		return 0
	}
	return 1
}

func (vm *VM) opSetPos(t *eclTask, in *Instr, layer int) {
	if t.owner == nil || len(in.Params) < 2 {
		return
	}
	l := &vm.actorFor(t.owner).layers[layer]
	if x := vm.resolveFloat(t, in.Params[0]); !eclUnset(float64(x)) {
		l.x = float64(x)
	}
	if y := vm.resolveFloat(t, in.Params[1]); !eclUnset(float64(y)) {
		l.y = float64(y)
	}
	l.pos, l.vel = nil, nil
}

func (vm *VM) opMovePosTime(t *eclTask, in *Instr, layer int) {
	if t.owner == nil || len(in.Params) < 4 {
		return
	}
	l := &vm.actorFor(t.owner).layers[layer]
	frames := int(vm.resolveInt(t, in.Params[0]))
	mode := int(vm.resolveInt(t, in.Params[1]))
	tx := float64(vm.resolveFloat(t, in.Params[2]))
	ty := float64(vm.resolveFloat(t, in.Params[3]))
	// Sentinel targets keep the current coordinate (FUN case 0x119).
	if eclUnset(tx) {
		tx = l.x
	}
	if eclUnset(ty) {
		ty = l.y
	}
	if frames <= 0 {
		l.x, l.y = tx, ty
		l.pos = nil
		return
	}
	if l.flip {
		tx = 2*l.x - tx // mirror the X motion around the move's start
	}
	l.vel = nil
	l.pos = &posTween{sx: l.x, sy: l.y, tx: tx, ty: ty, total: frames, mode: mode}
}

func (vm *VM) opSetVel(t *eclTask, in *Instr, layer int) {
	if t.owner == nil || len(in.Params) < 2 {
		return
	}
	l := &vm.actorFor(t.owner).layers[layer]
	angle := float64(vm.resolveFloat(t, in.Params[0]))
	speed := float64(vm.resolveFloat(t, in.Params[1]))
	l.setVel(angle, speed)
}

// opMoveCircle handles 288/290 (moveCircle): the heading rotates by a fixed
// step each frame, tracing a circular/spiral path at the given speed. thtk
// signature ffff = (angle, angleStep, _, speed); sentinel args keep the
// previous value (matching the 285/movePos "keep" convention).
func (vm *VM) opMoveCircle(t *eclTask, in *Instr, layer int) {
	if t.owner == nil || len(in.Params) < 4 {
		return
	}
	l := &vm.actorFor(t.owner).layers[layer]
	if a := float64(vm.resolveFloat(t, in.Params[0])); !eclUnset(a) {
		l.angle = a
	}
	if step := float64(vm.resolveFloat(t, in.Params[1])); !eclUnset(step) {
		l.angleVel = step
	}
	if sp := float64(vm.resolveFloat(t, in.Params[3])); !eclUnset(sp) {
		l.speed = sp
	}
	l.accel = 0
	l.pos, l.vel = nil, nil
}

func (vm *VM) opMoveVelTime(t *eclTask, in *Instr, layer int) {
	if t.owner == nil || len(in.Params) < 4 {
		return
	}
	l := &vm.actorFor(t.owner).layers[layer]
	frames := int(vm.resolveInt(t, in.Params[0]))
	mode := int(vm.resolveInt(t, in.Params[1]))
	angle := float64(vm.resolveFloat(t, in.Params[2]))
	speed := float64(vm.resolveFloat(t, in.Params[3]))
	if mode == 7 { // continuous acceleration: both args are per-frame deltas
		if eclUnset(speed) { // sentinel -> 0 in mode 7 (FUN case 0x11d)
			speed = 0
		}
		if eclUnset(angle) {
			angle = 0
		}
		l.accel = speed
		l.angleVel = angle
		l.vel = nil
		return
	}
	if eclUnset(angle) {
		angle = l.angle
	}
	if eclUnset(speed) {
		speed = l.speed
	}
	if frames <= 0 {
		l.setVel(angle, speed)
		return
	}
	l.pos = nil
	l.vel = &velTween{sa: l.angle, ss: l.speed, ta: angle, ts: speed, total: frames, mode: mode}
}

// opMoveRand ports FUN_0040e770 case 0x124 exactly: the heading comes from one
// ZUN-RNG draw r in [-1,1), shaped by where the enemy sits in the moveLimit
// box and which side the player is on, then the speed tweens from the given
// value down to 0 over `frames` (so the boss hops a bounded distance and
// halts). X bands are ±w/4 around the box center: inside the band the fan is
// a quarter circle opening toward the player's side; past the right margin it
// opens left (r*pi/3 + pi); past the left margin it opens right (r*pi/3).
// Outside the ±h/4 Y band the vertical sign is forced back toward the band.
func (vm *VM) opMoveRand(t *eclTask, in *Instr, layer int) {
	if t.owner == nil || len(in.Params) < 3 {
		return
	}
	a := vm.actorFor(t.owner)
	l := &a.layers[layer]
	frames := int(vm.resolveInt(t, in.Params[0]))
	mode := int(vm.resolveInt(t, in.Params[1]))
	speed := float64(vm.resolveFloat(t, in.Params[2]))

	fx := a.layers[0].x + a.layers[1].x
	fy := a.layers[0].y + a.layers[1].y
	r := vm.randSigned()
	var angle float64
	qw := math.Abs(a.moveLimit.w) * 0.25
	switch {
	case fx < a.moveLimit.x-qw: // past the left margin: fan to the right
		angle = r * (math.Pi / 3)
	case fx > a.moveLimit.x+qw: // past the right margin: fan to the left
		angle = float64(normAngle(float32(r*(math.Pi/3) + math.Pi)))
	case fx < vm.playerX-eclCenterX(): // player to the right: quarter fan right
		angle = r * (math.Pi / 2)
	default: // player to the left: quarter fan left
		angle = float64(normAngle(float32(r*(math.Pi/2) + math.Pi)))
	}
	qh := math.Abs(a.moveLimit.h) * 0.25
	switch {
	case fy < a.moveLimit.y-qh: // above the band: head down
		angle = math.Abs(angle)
	case fy > a.moveLimit.y+qh: // below the band: head up
		angle = -math.Abs(angle)
	}

	if frames <= 0 {
		l.setVel(angle, 0)
		return
	}
	l.pos = nil
	l.vel = &velTween{sa: angle, ss: speed, ta: angle, ts: 0, total: frames, mode: mode}
}

func (vm *VM) spawn(ctx *stage.Context, t *eclTask, in *Instr) {
	if ctx == nil || len(in.Params) < 6 {
		return
	}
	name := in.Params[0].Str
	x := float64(vm.resolveFloat(t, in.Params[1]))
	y := float64(vm.resolveFloat(t, in.Params[2]))

	// 270/271 use the mffSSSS layout (an extra arg before hp): an initial
	// heading/velocity at param[3] shifts hp→[4], score→[5], drop→[6].
	hpIdx, dropIdx := 3, 5
	extended := in.Opcode == 270 || in.Opcode == 271
	if extended && len(in.Params) >= 7 {
		hpIdx, dropIdx = 4, 6
	}
	hp := int(vm.resolveInt(t, in.Params[hpIdx]))
	drop := int(vm.resolveInt(t, in.Params[dropIdx]))

	// Create-opcode semantics from FUN_0040e770 (cases 0x100/0x101/0x104/0x105
	// and the conditional 0x109-0x10c that jump to the same labels):
	//   flip (local_78=1): 260,261,267,268,271 -> enemy mirrors its X movement.
	//   absolute position:  257,261,266,268 ; relative-to-parent: 256,260,265,267.
	flip := in.Opcode == 260 || in.Opcode == 261 || in.Opcode == 267 || in.Opcode == 268 || in.Opcode == 271
	absolute := in.Opcode == 257 || in.Opcode == 261 || in.Opcode == 266 || in.Opcode == 268
	if !absolute && t.owner != nil {
		if pa := vm.actors[t.owner]; pa != nil {
			x += pa.layers[0].x + pa.layers[1].x
			y += pa.layers[0].y + pa.layers[1].y
		}
	}

	typ := enemy.TypeFairy
	switch {
	case strings.Contains(name, "MBoss"):
		typ = enemy.TypeMidBoss
	case strings.Contains(name, "Boss"):
		typ = enemy.TypeBoss
	}
	wx, wy := eclToWorld(x, y)
	e := enemy.New(wx, wy, hp, typ)
	e.DropPower = drop
	e.DropPoint = 1
	if typ != enemy.TypeFairy {
		e.SpriteID = 1
		e.HitWidth, e.HitHeight = 48, 48
		e.BodyWidth, e.BodyHeight = 32, 32
	}
	vm.applySpawnVisual(e, name)
	ctx.AddEnemy(e)
	a := vm.actorFor(e)
	a.layers[0].x, a.layers[0].y = x, y
	a.layers[0].flip, a.layers[1].flip = flip, flip
	vm.startTask(name, e, nil, -1)
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
		e.DropLife += count
	case 5:
		e.DropBomb += count
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
	spawn(e.DropBomb, item.BombFragment)
	e.DropPower, e.DropPoint, e.DropLife, e.DropBomb = 0, 0, 0, 0
}

func (vm *VM) playStageSound(ctx *stage.Context, id int) {
	if ctx == nil || ctx.Audio == nil {
		return
	}
	switch id {
	case 7:
		ctx.Audio.PlaySE(audio.SEEnemy)
	case 15, 16:
		ctx.Audio.PlaySE(audio.SEShoot)
	case 18:
		ctx.Audio.PlaySE(audio.SECancel)
	}
}

func (vm *VM) hasBlockingEnemies(ctx *stage.Context, owner *enemy.Enemy) bool {
	if ctx == nil {
		return false
	}
	for _, e := range ctx.Enemies {
		if e == nil || !e.Active || e == owner {
			continue
		}
		if e.Type == enemy.TypeFairy || e.Type == enemy.TypeMidBoss || e.Type == enemy.TypeBoss {
			return true
		}
	}
	return false
}

func (vm *VM) rankPickInt(args []Param) int32 {
	return vm.resolveInt(nil, args[vm.rankIndex(len(args))])
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
// the boss spell card for the HUD and resets the countdown timer.
func (vm *VM) declareSpell(t *eclTask, in *Instr) {
	if len(in.Params) < 4 {
		return
	}
	name := in.Params[3].Str
	if utf, err := spellSJISDec.String(name); err == nil && utf != "" {
		name = utf
	}
	limit := int(vm.resolveInt(t, in.Params[1]))
	vm.spell = spellState{
		active:    true,
		name:      name,
		bonus:     int64(vm.resolveInt(t, in.Params[2])),
		timeLimit: limit,
		timeLeft:  limit,
		owner:     t.owner,
	}
}

func (vm *VM) addLifeMarker(t *eclTask, in *Instr) {
	if t.owner == nil || len(in.Params) < 3 {
		return
	}
	marker := lifeMarker{
		index: int(vm.resolveInt(t, in.Params[0])),
		hp:    float64(vm.resolveFloat(t, in.Params[1])),
		raw:   vm.resolveInt(t, in.Params[2]),
	}
	a := vm.actorFor(t.owner)
	for i := range a.markers {
		if a.markers[i].index == marker.index {
			a.markers[i] = marker
			return
		}
	}
	a.markers = append(a.markers, marker)
}

// --- small math helpers ---

func easeFrac(t, total, mode int) float64 {
	if total <= 0 {
		return 1
	}
	f := float64(t) / float64(total)
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	switch mode {
	case 1: // accelerate
		return f * f
	case 2, 4: // decelerate (common "glide to a stop")
		return 1 - (1-f)*(1-f)
	case 3: // smooth
		return f * f * (3 - 2*f)
	default: // linear
		return f
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// angDelta returns the shortest signed angular distance from a to b (-pi..pi].
func angDelta(a, b float64) float64 {
	d := math.Mod(b-a, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	} else if d <= -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// eclUnset reports the "keep previous value" sentinel (large negative, e.g.
// -999999 in stage01; _DAT_00470d58 in the binary).
func eclUnset(v float64) bool { return v <= -900000 }
