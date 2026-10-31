package ecl

import (
	"math"
	"reflect"
	"testing"

	"th10/entity/enemy"
	"th10/entity/item"
	"th10/game"
	"th10/scene/dialog"
	"th10/scene/stage"
	"th10/zun"
)

// newTestVM is a VM with no program, for driving ops and contexts directly.
func newTestVM() *VM {
	return &VM{
		prog:      &Program{SubByName: map[string]*Sub{}},
		actors:    map[*enemy.Enemy]*eclActor{},
		rng:       &zun.RNG{State: 1},
		gameSpeed: 1,
	}
}

func newTestContext() *stage.Context {
	return stage.NewContext(game.NewState(game.CharReimu, game.ShotA, game.DiffNormal))
}

// updateBullets is the stage's bullet step: the manager, then the sparks.
func updateBullets(ctx *stage.Context) {
	ctx.UpdateBullets()
	ctx.UpdateWorld()
}

// stepFrame runs one stage frame of the VM: the scripts, then (after the
// stage's collision pass) the interrupt check and timers.
func stepFrame(vm *VM, ctx *stage.Context) {
	vm.Update(ctx)
	vm.LateUpdate(ctx)
}

func activeEnemies(ctx *stage.Context) int {
	n := 0
	for _, e := range ctx.Enemies {
		if e != nil && e.Active {
			n++
		}
	}
	return n
}

// TestVMClockTimingFirstWave verifies the virtual-clock model: main waits the
// intro (~160 frames) before the first ins_257 spawn.
func TestVMClockTimingFirstWave(t *testing.T) {
	ctx := newTestContext()
	vm := NewStage01VM()

	for i := 0; i < 150; i++ {
		stepFrame(vm, ctx)
	}
	if got := activeEnemies(ctx); got != 0 {
		t.Fatalf("frame 150: %d enemies, want 0 (still in intro wait)", got)
	}
	for i := 150; i < 280; i++ {
		stepFrame(vm, ctx)
	}
	if got := activeEnemies(ctx); got < 3 {
		t.Fatalf("frame 280: %d enemies, want >=3 spawned", got)
	}
}

// TestVMSpawnWaveSpreadsAcross checks the expression machine: main's spawn loop
// does %B += 32 each iteration, so the wave fans out across distinct X columns.
func TestVMSpawnWaveSpreadsAcross(t *testing.T) {
	ctx := newTestContext()
	vm := NewStage01VM()
	for i := 0; i < 360; i++ {
		stepFrame(vm, ctx)
	}
	xs := map[int]bool{}
	moved := false
	for _, e := range ctx.Enemies {
		if e == nil || !e.Active {
			continue
		}
		xs[int(e.X/8)] = true
		if e.Y > game.FieldTop+8 {
			moved = true
		}
	}
	if len(xs) < 3 {
		t.Fatalf("first wave occupies %d distinct columns, want >=3 (expr-driven spread)", len(xs))
	}
	if !moved {
		t.Fatal("no enemy moved down into the field (movement integration failed)")
	}
}

// TestVMFiresDanmaku runs far enough that enemy/boss ET emitters produce bullets.
func TestVMFiresDanmaku(t *testing.T) {
	ctx := newTestContext()
	vm := NewStage01VM()
	maxBullets := 0
	for i := 0; i < 4000; i++ {
		stepFrame(vm, ctx)
		if n := ctx.Bullets.Live(); n > maxBullets {
			maxBullets = n
		}
		updateBullets(ctx)
	}
	if maxBullets == 0 {
		t.Fatal("no bullets fired across 4000 frames")
	}
}

