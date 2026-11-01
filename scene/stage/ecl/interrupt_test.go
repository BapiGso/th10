package ecl

import (
	"reflect"
	"testing"

	"th10/entity/enemy"
	"th10/scene/dialog"
	"th10/scene/stage"
)

// spawnStageSub loads a stage's ECL without running its main and creates an
// enemy running sub name, as ins_257 (absolute, unmirrored) at (0, 112) does.
func spawnStageSub(t *testing.T, eclPath, name string) (*VM, *stage.Context, *enemy.Enemy) {
	t.Helper()
	ctx := newTestContext()
	vm := NewStageVM(eclPath)
	vm.started = true // leave main unstarted
	vm.ctx = ctx
	vm.createOp(ctx, &eclTask{}, &Instr{Opcode: opEnmCreateA, Params: []Param{
		stringParam(name), floatParam(0), floatParam(112), intParam(100), intParam(0), intParam(0),
	}})
	// Enemies the new one creates in its first update are listed before it.
	n := len(ctx.Enemies)
	if n == 0 || !ctx.Enemies[n-1].Active {
		t.Fatalf("%s: %s did not spawn", eclPath, name)
	}
	return vm, ctx, ctx.Enemies[n-1]
}

func liveAsyncs(vm *VM, owner *enemy.Enemy) int {
	n := 0
	for _, task := range vm.actors[owner].asyncs {
		if !task.done {
			n++
		}
	}
	return n
}

// TestVMCreateRunsTimeZero: the create op runs the new enemy's time-0
// instructions on the spot, so Hina's setBoss(0)/setHP(9000)/sprite are in
// place before the spawner moves on (the stage's ins_340 relies on it).
func TestVMCreateRunsTimeZero(t *testing.T) {
	vm, _, boss := spawnStageSub(t, "ecl/stage02.ecl", "Boss")
	if vm.bossSlots[0] != boss || boss.ECLFlags&0x8000 == 0 {
		t.Fatalf("boss slot 0 = %p (flags %#x), want the new boss", vm.bossSlots[0], boss.ECLFlags)
	}
	if boss.HP != 9000 || boss.MaxHP != 9000 {
		t.Fatalf("HP %d/%d, want 9000/9000", boss.HP, boss.MaxHP)
	}
	if boss.Hidden {
		t.Fatal("boss still hidden after its time-0 ins_259")
	}
	// ins_335's 60, less the tick at the end of that first update.
	if got := vm.actors[boss].invuln.cur; got != 59 {
		t.Fatalf("invuln = %d, want 59 (ins_335(60), one update)", got)
	}
}

// reachBossCard1 runs Hina into Boss1 (its slot 0 = 2700 HP / 2700 frames ->
// BossCard1) and knocks her HP under the threshold.
func reachBossCard1(t *testing.T) (*VM, *stage.Context, *enemy.Enemy) {
	t.Helper()
	vm, ctx, boss := spawnStageSub(t, "ecl/stage02.ecl", "Boss")
	for i := 0; i < 70; i++ {
		stepFrame(vm, ctx)
	}
	a := vm.actors[boss]
	if s := a.slots[0]; s.hp != 2700 || s.time != 2700 || s.sub != "BossCard1" {
		t.Fatalf("slot 0 = %+v, want Boss1's (2700, 2700, BossCard1)", s)
	}
	if liveAsyncs(vm, boss) == 0 {
		t.Fatal("Boss1At not running before the interrupt")
	}
	vm.Update(ctx)
	if boss.TakeDamage(6400) { // 9000 -> 2600 <= 2700
		t.Fatal("the HP interrupt destroyed the boss")
	}
	return vm, ctx, boss
}

