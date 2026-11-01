package ecl

// TH10 ECL virtual machine — runtime over the binary IR (see loader.go).
//
// Faithful to the binary's model rather than thtk's decompiled text:
//   - Every enemy, the stage's "main" included, owns a primary context and
//     a list of async ones. The manager (FUN_0040d750) walks the enemy list
//     in creation order and updates each enemy in full (FUN_0040dc80):
//     movement (motion.go), then its contexts (FUN_0044fd10), the primary
//     first. An enemy created by an ECL op is updated once on the spot.
//   - Each context has a float clock (FUN_0044e1a0). Every instruction with
//     Time <= clock runs, then the clock advances by the game speed.
//     goto/if/unless carry a time arg that sets the clock; wait(83)
//     subtracts N from it, so what follows runs N frames later. A game op
//     that returns -1 (the 339/340 waits) ends the frame without advancing
//     the clock and runs again next frame.
//   - op 1, and op 10 with nothing to return to, end the context: an async
//     one is freed, the end of the primary one deletes the enemy.
//   - System opcodes (<256) are a typed stack machine (expr.c / FUN_0043ee30):
//     LOADI/F push, ASSIGNI/F pop->var, 50-84 arithmetic/compare/logic.
//   - Variables: param_mask bit => the 4-byte value is a var ref; id>=0 is a
//     local-frame byte offset, id==-1 pops the data stack, id<-1 is a special
//     var (FUN_0044fdb0). Float vars encode the id as a float bit pattern.
//
// Game ops (256-368) and danmaku (400-436) live in game_ops.go /
// bullet_ops.go, special variables in special.go; this file owns the core
// loop, control flow, and resolution.

import (
	"math"
	"path"
	"sort"

	"th10/entity/enemy"
	"th10/game"
	"th10/gamedata"
	"th10/scene/stage"
	"th10/zun"
)

// VM runs an ECL program for one stage.
type VM struct {
	prog   *Program
	actors map[*enemy.Enemy]*eclActor
	// order is the enemy list (DAT_00477704+0x58) in creation order.
	order []*eclActor
	// created counts the enemies made (DAT_00477704+0x64), the main first;
	// its parity picks a new enemy's death sound.
	created int32
	// main is the hidden enemy that runs the "main" sub; the stage ends with
	// it. It starts on the first Update.
	main    *eclActor
	started bool
	frame   int
	// rng is the game RNG (DAT_004918b0), shared with the bullet ANM VMs
	// through stage.Context.RNG.
	rng *zun.RNG

	// Dynamic rank (DAT_00474c98, -1024..1024): +1 every 60 frames of the
	// player's state timer while enemies are on screen and no dialog is open
	// (player update FUN_00425730). Consumed by the rank-tier ET opcodes
	// 422-427.
	rank int

	// The game speed (DAT_00476f78, set by ins_367): every context clock,
	// timer and tween advances by this much per frame.
	gameSpeed float32

	// snapshot of stage state for special-var resolution (set each frame)
	difficulty int
	playerX    float64
	playerY    float64
	hasPlayer  bool
	playerECL  [2]float32 // the player in ECL coordinates (float32)

	ctx *stage.Context // this frame's context, for the damage callback

	// Boss slots (DAT_00477704+0x10): ins_332 puts an enemy in one, ins_340
	// waits for slot 0 to empty, and the boss HUD follows slot 0.
	bossSlots [8]*enemy.Enemy
	// hudTimer is the boss timer readout (DAT_0047770c+0x9ec0), rewritten by
	// FUN_004127a0 whenever it checks an armed slot with a time limit.
	hudTimer int

	// screen shake (ins_337): magnitude decays linearly over the duration.
	shakeFrames int
	shakeTotal  int
	shakeMag    float64

	// card is the spell card state (DAT_004776f4) the bomb reads through
	// Context.Spell, driven by the declarations (342/348/357-359) and 343;
	// spell is the rest of the card for the HUD, whose timer is hudTimer.
	card  stage.SpellCard
	spell spellState
}

// spellState is the declared card's HUD details.
type spellState struct {
	name  string // UTF-8 (decoded from the ECL's Shift-JIS name)
	owner *enemy.Enemy
}