// TestVMBossSpellHUD verifies the boss HUD data flow: running stage 2 long
// enough to declare a spell card (357/359) yields a HUD with a decoded UTF-8
// name, a counting-down timer, and a valid HP fraction.
func TestVMBossSpellHUD(t *testing.T) {
	ctx := newTestContext()
	vm := NewStageVM("ecl/stage02.ecl")
	var sawSpell BossHUDInfoSnapshot
	for i := 0; i < 12000; i++ {
		stepFrame(vm, ctx)
		updateBullets(ctx)
		h := vm.BossHUD()
		if h.Active && h.HPFrac < 0 || h.HPFrac > 1 {
			t.Fatalf("frame %d: HPFrac out of range: %v", i, h.HPFrac)
		}
		if h.Spell && h.Name != "" {
			sawSpell = BossHUDInfoSnapshot{name: h.Name, timeLeft: h.TimeLeft, hp: h.HPFrac}
			break
		}
	}
	if sawSpell.name == "" {
		t.Fatal("no spell card declared/decoded across 12000 frames")
	}
	// The decoded name must be real UTF-8 Japanese (contains the 「」 quotes TH
	// spell names use), proving Shift-JIS decode worked.
	if !containsRune(sawSpell.name, '「') {
		t.Errorf("spell name %q lacks 「 — Shift-JIS decode likely failed", sawSpell.name)
	}
	t.Logf("spell HUD: name=%q timeLeft=%d hp=%.2f", sawSpell.name, sawSpell.timeLeft, sawSpell.hp)
}