// TestVMInterruptOnHP: falling to the slot's HP clamps the HP to it, frees
// the asyncs and runs BossCard1 at once (FUN_004127a0 after damage).
func TestVMInterruptOnHP(t *testing.T) {
	vm, ctx, boss := reachBossCard1(t)
	a := vm.actors[boss]
	if !boss.Active || boss.HP != 2700 {
		t.Fatalf("active %v HP %d, want alive at 2700", boss.Active, boss.HP)
	}
	if a.main.sub.Name != "BossCard1" {
		t.Fatalf("primary sub %q, want BossCard1", a.main.sub.Name)
	}
	if s := a.slots[0]; s.hp != 0 || s.time != 2700 || s.sub != "Boss2" || s.timeoutSub != "Boss2" {
		t.Fatalf("slot 0 = %+v, want BossCard1's (0, 2700, Boss2)", s)
	}
	if n := liveAsyncs(vm, boss); n != 0 {
		t.Fatalf("%d asyncs survived the interrupt", n)
	}
	if !vm.card.Active || vm.spell.owner != boss {
		t.Fatal("BossCard1's ins_357 did not declare the card")
	}
	breakers := 0
	for _, e := range ctx.Enemies {
		if ea := vm.actors[e]; e.Active && ea != nil && ea.main.sub.Name == "Ecl_EtBreak" {
			breakers++
			if !e.Hidden || e.Shootable() {
				t.Fatalf("Ecl_EtBreak enemy hidden %v shootable %v, want hidden and unshootable", e.Hidden, e.Shootable())
			}
		}
	}
	if breakers != 1 {
		t.Fatalf("%d Ecl_EtBreak enemies, want the one from BossCard1 (default.ecl)", breakers)
	}
	if boss.TakeDamage(5000); boss.HP != 2700 {
		t.Fatalf("HP %d after a hit during ins_335(60), want 2700", boss.HP)
	}

	vm.LateUpdate(ctx)
	if h := vm.BossHUD(); !h.Spell || h.TimeLeft != 45 {
		t.Fatalf("HUD spell %v time %d, want the card with 45s", h.Spell, h.TimeLeft)
	}
}

// TestVMInterruptOnTimeout: when the phase timer reaches the slot's time, the
// timeout sub takes over the next frame with the HP at the slot's value, the
// 0x10000 flag set and faith cut by 3000 (x10 here).
func TestVMInterruptOnTimeout(t *testing.T) {
	vm, ctx, boss := reachBossCard1(t)
	// The hit came after the update; FUN_0040dc80 deals it before the
	// timers, so finish the frame: phase timer 0 -> 1.
	vm.actors[boss].phase.tick(1)
	ctx.State.Faith = 90000
	for k := 1; k < 2700; k++ {
		stepFrame(vm, ctx)
	}
	if boss.ECLFlags&0x10000 != 0 || vm.hudTimer != 1 {
		t.Fatalf("frame 2699: flags %#x timer %d, want no timeout yet and 1s left", boss.ECLFlags, vm.hudTimer)
	}
	stepFrame(vm, ctx)
	a := vm.actors[boss]
	if boss.ECLFlags&0x10000 == 0 || a.main.sub.Name != "Boss2" || a.main.pc != 0 {
		t.Fatalf("frame 2700: flags %#x sub %q pc %d, want the timeout into Boss2", boss.ECLFlags, a.main.sub.Name, a.main.pc)
	}
	if boss.HP != 0 || ctx.State.Faith != 60000 {
		t.Fatalf("HP %d faith %d, want 0 and 60000", boss.HP, ctx.State.Faith)
	}
	stepFrame(vm, ctx)
	if !boss.Active || boss.HP != 13400 || a.slots[0].sub != "BossCard2" {
		t.Fatalf("after Boss2 ran: active %v HP %d slot %+v", boss.Active, boss.HP, a.slots[0])
	}
}

// TestVMBossDeadSurvivesZeroHP: BossDead sets flag 0x40, so running out of HP
// leaves Hina alive until its own delete() 60 frames later.
func TestVMBossDeadSurvivesZeroHP(t *testing.T) {
	vm, ctx, boss := spawnStageSub(t, "ecl/stage02.ecl", "Boss")
	a := vm.actors[boss]
	vm.interrupt(boss, a, "BossCard3")
	vm.stepTask(ctx, a.main)
	for i := 0; i < 61; i++ {
		stepFrame(vm, ctx)
	}
	if boss.TakeDamage(5 * (boss.HP + 100)) { // a fifth during the card
		t.Fatal("killed despite BossDead's flag 0x40")
	}
	if !boss.Active || boss.HP != 0 || a.main.sub.Name != "BossDead" {
		t.Fatalf("active %v HP %d sub %q, want BossDead at 0 HP", boss.Active, boss.HP, a.main.sub.Name)
	}
	for i := 0; i < 61 && boss.Active; i++ {
		stepFrame(vm, ctx)
	}
	if boss.Active || vm.bossSlots[0] != nil {
		t.Fatalf("active %v slot0 %p, want deleted and slot 0 free", boss.Active, vm.bossSlots[0])
	}
}