// eVal is a typed value on the data stack / in a variable (raw 32-bit bits).
type eVal struct {
	isFloat bool
	bits    uint32
}

func intVal(i int32) eVal     { return eVal{false, uint32(i)} }
func floatVal(f float32) eVal { return eVal{true, math.Float32bits(f)} }

func (v eVal) asInt() int32 {
	if v.isFloat {
		return int32(math.Float32frombits(v.bits))
	}
	return int32(v.bits)
}
func (v eVal) asFloat() float32 {
	if v.isFloat {
		return math.Float32frombits(v.bits)
	}
	return float32(int32(v.bits))
}

// eclTask is one running coroutine. Async calls spawn sibling tasks sharing the
// owner enemy; sync calls push onto the call stack.
type eclTask struct {
	sub     *Sub // nil: the sub was not found; the task ends on its next run
	pc      int
	clock   float32
	blocked bool // the last game op returned -1: rerun it next frame

	frame []uint32 // local variable slots, indexed by byte offset / 4
	stack []eVal
	calls []callFrame

	owner   *enemy.Enemy
	async   bool  // an async context; otherwise the owner's primary one
	asyncID int32 // callAsyncID slot, for targeted kill; -1 otherwise
	done    bool
}

type callFrame struct {
	sub   *Sub
	pc    int
	clock float32
	frame []uint32
}

// NewStage01VM loads the embedded binary Stage 1 ECL and starts its main sub.
func NewStage01VM() *VM { return NewStageVM("ecl/stage01.ecl") }

// NewStageVM loads an embedded binary stage ECL by asset path (e.g.
// "ecl/stage02.ecl") and starts its "main" sub. All TH10 stages share the
// identical SCPT V2 container and a "void main()" timeline entry, so the same
// loader+VM drives every stage; only the danmaku/movement data differs.
func NewStageVM(eclPath string) *VM {
	vm := &VM{
		prog:      &Program{SubByName: map[string]*Sub{}},
		actors:    map[*enemy.Enemy]*eclActor{},
		rng:       &zun.RNG{State: 0x1234},
		gameSpeed: 1,
	}
	if data, err := gamedata.ReadFile(eclPath); err == nil {
		if prog, perr := LoadProgram(data); perr == nil {
			vm.prog = prog
			// The ECLI list ("default.ecl": the Ecl_EtBreak bullet-clearing
			// enemies, et_ofs_r) shares the stage's sub namespace; th10.exe
			// looks every name up in one table (FUN_00450470).
			for _, name := range prog.EcliNames {
				if inc, err := gamedata.ReadFile(path.Join(path.Dir(eclPath), name)); err == nil {
					if incProg, err := LoadProgram(inc); err == nil {
						prog.Include(incProg)
					}
				}
			}
		}
	}
	if sub := vm.prog.SubByName["main"]; sub != nil {
		e := enemy.New(0, 0, 0, enemy.TypeFairy)
		e.Hidden = true
		e.ECLFlags = 0x10
		vm.main = newActor(e)
		vm.main.main = &eclTask{sub: sub, owner: e, asyncID: -1, frame: newFrame(sub, nil)}
		vm.created = 1
	}
	return vm
}

// MainDone reports whether the stage's "main" task has finished. The Stage 1
// main spawns the timeline, then blocks on deathWait (ins_340) until the boss
// is defeated and ends with delete() — so this is the faithful stage-clear
// signal (replacing the old fixed-frame cutoff).
func (vm *VM) MainDone() bool {
	return vm.main == nil || !vm.main.e.Active
}

// BossActive reports whether a boss-type enemy spawned by the VM is currently
// alive. Stage wrappers use this edge to switch to the boss BGM, since the ECL
// itself carries no music cue.
func (vm *VM) BossActive() bool {
	for e := range vm.actors {
		if e != nil && e.Active && e.Type == enemy.TypeBoss {
			return true
		}
	}
	return false
}