type BossHUDInfoSnapshot struct {
	name     string
	timeLeft int
	hp       float64
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func TestVMSpellBgDeclaresHUD(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	owner := enemy.New(100, 100, 1000, enemy.TypeBoss)
	vm.actorFor(owner).ran = true // as if updated this frame
	task := &eclTask{owner: owner}

	vm.execGameOp(ctx, task, &Instr{Opcode: opSetBoss, Params: []Param{intParam(0)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opSetInterrupt, Params: []Param{
		intParam(0), intParam(0), intParam(4200), stringParam("Next"),
	}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opSpellBg, Params: []Param{
		intParam(80),
		intParam(4200),
		intParam(1000000),
		stringParam("Test Spell"),
	}})
	vm.LateUpdate(ctx) // the slot check writes the timer: ceil(4200/60) = 70

	h := vm.BossHUD()
	if !h.Active || !h.Spell {
		t.Fatalf("BossHUD active/spell = %v/%v, want true/true", h.Active, h.Spell)
	}
	if h.Name != "Test Spell" || h.TimeLeft != 70 || h.Bonus != 1000000 {
		t.Fatalf("BossHUD = name %q time %d bonus %d", h.Name, h.TimeLeft, h.Bonus)
	}
}

func TestVMLifeMarkersExposeBossHUDFractions(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	owner := enemy.New(100, 100, 10000, enemy.TypeBoss)
	vm.actorFor(owner)
	task := &eclTask{owner: owner}

	vm.execGameOp(ctx, task, &Instr{Opcode: opSetBoss, Params: []Param{intParam(0)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opLifeMarker, Params: []Param{
		intParam(0),
		floatParam(2500),
		intParam(-24448),
	}})

	h := vm.BossHUD()
	if len(h.HPMarkers) != 1 {
		t.Fatalf("HPMarkers len = %d, want 1 (%v)", len(h.HPMarkers), h.HPMarkers)
	}
	if math.Abs(h.HPMarkers[0]-0.25) > 1e-6 {
		t.Fatalf("HPMarkers[0] = %v, want 0.25", h.HPMarkers[0])
	}
}

func TestVMDropAreaSpreadsOpcodeDrops(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	owner := enemy.New(100, 100, 100, enemy.TypeFairy)
	vm.actorFor(owner)
	task := &eclTask{owner: owner}

	vm.execGameOp(ctx, task, &Instr{Opcode: opDropClear})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropExtra, Params: []Param{intParam(1), intParam(4)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropArea, Params: []Param{floatParam(64), floatParam(32)}})
	vm.execGameOp(ctx, task, &Instr{Opcode: opDropItems})

	count := 0
	spread := false
	ctx.Items.Each(func(it *item.Item) {
		count++
		if it.X != owner.X || it.Y != owner.Y {
			spread = true
		}
		if it.X < owner.X-32 || it.X > owner.X+32 || it.Y < owner.Y-16 || it.Y > owner.Y+16 {
			t.Fatalf("item at %.2f,%.2f outside drop area", it.X, it.Y)
		}
	})
	if count != 4 {
		t.Fatalf("spawned %d items, want 4", count)
	}
	if !spread {
		t.Fatal("drop area did not move any spawned item away from owner position")
	}
}

func intParam(v int32) Param {
	return Param{Type: 'S', Raw: uint32(v)}
}

func floatParam(v float32) Param {
	return Param{Type: 'f', Raw: math.Float32bits(v)}
}

func stringParam(v string) Param {
	return Param{Type: 'x', Str: v}
}

// TestVMStableFullStage ensures a long run never panics (bad jumps, stack
// underflow, frame indexing, etc.).
func TestVMStableFullStage(t *testing.T) {
	ctx := newTestContext()
	vm := NewStage01VM()
	for i := 0; i < 8000; i++ {
		stepFrame(vm, ctx)
		updateBullets(ctx)
	}
}

// TestVMMSGBlocksMain: ECL 338 opens each .msg entry once and 339 holds the
// main task until the MSG signals (op 11) or ends, without re-opening or
// deadlocking. The player presses Z every 30 frames.
func TestVMMSGBlocksMain(t *testing.T) {
	ctx := newTestContext()
	ctx.LoadMSG("stage2", 0)
	vm := NewStageVM("ecl/stage02.ecl")

	var opened []int32
	var last *dialog.MSG
	for i := 0; i < 20000 && !vm.MainDone(); i++ {
		stepFrame(vm, ctx)
		updateBullets(ctx)
		if m := ctx.MSG(); m != nil && m != last {
			opened, last = append(opened, m.ID), m
			if mt := vm.main.main; mt.sub.Instrs[mt.pc].Opcode != opMSGWait {
				t.Fatalf("frame %d: main not waiting on dialog %d", i, m.ID)
			}
		}
		// Clear enemies so the boss's deathWait (340) between the two dialogs
		// passes and the timeline reaches the post dialog.
		for _, e := range ctx.Enemies {
			if e != nil {
				e.Active = false
			}
		}
		ctx.UpdateMSG(i%30 == 0, false)
	}
	// Stage 2 has two dialog events (bossPre, post). Each must open once.
	if !reflect.DeepEqual(opened, []int32{0, 1}) {
		t.Fatalf("opened %v, want [0 1] (bossPre + post)", opened)
	}
	if !vm.MainDone() || !ctx.Cleared() {
		t.Fatalf("main done %v, cleared %v; want both", vm.MainDone(), ctx.Cleared())
	}
}

// TestVMAllStagesStableAndFire runs each stage's VM for a long span: it must not
// panic (proving missing opcodes degrade to no-ops, not crashes) and must emit
// danmaku (proving the timeline/spawn/emitter machinery drives every stage).
func TestVMAllStagesStableAndFire(t *testing.T) {
	stages := []string{
		"ecl/stage02.ecl", "ecl/stage03.ecl", "ecl/stage04.ecl",
		"ecl/stage05.ecl", "ecl/stage06.ecl", "ecl/stage07.ecl",
	}
	for _, name := range stages {
		name := name
		t.Run(name, func(t *testing.T) {
			ctx := newTestContext()
			vm := NewStageVM(name)
			maxBullets := 0
			maxEnemies := 0
			for i := 0; i < 8000; i++ {
				stepFrame(vm, ctx)
				if n := ctx.Bullets.Live(); n > maxBullets {
					maxBullets = n
				}
				if n := activeEnemies(ctx); n > maxEnemies {
					maxEnemies = n
				}
				updateBullets(ctx)
			}
			if maxEnemies == 0 {
				t.Fatalf("%s: no enemies ever spawned", name)
			}
			if maxBullets == 0 {
				t.Fatalf("%s: no bullets fired across 8000 frames", name)
			}
			t.Logf("%s: peak enemies=%d peak bullets=%d", name, maxEnemies, maxBullets)
		})
	}
}