// TestVMMidBossDiesWithItsPhase: MBossDead has no flag 0x40, so it runs for
// that one frame and the midboss is destroyed by the same hit.
func TestVMMidBossDiesWithItsPhase(t *testing.T) {
	vm, ctx, mb := spawnStageSub(t, "ecl/stage02.ecl", "MBoss")
	if s := vm.actors[mb].slots[0]; s.sub != "MBossDead" || s.timeoutSub != "MBossEscape" || s.time != 2400 {
		t.Fatalf("slot 0 = %+v, want (0, 2400, MBossDead) with ins_341's MBossEscape", s)
	}
	for i := 0; i < 61; i++ { // MBossCard1's ins_335(60)
		stepFrame(vm, ctx)
	}
	if !mb.TakeDamage(5 * 5000) { // a fifth during the card
		t.Fatal("the hit that triggers MBossDead did not destroy the midboss")
	}
	if mb.Active || vm.bossSlots[0] != nil {
		t.Fatalf("active %v slot0 %p, want gone", mb.Active, vm.bossSlots[0])
	}
}

// TestVMBossWaitFollowsSlot0: ins_340 blocks only on boss slot 0.
func TestVMBossWaitFollowsSlot0(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	boss := enemy.New(0, 0, 100, enemy.TypeBoss)
	wait := &Instr{Opcode: opBossWait}
	task := &eclTask{}

	vm.execGameOp(ctx, &eclTask{owner: boss}, &Instr{Opcode: opSetBoss, Params: []Param{intParam(1)}})
	if vm.execGameOp(ctx, task, wait); task.blocked {
		t.Fatal("blocked on boss slot 1")
	}
	vm.execGameOp(ctx, &eclTask{owner: boss}, &Instr{Opcode: opSetBoss, Params: []Param{intParam(0)}})
	if vm.execGameOp(ctx, task, wait); !task.blocked {
		t.Fatal("did not block on boss slot 0")
	}
	task.blocked = false
	boss.Active = false
	if vm.execGameOp(ctx, task, wait); task.blocked {
		t.Fatal("still blocked after the boss died")
	}
}

func testSub(instrs ...Instr) *Sub {
	for i := range instrs {
		instrs[i].RankMask = 0xff
	}
	return &Sub{Name: "test", Instrs: instrs}
}

// TestVMWaitDelaysOneFrame: wait(n) steps the clock back n frames, so what
// follows wait(1) runs exactly one frame later.
func TestVMWaitDelaysOneFrame(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	e := enemy.New(0, 0, 100, enemy.TypeFairy)
	task := &eclTask{owner: e, sub: testSub(
		Instr{Opcode: opWait, Params: []Param{intParam(1)}},
		Instr{Opcode: opFlagSet, Params: []Param{intParam(4)}},
	)}
	if vm.stepTask(ctx, task); e.ECLFlags != 0 {
		t.Fatal("ran past wait(1) in the same frame")
	}
	if vm.stepTask(ctx, task); e.ECLFlags != 4 {
		t.Fatal("wait(1) took more than one frame")
	}
}

// TestVMBlockedOpRetries: an op that returns -1 holds the clock and runs again
// every frame until it passes.
func TestVMBlockedOpRetries(t *testing.T) {
	ctx := newTestContext()
	vm := newTestVM()
	boss := enemy.New(0, 0, 100, enemy.TypeBoss)
	vm.actorFor(boss)
	vm.bossSlots[0] = boss
	e := enemy.New(0, 0, 100, enemy.TypeFairy)
	task := &eclTask{owner: e, sub: testSub(
		Instr{Opcode: opBossWait},
		Instr{Opcode: opFlagSet, Params: []Param{intParam(4)}},
	)}
	for i := 0; i < 3; i++ {
		vm.stepTask(ctx, task)
	}
	if task.pc != 0 || task.clock != 0 || e.ECLFlags != 0 {
		t.Fatalf("pc %d clock %v flags %#x while blocked", task.pc, task.clock, e.ECLFlags)
	}
	boss.Active = false
	if vm.stepTask(ctx, task); e.ECLFlags != 4 {
		t.Fatal("did not continue once slot 0 emptied")
	}
}