// BossHUD returns the current boss/spell HUD snapshot for the enemy in boss
// slot 0 (ins_332(0); mid-bosses use it too).
func (vm *VM) BossHUD() stage.BossHUDInfo {
	boss := vm.bossSlots[0]
	if boss == nil || !boss.Active {
		return stage.BossHUDInfo{}
	}
	frac := 0.0
	if boss.MaxHP > 0 {
		frac = float64(boss.HP) / float64(boss.MaxHP)
		if frac < 0 {
			frac = 0
		} else if frac > 1 {
			frac = 1
		}
	}
	info := stage.BossHUDInfo{Active: true, HPFrac: frac}
	if a := vm.actors[boss]; a != nil {
		if boss.MaxHP > 0 {
			for _, m := range a.markers {
				mf := m.hp / float64(boss.MaxHP)
				if mf > 0 && mf < 1 {
					info.HPMarkers = append(info.HPMarkers, mf)
				}
			}
			sort.Float64s(info.HPMarkers)
		}
		if a.timed() {
			info.TimeLeft = vm.hudTimer
		}
	}
	if vm.card.Active && vm.spell.owner == boss {
		info.Spell = true
		info.Name = vm.spell.name
		info.Bonus = vm.card.Bonus
	}
	return info
}

// sweepEnemies is FUN_0040e6a0 (ECL 338, 345): every enemy but the stage's
// main and the 0x40/0xc000 ones (0x80 ones always) is flagged 0x20000, and
// the manager deletes it when its walk reaches it. Its death animation is
// cosmetic.
func (vm *VM) sweepEnemies() {
	for _, a := range vm.order {
		if f := a.e.ECLFlags; f&0x50 == 0 && f&0xc000 == 0 || f&0x80 != 0 {
			a.e.ECLFlags |= 0x20000
		}
	}
}

// ShakeOffset returns the current screen-shake displacement (dx, dy) in pixels,
// driven by ins_337. Magnitude decays linearly to 0 over the requested
// duration; the direction varies per frame. Returns (0,0) when not shaking.
func (vm *VM) ShakeOffset() (float64, float64) {
	if vm.shakeFrames <= 0 || vm.shakeTotal <= 0 {
		return 0, 0
	}
	mag := vm.shakeMag * float64(vm.shakeFrames) / float64(vm.shakeTotal)
	// Deterministic alternating direction (no Math.random in VM): drive the
	// offset from the frame counter, scaled by the decaying magnitude.
	phase := float64(vm.frame)
	dx := math.Cos(phase*2.3) * mag
	dy := math.Sin(phase*2.9) * mag
	return dx, dy
}

// Update runs the enemy manager (FUN_0040d750) for one frame.
func (vm *VM) Update(ctx *stage.Context) {
	vm.frame++
	if vm.shakeFrames > 0 {
		vm.shakeFrames--
	}
	vm.ctx = ctx
	vm.prune()
	if ctx != nil {
		ctx.ManagerOrder = vm.ManagerOrder
		ctx.Boss0 = vm.boss0Pos
		ctx.Spell = &vm.card
		if ctx.Anm != nil {
			vm.gameSpeed = ctx.Anm.GameSpeed // the player's respawn resets it
		}
	}

	// Dynamic rank (FUN_00425730): +1 when the player's state timer lands on
	// a multiple of 60 while enemies are on screen and no dialog is open,
	// clamped to [-1024, 1024]. It lives in the game state, where a death
	// also takes from it, and carries over between stages.
	if ctx != nil && ctx.State != nil {
		vm.rank = ctx.State.Rank
	}
	clock := int32(vm.frame)
	if ctx != nil && ctx.Player != nil {
		clock = ctx.Player.StateTimer.Cur
	}
	if len(vm.actors) > 0 && (ctx == nil || !ctx.DialogActive()) && clock%60 == 0 {
		if vm.rank++; vm.rank > 1024 {
			vm.rank = 1024
		}
	}
	if ctx != nil && ctx.State != nil {
		ctx.State.Rank = vm.rank
	}

	// Drop the spell HUD when the card's owner dies.
	if vm.card.Active && vm.spell.owner != nil && !vm.spell.owner.Active {
		vm.card.Active = false
	}
	if ctx != nil {
		if ctx.State != nil {
			vm.difficulty = ctx.State.Difficulty
		}
		if ctx.Player != nil {
			vm.playerX, vm.playerY, vm.hasPlayer = ctx.Player.X, ctx.Player.Y, true
			vm.playerECL = [2]float32{
				float32(ctx.Player.X - eclCenterX()),
				float32(ctx.Player.Y - float64(game.FieldTop)),
			}
		}
	}

	// The stage's main enemy is created (and so first updated) before the
	// list runs.
	if !vm.started {
		vm.started = true
		if ctx != nil && ctx.RNG != nil {
			vm.rng = ctx.RNG
		} else if ctx != nil && ctx.State != nil && ctx.State.HasSeed {
			vm.rng.State = ctx.State.RNGSeed
		}
		if a := vm.main; a != nil {
			vm.actors[a.e] = a
			vm.updateActor(a)
			vm.order = append(vm.order, a)
		}
	}

	// The list walk takes each enemy's successor before updating it, so an
	// enemy created by the last one waits for the next frame, its 0x400
	// still set from its first update.
	for i := 0; i < len(vm.order); i++ {
		a := vm.order[i]
		last := i == len(vm.order)-1
		if a.e.Active {
			if a.e.ECLFlags&0x20000 == 0 && vm.updateActor(a) {
				a.e.ECLFlags &^= 0x400
			} else {
				a.e.Active = false
			}
		}
		if last {
			break
		}
	}
}

