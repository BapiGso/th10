package ecl

// TH10 ECL virtual machine — runtime over the binary IR (see loader.go).
//
// Faithful to the binary's model rather than thtk's decompiled text:
//   - Each task has a virtual clock. An instruction runs once clock>=Time;
//     the clock advances 1 per frame while waiting. goto/if/unless carry a
//     time arg that sets the clock; wait(83) freezes the clock for N frames.
//     (Confirmed against Boss1/Boss1At1 instruction Time fields.)
//   - System opcodes (<256) are a typed stack machine (expr.c / FUN_0043ee30):
//     LOADI/F push, ASSIGNI/F pop->var, 50-84 arithmetic/compare/logic.
//   - Variables: param_mask bit => the 4-byte value is a var ref; id>=0 is a
//     local-frame byte offset, id==-1 pops the data stack, id<-1 is a special
//     var (FUN_0044fdb0). Float vars encode the id as a float bit pattern.
//
// Movement (282-299) and danmaku (400-436) handlers live in game_ops.go /
// bullet_ops.go; this file owns the core loop, control flow, and resolution.

import (
	"math"
	"sort"

	"th10/assets"
	"th10/entity/enemy"
	"th10/game"
	"th10/scene/stage"
)

// VM runs an ECL program for one stage.
type VM struct {
	prog     *Program
	tasks    []*eclTask
	actors   map[*enemy.Enemy]*eclActor
	frame    int
	rngState uint16 // ZUN RNG state (FUN_0044bb90); seed DAT_004918b0
	mainTask *eclTask // the "main" stage task; stage ends when it finishes

	// Dynamic rank (DAT_00474c98, -1024..1024): +1 every 60 frames while
	// enemies are on screen and no dialog is open (player update FUN_00425730).
	// Consumed by the rank-tier ET opcodes 422-427.
	rank int

	// Global time scale (DAT_00476f78, set by ins_367). Every script Timer and
	// tween advances by this much per frame; 1.0 = normal speed.
	timeScale float64
	tickAcc   float64 // fractional script-tick accumulator

	// snapshot of stage state for special-var resolution (set each frame)
	difficulty int
	playerX    float64
	playerY    float64
	hasPlayer  bool

	// escape/timeout handler registered by ins_341 (slot, subName): when the
	// current attack times out, the sub runs on the registering enemy
	// (e.g. "MBossEscape" makes the midboss fly away).
	escapeSub   string
	escapeOwner *enemy.Enemy

	// dialog bridge: ins_338 records the requested id; the next ins_339 fires
	// DialogFunc(id). If DialogFunc returns true (dialog shown), the task blocks
	// one frame at a time until DialogActive returns false (overlay dismissed).
	dialogID     int32
	dialogArmed  bool                // a 338 set an id that the next 339 consumes
	dialogFiring bool                // a 339 is currently waiting on an open overlay
	DialogFunc   func(id int32) bool // set by the stage wrapper; nil = no dialog
	DialogActive func() bool         // reports whether the dialog overlay is up

	// screen shake (ins_337): magnitude decays linearly over the duration.
	shakeFrames int
	shakeTotal  int
	shakeMag    float64

	// boss spell-card HUD state, driven by 357/359 (declare), 341/368 (timer),
	// and the boss enemy's HP. Decoded from the ECL itself.
	spell spellState
}

// spellState tracks the currently-declared boss spell card for the HUD.
type spellState struct {
	active    bool
	name      string // UTF-8 (decoded from the ECL's Shift-JIS 357/359 arg)
	bonus     int64  // base bonus score
	timeLimit int    // total frames (from 357/359 arg or a setTimeout)
	timeLeft  int    // remaining frames, counted down each VM frame
	owner     *enemy.Enemy
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
	sub   *Sub
	pc    int
	clock int32
	wait  int // explicit ins_83 frames remaining (clock frozen)

	frame []uint32 // local variable slots, indexed by byte offset / 4
	stack []eVal
	calls []callFrame

	owner   *enemy.Enemy
	asyncID int32 // callAsyncID slot, for targeted kill; -1 otherwise
	done    bool
}