// TestVMReturnEndsTask: op 10 at the outermost level ends the context; an
// async one is freed, the end of the primary one ends the enemy.
func TestVMReturnEndsTask(t *testing.T) {
	vm := newTestVM()
	e := enemy.New(0, 0, 100, enemy.TypeFairy)
	a := vm.actorFor(e)
	a.main = &eclTask{owner: e, sub: testSub(
		Instr{Opcode: opWait, Params: []Param{intParam(1)}},
		Instr{Opcode: opRetNormal},
	)}
	async := &eclTask{owner: e, async: true, sub: testSub(Instr{Opcode: opRetNormal})}
	a.asyncs = []*eclTask{async}
	if !vm.runActor(a) || !async.done || len(a.asyncs) != 0 {
		t.Fatalf("async: done %v, %d asyncs left; want the enemy kept and the async freed", async.done, len(a.asyncs))
	}
	if vm.runActor(a) || !a.main.done {
		t.Fatalf("primary: done %v, want done and the enemy ended", a.main.done)
	}
}

// TestVMBossesTimeOutToClear: with nobody shooting, every boss phase ends on
// its time slot, the last one's timeout sub is BossDead, and the stage main
// opens the post dialog, the .msg entry whose op 20 clears the stage. Main
// ends after it in stages 1-3; in 4-6 and Extra it loops forever, so only the
// MSG can clear them. The player passes each wait at once.
func TestVMBossesTimeOutToClear(t *testing.T) {
	for _, tc := range []struct {
		ecl, msg string
		within   int
		dialogs  []int32 // the entries main opens, in order
		mainEnds bool
	}{
		{"ecl/stage01.ecl", "stage1", 17000, []int32{0, 1}, true},
		{"ecl/stage02.ecl", "stage2", 21000, []int32{0, 1}, true},
		{"ecl/stage03.ecl", "stage3", 33000, []int32{0, 2, 3}, true}, // entry 1 follows MBossDead, not MBossEscape
		{"ecl/stage04.ecl", "stage4", 40000, []int32{0, 1}, false},
		{"ecl/stage05.ecl", "stage5", 45000, []int32{0, 1}, false},
		{"ecl/stage06.ecl", "stage6", 50000, []int32{0, 1}, false},
		{"ecl/stage07.ecl", "extra", 125000, []int32{0, 1, 2}, false},
	} {
		ctx := newTestContext()
		ctx.LoadMSG(tc.msg, 0)
		vm := NewStageVM(tc.ecl)
		var opened []int32
		var last *dialog.MSG
		step := func() {
			stepFrame(vm, ctx)
			updateBullets(ctx)
			if m := ctx.MSG(); m != nil && m != last {
				opened, last = append(opened, m.ID), m
			}
			ctx.UpdateMSG(true, false)
		}
		sawDead := false
		f := 0
		for ; f < tc.within && len(opened) < len(tc.dialogs); f++ {
			step()
			if b := vm.bossSlots[0]; b != nil && vm.actors[b].main.sub.Name == "BossDead" {
				sawDead = true
			}
		}
		if !reflect.DeepEqual(opened, tc.dialogs) || !sawDead {
			t.Errorf("%s: opened %v, want %v; reached BossDead %v after %d frames",
				tc.ecl, opened, tc.dialogs, sawDead, f)
			continue
		}
		for i := 0; i < 2000 && !ctx.Cleared(); i++ {
			step()
		}
		for i := 0; i < 600 && !vm.MainDone(); i++ {
			step()
		}
		if !ctx.Cleared() || vm.MainDone() != tc.mainEnds {
			t.Errorf("%s: cleared %v, main done %v after the post dialog, want true, %v",
				tc.ecl, ctx.Cleared(), vm.MainDone(), tc.mainEnds)
		}
	}
}