// ManagerOrder lists the live enemies in the enemy manager's list order
// (DAT_00477704+0x58), the hidden main first.
func (vm *VM) ManagerOrder() []*enemy.Enemy {
	out := make([]*enemy.Enemy, 0, len(vm.order))
	for _, a := range vm.order {
		if a.e.Active {
			out = append(out, a.e)
		}
	}
	return out
}

// updateActor is FUN_0040dc80, once per frame (flag 0x400): movement, the
// ECL, the damage the player deals (the hit may defeat the enemy or end its
// phase), the unconditional interrupt check (a phase that fires here starts
// next frame), then the invulnerability and phase timers. It reports false
// when the enemy is to be deleted.
func (vm *VM) updateActor(a *eclActor) bool {
	e := a.e
	if e.ECLFlags&0x400 != 0 {
		return true
	}
	e.ECLFlags |= 0x400
	var cam [3]float32
	if vm.ctx != nil && vm.ctx.STD != nil {
		cam = vm.ctx.STD.Cam.Delta
	}
	if !a.mv.step(&e.ECLFlags, vm.gameSpeed, cam) {
		return false
	}
	a.syncPos()
	vm.bombSwap(a)
	if !vm.runActor(a) {
		return false
	}
	e.ECLFlags &^= 0x2000
	if e.ECLFlags&0x11 == 0 && vm.ctx != nil && vm.ctx.Player != nil {
		if dmg := vm.ctx.EnemyDamage(e); dmg != 0 && vm.hurt(a, dmg) != hurtAlive {
			return false
		}
	}
	if name, ok := vm.checkInterrupt(e, a); ok {
		vm.interrupt(e, a, name)
	}
	if a.invuln.cur > 0 {
		a.invuln.add(-1, vm.gameSpeed)
	}
	a.phase.tick(vm.gameSpeed)
	return true
}

// bombSwap is ins_366's part of FUN_0040dc80: an enemy armed with it takes
// no damage (0x1) and shows its bomb sprites while a bomb runs, and gets its
// own back after.
func (vm *VM) bombSwap(a *eclActor) {
	e := a.e
	if e.ECLFlags&0x100000 == 0 {
		return
	}
	bomb := vm.ctx != nil && vm.ctx.Bomb != nil && vm.ctx.Bomb.Active()
	swapped := e.ECLFlags&0x200000 != 0
	switch {
	case bomb && !swapped:
		a.dirScript = a.bombScript
		e.ECLFlags |= 0x200001
	case !bomb && swapped:
		a.dirScript = a.savedScript
		e.ECLFlags &^= 0x200001
	}
}