type callFrame struct {
	sub   *Sub
	pc    int
	clock int32
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
		rngState:  0x1234,
		timeScale: 1,
	}
	if data, err := assets.Assets.ReadFile(eclPath); err == nil {
		if prog, perr := LoadProgram(data); perr == nil {
			vm.prog = prog
		}
	}
	vm.mainTask = vm.startTask("main", nil, nil, -1)
	return vm
}

// MainDone reports whether the stage's "main" task has finished. The Stage 1
// main spawns the timeline, then blocks on deathWait (ins_340) until the boss
// is defeated and ends with delete() — so this is the faithful stage-clear
// signal (replacing the old fixed-frame cutoff).
func (vm *VM) MainDone() bool {
	return vm.mainTask == nil || vm.mainTask.done
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

// BossHUD returns the current boss/spell HUD snapshot. The boss is the live
// TypeBoss enemy with the most HP (mid-boss + boss never co-exist in TH10).
func (vm *VM) BossHUD() stage.BossHUDInfo {
	var boss *enemy.Enemy
	for e := range vm.actors {
		if e == nil || !e.Active {
			continue
		}
		if e.Type == enemy.TypeBoss || e.Type == enemy.TypeMidBoss {
			if boss == nil || e.MaxHP > boss.MaxHP {
				boss = e
			}
		}
	}
	if boss == nil {
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
	if a := vm.actors[boss]; a != nil && boss.MaxHP > 0 {
		for _, m := range a.markers {
			mf := m.hp / float64(boss.MaxHP)
			if mf > 0 && mf < 1 {
				info.HPMarkers = append(info.HPMarkers, mf)
			}
		}
		sort.Float64s(info.HPMarkers)
	}
	if vm.spell.active && vm.spell.owner == boss {
		info.Spell = true
		info.Name = vm.spell.name
		info.Bonus = vm.spell.bonus
		info.TimeLeft = (vm.spell.timeLeft + 59) / 60 // frames → ceil seconds
	}
	return info
}

// SetDialogBridge wires the ECL dialog opcodes (338/339) to the stage. open is
// called with the ECL dialog id when a 339 fires and should display the overlay
// (returning true if it did); active reports whether the overlay is still up.
func (vm *VM) SetDialogBridge(open func(id int32) bool, active func() bool) {
	vm.DialogFunc = open
	vm.DialogActive = active
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

// Update advances movement and runs every live task one frame.
func (vm *VM) Update(ctx *stage.Context) {
	vm.frame++
	if vm.shakeFrames > 0 {
		vm.shakeFrames--
	}
	// ins_367 time scale: script Timers advance by timeScale per real frame
	// (velocity integration is not Timer-driven and keeps full speed). The
	// accumulator turns fractional scales into 0/1 whole script ticks.
	vm.tickAcc += vm.timeScale
	ticks := int(vm.tickAcc)
	vm.tickAcc -= float64(ticks)

	// Dynamic rank (FUN_00425730): +1 every 60 frames while enemies are on
	// screen and no dialog is open, clamped to [-1024, 1024].
	if len(vm.actors) > 0 && (vm.DialogActive == nil || !vm.DialogActive()) && vm.frame%60 == 0 {
		if vm.rank++; vm.rank > 1024 {
			vm.rank = 1024
		}
	}

	// Tick the spell-card countdown; drop the HUD when the card's owner dies.
	// On timeout, fire the ins_341 escape handler (e.g. "MBossEscape").
	if vm.spell.active {
		switch {
		case vm.spell.owner != nil && !vm.spell.owner.Active:
			vm.spell.active = false
		case vm.spell.timeLeft > 0:
			if vm.spell.timeLeft -= ticks; vm.spell.timeLeft <= 0 {
				vm.spell.timeLeft = 0
				vm.spell.active = false
				vm.fireEscape()
			}
		}
	}
	if ctx != nil {
		if ctx.State != nil {
			vm.difficulty = ctx.State.Difficulty
		}
		if ctx.Player != nil {
			vm.playerX, vm.playerY, vm.hasPlayer = ctx.Player.X, ctx.Player.Y, true
		}
	}
	// Integrate enemy movement first (positions used by this frame's logic).
	for e, a := range vm.actors {
		if !e.Active {
			delete(vm.actors, e)
			continue
		}
		a.integrate(e, ticks)
	}

	// Step every task that exists at the start of each tick. Tasks spawned
	// during stepping (startTask appends to vm.tasks, possibly reallocating)
	// run starting next tick, so the loop bound is fixed per round. We index
	// rather than range, and never reslice vm.tasks here, so an append can't
	// clobber a not-yet-visited entry (the old [:0] alias did exactly that
	// and silently dropped the spawner task).
	for tick := 0; tick < ticks; tick++ {
		n := len(vm.tasks)
		for i := 0; i < n; i++ {
			t := vm.tasks[i]
			if t.done || (t.owner != nil && !t.owner.Active) {
				continue
			}
			vm.stepTask(ctx, t)
		}
	}

	// Compact in place: drop finished tasks and tasks whose owner died.
	live := vm.tasks[:0]
	for _, t := range vm.tasks {
		if t.done || (t.owner != nil && !t.owner.Active) {
			continue
		}
		live = append(live, t)
	}
	vm.tasks = live
}

// fireEscape runs the ins_341-registered timeout sub on its enemy (once).
func (vm *VM) fireEscape() {
	if vm.escapeSub == "" || vm.escapeOwner == nil || !vm.escapeOwner.Active {
		return
	}
	vm.startTask(vm.escapeSub, vm.escapeOwner, nil, -1)
	vm.escapeSub = ""
}

// stepTask runs one frame of a task using the clock model described above.
func (vm *VM) stepTask(ctx *stage.Context, t *eclTask) {
	if t.wait > 0 {
		t.wait--
		return
	}
	diff := 1
	if ctx != nil && ctx.State != nil {
		diff = ctx.State.Difficulty
	}
	for budget := 0; budget < 100000; budget++ {
		if t.pc >= len(t.sub.Instrs) {
			if !vm.returnFromCall(t) {
				t.done = true
				return
			}
			continue
		}
		in := &t.sub.Instrs[t.pc]
		if in.Time > t.clock {
			break // not yet time; yield and let the clock catch up
		}
		if !in.rankAllows(diff) {
			t.pc++
			continue
		}
		vm.execInstr(ctx, t, in)
		if t.done {
			return
		}
		if t.wait > 0 {
			return // explicit wait: do not advance the clock this frame
		}
	}
	t.clock++ // one frame elapsed
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
		t.pc++
		return
	}
	vm.execSystemOp(ctx, t, in)
}

// execSystemOp implements opcodes 0-93: the stack machine + control flow.
// Most ops advance pc by 1; jumps/return set pc explicitly.
func (vm *VM) execSystemOp(ctx *stage.Context, t *eclTask, in *Instr) {
	switch in.Opcode {
	case opRetBig: // 1: delete owner + end this task tree
		if t.owner != nil && len(t.calls) == 0 {
			t.owner.Active = false
		}
		t.done = true
	case opRetNormal: // 10
		if !vm.returnFromCall(t) {
			t.done = true
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
	case opWait: // 83 S
		n := int(vm.resolveInt(t, in.Params[0]))
		if n > 0 {
			t.wait = n
		}
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
		angle := float64(vm.resolveFloat(t, in.Params[2]))
		radius := float64(vm.resolveFloat(t, in.Params[3]))
		vm.storeVar(t, in.Params[0], floatVal(float32(math.Cos(angle)*radius)))
		vm.storeVar(t, in.Params[1], floatVal(float32(math.Sin(angle)*radius)))
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
	t.clock = timeParam.Int()
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
		vm.startTask(name, t.owner, args, asyncID)
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

func (vm *VM) startTask(name string, owner *enemy.Enemy, args []eVal, asyncID int32) *eclTask {
	sub := vm.prog.SubByName[name]
	if sub == nil {
		return nil
	}
	t := &eclTask{sub: sub, owner: owner, asyncID: asyncID, frame: newFrame(sub, args)}
	vm.tasks = append(vm.tasks, t)
	return t
}

func (vm *VM) killAsync(current *eclTask, id int32) {
	for _, task := range vm.tasks {
		if task == current || task.owner != current.owner || task.owner == nil {
			continue
		}
		if id < 0 || task.asyncID == id {
			task.done = true
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
		return int32(vm.specialFloat(t, id))
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
		return // can't assign to a special var
	}
	vm.ensureFrame(t, int(id)+4)
	t.frame[id/4] = v.bits
}

func (vm *VM) loadVarInt(t *eclTask, p Param) int32 {
	if !p.IsVar {
		return p.Int()
	}
	return int32(vm.frameWord(t, p.VarID()))
}

func (vm *VM) loadVarFloat(t *eclTask, p Param) float32 {
	if !p.IsVar {
		return p.Float()
	}
	return math.Float32frombits(vm.frameWord(t, p.VarID()))
}

// specialFloat returns the value of a special/global variable (id < -1).
// Behavioral (Pareto) approximations; refine via the differential harness.
func (vm *VM) specialFloat(t *eclTask, id int32) float32 {
	switch id {
	case -9959: // difficulty / rank
		return float32(vm.difficulty)
	case -9986: // spell-practice/replay flag (0 in normal play)
		return 0
	case -9989: // angle from enemy to player
		return float32(vm.aimAngle(t))
	case -9987, -9999: // random float [0,1)
		return vm.randFloat()
	case -9998: // random angle [-pi, pi): FUN_0044bb90 draw scaled by pi
		return float32(vm.randSigned() * math.Pi)
	case -10000: // global frame counter
		return float32(vm.frame)
	default:
		return 0
	}
}

func (vm *VM) aimAngle(t *eclTask) float64 {
	if t.owner == nil || !vm.hasPlayer {
		return math.Pi / 2 // downward
	}
	return math.Atan2(vm.playerY-t.owner.Y, vm.playerX-t.owner.X)
}

// aimAngleAt returns atan2(player - (x,y)) like FUN_004073e0: exactly pi/2
// (down) when there is no player or the fire origin coincides with the player.
func (vm *VM) aimAngleAt(x, y float64) float64 {
	if !vm.hasPlayer || (vm.playerX == x && vm.playerY == y) {
		return math.Pi / 2
	}
	return math.Atan2(vm.playerY-y, vm.playerX-x)
}

// rngRaw advances the ZUN RNG (FUN_0044bb90) and returns the combined 32-bit
// draw: two 16-bit steps of w = rotl16((w ^ 0x9630) - 0x6553, 2) followed by
// w = rotl16((w ^ 0x9630) + 0x9aad, 2), giving hi<<16|lo. Bit-exact.
func (vm *VM) rngRaw() uint32 {
	w := (vm.rngState ^ 0x9630) - 0x6553
	w = w<<2 | w>>14
	u := (w ^ 0x9630) + 0x9aad
	u = u<<2 | u>>14
	vm.rngState = u
	return uint32(w)<<16 | uint32(u)
}

// randSigned returns the FUN_0044bb90 float draw. The binary computes
// raw * DAT_00470bec - DAT_00470afc; with afc = 1.0 (shared with the Timer
// tick) and bec = 2^-31 this is [-1, 1). Scale constants pending the round-2
// const dump; the integer sequence above is exact regardless.
func (vm *VM) randSigned() float64 {
	return float64(vm.rngRaw())*(1.0/2147483648.0) - 1.0
}

// randFloat returns a [0,1) draw for call sites whose original getter
// (FUN_0044bb20 family) is not yet dumped; derived from the same sequence.
func (vm *VM) randFloat() float32 {
	return float32(vm.rngRaw()) * (1.0 / 4294967296.0)
}

func normAngle(a float32) float32 {
	x := float64(a)
	for x > math.Pi {
		x -= 2 * math.Pi
	}
	for x <= -math.Pi {
		x += 2 * math.Pi
	}
	return float32(x)
}

func eclCenterX() float64 { return float64(game.FieldLeft+game.FieldRight) / 2 }

func eclToWorld(x, y float64) (float64, float64) {
	return eclCenterX() + x, float64(game.FieldTop) + y
}