// runActor is FUN_0044fd10: the primary context, then the async contexts
// that existed before it ran, newest first. The end of the primary context
// ends the enemy; an async one that ends is freed.
func (vm *VM) runActor(a *eclActor) bool {
	var buf [8]*eclTask
	snap := append(buf[:0], a.asyncs...)
	if a.main != nil && !vm.stepTask(vm.ctx, a.main) {
		return false
	}
	for _, t := range snap {
		if !t.done && !vm.stepTask(vm.ctx, t) {
			t.done = true
		}
	}
	n := 0
	for _, t := range a.asyncs {
		if !t.done {
			a.asyncs[n] = t
			n++
		}
	}
	clear(a.asyncs[n:])
	a.asyncs = a.asyncs[:n]
	return true
}

// stepTask runs one frame of a context using the clock model described
// above (FUN_0044e1a0). It reports false when the context has ended.
func (vm *VM) stepTask(ctx *stage.Context, t *eclTask) bool {
	if t.done {
		return false
	}
	if t.sub == nil {
		t.done = true // no instruction pointer: returns -1
		return false
	}
	diff := 1
	if ctx != nil && ctx.State != nil {
		diff = ctx.State.Difficulty
	}
	for budget := 0; budget < 100000; budget++ {
		if t.pc >= len(t.sub.Instrs) {
			if !vm.returnFromCall(t) {
				vm.end(t) // fell off the outermost sub: as op 10
				return false
			}
			continue
		}
		in := &t.sub.Instrs[t.pc]
		if float32(in.Time) > t.clock {
			break // not yet time; yield and let the clock catch up
		}
		if !in.rankAllows(diff) {
			t.pc++
			continue
		}
		if Trace != nil {
			Trace(t.sub.Name, in)
		}
		vm.execInstr(ctx, t, in)
		if t.done {
			return false
		}
		if t.blocked {
			t.blocked = false
			return true // retry this op next frame; the clock stands still
		}
	}
	t.clock += vm.gameSpeed
	return true
}

// Trace, when set, sees every instruction just before it runs. It is a
// debugging hook for tools/th10sim (TH10SIM_ETRACE).
var Trace func(sub string, in *Instr)

// end finishes t as op 1 does. Its caller frees an async context, or
// deletes the enemy for the primary one.
func (vm *VM) end(t *eclTask) {
	t.done = true
}

func (vm *VM) returnFromCall(t *eclTask) bool {
	if len(t.calls) == 0 {
		return false
	}
	last := t.calls[len(t.calls)-1]
	t.calls = t.calls[:len(t.calls)-1]
	t.sub = last.sub
	t.pc = last.pc
	t.clock = last.clock
	t.frame = last.frame
	return true
}

// execInstr dispatches one instruction. System opcodes (<256) are handled here;
// game opcodes (>=256) go to execGameOp.
func (vm *VM) execInstr(ctx *stage.Context, t *eclTask, in *Instr) {
	if in.Opcode >= 256 {
		vm.execGameOp(ctx, t, in)
		if !t.blocked {
			t.pc++
		}
		return
	}
	vm.execSystemOp(ctx, t, in)
}

// execSystemOp implements opcodes 0-93: the stack machine + control flow.
// Most ops advance pc by 1; jumps/return set pc explicitly.
func (vm *VM) execSystemOp(ctx *stage.Context, t *eclTask, in *Instr) {
	switch in.Opcode {
	case opRetBig: // 1
		vm.end(t)
	case opRetNormal: // 10: with nothing to return to, as op 1
		if !vm.returnFromCall(t) {
			vm.end(t)
		}
	case opCall: // 11 m*D
		vm.doCall(ctx, t, in, false, -1)
	case opCallAsync: // 15 m*D
		vm.doCall(ctx, t, in, true, -1)
	case opCallAsyncID: // 16 mS*D
		id := vm.resolveInt(t, in.Params[1])
		vm.doCall(ctx, t, in, true, id)
	case opKillAsyncID: // 17
		vm.killAsync(t, vm.resolveInt(t, in.Params[0]))
		t.pc++
	case opKillAllAsync: // 21
		vm.killAsync(t, -1)
		t.pc++
	case opStackAlloc: // 40: ensure frame holds this many bytes
		vm.ensureFrame(t, int(vm.resolveInt(t, in.Params[0])))
		t.pc++
	case opGoto: // 12 ot
		vm.jump(t, in, in.Params[0], in.Params[1])
	case opUnless: // 13 ot: pop; if !v jump
		if vm.popInt(t) == 0 {
			vm.jump(t, in, in.Params[0], in.Params[1])
		} else {
			t.pc++
		}
	case opIf: // 14 ot: pop; if v jump
		if vm.popInt(t) != 0 {
			vm.jump(t, in, in.Params[0], in.Params[1])
		} else {
			t.pc++
		}
	case opWait: // 83 S: the clock steps back n frames
		t.clock -= float32(vm.resolveInt(t, in.Params[0]))
		t.pc++
	case opLoadI: // 42 S
		t.push(intVal(vm.resolveInt(t, in.Params[0])))
		t.pc++
	case opLoadF: // 44 f
		t.push(floatVal(vm.resolveFloat(t, in.Params[0])))
		t.pc++
	case opAssignI: // 43 S: dest = pop
		vm.storeVar(t, in.Params[0], intVal(vm.popInt(t)))
		t.pc++
	case opAssignF: // 45 f: dest = pop
		vm.storeVar(t, in.Params[0], floatVal(vm.popFloat(t)))
		t.pc++
	case opDec: // 78 S: push old, var--
		v := vm.loadVarInt(t, in.Params[0])
		t.push(intVal(v))
		vm.storeVar(t, in.Params[0], intVal(v-1))
		t.pc++
	case opPolar: // 81 ffff: out_x, out_y = cos/sin(angle)*radius
		// FUN_0044e1a0 reads the radius first and normalises the angle
		// (FUN_0044bc70) before FUN_004501b0 takes its sine and cosine.
		radius := vm.resolveFloat(t, in.Params[3])
		angle := normAngle(vm.resolveFloat(t, in.Params[2]))
		x, y := polar(angle, radius)
		vm.storeVar(t, in.Params[0], floatVal(x))
		vm.storeVar(t, in.Params[1], floatVal(y))
		t.pc++
	case opValidRad: // 82 f: normalize angle var into (-pi, pi]
		v := vm.loadVarFloat(t, in.Params[0])
		vm.storeVar(t, in.Params[0], floatVal(normAngle(v)))
		t.pc++
	default:
		if !vm.execStackMath(t, in) {
			// Unknown system op: no-op, keep flow alive.
		}
		t.pc++
	}
}

// execStackMath handles the binary/unary expression opcodes (50-84). Returns
// false if the opcode is not a math op.
func (vm *VM) execStackMath(t *eclTask, in *Instr) bool {
	switch in.Opcode {
	case opAddI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(intVal(a + b))
	case opAddF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(floatVal(a + b))
	case opSubI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(intVal(a - b))
	case opSubF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(floatVal(a - b))
	case opMulI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(intVal(a * b))
	case opMulF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(floatVal(a * b))
	case opDivI:
		b, a := vm.popInt(t), vm.popInt(t)
		if b == 0 {
			t.push(intVal(0))
		} else {
			t.push(intVal(a / b))
		}
	case opDivF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		if b == 0 {
			t.push(floatVal(0))
		} else {
			t.push(floatVal(a / b))
		}
	case opMod:
		b, a := vm.popInt(t), vm.popInt(t)
		if b == 0 {
			t.push(intVal(0))
		} else {
			t.push(intVal(a % b))
		}
	case opEqI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a == b))
	case opEqF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(boolVal(a == b))
	case opNeI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a != b))
	case opNeF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(boolVal(a != b))
	case opLtI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a < b))
	case opLtF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(boolVal(a < b))
	case opLeI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a <= b))
	case opLeF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(boolVal(a <= b))
	case opGtI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a > b))
	case opGtF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(boolVal(a > b))
	case opGeI:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a >= b))
	case opGeF:
		b, a := vm.popFloat(t), vm.popFloat(t)
		t.push(boolVal(a >= b))
	case opNotI:
		t.push(boolVal(vm.popInt(t) == 0))
	case opNotF:
		t.push(boolVal(vm.popFloat(t) == 0))
	case opOr:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a != 0 || b != 0))
	case opAnd:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(boolVal(a != 0 && b != 0))
	case opXor:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(intVal(a ^ b))
	case opBOr:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(intVal(a | b))
	case opBAnd:
		b, a := vm.popInt(t), vm.popInt(t)
		t.push(intVal(a & b))
	case opSin:
		t.push(floatVal(float32(math.Sin(float64(vm.popFloat(t))))))
	case opCos:
		t.push(floatVal(float32(math.Cos(float64(vm.popFloat(t))))))
	case opNegI:
		t.push(intVal(-vm.popInt(t)))
	case opNegF:
		// FUN_0044e1a0 case 0x55 (0x44f812) negates the popped dword with an
		// integer neg, not fchs (an int is converted to float bits first), so
		// the result is the two's complement of the float's bits:
		// -0.032725f comes out as -124.979f (stage 3 Girl02_at).
		v := t.pop()
		bits := v.bits
		if !v.isFloat {
			bits = math.Float32bits(float32(int32(v.bits)))
		}
		t.push(eVal{true, uint32(-int32(bits))})
	case opSqrt:
		t.push(floatVal(float32(math.Sqrt(float64(vm.popFloat(t))))))
	default:
		return false
	}
	return true
}

func boolVal(b bool) eVal {
	if b {
		return intVal(1)
	}
	return intVal(0)
}

// --- control flow helpers ---

func (vm *VM) jump(t *eclTask, in *Instr, offsetParam, timeParam Param) {
	target, ok := t.sub.offsetIndex[in.Offset+offsetParam.Int()]
	if !ok {
		t.pc++
		return
	}
	t.pc = target
	t.clock = float32(timeParam.Int())
}

func (vm *VM) doCall(ctx *stage.Context, t *eclTask, in *Instr, async bool, asyncID int32) {
	name := in.Params[0].Str
	// Call arguments are the trailing D params.
	firstArg := 1
	if in.Opcode == opCallAsyncID {
		firstArg = 2
	}
	args := make([]eVal, 0, len(in.Params)-firstArg)
	for _, p := range in.Params[firstArg:] {
		args = append(args, vm.resolveCallArg(t, p))
	}
	if async {
		// A new async context goes to the head of the owner's list.
		if a := vm.actors[t.owner]; a != nil {
			nt := &eclTask{sub: vm.prog.SubByName[name], owner: t.owner, async: true, asyncID: asyncID}
			if nt.sub != nil {
				nt.frame = newFrame(nt.sub, args)
			}
			a.asyncs = append([]*eclTask{nt}, a.asyncs...)
		}
		t.pc++
		return
	}
	callee := vm.prog.SubByName[name]
	if callee == nil {
		t.pc++
		return
	}
	t.calls = append(t.calls, callFrame{sub: t.sub, pc: t.pc + 1, clock: t.clock, frame: t.frame})
	t.sub = callee
	t.pc = 0
	t.clock = 0
	t.frame = newFrame(callee, args)
}

// killAsync frees the owner's async contexts other than current: the first
// with the given id (op 17), or all of them for id < 0 (op 21). The primary
// context is never one of them.
func (vm *VM) killAsync(current *eclTask, id int32) {
	a := vm.actors[current.owner]
	if a == nil {
		return
	}
	for _, task := range a.asyncs {
		if task == current || task.done {
			continue
		}
		if id < 0 {
			task.done = true
		} else if task.asyncID == id {
			task.done = true
			return
		}
	}
}

// --- variable & stack machinery ---

func newFrame(sub *Sub, args []eVal) []uint32 {
	size := 0
	if len(sub.Instrs) > 0 && sub.Instrs[0].Opcode == opStackAlloc && len(sub.Instrs[0].Params) > 0 {
		size = int(sub.Instrs[0].Params[0].Int()) / 4
	}
	if size < len(args) {
		size = len(args)
	}
	f := make([]uint32, size)
	for i, a := range args {
		f[i] = a.bits
	}
	return f
}

func (vm *VM) ensureFrame(t *eclTask, byteSize int) {
	n := byteSize / 4
	for len(t.frame) < n {
		t.frame = append(t.frame, 0)
	}
}

func (t *eclTask) push(v eVal) { t.stack = append(t.stack, v) }

func (t *eclTask) pop() eVal {
	if len(t.stack) == 0 {
		return eVal{}
	}
	v := t.stack[len(t.stack)-1]
	t.stack = t.stack[:len(t.stack)-1]
	return v
}

func (vm *VM) popInt(t *eclTask) int32     { return t.pop().asInt() }
func (vm *VM) popFloat(t *eclTask) float32 { return t.pop().asFloat() }

// resolveInt/resolveFloat turn a param into a value (immediate, local var,
// stack pop, or special var). Faithful to the binary: the *resolver* decides
// how the raw 4 bytes encode the var id — FUN_0044fdb0 (int) reads it as an
// int, FUN_0044fe40 (float) reads it as a float bit pattern (so a 'D' cast
// slot holding -1.0f means "pop stack" on the float path, not id -1082130432).
func (vm *VM) resolveInt(t *eclTask, p Param) int32 {
	if !p.IsVar {
		return p.Int()
	}
	id := p.Int()
	switch {
	case id >= 0:
		return int32(vm.frameWord(t, id))
	case id == -1:
		return vm.popInt(t)
	default:
		return vm.specialInt(t, id)
	}
}

func (vm *VM) resolveFloat(t *eclTask, p Param) float32 {
	if !p.IsVar {
		return p.Float()
	}
	id := int32(p.Float())
	switch {
	case id >= 0:
		return math.Float32frombits(vm.frameWord(t, id))
	case id == -1:
		return vm.popFloat(t)
	default:
		return vm.specialFloat(t, id)
	}
}

// resolveCallArg resolves a 'D' (or stack/var) call argument keeping its type.
func (vm *VM) resolveCallArg(t *eclTask, p Param) eVal {
	if p.IsVar {
		if p.DFrom == 'f' || p.Type == 'f' {
			return floatVal(vm.resolveFloat(t, p))
		}
		return intVal(vm.resolveInt(t, p))
	}
	if p.DFrom == 'f' {
		return floatVal(p.Float())
	}
	return intVal(p.Int())
}

func (vm *VM) frameWord(t *eclTask, id int32) uint32 {
	idx := int(id) / 4
	if idx < 0 || idx >= len(t.frame) {
		return 0
	}
	return t.frame[idx]
}

func (vm *VM) storeVar(t *eclTask, p Param, v eVal) {
	id := p.VarID()
	if id < 0 {
		vm.storeSpecial(t, id, v)
		return
	}
	vm.ensureFrame(t, int(id)+4)
	t.frame[id/4] = v.bits
}

func (vm *VM) loadVarInt(t *eclTask, p Param) int32 {
	if !p.IsVar {
		return p.Int()
	}
	if id := p.VarID(); id < -1 {
		return vm.specialInt(t, id)
	}
	return int32(vm.frameWord(t, p.VarID()))
}

func (vm *VM) loadVarFloat(t *eclTask, p Param) float32 {
	if !p.IsVar {
		return p.Float()
	}
	if id := p.VarID(); id < -1 {
		return vm.specialFloat(t, id)
	}
	return math.Float32frombits(vm.frameWord(t, p.VarID()))
}

// aimAngleAt returns atan2(player - (x,y)) like FUN_004073e0: exactly pi/2
// (down) when there is no player or the fire origin coincides with the player.
func (vm *VM) aimAngleAt(x, y float64) float64 {
	if !vm.hasPlayer || (vm.playerX == x && vm.playerY == y) {
		return math.Pi / 2
	}
	return math.Atan2(vm.playerY-y, vm.playerX-x)
}

// rngRaw, randSigned and randFloat draw from the game RNG (zun.RNG).
func (vm *VM) rngRaw() uint32      { return vm.rng.Raw() }
func (vm *VM) randSigned() float32 { return vm.rng.Signed() }
func (vm *VM) randFloat() float32  { return vm.rng.Float() }

func eclCenterX() float64 { return float64(game.FieldLeft+game.FieldRight) / 2 }

func eclToWorld(x, y float64) (float64, float64) {
	return eclCenterX() + x, float64(game.FieldTop) + y
}
